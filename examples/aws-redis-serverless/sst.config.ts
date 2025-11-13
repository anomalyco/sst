/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-redis-serverless",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Serverless Redis doesn't require NAT Gateways
    const vpc = new sst.aws.Vpc("MyVpc");
    const redis = new sst.aws.Redis("MyRedis", {
      vpc,
      serverless: {
        dataStorage: { maximum: 10, unit: "GB" },
        ecpuPerSeconds: { maximum: 5000 }
      }
    });
    new sst.aws.Function("MyApp", {
      handler: "index.handler",
      url: true,
      vpc,
      link: [redis],
    });
  },
});
