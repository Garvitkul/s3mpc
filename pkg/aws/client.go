package aws

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"golang.org/x/time/rate"
)

// RetryConfig defines retry behavior configuration
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	BaseDelay     time.Duration `json:"base_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// S3Client wraps the AWS S3 client with retry logic and rate limiting
type S3Client struct {
	client      *s3.Client
	retryConfig RetryConfig
	rateLimiter *rate.Limiter
}

// ClientConfig contains configuration for creating an S3Client
type ClientConfig struct {
	Profile     string
	Region      string
	RetryConfig RetryConfig
	RateLimit   rate.Limit // requests per second
}

// NewS3Client creates a new S3Client with retry logic and rate limiting
func NewS3Client(ctx context.Context, cfg ClientConfig) (*S3Client, error) {
	// Load AWS configuration
	var awsConfig aws.Config
	var err error

	if cfg.Profile != "" {
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithSharedConfigProfile(cfg.Profile),
			config.WithRegion(cfg.Region),
		)
	} else {
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsConfig)

	// Set default rate limit if not specified (10 requests per second)
	rateLimit := cfg.RateLimit
	if rateLimit == 0 {
		rateLimit = 10
	}

	// Set default retry config if not specified
	retryConfig := cfg.RetryConfig
	if retryConfig.MaxRetries == 0 {
		retryConfig = DefaultRetryConfig()
	}

	return &S3Client{
		client:      s3Client,
		retryConfig: retryConfig,
		rateLimiter: rate.NewLimiter(rateLimit, int(rateLimit)),
	}, nil
}

// isRetryableError determines if an error should be retried
func (c *S3Client) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are retryable
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	// AWS service errors
	var apiErr *types.NoSuchBucket
	if errors.As(err, &apiErr) {
		return false // Not retryable
	}

	// Check for specific AWS error codes that are retryable
	errStr := err.Error()
	retryableErrors := []string{
		"RequestTimeout",
		"ServiceUnavailable",
		"InternalError",
		"SlowDown",
		"TooManyRequests",
		"RequestTimeTooSkewed",
	}

	for _, retryableErr := range retryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// calculateBackoffDelay calculates the delay for a retry attempt
func (c *S3Client) calculateBackoffDelay(attempt int) time.Duration {
	delay := time.Duration(float64(c.retryConfig.BaseDelay) * math.Pow(c.retryConfig.BackoffFactor, float64(attempt)))
	if delay > c.retryConfig.MaxDelay {
		delay = c.retryConfig.MaxDelay
	}
	return delay
}

// executeWithRetry executes a function with retry logic
func (c *S3Client) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// Wait for rate limiter
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter error: %w", err)
		}

		// Execute the operation
		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt == c.retryConfig.MaxRetries {
			break
		}

		// Check if error is retryable
		if !c.isRetryableError(err) {
			return err // Non-retryable error
		}

		// Calculate backoff delay
		delay := c.calculateBackoffDelay(attempt)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", c.retryConfig.MaxRetries, lastErr)
}

// ListBuckets lists all S3 buckets with retry logic
func (c *S3Client) ListBuckets(ctx context.Context) (*s3.ListBucketsOutput, error) {
	var result *s3.ListBucketsOutput
	var err error

	operation := func() error {
		result, err = c.client.ListBuckets(ctx, &s3.ListBucketsInput{})
		return err
	}

	if retryErr := c.executeWithRetry(ctx, operation); retryErr != nil {
		return nil, retryErr
	}

	return result, nil
}

// GetBucketLocation gets the region of a specific bucket with retry logic
func (c *S3Client) GetBucketLocation(ctx context.Context, bucket string) (*s3.GetBucketLocationOutput, error) {
	var result *s3.GetBucketLocationOutput
	var err error

	operation := func() error {
		result, err = c.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: aws.String(bucket),
		})
		return err
	}

	if retryErr := c.executeWithRetry(ctx, operation); retryErr != nil {
		return nil, retryErr
	}

	return result, nil
}

// ListMultipartUploads lists incomplete multipart uploads for a bucket with retry logic
func (c *S3Client) ListMultipartUploads(ctx context.Context, input *s3.ListMultipartUploadsInput) (*s3.ListMultipartUploadsOutput, error) {
	var result *s3.ListMultipartUploadsOutput
	var err error

	operation := func() error {
		result, err = c.client.ListMultipartUploads(ctx, input)
		return err
	}

	if retryErr := c.executeWithRetry(ctx, operation); retryErr != nil {
		return nil, retryErr
	}

	return result, nil
}

// ListParts lists parts of a multipart upload with retry logic
func (c *S3Client) ListParts(ctx context.Context, input *s3.ListPartsInput) (*s3.ListPartsOutput, error) {
	var result *s3.ListPartsOutput
	var err error

	operation := func() error {
		result, err = c.client.ListParts(ctx, input)
		return err
	}

	if retryErr := c.executeWithRetry(ctx, operation); retryErr != nil {
		return nil, retryErr
	}

	return result, nil
}

// AbortMultipartUpload aborts a multipart upload with retry logic
func (c *S3Client) AbortMultipartUpload(ctx context.Context, input *s3.AbortMultipartUploadInput) (*s3.AbortMultipartUploadOutput, error) {
	var result *s3.AbortMultipartUploadOutput
	var err error

	operation := func() error {
		result, err = c.client.AbortMultipartUpload(ctx, input)
		return err
	}

	if retryErr := c.executeWithRetry(ctx, operation); retryErr != nil {
		return nil, retryErr
	}

	return result, nil
}

// HeadBucket checks if a bucket exists and is accessible with retry logic
func (c *S3Client) HeadBucket(ctx context.Context, bucket string) (*s3.HeadBucketOutput, error) {
	var result *s3.HeadBucketOutput
	var err error

	operation := func() error {
		result, err = c.client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucket),
		})
		return err
	}

	if retryErr := c.executeWithRetry(ctx, operation); retryErr != nil {
		return nil, retryErr
	}

	return result, nil
}

// GetClient returns the underlying S3 client for advanced operations
func (c *S3Client) GetClient() *s3.Client {
	return c.client
}

// GetRetryConfig returns the current retry configuration
func (c *S3Client) GetRetryConfig() RetryConfig {
	return c.retryConfig
}

// UpdateRateLimit updates the rate limiter with a new limit
func (c *S3Client) UpdateRateLimit(limit rate.Limit) {
	c.rateLimiter.SetLimit(limit)
	c.rateLimiter.SetBurst(int(limit))
}