/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare Rate Limit
 *
 * This example creates a Cloudflare Rate Limit and a Worker that applies it.
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-rate-limit",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const rateLimit = new sst.cloudflare.RateLimit("MyRateLimit", {
      namespaceId: 1001,
      limit: 100,
      period: "1 minute",
    });

    const worker = new sst.cloudflare.Worker("MyWorker", {
      handler: "./index.ts",
      url: true,
      link: [rateLimit],
    });

    return {
      api: worker.url,
    };
  },
});
