/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-hono",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      version: "3.10.13",
      plugins: {
        example: "0.0.10",
        "sst-plugin-aws": "link:sst-plugin-aws",
      },
    };
  },
  async run() {
    new aws.s3.Bucket("MyBucket", {});
  },
});
