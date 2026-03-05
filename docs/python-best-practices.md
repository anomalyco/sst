# Python Lambda Best Practices

Best practices for Python Lambda functions with SST. The runtime works with any project structure while providing intelligent caching and performance optimizations.

## Imports

Use absolute imports. They're reliable across `sst dev` and `sst deploy`:

```python
# Good
from shared.utils import helper_function
from mypackage.submodule import MyClass

# Bad — relative imports can fail in Lambda
from .utils import helper_function
from ..shared import common_function
```

## Static File Access

Use paths relative to your handler file. Don't assume the working directory:

```python
from pathlib import Path

def load_config():
    config_path = Path(__file__).parent / "../../data/config.json"
    with open(config_path, 'r') as f:
        return json.load(f)
```

Cache file contents across invocations since Lambda containers are reused:

```python
from functools import lru_cache

@lru_cache(maxsize=1)
def get_config():
    config_path = Path(__file__).parent / "../data/config.json"
    with open(config_path, 'r') as f:
        return json.load(f)
```

## Package Structure

Maintain proper Python packages with `__init__.py` files:

```
project/
├── shared/
│   ├── __init__.py
│   └── utils.py
├── app/
│   ├── __init__.py
│   └── functions/
│       ├── __init__.py
│       └── api/
│           ├── __init__.py
│           └── handler.py
├── pyproject.toml
└── sst.config.ts
```

## File Inclusion and Exclusion

### Custom Configuration (pyproject.toml)

Control what goes into your deployment package with `[tool.sst]`:

```toml
[tool.sst]
include = [
    "config/*.json",
    "templates/*.html",
    "data/**/*.csv"
]

exclude = [
    "tests/**",
    "docs/**",
    "*.log",
    "large-datasets/**"
]
```

Include patterns take precedence over exclude patterns.

### Default Behavior

When no `[tool.sst]` is configured, SST applies sensible defaults:

**Included:** All `.py` files, directories containing Python files, `__init__.py`, static file directories (`data/`, `config/`, `templates/`, `static/`, `assets/`), `requirements.txt`.

**Excluded:** `__pycache__/`, `*.pyc`, `.git/`, `.venv/`, `venv/`, `.vscode/`, `.idea/`, `tests/`, `test/`, `.pytest_cache/`, `README.md`, `pyproject.toml`, `setup.py`, `node_modules/`, `.sst/`, `*.log`, `*.tmp`.

Note: Individual files matching `test_*.py` or `*_test.py` are NOT excluded by default — only `tests/` and `test/` directories. Add them to your exclude list if needed.

### Pattern Matching

Supports glob patterns: `*` (single segment), `**` (recursive), `?` (single char), `[abc]` (character class).

## Troubleshooting

### ModuleNotFoundError

1. Ensure `__init__.py` files exist in all package directories
2. Use absolute imports instead of relative imports
3. Verify the module is in the correct directory structure

### FileNotFoundError

1. Use paths relative to `__file__`, not the working directory
2. Check the file isn't in an excluded directory
3. Add it to `[tool.sst]` include if needed

### Import works locally but fails in Lambda

1. Switch from relative to absolute imports
2. Verify all `__init__.py` files are present
3. Enable debug logging: `SST_DEBUG=python:* sst deploy`

### Lambda Package Too Large

1. Use container mode for large dependencies (ML, etc.)
2. Add large data files to `[tool.sst]` exclude
3. Remove dev dependencies: `uv sync --no-dev`

### Cache Issues

```bash
# Clear SST cache
rm -rf .sst/cache

# Disable cache temporarily
SST_PYTHON_DISABLE_CACHE=true sst deploy
```
