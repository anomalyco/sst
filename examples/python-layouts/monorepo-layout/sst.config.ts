/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "monorepo-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws"
    };
  },
  async run() {
    // Auth service
    const auth = new sst.aws.Function("AuthService", {
      handler: "services/auth/handler.main",
      runtime: "python3.11",
      timeout: "15 seconds",
      url: true  // Enable function URL for auth endpoints
    });

    // API service
    const api = new sst.aws.Function("ApiService", {
      handler: "services/api/handler.main",
      runtime: "python3.11",
      timeout: "30 seconds",
      url: true  // Enable function URL
    });

    // Worker service
    const worker = new sst.aws.Function("WorkerService", {
      handler: "services/worker/handler.main",
      runtime: "python3.11",
      timeout: "5 minutes"
      // No URL needed for worker function
    });

    return {
      auth: auth.url,
      api: api.url,
      worker: worker.name
    };
  }
});