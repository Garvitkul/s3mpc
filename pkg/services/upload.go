package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awsclient "github.com/Garvitkul/s3mpc/pkg/aws"
	"github.com/Garvitkul/s3mpc/pkg/interfaces"
	pkgtypes "github.com/Garvitkul/s3mpc/pkg/types"
)

// S3UploadClientInterface defines the S3 operations needed by UploadService
type S3UploadClientInterface interface {
	ListMultipartUploads(ctx context.Context, input *s3.ListMultipartUploadsInput) (*s3.ListMultipartUploadsOutput, error)
	ListParts(ctx context.Context, input *s3.ListPartsInput) (*s3.ListPartsOutput, error)
	AbortMultipartUpload(ctx context.Context, input *s3.AbortMultipartUploadInput) (*s3.AbortMultipartUploadOutput, error)
}

// DeletionProgress represents progress information for deletion operations
type DeletionProgress struct {
	TotalUploads     int
	ProcessedUploads int64
	SuccessfulDeletes int64
	FailedDeletes    int64
	CurrentBucket    string
	StartTime        time.Time
	Errors           []DeletionError
}

// DeletionError represents an error that occurred during deletion
type DeletionError struct {
	Upload pkgtypes.MultipartUpload
	Error  error
	Time   time.Time
}

// DeletionResult represents the result of a deletion operation
type DeletionResult struct {
	TotalProcessed    int
	SuccessfulDeletes int
	FailedDeletes     int
	StorageFreed      int64
	Duration          time.Duration
	Errors            []DeletionError
}

// ProgressReporter defines the interface for reporting deletion progress
type ProgressReporter interface {
	ReportProgress(progress DeletionProgress)
	ReportCompletion(result DeletionResult)
}

// ConsoleProgressReporter implements ProgressReporter for console output
type ConsoleProgressReporter struct {
	writer io.Writer
	quiet  bool
}

// NewConsoleProgressReporter creates a new console progress reporter
func NewConsoleProgressReporter(writer io.Writer, quiet bool) *ConsoleProgressReporter {
	if writer == nil {
		writer = os.Stdout
	}
	return &ConsoleProgressReporter{
		writer: writer,
		quiet:  quiet,
	}
}

// ReportProgress reports deletion progress to console
func (r *ConsoleProgressReporter) ReportProgress(progress DeletionProgress) {
	if r.quiet {
		return
	}
	
	elapsed := time.Since(progress.StartTime)
	percentage := float64(progress.ProcessedUploads) / float64(progress.TotalUploads) * 100
	
	fmt.Fprintf(r.writer, "\rProgress: %d/%d (%.1f%%) | Success: %d | Failed: %d | Current: %s | Elapsed: %v",
		progress.ProcessedUploads, progress.TotalUploads, percentage,
		progress.SuccessfulDeletes, progress.FailedDeletes,
		progress.CurrentBucket, elapsed.Truncate(time.Second))
}

// ReportCompletion reports deletion completion to console
func (r *ConsoleProgressReporter) ReportCompletion(result DeletionResult) {
	if r.quiet {
		return
	}
	
	fmt.Fprintf(r.writer, "\n\nDeletion completed:\n")
	fmt.Fprintf(r.writer, "  Total processed: %d\n", result.TotalProcessed)
	fmt.Fprintf(r.writer, "  Successful deletions: %d\n", result.SuccessfulDeletes)
	fmt.Fprintf(r.writer, "  Failed deletions: %d\n", result.FailedDeletes)
	fmt.Fprintf(r.writer, "  Storage freed: %s\n", FormatBytes(result.StorageFreed))
	fmt.Fprintf(r.writer, "  Duration: %v\n", result.Duration.Truncate(time.Second))
	
	if len(result.Errors) > 0 {
		fmt.Fprintf(r.writer, "\nErrors encountered:\n")
		for i, err := range result.Errors {
			if i >= 10 { // Limit error display
				fmt.Fprintf(r.writer, "  ... and %d more errors\n", len(result.Errors)-10)
				break
			}
			fmt.Fprintf(r.writer, "  %s/%s: %v\n", err.Upload.Bucket, err.Upload.Key, err.Error)
		}
	}
}



// UploadService implements the interfaces.UploadService interface
type UploadService struct {
	client        S3UploadClientInterface
	bucketService interfaces.BucketService
	dryRunService interfaces.DryRunService
	concurrency   int
	progressReporter ProgressReporter
	confirmationReader io.Reader
	outputWriter io.Writer
	regionalClients map[string]S3UploadClientInterface
	clientMutex     sync.RWMutex
}

// NewUploadService creates a new UploadService instance
func NewUploadService(client *awsclient.S3Client, bucketService interfaces.BucketService, dryRunService interfaces.DryRunService) interfaces.UploadService {
	return &UploadService{
		client:             client,
		bucketService:      bucketService,
		dryRunService:      dryRunService,
		concurrency:        10, // Default concurrency
		progressReporter:   NewConsoleProgressReporter(os.Stdout, false),
		confirmationReader: os.Stdin,
		outputWriter:       os.Stdout,
		regionalClients:    make(map[string]S3UploadClientInterface),
	}
}

// NewUploadServiceWithConcurrency creates a new UploadService instance with custom concurrency
func NewUploadServiceWithConcurrency(client *awsclient.S3Client, bucketService interfaces.BucketService, dryRunService interfaces.DryRunService, concurrency int) interfaces.UploadService {
	return &UploadService{
		client:             client,
		bucketService:      bucketService,
		dryRunService:      dryRunService,
		concurrency:        concurrency,
		progressReporter:   NewConsoleProgressReporter(os.Stdout, false),
		confirmationReader: os.Stdin,
		outputWriter:       os.Stdout,
	}
}

// NewUploadServiceWithOptions creates a new UploadService instance with all options
func NewUploadServiceWithOptions(client *awsclient.S3Client, bucketService interfaces.BucketService, dryRunService interfaces.DryRunService, concurrency int, progressReporter ProgressReporter, confirmationReader io.Reader, outputWriter io.Writer) interfaces.UploadService {
	if progressReporter == nil {
		progressReporter = NewConsoleProgressReporter(os.Stdout, false)
	}
	if confirmationReader == nil {
		confirmationReader = os.Stdin
	}
	if outputWriter == nil {
		outputWriter = os.Stdout
	}
	
	return &UploadService{
		client:             client,
		bucketService:      bucketService,
		dryRunService:      dryRunService,
		concurrency:        concurrency,
		progressReporter:   progressReporter,
		confirmationReader: confirmationReader,
		outputWriter:       outputWriter,
	}
}

// ListUploads retrieves all incomplete multipart uploads
func (s *UploadService) ListUploads(ctx context.Context, opts pkgtypes.ListOptions) ([]pkgtypes.MultipartUpload, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid list options: %w", err)
	}

	// If a specific bucket is requested, list uploads for that bucket only
	if opts.BucketName != "" {
		region, err := s.bucketService.GetBucketRegion(ctx, opts.BucketName)
		if err != nil {
			return nil, fmt.Errorf("failed to get region for bucket %s: %w", opts.BucketName, err)
		}
		
		bucket := pkgtypes.Bucket{
			Name:   opts.BucketName,
			Region: region,
		}
		
		return s.listUploadsForBucket(ctx, bucket, opts)
	}

	// Get all buckets (don't filter by region yet)
	buckets, err := s.bucketService.ListBuckets(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	
	// Filter by region if specified
	if opts.Region != "" {
		var filteredBuckets []pkgtypes.Bucket
		for _, bucket := range buckets {
			if bucket.Region == opts.Region {
				filteredBuckets = append(filteredBuckets, bucket)
			}
		}
		buckets = filteredBuckets
	}

	// Process buckets concurrently
	return s.listUploadsForBuckets(ctx, buckets, opts)
}

// listUploadsForBuckets processes multiple buckets concurrently
func (s *UploadService) listUploadsForBuckets(ctx context.Context, buckets []pkgtypes.Bucket, opts pkgtypes.ListOptions) ([]pkgtypes.MultipartUpload, error) {
	type bucketResult struct {
		uploads []pkgtypes.MultipartUpload
		err     error
	}

	resultChan := make(chan bucketResult, len(buckets))
	semaphore := make(chan struct{}, s.concurrency)

	var wg sync.WaitGroup

	// Process each bucket concurrently
	for _, bucket := range buckets {
		wg.Add(1)
		go func(b pkgtypes.Bucket) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			uploads, err := s.listUploadsForBucket(ctx, b, opts)
			resultChan <- bucketResult{uploads: uploads, err: err}
		}(bucket)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allUploads []pkgtypes.MultipartUpload
	var errors []error

	for result := range resultChan {
		if result.err != nil {
			errors = append(errors, result.err)
			continue
		}
		allUploads = append(allUploads, result.uploads...)
	}

	// Apply pagination if specified
	if opts.Offset > 0 || opts.MaxResults > 0 {
		allUploads = s.applyPagination(allUploads, opts)
	}

	// Return partial results even if some buckets failed
	if len(errors) > 0 {
		// Log first few errors for debugging
		for i, err := range errors {
			if i >= 3 { // Limit to first 3 errors
				break
			}
			fmt.Fprintf(os.Stderr, "Bucket access error %d: %v\n", i+1, err)
		}
		if len(errors) > 3 {
			fmt.Fprintf(os.Stderr, "... and %d more errors\n", len(errors)-3)
		}
		return allUploads, fmt.Errorf("failed to list uploads for some buckets: %d errors occurred", len(errors))
	}

	return allUploads, nil
}

// listUploadsForBucket lists uploads for a single bucket
func (s *UploadService) listUploadsForBucket(ctx context.Context, bucket pkgtypes.Bucket, opts pkgtypes.ListOptions) ([]pkgtypes.MultipartUpload, error) {
	var allUploads []pkgtypes.MultipartUpload
	var keyMarker *string
	var uploadIDMarker *string

	for {
		input := &s3.ListMultipartUploadsInput{
			Bucket: aws.String(bucket.Name),
		}

		// Set pagination markers if available
		if keyMarker != nil {
			input.KeyMarker = keyMarker
		}
		if uploadIDMarker != nil {
			input.UploadIdMarker = uploadIDMarker
		}

		// Set max keys for pagination
		if opts.MaxResults > 0 {
			remaining := opts.MaxResults - len(allUploads)
			if remaining <= 0 {
				break
			}
			input.MaxUploads = aws.Int32(int32(remaining))
		}

		// Use region-specific client for this bucket
		regionalClient, err := s.getRegionalClient(ctx, bucket.Region)
		if err != nil {
			return nil, fmt.Errorf("failed to create regional client for bucket %s: %w", bucket.Name, err)
		}
		
		output, err := regionalClient.ListMultipartUploads(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list multipart uploads for bucket %s: %w", bucket.Name, err)
		}

		// Convert AWS uploads to our types
		for _, upload := range output.Uploads {
			if upload.Key == nil || upload.UploadId == nil || upload.Initiated == nil {
				continue
			}

			// Get storage class, default to STANDARD if not specified
			storageClass := "STANDARD"
			if upload.StorageClass != "" {
				storageClass = string(upload.StorageClass)
			}

			multipartUpload := pkgtypes.MultipartUpload{
				Bucket:       bucket.Name,
				Key:          *upload.Key,
				UploadID:     *upload.UploadId,
				Initiated:    *upload.Initiated,
				StorageClass: storageClass,
				Region:       bucket.Region,
				Size:         0, // Will be calculated separately if needed
			}

			allUploads = append(allUploads, multipartUpload)
		}

		// Check if there are more results
		if output.IsTruncated == nil || !*output.IsTruncated {
			break
		}

		// Set markers for next iteration
		keyMarker = output.NextKeyMarker
		uploadIDMarker = output.NextUploadIdMarker
	}

	return allUploads, nil
}

// applyPagination applies offset and limit to the results
func (s *UploadService) applyPagination(uploads []pkgtypes.MultipartUpload, opts pkgtypes.ListOptions) []pkgtypes.MultipartUpload {
	start := opts.Offset
	if start >= len(uploads) {
		return []pkgtypes.MultipartUpload{}
	}

	end := len(uploads)
	if opts.MaxResults > 0 && start+opts.MaxResults < end {
		end = start + opts.MaxResults
	}

	return uploads[start:end]
}

// GetUploadSize calculates the size of an incomplete upload
func (s *UploadService) GetUploadSize(ctx context.Context, upload pkgtypes.MultipartUpload) (int64, error) {
	if err := upload.Validate(); err != nil {
		return 0, fmt.Errorf("invalid upload: %w", err)
	}

	var totalSize int64
	var partNumberMarker *string

	for {
		input := &s3.ListPartsInput{
			Bucket:   aws.String(upload.Bucket),
			Key:      aws.String(upload.Key),
			UploadId: aws.String(upload.UploadID),
		}

		if partNumberMarker != nil {
			input.PartNumberMarker = partNumberMarker
		}

		output, err := s.client.ListParts(ctx, input)
		if err != nil {
			return 0, fmt.Errorf("failed to list parts for upload %s in bucket %s: %w", upload.UploadID, upload.Bucket, err)
		}

		// Sum up the sizes of all parts
		for _, part := range output.Parts {
			if part.Size != nil {
				totalSize += *part.Size
			}
		}

		// Check if there are more parts
		if output.IsTruncated == nil || !*output.IsTruncated {
			break
		}

		partNumberMarker = output.NextPartNumberMarker
	}

	return totalSize, nil
}

// DeleteUpload deletes a specific multipart upload
func (s *UploadService) DeleteUpload(ctx context.Context, upload pkgtypes.MultipartUpload) error {
	if err := upload.Validate(); err != nil {
		return fmt.Errorf("invalid upload: %w", err)
	}

	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(upload.Bucket),
		Key:      aws.String(upload.Key),
		UploadId: aws.String(upload.UploadID),
	}

	_, err := s.client.AbortMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload %s in bucket %s: %w", upload.UploadID, upload.Bucket, err)
	}

	return nil
}

// DeleteUploads deletes multiple uploads with options and safety features
func (s *UploadService) DeleteUploads(ctx context.Context, uploads []pkgtypes.MultipartUpload, opts pkgtypes.DeleteOptions) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid delete options: %w", err)
	}

	// Filter uploads based on options
	filteredUploads := s.filterUploadsForDeletion(uploads, opts)

	if len(filteredUploads) == 0 {
		return fmt.Errorf("no uploads match the specified criteria")
	}

	// Calculate total size for reporting
	var totalSize int64
	for _, upload := range filteredUploads {
		totalSize += upload.Size
	}

	if opts.DryRun {
		// Use the dry-run service for comprehensive dry-run functionality
		if s.dryRunService != nil {
			result, err := s.dryRunService.SimulateDeletion(ctx, filteredUploads, opts)
			if err != nil {
				return fmt.Errorf("dry-run simulation failed: %w", err)
			}
			s.reportDryRunResultsFromService(result)
		} else {
			// Fallback to legacy dry-run reporting
			s.reportDryRunResults(filteredUploads, totalSize)
		}
		return nil
	}

	// Show confirmation prompt unless --force is used
	if !opts.Force {
		confirmed, err := s.promptForConfirmation(filteredUploads, totalSize)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			return fmt.Errorf("deletion cancelled by user")
		}
	}

	// Delete uploads with progress reporting
	return s.deleteUploadsWithProgress(ctx, filteredUploads)
}

// filterUploadsForDeletion filters uploads based on delete options
func (s *UploadService) filterUploadsForDeletion(uploads []pkgtypes.MultipartUpload, opts pkgtypes.DeleteOptions) []pkgtypes.MultipartUpload {
	var filtered []pkgtypes.MultipartUpload

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

		// Filter by size if specified (requires size to be calculated)
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

// promptForConfirmation prompts the user for confirmation before deletion
func (s *UploadService) promptForConfirmation(uploads []pkgtypes.MultipartUpload, totalSize int64) (bool, error) {
	// Group uploads by bucket for summary
	bucketCounts := make(map[string]int)
	for _, upload := range uploads {
		bucketCounts[upload.Bucket]++
	}

	fmt.Fprintf(s.outputWriter, "\nDeletion Summary:\n")
	fmt.Fprintf(s.outputWriter, "  Total uploads to delete: %d\n", len(uploads))
	fmt.Fprintf(s.outputWriter, "  Total storage to free: %s\n", FormatBytes(totalSize))
	fmt.Fprintf(s.outputWriter, "  Buckets affected: %d\n", len(bucketCounts))
	
	if len(bucketCounts) <= 10 {
		fmt.Fprintf(s.outputWriter, "\nUploads per bucket:\n")
		for bucket, count := range bucketCounts {
			fmt.Fprintf(s.outputWriter, "  %s: %d uploads\n", bucket, count)
		}
	}

	fmt.Fprintf(s.outputWriter, "\nThis action cannot be undone. Are you sure you want to proceed? (y/N): ")

	reader := bufio.NewReader(s.confirmationReader)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// reportDryRunResults reports what would be deleted in a dry run (legacy method)
func (s *UploadService) reportDryRunResults(uploads []pkgtypes.MultipartUpload, totalSize int64) {
	// Group uploads by bucket for summary
	bucketCounts := make(map[string]int)
	bucketSizes := make(map[string]int64)
	
	for _, upload := range uploads {
		bucketCounts[upload.Bucket]++
		bucketSizes[upload.Bucket] += upload.Size
	}

	fmt.Fprintf(s.outputWriter, "\nDry Run Results:\n")
	fmt.Fprintf(s.outputWriter, "  Total uploads that would be deleted: %d\n", len(uploads))
	fmt.Fprintf(s.outputWriter, "  Total storage that would be freed: %s\n", FormatBytes(totalSize))
	fmt.Fprintf(s.outputWriter, "  Buckets that would be affected: %d\n", len(bucketCounts))
	
	fmt.Fprintf(s.outputWriter, "\nBreakdown by bucket:\n")
	for bucket, count := range bucketCounts {
		size := bucketSizes[bucket]
		fmt.Fprintf(s.outputWriter, "  %s: %d uploads (%s)\n", bucket, count, FormatBytes(size))
	}

	fmt.Fprintf(s.outputWriter, "\nTo execute this deletion, run the same command without --dry-run\n")
}

// reportDryRunResultsFromService reports comprehensive dry-run results from the dry-run service
func (s *UploadService) reportDryRunResultsFromService(result pkgtypes.DryRunResult) {
	fmt.Fprintf(s.outputWriter, "\nDry Run Results:\n")
	fmt.Fprintf(s.outputWriter, "  Total uploads that would be deleted: %d\n", result.TotalUploads)
	fmt.Fprintf(s.outputWriter, "  Total storage that would be freed: %s\n", FormatBytes(result.TotalSize))
	fmt.Fprintf(s.outputWriter, "  Estimated monthly cost savings: $%.2f %s\n", result.EstimatedSavings, result.Currency)
	fmt.Fprintf(s.outputWriter, "  Buckets that would be affected: %d\n", len(result.UploadsByBucket))
	
	if len(result.UploadsByBucket) > 0 {
		fmt.Fprintf(s.outputWriter, "\nBreakdown by bucket:\n")
		for bucket, count := range result.UploadsByBucket {
			size := result.SizeByBucket[bucket]
			savings := result.SavingsByBucket[bucket]
			fmt.Fprintf(s.outputWriter, "  %s: %d uploads (%s, $%.2f/month)\n", 
				bucket, count, FormatBytes(size), savings)
		}
	}
	
	if len(result.UploadsByRegion) > 1 {
		fmt.Fprintf(s.outputWriter, "\nBreakdown by region:\n")
		for region, count := range result.UploadsByRegion {
			size := result.SizeByRegion[region]
			savings := result.SavingsByRegion[region]
			fmt.Fprintf(s.outputWriter, "  %s: %d uploads (%s, $%.2f/month)\n", 
				region, count, FormatBytes(size), savings)
		}
	}
	
	if len(result.UploadsByStorageClass) > 1 {
		fmt.Fprintf(s.outputWriter, "\nBreakdown by storage class:\n")
		for storageClass, count := range result.UploadsByStorageClass {
			size := result.SizeByStorageClass[storageClass]
			savings := result.SavingsByStorageClass[storageClass]
			fmt.Fprintf(s.outputWriter, "  %s: %d uploads (%s, $%.2f/month)\n", 
				storageClass, count, FormatBytes(size), savings)
		}
	}
	
	if result.Filters != "" {
		fmt.Fprintf(s.outputWriter, "\nFilters applied: %s\n", result.Filters)
	}
	
	fmt.Fprintf(s.outputWriter, "\nTo execute this deletion, run the same command without --dry-run\n")
}

// deleteUploadsWithProgress deletes uploads with progress reporting
func (s *UploadService) deleteUploadsWithProgress(ctx context.Context, uploads []pkgtypes.MultipartUpload) error {
	if len(uploads) == 0 {
		return nil
	}

	startTime := time.Now()
	progress := DeletionProgress{
		TotalUploads:      len(uploads),
		ProcessedUploads:  0,
		SuccessfulDeletes: 0,
		FailedDeletes:     0,
		StartTime:         startTime,
		Errors:            make([]DeletionError, 0),
	}

	type deleteResult struct {
		upload pkgtypes.MultipartUpload
		err    error
	}

	resultChan := make(chan deleteResult, len(uploads))
	semaphore := make(chan struct{}, s.concurrency)

	var wg sync.WaitGroup
	var processedCount int64
	var successCount int64
	var failedCount int64
	var errors []DeletionError
	var errorsMutex sync.Mutex

	// Progress reporting goroutine
	progressTicker := time.NewTicker(1 * time.Second)
	defer progressTicker.Stop()

	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		for {
			select {
			case <-progressTicker.C:
				progress.ProcessedUploads = atomic.LoadInt64(&processedCount)
				progress.SuccessfulDeletes = atomic.LoadInt64(&successCount)
				progress.FailedDeletes = atomic.LoadInt64(&failedCount)
				s.progressReporter.ReportProgress(progress)
			case <-ctx.Done():
				return
			case <-progressDone:
				return
			}
		}
	}()

	// Delete each upload concurrently
	for _, upload := range uploads {
		wg.Add(1)
		go func(u pkgtypes.MultipartUpload) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Update current bucket for progress reporting
			progress.CurrentBucket = u.Bucket

			err := s.DeleteUpload(ctx, u)
			
			atomic.AddInt64(&processedCount, 1)
			
			if err != nil {
				atomic.AddInt64(&failedCount, 1)
				errorsMutex.Lock()
				errors = append(errors, DeletionError{
					Upload: u,
					Error:  err,
					Time:   time.Now(),
				})
				errorsMutex.Unlock()
			} else {
				atomic.AddInt64(&successCount, 1)
			}

			resultChan <- deleteResult{upload: u, err: err}
		}(upload)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
		progressDone <- struct{}{}
	}()

	// Collect results
	var totalStorageFreed int64
	for result := range resultChan {
		if result.err == nil {
			totalStorageFreed += result.upload.Size
		}
	}

	// Final progress report
	progress.ProcessedUploads = atomic.LoadInt64(&processedCount)
	progress.SuccessfulDeletes = atomic.LoadInt64(&successCount)
	progress.FailedDeletes = atomic.LoadInt64(&failedCount)
	s.progressReporter.ReportProgress(progress)

	// Report completion
	result := DeletionResult{
		TotalProcessed:    len(uploads),
		SuccessfulDeletes: int(atomic.LoadInt64(&successCount)),
		FailedDeletes:     int(atomic.LoadInt64(&failedCount)),
		StorageFreed:      totalStorageFreed,
		Duration:          time.Since(startTime),
		Errors:            errors,
	}
	s.progressReporter.ReportCompletion(result)

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete %d out of %d uploads", len(errors), len(uploads))
	}

	return nil
}

// deleteUploadsParallel deletes uploads in parallel (legacy method for backward compatibility)
func (s *UploadService) deleteUploadsParallel(ctx context.Context, uploads []pkgtypes.MultipartUpload) error {
	return s.deleteUploadsWithProgress(ctx, uploads)
}

// getRegionalClient returns a region-specific S3 client, creating it if needed
func (s *UploadService) getRegionalClient(ctx context.Context, region string) (S3UploadClientInterface, error) {
	// Initialize map if nil
	if s.regionalClients == nil {
		s.clientMutex.Lock()
		if s.regionalClients == nil {
			s.regionalClients = make(map[string]S3UploadClientInterface)
		}
		s.clientMutex.Unlock()
	}
	
	s.clientMutex.RLock()
	if client, exists := s.regionalClients[region]; exists {
		s.clientMutex.RUnlock()
		return client, nil
	}
	s.clientMutex.RUnlock()

	// Create new regional client
	s.clientMutex.Lock()
	defer s.clientMutex.Unlock()

	// Double-check after acquiring write lock
	if client, exists := s.regionalClients[region]; exists {
		return client, nil
	}

	// Create AWS client wrapper for this region
	clientConfig := awsclient.ClientConfig{
		Region:    region,
		RateLimit: 10.0,
	}
	
	client, err := awsclient.NewS3Client(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client for region %s: %w", region, err)
	}

	s.regionalClients[region] = client
	return client, nil
}