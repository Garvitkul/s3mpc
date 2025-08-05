package types

import (
	"testing"
	"time"
)

func TestMultipartUploadValidation(t *testing.T) {
	tests := []struct {
		name    string
		upload  MultipartUpload
		wantErr bool
	}{
		{
			name: "valid upload",
			upload: MultipartUpload{
				Bucket:       "test-bucket",
				Key:          "test-key",
				UploadID:     "test-upload-id",
				Initiated:    time.Now(),
				Size:         1024,
				StorageClass: "STANDARD",
				Region:       "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "empty bucket",
			upload: MultipartUpload{
				Bucket:       "",
				Key:          "test-key",
				UploadID:     "test-upload-id",
				Initiated:    time.Now(),
				Size:         1024,
				StorageClass: "STANDARD",
				Region:       "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "negative size",
			upload: MultipartUpload{
				Bucket:       "test-bucket",
				Key:          "test-key",
				UploadID:     "test-upload-id",
				Initiated:    time.Now(),
				Size:         -1,
				StorageClass: "STANDARD",
				Region:       "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "zero time",
			upload: MultipartUpload{
				Bucket:       "test-bucket",
				Key:          "test-key",
				UploadID:     "test-upload-id",
				Initiated:    time.Time{},
				Size:         1024,
				StorageClass: "STANDARD",
				Region:       "us-east-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.upload.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MultipartUpload.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListOptionsValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    ListOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: ListOptions{
				MaxResults: 100,
				Offset:     0,
			},
			wantErr: false,
		},
		{
			name: "negative max results",
			opts: ListOptions{
				MaxResults: -1,
				Offset:     0,
			},
			wantErr: true,
		},
		{
			name: "negative offset",
			opts: ListOptions{
				MaxResults: 100,
				Offset:     -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ListOptions.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteOptionsValidation(t *testing.T) {
	smallerThan := int64(100)
	largerThan := int64(50)
	invalidSmallerThan := int64(50)
	invalidLargerThan := int64(100)

	tests := []struct {
		name    string
		opts    DeleteOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: DeleteOptions{
				SmallerThan: &smallerThan,
				LargerThan:  &largerThan,
			},
			wantErr: false,
		},
		{
			name: "invalid size range",
			opts: DeleteOptions{
				SmallerThan: &invalidSmallerThan,
				LargerThan:  &invalidLargerThan,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteOptions.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}