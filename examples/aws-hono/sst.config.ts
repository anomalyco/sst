/// <reference path="./.sst/platform/config.d.ts" />
export default $config({
  app(input) {
    return {
      name: "aws-hono",
      removal: input?.stage === "production" ? "retain" : "remove",
      protect: input?.stage === "production",
      home: "aws",
      providers: {
        jetstream: { package: "@paynearme/pulumi-jetstream", version: "0.2.1" },
        stripe: { package: "@sst-provider/stripe-official", version: "0.2.1" },
        planetscale: { package: "@sst-provider/planetscale", version: "1.0.0" },
      },
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("MyBucket");
    new sst.aws.Function("Hono", {
      url: true,
      link: [bucket],
      handler: "src/index.handler",
    });
  },
});
