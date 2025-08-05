package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/s3mpc/s3mpc/pkg/interfaces"
	"github.com/s3mpc/s3mpc/pkg/types"
)

// Engine implements the FilterEngine interface
type Engine struct{}

// NewEngine creates a new filter engine
func NewEngine() interfaces.FilterEngine {
	return &Engine{}
}

// ParseFilter parses a filter string into a structured filter
func (e *Engine) ParseFilter(filterStr string) (interfaces.Filter, error) {
	if strings.TrimSpace(filterStr) == "" {
		return interfaces.Filter{}, nil
	}

	filter := interfaces.Filter{}
	
	// Split by comma for AND logic
	conditions := strings.Split(filterStr, ",")
	
	for _, condition := range conditions {
		condition = strings.TrimSpace(condition)
		if condition == "" {
			continue
		}
		
		if err := e.parseCondition(condition, &filter); err != nil {
			return interfaces.Filter{}, fmt.Errorf("invalid filter condition '%s': %w", condition, err)
		}
	}
	
	return filter, nil
}

// ValidateFilter validates filter syntax without parsing
func (e *Engine) ValidateFilter(filterStr string) error {
	_, err := e.ParseFilter(filterStr)
	return err
}

// parseCondition parses a single condition and updates the filter
func (e *Engine) parseCondition(condition string, filter *interfaces.Filter) error {
	// Regular expression to match: field operator value
	re := regexp.MustCompile(`^(\w+)\s*(>=|<=|!=|>|<|=)\s*(.+)$`)
	matches := re.FindStringSubmatch(condition)
	
	if len(matches) != 4 {
		return fmt.Errorf("invalid syntax, expected 'field operator value'")
	}
	
	field := strings.ToLower(matches[1])
	operator := matches[2]
	value := strings.TrimSpace(matches[3])
	
	switch field {
	case "age":
		if filter.Age != nil {
			return fmt.Errorf("age filter already specified")
		}
		if err := e.validateAgeOperator(operator); err != nil {
			return err
		}
		if err := e.validateAgeValue(value); err != nil {
			return err
		}
		filter.Age = &interfaces.AgeFilter{
			Operator: operator,
			Value:    value,
		}
		
	case "size":
		if filter.Size != nil {
			return fmt.Errorf("size filter already specified")
		}
		if err := e.validateSizeOperator(operator); err != nil {
			return err
		}
		if err := e.validateSizeValue(value); err != nil {
			return err
		}
		filter.Size = &interfaces.SizeFilter{
			Operator: operator,
			Value:    value,
		}
		
	case "storageclass":
		if filter.StorageClass != nil {
			return fmt.Errorf("storageClass filter already specified")
		}
		if err := e.validateStringOperator(operator); err != nil {
			return err
		}
		filter.StorageClass = &interfaces.StringFilter{
			Operator: operator,
			Value:    value,
		}
		
	case "region":
		if filter.Region != nil {
			return fmt.Errorf("region filter already specified")
		}
		if err := e.validateStringOperator(operator); err != nil {
			return err
		}
		filter.Region = &interfaces.StringFilter{
			Operator: operator,
			Value:    value,
		}
		
	case "bucket":
		if filter.Bucket != nil {
			return fmt.Errorf("bucket filter already specified")
		}
		if err := e.validateStringOperator(operator); err != nil {
			return err
		}
		filter.Bucket = &interfaces.StringFilter{
			Operator: operator,
			Value:    value,
		}
		
	default:
		return fmt.Errorf("unsupported field '%s', supported fields: age, size, storageClass, region, bucket", field)
	}
	
	return nil
}

// validateAgeOperator validates operators for age filters
func (e *Engine) validateAgeOperator(operator string) error {
	validOperators := map[string]bool{
		">": true, "<": true, ">=": true, "<=": true, "=": true, "!=": true,
	}
	if !validOperators[operator] {
		return fmt.Errorf("invalid operator '%s' for age field, supported: >, <, >=, <=, =, !=", operator)
	}
	return nil
}

// validateSizeOperator validates operators for size filters
func (e *Engine) validateSizeOperator(operator string) error {
	validOperators := map[string]bool{
		">": true, "<": true, ">=": true, "<=": true, "=": true, "!=": true,
	}
	if !validOperators[operator] {
		return fmt.Errorf("invalid operator '%s' for size field, supported: >, <, >=, <=, =, !=", operator)
	}
	return nil
}

// validateStringOperator validates operators for string filters
func (e *Engine) validateStringOperator(operator string) error {
	validOperators := map[string]bool{
		"=": true, "!=": true,
	}
	if !validOperators[operator] {
		return fmt.Errorf("invalid operator '%s' for string field, supported: =, !=", operator)
	}
	return nil
}

// validateAgeValue validates age value format
func (e *Engine) validateAgeValue(value string) error {
	_, err := e.parseAgeDuration(value)
	return err
}

// validateSizeValue validates size value format
func (e *Engine) validateSizeValue(value string) error {
	_, err := e.parseSizeBytes(value)
	return err
}

// parseAgeDuration parses age duration from string (e.g., "7d", "1w", "1m", "1y")
func (e *Engine) parseAgeDuration(value string) (time.Duration, error) {
	if len(value) < 2 {
		return 0, fmt.Errorf("invalid age format '%s', expected format like '7d', '1w', '1m', '1y'", value)
	}
	
	numStr := value[:len(value)-1]
	unit := strings.ToLower(value[len(value)-1:])
	
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in age '%s': %w", value, err)
	}
	
	if num < 0 {
		return 0, fmt.Errorf("age cannot be negative: %s", value)
	}
	
	var duration time.Duration
	switch unit {
	case "d":
		duration = time.Duration(num * float64(24*time.Hour))
	case "w":
		duration = time.Duration(num * float64(7*24*time.Hour))
	case "m":
		duration = time.Duration(num * float64(30*24*time.Hour)) // Approximate month
	case "y":
		duration = time.Duration(num * float64(365*24*time.Hour)) // Approximate year
	default:
		return 0, fmt.Errorf("invalid age unit '%s', supported units: d (days), w (weeks), m (months), y (years)", unit)
	}
	
	return duration, nil
}

// parseSizeBytes parses size from string (e.g., "100MB", "1GB", "500KB")
func (e *Engine) parseSizeBytes(value string) (int64, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	
	if len(value) < 2 {
		return 0, fmt.Errorf("invalid size format '%s', expected format like '100MB', '1GB'", value)
	}
	
	// Handle case where value is just a number (assume bytes)
	if num, err := strconv.ParseInt(value, 10, 64); err == nil {
		if num < 0 {
			return 0, fmt.Errorf("size cannot be negative: %s", value)
		}
		return num, nil
	}
	
	// Extract number and unit
	var numStr, unit string
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] >= '0' && value[i] <= '9' || value[i] == '.' {
			numStr = value[:i+1]
			unit = value[i+1:]
			break
		}
	}
	
	if numStr == "" {
		return 0, fmt.Errorf("invalid size format '%s', no number found", value)
	}
	
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in size '%s': %w", value, err)
	}
	
	if num < 0 {
		return 0, fmt.Errorf("size cannot be negative: %s", value)
	}
	
	var multiplier int64
	switch unit {
	case "B", "":
		multiplier = 1
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	case "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("invalid size unit '%s', supported units: B, KB, MB, GB, TB", unit)
	}
	
	return int64(num * float64(multiplier)), nil
}

// ApplyFilter applies a filter to a list of uploads
func (e *Engine) ApplyFilter(uploads []types.MultipartUpload, filter interfaces.Filter) []types.MultipartUpload {
	if e.isEmptyFilter(filter) {
		return uploads
	}
	
	var filtered []types.MultipartUpload
	
	for _, upload := range uploads {
		if e.matchesFilter(upload, filter) {
			filtered = append(filtered, upload)
		}
	}
	
	return filtered
}

// isEmptyFilter checks if the filter is empty
func (e *Engine) isEmptyFilter(filter interfaces.Filter) bool {
	return filter.Age == nil && filter.Size == nil && filter.StorageClass == nil && 
		   filter.Region == nil && filter.Bucket == nil
}

// matchesFilter checks if an upload matches the filter criteria
func (e *Engine) matchesFilter(upload types.MultipartUpload, filter interfaces.Filter) bool {
	// All conditions must match (AND logic)
	
	if filter.Age != nil && !e.matchesAgeFilter(upload, *filter.Age) {
		return false
	}
	
	if filter.Size != nil && !e.matchesSizeFilter(upload, *filter.Size) {
		return false
	}
	
	if filter.StorageClass != nil && !e.matchesStringFilter(upload.StorageClass, *filter.StorageClass) {
		return false
	}
	
	if filter.Region != nil && !e.matchesStringFilter(upload.Region, *filter.Region) {
		return false
	}
	
	if filter.Bucket != nil && !e.matchesStringFilter(upload.Bucket, *filter.Bucket) {
		return false
	}
	
	return true
}

// matchesAgeFilter checks if upload matches age filter
func (e *Engine) matchesAgeFilter(upload types.MultipartUpload, filter interfaces.AgeFilter) bool {
	uploadAge := time.Since(upload.Initiated)
	filterDuration, err := e.parseAgeDuration(filter.Value)
	if err != nil {
		// This should not happen if validation was done properly
		return false
	}
	
	switch filter.Operator {
	case ">":
		return uploadAge > filterDuration
	case "<":
		return uploadAge < filterDuration
	case ">=":
		return uploadAge >= filterDuration
	case "<=":
		return uploadAge <= filterDuration
	case "=":
		// For age equality, we allow some tolerance (within 1 hour)
		diff := uploadAge - filterDuration
		if diff < 0 {
			diff = -diff
		}
		return diff <= time.Hour
	case "!=":
		// For age inequality, we check if it's outside the tolerance
		diff := uploadAge - filterDuration
		if diff < 0 {
			diff = -diff
		}
		return diff > time.Hour
	default:
		return false
	}
}

// matchesSizeFilter checks if upload matches size filter
func (e *Engine) matchesSizeFilter(upload types.MultipartUpload, filter interfaces.SizeFilter) bool {
	filterSize, err := e.parseSizeBytes(filter.Value)
	if err != nil {
		// This should not happen if validation was done properly
		return false
	}
	
	switch filter.Operator {
	case ">":
		return upload.Size > filterSize
	case "<":
		return upload.Size < filterSize
	case ">=":
		return upload.Size >= filterSize
	case "<=":
		return upload.Size <= filterSize
	case "=":
		return upload.Size == filterSize
	case "!=":
		return upload.Size != filterSize
	default:
		return false
	}
}

// matchesStringFilter checks if a string value matches string filter
func (e *Engine) matchesStringFilter(value string, filter interfaces.StringFilter) bool {
	switch filter.Operator {
	case "=":
		return strings.EqualFold(value, filter.Value)
	case "!=":
		return !strings.EqualFold(value, filter.Value)
	default:
		return false
	}
}