/// <reference path="./.sst/platform/config.d.ts" />

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
