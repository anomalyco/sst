/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Smoke Test with HTTP API
 *
 * Deploys a Lambda function with a before-traffic smoke test using API Gateway HTTP API.
 *
 * A before-traffic hook validates the new version before any traffic shifts to it.
 * If validation fails, the deployment is aborted.
 *
 * #### Create the function with rollout
 *
 * ```ts title="sst.config.ts"
 * const fn = new sst.aws.Function("Function", {
 *   handler: "src/api.handler",
 *   rollout: {
 *     type: "all-at-once",
 *     beforeTraffic: {
 *       handler: "src/before-traffic.handler",
 *       link: [testApi],
 *     },
 *   },
 * });
 * ```
 *
 * #### Route traffic through API Gateway
 *
 * Two API Gateway HTTP API endpoints expose the function:
 * - `api` — routes to the stable alias managed by CodeDeploy
 * - `testApi` — routes to the latest published version for pre-deployment testing
 *
 * ```ts title="sst.config.ts"
 * const api = new sst.aws.ApiGatewayV2("Api");
 * api.route("$default", fn);
 *
 * const testApi = new sst.aws.ApiGatewayV2("TestApi");
 * testApi.route("$default", fn.latestTargetArn);
 * ```
 *
 * #### Full config
 *
 * ```ts title="sst.config.ts"
 * const api = new sst.aws.ApiGatewayV2("Api");
 * const testApi = new sst.aws.ApiGatewayV2("TestApi");
 *
 * const fn = new sst.aws.Function("Function", {
 *   handler: "src/api.handler",
 *   rollout: {
 *     type: "all-at-once",
 *     beforeTraffic: {
 *       handler: "src/before-traffic.handler",
 *       link: [testApi],
 *     },
 *   },
 * });
 *
 * api.route("$default", fn);
 * testApi.route("$default", fn.latestTargetArn);
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-smoke-test-http-api",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const api = new sst.aws.ApiGatewayV2("Api");
    const testApi = new sst.aws.ApiGatewayV2("TestApi");

    const fn = new sst.aws.Function("Function", {
      handler: "src/api.handler",
      rollout: {
        type: "all-at-once",
        beforeTraffic: {
          handler: "src/before-traffic.handler",
          link: [testApi],
        },
      },
      // Rollout only runs when function code changes. Set to false to deploy
      // actual code since sst dev deploys a stub that never changes.
      dev: false,
    });

    api.route("$default", fn);
    testApi.route("$default", fn.latestTargetArn);
  },
});
