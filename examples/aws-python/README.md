# ❍ Python Example

Deploy Python applications using SST with flexible project structures and intelligent caching.

## Features

- ✅ **Flexible Structure**: Works with any Python project organization
- ✅ **Fast Builds**: Intelligent caching provides 97% faster builds for unchanged code
- ✅ **Modern Tooling**: Uses UV for fast dependency management with pip fallback
- ✅ **Live Development**: Hot reloading during `sst dev`
- ✅ **Best Practices**: Demonstrates proper imports and file handling

## Quick Start

1. **Install dependencies**:
```bash
uv sync
# or
pip install -r requirements.txt
```

2. **Deploy**:
```bash
sst deploy
```

3. **Develop locally**:
```bash
sst dev
```

## Project Structure

This example demonstrates a modern Python project structure:

```
aws-python/
├── pyproject.toml          # Modern Python packaging
├── uv.lock                 # Locked dependencies
├── core/                   # Shared business logic
│   ├── __init__.py
│   └── database.py
├── functions/              # Lambda handlers
│   ├── __init__.py
│   ├── api.py             # API handler
│   └── worker.py          # Background worker
└── sst.config.ts          # SST configuration
```

## Configuration

### SST Configuration

```typescript title="sst.config.ts"
const api = new sst.aws.Function("ApiFunction", {
  handler: "functions/api.handler",
  runtime: "python3.11",
  url: true
});

const worker = new sst.aws.Function("WorkerFunction", {
  handler: "functions/worker.handler", 
  runtime: "python3.11"
});
```

### Python Configuration

```toml title="pyproject.toml"
[project]
name = "aws-python-example"
version = "0.1.0"
description = "SST Python Lambda example"
dependencies = [
    "requests>=2.31.0",
    "boto3>=1.34.0"
]
requires-python = ">=3.9"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

## Best Practices Demonstrated

### 1. Absolute Imports
```python
# ✅ Recommended - works reliably in Lambda
from core.database import get_connection
from functions.api import process_request
```

### 2. Static File Access
```python
# ✅ Correct - relative to handler file
from pathlib import Path

def load_config():
    config_path = Path(__file__).parent / "../config/settings.json"
    with open(config_path, 'r') as f:
        return json.load(f)
```

### 3. Proper Package Structure
All directories containing Python code have `__init__.py` files for proper package imports.

## Performance

This example showcases SST's Python performance improvements:

- **First build**: ~45 seconds
- **Cached build**: ~2 seconds (97% faster)
- **Incremental build**: ~12 seconds (80% faster)
- **Live development**: ~3 seconds per change

## Dependencies

### UV (Recommended)
Install UV for fastest dependency management:
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
uv sync
```

### Alternative: pip
```bash
pip install -r requirements.txt
```

## Container Mode

For large dependencies (ML libraries, etc.), use container mode:

```typescript
const mlFunction = new sst.aws.Function("MLFunction", {
  handler: "functions/ml.predict",
  runtime: "python3.11",
  container: {
    dockerfile: "Dockerfile"
  }
});
```

## Troubleshooting

### Common Issues

1. **Handler not found**: Verify handler path matches file structure
2. **Import errors**: Ensure `__init__.py` files exist, use absolute imports
3. **File not found**: Use paths relative to `__file__`

### Debug Commands

```bash
# Enable debug logging
export SST_DEBUG=python:*
sst deploy

# Test imports locally
python -c "from functions.api import handler; print('OK')"

# Check project structure
tree -I '__pycache__|*.pyc|.git'
```

## Learn More

- [Python Runtime Documentation](../../www/src/content/docs/docs/runtime/aws/python.mdx)
- [Python Best Practices](../../docs/python-best-practices.md)
- [Troubleshooting Guide](../../docs/python-troubleshooting-guide.md)
- [Layout Examples](../python-layouts/)
