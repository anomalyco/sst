/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Python Workspace Layout
 *
 * Use the standard `src/` layout for a single Python package.
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
 * Different functions in the same module can still become separate Lambdas.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("Api", {
 *   handler: "src/mypackage/handler.api_handler",
 *   runtime: "python3.11",
 *   url: true,
 * });
 *
 * new sst.aws.Function("Worker", {
 *   handler: "src/mypackage/handler.worker_handler",
 *   runtime: "python3.11",
 * });
 * ```
 *
 * Both handlers share the same package and imports.
 */
export default $config({
  app(input) {
    return {
      name: "workspace-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const api = new sst.aws.Function("Api", {
      handler: "src/mypackage/handler.api_handler",
      runtime: "python3.11",
      timeout: "30 seconds",
      url: true,
    });

    const worker = new sst.aws.Function("Worker", {
      handler: "src/mypackage/handler.worker_handler",
      runtime: "python3.11",
      timeout: "5 minutes",
    });

    return {
      api: api.url,
      worker: worker.name,
    };
  },
});
