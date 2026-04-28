/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-worker",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const bucket = new sst.cloudflare.Bucket("MyBucket");
    const secret = new sst.Secret("MySecret", "cloudflare-worker-secret");
    const worker = new sst.cloudflare.Worker("MyWorker", {
      handler: "./index.ts",
      link: [bucket, secret],
      url: true,
      environment: {
        API_URL: "https://example.com/cloudflare-worker",
      },
    });

    return {
      api: worker.url,
    };
  },
});
