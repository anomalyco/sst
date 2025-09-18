# Python Lambda Troubleshooting Guide

This guide helps you resolve common issues when developing and deploying Python Lambda functions with SST.

## Common Configuration Issues

### Handler Not Found

**Error Message:**
```
Error: Handler 'src/myapp/handler.py' not found

Searched in:
├── src/myapp/handler.py
├── myapp/handler.py
├── handler.py
└── app/handler.py

💡 Suggestions:
• Check that your handler file exists
• Verify the handler path in your Function configuration
• Ensure the file has a .py extension
```

**Solutions:**

1. **Verify file exists:**
```bash
ls -la src/myapp/handler.py
```

2. **Check handler path in SST config:**
```typescript
// File: src/myapp/handler.py, function: main
new Function("MyFunction", {
  handler: "src/myapp/handler.main"  // Must match file structure
})
```

3. **Common path mistakes:**
```typescript
// ❌ Wrong - missing .py extension in path
handler: "src/myapp/handler.py.main"

// ❌ Wrong - incorrect directory structure
handler: "myapp/handler.main"  // when file is in src/myapp/

// ✅ Correct - matches file structure
handler: "src/myapp/handler.main"
```

### Import Errors

**Error Message:**
```
ModuleNotFoundError: No module named 'mymodule'
```

**Solutions:**

1. **Add missing `__init__.py` files:**
```bash
# Ensure all package directories have __init__.py
touch src/__init__.py
touch src/mypackage/__init__.py
touch shared/__init__.py
```

2. **Use absolute imports:**
```python
# ✅ Recommended - absolute imports
from shared.utils import helper_function
from mypackage.submodule import MyClass

# ❌ Avoid - relative imports can fail
from .utils import helper_function
from ..shared import common_function
```

3. **Check project structure:**
```
project/
├── shared/
│   ├── __init__.py          # Required
│   └── utils.py
├── src/
│   ├── __init__.py          # Required
│   └── mypackage/
│       ├── __init__.py      # Required
│       └── handler.py
└── pyproject.toml
```

### File Not Found Errors

**Error Message:**
```
FileNotFoundError: [Errno 2] No such file or directory: 'config.json'
```

**Solutions:**

1. **Use paths relative to handler file:**
```python
import json
from pathlib import Path

# ✅ Correct - relative to handler file
def load_config():
    config_path = Path(__file__).parent / "../../data/config.json"
    with open(config_path, 'r') as f:
        return json.load(f)

# ❌ Wrong - assumes working directory
def bad_load_config():
    with open("data/config.json", 'r') as f:  # May fail
        return json.load(f)
```

2. **Verify file exists in project:**
```bash
find . -name "config.json" -type f
```

3. **Check file is not excluded:**
```bash
# Files in these directories are automatically excluded:
# .git, .gitignore, __pycache__, .pytest_cache, .venv, venv
```

### Dependency Issues

**Error Message:**
```
ImportError: No module named 'requests'
```

**Solutions:**

1. **For pyproject.toml projects:**
```bash
# Add missing dependency
uv add requests

# Or manually add to pyproject.toml
[project]
dependencies = [
    "requests>=2.31.0"
]
```

2. **For requirements.txt projects:**
```bash
# Add to requirements.txt
echo "requests>=2.31.0" >> requirements.txt

# Install locally to test
pip install -r requirements.txt
```

3. **Clear cache and rebuild:**
```bash
# Clear SST cache
rm -rf .sst/cache

# Clear UV cache (if using UV)
uv cache clean

# Redeploy
sst deploy
```

### Build Performance Issues

**Error Message:**
```
Build taking too long (>5 minutes)
```

**Solutions:**

1. **Enable caching (should be automatic):**
```bash
export SST_PYTHON_ENABLE_CACHE="true"
```

2. **Check cache hit rate:**
```bash
# Enable debug logging
export SST_DEBUG=python:cache
sst deploy
```

3. **Optimize dependencies:**
```bash
# Remove unused dependencies
uv remove unused-package

# Use UV for faster installs
curl -LsSf https://astral.sh/uv/install.sh | sh
```

4. **Enable parallel builds:**
```bash
export SST_PYTHON_PARALLEL_BUILDS="true"
export SST_PYTHON_MAX_PARALLEL="4"
```

## Development Issues

### Live Development Not Working

**Error Message:**
```
Function failed to start in dev mode
```

**Solutions:**

1. **Check Python version compatibility:**
```toml
# In pyproject.toml - match Lambda runtime
[project]
requires-python = ">=3.11"
```

2. **Verify handler function exists:**
```python
# In your handler file
def main(event, context):  # Function name must match config
    return {"statusCode": 200}
```

3. **Test handler locally:**
```bash
python -c "from src.mypackage.handler import main; print('Handler found')"
```

### Hot Reloading Not Working

**Solutions:**

1. **Check file watching:**
```bash
# Ensure files are not in excluded directories
# .git, node_modules, .venv, __pycache__ are ignored
```

2. **Restart dev mode:**
```bash
# Stop and restart
sst dev
```

3. **Check for syntax errors:**
```bash
python -m py_compile src/mypackage/handler.py
```

## Deployment Issues

### Lambda Package Too Large

**Error Message:**
```
Unzipped size must be smaller than 262144000 bytes
```

**Solutions:**

1. **Use container mode for large dependencies:**
```typescript
new Function("MLFunction", {
  handler: "src/ml/handler.predict",
  runtime: "python3.11",
  container: {
    dockerfile: "Dockerfile"
  }
})
```

2. **Remove unnecessary files:**
```bash
# Check what's being included
export SST_DEBUG=python:package
sst deploy
```

3. **Optimize dependencies:**
```bash
# Remove dev dependencies in production
uv sync --no-dev
```

### Memory or Timeout Issues

**Error Message:**
```
Task timed out after 30.00 seconds
```

**Solutions:**

1. **Increase timeout:**
```typescript
new Function("SlowFunction", {
  handler: "src/handler.main",
  timeout: "5 minutes",  // Increase as needed
  memory: "1024 MB"      // Increase memory if needed
})
```

2. **Optimize code:**
```python
# Cache expensive operations
from functools import lru_cache

@lru_cache(maxsize=1)
def get_config():
    # Expensive operation cached
    return load_config()
```

### Permission Issues

**Error Message:**
```
AccessDenied: User is not authorized to perform action
```

**Solutions:**

1. **Add required permissions:**
```typescript
new Function("MyFunction", {
  handler: "src/handler.main",
  permissions: [
    {
      actions: ["s3:GetObject"],
      resources: ["arn:aws:s3:::my-bucket/*"]
    }
  ]
})
```

2. **Check IAM credentials:**
```bash
aws sts get-caller-identity
```

## Debug Commands

### Project Structure Verification

```bash
# Check project structure
tree -I '__pycache__|*.pyc|.git|.venv'

# Verify Python files
find . -name "*.py" -type f | head -10

# Check for __init__.py files
find . -name "__init__.py" -type f
```

### Dependency Verification

```bash
# Check installed packages
pip list
# or
uv tree

# Verify specific package
python -c "import requests; print(requests.__version__)"

# Check Python version
python --version
which python
```

### Import Testing

```bash
# Test imports locally
python -c "from src.mypackage import handler; print('OK')"

# Test specific function
python -c "from src.mypackage.handler import main; print('Handler found')"

# Test file access
python -c "from pathlib import Path; print(Path('data/config.json').exists())"
```

### SST Debug Commands

```bash
# Enable all Python debugging
export SST_DEBUG=python:*
sst deploy

# Enable specific debugging
export SST_DEBUG=python:cache,python:build
sst deploy

# Check SST configuration
sst list functions

# View function logs
sst logs --function MyFunction --tail
```

### Cache Debugging

```bash
# Check cache status
ls -la .sst/cache/python/

# Clear cache
rm -rf .sst/cache

# Disable cache temporarily
export SST_PYTHON_DISABLE_CACHE="true"
sst deploy
```

## Performance Debugging

### Build Time Analysis

```bash
# Enable performance logging
export SST_DEBUG=python:performance
sst deploy

# Time builds
time sst deploy

# Check cache hit rate
export SST_DEBUG=python:cache
sst deploy | grep "Cache hit"
```

### Memory Usage

```bash
# Monitor memory during build
export SST_PYTHON_MEMORY_PROFILE="true"
sst deploy

# Check Lambda memory usage
sst logs --function MyFunction | grep "Memory"
```

## Getting Help

### Information to Include

When asking for help, include:

1. **SST version:**
```bash
sst version
```

2. **Python version:**
```bash
python --version
```

3. **Project structure:**
```bash
tree -I '__pycache__|*.pyc|.git' | head -20
```

4. **Error messages:**
```bash
# Full error output
sst deploy 2>&1 | tee error.log
```

5. **Debug logs:**
```bash
SST_DEBUG=python:* sst deploy 2>&1 | tee debug.log
```

### Support Resources

- **Documentation**: [Python Runtime Guide](../www/src/content/docs/docs/runtime/aws/python.mdx)
- **Examples**: [Python Layout Examples](../examples/python-layouts/)
- **Discord**: https://discord.gg/sst
- **GitHub Issues**: https://github.com/sst/sst/issues

### Quick Fixes Checklist

Before asking for help, try these quick fixes:

- [ ] Clear cache: `rm -rf .sst/cache`
- [ ] Check file paths match SST config
- [ ] Verify `__init__.py` files exist
- [ ] Test imports locally
- [ ] Check Python version compatibility
- [ ] Verify dependencies are installed
- [ ] Enable debug logging: `export SST_DEBUG=python:*`
- [ ] Try with a simple handler first

## Prevention Tips

### Project Setup

1. **Use modern Python packaging:**
```toml
# pyproject.toml
[project]
name = "my-project"
dependencies = ["requests>=2.31.0"]
requires-python = ">=3.9"
```

2. **Maintain proper structure:**
```bash
# Always include __init__.py in packages
touch src/__init__.py
touch src/mypackage/__init__.py
```

3. **Use absolute imports:**
```python
# Always prefer absolute imports
from mypackage.utils import helper
```

### Development Workflow

1. **Test locally first:**
```bash
python -c "from src.mypackage.handler import main; print('OK')"
```

2. **Use version control:**
```bash
git add .
git commit -m "Working version before changes"
```

3. **Monitor build performance:**
```bash
# Check build times regularly
time sst deploy
```

### Deployment Best Practices

1. **Pin dependency versions:**
```toml
dependencies = [
    "requests==2.31.0",  # Pinned for consistency
    "boto3>=1.34.0"      # Minimum version
]
```

2. **Use staging environments:**
```bash
sst deploy --stage staging
# Test thoroughly before production
sst deploy --stage production
```

3. **Monitor function performance:**
```bash
sst logs --function MyFunction --tail
```

This troubleshooting guide covers the most common issues you'll encounter with Python Lambda functions in SST. Most problems can be resolved by following the solutions provided above.