package container

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"golang.org/x/time/rate"

	"github.com/s3mpc/s3mpc/internal/config"
	"github.com/s3mpc/s3mpc/internal/logging"
	"github.com/s3mpc/s3mpc/pkg/aws"
	"github.com/s3mpc/s3mpc/pkg/filter"
	"github.com/s3mpc/s3mpc/pkg/interfaces"
	"github.com/s3mpc/s3mpc/pkg/services"
)

// Container holds all service dependencies
type Container struct {
	// Configuration
	config *config.Config
	
	// AWS clients
	s3Client        *s3.Client
	s3ClientWrapper *aws.S3Client
	pricingClient   *pricing.Client
	
	// Core services
	uploadService     interfaces.UploadService
	bucketService     interfaces.BucketService
	costCalculator    interfaces.CostCalculator
	filterEngine      interfaces.FilterEngine
	ageService        interfaces.AgeService
	dryRunService     interfaces.DryRunService
	exportService     interfaces.ExportService
	outputFormatter   interfaces.OutputFormatter
	sizeService       interfaces.SizeService
	
	// Logging
	logger *logging.Logger
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *config.Config) (*Container, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	
	container := &Container{
		config: cfg,
	}
	
	// Initialize logging first
	if err := container.initializeLogging(); err != nil {
		return nil, fmt.Errorf("failed to initialize logging: %w", err)
	}
	
	if err := container.initializeAWSClients(); err != nil {
		return nil, fmt.Errorf("failed to initialize AWS clients: %w", err)
	}
	
	if err := container.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}
	
	return container, nil
}

// initializeAWSClients sets up AWS service clients
func (c *Container) initializeAWSClients() error {
	ctx := context.Background()
	
	// Load AWS configuration
	var opts []func(*awsconfig.LoadOptions) error
	
	awsConfig := c.config.AWS()
	if awsConfig.Profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(awsConfig.Profile))
	}
	
	if awsConfig.Region != "" {
		opts = append(opts, awsconfig.WithRegion(awsConfig.Region))
	}
	
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	// Initialize S3 client
	c.s3Client = s3.NewFromConfig(cfg)
	
	// Initialize S3 client wrapper with retry logic and rate limiting
	awsConf := c.config.AWS()
	perfConfig := c.config.Performance()
	s3ClientConfig := aws.ClientConfig{
		Profile:   awsConf.Profile,
		Region:    awsConf.Region,
		RateLimit: rate.Limit(perfConfig.RateLimitRPS),
	}
	
	c.s3ClientWrapper, err = aws.NewS3Client(ctx, s3ClientConfig)
	if err != nil {
		return fmt.Errorf("failed to create S3 client wrapper: %w", err)
	}
	
	// Initialize Pricing client (always use us-east-1 for pricing API)
	pricingCfg := cfg.Copy()
	pricingCfg.Region = "us-east-1"
	c.pricingClient = pricing.NewFromConfig(pricingCfg)
	
	return nil
}

// initializeServices sets up service implementations
func (c *Container) initializeServices() error {
	// Initialize bucket service
	c.bucketService = services.NewBucketService(c.s3ClientWrapper)
	
	// Initialize cost calculator
	c.costCalculator = services.NewCostService()
	
	// Initialize filter engine
	c.filterEngine = filter.NewEngine()
	
	// Initialize age service
	c.ageService = services.NewAgeService()
	
	// Initialize dry-run service
	c.dryRunService = services.NewDryRunService(c.costCalculator)
	
	// Initialize export service
	c.exportService = services.NewExportService()
	
	// Initialize output formatter
	c.outputFormatter = services.NewOutputFormatter()
	
	// Initialize upload service with dry-run service
	c.uploadService = services.NewUploadServiceWithConcurrency(
		c.s3ClientWrapper, 
		c.bucketService, 
		c.dryRunService, 
		c.config.Performance().Concurrency,
	)
	
	// Initialize size service (depends on upload service)
	c.sizeService = services.NewSizeServiceWithConcurrency(c.uploadService, c.config.Performance().Concurrency)
	
	return nil
}

// GetUploadService returns the upload service instance
func (c *Container) GetUploadService() interfaces.UploadService {
	return c.uploadService
}

// GetBucketService returns the bucket service instance
func (c *Container) GetBucketService() interfaces.BucketService {
	return c.bucketService
}

// GetCostCalculator returns the cost calculator instance
func (c *Container) GetCostCalculator() interfaces.CostCalculator {
	return c.costCalculator
}

// GetFilterEngine returns the filter engine instance
func (c *Container) GetFilterEngine() interfaces.FilterEngine {
	return c.filterEngine
}

// GetAgeService returns the age service instance
func (c *Container) GetAgeService() interfaces.AgeService {
	return c.ageService
}

// GetDryRunService returns the dry-run service instance
func (c *Container) GetDryRunService() interfaces.DryRunService {
	return c.dryRunService
}

// GetExportService returns the export service instance
func (c *Container) GetExportService() interfaces.ExportService {
	return c.exportService
}

// GetOutputFormatter returns the output formatter instance
func (c *Container) GetOutputFormatter() interfaces.OutputFormatter {
	return c.outputFormatter
}

// GetSizeService returns the size service instance
func (c *Container) GetSizeService() interfaces.SizeService {
	return c.sizeService
}

// GetS3Client returns the S3 client
func (c *Container) GetS3Client() *s3.Client {
	return c.s3Client
}

// GetS3ClientWrapper returns the S3 client wrapper
func (c *Container) GetS3ClientWrapper() *aws.S3Client {
	return c.s3ClientWrapper
}

// GetPricingClient returns the Pricing client
func (c *Container) GetPricingClient() *pricing.Client {
	return c.pricingClient
}

// GetConfig returns the container configuration
func (c *Container) GetConfig() *config.Config {
	return c.config
}

// SetUploadService sets the upload service (for dependency injection)
func (c *Container) SetUploadService(service interfaces.UploadService) {
	c.uploadService = service
}

// SetBucketService sets the bucket service (for dependency injection)
func (c *Container) SetBucketService(service interfaces.BucketService) {
	c.bucketService = service
}

// SetCostCalculator sets the cost calculator (for dependency injection)
func (c *Container) SetCostCalculator(calculator interfaces.CostCalculator) {
	c.costCalculator = calculator
}

// SetFilterEngine sets the filter engine (for dependency injection)
func (c *Container) SetFilterEngine(engine interfaces.FilterEngine) {
	c.filterEngine = engine
}

// SetAgeService sets the age service (for dependency injection)
func (c *Container) SetAgeService(service interfaces.AgeService) {
	c.ageService = service
}

// SetDryRunService sets the dry-run service (for dependency injection)
func (c *Container) SetDryRunService(service interfaces.DryRunService) {
	c.dryRunService = service
}

// SetExportService sets the export service (for dependency injection)
func (c *Container) SetExportService(service interfaces.ExportService) {
	c.exportService = service
}

// SetOutputFormatter sets the output formatter (for dependency injection)
func (c *Container) SetOutputFormatter(formatter interfaces.OutputFormatter) {
	c.outputFormatter = formatter
}

// SetSizeService sets the size service (for dependency injection)
func (c *Container) SetSizeService(service interfaces.SizeService) {
	c.sizeService = service
}

// GetLogger returns the logger instance
func (c *Container) GetLogger() *logging.Logger {
	return c.logger
}

// initializeLogging sets up the logging system
func (c *Container) initializeLogging() error {
	var loggers []*logging.Logger
	
	// Console logger
	appConfig := c.config.App()
	loggingConfig := c.config.Logging()
	consoleLogger := logging.NewConsoleLogger(appConfig.Verbose, appConfig.Quiet)
	loggers = append(loggers, consoleLogger)
	
	// File logger if specified
	if loggingConfig.File != "" {
		level := logging.LevelInfo
		if appConfig.Verbose {
			level = logging.LevelDebug
		}
		
		fileLogger, err := logging.NewFileLogger(loggingConfig.File, level)
		if err != nil {
			return fmt.Errorf("failed to create file logger: %w", err)
		}
		loggers = append(loggers, fileLogger)
	}
	
	// Create multi-logger or single logger
	if len(loggers) == 1 {
		c.logger = loggers[0]
	} else {
		c.logger = logging.NewMultiLogger(loggers...)
	}
	
	// Set as global logger
	logging.SetGlobalLogger(c.logger)
	
	c.logger.Info("Logging system initialized", map[string]interface{}{
		"verbose":  appConfig.Verbose,
		"quiet":    appConfig.Quiet,
		"log_file": loggingConfig.File,
	})
	
	return nil
}