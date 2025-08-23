# Python Lambda Migration Guide

This guide helps you migrate your existing Python Lambda functions to take advantage of the new flexible layouts, intelligent caching, and performance improvements in SST v3.

## Overview

The Python Lambda improvements introduce several changes that can significantly improve your development experience:

- **Flexible project layouts** - No more rigid `src/{package}/handler.py` requirement
- **Intelligent caching** - Faster builds through smart change detection
- **Modern Python tooling** - Support for `pyproject.toml` and UV
- **Better error handling** - More helpful error messages and fallbacks

## Migration Scenarios

### Scenario 1: No Changes Required (Recommended Start)

**Current setup**: Using `src/{package}/handler.py` structure
**Action**: None - your existing code will continue to work

Your existing functions will continue to work without any changes. The new system is fully backward compatible.

```typescript
// This continues to work exactly as before
new Function("MyFunction", {
  handler: "src/mypackage/handler.main",
  runtime: "python3.11"
})
```

**Benefits you get immediately**:
- ✅ Intelligent caching (faster builds)
- ✅ Better error messages
- ✅ Progress reporting
- ✅ Fallback mechanisms

### Scenario 2: Modernize Dependencies

**Current setup**: Using `requirements.txt`
**Goal**: Adopt `pyproject.toml` and UV for faster dependency management

#### Step-by-step migration:

1. **Install UV** (if not already installed):
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

2. **Initialize UV project**:
```bash
# In your project directory
uv init --no-readme
```

3. **Add your existing dependencies**:
```bash
# If you have requirements.txt
uv add $(cat requirements.txt | grep -v '^#' | grep -v '^$' | tr '\n' ' ')

# Or add dependencies individually
uv add requests boto3 click
```

4. **Verify dependencies**:
```bash
uv tree
```

5. **Test your function**:
```bash
sst dev
```

6. **Remove old files** (when ready):
```bash
rm requirements.txt
```

**Benefits**:
- ⚡ Faster dependency installation
- 🔒 Better dependency locking
- 📦 Modern Python packaging
- 🚀 Improved build performance

### Scenario 3: Flexible Layout

**Current setup**: Rigid `src/{package}/handler.py` structure
**Goal**: Move handlers to more convenient locations

#### Option A: Flatten Structure

```bash
# Before
src/mypackage/handler.py

# After  
handler.py
```

**Migration steps**:
1. Move handler file:
```bash
mv src/mypackage/handler.py ./handler.py
```

2. Update SST configuration:
```typescript
// Before
new Function("MyFunction", {
  handler: "src/mypackage/handler.main"
})

// After
new Function("MyFunction", {
  handler: "handler.main"
})
```

3. Update imports in handler (if needed):
```python
# Update any relative imports
# from .utils import helper
# to
# from utils import helper
```

#### Option B: Organize by Function

```bash
# Before
src/mypackage/handler.py

# After
functions/
├── api/
│   └── handler.py
├── worker/
│   └── handler.py
└── auth/
    └── handler.py
```

**Migration steps**:
1. Create function directories:
```bash
mkdir -p functions/api functions/worker functions/auth
```

2. Move and organize handlers:
```bash
cp src/mypackage/handler.py functions/api/handler.py
# Customize each handler for its specific purpose
```

3. Update SST configuration:
```typescript
new Function("ApiFunction", {
  handler: "functions/api/handler.main"
})

new Function("WorkerFunction", {
  handler: "functions/worker/handler.main"
})
```

### Scenario 4: Complete Modernization

**Current setup**: Old structure with `requirements.txt`
**Goal**: Modern workspace layout with `pyproject.toml` and UV

#### Migration steps:

1. **Create new project structure**:
```bash
# Create modern structure
mkdir -p src/myproject
mkdir -p tests

# Move existing code
mv src/mypackage/* src/myproject/
# or
mv *.py src/myproject/

# Ensure __init__.py exists
touch src/myproject/__init__.py
```

2. **Initialize modern Python project**:
```bash
uv init --no-readme
```

3. **Configure pyproject.toml**:
```toml
[project]
name = "my-lambda-project"
version = "0.1.0"
description = "My Lambda functions"
dependencies = [
    "requests>=2.31.0",
    "boto3>=1.34.0"
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0.0",
    "black>=23.0.0"
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

4. **Update SST configuration**:
```typescript
new Function("MyFunction", {
  handler: "src/myproject/handler.main",
  runtime: "python3.11"
})
```

5. **Test and deploy**:
```bash
uv sync
sst dev
```

## Migration from Poetry

If you're currently using Poetry, you can migrate to UV while keeping most of your configuration:

### Step-by-step Poetry to UV migration:

1. **Keep the `[project]` section** of your `pyproject.toml`:
```toml
[project]
name = "my-project"
version = "0.1.0"
dependencies = [
    "requests>=2.31.0"
]
```

2. **Remove Poetry-specific sections**:
```toml
# Remove these sections:
# [tool.poetry]
# [tool.poetry.dependencies]
# [tool.poetry.dev-dependencies]
```

3. **Generate UV lock file**:
```bash
uv lock
```

4. **Remove Poetry files**:
```bash
rm poetry.lock
```

5. **Update scripts** (if any):
```bash
# Replace poetry commands
# poetry install → uv sync
# poetry add package → uv add package
# poetry run pytest → uv run pytest
```

## Migration from Django Projects

If you have Django projects with Lambda handlers:

### Current structure:
```
myproject/
├── manage.py
├── myproject/
│   ├── settings.py
│   └── urls.py
├── myapp/
│   ├── models.py
│   ├── views.py
│   └── lambda_handlers.py
└── requirements.txt
```

### Migration options:

#### Option 1: Keep Django Structure
```typescript
// No changes needed - this works as-is
new Function("DjangoFunction", {
  handler: "myapp/lambda_handlers.my_handler",
  runtime: "python3.11"
})
```

#### Option 2: Separate Lambda Handlers
```bash
# Create dedicated handlers directory
mkdir lambda_handlers
mv myapp/lambda_handlers.py lambda_handlers/api.py
```

```typescript
new Function("ApiFunction", {
  handler: "lambda_handlers/api.my_handler",
  runtime: "python3.11"
})
```

## Performance Optimization Migration

### Enable All Optimizations

Update your SST configuration to enable all performance features:

```typescript
// sst.config.ts
export default $config({
  app(input) {
    return {
      name: "my-app",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws"
    };
  },
  async run() {
    // Enable optimizations globally
    const pythonConfig = {
      runtime: "python3.11",
      timeout: "30 seconds",
      // Enable all optimizations
      environment: {
        SST_PYTHON_ENABLE_CACHE: "true",
        SST_PYTHON_PARALLEL_BUILDS: "true",
        SST_PYTHON_SHOW_PROGRESS: "true"
      }
    };

    const api = new sst.aws.Function("ApiFunction", {
      handler: "src/myproject/handler.api_handler",
      ...pythonConfig
    });

    return { api: api.url };
  }
});
```

### Cache Configuration

Optimize cache settings for your project:

```bash
# Environment variables for cache tuning
export SST_PYTHON_CACHE_DIR=".sst/python-cache"
export SST_PYTHON_CACHE_SIZE="2GB"
export SST_PYTHON_CACHE_AGE="48h"
export SST_PYTHON_MAX_PARALLEL="4"
```

## Testing Your Migration

### Before Migration Checklist

1. **Document current setup**:
```bash
# Save current structure
tree > before-migration.txt

# Save current dependencies
cp requirements.txt requirements-backup.txt
# or
cp pyproject.toml pyproject-backup.toml
```

2. **Test current deployment**:
```bash
sst deploy --stage migration-test
```

3. **Measure current performance**:
```bash
time sst deploy --stage migration-test
```

### After Migration Testing

1. **Test new structure**:
```bash
# Test locally first
sst dev

# Test deployment
sst deploy --stage migration-test
```

2. **Verify functionality**:
```bash
# Test your endpoints/functions
curl https://your-api-url/health

# Check logs
sst logs --stage migration-test
```

3. **Measure performance improvement**:
```bash
# First build (should be similar)
time sst deploy --stage migration-test

# Second build (should be much faster)
time sst deploy --stage migration-test
```

## Rollback Plan

If you encounter issues during migration:

### Quick Rollback

1. **Restore files**:
```bash
# Restore from backup
cp requirements-backup.txt requirements.txt
# or
cp pyproject-backup.toml pyproject.toml

# Restore file structure if changed
git checkout -- .
```

2. **Deploy previous version**:
```bash
sst deploy --stage migration-test
```

### Gradual Migration

Instead of migrating everything at once:

1. **Migrate one function at a time**:
```typescript
// Keep old functions unchanged
new Function("OldFunction", {
  handler: "src/mypackage/handler.old_handler"
})

// Migrate new functions
new Function("NewFunction", {
  handler: "functions/new/handler.new_handler"
})
```

2. **Test each migration step**:
```bash
sst deploy --stage migration-test
```

3. **Gradually migrate all functions**

## Common Migration Issues

### Import Errors

**Problem**: `ModuleNotFoundError` after moving files

**Solution**: Update import statements and follow best practices
```python
# ❌ Avoid relative imports (can be unreliable in Lambda)
from .utils import helper
from ..shared import common

# ✅ Use absolute imports (recommended)
from myproject.utils import helper
from shared import common
from app.functions.auth.handler import auth_main
```

**Best Practice**: Always use absolute imports for Lambda functions. The deployment system preserves your directory structure and maintains proper Python package structure with `__init__.py` files.

### Path Issues

**Problem**: Handler not found after restructuring

**Solution**: Verify handler paths match file structure
```typescript
// File: functions/api/handler.py
// Function: def main(event, context):
new Function("ApiFunction", {
  handler: "functions/api/handler.main"  // Correct path
})
```

### Static File Access Issues

**Problem**: `FileNotFoundError` when trying to read config files, templates, or other static assets

**Solution**: Use paths relative to your handler file
```python
import os
import json
from pathlib import Path

# ✅ Recommended approaches for static file access
def load_config():
    # Method 1: Using pathlib (modern)
    config_path = Path(__file__).parent / "../../data/config.json"
    with open(config_path, 'r') as f:
        return json.load(f)

def load_template():
    # Method 2: Using os.path
    template_path = os.path.join(os.path.dirname(__file__), "../templates/email.html")
    with open(template_path, 'r') as f:
        return f.read()

# ❌ Avoid working directory assumptions
def bad_example():
    # This may fail in Lambda environment
    with open("data/config.json", 'r') as f:
        return json.load(f)
```

**Best Practice**: The deployment system preserves your entire directory structure, including static files. Use paths relative to `__file__` for reliable file access in Lambda.

### Dependency Issues

**Problem**: Dependencies not installing correctly

**Solution**: Clear cache and reinstall
```bash
# For UV
uv cache clean
uv sync

# For pip
pip cache purge
pip install -r requirements.txt

# Clear SST cache
rm -rf .sst/cache
```

### Performance Regression

**Problem**: Builds are slower after migration

**Solution**: Check cache configuration
```bash
# Enable debug logging
export SST_DEBUG=python:*
sst deploy

# Check cache status
ls -la .sst/cache/python/
```

## Best Practices for Migration

### 1. Start Small
- Migrate one function at a time
- Test each change thoroughly
- Keep old structure until confident

### 2. Use Version Control
```bash
# Create migration branch
git checkout -b python-migration

# Commit each step
git add .
git commit -m "Step 1: Add pyproject.toml"
```

### 3. Test Thoroughly
- Test in development environment first
- Use staging environment for validation
- Monitor performance metrics

### 4. Document Changes
- Update README files
- Document new development workflow
- Share knowledge with team

### 5. Monitor After Migration
- Watch build times
- Monitor error rates
- Check cache hit rates

## Getting Help

### Debug Information

Enable debug logging to troubleshoot issues:
```bash
export SST_DEBUG=python:*
sst deploy
```

### Common Debug Commands

```bash
# Check project structure
tree -I '__pycache__|*.pyc|.git'

# Verify Python environment
python --version
which python

# Check dependencies
uv tree
# or
pip list

# Test imports
python -c "from src.myproject import handler; print('OK')"

# Check SST configuration
sst list functions
```

### Support Resources

- **Documentation**: [Python Lambda Improvements](./python-lambda-improvements.md)
- **Examples**: [Layout Examples](../examples/python-layouts/)
- **Discord**: https://discord.gg/sst
- **GitHub Issues**: https://github.com/sst/sst/issues

### Reporting Migration Issues

When reporting issues, include:

1. **Before/after structure**:
```bash
# Before migration
tree before/

# After migration  
tree after/
```

2. **SST configuration**:
```typescript
// Your sst.config.ts
```

3. **Error messages**:
```bash
# Full error output
sst deploy 2>&1 | tee error.log
```

4. **Debug logs**:
```bash
SST_DEBUG=python:* sst deploy 2>&1 | tee debug.log
```

## Migration Checklist

### Pre-Migration
- [ ] Backup current project
- [ ] Document current structure
- [ ] Test current deployment
- [ ] Measure current performance
- [ ] Plan rollback strategy

### Migration Steps
- [ ] Choose migration scenario
- [ ] Update dependencies (if applicable)
- [ ] Restructure files (if applicable)
- [ ] Update SST configuration
- [ ] Update import statements
- [ ] Test locally with `sst dev`

### Post-Migration
- [ ] Test deployment
- [ ] Verify functionality
- [ ] Measure performance improvement
- [ ] Update documentation
- [ ] Train team on new structure
- [ ] Monitor for issues

### Cleanup
- [ ] Remove backup files
- [ ] Clean up old cache
- [ ] Update CI/CD pipelines
- [ ] Archive old documentation

## Success Metrics

After migration, you should see:

- **Build time reduction**: 50-90% for unchanged code
- **Cache hit rate**: 80-95% in development
- **Error clarity**: More actionable error messages
- **Development speed**: Faster iteration cycles

## Conclusion

The Python Lambda improvements provide significant benefits with minimal migration effort. Start with the no-changes approach to get immediate benefits, then gradually adopt modern practices as your project evolves.

Remember: **backward compatibility is maintained**, so you can migrate at your own pace without breaking existing functionality.