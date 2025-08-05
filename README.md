# s3mpc - S3 MultiPart Cleaner

A powerful command-line tool for managing incomplete S3 multipart uploads across AWS accounts. s3mpc helps you discover, analyze, and clean up incomplete uploads that can accumulate over time and incur unnecessary storage costs.

## Features

- **Storage Analysis**: Calculate total storage usage of incomplete multipart uploads
- **Cost Estimation**: Estimate monthly storage costs with regional pricing
- **Age Distribution**: Analyze upload age patterns to identify abandoned uploads
- **Flexible Filtering**: Filter uploads by age, size, storage class, region, or bucket
- **Safe Deletion**: Delete uploads with confirmation prompts and dry-run mode
- **Data Export**: Export upload data to CSV or JSON for further analysis
- **Concurrent Processing**: Configurable concurrency for optimal performance
- **AWS Integration**: Uses standard AWS credential chain with retry logic

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/s3mpc/s3mpc.git
cd s3mpc

# Build the binary
make build

# Or use the build script
./build.sh
```

### Using Go Install

```bash
go install github.com/s3mpc/s3mpc/cmd/s3mpc@latest
```

## Quick Start

### Prerequisites

- AWS credentials configured (via AWS CLI, environment variables, or IAM roles)
- Required IAM permissions:
  - `s3:ListAllMyBuckets`
  - `s3:ListBucket`
  - `s3:ListMultipartUploads`
  - `s3:AbortMultipartUpload` (for deletion)
  - `pricing:GetProducts` (for cost calculations)

### Basic Usage

```bash
# Show total storage usage
s3mpc size

# Calculate estimated costs
s3mpc cost

# List all incomplete uploads
s3mpc list

# Show age distribution
s3mpc age

# Delete uploads older than 7 days (with confirmation)
s3mpc delete --older-than 7d

# Preview what would be deleted (dry run)
s3mpc delete --older-than 30d --dry-run

# Export data to CSV
s3mpc export --format csv
```

## Commands

### `size` - Storage Usage Analysis

Calculate and display total storage usage of incomplete multipart uploads.

```bash
# Basic usage
s3mpc size

# Show per-bucket breakdown
s3mpc size --bucket

# Output in JSON format
s3mpc size --json

# Focus on specific region
s3mpc --region us-west-2 size
```

### `cost` - Cost Estimation

Calculate estimated monthly storage costs based on AWS S3 pricing.

```bash
# Show total estimated costs
s3mpc cost

# Show cost breakdown by storage class
s3mpc cost --storage-class

# Output in JSON format
s3mpc cost --json
```

### `list` - Detailed Upload Listing

List incomplete uploads with detailed information including bucket, key, upload ID, age, size, and storage class.

```bash
# List all uploads
s3mpc list

# List uploads for specific bucket
s3mpc list --bucket my-bucket

# Filter uploads older than 7 days
s3mpc list --filter "age>7d"

# Sort by size and limit results
s3mpc list --sort-by size --limit 10

# Use pagination
s3mpc list --limit 20 --offset 40
```

### `age` - Age Distribution Analysis

Display age distribution of uploads in time buckets to identify abandoned uploads.

```bash
# Show age distribution for all uploads
s3mpc age

# Show age distribution for specific bucket
s3mpc age --bucket my-bucket

# Output in JSON format
s3mpc age --json
```

### `delete` - Safe Upload Deletion

Delete incomplete uploads with safety features and filtering options.

```bash
# Delete all uploads (with confirmation)
s3mpc delete

# Delete uploads older than 7 days
s3mpc delete --older-than 7d

# Delete small uploads without confirmation
s3mpc delete --smaller-than 100MB --force

# Preview what would be deleted
s3mpc delete --older-than 30d --dry-run

# Delete from specific bucket
s3mpc delete --bucket my-bucket --force
```

### `export` - Data Export

Export upload data to structured files for analysis or reporting.

```bash
# Export all uploads to CSV
s3mpc export

# Export to JSON format
s3mpc export --format json

# Export specific bucket
s3mpc export --bucket my-bucket --format csv

# Export with filter
s3mpc export --filter "age>7d" --format json

# Export to specific file
s3mpc export --output my-uploads.csv
```

## Filtering

s3mpc supports powerful filtering syntax for precise upload selection:

### Filter Fields
- `age` - Upload age (e.g., `7d`, `1w`, `1m`, `1y`)
- `size` - Upload size (e.g., `100MB`, `1GB`, `500KB`)
- `storageClass` - Storage class (e.g., `STANDARD`, `STANDARD_IA`)
- `region` - AWS region (e.g., `us-east-1`, `eu-west-1`)
- `bucket` - Bucket name

### Filter Operators
- `>`, `<`, `>=`, `<=` - Comparison operators
- `=`, `!=` - Equality operators

### Filter Examples
```bash
# Uploads older than 7 days
--filter "age>7d"

# Large uploads in STANDARD storage
--filter "size>1GB,storageClass=STANDARD"

# Old uploads in specific region
--filter "age>30d,region=us-east-1"

# Small uploads in specific bucket
--filter "size<100MB,bucket=my-bucket"
```

## Global Options

- `--profile` - AWS profile to use
- `--region` - AWS region to focus on
- `--concurrency` - Number of concurrent operations (default: 10)
- `--verbose` - Enable verbose logging
- `--quiet` - Suppress non-essential output
- `--log-file` - Write logs to file

## Configuration

s3mpc uses the standard AWS credential chain and can be configured via:

### Environment Variables
- `AWS_PROFILE` - AWS profile
- `AWS_REGION` - AWS region
- `S3MPC_CONCURRENCY` - Concurrency level
- `S3MPC_VERBOSE` - Enable verbose logging
- `S3MPC_QUIET` - Enable quiet mode
- `S3MPC_LOG_FILE` - Log file path

### AWS Credentials
s3mpc supports all standard AWS credential methods:
- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- AWS credentials file (`~/.aws/credentials`)
- AWS config file (`~/.aws/config`)
- IAM roles for EC2 instances
- IAM roles for ECS tasks
- IAM roles for Lambda functions

## Examples

### Clean up old uploads across all buckets
```bash
# First, see what would be deleted
s3mpc delete --older-than 7d --dry-run

# If satisfied, execute the deletion
s3mpc delete --older-than 7d --force
```

### Analyze storage usage by bucket
```bash
# Get detailed size breakdown
s3mpc size --bucket --json > storage-report.json

# Export detailed upload list
s3mpc export --format csv --output detailed-uploads.csv
```

### Focus on specific bucket cleanup
```bash
# Analyze specific bucket
s3mpc age --bucket my-important-bucket

# Clean up old uploads in that bucket
s3mpc delete --bucket my-important-bucket --older-than 30d
```

## Building from Source

### Prerequisites
- Go 1.21 or later
- Git (for version information)

### Build Commands
```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Clean build artifacts
make clean

# Install to GOPATH/bin
make install
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run tests: `make test`
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/s3mpc/s3mpc).

## Security

s3mpc follows AWS security best practices:
- Uses standard AWS credential chain
- Supports IAM roles and temporary credentials
- Logs all destructive operations
- Validates all user inputs
- Implements proper error handling

For security issues, please report them privately to the maintainers.