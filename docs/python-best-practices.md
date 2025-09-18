# Python Lambda Best Practices

This document outlines best practices for developing Python Lambda functions with SST. The simplified runtime system works with any project structure while providing intelligent caching and performance optimizations.

## Module Imports

### ✅ Recommended: Absolute Imports

Always use absolute imports for the most reliable behavior in Lambda environments:

```python
# ✅ Recommended - absolute imports work reliably
from shared.utils import helper_function
from app.functions.auth.handler import auth_main
from mypackage.submodule import MyClass
from core.database import get_connection
```

**Why absolute imports work better:**
- Consistent behavior between local development and Lambda
- No ambiguity about module resolution
- Works regardless of how Python is invoked
- Compatible with all Python packaging tools

### ❌ Avoid: Relative Imports

Relative imports can be unreliable in Lambda environments:

```python
# ❌ Not recommended - can fail in Lambda
from .utils import helper_function
from ..shared import common_function
from ...core import database
```

**Why relative imports can fail:**
- Lambda execution context may not preserve package structure expectations
- Different behavior between `sst dev` and `sst deploy`
- Fragile when code is moved or restructured

## Static File Access

When your Lambda function needs to read static files (config files, templates, data files), use paths relative to your handler file.

### ✅ Recommended Approaches

**Method 1: Using pathlib (Modern Python)**
```python
import json
from pathlib import Path

def load_config():
    config_path = Path(__file__).parent / "../../data/config.json"
    with open(config_path, 'r') as f:
        return json.load(f)

def load_template():
    template_path = Path(__file__).parent / "../templates/email.html"
    with open(template_path, 'r') as f:
        return f.read()
```

**Method 2: Using os.path (Traditional)**
```python
import os
import json

def load_config():
    config_path = os.path.join(os.path.dirname(__file__), "../../data/config.json")
    with open(config_path, 'r') as f:
        return json.load(f)

def load_template():
    template_path = os.path.join(os.path.dirname(__file__), "../templates/email.html")
    with open(template_path, 'r') as f:
        return f.read()
```

**Method 3: Lambda-specific absolute paths (Advanced)**
```python
import os
import json

def load_config():
    # In Lambda, code is always at /var/task
    config_path = "/var/task/app/data/config.json"
    with open(config_path, 'r') as f:
        return json.load(f)
```

### ❌ Avoid: Working Directory Assumptions

Don't rely on the current working directory, as it may not be what you expect:

```python
# ❌ Not recommended - assumes working directory
def bad_load_config():
    with open("data/config.json", 'r') as f:  # May fail
        return json.load(f)

# ❌ Not recommended - hardcoded local paths
def bad_load_template():
    with open("/home/user/project/templates/email.html", 'r') as f:  # Will fail
        return f.read()
```

## Project Structure Best Practices

### Package Structure

SST works with any project structure, but maintaining proper Python package structure with `__init__.py` files ensures reliable imports:

```
project/
├── shared/
│   ├── __init__.py          # Required for imports
│   ├── utils.py
│   └── database.py
├── app/
│   ├── __init__.py          # Required for imports
│   ├── data/
│   │   ├── config.json      # Static files preserved
│   │   └── templates/
│   │       └── email.html
│   └── functions/
│       ├── __init__.py      # Required for imports
│       ├── api/
│       │   ├── __init__.py  # Required for imports
│       │   └── handler.py
│       └── worker/
│           ├── __init__.py  # Required for imports
│           └── handler.py
├── pyproject.toml
└── sst.config.ts
```

### File Inclusion and Exclusion

SST uses a smart two-layer system to determine which files are included in your Lambda deployment package:

### Layer 1: Custom Configuration (pyproject.toml)

You can explicitly control file inclusion/exclusion by adding a `[tool.sst]` section to your `pyproject.toml`:

```toml
[tool.sst]
# Files to explicitly include (overrides standard exclusions)
include = [
    "config/*.json",           # Include all JSON config files
    "templates/*.html",        # Include HTML templates
    "static/critical.css",     # Include specific critical CSS
    ".env.production",         # Include production environment file
    "data/**/*.csv"            # Include all CSV files in data subdirectories
]

# Files to explicitly exclude (in addition to standard exclusions)
exclude = [
    "tests/**",               # Exclude all test files
    "docs/**",                # Exclude documentation
    "*.log",                  # Exclude log files
    "temp/**",                # Exclude temporary files
    "scripts/dev-*",          # Exclude development scripts
    "*.backup",               # Exclude backup files
    "large-datasets/**"       # Exclude large data files
]
```

**Important**: Include patterns take precedence over exclude patterns. If a file matches both, it will be included.

### Layer 2: Standard Python Conventions (Automatic)

When no custom configuration is found, SST applies sensible defaults:

**✅ Automatically Included:**
- All Python files (`.py`)
- All directories containing Python files
- Package structure files (`__init__.py`)
- Static files and assets in common directories (`data/`, `config/`, `templates/`, `static/`, `assets/`)
- Essential configuration files (`requirements.txt`)

**❌ Automatically Excluded:**
- **Build artifacts**: `__pycache__/`, `*.pyc`, `*.pyo`, `*.pyd`, `*.egg-info/`
- **Version control**: `.git/`, `.gitignore`, `.gitattributes`
- **Virtual environments**: `.venv/`, `venv/`, `env/`
- **IDE files**: `.vscode/`, `.idea/`, `*.swp`, `.DS_Store`
- **Test artifacts**: `.pytest_cache/`, `.coverage`, `htmlcov/`
- **Documentation**: `README.md`, `CHANGELOG.md`, `LICENSE`
- **Development config**: `pyproject.toml`, `setup.py`, `tox.ini`, `Makefile`
- **Development dependencies**: `requirements-dev.txt`, `.pre-commit-config.yaml`
- **Node.js artifacts**: `node_modules/`, `package-lock.json` (for mixed projects)
- **Temporary files**: `*.log`, `*.tmp`, `tmp/`, `temp/`
- **SST cache**: `.sst/`

### Pattern Matching

The system supports glob-style patterns:

- `*` matches any characters within a single path segment
- `**` matches any characters across multiple path segments (recursive)
- `?` matches any single character
- `[abc]` matches any character in the brackets
- Directory patterns ending with `/` match directories and their contents

**Pattern Examples:**

```toml
[tool.sst]
include = [
    "*.py",                   # All Python files in root
    "config/**/*.json",       # All JSON files in config subdirectories
    "static/css/main.css",    # Specific file
    "templates/",             # Entire templates directory
    "data/[0-9]*.csv",       # CSV files starting with digits
    "assets/**/*.{png,jpg}"   # Image files in assets (requires shell expansion)
]

exclude = [
    "test_*.py",              # Test files starting with test_
    "**/*_test.py",           # Test files ending with _test.py anywhere
    "temp/**",                # Everything in temp directory
    "*.log",                  # All log files
    "cache/[0-9][0-9][0-9][0-9]/**"  # Cache directories with 4-digit names
]
```

### Common Use Cases

**Including Configuration Files:**
```toml
[tool.sst]
include = [
    "config/production.json",
    "config/staging.json",
    ".env.production",
    "data/lookup-tables/*.csv"
]
```

**Excluding Large Development Files:**
```toml
[tool.sst]
exclude = [
    "datasets/**",            # Large training datasets
    "notebooks/**",           # Jupyter notebooks
    "experiments/**",         # ML experiment files
    "*.pkl",                  # Pickle files (often large)
    "*.model"                 # Model files
]
```

**Mixed Project (Python + Frontend):**
```toml
[tool.sst]
include = [
    "static/dist/**",         # Built frontend assets
    "templates/**"            # Server-side templates
]
exclude = [
    "frontend/src/**",        # Frontend source (not needed at runtime)
    "frontend/node_modules/**", # Frontend dependencies
    "*.scss",                 # Sass source files
    "webpack.config.js"       # Build configuration
]
```

### Troubleshooting File Inclusion

**Check what files are being included:**
```bash
# Enable debug logging to see file filtering
export SST_DEBUG=python:filter
sst deploy
```

**Common issues:**

1. **Required file is excluded:**
   ```toml
   [tool.sst]
   include = ["path/to/required/file.json"]
   ```

2. **Package too large:**
   ```toml
   [tool.sst]
   exclude = [
       "large-data/**",
       "*.pkl",
       "datasets/**"
   ]
   ```

3. **Missing static assets:**
   ```toml
   [tool.sst]
   include = [
       "static/**",
       "templates/**",
       "assets/**"
   ]
   ```

### Migration from Previous Versions

If you're upgrading from an older version of SST, your projects will continue to work without changes. The new system:

- ✅ Maintains backward compatibility
- ✅ Uses the same sensible defaults
- ✅ Provides better control when needed
- ✅ Eliminates brittle layout detection logic

You only need to add `[tool.sst]` configuration if you want to customize the default behavior.

## Real-World Examples

### Example 1: API Handler with Config

```python
# File: app/functions/api/handler.py
import json
import os
from pathlib import Path
from shared.utils import format_response, validate_request

def main(event, context):
    try:
        # Load configuration
        config_path = Path(__file__).parent / "../../data/config.json"
        with open(config_path, 'r') as f:
            config = json.load(f)
        
        # Use shared utilities
        if not validate_request(event):
            return format_response(400, {"error": "Invalid request"})
        
        # Process request
        return format_response(200, {
            "message": "Success",
            "app_name": config["app_name"],
            "version": config["version"]
        })
        
    except Exception as e:
        return format_response(500, {"error": str(e)})
```

### Example 2: Worker with Template

```python
# File: app/functions/worker/handler.py
import os
from pathlib import Path
from shared.email import send_email
from shared.database import get_user

def main(event, context):
    try:
        user_id = event["user_id"]
        user = get_user(user_id)
        
        # Load email template
        template_path = Path(__file__).parent / "../../templates/welcome.html"
        with open(template_path, 'r') as f:
            template = f.read()
        
        # Customize template
        email_content = template.replace("{{name}}", user["name"])
        
        # Send email
        send_email(user["email"], "Welcome!", email_content)
        
        return {"status": "success", "user_id": user_id}
        
    except Exception as e:
        return {"status": "error", "message": str(e)}
```

### Example 3: Multi-Package Import

```python
# File: functions/api/handler.py
from core.database import Database
from core.auth import authenticate
from shared.utils import format_response
from shared.validation import validate_email

def main(event, context):
    try:
        # Authenticate request
        user = authenticate(event.get("headers", {}))
        if not user:
            return format_response(401, {"error": "Unauthorized"})
        
        # Validate input
        email = event.get("body", {}).get("email")
        if not validate_email(email):
            return format_response(400, {"error": "Invalid email"})
        
        # Database operation
        db = Database()
        result = db.update_user_email(user["id"], email)
        
        return format_response(200, {"success": True, "result": result})
        
    except Exception as e:
        return format_response(500, {"error": str(e)})
```

## Testing Your Code

### Local Testing

Test your imports and file access locally:

```python
#!/usr/bin/env python3
"""Test script to validate imports and file access."""

def test_imports():
    """Test that all imports work correctly."""
    try:
        from shared.utils import format_response
        from app.functions.api.handler import main
        print("✅ All imports successful")
        return True
    except ImportError as e:
        print(f"❌ Import failed: {e}")
        return False

def test_file_access():
    """Test that static files can be accessed."""
    from pathlib import Path
    
    try:
        config_path = Path(__file__).parent / "app/data/config.json"
        with open(config_path, 'r') as f:
            config = json.load(f)
        print(f"✅ Config loaded: {config.get('app_name', 'unknown')}")
        return True
    except Exception as e:
        print(f"❌ File access failed: {e}")
        return False

if __name__ == "__main__":
    import json
    success = test_imports() and test_file_access()
    exit(0 if success else 1)
```

### Lambda Testing

Create a test function to validate deployment:

```python
# File: functions/test/handler.py
import json
import os
import sys
from pathlib import Path

def main(event, context):
    """Test function to validate deployment capabilities."""
    results = {
        "imports": test_imports(),
        "files": test_file_access(),
        "environment": get_environment_info()
    }
    
    return {
        "statusCode": 200,
        "body": json.dumps(results, indent=2)
    }

def test_imports():
    """Test various import scenarios."""
    try:
        from shared.utils import format_response
        from app.functions.api.handler import main
        return {"status": "success", "message": "All imports work"}
    except Exception as e:
        return {"status": "error", "message": str(e)}

def test_file_access():
    """Test static file access."""
    try:
        config_path = Path(__file__).parent / "../../data/config.json"
        with open(config_path, 'r') as f:
            config = json.load(f)
        return {"status": "success", "config": config}
    except Exception as e:
        return {"status": "error", "message": str(e)}

def get_environment_info():
    """Get Lambda environment information."""
    return {
        "python_version": sys.version,
        "working_directory": os.getcwd(),
        "handler_location": __file__,
        "is_lambda": os.path.exists("/var/runtime")
    }
```

## Common Issues and Solutions

### Issue: ModuleNotFoundError

**Problem**: `ModuleNotFoundError: No module named 'mymodule'`

**Solutions**:
1. Ensure `__init__.py` files exist in all package directories
2. Use absolute imports instead of relative imports
3. Verify the module is in the correct directory structure

### Issue: FileNotFoundError

**Problem**: `FileNotFoundError: [Errno 2] No such file or directory: 'config.json'`

**Solutions**:
1. Use paths relative to `__file__` instead of working directory
2. Verify the file exists in your project structure
3. Check that the file is not in an excluded directory

### Issue: Import works locally but fails in Lambda

**Problem**: Code works with `sst dev` but fails with `sst deploy`

**Solutions**:
1. Switch from relative to absolute imports
2. Test with the Lambda test function shown above
3. Check that all required `__init__.py` files are present

## Performance Considerations

### Import Performance

- Absolute imports are typically faster than relative imports
- Import modules at the module level, not inside functions (when possible)
- Use `from module import function` for frequently used functions

### File Access Performance

- Cache file contents when possible (Lambda containers are reused)
- Use appropriate file reading methods for your data size
- Consider using environment variables for small configuration values

### Example: Efficient Configuration Loading

```python
import json
import os
from pathlib import Path
from functools import lru_cache

@lru_cache(maxsize=1)
def get_config():
    """Load configuration once and cache it."""
    config_path = Path(__file__).parent / "../data/config.json"
    with open(config_path, 'r') as f:
        return json.load(f)

def main(event, context):
    config = get_config()  # Cached after first call
    # ... rest of handler
```

## Migration from Other Patterns

### From Relative Imports

```python
# Before (relative imports)
from .utils import helper
from ..shared import common
from ...core import database

# After (absolute imports)
from mypackage.utils import helper
from shared import common
from core import database
```

### From Working Directory Paths

```python
# Before (working directory)
with open("config.json", 'r') as f:
    config = json.load(f)

# After (relative to handler)
from pathlib import Path
config_path = Path(__file__).parent / "../config.json"
with open(config_path, 'r') as f:
    config = json.load(f)
```

## Summary

Following these best practices ensures your Python Lambda functions work reliably in both development (`sst dev`) and production (`sst deploy`) environments:

1. **Use absolute imports** for all module dependencies
2. **Use file paths relative to `__file__`** for static file access
3. **Maintain proper package structure** with `__init__.py` files
4. **Test both locally and in Lambda** to verify behavior
5. **Cache file contents** when appropriate for performance

The SST deployment system preserves your directory structure and handles dependencies automatically, so following these patterns ensures consistent, reliable behavior across all environments.