# SST Python SDK

The Python SDK for [SST](https://sst.dev) lets you access linked resources in your Python Lambda functions.

## Installation

```bash
pip install sst-sdk
```

Or with uv:

```bash
uv add sst-sdk
```

## Migrating from the Git dependency

If you were previously installing the SDK from GitHub:

```toml
# Before
[project]
dependencies = ["sst"]

[tool.uv.sources]
sst = { git = "https://github.com/anomalyco/sst", subdirectory = "sdk/python" }
```

Update your `pyproject.toml` to use the PyPI package instead:

```toml
# After
[project]
dependencies = ["sst-sdk"]
```

That's it — remove the `[tool.uv.sources]` entry for `sst` and replace the dependency name. No code changes needed; `from sst import Resource` works the same way.

## Usage

Use `Resource` to access any resource linked to your function in `sst.config.ts`:

```python
from sst import Resource

# Access linked resources by name
bucket_name = Resource.MyBucket.name
table_name = Resource.MyTable.name
```

Resources are defined and linked in your `sst.config.ts`:

```ts
const bucket = new sst.aws.Bucket("MyBucket");

new sst.aws.Function("MyFunction", {
  handler: "handler.main",
  link: [bucket],
});
```

The SDK reads resource bindings from encrypted environment variables set by SST at deploy time. In `sst dev`, resources are available automatically through the local development bridge.

## Supported Python Versions

- Python 3.9+

## Links

- [SST Documentation](https://sst.dev/docs/)
- [SDK Reference](https://sst.dev/docs/reference/sdk/#python)
- [Python Examples](https://github.com/anomalyco/sst/tree/dev/examples/aws-python)
- [GitHub](https://github.com/anomalyco/sst)
