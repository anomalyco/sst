/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Shared ALB
 *
 * Creates a standalone ALB that is shared across multiple services. Each service
 * attaches to the ALB with its own routing rules via the `loadBalancer.instance` prop.
 *
 * The ALB owns the listeners and default actions. Services attach by specifying
 * path-based (or header/query-based) conditions and explicit priorities.
 *
 * ```ts title="sst.config.ts"
 * const alb = new sst.aws.Alb("SharedAlb", {
 *   vpc,
 *   listeners: [
 *     { port: 80, protocol: "http", defaultAction: { redirect: { port: 443, protocol: "https" } } },
 *     { port: 443, protocol: "https" },
 *   ],
 * });
 * ```
 *
 * Services attach to the ALB with routing rules:
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("Api", {
 *   cluster,
 *   image: "api:latest",
 *   loadBalancer: {
 *     instance: alb,
 *     rules: [
 *       {
 *         listen: "443/https",
 *         forward: "8080/http",
 *         conditions: { path: "/api/*" },
 *         priority: 100,
 *       },
 *     ],
 *   },
 * });
 * ```
 *
 * This example creates:
 * - A shared ALB with a single HTTP listener (no domain/cert needed)
 * - An API service that handles `/api/*` requests
 * - A Web service that handles `/app/*` requests
 *
 * Deploy the "creator" stage first, then deploy "consumer" to test Alb.get():
 * ```sh
 * go run ../../cmd/sst deploy --stage creator
 * go run ../../cmd/sst deploy --stage consumer
 * ```
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
    // "creator" stage: creates VPC, Cluster, ALB, and the API service
    // "consumer" stage: references the ALB via Alb.get() and attaches a Web service
    if ($app.stage === "consumer") {
      // Reference the ALB created by the "creator" stage.
      const alb = sst.aws.Alb.get("SharedAlb", process.env.ALB_ARN!);

      const vpc = sst.aws.Vpc.get("MyVpc", process.env.VPC_ID!);
      const cluster = sst.aws.Cluster.get("MyCluster", {
        id: process.env.CLUSTER_ID!,
        vpc,
      });

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

      return {
        url: alb.url,
      };
    }

    // Default ("creator") stage
    const vpc = new sst.aws.Vpc("MyVpc");
    const cluster = new sst.aws.Cluster("MyCluster", { vpc });

    const alb = new sst.aws.Alb("SharedAlb", {
      vpc,
      listeners: [
        { port: 80, protocol: "http" },
      ],
    });

    // API service — handles /api/* on the shared ALB
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
      albArn: alb.arn,
      vpcId: vpc.id,
      clusterId: cluster.nodes.cluster.id,
    };
  },
});
