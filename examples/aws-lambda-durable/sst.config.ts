/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Durable
 *
 * Creates an [AWS Lambda durable workflow](https://docs.aws.amazon.com/lambda/latest/dg/durable-functions.html)
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-durable",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const durableWorkflow = new sst.aws.Workflow("Durable", {
      handler: "src/index.handler",
    });

    const resolver = new sst.aws.Function("Resolver", {
      handler: "src/resolver.handler",
      url: true,
      link: [durableWorkflow],
    });

    const invoker = new sst.aws.Function("Invoker", {
      handler: "src/invoker.handler",
      url: true,
      link: [durableWorkflow, resolver],
    });

    return {
      workflow: durableWorkflow.name,
      invoker: invoker.url,
      resolver: resolver.url,
    };
  },
});
