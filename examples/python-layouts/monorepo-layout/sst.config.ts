/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Python Monorepo Layout
 *
 * Put each service in its own workspace package and share code from a separate
 * package.
 *
 * ```txt
 * ├── sst.config.ts
 * ├── pyproject.toml
 * ├── shared
 * │   ├── pyproject.toml
 * │   └── shared
 * │       ├── __init__.py
 * │       └── utils.py
 * └── services
 *     ├── api
 *     │   ├── pyproject.toml
 *     │   └── handler.py
 *     └── worker
 *         ├── pyproject.toml
 *         └── handler.py
 * ```
 *
 * Each service has its own `pyproject.toml` and points at the shared workspace
 * package with `workspace = true`.
 *
 * ```toml title="services/api/pyproject.toml"
 * [tool.uv.sources]
 * shared = { workspace = true }
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "monorepo-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const api = new sst.aws.Function("ApiService", {
      handler: "services/api/handler.main",
      runtime: "python3.12",
      url: true,
    });

    const worker = new sst.aws.Function("WorkerService", {
      handler: "services/worker/handler.main",
      runtime: "python3.12",
      timeout: "5 minutes",
    });

    return {
      api: api.url,
      worker: worker.name,
    };
  },
});
