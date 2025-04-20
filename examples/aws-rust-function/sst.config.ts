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
    const push = new sst.aws.Function("push", {
      runtime: "rust",
      handler: "./.push",
      url: true,
      architecture: "arm64",
      link: [bucket],
    });
    const pop = new sst.aws.Function("pop", {
      runtime: "rust",
      handler: "./.pop",
      url: true,
      architecture: "arm64",
      link: [bucket],
    });

    return { push_url: push.url, pop_url: pop.url };
  },
});
