# Monorepo Layout Example

This example demonstrates a monorepo structure with multiple Python services in a single repository.

## Project Structure

```
monorepo-layout/
├── pyproject.toml          # Root project configuration
├── sst.config.ts          # SST configuration
├── services/              # Individual services
│   ├── auth/              # Authentication service
│   │   ├── pyproject.toml
│   │   └── handler.py
│   ├── api/               # API service
│   │   ├── pyproject.toml
│   │   └── handler.py
│   └── worker/            # Worker service
│       ├── pyproject.toml
│       └── handler.py
├── shared/                # Shared utilities
│   ├── __init__.py
│   └── utils.py
└── README.md              # This file
```

## Features

- ✅ Multiple services in one repository
- ✅ Shared dependencies and utilities
- ✅ Independent service deployments
- ✅ Consistent tooling across services
- ✅ Efficient dependency management

## Setup

### Prerequisites

- Python 3.9+ installed
- UV installed (recommended)
- SST v3 installed

### Installation

1. **Navigate to the example**:
```bash
cd examples/python-layouts/monorepo-layout
```

2. **Install dependencies**:
```bash
uv sync
```

3. **Deploy**:
```bash
sst deploy
```

## Benefits

1. **Code Sharing**: Shared utilities and dependencies
2. **Consistency**: Same tooling and standards across services
3. **Efficiency**: Reduced duplication and maintenance
4. **Coordination**: Easier cross-service changes

## When to Use

- Microservices architecture
- Multiple related Lambda functions
- Team collaboration on related services
- Shared business logic across functions