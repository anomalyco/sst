/// <reference path="./.sst/platform/config.d.ts" />

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