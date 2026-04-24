/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-ai-search",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    // Bind to the "default" namespace — every Cloudflare account has one
    // automatically. Change this to target a different namespace.
    const search = new sst.cloudflare.AiSearch("Search", {
      namespace: "default",
    });

    const worker = new sst.cloudflare.Worker("Worker", {
      handler: "./index.ts",
      link: [search],
      url: true,
    });

    return {
      api: worker.url,
    };
  },
});
