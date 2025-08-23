/// <reference path="./.sst/platform/config.d.ts" />

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