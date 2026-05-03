# Publishing sst-sdk to PyPI

The Python SDK publishes automatically as part of the SST release workflow
(`.github/workflows/release.yml`), alongside the JS and Rust SDKs. It triggers
on every version tag push.

## One-time setup

### 1. Configure trusted publishing on PyPI

Go to https://pypi.org/manage/account/publishing/ and create a new pending publisher:

- **PyPI project name**: `sst-sdk`
- **Owner**: `sst` (the GitHub org)
- **Repository**: `sst`
- **Workflow name**: `release.yml`
- **Environment name**: (leave blank — the release job doesn't use a named environment)

This configures OIDC trusted publishing — no API tokens needed.

### 2. Verify permissions

The release workflow already has `id-token: write` permission, which is all
`pypa/gh-action-pypi-publish` needs.

## How it works

When a tag is pushed:

1. The release workflow runs `goreleaser` to build the CLI
2. Publishes the JS SDK to npm
3. Publishes the Rust SDK to crates.io
4. **Publishes the Python SDK to PyPI**:
   - Reads the version from `dist/metadata.json` (same source as other SDKs)
   - Updates `pyproject.toml` with that version
   - Builds with `uv build`
   - Publishes via `pypa/gh-action-pypi-publish`
5. Announces on Discord

The Python SDK version stays in sync with the CLI and other SDKs automatically.

## Testing locally

Build the package without publishing:

```bash
cd sdk/python
uv build
```

This creates `dist/sst_sdk-{version}.tar.gz` and `dist/sst_sdk-{version}-py3-none-any.whl`.

Test the install:

```bash
pip install dist/sst_sdk-*.whl
python -c "from sst import Resource; print('OK')"
```
