#!/bin/bash

echo "Creating GitHub release for SST Python Runtime Fixes v6.0.0..."

# Get the current commit hash for reference
COMMIT_HASH=$(git rev-parse --short HEAD)
echo "Building from commit: $COMMIT_HASH"

# Build binaries for all platforms
echo "Building binaries..."
GOOS=linux GOARCH=amd64 go build -o sst-linux-amd64 ./cmd/sst
GOOS=linux GOARCH=arm64 go build -o sst-linux-arm64 ./cmd/sst
GOOS=darwin GOARCH=amd64 go build -o sst-darwin-amd64 ./cmd/sst
GOOS=darwin GOARCH=arm64 go build -o sst-darwin-arm64 ./cmd/sst
GOOS=windows GOARCH=amd64 go build -o sst-windows-amd64.exe ./cmd/sst

echo "Binaries built successfully!"

# Create the release
gh release create python-fixes-v6.0.0 \
  --title "SST Python Runtime Fixes v6.0.0" \
  --notes "## 🚀 SST Python Runtime Fixes v6.0.0

**Built from**: python-deployment-fixes branch (commit: $COMMIT_HASH)

### 🎯 What's New in v6.0.0
- ✅ **Enhanced Dependency Detection**: Improved detection of uv.lock and pyproject.toml changes
- ✅ **Better Deployment Filtering**: Properly excludes test files and dev dependencies from deployments
- ✅ **Content-Based Change Detection**: More accurate detection of meaningful file changes
- ✅ **Modern UV Layout Support**: Full support for modern uv project structures
- ✅ **Infinite Restart Prevention**: Fixed file ignore logic to prevent restart loops

### 🔧 Key Improvements
**Dependency Change Detection**:
- Tracks uv.lock and pyproject.toml modifications
- Triggers proper rebuilds when dependencies change
- Handles both modern and legacy project layouts

**Deployment Filtering**:
- Excludes test files (test_*.py, *_test.py, conftest.py, tests/ directories)
- Filters out dev dependencies
- Reduces deployment package size
- Faster deployments

**File Change Handling**:
- Ignores files that should not trigger restarts (pycache, .pyc, etc)
- Prevents infinite restart loops
- Better logging for debugging

### 🔧 Previous Fixes (v5.0.0)
- ✅ Smart File Syncing for modern layouts
- ✅ PYTHONPATH Support
- ✅ Simplified Bridge
- ✅ Layout Detection

### 🔧 Previous Fixes (v4.0.0)
- ✅ Lazy Restart Implementation
- ✅ Restart Signal Tracking
- ✅ Race Condition Fix

### 🔧 Previous Fixes (v3.0.0)
- ✅ Source Code Project Detection
- ✅ Smart Dependency Handling
- ✅ Runtime Dependency Fix

### 📥 Download & Use
1. Download the binary for your platform
2. Make executable: \`chmod +x sst-*\`
3. Use in your projects: \`./sst-linux-amd64 dev\`

### 🔗 Permanent URLs
- **Linux x64**: https://github.com/subssn21/sst/releases/download/python-fixes-v6.0.0/sst-linux-amd64
- **Linux ARM64**: https://github.com/subssn21/sst/releases/download/python-fixes-v6.0.0/sst-linux-arm64
- **macOS Intel**: https://github.com/subssn21/sst/releases/download/python-fixes-v6.0.0/sst-darwin-amd64  
- **macOS ARM**: https://github.com/subssn21/sst/releases/download/python-fixes-v6.0.0/sst-darwin-arm64
- **Windows**: https://github.com/subssn21/sst/releases/download/python-fixes-v6.0.0/sst-windows-amd64.exe

### ⚠️ Key Improvement
This release focuses on stability and correctness - better dependency detection, cleaner deployments, and prevention of infinite restart loops.

🧪 **Pre-release for testing**" \
  --prerelease \
  sst-linux-amd64 sst-linux-arm64 sst-darwin-amd64 sst-darwin-arm64 sst-windows-amd64.exe

echo "✅ Release created! Check: https://github.com/subssn21/sst/releases"
echo "🔗 Linux x64 binary URL: https://github.com/subssn21/sst/releases/download/python-fixes-v6.0.0/sst-linux-amd64"
echo "🔗 Linux ARM64 binary URL: https://github.com/subssn21/sst/releases/download/python-fixes-v6.0.0/sst-linux-arm64"
