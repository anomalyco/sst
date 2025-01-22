/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Cluster with API Gateway
 *
 * Expose a service through API Gateway HTTP API using a VPC link.
 *
 * This is an alternative to using a load balancer. Since API Gateway is pay per request, it
 * works out a lot cheaper for services that don't get a lot of traffic.
 *
 * You need to specify which port in your service will be exposed through API Gateway.
 *
 * ```ts title="sst.config.ts" {3}
 * const service = cluster.addService("MyService", {
 *   serviceRegistry: {
 *     port: 80,
 *   },
 * });
 * ```
 *
 * Your API Gateway HTTP API also needs to be in the same VPC as the service.
 */
export default $config({
  app(input) {
    return {
      name: "aws-cluster-vpclink",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc", {
      // VPC links (created by `api.routePrivate()`) are not supported in all availability zones.
      // Here is the list of AZ's that support VPC link: https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-vpc-links.html
      // You still need to map these AZ ID's to your account's AZ's by executing `aws ec2 describe-availability-zones`.
      // Only include AZ's that support VPC link to prevent the following error:
      // "operation error ApiGatewayV2: BadRequestException: Subnet is in Availability Zone 'euw3-az2' where service is not available"
      // az: ["eu-west-3a", "eu-west-3c"],
    });
    const cluster = new sst.aws.Cluster("MyCluster", { vpc });
    const service = cluster.addService("MyService", {
      serviceRegistry: {
        port: 80,
      },
    });

    const api = new sst.aws.ApiGatewayV2("MyApi", { vpc });
    api.routePrivate("$default", service.nodes.cloudmapService.arn);
  },
});
