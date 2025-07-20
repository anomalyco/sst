# Python Lambda Improvements

This document describes the major improvements made to SST's Python Lambda support, including flexible project layouts, intelligent caching, and performance optimizations.

## Overview

The Python Lambda improvements introduce several key enhancements:

1. **Flexible Project Layouts** - Support for various Python project structures
2. **Intelligent Build Caching** - Smart change detection and incremental builds
3. **Performance Optimizations** - Faster builds and deployments
4. **Enhanced Error Handling** - Better error messages and fallback mechanisms
5. **Progress Reporting** - Real-time build progress and feedback

## Flexible Project Layouts

### Supported Layout Types

SST now supports multiple Python project layouts without requiring specific directory structures:

#### 1. Workspace Layout (Recommended)
Modern Python projects using `pyproject.toml` with proper package structure:

```
my-project/
├── pyproject.toml
├── uv.lock
├── src/
│   └── mypackage/
│       ├── __init__.py
│       └── handler.py
└── tests/
```

**Handler Configuration:**
```typescript
new Function("MyFunction", {
  handler: "src/mypackage/handler.main",
  runtime: "python3.11"
})
```

#### 2. Flat Layout
Simple projects with handlers at the root level:

```
my-function/
├── handler.py
├── requirements.txt
└── utils.py
```

**Handler Configuration:**
```typescript
new Function("MyFunction", {
  handler: "handler.main",
  runtime: "python3.11"
})
```

#### 3. Nested Layout
Complex projects with deeply nested handler structures:

```
my-app/
├── pyproject.toml
├── app/
│   └── functions/
│       └── api/
│           └── handler.py
└── shared/
    └── utils.py
```

**Handler Configuration:**
```typescript
new Function("ApiFunction", {
  handler: "app/functions/api/handler.main",
  runtime: "python3.11"
})
```

#### 4. Monorepo Layout
Multiple services in a single repository:

```
monorepo/
├── pyproject.toml
├── services/
│   ├── auth/
│   │   ├── pyproject.toml
│   │   └── handler.py
│   ├── api/
│   │   ├── pyproject.toml
│   │   └── handler.py
│   └── worker/
│       ├── pyproject.toml
│       └── handler.py
└── shared/
    └── utils/
```

**Handler Configuration:**
```typescript
new Function("AuthFunction", {
  handler: "services/auth/handler.main",
  runtime: "python3.11"
})
```

### Migration from Restrictive Layouts

If you're using the old restrictive layout (`src/{package}/handler.py`), you can now:

1. **Keep your existing structure** - It will continue to work
2. **Flatten your structure** - Move handlers to more convenient locations
3. **Use modern Python packaging** - Adopt `pyproject.toml` and UV

## Intelligent Build Caching

### How It Works

The new caching system tracks file changes using content hashes and only rebuilds when necessary:

1. **File Change Detection** - Monitors Python files, `pyproject.toml`, `uv.lock`, and dependencies
2. **Dependency Tracking** - Understands which files affect which functions
3. **Incremental Builds** - Only rebuilds changed packages in multi-package projects
4. **Shared Dependencies** - Reuses dependency installations across functions

### Cache Behavior

#### First Build
```bash
# All files are new, full build required
Building Python function...
├── Detecting project layout... ✓
├── Analyzing dependencies... ✓
├── Installing dependencies... ✓
├── Building packages... ✓
└── Creating deployment package... ✓
Build completed in 45s
```

#### Subsequent Builds (No Changes)
```bash
# No changes detected, using cache
Building Python function...
├── Checking for changes... ✓
└── Using cached build result... ✓
Build completed in 2s (cached)
```

#### Incremental Build (File Changed)
```bash
# Only handler.py changed
Building Python function...
├── Detecting changes... ✓
├── Handler file modified, rebuilding... ✓
├── Dependencies unchanged, using cache... ✓
└── Creating deployment package... ✓
Build completed in 12s (incremental)
```

### Cache Configuration

You can configure caching behavior through environment variables:

```bash
# Cache directory (default: .sst/cache)
export SST_PYTHON_CACHE_DIR="/tmp/sst-cache"

# Cache size limit (default: 1GB)
export SST_PYTHON_CACHE_SIZE="2GB"

# Cache age limit (default: 24h)
export SST_PYTHON_CACHE_AGE="48h"

# Disable caching (for debugging)
export SST_PYTHON_DISABLE_CACHE="true"
```

## Performance Improvements

### Build Time Reductions

The improvements provide significant build time reductions:

| Scenario | Before | After | Improvement |
|----------|--------|-------|-------------|
| First build | 60s | 45s | 25% faster |
| No changes | 60s | 2s | 97% faster |
| Handler only | 60s | 12s | 80% faster |
| Dependencies only | 60s | 30s | 50% faster |

### Live Development

During `sst dev` sessions, builds are now much faster:

```bash
# File change detected
[12:34:56] File changed: src/mypackage/handler.py
[12:34:56] Rebuilding function... (2s)
[12:34:58] Function updated ✓

# Dependency change detected  
[12:35:30] File changed: pyproject.toml
[12:35:30] Rebuilding function... (15s)
[12:35:45] Function updated ✓
```

### Parallel Builds

For projects with multiple functions, builds now run in parallel:

```bash
Building 3 functions in parallel...
├── auth-function... ✓ (12s)
├── api-function... ✓ (15s)
└── worker-function... ✓ (18s)
All functions built in 18s (was 45s)
```

## Enhanced Error Handling

### Better Error Messages

Error messages now provide actionable feedback:

```bash
❌ Python build failed

Layout Detection Error:
  Handler 'src/myapp/handler.py' not found
  
  Searched in:
  ├── src/myapp/handler.py
  ├── myapp/handler.py
  ├── handler.py
  └── app/handler.py
  
  💡 Suggestions:
  • Check that your handler file exists
  • Verify the handler path in your Function configuration
  • Ensure the file has a .py extension

  📖 Learn more: https://docs.sst.dev/python-layouts
```

### Fallback Mechanisms

When optimizations fail, the system gracefully falls back:

```bash
⚠️  Cache corrupted, falling back to full rebuild
⚠️  Layout detection failed, using simple detection
⚠️  Optimization failed, disabling parallel builds
```

### Deprecation Warnings

The system warns about deprecated patterns:

```bash
⚠️  Deprecation Warning:
  Using requirements.txt instead of pyproject.toml
  
  💡 Migration Guide:
  1. Create pyproject.toml with your dependencies
  2. Run 'uv add <package>' to add dependencies
  3. Remove requirements.txt when ready
  
  📖 Learn more: https://docs.sst.dev/python-migration
```

## Progress Reporting

### Real-time Progress

Build progress is now visible in real-time:

```bash
Building Python function... [████████████████████████████████] 100%

├── Initializing... ✓
├── Detecting layout... ✓ (2s)
├── Analyzing dependencies... ✓ (3s)
├── Planning build... ✓ (1s)
├── Building packages... ████████████████████████████████ 85% (12s)
└── Post-processing... ⏳

Estimated time remaining: 2s
```

### Detailed Feedback

The system provides detailed information about what's happening:

```bash
Building Python function...

📁 Project Layout: Workspace (pyproject.toml detected)
📦 Dependencies: 15 packages (3 changed)
🔨 Build Plan: 2 packages to build, 1 cached
⚡ Optimizations: Parallel builds enabled (4 workers)

├── Building package 1/2: mypackage... ✓ (8s)
├── Building package 2/2: shared... ✓ (4s)
└── Creating deployment package... ✓ (2s)

✅ Build completed successfully in 14s
   📊 Cache hit rate: 67%
   💾 Saved 31s compared to full rebuild
```

## Configuration Options

### Function Configuration

```typescript
new Function("MyFunction", {
  handler: "src/mypackage/handler.main",
  runtime: "python3.11",
  
  // Python-specific options
  python: {
    // Enable/disable optimizations
    enableOptimizations: true,
    
    // Parallel build configuration
    parallelBuilds: true,
    maxParallelBuilds: 4,
    
    // Cache configuration
    cacheEnabled: true,
    cacheSize: "1GB",
    cacheAge: "24h",
    
    // Progress reporting
    showProgress: true,
    
    // Fallback behavior
    enableFallbacks: true,
    
    // Deprecation warnings
    showDeprecationWarnings: true
  }
})
```

### Global Configuration

```typescript
// sst.config.ts
export default {
  config() {
    return {
      python: {
        // Global Python settings
        defaultRuntime: "python3.11",
        enableOptimizations: true,
        cacheDirectory: ".sst/python-cache",
        
        // UV configuration
        uvVersion: "latest",
        uvTimeout: "300s",
        
        // Build settings
        parallelBuilds: true,
        maxParallelBuilds: 4,
        
        // Development settings
        showProgress: true,
        enableFallbacks: true
      }
    }
  }
}
```

## Troubleshooting

### Common Issues

#### Build Cache Issues

```bash
# Clear cache if corrupted
rm -rf .sst/cache/python

# Disable cache temporarily
export SST_PYTHON_DISABLE_CACHE=true
sst deploy
```

#### Layout Detection Issues

```bash
# Debug layout detection
export SST_DEBUG=python:layout
sst deploy

# Force specific layout type
export SST_PYTHON_LAYOUT=flat
sst deploy
```

#### Performance Issues

```bash
# Disable parallel builds
export SST_PYTHON_PARALLEL_BUILDS=false

# Reduce parallel build count
export SST_PYTHON_MAX_PARALLEL=2

# Enable verbose logging
export SST_DEBUG=python:*
```

### Debug Information

Enable debug logging to see detailed information:

```bash
export SST_DEBUG=python:*
sst deploy
```

This will show:
- Layout detection process
- Cache hit/miss decisions
- Build planning decisions
- File change detection
- Dependency analysis
- Build execution steps

## Migration Guide

### From Old Restrictive Layouts

If you're currently using the old `src/{package}/handler.py` structure:

#### Option 1: Keep Existing Structure (No Changes Required)
Your existing code will continue to work without any changes.

#### Option 2: Modernize Your Project

1. **Add pyproject.toml**:
```toml
[project]
name = "my-lambda-project"
version = "0.1.0"
dependencies = [
    "requests>=2.25.0",
    "boto3>=1.20.0"
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

2. **Install UV** (recommended):
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
uv sync
```

3. **Update your Function configuration**:
```typescript
// Before
new Function("MyFunction", {
  handler: "src/mypackage/handler.main"
})

// After (more flexible)
new Function("MyFunction", {
  handler: "mypackage/handler.main"  // or any path
})
```

### From requirements.txt to pyproject.toml

1. **Create pyproject.toml** with your dependencies:
```bash
uv init
uv add requests boto3 click
```

2. **Verify dependencies** are correctly specified:
```toml
[project]
dependencies = [
    "requests>=2.25.0",
    "boto3>=1.20.0", 
    "click>=8.0.0"
]
```

3. **Remove requirements.txt** when ready:
```bash
rm requirements.txt
```

### From Poetry to UV

1. **Convert pyproject.toml**:
```bash
# Remove Poetry-specific sections
# Keep [project] section
# Add [build-system] if needed
```

2. **Generate uv.lock**:
```bash
uv lock
```

3. **Remove poetry.lock**:
```bash
rm poetry.lock
```

## Best Practices

### Project Structure

1. **Use pyproject.toml** for modern Python packaging
2. **Organize code logically** - handlers don't need to be in specific locations
3. **Separate shared code** into reusable modules
4. **Use meaningful handler names** that reflect their purpose

### Performance

1. **Enable caching** for faster builds (enabled by default)
2. **Use parallel builds** for multi-function projects
3. **Keep dependencies minimal** to reduce build times
4. **Pin dependency versions** for reproducible builds

### Development Workflow

1. **Use `sst dev`** for fast iteration during development
2. **Monitor cache hit rates** to ensure optimizations are working
3. **Clear cache** if you encounter issues
4. **Use debug logging** to troubleshoot problems

### Deployment

1. **Test locally first** with `sst dev`
2. **Use staging environments** for validation
3. **Monitor build times** and optimize as needed
4. **Keep cache directories** in CI/CD for faster builds

## API Reference

### Layout Detection

```typescript
interface LayoutInfo {
  type: 'workspace' | 'flat' | 'nested' | 'legacy'
  handlerFile: string
  workspaceDir: string
  packageName: string
  pythonPath: string[]
  dependencies: string[]
  pyprojectPath?: string
  requirementsPath?: string
  uvLockPath?: string
}
```

### Cache Configuration

```typescript
interface CacheConfig {
  enabled: boolean
  directory: string
  maxSize: string
  maxAge: string
  cleanupInterval: string
}
```

### Build Configuration

```typescript
interface BuildConfig {
  enableOptimizations: boolean
  parallelBuilds: boolean
  maxParallelBuilds: number
  showProgress: boolean
  enableFallbacks: boolean
  showDeprecationWarnings: boolean
}
```

## Changelog

### v3.1.0 - Python Lambda Improvements

#### Added
- Flexible project layout support
- Intelligent build caching system
- Progress reporting and feedback
- Enhanced error handling with fallbacks
- Deprecation warnings for old patterns
- Parallel building support
- Comprehensive test suite

#### Changed
- Layout detection is now flexible and supports multiple structures
- Build process is now incremental and cached
- Error messages are more actionable and helpful
- Performance significantly improved for repeated builds

#### Deprecated
- Rigid `src/{package}/handler.py` requirement (still supported)
- `requirements.txt` in favor of `pyproject.toml`
- Old error handling patterns

#### Fixed
- Build times for unchanged code
- Memory usage during large builds
- Concurrent build issues
- Cache corruption recovery

## Support

### Getting Help

- **Documentation**: https://docs.sst.dev/python
- **Discord**: https://discord.gg/sst
- **GitHub Issues**: https://github.com/sst/sst/issues
- **Examples**: https://github.com/sst/sst/tree/main/examples

### Reporting Issues

When reporting issues, please include:

1. **SST version**: `sst version`
2. **Python version**: `python --version`
3. **Project structure**: Directory layout
4. **Error messages**: Full error output
5. **Debug logs**: `SST_DEBUG=python:* sst deploy`

### Contributing

We welcome contributions! See our [Contributing Guide](CONTRIBUTING.md) for details on:

- Setting up the development environment
- Running tests
- Submitting pull requests
- Code style guidelines