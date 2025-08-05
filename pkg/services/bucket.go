package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	awsclient "github.com/Garvitkul/s3mpc/pkg/aws"
	"github.com/Garvitkul/s3mpc/pkg/interfaces"
	pkgtypes "github.com/Garvitkul/s3mpc/pkg/types"
)

// S3ClientInterface defines the S3 operations needed by BucketService
type S3ClientInterface interface {
	ListBuckets(ctx context.Context) (*s3.ListBucketsOutput, error)
	GetBucketLocation(ctx context.Context, bucket string) (*s3.GetBucketLocationOutput, error)
}

// BucketService implements the interfaces.BucketService interface
type BucketService struct {
	client      S3ClientInterface
	regionCache map[string]string
	cacheMutex  sync.RWMutex
	cacheExpiry time.Duration
	cacheTime   map[string]time.Time
}

// NewBucketService creates a new BucketService instance
func NewBucketService(client *awsclient.S3Client) interfaces.BucketService {
	return &BucketService{
		client:      client,
		regionCache: make(map[string]string),
		cacheTime:   make(map[string]time.Time),
		cacheExpiry: 1 * time.Hour, // Cache regions for 1 hour
	}
}

// ListBuckets retrieves all accessible S3 buckets
func (s *BucketService) ListBuckets(ctx context.Context, region string) ([]pkgtypes.Bucket, error) {
	// List all buckets
	output, err := s.client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	var buckets []pkgtypes.Bucket
	
	// If region is specified, filter buckets by region
	if region != "" {
		return s.filterBucketsByRegion(ctx, output.Buckets, region)
	}

	// Convert AWS bucket types to our bucket types
	for _, bucket := range output.Buckets {
		if bucket.Name == nil {
			continue
		}

		bucketRegion, err := s.GetBucketRegion(ctx, *bucket.Name)
		if err != nil {
			// Log error but continue with other buckets
			// In a real implementation, you might want to use a proper logger
			continue
		}

		buckets = append(buckets, pkgtypes.Bucket{
			Name:   *bucket.Name,
			Region: bucketRegion,
		})
	}

	return buckets, nil
}

// ListBucketsInRegion retrieves buckets in a specific region
func (s *BucketService) ListBucketsInRegion(ctx context.Context, region string) ([]pkgtypes.Bucket, error) {
	return s.ListBuckets(ctx, region)
}

// GetBucketRegion retrieves the region for a specific bucket with caching
func (s *BucketService) GetBucketRegion(ctx context.Context, bucketName string) (string, error) {
	// Check cache first
	s.cacheMutex.RLock()
	if cachedRegion, exists := s.regionCache[bucketName]; exists {
		if cacheTime, timeExists := s.cacheTime[bucketName]; timeExists {
			if time.Since(cacheTime) < s.cacheExpiry {
				s.cacheMutex.RUnlock()
				return cachedRegion, nil
			}
		}
	}
	s.cacheMutex.RUnlock()

	// Cache miss or expired, fetch from AWS
	output, err := s.client.GetBucketLocation(ctx, bucketName)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket location for %s: %w", bucketName, err)
	}

	// AWS returns empty string for us-east-1
	region := "us-east-1"
	if output.LocationConstraint != "" {
		region = string(output.LocationConstraint)
	}

	// Update cache
	s.cacheMutex.Lock()
	s.regionCache[bucketName] = region
	s.cacheTime[bucketName] = time.Now()
	s.cacheMutex.Unlock()

	return region, nil
}

// filterBucketsByRegion filters buckets by the specified region
func (s *BucketService) filterBucketsByRegion(ctx context.Context, awsBuckets []types.Bucket, targetRegion string) ([]pkgtypes.Bucket, error) {
	var buckets []pkgtypes.Bucket
	
	// Use a channel to collect results from concurrent goroutines
	type bucketResult struct {
		bucket pkgtypes.Bucket
		err    error
	}
	
	resultChan := make(chan bucketResult, len(awsBuckets))
	
	// Process buckets concurrently
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent operations to 10
	
	for _, bucket := range awsBuckets {
		if bucket.Name == nil {
			continue
		}
		
		wg.Add(1)
		go func(bucketName string) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			region, err := s.GetBucketRegion(ctx, bucketName)
			if err != nil {
				resultChan <- bucketResult{err: err}
				return
			}
			
			// Only include buckets in the target region
			if region == targetRegion {
				resultChan <- bucketResult{
					bucket: pkgtypes.Bucket{
						Name:   bucketName,
						Region: region,
					},
				}
			}
		}(*bucket.Name)
	}
	
	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results
	var errors []error
	for result := range resultChan {
		if result.err != nil {
			errors = append(errors, result.err)
			continue
		}
		buckets = append(buckets, result.bucket)
	}
	
	// If we have errors but also some successful results, we might want to return partial results
	// For now, we'll return an error if any bucket failed
	if len(errors) > 0 {
		return buckets, fmt.Errorf("failed to get region for some buckets: %d errors occurred", len(errors))
	}
	
	return buckets, nil
}

// ClearRegionCache clears the region cache (useful for testing)
func (s *BucketService) ClearRegionCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	
	s.regionCache = make(map[string]string)
	s.cacheTime = make(map[string]time.Time)
}

// GetCacheStats returns cache statistics (useful for monitoring)
func (s *BucketService) GetCacheStats() map[string]interface{} {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	
	return map[string]interface{}{
		"cached_regions": len(s.regionCache),
		"cache_expiry":   s.cacheExpiry.String(),
	}
}