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

> **Note**: When deploying with SST, the SDK is automatically included — you don't need to install it manually. This package is for local development and testing.

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
- [SDK Reference](https://sst.dev/docs/reference/sdk/)
- [Python on SST](https://sst.dev/docs/examples/#aws-lambda-python)
- [GitHub](https://github.com/sst/sst)
