# Modern UV Project Structure Tests

This example tests SST's support for modern `uv` project structures including:

1. **`[tool.uv] package = true`** - Enables packaging for editable installs
2. **Root entry point with src/ layout** - Handler in root importing from `src/`
3. **UV workspaces** - Multiple packages with cross-package dependencies

## Project Structure

```
python-modern-uv/
├── pyproject.toml              # Root with [tool.uv] package = true
├── handler.py                  # Entry point in root
├── src/
│   └── myapp/                  # Code in src/ layout
│       ├── __init__.py
│       └── utils.py
└── packages/
    ├── api/
    │   ├── pyproject.toml      # [tool.uv] package = true
    │   └── src/
    │       └── api/
    │           ├── handler.py
    │           └── models.py
    └── worker/
        ├── pyproject.toml      # [tool.uv] package = true
        └── src/
            └── worker/
                └── handler.py  # Imports from api package
```

## Test Scenarios

### Test 1: Root Handler with src/ Layout
**Handler**: `handler.lambda_handler`
**Tests**: 
- ✅ `[tool.uv] package = true` makes root package importable
- ✅ Can import from `src/myapp/` without explicit path manipulation
- ✅ Works in both `sst dev` and deployment

**Expected behavior**:
```python
from src.myapp import utils  # Should work without PYTHONPATH hacks
```

### Test 2: Package Handler
**Handler**: `packages/api/src/api/handler.lambda_handler`
**Tests**:
- ✅ Package with `[tool.uv] package = true` is importable
- ✅ Relative imports within package work
- ✅ External dependencies (requests) are available

**Expected behavior**:
```python
from api import models  # Should work because package = true
```

### Test 3: Workspace Cross-Package Imports
**Handler**: `packages/worker/src/worker/handler.lambda_handler`
**Tests**:
- ✅ Workspace member can import from another workspace member
- ✅ `[tool.uv.sources] api = { workspace = true }` is respected
- ✅ Deployment bundles both packages correctly

**Expected behavior**:
```python
from api import models  # Should work via workspace dependency
```

### Test 4: Worker-Only Dependency Isolation
**Handler**: `packages/worker/src/worker/handler.lambda_handler`
**Tests**:
- ✅ Worker has `arrow` dependency that api and root do NOT have
- ✅ Arrow is only bundled with worker function, not api or root
- ✅ Verifies per-package dependency isolation in workspaces

**Expected behavior**:
- Worker's `pyproject.toml` includes `arrow>=1.3.0`
- API's `pyproject.toml` does NOT include `arrow`
- Root's `pyproject.toml` does NOT include `arrow`
- Worker handler imports and uses `arrow` successfully

## Running Tests

### Development Mode
```bash
cd examples/python-modern-uv
uv sync  # Should make all packages importable
sst dev
```

**Expected**: All three functions should start without import errors

### Alternative: Using package.json Scripts

You can automate `uv sync` by adding it to your package.json:

```json
{
  "scripts": {
    "dev": "uv sync && sst dev",
    "deploy": "uv sync && sst deploy"
  }
}
```

Then just run:
```bash
npm run dev
```

### Deployment Mode
```bash
sst deploy
```

**Expected**: 
- All packages are built correctly
- Dependencies are resolved
- Cross-package imports work in Lambda

## Known Issues to Check

1. **PYTHONPATH in dev mode**: Does SST set PYTHONPATH to include src/?
2. **uv sync before dev**: Does SST call `uv sync` before starting dev mode?
3. **Deployment bundling**: Are workspace packages bundled correctly?
4. **Editable installs**: Does deployment convert editable installs to regular installs?

## Success Criteria

✅ **Dev mode**: All handlers start and can handle requests
✅ **Deployment**: All functions deploy successfully
✅ **Imports**: No ImportError in logs
✅ **Cross-package**: Worker can import from API package
✅ **Performance**: No unnecessary file copying in dev mode
