/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Router protection with OAC
 *
 * Creates a router with Origin Access Control (OAC) to secure Lambda function URLs
 * behind CloudFront. Direct access to the Lambda URL returns 403.
 */
export default $config({
  app(input) {
    return {
      name: "aws-router-protection",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const router = new sst.aws.Router("MyRouter", {
      protection: "oac",
    });

    const api = new sst.aws.Function("MyApi", {
      handler: "api.handler",
      url: {
        router: { instance: router, path: "/api" },
      },
    });

    return {
      router: router.url,
      api: api.url,
    };
  },
});
