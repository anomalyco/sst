/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Lambda Durable Function using Python Runtime
 *
 * Creates an [Durable Function](https://docs.aws.amazon.com/lambda/latest/dg/durable-functions.html) using the Python runtime.
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
    const durableFunction = new sst.aws.Function("Durable", {
      handler: "src/main.handler",
			runtime: "python3.13",
      durable: true,
    });

    new sst.aws.Function("Resolver", {
      handler: "src/resolver.handler",
			runtime: "python3.13",
      url: true,
      link: [durableFunction],
    });

    new sst.aws.Function("Invoker", {
      handler: "src/invoker.handler",
			runtime: "python3.13",
      url: true,
      link: [durableFunction],
    });
  },
});
