/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Subscribe to queues
 *
 * Create an SQS queue, subscribe to it, and publish to it from a function.
 */
export default $config({
  app(input) {
    return {
      name: "aws-queue",
      home: "aws",
      removal: input.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const queue = new sst.aws.Queue("MyQueue");
    queue.subscribe("subscriber.handler", {
      batch: {
        partialResponses: true,
      }
    });

    const app = new sst.aws.Function("MyApp", {
      handler: "publisher.handler",
      link: [queue],
      url: true,
    });

    return {
      app: app.url,
      queue: queue.url,
    };
  },
});
