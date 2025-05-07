/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "mc",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MCVPC");
    const cluster = new sst.aws.Cluster("MCCluster", { vpc });
    const filesystem = new sst.aws.Efs("MCFilesystem", { vpc });
    new sst.aws.Service("MCServiceTCP", {
      cluster,
      loadBalancer: {
        rules: [{ listen: "25565/tcp" }],
        domain: {
          name: "mc.example.com",
          dns: sst.cloudflare.dns({ zone: "<cloudflare token>" }),
        },
      },
      link: [filesystem],
      volumes: [{ efs: filesystem, path: "/data" }],
      image: "itzg/minecraft-server",
      cpu: "4 vCPU",
      memory: "8 GB",
      environment: {
        EULA: "TRUE",
      },
    });
    new sst.aws.Service("MCServiceUDP", {
      cluster,
      loadBalancer: {
        rules: [{ listen: "25565/udp" }],
      },
      link: [filesystem],
      volumes: [{ efs: filesystem, path: "/data" }],
      image: "itzg/minecraft-server",
      cpu: "4 vCPU",
      memory: "8 GB",
      environment: {
        EULA: "TRUE",
      },
    });
  },
});
