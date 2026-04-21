/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare TanStack Start
 *
 * Deploy a [TanStack Start](https://tanstack.com/start/latest) app to Cloudflare.
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-tanstack-start",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const kv = new sst.cloudflare.Kv("MyKv");

    new sst.cloudflare.TanStackStart("MyWeb", {
      link: [kv],
    });
  },
});
