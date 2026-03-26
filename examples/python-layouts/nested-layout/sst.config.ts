/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Python Nested Layout
 *
 * Keep handlers in nested folders while sharing one `pyproject.toml` at the
 * project root.
 *
 * ```txt
 * ├── sst.config.ts
 * ├── pyproject.toml
 * ├── shared
 * │   └── utils.py
 * └── app
 *     ├── data
 *     │   └── config.json
 *     └── functions
 *         ├── api
 *         │   └── handler.py
 *         ├── auth
 *         │   └── handler.py
 *         └── worker
 *             └── handler.py
 * ```
 *
 * SST walks up from the handler path to find the nearest `pyproject.toml`.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("Api", {
 *   handler: "app/functions/api/handler.main",
 *   runtime: "python3.11",
 *   url: true,
 * });
 * ```
 *
 * Nested handlers can still read static files relative to `__file__`.
 *
 * ```py title="app/functions/api/handler.py"
 * from pathlib import Path
 * config = Path(__file__).parents[2] / "data" / "config.json"
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "nested-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const api = new sst.aws.Function("Api", {
      handler: "app/functions/api/handler.main",
      runtime: "python3.11",
      url: true,
    });

    const worker = new sst.aws.Function("Worker", {
      handler: "app/functions/worker/handler.main",
      runtime: "python3.11",
      timeout: "5 minutes",
    });

    const auth = new sst.aws.Function("Auth", {
      handler: "app/functions/auth/handler.main",
      runtime: "python3.11",
      url: true,
    });

    return {
      api: api.url,
      worker: worker.name,
      auth: auth.url,
    };
  },
});
