/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare AI Search Namespace
 *
 * Bind to an AI Search namespace to manage multiple instances at runtime. You
 * can create, list, search, and delete instances dynamically — useful for
 * multi-tenant apps or admin tools.
 *
 * Every Cloudflare account has a `default` namespace. You can also create
 * custom namespaces to isolate groups of instances.
 *
 * For a simpler single-instance setup, see the `cloudflare-ai-search` example.
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-ai-search-namespace",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const search = new sst.cloudflare.AiSearch("Search", {
      namespace: "default",
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
