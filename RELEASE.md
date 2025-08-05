# Release Guide

This document outlines the steps to make s3mpc globally available for public use.

## Pre-Release Checklist

- [ ] All tests pass (`go test ./...`)
- [ ] Binary builds successfully (`make build`)
- [ ] Documentation is up to date
- [ ] Version numbers are updated
- [ ] CHANGELOG.md is updated

## Release Process

### 1. Prepare the Repository

1. **Create a GitHub repository**:
   ```bash
   # Initialize git if not already done
   git init
   git add .
   git commit -m "Initial commit"
   
   # Add remote origin
   git remote add origin https://github.com/yourusername/s3mpc.git
   git push -u origin main
   ```

2. **Set up repository structure**:
   - Ensure all files are committed
   - Add proper .gitignore
   - Include LICENSE file
   - Add comprehensive README.md

### 2. GitHub Releases

1. **Create a release**:
   - Go to GitHub repository → Releases → Create a new release
   - Tag version: `v1.0.0`
   - Release title: `s3mpc v1.0.0`
   - Description: Include changelog and installation instructions

2. **Build multi-platform binaries**:
   ```bash
   # Build for all platforms
   make build-all
   
   # This creates:
   # - s3mpc-linux-amd64
   # - s3mpc-darwin-amd64
   # - s3mpc-darwin-arm64
   # - s3mpc-windows-amd64.exe
   ```

3. **Upload binaries to release**:
   - Attach all platform binaries to the GitHub release
   - Include checksums file

### 3. Package Managers

#### Go Modules
The tool is automatically available via:
```bash
go install github.com/yourusername/s3mpc/cmd/s3mpc@latest
```

#### Homebrew (macOS/Linux)
1. **Create a Homebrew formula**:
   ```ruby
   # Formula/s3mpc.rb
   class S3mpc < Formula
     desc "S3 MultiPart Cleaner - Manage incomplete S3 multipart uploads"
     homepage "https://github.com/yourusername/s3mpc"
     url "https://github.com/yourusername/s3mpc/archive/v1.0.0.tar.gz"
     sha256 "your-sha256-hash"
     license "MIT"

     depends_on "go" => :build

     def install
       system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/s3mpc"
     end

     test do
       system "#{bin}/s3mpc", "version"
     end
   end
   ```

2. **Submit to Homebrew**:
   - Fork homebrew-core
   - Add formula
   - Submit pull request

#### Snap (Linux)
1. **Create snapcraft.yaml**:
   ```yaml
   name: s3mpc
   version: '1.0.0'
   summary: S3 MultiPart Cleaner
   description: |
     A command-line tool for managing incomplete S3 multipart uploads.
     Helps discover, analyze, and clean up incomplete uploads.

   grade: stable
   confinement: strict

   parts:
     s3mpc:
       plugin: go
       source: .
       build-snaps: [go]

   apps:
     s3mpc:
       command: bin/s3mpc
       plugs: [network, home]
   ```

2. **Build and publish**:
   ```bash
   snapcraft
   snapcraft upload s3mpc_1.0.0_amd64.snap
   ```

#### Chocolatey (Windows)
1. **Create chocolatey package**:
   ```xml
   <!-- s3mpc.nuspec -->
   <?xml version="1.0" encoding="utf-8"?>
   <package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
     <metadata>
       <id>s3mpc</id>
       <version>1.0.0</version>
       <title>S3 MultiPart Cleaner</title>
       <authors>Your Name</authors>
       <description>Command-line tool for managing incomplete S3 multipart uploads</description>
       <projectUrl>https://github.com/yourusername/s3mpc</projectUrl>
       <licenseUrl>https://github.com/yourusername/s3mpc/blob/main/LICENSE</licenseUrl>
       <tags>aws s3 cli tool</tags>
     </metadata>
   </package>
   ```

### 4. Container Images

#### Docker Hub
1. **Create Dockerfile**:
   ```dockerfile
   FROM golang:1.21-alpine AS builder
   WORKDIR /app
   COPY . .
   RUN go build -o s3mpc cmd/s3mpc/main.go

   FROM alpine:latest
   RUN apk --no-cache add ca-certificates
   WORKDIR /root/
   COPY --from=builder /app/s3mpc .
   ENTRYPOINT ["./s3mpc"]
   ```

2. **Build and push**:
   ```bash
   docker build -t yourusername/s3mpc:latest .
   docker push yourusername/s3mpc:latest
   ```

### 5. Documentation Sites

#### GitHub Pages
1. **Create docs site**:
   - Enable GitHub Pages
   - Use Jekyll or static site generator
   - Include comprehensive documentation

#### Package Documentation
- Ensure pkg.go.dev documentation is complete
- Add examples and usage guides

### 6. Community and Distribution

#### Package Repositories
1. **Arch Linux AUR**:
   - Create PKGBUILD
   - Submit to AUR

2. **Debian/Ubuntu**:
   - Create .deb package
   - Submit to repositories

3. **RPM-based distributions**:
   - Create .rpm package
   - Submit to repositories

#### Cloud Marketplaces
1. **AWS Marketplace**:
   - Package as AMI or container
   - Submit for approval

2. **Azure Marketplace**:
   - Create Azure application
   - Submit for certification

### 7. Automation

#### GitHub Actions
Create `.github/workflows/release.yml`:
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Build binaries
      run: make build-all
    
    - name: Create Release
      uses: softprops/action-gh-release@v1
      with:
        files: |
          s3mpc-*
        generate_release_notes: true
```

#### Automated Testing
- Set up CI/CD pipeline
- Run tests on multiple platforms
- Automated security scanning

### 8. Marketing and Adoption

#### Technical Communities
- Post on Reddit (r/aws, r/golang, r/devops)
- Share on Hacker News
- Write blog posts
- Create video tutorials

#### Documentation
- Comprehensive README
- Usage examples
- Best practices guide
- Troubleshooting guide

#### Support Channels
- GitHub Issues for bug reports
- Discussions for questions
- Discord/Slack community

## Post-Release

### Monitoring
- Track download statistics
- Monitor GitHub stars/forks
- Collect user feedback

### Maintenance
- Regular security updates
- Bug fixes
- Feature enhancements
- Documentation updates

### Version Management
- Follow semantic versioning
- Maintain changelog
- Deprecation notices
- Migration guides

## Success Metrics

- Download counts
- GitHub stars
- Community contributions
- Issue resolution time
- User satisfaction

## Legal Considerations

- Ensure proper licensing (MIT recommended)
- Include copyright notices
- Comply with AWS terms of service
- Consider trademark issues