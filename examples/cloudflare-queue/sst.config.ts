/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare Queue
 *
 * This example creates a Cloudflare Queue with a producer and consumer worker.
 *
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-queue",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const queue = new sst.cloudflare.Queue("MyQueue");

    queue.subscribe("consumer.ts");

    const producer = new sst.cloudflare.Worker("Producer", {
      handler: "producer.ts",
      link: [queue],
      url: true,
    });

    return {
      url: producer.url,
    };
  },
});
