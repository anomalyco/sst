/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Lambda Durable Function
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
      handler: "index.handler",
      durable: true,
    });

    new sst.aws.Function("Resolver", {
      handler: "resolver.handler",
      url: true,
      link: [durableFunction],
    });

    new sst.aws.Function("Invoker", {
      handler: "invoker.handler",
      url: true,
      link: [durableFunction],
    });
  },
});
