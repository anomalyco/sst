/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Smoke Test with Function URL
 *
 * Deploys a Lambda function with a before-traffic smoke test using function URLs.
 *
 * The function has two URLs:
 * - `url` — the stable endpoint, pointing to the alias managed by CodeDeploy
 * - `latestUrl` — points to the latest published version for pre-deployment testing
 *
 * A before-traffic hook validates the new version before any traffic shifts to it.
 * If validation fails, the deployment is aborted.
 *
 * ```ts title="sst.config.ts"
 * const fn = new sst.aws.Function("Function", {
 *   handler: "src/api.handler",
 *   rollout: {
 *     type: "all-at-once",
 *     beforeTraffic: "src/before-traffic.handler",
 *     latestUrl: true,
 *   },
 *   url: true,
 * });
 *
 * // Stable endpoint — traffic shifts here after validation
 * fn.url;
 * // Latest version endpoint — for pre-deployment testing
 * fn.latestUrl;
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-smoke-test-function-url",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const fn = new sst.aws.Function("Function", {
      handler: "src/api.handler",
      rollout: {
        type: "all-at-once",
        beforeTraffic: "src/before-traffic.handler",
        latestUrl: true,
      },
      url: true,
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
