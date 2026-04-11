/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Smoke Test with Router
 *
 * Deploys a Lambda function with a before-traffic smoke test using an SST Router.
 *
 * A before-traffic hook validates the new version before any traffic shifts to it.
 * If validation fails, the deployment is aborted.
 *
 * #### Create the router and function with rollout
 *
 * The function's `url` is the stable endpoint at `/api`. The `latestUrl` exposes the
 * latest published version at `/api/test` for pre-deployment testing.
 *
 * ```ts title="sst.config.ts"
 * const router = new sst.aws.Router("Router");
 *
 * const fn = new sst.aws.Function("Function", {
 *   handler: "src/api.handler",
 *   rollout: {
 *     type: "all-at-once",
 *     latestUrl: { router: { instance: router, path: "/api/test" } },
 *     beforeTraffic: "src/before-traffic.handler",
 *   },
 *   url: {
 *     router: { instance: router, path: "/api" },
 *   },
 * });
 * ```
 *
 * #### Access the URLs
 *
 * ```ts title="sst.config.ts"
 * // Stable endpoint — traffic shifts here after validation
 * fn.url;
 * // Latest version endpoint — for pre-deployment testing
 * fn.latestUrl;
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-smoke-test-router",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const router = new sst.aws.Router("Router");

    const fn = new sst.aws.Function("Function", {
      handler: "src/api.handler",
      rollout: {
        type: "all-at-once",
        latestUrl: { router: { instance: router, path: "/api/test" } },
        beforeTraffic: "src/before-traffic.handler",
      },
      url: {
        router: { instance: router, path: "/api" },
      },
      // Rollout only runs when function code changes. Set to false to deploy
      // actual code since sst dev deploys a stub that never changes.
      dev: false,
    });

    return {
      url: fn.url,
      latestUrl: fn.latestUrl,
    };
  },
});
