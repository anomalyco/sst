# Python Migration Guide

SST's Python runtime works with any project structure. All existing projects continue to work without changes. This guide covers optional modernization steps.

## Do I Need to Migrate?

No. Existing projects work as-is and automatically get:

- Faster builds through intelligent caching
- Better error messages
- Improved reliability

## Optional: Modernize Dependencies

If you're using `requirements.txt`, you can switch to `pyproject.toml` + UV for faster dependency management.

1. Install UV and initialize:
   ```bash
   curl -LsSf https://astral.sh/uv/install.sh | sh
   uv init --no-readme
   ```

2. Add dependencies:
   ```bash
   uv add requests boto3
   ```

3. Test and remove old files:
   ```bash
   sst dev
   rm requirements.txt
   ```

### From Poetry to UV

1. Keep the `[project]` section, remove `[tool.poetry]` sections
2. Run `uv lock`
3. Remove `poetry.lock`
4. Replace commands: `poetry install` → `uv sync`, `poetry add` → `uv add`

## Optional: Reorganize Project Structure

Handlers can live anywhere. Some common layouts:

**Simple:**
```
my-function/
├── handler.py
├── pyproject.toml
└── utils.py
```

**Modern package:**
```
my-project/
├── pyproject.toml
├── src/
│   └── mypackage/
│       ├── __init__.py
│       └── handler.py
└── tests/
```

**By function:**
```
my-app/
├── pyproject.toml
├── functions/
│   ├── api/handler.py
│   └── worker/handler.py
└── shared/
    └── utils.py
```

Update your SST config to match:

```typescript
new Function("ApiFunction", {
  handler: "functions/api/handler.main",
  runtime: "python3.11"
})
```

## After Migration

If something breaks:

1. Verify handler path matches your file structure
2. Test imports locally: `python -c "from mypackage.handler import main"`
3. Enable debug logging: `SST_DEBUG=python:* sst deploy`
4. Clear cache: `rm -rf .sst/cache`

See [python-best-practices.md](./python-best-practices.md) for import and file access patterns, and [python-uv-workspace-resolution.md](./python-uv-workspace-resolution.md) for monorepo/workspace setups.
