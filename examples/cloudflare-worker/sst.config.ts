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
    const worker = new sst.cloudflare.Worker("MyWorker", {
      handler: "./index.ts",
      link: [bucket],
      url: true,
      compatibility: {
        date: "2026-02-05",
        flags: ["nodejs_compat", "enable_nodejs_http_modules"],
      },
    });

    return {
      api: worker.url,
    };
  },
});
