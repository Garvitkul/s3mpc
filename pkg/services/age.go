package services

import (
	"context"
	"time"

	"github.com/s3mpc/s3mpc/pkg/interfaces"
	"github.com/s3mpc/s3mpc/pkg/types"
)

// ageService implements the AgeService interface
type ageService struct{}

// NewAgeService creates a new age service instance
func NewAgeService() interfaces.AgeService {
	return &ageService{}
}

// CalculateAgeDistribution calculates age distribution of uploads
func (s *ageService) CalculateAgeDistribution(ctx context.Context, uploads []types.MultipartUpload) (types.AgeDistribution, error) {
	// Define age buckets based on requirements: 1 day, 1 week, 1 month, 3 months, 6 months, 1 year+
	buckets := []types.AgeBucket{
		{Label: "1 day", MinAge: 0, MaxAge: 24 * time.Hour, Count: 0, TotalSize: 0},
		{Label: "1 week", MinAge: 24 * time.Hour, MaxAge: 7 * 24 * time.Hour, Count: 0, TotalSize: 0},
		{Label: "1 month", MinAge: 7 * 24 * time.Hour, MaxAge: 30 * 24 * time.Hour, Count: 0, TotalSize: 0},
		{Label: "3 months", MinAge: 30 * 24 * time.Hour, MaxAge: 90 * 24 * time.Hour, Count: 0, TotalSize: 0},
		{Label: "6 months", MinAge: 90 * 24 * time.Hour, MaxAge: 180 * 24 * time.Hour, Count: 0, TotalSize: 0},
		{Label: "1 year+", MinAge: 180 * 24 * time.Hour, MaxAge: time.Duration(0), Count: 0, TotalSize: 0}, // MaxAge 0 means no upper limit
	}

	now := time.Now()

	// Categorize each upload into appropriate age bucket
	for _, upload := range uploads {
		age := now.Sub(upload.Initiated)
		
		// Find the appropriate bucket for this upload
		for i := range buckets {
			bucket := &buckets[i]
			
			// For the last bucket (1 year+), MaxAge of 0 means no upper limit
			if bucket.MaxAge == 0 {
				if age >= bucket.MinAge {
					bucket.Count++
					bucket.TotalSize += upload.Size
					break
				}
			} else {
				// For other buckets, check if age falls within the range
				if age >= bucket.MinAge && age < bucket.MaxAge {
					bucket.Count++
					bucket.TotalSize += upload.Size
					break
				}
			}
		}
	}

	return types.AgeDistribution{Buckets: buckets}, nil
}

// GetAgeDistributionForBucket calculates age distribution for a specific bucket
func (s *ageService) GetAgeDistributionForBucket(ctx context.Context, uploads []types.MultipartUpload, bucketName string) (types.AgeDistribution, error) {
	// Filter uploads for the specific bucket
	var bucketUploads []types.MultipartUpload
	for _, upload := range uploads {
		if upload.Bucket == bucketName {
			bucketUploads = append(bucketUploads, upload)
		}
	}

	// Calculate age distribution for filtered uploads
	return s.CalculateAgeDistribution(ctx, bucketUploads)
}

// IsOlderThanSevenDays checks if an upload is older than 7 days (for highlighting)
func (s *ageService) IsOlderThanSevenDays(upload types.MultipartUpload) bool {
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	return upload.Initiated.Before(sevenDaysAgo)
}