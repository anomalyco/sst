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
