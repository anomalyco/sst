/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-rust-lambda",
      removal: input?.stage === "production" ? "retain" : "remove",
      protect: ["production"].includes(input?.stage),
      home: "aws",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("Bucket");
    const lambda = new sst.aws.Function("RustFunction", {
      runtime: "rust",
      handler: "./",
      url: true,
      architecture: "arm64",
      link: [bucket],
    });

    return { url: lambda.url };
  },
});
