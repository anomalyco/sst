/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Linkable env vars
 *
 * Pass SST link env vars to a native `aws.ecs.TaskDefinition` container using
 * `sst.Linkable.env()`. This lets `Resource.MyResource` work at runtime
 * in compute not managed by SST.
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-linkable-env",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Create an SST bucket
    const bucket = new sst.aws.Bucket("MyBucket");

    // Create a custom linkable
    const linkable = new sst.Linkable("MyLinkable", {
      properties: {
        foo: "bar",
      },
    });

    // Create VPC and ECS cluster using native AWS resources
    const vpc = new aws.ec2.Vpc("Vpc", { cidrBlock: "10.0.0.0/16" });
    const subnet = new aws.ec2.Subnet("Subnet", {
      vpcId: vpc.id,
      cidrBlock: "10.0.0.0/24",
    });
    const cluster = new aws.ecs.Cluster("Cluster");

    // Linkable.env() returns a Record<string, string>, but ECS expects
    // environment as an array of { name, value } objects
    const environment = sst.Linkable.env([bucket, linkable]).apply((env) =>
      Object.entries(env).map(([name, value]) => ({ name, value })),
    );

    const taskDefinition = new aws.ecs.TaskDefinition("TaskDefinition", {
      family: $interpolate`${$app.name}-${$app.stage}`,
      cpu: "256",
      memory: "512",
      networkMode: "awsvpc",
      requiresCompatibilities: ["FARGATE"],
      containerDefinitions: $jsonStringify([
        {
          name: "app",
          image: "public.ecr.aws/docker/library/node:20-slim",
          essential: true,
          environment,
        },
      ]),
    });

    new aws.ecs.Service("Service", {
      cluster: cluster.arn,
      taskDefinition: taskDefinition.arn,
      desiredCount: 0,
      launchType: "FARGATE",
      networkConfiguration: {
        subnets: [subnet.id],
      },
    });
  },
});
