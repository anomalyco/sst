/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS ECS GPUs
 *
 * A minimal ECS service running on ECS Managed Instances with a GPU-enabled host.
 * The service uses top-level `gpu`, `cpu`, `memory`, and `storage` settings, while
 * the managed instances IAM resources remain customizable through `transform`.
 *
 * A private API Gateway HTTP API is used to test the service without exposing a public
 * load balancer.
 */
export default $config({
  app(input) {
    return {
      name: "service-gpu-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc");
    const cluster = new sst.aws.Cluster("MyCluster", { vpc });

    // Provisions g4dn.xlarge
    const service = new sst.aws.Service("MyService", {
      cluster,
      image: { context: "./" },
      gpu: "nvidia/t4",
      cpu: "4 vCPU",
      memory: "10 GB",
      serviceRegistry: {
        port: 8000,
      },
    });

    const api = new sst.aws.ApiGatewayV2("MyApi", { vpc });
    api.routePrivate("$default", service.nodes.cloudmapService.arn);

    return {
      service: service.service,
      api: api.url,
    };
  },
});
