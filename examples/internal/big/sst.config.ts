/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-big",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    for (let i = 0; i < 10; i++) {
      new sst.aws.Function("MyFunction" + i, {
        handler: "index.handler",
      });
    }
  },
});
