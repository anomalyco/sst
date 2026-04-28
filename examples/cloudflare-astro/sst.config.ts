/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare Astro
 *
 * Deploy an [Astro](https://astro.build) site to Cloudflare.
 */
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
    const secret = new sst.Secret("MySecret", "cloudflare-astro-secret");

    new sst.cloudflare.Astro("MyWeb", {
      link: [bucket, kv, secret],
      environment: {
        API_URL: "https://example.com/cloudflare-astro",
      },
    });
  },
});
