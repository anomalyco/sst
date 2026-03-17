/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Shared ALB
 *
 * Creates a standalone ALB shared across multiple services.
 * Shows advanced routing with path conditions, header conditions, and health checks.
 *
 * ```ts title="sst.config.ts"
 * const alb = new sst.aws.Alb("SharedAlb", {
 *   vpc,
 *   listeners: [
 *     { port: 80, protocol: "http" },
 *   ],
 * });
 * ```
 *
 * Services can use header-based routing in addition to path-based:
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("InternalApi", {
 *   cluster,
 *   image: { context: "api/" },
 *   loadBalancer: {
 *     instance: alb,
 *     rules: [
 *       {
 *         listen: "80/http",
 *         forward: "3000/http",
 *         conditions: {
 *           path: "/api/*",
 *           header: { name: "x-internal", values: ["true"] },
 *         },
 *         priority: 50,
 *       },
 *     ],
 *   },
 * });
 * ```
 *
 * This example creates:
 * - A shared ALB with an HTTP listener
 * - An API service with path-based routing and custom health check
 * - A Web service with path-based routing
 * - Both services share the same ALB
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

    // Create a shared ALB with an HTTP listener
    const alb = new sst.aws.Alb("SharedAlb", {
      vpc,
      listeners: [{ port: 80, protocol: "http" }],
    });

    // API service — handles /api/* with a custom health check path
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
        health: {
          "3000/http": {
            path: "/api/health",
            interval: "10 seconds",
            timeout: "5 seconds",
            healthyThreshold: 2,
            unhealthyThreshold: 3,
          },
        },
      },
    });

    // Web service — handles everything else under /app/*
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
        health: {
          "3000/http": {
            path: "/app/health",
            interval: "10 seconds",
            timeout: "5 seconds",
          },
        },
      },
    });

    return {
      url: alb.url,
    };
  },
});
