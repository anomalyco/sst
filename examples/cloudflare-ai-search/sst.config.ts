/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare AI Search
 *
 * Bind to a single AI Search instance and link it to a worker. The instance
 * must already exist in your Cloudflare account — you can create one in the
 * dashboard or with the namespace binding example.
 *
 * Once linked, you can search your indexed content and get AI-generated
 * answers using chat completions.
 */
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
      instance: "my-docs",
    });

    const worker = new sst.cloudflare.Worker("Worker", {
      handler: "index.ts",
      link: [search],
      url: true,
    });

    return {
      url: worker.url,
    };
  },
});
