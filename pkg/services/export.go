package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/s3mpc/s3mpc/pkg/interfaces"
	"github.com/s3mpc/s3mpc/pkg/types"
)

// ExportService implements the interfaces.ExportService interface
type ExportService struct{}

// NewExportService creates a new ExportService instance
func NewExportService() interfaces.ExportService {
	return &ExportService{}
}

// ExportToCSV exports uploads to CSV format
func (e *ExportService) ExportToCSV(ctx context.Context, uploads []types.MultipartUpload, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	header := []string{
		"bucket",
		"key",
		"upload_id",
		"initiated",
		"age_days",
		"size",
		"storage_class",
		"region",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write upload data
	for _, upload := range uploads {
		ageDays := int(time.Since(upload.Initiated).Hours() / 24)
		
		record := []string{
			upload.Bucket,
			upload.Key,
			upload.UploadID,
			upload.Initiated.Format("2006-01-02T15:04:05Z"),
			strconv.Itoa(ageDays),
			strconv.FormatInt(upload.Size, 10),
			upload.StorageClass,
			upload.Region,
		}
		
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

// ExportToJSON exports uploads to JSON format
func (e *ExportService) ExportToJSON(ctx context.Context, uploads []types.MultipartUpload, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	// Create export data structure
	exportData := struct {
		ExportedAt time.Time                `json:"exported_at"`
		TotalCount int                      `json:"total_count"`
		Uploads    []types.MultipartUpload  `json:"uploads"`
	}{
		ExportedAt: time.Now(),
		TotalCount: len(uploads),
		Uploads:    uploads,
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(exportData); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// GenerateExportFilename generates a filename for export results
func (e *ExportService) GenerateExportFilename(command string, format string) string {
	timestamp := time.Now().Format("20060102_1504")
	
	// Sanitize command name
	sanitizedCommand := strings.ReplaceAll(command, " ", "_")
	sanitizedCommand = strings.ReplaceAll(sanitizedCommand, "-", "_")
	
	// Ensure format is lowercase
	format = strings.ToLower(format)
	if format != "csv" && format != "json" {
		format = "json" // Default to JSON
	}
	
	return fmt.Sprintf("s3mpc_%s_export_%s.%s", sanitizedCommand, timestamp, format)
}

// StreamExportToCSV exports large datasets to CSV with streaming
func (e *ExportService) StreamExportToCSV(ctx context.Context, uploads <-chan types.MultipartUpload, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	header := []string{
		"bucket",
		"key",
		"upload_id",
		"initiated",
		"age_days",
		"size",
		"storage_class",
		"region",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Stream upload data
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case upload, ok := <-uploads:
			if !ok {
				// Channel closed, we're done
				return nil
			}
			
			ageDays := int(time.Since(upload.Initiated).Hours() / 24)
			
			record := []string{
				upload.Bucket,
				upload.Key,
				upload.UploadID,
				upload.Initiated.Format("2006-01-02T15:04:05Z"),
				strconv.Itoa(ageDays),
				strconv.FormatInt(upload.Size, 10),
				upload.StorageClass,
				upload.Region,
			}
			
			if err := writer.Write(record); err != nil {
				return fmt.Errorf("failed to write CSV record: %w", err)
			}
			
			// Flush periodically to avoid memory buildup
			writer.Flush()
		}
	}
}

// StreamExportToJSON exports large datasets to JSON with streaming
func (e *ExportService) StreamExportToJSON(ctx context.Context, uploads <-chan types.MultipartUpload, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	// Write JSON structure manually for streaming
	if _, err := file.WriteString("{\n"); err != nil {
		return fmt.Errorf("failed to write JSON opening: %w", err)
	}
	
	// Write metadata
	exportedAt := time.Now().Format("2006-01-02T15:04:05Z")
	if _, err := file.WriteString(fmt.Sprintf("  \"exported_at\": \"%s\",\n", exportedAt)); err != nil {
		return fmt.Errorf("failed to write exported_at: %w", err)
	}
	
	if _, err := file.WriteString("  \"uploads\": [\n"); err != nil {
		return fmt.Errorf("failed to write uploads array opening: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("    ", "  ")
	
	first := true
	count := 0
	
	// Stream upload data
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case upload, ok := <-uploads:
			if !ok {
				// Channel closed, finish the JSON structure
				if _, err := file.WriteString("\n  ],\n"); err != nil {
					return fmt.Errorf("failed to write uploads array closing: %w", err)
				}
				
				if _, err := file.WriteString(fmt.Sprintf("  \"total_count\": %d\n", count)); err != nil {
					return fmt.Errorf("failed to write total_count: %w", err)
				}
				
				if _, err := file.WriteString("}\n"); err != nil {
					return fmt.Errorf("failed to write JSON closing: %w", err)
				}
				
				return nil
			}
			
			if !first {
				if _, err := file.WriteString(",\n"); err != nil {
					return fmt.Errorf("failed to write JSON separator: %w", err)
				}
			}
			first = false
			count++
			
			// Encode the upload without newline
			uploadJSON, err := json.MarshalIndent(upload, "    ", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal upload: %w", err)
			}
			
			if _, err := file.Write(uploadJSON); err != nil {
				return fmt.Errorf("failed to write upload JSON: %w", err)
			}
		}
	}
}