package filter

import (
	"testing"
	"time"

	"github.com/s3mpc/s3mpc/pkg/types"
)

func TestParseFilter(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name      string
		filterStr string
		wantErr   bool
	}{
		{
			name:      "empty filter",
			filterStr: "",
			wantErr:   false,
		},
		{
			name:      "valid age filter",
			filterStr: "age>7d",
			wantErr:   false,
		},
		{
			name:      "valid size filter",
			filterStr: "size<100MB",
			wantErr:   false,
		},
		{
			name:      "valid storage class filter",
			filterStr: "storageClass=STANDARD",
			wantErr:   false,
		},
		{
			name:      "multiple filters",
			filterStr: "age>7d,size<100MB,storageClass=STANDARD",
			wantErr:   false,
		},
		{
			name:      "invalid field",
			filterStr: "invalid>7d",
			wantErr:   true,
		},
		{
			name:      "invalid operator for string field",
			filterStr: "storageClass>STANDARD",
			wantErr:   true,
		},
		{
			name:      "invalid age format",
			filterStr: "age>invalid",
			wantErr:   true,
		},
		{
			name:      "invalid size format",
			filterStr: "size<invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.ParseFilter(tt.filterStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyFilter(t *testing.T) {
	engine := NewEngine()

	// Create test uploads
	now := time.Now()
	uploads := []types.MultipartUpload{
		{
			Bucket:       "bucket1",
			Key:          "key1",
			UploadID:     "upload1",
			Initiated:    now.Add(-10 * 24 * time.Hour), // 10 days ago
			Size:         50 * 1024 * 1024,              // 50MB
			StorageClass: "STANDARD",
			Region:       "us-east-1",
		},
		{
			Bucket:       "bucket2",
			Key:          "key2",
			UploadID:     "upload2",
			Initiated:    now.Add(-5 * 24 * time.Hour), // 5 days ago
			Size:         200 * 1024 * 1024,            // 200MB
			StorageClass: "STANDARD_IA",
			Region:       "us-west-2",
		},
		{
			Bucket:       "bucket1",
			Key:          "key3",
			UploadID:     "upload3",
			Initiated:    now.Add(-1 * 24 * time.Hour), // 1 day ago
			Size:         1024 * 1024 * 1024,           // 1GB
			StorageClass: "STANDARD",
			Region:       "us-east-1",
		},
	}

	tests := []struct {
		name         string
		filterStr    string
		expectedLen  int
		wantErr      bool
	}{
		{
			name:        "no filter",
			filterStr:   "",
			expectedLen: 3,
			wantErr:     false,
		},
		{
			name:        "age filter - older than 7 days",
			filterStr:   "age>7d",
			expectedLen: 1, // Only the 10-day-old upload
			wantErr:     false,
		},
		{
			name:        "size filter - smaller than 100MB",
			filterStr:   "size<100MB",
			expectedLen: 1, // Only the 50MB upload
			wantErr:     false,
		},
		{
			name:        "storage class filter",
			filterStr:   "storageClass=STANDARD",
			expectedLen: 2, // Two STANDARD uploads
			wantErr:     false,
		},
		{
			name:        "bucket filter",
			filterStr:   "bucket=bucket1",
			expectedLen: 2, // Two uploads in bucket1
			wantErr:     false,
		},
		{
			name:        "region filter",
			filterStr:   "region=us-east-1",
			expectedLen: 2, // Two uploads in us-east-1
			wantErr:     false,
		},
		{
			name:        "combined filters",
			filterStr:   "age>7d,storageClass=STANDARD",
			expectedLen: 1, // Only the 10-day-old STANDARD upload
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := engine.ParseFilter(tt.filterStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				filtered := engine.ApplyFilter(uploads, filter)
				if len(filtered) != tt.expectedLen {
					t.Errorf("ApplyFilter() returned %d uploads, expected %d", len(filtered), tt.expectedLen)
				}
			}
		})
	}
}