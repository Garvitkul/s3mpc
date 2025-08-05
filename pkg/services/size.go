package services

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/s3mpc/s3mpc/pkg/interfaces"
	"github.com/s3mpc/s3mpc/pkg/types"
)

// SizeService handles size calculation and reporting operations
type SizeService struct {
	uploadService interfaces.UploadService
	concurrency   int
}

// NewSizeService creates a new SizeService instance
func NewSizeService(uploadService interfaces.UploadService) *SizeService {
	return &SizeService{
		uploadService: uploadService,
		concurrency:   10, // Default concurrency
	}
}

// NewSizeServiceWithConcurrency creates a new SizeService instance with custom concurrency
func NewSizeServiceWithConcurrency(uploadService interfaces.UploadService, concurrency int) *SizeService {
	return &SizeService{
		uploadService: uploadService,
		concurrency:   concurrency,
	}
}

// CalculateTotalSize calculates the total size of all incomplete multipart uploads
func (s *SizeService) CalculateTotalSize(ctx context.Context, opts types.ListOptions) (*types.SizeReport, error) {
	// Get all uploads
	uploads, err := s.uploadService.ListUploads(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list uploads: %w", err)
	}

	// Calculate sizes for all uploads concurrently
	uploadsWithSizes, inaccessibleBuckets, err := s.calculateUploadSizes(ctx, uploads)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate upload sizes: %w", err)
	}

	// Generate size report
	report := s.generateSizeReport(uploadsWithSizes, inaccessibleBuckets)
	
	if err := report.Validate(); err != nil {
		return nil, fmt.Errorf("invalid size report: %w", err)
	}

	return report, nil
}

// CalculateBucketSizes calculates sizes grouped by bucket
func (s *SizeService) CalculateBucketSizes(ctx context.Context, opts types.ListOptions) (*types.SizeReport, error) {
	report, err := s.CalculateTotalSize(ctx, opts)
	if err != nil {
		return nil, err
	}

	// The report already contains per-bucket breakdown in ByBucket field
	return report, nil
}

// calculateUploadSizes calculates sizes for all uploads concurrently
func (s *SizeService) calculateUploadSizes(ctx context.Context, uploads []types.MultipartUpload) ([]types.MultipartUpload, []string, error) {
	if len(uploads) == 0 {
		return uploads, nil, nil
	}

	type uploadResult struct {
		upload               types.MultipartUpload
		err                  error
		inaccessibleBucket   string
	}

	resultChan := make(chan uploadResult, len(uploads))
	semaphore := make(chan struct{}, s.concurrency)

	var wg sync.WaitGroup

	// Calculate size for each upload concurrently
	for _, upload := range uploads {
		wg.Add(1)
		go func(u types.MultipartUpload) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			size, err := s.uploadService.GetUploadSize(ctx, u)
			if err != nil {
				// Check if this is an access denied error for the bucket
				resultChan <- uploadResult{
					upload:             u,
					err:                err,
					inaccessibleBucket: u.Bucket,
				}
				return
			}

			// Update upload with calculated size
			u.Size = size
			resultChan <- uploadResult{upload: u}
		}(upload)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var uploadsWithSizes []types.MultipartUpload
	var inaccessibleBuckets []string
	var errors []error

	bucketErrorMap := make(map[string]bool) // Track which buckets had errors

	for result := range resultChan {
		if result.err != nil {
			errors = append(errors, result.err)
			if result.inaccessibleBucket != "" && !bucketErrorMap[result.inaccessibleBucket] {
				inaccessibleBuckets = append(inaccessibleBuckets, result.inaccessibleBucket)
				bucketErrorMap[result.inaccessibleBucket] = true
			}
			continue
		}
		uploadsWithSizes = append(uploadsWithSizes, result.upload)
	}

	// Return partial results even if some uploads failed
	return uploadsWithSizes, inaccessibleBuckets, nil
}

// generateSizeReport creates a comprehensive size report from uploads
func (s *SizeService) generateSizeReport(uploads []types.MultipartUpload, inaccessibleBuckets []string) *types.SizeReport {
	report := &types.SizeReport{
		ByStorageClass:      make(map[string]int64),
		ByBucket:            make(map[string]int64),
		InaccessibleBuckets: inaccessibleBuckets,
	}

	// Aggregate data
	for _, upload := range uploads {
		report.TotalSize += upload.Size
		report.TotalCount++

		// Aggregate by storage class
		report.ByStorageClass[upload.StorageClass] += upload.Size

		// Aggregate by bucket
		report.ByBucket[upload.Bucket] += upload.Size
	}

	return report
}

// GetSortedBucketSizes returns bucket sizes sorted by size in descending order
func (s *SizeService) GetSortedBucketSizes(report *types.SizeReport) []interfaces.BucketSize {
	var bucketSizes []interfaces.BucketSize

	for bucket, size := range report.ByBucket {
		bucketSizes = append(bucketSizes, interfaces.BucketSize{
			Bucket: bucket,
			Size:   size,
		})
	}

	// Sort by size in descending order
	sort.Slice(bucketSizes, func(i, j int) bool {
		return bucketSizes[i].Size > bucketSizes[j].Size
	})

	return bucketSizes
}



// FormatSize formats a size in bytes to human-readable format
func FormatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	if exp >= len(units) {
		// For extremely large sizes, use the largest unit
		exp = len(units) - 1
		div = int64(1)
		for i := 0; i <= exp; i++ {
			div *= unit
		}
	}

	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// ParseSize parses a human-readable size string to bytes
func ParseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("size string cannot be empty")
	}

	var value float64
	var unit string

	// Parse the size string
	n, err := fmt.Sscanf(sizeStr, "%f %s", &value, &unit)
	if err != nil || n != 2 {
		// Try parsing without space
		n, err = fmt.Sscanf(sizeStr, "%f%s", &value, &unit)
		if err != nil || n != 2 {
			return 0, fmt.Errorf("invalid size format: %s", sizeStr)
		}
	}

	if value < 0 {
		return 0, fmt.Errorf("size cannot be negative: %s", sizeStr)
	}

	// Convert unit to bytes
	multiplier := int64(1)
	switch unit {
	case "B", "b":
		multiplier = 1
	case "KB", "kb", "K", "k":
		multiplier = 1024
	case "MB", "mb", "M", "m":
		multiplier = 1024 * 1024
	case "GB", "gb", "G", "g":
		multiplier = 1024 * 1024 * 1024
	case "TB", "tb", "T", "t":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "PB", "pb", "P", "p":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}

	bytes := int64(value * float64(multiplier))
	return bytes, nil
}

// GetStorageClassBreakdown returns a formatted breakdown by storage class
func (s *SizeService) GetStorageClassBreakdown(report *types.SizeReport) []interfaces.StorageClassSize {
	var breakdown []interfaces.StorageClassSize

	for storageClass, size := range report.ByStorageClass {
		breakdown = append(breakdown, interfaces.StorageClassSize{
			StorageClass: storageClass,
			Size:         size,
			Formatted:    FormatSize(size),
		})
	}

	// Sort by size in descending order
	sort.Slice(breakdown, func(i, j int) bool {
		return breakdown[i].Size > breakdown[j].Size
	})

	return breakdown
}

// StorageClassSize represents a storage class and its total size
