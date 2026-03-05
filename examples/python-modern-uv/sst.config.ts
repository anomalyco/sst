/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Python uv Workspaces
 *
 * Deploys Python Lambda functions from a
 * [uv workspace](https://docs.astral.sh/uv/concepts/projects/workspaces/) with
 * multiple packages and a `src/` layout.
 *
 * SST traverses up from the handler path to find the nearest `pyproject.toml` and
 * uses it to resolve dependencies. Workspace members that depend on each other are
 * bundled automatically.
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
 *     └── worker
 *         ├── pyproject.toml
 *         └── src/worker
 *             ├── __init__.py
 *             └── handler.py
 * ```
 *
 * If your root package uses a `src/` layout, configure hatch so the package is
 * importable by its name rather than `src.myapp`.
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
 * Each workspace member can be deployed as its own function. Members that import
 * other members will have those dependencies bundled in.
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
    // Test 1: Root entry point with src/ layout
    const rootHandler = new sst.aws.Function("RootHandler", {
      handler: "handler.lambda_handler",
      runtime: "python3.11",
      url: true,
    });

    // Test 2: Package with [tool.uv] package = true
    const packageHandler = new sst.aws.Function("PackageHandler", {
      handler: "packages/api/src/api/handler.lambda_handler",
      runtime: "python3.11",
      url: true,
    });

    // Test 3: Workspace member importing a shared workspace package
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
