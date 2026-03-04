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
    // API function with nested layout
    const api = new sst.aws.Function("ApiFunction", {
      handler: "app/functions/api/handler.main",
      runtime: "python3.11",
      timeout: "30 seconds",
      url: true  // Enable function URL
    });

    // Worker function in different nested location
    const worker = new sst.aws.Function("WorkerFunction", {
      handler: "app/functions/worker/handler.main",
      runtime: "python3.11",
      timeout: "5 minutes"
      // No URL needed for worker function
    });

    // Auth function in another nested location
    const auth = new sst.aws.Function("AuthFunction", {
      handler: "app/functions/auth/handler.main",
      runtime: "python3.11",
      timeout: "15 seconds",
      url: true  // Enable function URL for auth endpoints
    });

    // Test function to validate deployment capabilities
    const test = new sst.aws.Function("TestFunction", {
      handler: "app/functions/test/handler.main",
      runtime: "python3.11",
      timeout: "30 seconds",
      url: true  // Enable function URL for testing
    });

    return {
      api: api.url,
      worker: worker.name,
      auth: auth.url,
      test: test.url
    };
  }
});