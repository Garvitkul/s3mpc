package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Garvitkul/s3mpc/pkg/interfaces"
	"github.com/Garvitkul/s3mpc/pkg/types"
)

// DryRunService implements the interfaces.DryRunService interface
type DryRunService struct {
	costCalculator interfaces.CostCalculator
}

// NewDryRunService creates a new DryRunService instance
func NewDryRunService(costCalculator interfaces.CostCalculator) interfaces.DryRunService {
	return &DryRunService{
		costCalculator: costCalculator,
	}
}

// SimulateDeletion simulates deletion without executing it
func (d *DryRunService) SimulateDeletion(ctx context.Context, uploads []types.MultipartUpload, opts types.DeleteOptions) (types.DryRunResult, error) {
	if err := opts.Validate(); err != nil {
		return types.DryRunResult{}, fmt.Errorf("invalid delete options: %w", err)
	}

	// Filter uploads based on options (same logic as actual deletion)
	filteredUploads := d.filterUploadsForDeletion(uploads, opts)

	// Calculate cost savings
	estimatedSavings, err := d.costCalculator.EstimateSavings(ctx, filteredUploads)
	if err != nil {
		// If cost calculation fails, continue with 0 savings
		estimatedSavings = 0
	}

	// Generate breakdown statistics
	result := types.DryRunResult{
		TotalUploads:          len(filteredUploads),
		TotalSize:             d.calculateTotalSize(filteredUploads),
		EstimatedSavings:      estimatedSavings,
		Currency:              "USD",
		UploadsByBucket:       make(map[string]int),
		SizeByBucket:          make(map[string]int64),
		SavingsByBucket:       make(map[string]float64),
		UploadsByRegion:       make(map[string]int),
		SizeByRegion:          make(map[string]int64),
		SavingsByRegion:       make(map[string]float64),
		UploadsByStorageClass: make(map[string]int),
		SizeByStorageClass:    make(map[string]int64),
		SavingsByStorageClass: make(map[string]float64),
		Uploads:               filteredUploads,
		GeneratedAt:           time.Now(),
		Command:               d.buildCommandString(opts),
		Filters:               d.buildFilterString(opts),
	}

	// Calculate breakdowns
	d.calculateBreakdowns(ctx, filteredUploads, &result)

	return result, nil
}

// SaveDryRunResult saves dry-run results to a file
func (d *DryRunService) SaveDryRunResult(result types.DryRunResult, filename string) error {
	if err := result.Validate(); err != nil {
		return fmt.Errorf("invalid dry-run result: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Determine format based on file extension
	ext := strings.ToLower(filepath.Ext(filename))
	
	switch ext {
	case ".json":
		return d.saveAsJSON(result, filename)
	case ".csv":
		return d.saveAsCSV(result, filename)
	default:
		// Default to JSON if no extension or unknown extension
		return d.saveAsJSON(result, filename)
	}
}

// GenerateFilename generates a filename for dry-run results
func (d *DryRunService) GenerateFilename(command string, format string) string {
	timestamp := time.Now().Format("20060102_1504")
	
	// Sanitize command name
	sanitizedCommand := strings.ReplaceAll(command, " ", "_")
	sanitizedCommand = strings.ReplaceAll(sanitizedCommand, "-", "_")
	
	// Ensure format is lowercase
	format = strings.ToLower(format)
	if format != "csv" && format != "json" {
		format = "json" // Default to JSON
	}
	
	return fmt.Sprintf("s3mpc_%s_dryrun_%s.%s", sanitizedCommand, timestamp, format)
}

// filterUploadsForDeletion filters uploads based on delete options
func (d *DryRunService) filterUploadsForDeletion(uploads []types.MultipartUpload, opts types.DeleteOptions) []types.MultipartUpload {
	var filtered []types.MultipartUpload

	for _, upload := range uploads {
		// Filter by bucket if specified
		if opts.BucketName != "" && upload.Bucket != opts.BucketName {
			continue
		}

		// Filter by age if specified
		if opts.OlderThan != nil {
			age := time.Since(upload.Initiated)
			if age < *opts.OlderThan {
				continue
			}
		}

		// Filter by size if specified
		if opts.SmallerThan != nil && upload.Size >= *opts.SmallerThan {
			continue
		}

		if opts.LargerThan != nil && upload.Size <= *opts.LargerThan {
			continue
		}

		filtered = append(filtered, upload)
	}

	return filtered
}

// calculateTotalSize calculates the total size of uploads
func (d *DryRunService) calculateTotalSize(uploads []types.MultipartUpload) int64 {
	var total int64
	for _, upload := range uploads {
		total += upload.Size
	}
	return total
}

// calculateBreakdowns calculates various breakdown statistics
func (d *DryRunService) calculateBreakdowns(ctx context.Context, uploads []types.MultipartUpload, result *types.DryRunResult) {
	// Group uploads by bucket, region, and storage class
	for _, upload := range uploads {
		// By bucket
		result.UploadsByBucket[upload.Bucket]++
		result.SizeByBucket[upload.Bucket] += upload.Size

		// By region
		result.UploadsByRegion[upload.Region]++
		result.SizeByRegion[upload.Region] += upload.Size

		// By storage class
		result.UploadsByStorageClass[upload.StorageClass]++
		result.SizeByStorageClass[upload.StorageClass] += upload.Size
	}

	// Calculate cost savings breakdowns
	d.calculateCostBreakdowns(ctx, uploads, result)
}

// calculateCostBreakdowns calculates cost savings breakdowns
func (d *DryRunService) calculateCostBreakdowns(ctx context.Context, uploads []types.MultipartUpload, result *types.DryRunResult) {
	// Group uploads by bucket for cost calculation
	bucketUploads := make(map[string][]types.MultipartUpload)
	regionUploads := make(map[string][]types.MultipartUpload)
	storageClassUploads := make(map[string][]types.MultipartUpload)

	for _, upload := range uploads {
		bucketUploads[upload.Bucket] = append(bucketUploads[upload.Bucket], upload)
		regionUploads[upload.Region] = append(regionUploads[upload.Region], upload)
		storageClassUploads[upload.StorageClass] = append(storageClassUploads[upload.StorageClass], upload)
	}

	// Calculate savings by bucket
	for bucket, bucketUploadList := range bucketUploads {
		if savings, err := d.costCalculator.EstimateSavings(ctx, bucketUploadList); err == nil {
			result.SavingsByBucket[bucket] = savings
		}
	}

	// Calculate savings by region
	for region, regionUploadList := range regionUploads {
		if savings, err := d.costCalculator.EstimateSavings(ctx, regionUploadList); err == nil {
			result.SavingsByRegion[region] = savings
		}
	}

	// Calculate savings by storage class
	for storageClass, storageClassUploadList := range storageClassUploads {
		if savings, err := d.costCalculator.EstimateSavings(ctx, storageClassUploadList); err == nil {
			result.SavingsByStorageClass[storageClass] = savings
		}
	}
}

// buildCommandString builds a command string representation
func (d *DryRunService) buildCommandString(opts types.DeleteOptions) string {
	parts := []string{"delete"}

	if opts.BucketName != "" {
		parts = append(parts, fmt.Sprintf("-b %s", opts.BucketName))
	}

	if opts.OlderThan != nil {
		parts = append(parts, fmt.Sprintf("--older-than %s", d.formatDuration(*opts.OlderThan)))
	}

	if opts.SmallerThan != nil {
		parts = append(parts, fmt.Sprintf("--smaller-than %s", d.formatBytes(*opts.SmallerThan)))
	}

	if opts.LargerThan != nil {
		parts = append(parts, fmt.Sprintf("--larger-than %s", d.formatBytes(*opts.LargerThan)))
	}

	if opts.Force {
		parts = append(parts, "--force")
	}

	if opts.Quiet {
		parts = append(parts, "--quiet")
	}

	return strings.Join(parts, " ")
}

// buildFilterString builds a filter string representation
func (d *DryRunService) buildFilterString(opts types.DeleteOptions) string {
	var filters []string

	if opts.OlderThan != nil {
		filters = append(filters, fmt.Sprintf("age>%s", d.formatDuration(*opts.OlderThan)))
	}

	if opts.SmallerThan != nil {
		filters = append(filters, fmt.Sprintf("size<%s", d.formatBytes(*opts.SmallerThan)))
	}

	if opts.LargerThan != nil {
		filters = append(filters, fmt.Sprintf("size>%s", d.formatBytes(*opts.LargerThan)))
	}

	if opts.BucketName != "" {
		filters = append(filters, fmt.Sprintf("bucket=%s", opts.BucketName))
	}

	return strings.Join(filters, ",")
}

// formatDuration formats a duration for display
func (d *DryRunService) formatDuration(duration time.Duration) string {
	days := int(duration.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	
	hours := int(duration.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	
	minutes := int(duration.Minutes())
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	
	return fmt.Sprintf("%ds", int(duration.Seconds()))
}

// formatBytes formats bytes for display
func (d *DryRunService) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// saveAsJSON saves the result as JSON
func (d *DryRunService) saveAsJSON(result types.DryRunResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// saveAsCSV saves the result as CSV
func (d *DryRunService) saveAsCSV(result types.DryRunResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	// Write CSV header
	header := "bucket,key,upload_id,initiated,age_days,size,storage_class,region,estimated_monthly_cost\n"
	if _, err := file.WriteString(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write upload data
	for _, upload := range result.Uploads {
		ageDays := int(time.Since(upload.Initiated).Hours() / 24)
		
		// Calculate individual upload cost (simplified)
		sizeGB := float64(upload.Size) / (1024 * 1024 * 1024)
		estimatedCost := sizeGB * 0.023 // Use default STANDARD pricing
		
		line := fmt.Sprintf("%s,%s,%s,%s,%d,%d,%s,%s,%.6f\n",
			d.escapeCSV(upload.Bucket),
			d.escapeCSV(upload.Key),
			d.escapeCSV(upload.UploadID),
			upload.Initiated.Format("2006-01-02T15:04:05Z"),
			ageDays,
			upload.Size,
			d.escapeCSV(upload.StorageClass),
			d.escapeCSV(upload.Region),
			estimatedCost,
		)
		
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write CSV line: %w", err)
		}
	}

	return nil
}

// escapeCSV escapes CSV values that contain commas or quotes
func (d *DryRunService) escapeCSV(value string) string {
	if strings.Contains(value, ",") || strings.Contains(value, "\"") || strings.Contains(value, "\n") {
		// Escape quotes by doubling them and wrap in quotes
		escaped := strings.ReplaceAll(value, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}
	return value
}