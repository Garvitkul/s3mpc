package types

import (
	"fmt"
	"strings"
	"time"
)

// MultipartUpload represents an incomplete multipart upload
type MultipartUpload struct {
	Bucket       string    `json:"bucket" csv:"bucket"`
	Key          string    `json:"key" csv:"key"`
	UploadID     string    `json:"upload_id" csv:"upload_id"`
	Initiated    time.Time `json:"initiated" csv:"initiated"`
	Size         int64     `json:"size" csv:"size"`
	StorageClass string    `json:"storage_class" csv:"storage_class"`
	Region       string    `json:"region" csv:"region"`
}

// Bucket represents an S3 bucket
type Bucket struct {
	Name    string            `json:"name" csv:"name"`
	Region  string            `json:"region" csv:"region"`
	Uploads []MultipartUpload `json:"uploads,omitempty" csv:"-"`
}

// SizeReport represents storage usage information
type SizeReport struct {
	TotalSize           int64             `json:"total_size" csv:"total_size"`
	TotalCount          int               `json:"total_count" csv:"total_count"`
	ByStorageClass      map[string]int64  `json:"by_storage_class" csv:"-"`
	ByBucket            map[string]int64  `json:"by_bucket" csv:"-"`
	InaccessibleBuckets []string          `json:"inaccessible_buckets" csv:"-"`
}

// CostBreakdown represents cost analysis
type CostBreakdown struct {
	TotalMonthlyCost float64            `json:"total_monthly_cost" csv:"total_monthly_cost"`
	ByRegion         map[string]float64 `json:"by_region" csv:"-"`
	ByStorageClass   map[string]float64 `json:"by_storage_class" csv:"-"`
	Currency         string             `json:"currency" csv:"currency"`
}

// AgeDistribution represents upload age analysis
type AgeDistribution struct {
	Buckets []AgeBucket `json:"buckets"`
}

// AgeBucket represents an age bucket in distribution analysis
type AgeBucket struct {
	Label     string        `json:"label"`
	MinAge    time.Duration `json:"min_age"`
	MaxAge    time.Duration `json:"max_age"`
	Count     int           `json:"count"`
	TotalSize int64         `json:"total_size"`
}

// ListOptions contains options for listing operations
type ListOptions struct {
	Region      string
	BucketName  string
	MaxResults  int
	Offset      int
}

// DeleteOptions contains options for delete operations
type DeleteOptions struct {
	DryRun      bool
	Force       bool
	OlderThan   *time.Duration
	SmallerThan *int64
	LargerThan  *int64
	BucketName  string
	Quiet       bool
}

// ExportOptions contains options for export operations
type ExportOptions struct {
	Format     string // csv, json
	OutputFile string
	Filter     string
}

// DryRunResult represents the result of a dry-run deletion operation
type DryRunResult struct {
	TotalUploads        int                    `json:"total_uploads"`
	TotalSize           int64                  `json:"total_size"`
	EstimatedSavings    float64                `json:"estimated_savings"`
	Currency            string                 `json:"currency"`
	UploadsByBucket     map[string]int         `json:"uploads_by_bucket"`
	SizeByBucket        map[string]int64       `json:"size_by_bucket"`
	SavingsByBucket     map[string]float64     `json:"savings_by_bucket"`
	UploadsByRegion     map[string]int         `json:"uploads_by_region"`
	SizeByRegion        map[string]int64       `json:"size_by_region"`
	SavingsByRegion     map[string]float64     `json:"savings_by_region"`
	UploadsByStorageClass map[string]int       `json:"uploads_by_storage_class"`
	SizeByStorageClass  map[string]int64       `json:"size_by_storage_class"`
	SavingsByStorageClass map[string]float64   `json:"savings_by_storage_class"`
	Uploads             []MultipartUpload      `json:"uploads,omitempty"`
	GeneratedAt         time.Time              `json:"generated_at"`
	Command             string                 `json:"command"`
	Filters             string                 `json:"filters,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// Validate validates a MultipartUpload struct
func (m *MultipartUpload) Validate() error {
	if strings.TrimSpace(m.Bucket) == "" {
		return ValidationError{Field: "Bucket", Message: "bucket name cannot be empty"}
	}
	
	if strings.TrimSpace(m.Key) == "" {
		return ValidationError{Field: "Key", Message: "key cannot be empty"}
	}
	
	if strings.TrimSpace(m.UploadID) == "" {
		return ValidationError{Field: "UploadID", Message: "upload ID cannot be empty"}
	}
	
	if m.Initiated.IsZero() {
		return ValidationError{Field: "Initiated", Message: "initiated time cannot be zero"}
	}
	
	if m.Size < 0 {
		return ValidationError{Field: "Size", Message: "size cannot be negative"}
	}
	
	if strings.TrimSpace(m.StorageClass) == "" {
		return ValidationError{Field: "StorageClass", Message: "storage class cannot be empty"}
	}
	
	if strings.TrimSpace(m.Region) == "" {
		return ValidationError{Field: "Region", Message: "region cannot be empty"}
	}
	
	return nil
}

// Validate validates a Bucket struct
func (b *Bucket) Validate() error {
	if strings.TrimSpace(b.Name) == "" {
		return ValidationError{Field: "Name", Message: "bucket name cannot be empty"}
	}
	
	if strings.TrimSpace(b.Region) == "" {
		return ValidationError{Field: "Region", Message: "region cannot be empty"}
	}
	
	// Validate all uploads in the bucket
	for i, upload := range b.Uploads {
		if err := upload.Validate(); err != nil {
			return fmt.Errorf("upload at index %d: %w", i, err)
		}
	}
	
	return nil
}

// Validate validates a SizeReport struct
func (s *SizeReport) Validate() error {
	if s.TotalSize < 0 {
		return ValidationError{Field: "TotalSize", Message: "total size cannot be negative"}
	}
	
	if s.TotalCount < 0 {
		return ValidationError{Field: "TotalCount", Message: "total count cannot be negative"}
	}
	
	// Validate that breakdown maps don't contain negative values
	for storageClass, size := range s.ByStorageClass {
		if size < 0 {
			return ValidationError{Field: "ByStorageClass", Message: fmt.Sprintf("size for storage class '%s' cannot be negative", storageClass)}
		}
	}
	
	for bucket, size := range s.ByBucket {
		if size < 0 {
			return ValidationError{Field: "ByBucket", Message: fmt.Sprintf("size for bucket '%s' cannot be negative", bucket)}
		}
	}
	
	return nil
}

// Validate validates a CostBreakdown struct
func (c *CostBreakdown) Validate() error {
	if c.TotalMonthlyCost < 0 {
		return ValidationError{Field: "TotalMonthlyCost", Message: "total monthly cost cannot be negative"}
	}
	
	if strings.TrimSpace(c.Currency) == "" {
		return ValidationError{Field: "Currency", Message: "currency cannot be empty"}
	}
	
	// Validate that breakdown maps don't contain negative values
	for region, cost := range c.ByRegion {
		if cost < 0 {
			return ValidationError{Field: "ByRegion", Message: fmt.Sprintf("cost for region '%s' cannot be negative", region)}
		}
	}
	
	for storageClass, cost := range c.ByStorageClass {
		if cost < 0 {
			return ValidationError{Field: "ByStorageClass", Message: fmt.Sprintf("cost for storage class '%s' cannot be negative", storageClass)}
		}
	}
	
	return nil
}

// Validate validates an AgeDistribution struct
func (a *AgeDistribution) Validate() error {
	for i, bucket := range a.Buckets {
		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("age bucket at index %d: %w", i, err)
		}
	}
	return nil
}

// Validate validates an AgeBucket struct
func (a *AgeBucket) Validate() error {
	if strings.TrimSpace(a.Label) == "" {
		return ValidationError{Field: "Label", Message: "label cannot be empty"}
	}
	
	if a.MinAge < 0 {
		return ValidationError{Field: "MinAge", Message: "min age cannot be negative"}
	}
	
	if a.MaxAge < 0 {
		return ValidationError{Field: "MaxAge", Message: "max age cannot be negative"}
	}
	
	// MaxAge of 0 means no upper limit, so skip the comparison in that case
	if a.MaxAge != 0 && a.MaxAge < a.MinAge {
		return ValidationError{Field: "MaxAge", Message: "max age cannot be less than min age"}
	}
	
	if a.Count < 0 {
		return ValidationError{Field: "Count", Message: "count cannot be negative"}
	}
	
	if a.TotalSize < 0 {
		return ValidationError{Field: "TotalSize", Message: "total size cannot be negative"}
	}
	
	return nil
}

// Validate validates ListOptions struct
func (l *ListOptions) Validate() error {
	if l.MaxResults < 0 {
		return ValidationError{Field: "MaxResults", Message: "max results cannot be negative"}
	}
	
	if l.Offset < 0 {
		return ValidationError{Field: "Offset", Message: "offset cannot be negative"}
	}
	
	return nil
}

// Validate validates DeleteOptions struct
func (d *DeleteOptions) Validate() error {
	if d.SmallerThan != nil && *d.SmallerThan < 0 {
		return ValidationError{Field: "SmallerThan", Message: "smaller than value cannot be negative"}
	}
	
	if d.LargerThan != nil && *d.LargerThan < 0 {
		return ValidationError{Field: "LargerThan", Message: "larger than value cannot be negative"}
	}
	
	// SmallerThan should be greater than LargerThan (e.g., delete files smaller than 100MB but larger than 50MB)
	if d.SmallerThan != nil && d.LargerThan != nil && *d.SmallerThan <= *d.LargerThan {
		return ValidationError{Field: "SmallerThan", Message: "smaller than value must be greater than larger than value"}
	}
	
	return nil
}

// Validate validates ExportOptions struct
func (e *ExportOptions) Validate() error {
	validFormats := map[string]bool{
		"csv":  true,
		"json": true,
	}
	
	if !validFormats[strings.ToLower(e.Format)] {
		return ValidationError{Field: "Format", Message: "format must be 'csv' or 'json'"}
	}
	
	if strings.TrimSpace(e.OutputFile) == "" {
		return ValidationError{Field: "OutputFile", Message: "output file cannot be empty"}
	}
	
	return nil
}

// Validate validates DryRunResult struct
func (d *DryRunResult) Validate() error {
	if d.TotalUploads < 0 {
		return ValidationError{Field: "TotalUploads", Message: "total uploads cannot be negative"}
	}
	
	if d.TotalSize < 0 {
		return ValidationError{Field: "TotalSize", Message: "total size cannot be negative"}
	}
	
	if d.EstimatedSavings < 0 {
		return ValidationError{Field: "EstimatedSavings", Message: "estimated savings cannot be negative"}
	}
	
	if strings.TrimSpace(d.Currency) == "" {
		return ValidationError{Field: "Currency", Message: "currency cannot be empty"}
	}
	
	if d.GeneratedAt.IsZero() {
		return ValidationError{Field: "GeneratedAt", Message: "generated at time cannot be zero"}
	}
	
	if strings.TrimSpace(d.Command) == "" {
		return ValidationError{Field: "Command", Message: "command cannot be empty"}
	}
	
	// Validate breakdown maps don't contain negative values
	for bucket, count := range d.UploadsByBucket {
		if count < 0 {
			return ValidationError{Field: "UploadsByBucket", Message: fmt.Sprintf("upload count for bucket '%s' cannot be negative", bucket)}
		}
	}
	
	for bucket, size := range d.SizeByBucket {
		if size < 0 {
			return ValidationError{Field: "SizeByBucket", Message: fmt.Sprintf("size for bucket '%s' cannot be negative", bucket)}
		}
	}
	
	for bucket, savings := range d.SavingsByBucket {
		if savings < 0 {
			return ValidationError{Field: "SavingsByBucket", Message: fmt.Sprintf("savings for bucket '%s' cannot be negative", bucket)}
		}
	}
	
	// Validate all uploads in the result
	for i, upload := range d.Uploads {
		if err := upload.Validate(); err != nil {
			return fmt.Errorf("upload at index %d: %w", i, err)
		}
	}
	
	return nil
}