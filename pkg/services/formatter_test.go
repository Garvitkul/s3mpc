package services

import (
	"strings"
	"testing"
	"time"

	"github.com/Garvitkul/s3mpc/pkg/types"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			bytes:    1572864, // 1.5 MB
			expected: "1.5 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1610612736, // 1.5 GB
			expected: "1.5 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatUploads(t *testing.T) {
	formatter := NewOutputFormatter()

	// Test empty uploads
	result := formatter.FormatUploads([]types.MultipartUpload{}, true)
	if !strings.Contains(result, "No incomplete multipart uploads found") {
		t.Errorf("Expected message for empty uploads, got: %s", result)
	}

	// Test with uploads
	uploads := []types.MultipartUpload{
		{
			Bucket:       "test-bucket",
			Key:          "test-key",
			UploadID:     "test-upload-id",
			Initiated:    time.Now().Add(-24 * time.Hour),
			Size:         1024 * 1024, // 1MB
			StorageClass: "STANDARD",
			Region:       "us-east-1",
		},
	}

	result = formatter.FormatUploads(uploads, true)
	if !strings.Contains(result, "test-bucket") {
		t.Errorf("Expected bucket name in output, got: %s", result)
	}
	if !strings.Contains(result, "STANDARD") {
		t.Errorf("Expected storage class in output, got: %s", result)
	}
}

func TestFormatSizeReport(t *testing.T) {
	formatter := NewOutputFormatter()

	report := types.SizeReport{
		TotalSize:  2048,
		TotalCount: 2,
		ByBucket: map[string]int64{
			"bucket1": 1024,
			"bucket2": 1024,
		},
		ByStorageClass: map[string]int64{
			"STANDARD": 2048,
		},
	}

	result := formatter.FormatSizeReport(report)
	if !strings.Contains(result, "Total incomplete multipart uploads: 2") {
		t.Errorf("Expected total count in output, got: %s", result)
	}
	if !strings.Contains(result, "bucket1") || !strings.Contains(result, "bucket2") {
		t.Errorf("Expected bucket names in output, got: %s", result)
	}
}

func TestFormatJSON(t *testing.T) {
	formatter := NewOutputFormatter()

	data := map[string]interface{}{
		"test": "value",
		"number": 42,
	}

	result, err := formatter.FormatJSON(data)
	if err != nil {
		t.Errorf("FormatJSON() error = %v", err)
	}
	if !strings.Contains(result, "test") || !strings.Contains(result, "value") {
		t.Errorf("Expected JSON content in output, got: %s", result)
	}
}