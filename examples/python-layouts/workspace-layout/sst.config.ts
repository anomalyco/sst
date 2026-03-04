/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Python Workspace Layout
 *
 * A single-package project using the standard `src/` layout recommended by the
 * [Python packaging guide](https://packaging.python.org/en/latest/discussions/src-layout-vs-flat-layout/).
 * Multiple handlers live inside the same package.
 *
 * ```txt
 * ├── sst.config.ts
 * ├── pyproject.toml
 * └── src
 *     └── mypackage
 *         ├── __init__.py
 *         ├── handler.py
 *         └── utils.py
 * ```
 *
 * Different handler functions in the same file can be deployed as separate Lambda
 * functions.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("ApiFunction", {
 *   handler: "src/mypackage/handler.api_handler",
 *   runtime: "python3.11",
 *   url: true,
 * });
 *
 * new sst.aws.Function("WorkerFunction", {
 *   handler: "src/mypackage/handler.worker_handler",
 *   runtime: "python3.11",
 * });
 * ```
 *
 * Since both functions share the same package, they get the same dependencies and
 * source code bundled in.
 */
export default $config({
  app(input) {
    return {
      name: "workspace-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws"
    };
  },
  async run() {
    // API function with workspace layout
    const api = new sst.aws.Function("ApiFunction", {
      handler: "src/mypackage/handler.api_handler",
      runtime: "python3.11",
      timeout: "30 seconds",
      url: true  // Enable function URL
    });

    // Worker function sharing the same package
    const worker = new sst.aws.Function("WorkerFunction", {
      handler: "src/mypackage/handler.worker_handler",
      runtime: "python3.11",
      timeout: "5 minutes"
      // No URL needed for worker function
    });

    return {
      api: api.url,
      worker: worker.name
    };
  }
});