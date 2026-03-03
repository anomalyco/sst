/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-hono",
      home: "cloudflare",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const ai = new sst.cloudflare.Ai("Ai");
    const bucket = new sst.cloudflare.Bucket("MyBucket");
    const hono = new sst.cloudflare.Worker("Hono", {
      url: true,
      link: [bucket, ai],
      handler: "index.ts",
    });

    return {
      api: hono.url,
    };
  },
});
