/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Aurora DSQL with Drizzle
 *
 * In this example, we use Drizzle ORM with an Aurora DSQL cluster.
 *
 * ```ts title="sst.config.ts"
 * const cluster = new sst.aws.Dsql("MyCluster");
 * ```
 *
 * And link it to a Lambda function.
 *
 * ```ts title="sst.config.ts" {4}
 * new sst.aws.Function("MyApi", {
 *   handler: "src/api.handler",
 *   link: [cluster],
 *   url: true,
 * });
 * ```
 *
 * Push the Drizzle schema to the database.
 *
 * ```bash
 * sst shell -- bun run push.ts
 * ```
 *
 * Now in the function we can connect to the cluster using Drizzle with the DSQL connector.
 * Learn more about [DSQL Node.js connectors](https://docs.aws.amazon.com/aurora-dsql/latest/userguide/SECTION_Node-js-connectors.html).
 *
 * ```ts title="src/drizzle.ts"
 * import { drizzle } from "drizzle-orm/node-postgres";
 * import { AuroraDSQLPool } from "@aws/aurora-dsql-node-postgres-connector";
 * import { Resource } from "sst";
 *
 * const pool = new AuroraDSQLPool({
 *   host: Resource.MyCluster.endpoint,
 *   user: "admin",
 * });
 *
 * export const db = drizzle(pool, { schema });
 * ```
 */
export default $config({
  app(input) {
    return {
      name: "aws-dsql-drizzle",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const cluster = new sst.aws.Dsql("MyCluster");

    new sst.aws.Function("MyApi", {
      handler: "src/api.handler",
      link: [cluster],
      url: true,
    });

    return {
      endpoint: cluster.endpoint,
      region: cluster.region,
    };
  },
});
