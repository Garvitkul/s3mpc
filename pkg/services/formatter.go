package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/s3mpc/s3mpc/pkg/interfaces"
	"github.com/s3mpc/s3mpc/pkg/types"
)

// OutputFormatter implements the interfaces.OutputFormatter interface
type OutputFormatter struct{}

// NewOutputFormatter creates a new OutputFormatter instance
func NewOutputFormatter() interfaces.OutputFormatter {
	return &OutputFormatter{}
}

// FormatUploads formats uploads for human-readable console output
func (f *OutputFormatter) FormatUploads(uploads []types.MultipartUpload, showDetails bool) string {
	if len(uploads) == 0 {
		return "No incomplete multipart uploads found."
	}

	var result strings.Builder
	
	if showDetails {
		// Detailed format with table
		headers := []string{"Bucket", "Key", "Upload ID", "Initiated", "Age", "Size", "Storage Class", "Region"}
		var rows [][]string
		
		for _, upload := range uploads {
			age := time.Since(upload.Initiated)
			ageStr := formatDuration(age)
			sizeStr := FormatBytes(upload.Size)
			
			rows = append(rows, []string{
				upload.Bucket,
				truncateString(upload.Key, 40),
				truncateString(upload.UploadID, 20),
				upload.Initiated.Format("2006-01-02 15:04"),
				ageStr,
				sizeStr,
				upload.StorageClass,
				upload.Region,
			})
		}
		
		result.WriteString(f.FormatTable(headers, rows))
	} else {
		// Summary format
		result.WriteString(fmt.Sprintf("Found %d incomplete multipart uploads:\n\n", len(uploads)))
		
		// Group by bucket for summary
		bucketCounts := make(map[string]int)
		bucketSizes := make(map[string]int64)
		
		for _, upload := range uploads {
			bucketCounts[upload.Bucket]++
			bucketSizes[upload.Bucket] += upload.Size
		}
		
		// Sort buckets by name
		var buckets []string
		for bucket := range bucketCounts {
			buckets = append(buckets, bucket)
		}
		sort.Strings(buckets)
		
		for _, bucket := range buckets {
			count := bucketCounts[bucket]
			size := bucketSizes[bucket]
			result.WriteString(fmt.Sprintf("  %s: %d uploads (%s)\n", bucket, count, FormatBytes(size)))
		}
	}
	
	return result.String()
}

// FormatSizeReport formats size report for console output
func (f *OutputFormatter) FormatSizeReport(report types.SizeReport) string {
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("Total incomplete multipart uploads: %d\n", report.TotalCount))
	result.WriteString(fmt.Sprintf("Total storage used: %s\n\n", FormatBytes(report.TotalSize)))
	
	if len(report.ByBucket) > 0 {
		result.WriteString("Breakdown by bucket:\n")
		
		// Sort buckets by size (descending)
		type bucketSize struct {
			name string
			size int64
		}
		
		var buckets []bucketSize
		for bucket, size := range report.ByBucket {
			buckets = append(buckets, bucketSize{bucket, size})
		}
		
		sort.Slice(buckets, func(i, j int) bool {
			return buckets[i].size > buckets[j].size
		})
		
		for _, bucket := range buckets {
			percentage := float64(bucket.size) / float64(report.TotalSize) * 100
			result.WriteString(fmt.Sprintf("  %s: %s (%.1f%%)\n", bucket.name, FormatBytes(bucket.size), percentage))
		}
		result.WriteString("\n")
	}
	
	if len(report.ByStorageClass) > 0 {
		result.WriteString("Breakdown by storage class:\n")
		
		// Sort storage classes by size (descending)
		type storageClassSize struct {
			class string
			size  int64
		}
		
		var storageClasses []storageClassSize
		for class, size := range report.ByStorageClass {
			storageClasses = append(storageClasses, storageClassSize{class, size})
		}
		
		sort.Slice(storageClasses, func(i, j int) bool {
			return storageClasses[i].size > storageClasses[j].size
		})
		
		for _, sc := range storageClasses {
			percentage := float64(sc.size) / float64(report.TotalSize) * 100
			result.WriteString(fmt.Sprintf("  %s: %s (%.1f%%)\n", sc.class, FormatBytes(sc.size), percentage))
		}
		result.WriteString("\n")
	}
	
	if len(report.InaccessibleBuckets) > 0 {
		result.WriteString("Inaccessible buckets:\n")
		for _, bucket := range report.InaccessibleBuckets {
			result.WriteString(fmt.Sprintf("  %s\n", bucket))
		}
	}
	
	return result.String()
}

// FormatCostBreakdown formats cost breakdown for console output
func (f *OutputFormatter) FormatCostBreakdown(breakdown types.CostBreakdown) string {
	var result strings.Builder
	
	result.WriteString(fmt.Sprintf("Total estimated monthly cost: $%.2f %s\n\n", breakdown.TotalMonthlyCost, breakdown.Currency))
	
	if len(breakdown.ByRegion) > 0 {
		result.WriteString("Breakdown by region:\n")
		
		// Sort regions by cost (descending)
		type regionCost struct {
			region string
			cost   float64
		}
		
		var regions []regionCost
		for region, cost := range breakdown.ByRegion {
			regions = append(regions, regionCost{region, cost})
		}
		
		sort.Slice(regions, func(i, j int) bool {
			return regions[i].cost > regions[j].cost
		})
		
		for _, region := range regions {
			percentage := region.cost / breakdown.TotalMonthlyCost * 100
			result.WriteString(fmt.Sprintf("  %s: $%.2f (%.1f%%)\n", region.region, region.cost, percentage))
		}
		result.WriteString("\n")
	}
	
	if len(breakdown.ByStorageClass) > 0 {
		result.WriteString("Breakdown by storage class:\n")
		
		// Sort storage classes by cost (descending)
		type storageClassCost struct {
			class string
			cost  float64
		}
		
		var storageClasses []storageClassCost
		for class, cost := range breakdown.ByStorageClass {
			storageClasses = append(storageClasses, storageClassCost{class, cost})
		}
		
		sort.Slice(storageClasses, func(i, j int) bool {
			return storageClasses[i].cost > storageClasses[j].cost
		})
		
		for _, sc := range storageClasses {
			percentage := sc.cost / breakdown.TotalMonthlyCost * 100
			result.WriteString(fmt.Sprintf("  %s: $%.2f (%.1f%%)\n", sc.class, sc.cost, percentage))
		}
	}
	
	return result.String()
}

// FormatAgeDistribution formats age distribution for console output
func (f *OutputFormatter) FormatAgeDistribution(distribution types.AgeDistribution) string {
	var result strings.Builder
	
	if len(distribution.Buckets) == 0 {
		return "No age distribution data available."
	}
	
	result.WriteString("Age distribution of incomplete multipart uploads:\n\n")
	
	// Calculate total for percentages
	var totalCount int
	var totalSize int64
	for _, bucket := range distribution.Buckets {
		totalCount += bucket.Count
		totalSize += bucket.TotalSize
	}
	
	headers := []string{"Age Range", "Count", "Percentage", "Total Size", "Size Percentage"}
	var rows [][]string
	
	for _, bucket := range distribution.Buckets {
		countPercentage := float64(bucket.Count) / float64(totalCount) * 100
		sizePercentage := float64(bucket.TotalSize) / float64(totalSize) * 100
		
		rows = append(rows, []string{
			bucket.Label,
			fmt.Sprintf("%d", bucket.Count),
			fmt.Sprintf("%.1f%%", countPercentage),
			FormatBytes(bucket.TotalSize),
			fmt.Sprintf("%.1f%%", sizePercentage),
		})
	}
	
	result.WriteString(f.FormatTable(headers, rows))
	result.WriteString(fmt.Sprintf("\nTotal: %d uploads, %s\n", totalCount, FormatBytes(totalSize)))
	
	// Highlight uploads older than 7 days
	var oldUploads int
	var oldSize int64
	sevenDays := 7 * 24 * time.Hour
	
	for _, bucket := range distribution.Buckets {
		if bucket.MinAge >= sevenDays {
			oldUploads += bucket.Count
			oldSize += bucket.TotalSize
		}
	}
	
	if oldUploads > 0 {
		result.WriteString(fmt.Sprintf("\n⚠️  %d uploads (%.1f%%) are older than 7 days, consuming %s\n", 
			oldUploads, float64(oldUploads)/float64(totalCount)*100, FormatBytes(oldSize)))
	}
	
	return result.String()
}

// FormatJSON formats any data structure as JSON
func (f *OutputFormatter) FormatJSON(data interface{}) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(jsonData), nil
}

// FormatTable formats data as a table with headers and rows
func (f *OutputFormatter) FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}
	
	// Calculate column widths
	colWidths := make([]int, len(headers))
	
	// Initialize with header widths
	for i, header := range headers {
		colWidths[i] = len(header)
	}
	
	// Update with row data widths
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}
	
	var result strings.Builder
	
	// Format header
	for i, header := range headers {
		if i > 0 {
			result.WriteString("  ")
		}
		result.WriteString(fmt.Sprintf("%-*s", colWidths[i], header))
	}
	result.WriteString("\n")
	
	// Format separator
	for i, width := range colWidths {
		if i > 0 {
			result.WriteString("  ")
		}
		result.WriteString(strings.Repeat("-", width))
	}
	result.WriteString("\n")
	
	// Format rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				result.WriteString("  ")
			}
			if i < len(colWidths) {
				result.WriteString(fmt.Sprintf("%-*s", colWidths[i], cell))
			} else {
				result.WriteString(cell)
			}
		}
		result.WriteString("\n")
	}
	
	return result.String()
}

// Helper functions

// formatDuration formats a duration for display
func formatDuration(duration time.Duration) string {
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

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}