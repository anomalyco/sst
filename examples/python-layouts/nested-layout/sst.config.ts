/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Python Nested Layout
 *
 * A single-package project where handlers live in nested directories under an
 * `app/` folder. Shared utilities and static data files are accessed via relative
 * paths.
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
 * All handlers share the same `pyproject.toml` at the root. SST finds it by
 * traversing up from the handler path.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("ApiFunction", {
 *   handler: "app/functions/api/handler.main",
 *   runtime: "python3.11",
 *   url: true,
 * });
 * ```
 *
 * Handlers can access static files using paths relative to `__file__`.
 *
 * ```py title="app/functions/api/handler.py"
 * from pathlib import Path
 * config = Path(__file__).parent / "../../data/config.json"
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "nested-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws"
    };
  },
  async run() {
    const api = new sst.aws.Function("ApiFunction", {
      handler: "app/functions/api/handler.main",
      runtime: "python3.11",
      url: true,
    });

    const worker = new sst.aws.Function("WorkerFunction", {
      handler: "app/functions/worker/handler.main",
      runtime: "python3.11",
      timeout: "5 minutes",
    });

    const auth = new sst.aws.Function("AuthFunction", {
      handler: "app/functions/auth/handler.main",
      runtime: "python3.11",
      url: true,
    });

    return {
      api: api.url,
      worker: worker.name,
      auth: auth.url
    };
  }
});