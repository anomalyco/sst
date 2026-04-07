/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Workflow
 *
 * Creates an [AWS Lambda durable workflow](https://docs.aws.amazon.com/lambda/latest/dg/durable-functions.html)
 */
export default $config({
  app(input) {
    return {
      name: "aws-workflow",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const workflow = new sst.aws.Workflow("Workflow", {
      handler: "src/workflow.handler",
    });

    const resolver = new sst.aws.Function("Resolver", {
      handler: "src/resolver.handler",
      url: true,
      link: [workflow],
    });

    const invoker = new sst.aws.Function("Invoker", {
      handler: "src/invoker.handler",
      url: true,
      link: [workflow, resolver],
    });

    return {
      workflow: workflow.name,
      invoker: invoker.url,
      resolver: resolver.url,
    };
  },
});
