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
 * ```ts title="sst.config.ts" {4}
 * const service = new sst.aws.Service("MyService", {
 *   cluster,
 *   serviceRegistry: {
 *     port: 80,
 *   },
 * });
 * ```
 *
 * A couple of things to note:
 *
 * 1. Your API Gateway HTTP API also needs to be in the **same VPC** as the service.
 *
 * 2. You also need to verify that your VPC's [**availability zones support VPC link**](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-vpc-links.html#http-api-vpc-link-availability).
 *
 * 3. Run `aws ec2 describe-availability-zones` to get a list of AZs for your
 *    account.
 *
 * 4. Only list the AZ ID's that support VPC link.
 *    ```ts title="sst.config.ts" {4}
 *    vpc: {
 *      az: ["eu-west-3a", "eu-west-3c"]
 *    }
 *    ```
 *    If the VPC picks an AZ automatically that doesn't support VPC link, you'll get
 *    the following error:
 *    ```
 *    operation error ApiGatewayV2: BadRequestException: Subnet is in Availability
 *    Zone 'euw3-az2' where service is not available
 *    ```
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
      // Pick at least two AZs that support VPC link
      // az: ["eu-west-3a", "eu-west-3c"],
    });
    const cluster = new sst.aws.Cluster("MyCluster", { vpc });
    const service = new sst.aws.Service("MyService", {
      cluster,
      serviceRegistry: {
        port: 80,
      },
    });

    const api = new sst.aws.ApiGatewayV2("MyApi", { vpc });
    api.routePrivate("$default", service.nodes.cloudmapService.arn);
  },
});
