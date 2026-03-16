/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Shared ALB
 *
 * Creates a standalone ALB that is shared across stages. In dev, the ALB is
 * referenced via `Alb.get()`. In production, it's created fresh.
 *
 * Uses the `$dev ? get : new` pattern to share infrastructure across stages.
 */
export default $config({
  app(input) {
    return {
      name: "aws-shared-alb",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      providers: {
        aws: {
          region: "us-east-1",
        },
      },
    };
  },
  async run() {
    const vpc = $dev
      ? sst.aws.Vpc.get("MyVpc", "vpc-xxx")
      : new sst.aws.Vpc("MyVpc");

    const cluster = $dev
      ? sst.aws.Cluster.get("MyCluster", { id: "cluster-xxx", vpc })
      : new sst.aws.Cluster("MyCluster", { vpc });

    const alb = $dev
      ? sst.aws.Alb.get("SharedAlb", "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/xxx")
      : new sst.aws.Alb("SharedAlb", {
          vpc,
          listeners: [
            { port: 80, protocol: "http" },
          ],
        });

    if ($dev) {
      new sst.aws.Service("Web", {
        cluster,
        image: { context: "web/" },
        loadBalancer: {
          instance: alb,
          rules: [
            {
              listen: "80/http",
              forward: "3000/http",
              conditions: { path: "/app/*" },
              priority: 200,
            },
          ],
        },
      });
    }

    new sst.aws.Service("Api", {
      cluster,
      image: { context: "api/" },
      loadBalancer: {
        instance: alb,
        rules: [
          {
            listen: "80/http",
            forward: "3000/http",
            conditions: { path: "/api/*" },
            priority: 100,
          },
        ],
      },
    });

    return {
      url: alb.url,
    };
  },
});
