/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Python Flat Layout
 *
 * The simplest Python project layout for Lambda. All source files live in the
 * project root alongside `pyproject.toml`.
 *
 * ```txt
 * ├── sst.config.ts
 * ├── pyproject.toml
 * ├── handler.py
 * └── utils.py
 * ```
 *
 * The handler points directly to a file and function in the root.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("ApiFunction", {
 *   handler: "handler.main",
 *   runtime: "python3.11",
 *   url: true,
 * });
 * ```
 *
 * Multiple handlers can share the same source files. This is useful when you have
 * an API handler and a background worker that share utility code.
 */
export default $config({
  app(input) {
    return {
      name: "flat-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws"
    };
  },
  async run() {
    // Simple API function
    const api = new sst.aws.Function("ApiFunction", {
      handler: "handler.main",
      runtime: "python3.11",
      timeout: "30 seconds",
      url: true  // Enable function URL
    });

    // Simple worker function
    const worker = new sst.aws.Function("WorkerFunction", {
      handler: "handler.worker",
      runtime: "python3.11",
      timeout: "2 minutes"
      // No URL needed for worker function
    });

    return {
      api: api.url,
      worker: worker.name
    };
  }
});