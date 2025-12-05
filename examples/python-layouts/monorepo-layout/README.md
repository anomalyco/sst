# Multi-Service Structure Example

This example demonstrates a multi-service structure with multiple Python services in a single repository, featuring **per-package dependency isolation** for optimized Lambda deployment sizes.

## Project Structure

```
monorepo-layout/
├── pyproject.toml          # Root workspace configuration
├── sst.config.ts           # SST configuration
├── uv.lock                 # Locked dependencies
├── services/               # Individual services
│   ├── api/                # API service (FastAPI, boto3)
│   │   ├── pyproject.toml  # API-specific dependencies
│   │   └── handler.py
│   ├── auth/               # Authentication service (PyJWT, bcrypt)
│   │   ├── pyproject.toml  # Auth-specific dependencies
│   │   └── handler.py
│   └── worker/             # Worker service (Celery, Redis)
│       ├── pyproject.toml  # Worker-specific dependencies
│       └── handler.py
├── shared/                 # Shared utilities package
│   ├── pyproject.toml      # Shared package definition
│   ├── __init__.py
│   └── utils.py
└── README.md
```

## Features

- ✅ **Per-Package Dependency Isolation**: Each service only includes its own dependencies
- ✅ Multiple services in one repository with shared utilities
- ✅ Independent service deployments with optimized bundle sizes
- ✅ UV workspaces for consistent dependency management
- ✅ Shared code as a proper Python package

## Per-Package Dependency Isolation

The key feature of this layout is that **each Lambda function only includes its own dependencies**, not the entire workspace's dependencies. This dramatically reduces cold start times for services with lightweight dependencies.

### Example Results

| Service           | Dependencies                       | Bundle Size |
| ----------------- | ---------------------------------- | ----------- |
| **ApiService**    | boto3, requests, fastapi, pydantic | ~19 MB      |
| **AuthService**   | pyjwt, bcrypt                      | ~337 KB     |
| **WorkerService** | celery, redis                      | ~2.8 MB     |

Without isolation, all three would be ~22 MB each!

### How It Works

1. **Root `pyproject.toml`**: Defines the UV workspace with minimal dependencies
2. **Service `pyproject.toml`**: Each service declares only its specific dependencies
3. **Shared package**: Common utilities are a proper package that services depend on
4. **SST builds**: Automatically detects workspace members and exports only their dependencies

### Service Configuration

Each service's `pyproject.toml` should:

1. Define a unique `[project].name`
2. List only dependencies needed by that service
3. Reference `shared` using `{ workspace = true }` to include the shared utilities

Example (`services/api/pyproject.toml`):

```toml
[project]
name = "api-service"
version = "0.1.0"
dependencies = [
    "boto3>=1.34.0",
    "fastapi>=0.104.0",
    "shared",  # Local shared package from workspace
]

# Reference workspace members using { workspace = true }
# This tells UV to resolve 'shared' from the workspace defined in root pyproject.toml
[tool.uv.sources]
shared = { workspace = true }
```

The root `pyproject.toml` defines which packages are workspace members:

```toml
[tool.uv.workspace]
members = ["services/*", "shared"]
```

SST automatically:

1. Detects workspace members and their dependencies
2. Exports only the dependencies needed by each service
3. Copies workspace package source code into Lambda artifacts
4. Filters out editable installs that won't work in Lambda

## Setup

### Prerequisites

- Python 3.12+ installed
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

## Use Case: Heavy Dependencies

This pattern is ideal when you have:

- One function needing heavy ML dependencies (e.g., `pandas`)
- Other functions that are lightweight API handlers

Instead of every Lambda being bloated by pandas, only the function that needs the heavy dependencies gets them.

## Benefits

1. **Reduced Cold Starts**: Smaller bundles mean faster Lambda initialization
2. **Cost Optimization**: Smaller deployments, faster uploads
3. **Code Sharing**: Shared utilities without dependency bloat
4. **Independent Scaling**: Each service sized appropriately for its needs

## When to Use

- Microservices with varying dependency requirements
- Functions where cold start time is critical
- Heavy ML/data processing isolated to specific functions
- Large monorepos with diverse Lambda functions
