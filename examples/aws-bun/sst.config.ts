/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Bun container
 *
 * Deploy a Bun app in a container with a linked S3 bucket.
 */
export default $config({
  app(input) {
    return {
      name: "aws-bun",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc");
    const bucket = new sst.aws.Bucket("MyBucket");

    const cluster = new sst.aws.Cluster("MyCluster", { vpc });
    new sst.aws.Service("MyService", {
      cluster,
      loadBalancer: {
        ports: [{ listen: "80/http", forward: "3000/http" }],
      },
      dev: {
        command: "bun dev",
      },
      link: [bucket],
    });
  },
});
