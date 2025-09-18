# Python Lambda Project Examples

This directory contains examples of different Python project structures supported by SST. The system automatically detects your project configuration and builds accordingly - no specific "layout types" required.

## Examples Overview

| Example | Structure | Description | Best For |
|---------|-----------|-------------|----------|
| [workspace-layout](./workspace-layout/) | Modern | Python project with pyproject.toml | New projects, teams |
| [flat-layout](./flat-layout/) | Simple | Handler at root level | Simple functions, prototypes |
| [nested-layout](./nested-layout/) | Complex | Nested directory structure | Large applications |
| [monorepo-layout](./monorepo-layout/) | Multi-service | Multiple services in one repository | Microservices, teams |

## Quick Start

Each example includes:
- Complete project structure
- SST configuration
- Handler implementation
- Dependencies configuration
- README with setup instructions

## Running Examples

1. **Choose an example**:
```bash
cd examples/python-layouts/workspace-layout
```

2. **Install dependencies**:
```bash
# For UV-based projects
uv sync

# For Poetry projects
poetry install

# For pip-based projects
pip install -r requirements.txt
```

3. **Deploy with SST**:
```bash
sst deploy
```

## Structure Comparison

### File Structure Examples

```
workspace-layout/           flat-layout/              nested-layout/
├── pyproject.toml         ├── handler.py            ├── pyproject.toml
├── uv.lock               ├── requirements.txt       ├── app/
├── src/                  └── utils.py               │   └── functions/
│   └── mypackage/                                   │       └── api/
│       ├── __init__.py                              │           └── handler.py
│       └── handler.py                               └── shared/
└── sst.config.ts                                        └── utils.py
```

### Handler Configuration Examples

```typescript
// Modern structure
new Function("ModernFunction", {
  handler: "src/mypackage/handler.main"
})

// Simple structure  
new Function("SimpleFunction", {
  handler: "handler.main"
})

// Complex structure
new Function("ComplexFunction", {
  handler: "app/functions/api/handler.main"
})
```

## Performance Comparison

| Structure | First Build | Cached Build | Incremental |
|-----------|-------------|--------------|-------------|
| Modern | 45s | 2s | 12s |
| Simple | 30s | 1s | 8s |
| Complex | 50s | 2s | 15s |
| Multi-service | 60s | 3s | 20s |

## Best Practices by Structure

### Modern Structure (Recommended)
- ✅ Use for new projects
- ✅ Best caching performance
- ✅ Modern Python packaging
- ✅ Team collaboration friendly

### Simple Structure
- ✅ Use for simple functions
- ✅ Fastest builds
- ✅ Easy to understand
- ⚠️ Limited scalability

### Complex Structure
- ✅ Use for large applications
- ✅ Good organization
- ⚠️ Slightly slower builds
- ⚠️ More complex setup

### Multi-service Structure
- ✅ Use for microservices
- ✅ Shared dependencies
- ✅ Consistent tooling
- ⚠️ Complex dependency management

## Migration Paths

### From Simple to Modern Structure
```bash
# 1. Create pyproject.toml
uv init

# 2. Move handler to src/
mkdir -p src/mypackage
mv handler.py src/mypackage/

# 3. Update SST config
# handler: "handler.main" → "src/mypackage/handler.main"
```

### From requirements.txt to pyproject.toml
```bash
# 1. Initialize UV project
uv init

# 2. Add dependencies
uv add $(cat requirements.txt)

# 3. Remove old file
rm requirements.txt
```

### From Poetry to UV
```bash
# 1. Keep pyproject.toml [project] section
# 2. Remove [tool.poetry] sections
# 3. Generate new lock file
uv lock

# 4. Remove poetry.lock
rm poetry.lock
```

## Troubleshooting

### Common Issues

#### Handler Not Found
```bash
# Enable debug logging
export SST_DEBUG=python:*
sst deploy
```

#### Build Cache Issues
```bash
# Clear cache
rm -rf .sst/cache

# Disable cache temporarily
export SST_PYTHON_DISABLE_CACHE=true
```

#### Dependency Issues
```bash
# Regenerate lock file
uv lock --upgrade

# Clear UV cache
uv cache clean
```

### Getting Help

- Check the main [documentation](../../docs/python-lambda-improvements.md)
- Join our [Discord](https://discord.gg/sst)
- Open an [issue](https://github.com/sst/sst/issues)

## Contributing

To add a new example:

1. Create a new directory with descriptive name
2. Include complete working project
3. Add README with setup instructions
4. Update this main README
5. Test the example works with `sst deploy`
6. Submit a pull request

### Example Template

```
new-example/
├── README.md              # Setup and usage instructions
├── sst.config.ts         # SST configuration
├── pyproject.toml        # Python dependencies (if applicable)
├── requirements.txt      # Alternative dependencies (if applicable)
├── src/                  # Source code
│   └── handler.py
└── tests/                # Tests (optional)
    └── test_handler.py
```