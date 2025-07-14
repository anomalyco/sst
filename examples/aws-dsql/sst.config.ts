/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-dsql",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Single region cluster
    const cluster = new sst.aws.Dsql("MyCluster", {
      deletionProtection: false,
      tags: {
        Environment: $app.stage,
        Example: "aws-dsql",
        Owner: "sst-team",
      },
    });

    // Create a function that can connect to the cluster
    const fn = new sst.aws.Function("MyFunction", {
      handler: "src/lambda.handler",
      link: [cluster],
    });

    return {
      cluster: {
        arn: cluster.arn,
        identifier: cluster.identifier,
        endpoint: cluster.publicEndpoint,
      },
      function: fn.arn,
    };
  },
});
