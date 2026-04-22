/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Workflow Cron
 *
 * Creates an [AWS Lambda durable workflow](https://docs.aws.amazon.com/lambda/latest/dg/durable-functions.html)
 * and triggers it on a schedule using [`CronV2`](/docs/component/aws/cron-v2).
 *
 * Since `CronV2` accepts a `Workflow`, the setup is just:
 *
 * ```ts title="sst.config.ts"
 * const workflow = new sst.aws.Workflow("MyWorkflow", {
 *   handler: "src/workflow.handler",
 * });
 *
 * new sst.aws.CronV2("MyCron", {
 *   schedule: "rate(1 minute)",
 *   function: workflow,
 * });
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "aws-workflow-cron",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const workflow = new sst.aws.Workflow("MyWorkflow", {
      handler: "src/workflow.handler",
    });

    const cron = new sst.aws.CronV2("MyCron", {
      schedule: "rate(1 minute)",
      function: workflow,
    });

    return {
      schedule: cron.nodes.schedule.name,
      workflow: workflow.name,
    };
  },
});
