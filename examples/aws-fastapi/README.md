# ❍ FastAPI Python Example

Deploy FastAPI applications using SST with flexible project structures and intelligent caching.

## Features

- ✅ **FastAPI Integration**: Full FastAPI support with automatic routing
- ✅ **Flexible Structure**: Works with any Python project organization  
- ✅ **Fast Builds**: Intelligent caching provides 97% faster builds for unchanged code
- ✅ **Modern Tooling**: Uses UV for fast dependency management with pip fallback
- ✅ **Live Development**: Hot reloading during `sst dev`
- ✅ **Container Support**: Easy deployment of complex FastAPI applications

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

This example demonstrates a FastAPI project structure:

```
aws-fastapi/
├── pyproject.toml          # Modern Python packaging
├── uv.lock                 # Locked dependencies
├── core/                   # Shared business logic
│   ├── __init__.py
│   └── models.py
├── functions/              # FastAPI handlers
│   ├── __init__.py
│   └── api.py             # FastAPI application
└── sst.config.ts          # SST configuration
```

## Configuration

### SST Configuration

```typescript title="sst.config.ts"
const api = new sst.aws.Function("FastApiFunction", {
  handler: "functions/api.handler",
  runtime: "python3.11",
  url: true
});
```

### Python Configuration

```toml title="pyproject.toml"
[project]
name = "aws-fastapi-example"
version = "0.1.0"
description = "SST FastAPI Lambda example"
dependencies = [
    "fastapi>=0.104.0",
    "mangum>=0.17.0",  # ASGI adapter for Lambda
    "uvicorn>=0.24.0"
]
requires-python = ">=3.9"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

## FastAPI Handler Example

```python title="functions/api.py"
from fastapi import FastAPI
from mangum import Mangum
from core.models import Item

app = FastAPI(title="My FastAPI App")

@app.get("/")
def read_root():
    return {"message": "Hello from FastAPI on Lambda!"}

@app.get("/items/{item_id}")
def read_item(item_id: int, q: str = None):
    return {"item_id": item_id, "q": q}

@app.post("/items/")
def create_item(item: Item):
    return {"message": f"Created item: {item.name}"}

# Lambda handler
handler = Mangum(app)
```

## Best Practices Demonstrated

### 1. ASGI Adapter
Uses Mangum to adapt FastAPI (ASGI) for AWS Lambda environment.

### 2. Absolute Imports
```python
# ✅ Recommended - works reliably in Lambda
from core.models import Item
from core.database import get_connection
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

## Container Mode (Recommended for FastAPI)

For FastAPI applications with many dependencies, container mode often works better:

```typescript title="sst.config.ts"
const api = new sst.aws.Function("FastApiFunction", {
  handler: "functions/api.handler",
  runtime: "python3.11",
  container: {
    dockerfile: "Dockerfile"
  },
  url: true
});
```

```dockerfile title="Dockerfile"
FROM public.ecr.aws/lambda/python:3.11

# Copy requirements and install dependencies
COPY pyproject.toml uv.lock ./
RUN pip install uv && uv sync --frozen

# Copy application code
COPY . .

# Set the CMD to your handler
CMD ["functions.api.handler"]
```

## API Documentation

FastAPI automatically generates API documentation:

- **Swagger UI**: `https://your-api-url/docs`
- **ReDoc**: `https://your-api-url/redoc`
- **OpenAPI JSON**: `https://your-api-url/openapi.json`

## Troubleshooting

### Common Issues

1. **Handler not found**: Verify handler path matches file structure
2. **Import errors**: Ensure `__init__.py` files exist, use absolute imports
3. **ASGI adapter issues**: Ensure Mangum is properly configured

### Debug Commands

```bash
# Enable debug logging
export SST_DEBUG=python:*
sst deploy

# Test FastAPI locally
uvicorn functions.api:app --reload

# Test imports
python -c "from functions.api import handler; print('OK')"
```

## Learn More

- [FastAPI Documentation](https://fastapi.tiangolo.com/)
- [Mangum Documentation](https://mangum.io/)
- [Python Runtime Documentation](../../www/src/content/docs/docs/runtime/aws/python.mdx)
- [Python Best Practices](../../docs/python-best-practices.md)
- [Troubleshooting Guide](../../docs/python-troubleshooting-guide.md)
