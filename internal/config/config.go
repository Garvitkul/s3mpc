package config

// Config holds container configuration
type Config struct {
	AWSProfile   string
	AWSRegion    string
	Concurrency  int
	RateLimitRPS float64
	Verbose      bool
	Quiet        bool
	LogFile      string
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Concurrency:  10,
		RateLimitRPS: 10.0,
		Verbose:      false,
		Quiet:        false,
	}
}

// AWS returns AWS configuration
func (c *Config) AWS() AWSConfig {
	return AWSConfig{
		Profile: c.AWSProfile,
		Region:  c.AWSRegion,
	}
}

// Performance returns performance configuration
func (c *Config) Performance() PerformanceConfig {
	return PerformanceConfig{
		Concurrency:  c.Concurrency,
		RateLimitRPS: c.RateLimitRPS,
	}
}

// App returns application configuration
func (c *Config) App() AppConfig {
	return AppConfig{
		Verbose: c.Verbose,
		Quiet:   c.Quiet,
	}
}

// Logging returns logging configuration
func (c *Config) Logging() LoggingConfig {
	return LoggingConfig{
		File: c.LogFile,
	}
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	Profile string
	Region  string
}

// PerformanceConfig holds performance-related configuration
type PerformanceConfig struct {
	Concurrency  int
	RateLimitRPS float64
}

// AppConfig holds application-level configuration
type AppConfig struct {
	Verbose bool
	Quiet   bool
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	File string
}