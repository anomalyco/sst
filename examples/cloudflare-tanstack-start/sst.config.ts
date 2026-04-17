/// <reference path="./.sst/platform/config.d.ts" />

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

    const site = new sst.cloudflare.TanStackStart("MyWeb", {
      link: [kv],
    });

    return {
      url: site.url,
    };
  },
});
