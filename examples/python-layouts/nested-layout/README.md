# Nested Layout Example

This example demonstrates a complex nested directory structure for organizing Lambda functions.

## Project Structure

```
nested-layout/
├── pyproject.toml          # Project configuration
├── sst.config.ts          # SST configuration
├── app/                   # Application directory
│   └── functions/         # Functions directory
│       ├── api/           # API functions
│       │   └── handler.py
│       ├── auth/          # Auth functions
│       │   └── handler.py
│       └── worker/        # Worker functions
│           └── handler.py
├── shared/                # Shared utilities
│   ├── __init__.py
│   ├── utils.py
│   └── models.py
└── README.md              # This file
```

## Features

- ✅ Organized structure for complex applications
- ✅ Clear separation of concerns
- ✅ Shared utilities across functions
- ✅ Scalable for large projects

## Setup

1. **Install dependencies**:
```bash
uv sync
```

2. **Deploy**:
```bash
sst deploy
```

## Usage

This layout is perfect for:
- Large applications with many functions
- Team projects with clear organization
- Microservices architectures
- Complex business logic separation