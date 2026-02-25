/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Subscribe to queues with dead-letter queue
 *
 * Messages not processed successfully by the primary subscriber function will be sent to the dead-letter queue after the retry limit is reached.
 */
export default $config({
  app(input) {
    return {
      name: "aws-dead-letter-queue",
      home: "aws",
      removal: input.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    // create dead letter queue
    const dlq = new sst.aws.Queue("DeadLetterQueue");
    dlq.subscribe("subscriber.dlq");

    // create main queue
    const queue = new sst.aws.Queue("MyQueue", {
      dlq: dlq.arn,
    });
    queue.subscribe("subscriber.main");

    const app = new sst.aws.Function("MyApp", {
      handler: "publisher.handler",
      link: [queue],
      url: true,
    });

    return {
      app: app.url,
      queue: queue.url,
      dlq: dlq.url,
    };
  },
});
