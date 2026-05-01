/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare Workflow
 *
 * This example creates a Cloudflare Workflow and a Worker that triggers it.
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-workflow",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const processor = new sst.cloudflare.Workflow("OrderProcessor", {
      handler: "./workflow.ts",
      className: "OrderProcessor",
    });

    const api = new sst.cloudflare.Worker("Api", {
      handler: "./api.ts",
      link: [processor],
      url: true,
    });

    return {
      api: api.url,
    };
  },
});
