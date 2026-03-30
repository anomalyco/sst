/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-cloudflare-combined",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      providers: {
        cloudflare: "6.13.0",
      },
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("MyBucket");
    const api = new sst.aws.Function("Hono", {
      url: true,
      link: [bucket],
      handler: "src/index.handler",
    });

    const worker = new sst.cloudflare.Worker("MyWorker", {
      handler: "./worker.ts",
      url: true,
    });

    return {
      api: api.url,
      worker: worker.url,
    };
  },
});
