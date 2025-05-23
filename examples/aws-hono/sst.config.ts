/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-hono",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    console.log(sst.app);
    console.log(sst.paths);
    new sst.aws.Bucket("MyBucket");
    new sst.aws.Function("MyFunction", {
      handler: "./src/index.handler",
      url: true,
    });
  },
});
