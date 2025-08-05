package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Garvitkul/s3mpc/pkg/types"
)

// CostService implements the CostCalculator interface
type CostService struct {
	pricingData map[string]map[string]float64 // region -> storage class -> price per GB per month
}

// NewCostService creates a new CostService with AWS S3 pricing data
func NewCostService() *CostService {
	return &CostService{
		pricingData: getAWSS3PricingData(),
	}
}

// CalculateStorageCost calculates storage costs for uploads
func (c *CostService) CalculateStorageCost(ctx context.Context, uploads []types.MultipartUpload) (types.CostBreakdown, error) {
	if len(uploads) == 0 {
		return types.CostBreakdown{
			TotalMonthlyCost: 0.0,
			ByRegion:         make(map[string]float64),
			ByStorageClass:   make(map[string]float64),
			Currency:         "USD",
		}, nil
	}

	breakdown := types.CostBreakdown{
		ByRegion:       make(map[string]float64),
		ByStorageClass: make(map[string]float64),
		Currency:       "USD",
	}

	var totalCost float64

	for _, upload := range uploads {
		// Convert size from bytes to GB
		sizeGB := float64(upload.Size) / (1024 * 1024 * 1024)
		
		// Get pricing for this region and storage class
		price, err := c.GetRegionalPricing(ctx, upload.Region, upload.StorageClass)
		if err != nil {
			// If we can't get pricing, use a default estimate
			price = c.getDefaultPricing(upload.StorageClass)
		}

		// Calculate monthly cost for this upload
		monthlyCost := sizeGB * price

		// Add to totals
		totalCost += monthlyCost
		breakdown.ByRegion[upload.Region] += monthlyCost
		breakdown.ByStorageClass[upload.StorageClass] += monthlyCost
	}

	breakdown.TotalMonthlyCost = totalCost

	return breakdown, nil
}

// GetRegionalPricing retrieves pricing for a region and storage class
func (c *CostService) GetRegionalPricing(ctx context.Context, region, storageClass string) (float64, error) {
	// Normalize region name
	normalizedRegion := c.normalizeRegion(region)
	
	// Normalize storage class name
	normalizedStorageClass := c.normalizeStorageClass(storageClass)

	// Check if we have pricing data for this region
	regionPricing, exists := c.pricingData[normalizedRegion]
	if !exists {
		return 0, fmt.Errorf("pricing data not available for region: %s", region)
	}

	// Check if we have pricing data for this storage class
	price, exists := regionPricing[normalizedStorageClass]
	if !exists {
		return 0, fmt.Errorf("pricing data not available for storage class: %s in region: %s", storageClass, region)
	}

	return price, nil
}

// EstimateSavings calculates potential cost savings from deletion
func (c *CostService) EstimateSavings(ctx context.Context, uploads []types.MultipartUpload) (float64, error) {
	breakdown, err := c.CalculateStorageCost(ctx, uploads)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate cost breakdown: %w", err)
	}

	// The savings is the total monthly cost that would be eliminated
	return breakdown.TotalMonthlyCost, nil
}

// normalizeRegion normalizes region names to match our pricing data keys
func (c *CostService) normalizeRegion(region string) string {
	// Handle common region name variations
	region = strings.ToLower(strings.TrimSpace(region))
	
	// Map some common variations
	regionMap := map[string]string{
		"us-east-1":      "us-east-1",
		"us-east-2":      "us-east-2",
		"us-west-1":      "us-west-1",
		"us-west-2":      "us-west-2",
		"eu-west-1":      "eu-west-1",
		"eu-west-2":      "eu-west-2",
		"eu-west-3":      "eu-west-3",
		"eu-central-1":   "eu-central-1",
		"ap-southeast-1": "ap-southeast-1",
		"ap-southeast-2": "ap-southeast-2",
		"ap-northeast-1": "ap-northeast-1",
		"ap-northeast-2": "ap-northeast-2",
		"ap-south-1":     "ap-south-1",
		"sa-east-1":      "sa-east-1",
		"ca-central-1":   "ca-central-1",
	}

	if normalized, exists := regionMap[region]; exists {
		return normalized
	}

	return region
}

// normalizeStorageClass normalizes storage class names to match our pricing data keys
func (c *CostService) normalizeStorageClass(storageClass string) string {
	// Handle common storage class name variations
	storageClass = strings.ToUpper(strings.TrimSpace(storageClass))
	
	// Map some common variations
	classMap := map[string]string{
		"STANDARD":                    "STANDARD",
		"STANDARD_IA":                 "STANDARD_IA",
		"STANDARD-IA":                 "STANDARD_IA",
		"ONEZONE_IA":                  "ONEZONE_IA",
		"ONEZONE-IA":                  "ONEZONE_IA",
		"REDUCED_REDUNDANCY":          "REDUCED_REDUNDANCY",
		"GLACIER":                     "GLACIER",
		"GLACIER_IR":                  "GLACIER_IR",
		"GLACIER-IR":                  "GLACIER_IR",
		"DEEP_ARCHIVE":                "DEEP_ARCHIVE",
		"DEEP-ARCHIVE":                "DEEP_ARCHIVE",
		"INTELLIGENT_TIERING":         "INTELLIGENT_TIERING",
		"INTELLIGENT-TIERING":         "INTELLIGENT_TIERING",
	}

	if normalized, exists := classMap[storageClass]; exists {
		return normalized
	}

	return storageClass
}

// getDefaultPricing returns default pricing when specific pricing is not available
func (c *CostService) getDefaultPricing(storageClass string) float64 {
	// Default pricing based on US East 1 rates (as of 2024)
	defaultPrices := map[string]float64{
		"STANDARD":             0.023, // $0.023 per GB per month
		"STANDARD_IA":          0.0125, // $0.0125 per GB per month
		"ONEZONE_IA":           0.01,   // $0.01 per GB per month
		"REDUCED_REDUNDANCY":   0.024,  // $0.024 per GB per month
		"GLACIER":              0.004,  // $0.004 per GB per month
		"GLACIER_IR":           0.004,  // $0.004 per GB per month
		"DEEP_ARCHIVE":         0.00099, // $0.00099 per GB per month
		"INTELLIGENT_TIERING":  0.0125,  // $0.0125 per GB per month (average)
	}

	normalizedClass := c.normalizeStorageClass(storageClass)
	if price, exists := defaultPrices[normalizedClass]; exists {
		return price
	}

	// If we don't recognize the storage class, use STANDARD pricing
	return defaultPrices["STANDARD"]
}

// getAWSS3PricingData returns AWS S3 pricing data for different regions and storage classes
// Prices are in USD per GB per month (as of 2024)
func getAWSS3PricingData() map[string]map[string]float64 {
	return map[string]map[string]float64{
		"us-east-1": {
			"STANDARD":             0.023,
			"STANDARD_IA":          0.0125,
			"ONEZONE_IA":           0.01,
			"REDUCED_REDUNDANCY":   0.024,
			"GLACIER":              0.004,
			"GLACIER_IR":           0.004,
			"DEEP_ARCHIVE":         0.00099,
			"INTELLIGENT_TIERING":  0.0125,
		},
		"us-east-2": {
			"STANDARD":             0.023,
			"STANDARD_IA":          0.0125,
			"ONEZONE_IA":           0.01,
			"REDUCED_REDUNDANCY":   0.024,
			"GLACIER":              0.004,
			"GLACIER_IR":           0.004,
			"DEEP_ARCHIVE":         0.00099,
			"INTELLIGENT_TIERING":  0.0125,
		},
		"us-west-1": {
			"STANDARD":             0.026,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.027,
			"GLACIER":              0.004,
			"GLACIER_IR":           0.004,
			"DEEP_ARCHIVE":         0.00099,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"us-west-2": {
			"STANDARD":             0.023,
			"STANDARD_IA":          0.0125,
			"ONEZONE_IA":           0.01,
			"REDUCED_REDUNDANCY":   0.024,
			"GLACIER":              0.004,
			"GLACIER_IR":           0.004,
			"DEEP_ARCHIVE":         0.00099,
			"INTELLIGENT_TIERING":  0.0125,
		},
		"eu-west-1": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"eu-west-2": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"eu-west-3": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"eu-central-1": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"ap-southeast-1": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"ap-southeast-2": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"ap-northeast-1": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"ap-northeast-2": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"ap-south-1": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
		"sa-east-1": {
			"STANDARD":             0.027,
			"STANDARD_IA":          0.015,
			"ONEZONE_IA":           0.012,
			"REDUCED_REDUNDANCY":   0.028,
			"GLACIER":              0.0048,
			"GLACIER_IR":           0.0048,
			"DEEP_ARCHIVE":         0.00115,
			"INTELLIGENT_TIERING":  0.015,
		},
		"ca-central-1": {
			"STANDARD":             0.025,
			"STANDARD_IA":          0.0138,
			"ONEZONE_IA":           0.011,
			"REDUCED_REDUNDANCY":   0.026,
			"GLACIER":              0.0045,
			"GLACIER_IR":           0.0045,
			"DEEP_ARCHIVE":         0.00108,
			"INTELLIGENT_TIERING":  0.0138,
		},
	}
}