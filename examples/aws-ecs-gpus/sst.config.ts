/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS ECS GPUs
 *
 * A minimal ECS service running on ECS Managed Instances with a GPU-enabled host.
 * The service uses top-level `gpu`, `cpu`, `memory`, and `storage` settings, while
 * the managed instances IAM resources remain customizable through `transform`.
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

    const service = new sst.aws.Service("MyService", {
      cluster,
      image: { context: "./" },
      gpu: "nvidia/t4",
      cpu: "4 vCPU",
      memory: "16 GB",
      loadBalancer: {
        ports: [{ listen: "80/http", forward: "8000/http" }],
      },
    });

    return {
      url: service.url,
    };
  },
});
