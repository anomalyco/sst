/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Cluster Logging with Fluent Bit and Datadog
 *
 * This example, shows how to use the Fluent Bit logging driver with Datadog for your services.
 */
export default $config({
  app(input) {
    return {
      name: "aws-cluster-logging-fluentbit-datadog",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc");

    const cluster = new sst.aws.Cluster("MyCluster", { vpc });
    new sst.aws.Service("MyService", {
      cluster,
      loadBalancer: {
        ports: [{ listen: "80/http", container: "app" }],
      },
      containers: [
        {
          name: "app",
          image: {
            dockerfile: "./Dockerfile",
          },
          logging: {
            driver: "awsfirelens",
            options: {
              Name: "datadog",
              apikey: "<replace-me>",
              Host: "http-intake.logs.datadoghq.com",
              dd_service: "aws-cluster-logging-fluentbit-datadog",
              dd_source: "aws-cluster-logging-fluentbit-datadog",
              TLS: "on",
              provider: "ecs",
            },
          },
        },
        {
          name: "log-router",
          image: "public.ecr.aws/aws-observability/aws-for-fluent-bit:stable",
          firelens: {
            type: "fluentbit",
            options: {
              "enable-ecs-log-metadata": "true",
            },
          },
          logging: {
            name: `/sst/cluster/${$app.name}-${$app.stage}-MyCluster/MyService/log-router`,
          },
        },
      ],
    });
  },
});
