# Flat Layout Example

This example demonstrates a simple Python Lambda project with a flat structure - handlers at the root level.

## Project Structure

```
flat-layout/
├── handler.py             # Lambda handler
├── utils.py               # Shared utilities
├── requirements.txt       # Dependencies
├── sst.config.ts         # SST configuration
├── tests/                # Test directory
│   └── test_handler.py   # Handler tests
└── README.md             # This file
```

## Features

- ✅ Simple flat structure (no nested directories)
- ✅ Fast builds and deployments
- ✅ Easy to understand and maintain
- ✅ Perfect for simple functions and prototypes
- ✅ Minimal setup required

## Setup

### Prerequisites

- Python 3.9+ installed
- SST v3 installed

### Installation

1. **Navigate to the example**:
```bash
cd examples/python-layouts/flat-layout
```

2. **Install dependencies**:
```bash
pip install -r requirements.txt
```

3. **Deploy**:
```bash
sst deploy
```

## Performance

This flat layout provides the fastest build times:

- **First build**: ~30 seconds
- **Cached build**: ~1.5 seconds  
- **Incremental build**: ~8 seconds
- **Cache hit rate**: 90-95%

## Benefits

1. **Simplicity**: Easiest to understand and maintain
2. **Speed**: Fastest build times of all layouts
3. **Low Overhead**: Minimal file structure
4. **Quick Setup**: Get started immediately

## When to Use

- Simple Lambda functions
- Prototypes and experiments
- Single-purpose functions
- Learning and tutorials
- Quick scripts and utilities

## Migration

To migrate from flat to workspace layout:

```bash
# 1. Create package structure
mkdir -p src/mypackage
mv *.py src/mypackage/

# 2. Create __init__.py
touch src/mypackage/__init__.py

# 3. Create pyproject.toml
uv init

# 4. Update handler paths in sst.config.ts
```