/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare Durable Object
 *
 * This example creates a Durable Object and links it to a worker.
 *
 * Send a `GET` request to the `url` output. The worker calls the Durable
 * Object, and the Durable Object logs the current count.
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-durable-object",
      home: "cloudflare",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const counter = new sst.cloudflare.DurableObject("Counter");

    const api = new sst.cloudflare.Worker("Api", {
      durableObjectMigrations: [
        {
          tag: "v1",
          newSqliteClasses: [counter.className],
        },
      ],
      handler: "worker.ts",
      link: [counter],
      url: true,
    });

    return {
      url: api.url,
    };
  },
});
