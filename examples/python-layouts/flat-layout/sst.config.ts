/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Python Flat Layout
 *
 * The smallest Python layout SST supports. The handler lives at the project
 * root alongside `pyproject.toml`.
 *
 * ```txt
 * ├── sst.config.ts
 * ├── pyproject.toml
 * ├── handler.py
 * └── utils.py
 * ```
 *
 * Point the handler directly at a root file.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("ApiFunction", {
 *   handler: "handler.main",
 *   runtime: "python3.11",
 *   url: true,
 * });
 * ```
 *
 * Multiple functions can still share the same root files.
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
    const api = new sst.aws.Function("ApiFunction", {
      handler: "handler.main",
      runtime: "python3.11",
      url: true,
    });

    const worker = new sst.aws.Function("WorkerFunction", {
      handler: "handler.worker",
      runtime: "python3.11",
    });

    return {
      api: api.url,
      worker: worker.name,
    };
  }
});
