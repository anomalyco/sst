/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Smoke Test
 *
 * Deploys a Lambda function with a before-traffic smoke test that invokes the
 * new version directly using the AWS Lambda SDK.
 *
 * A before-traffic hook invokes the new version via `Resource.Function.name`
 * and validates the response before any traffic shifts to it. If validation
 * fails, the deployment is aborted.
 *
 * #### Create the function with rollout
 *
 * The before-traffic hook is linked to the function so it can access the
 * function name at runtime via `Resource.Function.name`.
 *
 * ```ts title="sst.config.ts"
 * const fn = new sst.aws.Function("Function", {
 *   handler: "src/api.handler",
 *   rollout: {
 *     type: "all-at-once",
 *     beforeTraffic: "src/before-traffic.handler",
 *   },
 * });
 * ```
 *
 * #### Invoke the new version in the before-traffic hook
 *
 * ```ts title="src/before-traffic.ts"
 * const resp = await lambda.send(
 *   new InvokeCommand({
 *     FunctionName: Resource.Function.name,
 *     Qualifier: Resource.Function.latestQualifier,
 *     Payload: JSON.stringify({ health: true }),
 *   }),
 * );
 * ```
 *
 * #### Report the result
 *
 * ```ts title="src/before-traffic.ts"
 * import { rollout } from "sst/aws/rollout";
 *
 * export const handler = rollout.handler(async (event) => {
 *   const result = // ... invoke and validate the new version
 *   await rollout.report(event, result ? "Succeeded" : "Failed");
 * });
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-smoke-test",
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
      },
      // Rollout only runs when function code changes. Set to false to deploy
      // actual code since sst dev deploys a stub that never changes.
      dev: false,
    });

    new sst.x.DevCommand("InvokeStable", {
      dev: {
        autostart: false,
        command: $interpolate`aws lambda invoke --function-name ${fn.targetArn} --payload '{}' /dev/stdout`,
      },
    });

    new sst.x.DevCommand("InvokeLatest", {
      dev: {
        autostart: false,
        command: $interpolate`aws lambda invoke --function-name ${fn.latestTargetArn} --payload '{}' /dev/stdout`,
      },
    });
  },
});
