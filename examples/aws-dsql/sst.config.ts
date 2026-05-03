/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Aurora DSQL
 *
 * In this example, we deploy an Aurora DSQL cluster.
 *
 * ```ts title="sst.config.ts"
 * const cluster = new sst.aws.Dsql("MyCluster");
 * ```
 *
 * And link it to a Lambda function.
 *
 * ```ts title="sst.config.ts" {4}
 * new sst.aws.Function("MyFunction", {
 *   handler: "lambda.handler",
 *   link: [cluster],
 *   url: true,
 * });
 * ```
 *
 * Now in the function we can connect to the cluster using the DSQL connector.
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
      name: "aws-dsql",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const cluster = new sst.aws.Dsql("MyCluster", {});

    const fn = new sst.aws.Function("MyFunction", {
      handler: "lambda.handler",
      link: [cluster],
      url: true,
    });

    return {
      endpoint: cluster.endpoint,
      region: cluster.region,
    };
  },
});