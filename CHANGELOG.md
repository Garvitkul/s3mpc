# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01-05

### Added
- Initial release of s3mpc (S3 MultiPart Cleaner)
- **Core Commands**:
  - `size` - Calculate storage usage of incomplete multipart uploads
  - `cost` - Estimate monthly storage costs with regional pricing
  - `list` - List incomplete uploads with detailed information
  - `age` - Show age distribution analysis of uploads
  - `delete` - Safely delete incomplete uploads with confirmation
  - `export` - Export upload data to CSV or JSON formats
  - `version` - Show version information

- **Key Features**:
  - **Storage Analysis**: Calculate total storage usage across all buckets
  - **Cost Estimation**: Regional pricing support for accurate cost calculations
  - **Age Distribution**: Analyze upload patterns to identify abandoned uploads
  - **Flexible Filtering**: Filter by age, size, storage class, region, or bucket
  - **Safe Deletion**: Confirmation prompts and dry-run mode for safety
  - **Data Export**: Export to CSV or JSON for further analysis
  - **Concurrent Processing**: Configurable concurrency for optimal performance
  - **AWS Integration**: Standard AWS credential chain with retry logic

- **Advanced Capabilities**:
  - Multi-region support with automatic region detection
  - Comprehensive error handling and retry mechanisms
  - Rate limiting to prevent API throttling
  - Detailed progress reporting for long-running operations
  - Structured logging with configurable verbosity
  - Input validation and data integrity checks

- **Output Formats**:
  - Human-readable console output with tables
  - JSON output for programmatic use
  - CSV export for spreadsheet analysis
  - Detailed breakdown by bucket, region, and storage class

- **Safety Features**:
  - Dry-run mode for all destructive operations
  - Interactive confirmation prompts
  - Comprehensive validation of user inputs
  - Graceful error handling and recovery

- **Performance Optimizations**:
  - Concurrent processing of multiple buckets
  - Efficient pagination for large datasets
  - Memory-efficient streaming for exports
  - Caching of bucket region information

### Technical Details
- **Language**: Go 1.21+
- **Dependencies**: AWS SDK v2, Cobra CLI framework
- **Architecture**: Clean architecture with dependency injection
- **Testing**: Comprehensive unit tests with >80% coverage
- **Documentation**: Complete API documentation and usage examples

### Supported Platforms
- Linux (amd64)
- macOS (amd64, arm64)
- Windows (amd64)

### AWS Permissions Required
- `s3:ListAllMyBuckets`
- `s3:ListBucket`
- `s3:ListMultipartUploads`
- `s3:AbortMultipartUpload` (for deletion operations)
- `pricing:GetProducts` (for cost calculations)

### Installation Methods
- Direct binary download from GitHub releases
- Go install: `go install github.com/s3mpc/s3mpc/cmd/s3mpc@latest`
- Build from source with comprehensive Makefile
- Future support for Homebrew, Snap, and Chocolatey

[1.0.0]: https://github.com/s3mpc/s3mpc/releases/tag/v1.0.0