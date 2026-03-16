/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Aurora DSQL in a VPC
 *
 * In this example, we connect to an Aurora DSQL cluster privately from a Lambda
 * function using VPC endpoints, without routing traffic over the public internet.
 *
 * Create a VPC, then create the cluster with a connection endpoint inside it.
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 *
 * const cluster = new sst.aws.Dsql("MyCluster", {
 *   vpc: {
 *     instance: vpc,
 *     endpoints: { connection: true },
 *   },
 * });
 * ```
 *
 * Link the cluster to a function that's also in the VPC. The linked `endpoint` will
 * automatically resolve to the private VPC endpoint hostname instead of the public one.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("MyFunction", {
 *   handler: "lambda.handler",
 *   vpc,
 *   link: [cluster],
 * });
 * ```
 *
 * Connect from your function using the DSQL connector — no config changes needed.
 * Learn more about [DSQL Node.js connectors](https://docs.aws.amazon.com/aurora-dsql/latest/userguide/SECTION_Node-js-connectors.html).
 *
 * ```ts title="lambda.ts"
 * import { AuroraDSQLClient } from "@aws/aurora-dsql-node-postgres-connector";
 * import { Resource } from "sst";
 *
 * const client = new AuroraDSQLClient({
 *   host: Resource.MyCluster.endpoint,
 *   user: "admin",
 * });
 *
 * await client.connect();
 * const result = await client.query("SELECT NOW()");
 * await client.end();
 * ```
 */

export default $config({
  app(input) {
    return {
      name: "aws-dsql-vpc",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("singleClusterVpc");

    const cluster = new sst.aws.Dsql("MyCluster", {
      vpc: {
        instance: vpc,
        endpoints: {
          connection: true,
          management: false,
        },
      },
    });

    const fn = new sst.aws.Function("MyFunction", {
      handler: "lambda.handler",
      vpc,
      link: [cluster],
      url: true,
    });

    return {
      endpoint: cluster.endpoint,
      region: cluster.region,
    };
  },
});
