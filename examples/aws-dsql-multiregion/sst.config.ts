/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-dsql-multiregion",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Create a multi-region cluster
    const cluster = new sst.aws.Dsql("MyCluster", {
      multiRegion: {
        witnessRegion: "us-west-2",
        peerRegion: "us-east-2",
      },
    });

    // Create a function that can connect to the cluster
    const fn = new sst.aws.Function("MyFunction", {
      handler: "src/lambda.handler",
      link: [cluster],
    });

    return {
      function: fn.arn,
    };
  },
});
