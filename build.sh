#!/bin/bash

# Build script for s3mpc
set -e

# Version information
VERSION=${VERSION:-"1.0.0"}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT} -X main.GitBranch=${GIT_BRANCH}"

echo "Building s3mpc..."
echo "Version: ${VERSION}"
echo "Build Time: ${BUILD_TIME}"
echo "Git Commit: ${GIT_COMMIT}"
echo "Git Branch: ${GIT_BRANCH}"

# Clean previous builds
rm -f s3mpc

# Build for current platform
go build -ldflags "${LDFLAGS}" -o s3mpc cmd/s3mpc/main.go

echo "Build completed successfully!"
echo "Binary: ./s3mpc"

# Make executable
chmod +x s3mpc

# Test the build
echo "Testing build..."
./s3mpc version
echo "Build test passed!"