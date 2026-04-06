/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Durable
 *
 * Creates an [Durable Function](https://docs.aws.amazon.com/lambda/latest/dg/durable-functions.html)
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
    const durableFunction = new sst.aws.Function("Durable", {
      handler: "src/index.handler",
      durable: true,
      url: true,
    });

    new sst.aws.Function("Resolver", {
      handler: "src/resolver.handler",
      url: true,
      link: [durableFunction],
    });

    new sst.aws.Function("Invoker", {
      handler: "src/invoker.handler",
      url: true,
      link: [durableFunction],
    });
  },
});
