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

    new sst.cloudflare.TanStackStart("MyWeb", {
      link: [kv],
    });
  },
});
