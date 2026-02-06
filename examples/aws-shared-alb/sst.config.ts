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
 *   image: { context: "api/" },
 *   loadBalancer: {
 *     instance: alb,
 *     rules: [
 *       {
 *         listener: "443/https",
 *         forward: "3000/http",
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
    const vpc = new sst.aws.Vpc("MyVpc");
    const cluster = new sst.aws.Cluster("MyCluster", { vpc });

    // Create a shared ALB with a single HTTP listener (no domain/cert needed)
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
            listener: "80/http",
            forward: "3000/http",
            conditions: { path: "/api/*" },
            priority: 100,
          },
        ],
      },
    });

    // Web service — handles /app/* on the shared ALB
    new sst.aws.Service("Web", {
      cluster,
      image: { context: "web/" },
      loadBalancer: {
        instance: alb,
        rules: [
          {
            listener: "80/http",
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
  },
});
