/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Python Monorepo Layout
 *
 * A uv workspace with multiple services that share a common library. Each service
 * has its own `pyproject.toml` and handler, while shared code lives in a separate
 * workspace package.
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
 *     ├── auth
 *     │   ├── pyproject.toml
 *     │   └── handler.py
 *     └── worker
 *         ├── pyproject.toml
 *         └── handler.py
 * ```
 *
 * The shared package uses a proper nested structure so it installs under the
 * `shared` namespace.
 *
 * ```toml title="shared/pyproject.toml"
 * [tool.hatch.build.targets.wheel]
 * packages = ["shared"]
 * ```
 *
 * Each service declares the shared package as a workspace dependency.
 *
 * ```toml title="services/api/pyproject.toml"
 * [tool.uv.sources]
 * shared = { workspace = true }
 * ```
 *
 * SST bundles the shared code into each function automatically.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("ApiService", {
 *   handler: "services/api/handler.main",
 *   runtime: "python3.12",
 *   url: true,
 * });
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
    // Auth service - has its own pyproject.toml in services/auth/
    const auth = new sst.aws.Function("AuthService", {
      handler: "services/auth/handler.main",
      runtime: "python3.12",
      timeout: "15 seconds",
      url: true, // Enable function URL for auth endpoints
    });

    // API service - uses root pyproject.toml
    const api = new sst.aws.Function("ApiService", {
      handler: "services/api/handler.main",
      runtime: "python3.12",
      timeout: "30 seconds",
      url: true, // Enable function URL
    });

    // Worker service - has its own pyproject.toml in services/worker/
    const worker = new sst.aws.Function("WorkerService", {
      handler: "services/worker/handler.main",
      runtime: "python3.12",
      timeout: "5 minutes",
      // No URL needed for worker function
    });

    return {
      auth: auth.url,
      api: api.url,
      worker: worker.name,
    };
  },
});
