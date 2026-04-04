/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Durable Python
 *
 * Creates an [AWS Lambda durable workflow](https://docs.aws.amazon.com/lambda/latest/dg/durable-functions.html) using the Python runtime.
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-durable-python",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const durableWorkflow = new sst.aws.Workflow("Durable", {
      handler: "durable/main.handler",
      runtime: "python3.13",
    });

    new sst.aws.Function("Resolver", {
      handler: "resolver/main.handler",
      runtime: "python3.13",
      url: true,
      link: [durableWorkflow],
    });

    new sst.aws.Function("Invoker", {
      handler: "invoker/main.handler",
      runtime: "python3.13",
      url: true,
      link: [durableWorkflow],
    });
  },
});
