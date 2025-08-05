package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Garvitkul/s3mpc/internal/config"
	"github.com/Garvitkul/s3mpc/internal/container"
	"github.com/Garvitkul/s3mpc/pkg/types"
)

// App represents the main application
type App struct {
	container *container.Container
	rootCmd   *cobra.Command
}

// NewApp creates a new application instance
func NewApp() *App {
	app := &App{}
	app.setupCommands()
	return app
}

// Run executes the application with the given arguments
func (a *App) Run(ctx context.Context, args []string) error {
	a.rootCmd.SetArgs(args)
	return a.rootCmd.ExecuteContext(ctx)
}

// setupCommands initializes the CLI command structure
func (a *App) setupCommands() {
	a.rootCmd = &cobra.Command{
		Use:   "s3mpc",
		Short: "S3 MultiPart Cleaner - Manage incomplete S3 multipart uploads",
		Long: `s3mpc is a command-line tool for managing incomplete S3 multipart uploads.
It helps you discover, analyze, and clean up incomplete uploads across all your S3 buckets.`,
		PersistentPreRunE: a.initializeContainer,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --version flag
			if version, _ := cmd.Flags().GetBool("version"); version {
				cmd.Println("s3mpc version 1.0.0")
				return nil
			}
			// Show help if no subcommand is provided
			return cmd.Help()
		},
	}

	// Global flags
	a.rootCmd.PersistentFlags().String("profile", "", "AWS profile to use")
	a.rootCmd.PersistentFlags().String("region", "", "AWS region to focus on")
	a.rootCmd.PersistentFlags().Int("concurrency", 10, "Number of concurrent operations")
	a.rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose logging")
	a.rootCmd.PersistentFlags().Bool("quiet", false, "Suppress non-essential output")
	a.rootCmd.PersistentFlags().String("log-file", "", "Write logs to file")
	a.rootCmd.Flags().BoolP("version", "v", false, "Show version information")

	// Add version command
	a.rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("s3mpc version 1.0.0")
		},
	})

	// Add commands
	a.addSizeCommand()
	a.addCostCommand()
	a.addListCommand()
	a.addAgeCommand()
	a.addDeleteCommand()
	a.addExportCommand()
}

// initializeContainer sets up the dependency injection container
func (a *App) initializeContainer(cmd *cobra.Command, args []string) error {
	// Get flag values
	profile, _ := cmd.Flags().GetString("profile")
	region, _ := cmd.Flags().GetString("region")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	verbose, _ := cmd.Flags().GetBool("verbose")
	quiet, _ := cmd.Flags().GetBool("quiet")
	logFile, _ := cmd.Flags().GetString("log-file")

	// Validate configuration
	if err := a.validateConfig(profile, region, concurrency, verbose, quiet, logFile); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create container configuration
	cfg := &config.Config{
		AWSProfile:  profile,
		AWSRegion:   region,
		Concurrency: concurrency,
		Verbose:     verbose,
		LogFile:     logFile,
	}

	// Initialize container
	var err error
	a.container, err = container.NewContainer(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	return nil
}

// validateConfig validates the configuration parameters
func (a *App) validateConfig(profile, region string, concurrency int, verbose, quiet bool, logFile string) error {
	// Validate concurrency
	if concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1, got %d", concurrency)
	}
	if concurrency > 100 {
		return fmt.Errorf("concurrency cannot exceed 100, got %d", concurrency)
	}

	// Validate that verbose and quiet are not both set
	if verbose && quiet {
		return fmt.Errorf("cannot use both --verbose and --quiet flags")
	}

	// Validate log file path if specified
	if logFile != "" {
		if strings.TrimSpace(logFile) == "" {
			return fmt.Errorf("log file path cannot be empty")
		}
	}

	// Validate AWS region format if specified
	if region != "" {
		if !a.isValidAWSRegion(region) {
			return fmt.Errorf("invalid AWS region format: %q", region)
		}
	}

	return nil
}

// isValidAWSRegion checks if the region format is valid
func (a *App) isValidAWSRegion(region string) bool {
	// Basic validation for AWS region format
	if len(region) < 9 {
		return false
	}
	
	parts := strings.Split(region, "-")
	if len(parts) < 3 {
		return false
	}
	
	// Last part should be a number
	lastPart := parts[len(parts)-1]
	if len(lastPart) == 0 {
		return false
	}
	
	for _, r := range lastPart {
		if r < '0' || r > '9' {
			return false
		}
	}
	
	return true
}

// Command implementations
func (a *App) addSizeCommand() {
	cmd := &cobra.Command{
		Use:   "size",
		Short: "Show storage usage of incomplete uploads",
		RunE:  a.runSizeCommand,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().BoolP("bucket", "b", false, "Show per-bucket breakdown")
	a.rootCmd.AddCommand(cmd)
}

func (a *App) runSizeCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	jsonOutput, _ := cmd.Flags().GetBool("json")
	bucketBreakdown, _ := cmd.Flags().GetBool("bucket")
	
	sizeService := a.container.GetSizeService()
	formatter := a.container.GetOutputFormatter()
	
	report, err := sizeService.CalculateTotalSize(ctx, types.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to calculate size: %w", err)
	}
	
	if report.TotalCount == 0 {
		if jsonOutput {
			result := map[string]interface{}{
				"total_uploads": 0,
				"total_size":    0,
				"message":       "No incomplete multipart uploads found",
			}
			jsonStr, err := formatter.FormatJSON(result)
			if err != nil {
				return fmt.Errorf("failed to format JSON output: %w", err)
			}
			cmd.Println(jsonStr)
		} else {
			cmd.Println("No incomplete multipart uploads found.")
		}
		return nil
	}
	
	if jsonOutput {
		jsonStr, err := formatter.FormatJSON(report)
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %w", err)
		}
		cmd.Println(jsonStr)
	} else {
		if !bucketBreakdown {
			report.ByBucket = make(map[string]int64)
		}
		
		output := formatter.FormatSizeReport(*report)
		cmd.Print(output)
	}
	
	return nil
}

func (a *App) addCostCommand() {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Calculate estimated storage costs",
		RunE:  a.runCostCommand,
	}
	cmd.Flags().Bool("storage-class", false, "Show cost breakdown by storage class")
	cmd.Flags().Bool("json", false, "Output in JSON format")
	a.rootCmd.AddCommand(cmd)
}

func (a *App) runCostCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	storageClassBreakdown, _ := cmd.Flags().GetBool("storage-class")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	
	uploadService := a.container.GetUploadService()
	costCalculator := a.container.GetCostCalculator()
	formatter := a.container.GetOutputFormatter()
	
	uploads, err := uploadService.ListUploads(ctx, types.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list uploads: %w", err)
	}
	
	if len(uploads) == 0 {
		if jsonOutput {
			result := map[string]interface{}{
				"total_monthly_cost": 0.0,
				"currency":           "USD",
				"message":            "No incomplete multipart uploads found",
			}
			jsonStr, err := formatter.FormatJSON(result)
			if err != nil {
				return fmt.Errorf("failed to format JSON output: %w", err)
			}
			cmd.Println(jsonStr)
		} else {
			cmd.Println("No incomplete multipart uploads found.")
		}
		return nil
	}
	
	breakdown, err := costCalculator.CalculateStorageCost(ctx, uploads)
	if err != nil {
		return fmt.Errorf("failed to calculate costs: %w", err)
	}
	
	if jsonOutput {
		jsonStr, err := formatter.FormatJSON(breakdown)
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %w", err)
		}
		cmd.Println(jsonStr)
	} else {
		if !storageClassBreakdown {
			breakdown.ByStorageClass = make(map[string]float64)
		}
		
		output := formatter.FormatCostBreakdown(breakdown)
		cmd.Print(output)
	}
	
	return nil
}

func (a *App) addListCommand() {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List incomplete uploads with details",
		RunE:  a.runListCommand,
	}
	cmd.Flags().StringP("bucket", "b", "", "List uploads for specific bucket")
	cmd.Flags().String("filter", "", "Filter uploads using query syntax")
	cmd.Flags().String("sort-by", "age", "Sort by: age, size, bucket")
	cmd.Flags().Int("limit", 0, "Limit number of results")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().Bool("json", false, "Output in JSON format")
	a.rootCmd.AddCommand(cmd)
}

func (a *App) runListCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	bucketName, _ := cmd.Flags().GetString("bucket")
	filterStr, _ := cmd.Flags().GetString("filter")
	sortBy, _ := cmd.Flags().GetString("sort-by")
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	
	uploadService := a.container.GetUploadService()
	filterEngine := a.container.GetFilterEngine()
	formatter := a.container.GetOutputFormatter()
	
	listOpts := types.ListOptions{
		BucketName: bucketName,
		MaxResults: limit,
		Offset:     offset,
	}
	
	uploads, err := uploadService.ListUploads(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list uploads: %w", err)
	}
	
	if filterStr != "" {
		filter, err := filterEngine.ParseFilter(filterStr)
		if err != nil {
			return fmt.Errorf("invalid filter syntax: %w", err)
		}
		uploads = filterEngine.ApplyFilter(uploads, filter)
	}
	
	uploads = a.sortUploads(uploads, sortBy)
	
	if offset > 0 {
		if offset >= len(uploads) {
			uploads = []types.MultipartUpload{}
		} else {
			uploads = uploads[offset:]
		}
	}
	
	if limit > 0 && len(uploads) > limit {
		uploads = uploads[:limit]
	}
	
	if jsonOutput {
		result := map[string]interface{}{
			"uploads":     uploads,
			"total_count": len(uploads),
			"filter":      filterStr,
			"sort_by":     sortBy,
			"limit":       limit,
			"offset":      offset,
		}
		jsonStr, err := formatter.FormatJSON(result)
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %w", err)
		}
		cmd.Println(jsonStr)
	} else {
		if len(uploads) == 0 {
			cmd.Println("No incomplete multipart uploads found.")
			return nil
		}
		
		output := formatter.FormatUploads(uploads, true)
		cmd.Print(output)
		
		if limit > 0 || offset > 0 {
			cmd.Printf("\nShowing %d uploads", len(uploads))
			if offset > 0 {
				cmd.Printf(" (offset: %d)", offset)
			}
			if limit > 0 {
				cmd.Printf(" (limit: %d)", limit)
			}
			cmd.Println()
		}
	}
	
	return nil
}

func (a *App) sortUploads(uploads []types.MultipartUpload, sortBy string) []types.MultipartUpload {
	sorted := make([]types.MultipartUpload, len(uploads))
	copy(sorted, uploads)
	
	switch sortBy {
	case "age":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Initiated.Before(sorted[j].Initiated)
		})
	case "size":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Size > sorted[j].Size
		})
	case "bucket":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Bucket < sorted[j].Bucket
		})
	}
	
	return sorted
}

func (a *App) addAgeCommand() {
	cmd := &cobra.Command{
		Use:   "age",
		Short: "Show age distribution of uploads",
		RunE:  a.runAgeCommand,
	}
	cmd.Flags().StringP("bucket", "b", "", "Show age distribution for specific bucket")
	cmd.Flags().Bool("json", false, "Output in JSON format")
	a.rootCmd.AddCommand(cmd)
}

func (a *App) runAgeCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	bucketName, _ := cmd.Flags().GetString("bucket")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	
	uploadService := a.container.GetUploadService()
	ageService := a.container.GetAgeService()
	formatter := a.container.GetOutputFormatter()
	
	listOpts := types.ListOptions{
		BucketName: bucketName,
	}
	
	uploads, err := uploadService.ListUploads(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list uploads: %w", err)
	}
	
	if len(uploads) == 0 {
		if jsonOutput {
			result := map[string]interface{}{
				"buckets": []interface{}{},
				"message": "No incomplete multipart uploads found",
			}
			jsonStr, err := formatter.FormatJSON(result)
			if err != nil {
				return fmt.Errorf("failed to format JSON output: %w", err)
			}
			cmd.Println(jsonStr)
		} else {
			cmd.Println("No incomplete multipart uploads found.")
		}
		return nil
	}
	
	var distribution types.AgeDistribution
	if bucketName != "" {
		distribution, err = ageService.GetAgeDistributionForBucket(ctx, uploads, bucketName)
	} else {
		distribution, err = ageService.CalculateAgeDistribution(ctx, uploads)
	}
	
	if err != nil {
		return fmt.Errorf("failed to calculate age distribution: %w", err)
	}
	
	if jsonOutput {
		jsonStr, err := formatter.FormatJSON(distribution)
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %w", err)
		}
		cmd.Println(jsonStr)
	} else {
		output := formatter.FormatAgeDistribution(distribution)
		cmd.Print(output)
		
		if bucketName != "" {
			cmd.Printf("\nAge distribution for bucket: %q\n", bucketName)
		}
	}
	
	return nil
}

func (a *App) addDeleteCommand() {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete incomplete uploads",
		RunE:  a.runDeleteCommand,
	}
	cmd.Flags().Bool("force", false, "Skip confirmation prompts")
	cmd.Flags().Bool("dry-run", false, "Show what would be deleted without deleting")
	cmd.Flags().String("older-than", "", "Delete uploads older than specified duration (e.g., 7d, 1w, 1m)")
	cmd.Flags().String("smaller-than", "", "Delete uploads smaller than specified size (e.g., 100MB, 1GB)")
	cmd.Flags().String("larger-than", "", "Delete uploads larger than specified size (e.g., 100MB, 1GB)")
	cmd.Flags().StringP("bucket", "b", "", "Delete uploads from specific bucket")
	a.rootCmd.AddCommand(cmd)
}

func (a *App) runDeleteCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	olderThan, _ := cmd.Flags().GetString("older-than")
	smallerThan, _ := cmd.Flags().GetString("smaller-than")
	largerThan, _ := cmd.Flags().GetString("larger-than")
	bucketName, _ := cmd.Flags().GetString("bucket")
	
	uploadService := a.container.GetUploadService()
	
	deleteOpts := types.DeleteOptions{
		Force:      force,
		DryRun:     dryRun,
		BucketName: bucketName,
		Quiet:      false,
	}
	
	if olderThan != "" {
		duration, err := a.parseDuration(olderThan)
		if err != nil {
			return fmt.Errorf("invalid --older-than value: %w", err)
		}
		deleteOpts.OlderThan = &duration
	}
	
	if smallerThan != "" {
		size, err := a.parseSize(smallerThan)
		if err != nil {
			return fmt.Errorf("invalid --smaller-than value: %w", err)
		}
		deleteOpts.SmallerThan = &size
	}
	
	if largerThan != "" {
		size, err := a.parseSize(largerThan)
		if err != nil {
			return fmt.Errorf("invalid --larger-than value: %w", err)
		}
		deleteOpts.LargerThan = &size
	}
	
	listOpts := types.ListOptions{
		BucketName: bucketName,
	}
	
	uploads, err := uploadService.ListUploads(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list uploads: %w", err)
	}
	
	if len(uploads) == 0 {
		cmd.Println("No incomplete multipart uploads found.")
		return nil
	}
	
	err = uploadService.DeleteUploads(ctx, uploads, deleteOpts)
	if err != nil {
		return fmt.Errorf("failed to delete uploads: %w", err)
	}
	
	return nil
}

func (a *App) parseDuration(durationStr string) (time.Duration, error) {
	if len(durationStr) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}
	
	unit := durationStr[len(durationStr)-1:]
	valueStr := durationStr[:len(durationStr)-1]
	
	value := 0
	for _, r := range valueStr {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid duration format")
		}
		value = value*10 + int(r-'0')
	}
	
	switch unit {
	case "s":
		return time.Duration(value) * time.Second, nil
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %q (use s, m, h, d, w)", unit)
	}
}

func (a *App) parseSize(sizeStr string) (int64, error) {
	if len(sizeStr) < 2 {
		return 0, fmt.Errorf("invalid size format")
	}
	
	var valueStr string
	var unit string
	
	for i := len(sizeStr) - 1; i >= 0; i-- {
		if sizeStr[i] >= '0' && sizeStr[i] <= '9' || sizeStr[i] == '.' {
			valueStr = sizeStr[:i+1]
			unit = sizeStr[i+1:]
			break
		}
	}
	
	if valueStr == "" {
		return 0, fmt.Errorf("invalid size format")
	}
	
	var value float64
	dotFound := false
	intPart := 0
	fracPart := 0
	fracDigits := 0
	
	for _, r := range valueStr {
		if r == '.' {
			if dotFound {
				return 0, fmt.Errorf("invalid size format")
			}
			dotFound = true
		} else if r >= '0' && r <= '9' {
			if dotFound {
				fracPart = fracPart*10 + int(r-'0')
				fracDigits++
			} else {
				intPart = intPart*10 + int(r-'0')
			}
		} else {
			return 0, fmt.Errorf("invalid size format")
		}
	}
	
	value = float64(intPart)
	if fracDigits > 0 {
		fracValue := float64(fracPart)
		for i := 0; i < fracDigits; i++ {
			fracValue /= 10
		}
		value += fracValue
	}
	
	switch strings.ToUpper(unit) {
	case "B", "":
		return int64(value), nil
	case "KB", "K":
		return int64(value * 1024), nil
	case "MB", "M":
		return int64(value * 1024 * 1024), nil
	case "GB", "G":
		return int64(value * 1024 * 1024 * 1024), nil
	case "TB", "T":
		return int64(value * 1024 * 1024 * 1024 * 1024), nil
	default:
		return 0, fmt.Errorf("unsupported size unit: %q (use B, KB, MB, GB, TB)", unit)
	}
}

func (a *App) addExportCommand() {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export upload data to files",
		RunE:  a.runExportCommand,
	}
	cmd.Flags().String("format", "csv", "Export format: csv, json")
	cmd.Flags().String("filter", "", "Filter uploads using query syntax")
	cmd.Flags().StringP("bucket", "b", "", "Export uploads from specific bucket")
	cmd.Flags().StringP("output", "o", "", "Output file path (auto-generated if not specified)")
	a.rootCmd.AddCommand(cmd)
}

func (a *App) runExportCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	format, _ := cmd.Flags().GetString("format")
	filterStr, _ := cmd.Flags().GetString("filter")
	bucketName, _ := cmd.Flags().GetString("bucket")
	outputFile, _ := cmd.Flags().GetString("output")
	
	if format != "csv" && format != "json" {
		return fmt.Errorf("invalid format: %q (must be csv or json)", format)
	}
	
	uploadService := a.container.GetUploadService()
	exportService := a.container.GetExportService()
	filterEngine := a.container.GetFilterEngine()
	
	listOpts := types.ListOptions{
		BucketName: bucketName,
	}
	
	uploads, err := uploadService.ListUploads(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list uploads: %w", err)
	}
	
	if filterStr != "" {
		filter, err := filterEngine.ParseFilter(filterStr)
		if err != nil {
			return fmt.Errorf("invalid filter syntax: %w", err)
		}
		uploads = filterEngine.ApplyFilter(uploads, filter)
	}
	
	if len(uploads) == 0 {
		cmd.Println("No uploads found to export.")
		return nil
	}
	
	if outputFile == "" {
		commandStr := "export"
		if bucketName != "" {
			commandStr += "_" + bucketName
		}
		if filterStr != "" {
			commandStr += "_filtered"
		}
		outputFile = exportService.GenerateExportFilename(commandStr, format)
	}
	
	switch format {
	case "csv":
		err = exportService.ExportToCSV(ctx, uploads, outputFile)
	case "json":
		err = exportService.ExportToJSON(ctx, uploads, outputFile)
	}
	
	if err != nil {
		return fmt.Errorf("failed to export data: %w", err)
	}
	
	cmd.Printf("Successfully exported %d uploads to %q\n", len(uploads), outputFile)
	
	var totalSize int64
	bucketCounts := make(map[string]int)
	for _, upload := range uploads {
		totalSize += upload.Size
		bucketCounts[upload.Bucket]++
	}
	
	cmd.Printf("Total size: %s\n", FormatBytes(totalSize))
	cmd.Printf("Buckets: %d\n", len(bucketCounts))
	
	if len(bucketCounts) <= 5 {
		cmd.Println("Bucket breakdown:")
		for bucket, count := range bucketCounts {
			cmd.Printf("  %q: %d uploads\n", bucket, count)
		}
	}
	
	return nil
}

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}