/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "workspace-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws"
    };
  },
  async run() {
    // API function with workspace layout
    const api = new sst.aws.Function("ApiFunction", {
      handler: "src/mypackage/handler.api_handler",
      runtime: "python3.11",
      timeout: "30 seconds",
      url: true  // Enable function URL
    });

    // Worker function sharing the same package
    const worker = new sst.aws.Function("WorkerFunction", {
      handler: "src/mypackage/handler.worker_handler",
      runtime: "python3.11",
      timeout: "5 minutes"
      // No URL needed for worker function
    });

    return {
      api: api.url,
      worker: worker.name
    };
  }
});