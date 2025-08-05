# Installation Guide

This guide provides multiple ways to install s3mpc globally on your system.

## Prerequisites

- Go 1.21 or later (for building from source)
- AWS credentials configured (via AWS CLI, environment variables, or IAM roles)
- Required IAM permissions:
  - `s3:ListAllMyBuckets`
  - `s3:ListBucket`
  - `s3:ListMultipartUploads`
  - `s3:AbortMultipartUpload` (for deletion)
  - `pricing:GetProducts` (for cost calculations)

## Installation Methods

### Method 1: Download Pre-built Binary (Recommended)

1. Go to the [Releases page](https://github.com/s3mpc/s3mpc/releases)
2. Download the appropriate binary for your platform:
   - `s3mpc-linux-amd64` for Linux
   - `s3mpc-darwin-amd64` for macOS (Intel)
   - `s3mpc-darwin-arm64` for macOS (Apple Silicon)
   - `s3mpc-windows-amd64.exe` for Windows

3. Make the binary executable and move it to your PATH:

**Linux/macOS:**
```bash
# Download and install
curl -L -o s3mpc https://github.com/s3mpc/s3mpc/releases/latest/download/s3mpc-linux-amd64
chmod +x s3mpc
sudo mv s3mpc /usr/local/bin/

# Verify installation
s3mpc version
```

**Windows:**
```powershell
# Download s3mpc-windows-amd64.exe
# Rename to s3mpc.exe and add to your PATH
```

### Method 2: Install via Go

If you have Go installed, you can install s3mpc directly:

```bash
go install github.com/s3mpc/s3mpc/cmd/s3mpc@latest
```

This will install the binary to `$GOPATH/bin` (or `$HOME/go/bin` if GOPATH is not set).

### Method 3: Build from Source

1. Clone the repository:
```bash
git clone https://github.com/s3mpc/s3mpc.git
cd s3mpc
```

2. Build and install:
```bash
make build
sudo cp s3mpc /usr/local/bin/
```

Or use the install target:
```bash
make install
```

### Method 4: Using Package Managers

#### Homebrew (macOS/Linux)
```bash
# Add the tap (once available)
brew tap s3mpc/s3mpc
brew install s3mpc
```

#### Snap (Linux)
```bash
# Once available on Snap Store
sudo snap install s3mpc
```

#### Chocolatey (Windows)
```powershell
# Once available on Chocolatey
choco install s3mpc
```

## Verification

After installation, verify that s3mpc is working correctly:

```bash
# Check version
s3mpc version

# Check help
s3mpc --help

# Test with your AWS credentials (this will list your buckets)
s3mpc size --dry-run
```

## Configuration

### AWS Credentials

s3mpc uses the standard AWS credential chain. Configure your credentials using one of these methods:

1. **AWS CLI** (recommended):
```bash
aws configure
```

2. **Environment variables**:
```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_DEFAULT_REGION=us-east-1
```

3. **IAM roles** (for EC2/ECS/Lambda)

4. **Shared credentials file** (`~/.aws/credentials`)

### Shell Completion

Enable shell completion for better user experience:

**Bash:**
```bash
s3mpc completion bash > /etc/bash_completion.d/s3mpc
```

**Zsh:**
```bash
s3mpc completion zsh > "${fpath[1]}/_s3mpc"
```

**Fish:**
```bash
s3mpc completion fish > ~/.config/fish/completions/s3mpc.fish
```

**PowerShell:**
```powershell
s3mpc completion powershell > s3mpc.ps1
```

## Troubleshooting

### Common Issues

1. **Permission denied**: Make sure the binary is executable (`chmod +x s3mpc`)
2. **Command not found**: Ensure the binary is in your PATH
3. **AWS credentials not found**: Configure AWS credentials as described above
4. **Access denied to S3**: Ensure your AWS credentials have the required permissions

### Getting Help

- Check the built-in help: `s3mpc --help`
- View command-specific help: `s3mpc [command] --help`
- Report issues: [GitHub Issues](https://github.com/s3mpc/s3mpc/issues)

## Uninstallation

To remove s3mpc:

```bash
# If installed to /usr/local/bin
sudo rm /usr/local/bin/s3mpc

# If installed via Go
rm $GOPATH/bin/s3mpc

# If installed via package manager, use the appropriate uninstall command
```