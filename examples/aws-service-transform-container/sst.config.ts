/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-service-transform-container",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc");

    const cluster = new sst.aws.Cluster("MyCluster", { vpc });
    new sst.aws.Service("MyService", {
      cluster,
      transform: {
        taskDefinition: (args) => {
          // "containerDefinitions" is a JSON string, parse first
          let value = $jsonParse(args.containerDefinitions);

          // Update "portMappings"
          value = value.apply((containerDefinitions) => {
            containerDefinitions[0].portMappings = [
              {
                containerPort: 80,
                protocol: "tcp",
              },
            ];
            return containerDefinitions;
          });

          // Convert back to JSON string
          args.containerDefinitions = $jsonStringify(value);
        },
      },
    });
  },
});
