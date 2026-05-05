/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Python uv Workspaces
 *
 * Deploy Python Lambda functions from a
 * [uv workspace](https://docs.astral.sh/uv/concepts/projects/workspaces/) that
 * uses `package = true` and a `src/` layout.
 *
 * SST traverses up from the handler path to find the nearest `pyproject.toml`.
 *
 * ```txt
 * ├── sst.config.ts
 * ├── pyproject.toml
 * ├── handler.py
 * ├── src
 * │   └── myapp
 * │       ├── __init__.py
 * │       └── utils.py
 * └── packages
 *     ├── api
 *     │   ├── pyproject.toml
 *     │   └── src/api
 *     │       ├── __init__.py
 *     │       └── handler.py
 *     ├── shared
 *     │   ├── pyproject.toml
 *     │   └── src/shared
 *     │       ├── __init__.py
 *     │       └── models.py
 *     └── worker
 *         ├── pyproject.toml
 *         └── src/worker
 *             ├── __init__.py
 *             └── handler.py
 * ```
 *
 * With `package = true`, the package is importable by name instead of `src.*`.
 *
 * ```toml title="pyproject.toml"
 * [tool.hatch.build.targets.wheel]
 * packages = ["src/myapp"]
 * ```
 *
 * Then import it normally in your handler.
 *
 * ```py title="handler.py"
 * from myapp import utils
 * ```
 *
 * Each workspace member can become its own function.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("PackageHandler", {
 *   handler: "packages/api/src/api/handler.lambda_handler",
 *   runtime: "python3.11",
 *   url: true,
 * });
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "python-modern-uv",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const rootHandler = new sst.aws.Function("RootHandler", {
      handler: "handler.lambda_handler",
      runtime: "python3.11",
      url: true,
    });

    const packageHandler = new sst.aws.Function("PackageHandler", {
      handler: "packages/api/src/api/handler.lambda_handler",
      runtime: "python3.11",
      url: true,
    });

    const workspaceHandler = new sst.aws.Function("WorkspaceHandler", {
      handler: "packages/worker/src/worker/handler.lambda_handler",
      runtime: "python3.11",
      url: true,
    });

    return {
      rootHandler: rootHandler.url,
      packageHandler: packageHandler.url,
      workspaceHandler: workspaceHandler.url,
    };
  },
});
