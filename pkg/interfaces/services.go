package interfaces

import (
	"context"

	"github.com/Garvitkul/s3mpc/pkg/types"
)

// UploadService handles multipart upload operations
type UploadService interface {
	// ListUploads retrieves all incomplete multipart uploads
	ListUploads(ctx context.Context, opts types.ListOptions) ([]types.MultipartUpload, error)
	
	// DeleteUpload deletes a specific multipart upload
	DeleteUpload(ctx context.Context, upload types.MultipartUpload) error
	
	// GetUploadSize calculates the size of an incomplete upload
	GetUploadSize(ctx context.Context, upload types.MultipartUpload) (int64, error)
	
	// DeleteUploads deletes multiple uploads with options
	DeleteUploads(ctx context.Context, uploads []types.MultipartUpload, opts types.DeleteOptions) error
}

// BucketService handles S3 bucket operations
type BucketService interface {
	// ListBuckets retrieves all accessible S3 buckets
	ListBuckets(ctx context.Context, region string) ([]types.Bucket, error)
	
	// GetBucketRegion retrieves the region for a specific bucket
	GetBucketRegion(ctx context.Context, bucketName string) (string, error)
	
	// ListBucketsInRegion retrieves buckets in a specific region
	ListBucketsInRegion(ctx context.Context, region string) ([]types.Bucket, error)
	
	// ClearRegionCache clears the region cache (useful for testing)
	ClearRegionCache()
	
	// GetCacheStats returns cache statistics (useful for monitoring)
	GetCacheStats() map[string]interface{}
}

// CostCalculator handles pricing calculations
type CostCalculator interface {
	// CalculateStorageCost calculates storage costs for uploads
	CalculateStorageCost(ctx context.Context, uploads []types.MultipartUpload) (types.CostBreakdown, error)
	
	// GetRegionalPricing retrieves pricing for a region and storage class
	GetRegionalPricing(ctx context.Context, region, storageClass string) (float64, error)
	
	// EstimateSavings calculates potential cost savings from deletion
	EstimateSavings(ctx context.Context, uploads []types.MultipartUpload) (float64, error)
}

// AgeService handles age analysis and distribution calculations
type AgeService interface {
	// CalculateAgeDistribution calculates age distribution of uploads
	CalculateAgeDistribution(ctx context.Context, uploads []types.MultipartUpload) (types.AgeDistribution, error)
	
	// GetAgeDistributionForBucket calculates age distribution for a specific bucket
	GetAgeDistributionForBucket(ctx context.Context, uploads []types.MultipartUpload, bucketName string) (types.AgeDistribution, error)
	
	// IsOlderThanSevenDays checks if an upload is older than 7 days (for highlighting)
	IsOlderThanSevenDays(upload types.MultipartUpload) bool
}

// FilterEngine handles query parsing and filtering
type FilterEngine interface {
	// ParseFilter parses a filter string into a structured filter
	ParseFilter(filterStr string) (Filter, error)
	
	// ApplyFilter applies a filter to a list of uploads
	ApplyFilter(uploads []types.MultipartUpload, filter Filter) []types.MultipartUpload
	
	// ValidateFilter validates filter syntax
	ValidateFilter(filterStr string) error
}

// DryRunService handles dry-run operations and result generation
type DryRunService interface {
	// SimulateDeletion simulates deletion without executing it
	SimulateDeletion(ctx context.Context, uploads []types.MultipartUpload, opts types.DeleteOptions) (types.DryRunResult, error)
	
	// SaveDryRunResult saves dry-run results to a file
	SaveDryRunResult(result types.DryRunResult, filename string) error
	
	// GenerateFilename generates a filename for dry-run results
	GenerateFilename(command string, format string) string
}

// Filter represents parsed filter criteria
type Filter struct {
	Age          *AgeFilter
	Size         *SizeFilter
	StorageClass *StringFilter
	Region       *StringFilter
	Bucket       *StringFilter
}

// AgeFilter represents age-based filtering
type AgeFilter struct {
	Operator string // >, <, >=, <=, =, !=
	Value    string // e.g., "7d", "1w", "1m"
}

// SizeFilter represents size-based filtering
type SizeFilter struct {
	Operator string // >, <, >=, <=, =, !=
	Value    string // e.g., "100MB", "1GB"
}

// StringFilter represents string-based filtering
type StringFilter struct {
	Operator string // =, !=
	Value    string
}

// ExportService handles data export operations
type ExportService interface {
	// ExportToCSV exports uploads to CSV format
	ExportToCSV(ctx context.Context, uploads []types.MultipartUpload, filename string) error
	
	// ExportToJSON exports uploads to JSON format
	ExportToJSON(ctx context.Context, uploads []types.MultipartUpload, filename string) error
	
	// GenerateExportFilename generates a filename for export results
	GenerateExportFilename(command string, format string) string
	
	// StreamExportToCSV exports large datasets to CSV with streaming
	StreamExportToCSV(ctx context.Context, uploads <-chan types.MultipartUpload, filename string) error
	
	// StreamExportToJSON exports large datasets to JSON with streaming
	StreamExportToJSON(ctx context.Context, uploads <-chan types.MultipartUpload, filename string) error
}

// OutputFormatter handles different output formats for console display
type OutputFormatter interface {
	// FormatUploads formats uploads for human-readable console output
	FormatUploads(uploads []types.MultipartUpload, showDetails bool) string
	
	// FormatSizeReport formats size report for console output
	FormatSizeReport(report types.SizeReport) string
	
	// FormatCostBreakdown formats cost breakdown for console output
	FormatCostBreakdown(breakdown types.CostBreakdown) string
	
	// FormatAgeDistribution formats age distribution for console output
	FormatAgeDistribution(distribution types.AgeDistribution) string
	
	// FormatJSON formats any data structure as JSON
	FormatJSON(data interface{}) (string, error)
	
	// FormatTable formats data as a table with headers and rows
	FormatTable(headers []string, rows [][]string) string
}

// SizeService handles size calculation and reporting
type SizeService interface {
	// CalculateTotalSize calculates the total size of all incomplete multipart uploads
	CalculateTotalSize(ctx context.Context, opts types.ListOptions) (*types.SizeReport, error)
	
	// CalculateBucketSizes calculates sizes grouped by bucket
	CalculateBucketSizes(ctx context.Context, opts types.ListOptions) (*types.SizeReport, error)
	
	// GetSortedBucketSizes returns bucket sizes sorted by size in descending order
	GetSortedBucketSizes(report *types.SizeReport) []BucketSize
	
	// GetStorageClassBreakdown returns a formatted breakdown by storage class
	GetStorageClassBreakdown(report *types.SizeReport) []StorageClassSize
}

// BucketSize represents a bucket with its size for sorting
type BucketSize struct {
	Bucket string `json:"bucket"`
	Size   int64  `json:"size"`
}

// StorageClassSize represents a storage class with its size
type StorageClassSize struct {
	StorageClass string `json:"storage_class"`
	Size         int64  `json:"size"`
	Formatted    string `json:"formatted"`
}