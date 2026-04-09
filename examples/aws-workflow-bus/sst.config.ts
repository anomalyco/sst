/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Workflow Bus
 *
 * Creates an [AWS Lambda durable workflow](https://docs.aws.amazon.com/lambda/latest/dg/durable-functions.html)
 * and triggers it with a [`Bus`](/docs/component/aws/bus).
 */
export default $config({
  app(input) {
    return {
      name: "aws-workflow-bus",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const workflow = new sst.aws.Workflow("MyWorkflow", {
      handler: "src/workflow.handler",
    });

    const bus = new sst.aws.Bus("Bus");

    bus.subscribe("Workflow", workflow, {
      pattern: {
        detailType: ["app.workflow.requested"],
      },
    });

    const publisher = new sst.aws.Function("Publisher", {
      handler: "src/publisher.handler",
      url: true,
      link: [bus],
    });

    return {
      bus: bus.name,
      publisher: publisher.url,
      workflow: workflow.name,
    };
  },
});
