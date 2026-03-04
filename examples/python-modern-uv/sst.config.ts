/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "python-modern-uv",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Test 1: Root entry point with src/ layout
    const rootHandler = new sst.aws.Function("RootHandler", {
      handler: "handler.lambda_handler",
      runtime: "python3.11",
      url: true,
    });

    // Test 2: Package with [tool.uv] package = true
    const packageHandler = new sst.aws.Function("PackageHandler", {
      handler: "packages/api/src/api/handler.lambda_handler",
      runtime: "python3.11",
      url: true,
    });

    // Test 3: Workspace member importing another member
    const workspaceHandler = new sst.aws.Function("WorkspaceHandler", {
      handler: "packages/worker/src/worker/handler.lambda_handler",
      runtime: "python3.11",
      url: true,
    });

    return {
      rootHandler: rootHandler.url,
      packageHandler: packageHandler.url,
      workspaceHandler: workspaceHandler.url,
    };
  },
});
