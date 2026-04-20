/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-astro",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const bucket = new sst.cloudflare.Bucket("MyBucket");
    const kv = new sst.cloudflare.Kv("MyKv");

    new sst.cloudflare.Astro("MyWeb", {
      link: [bucket, kv],
    });
  },
});
