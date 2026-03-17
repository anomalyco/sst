/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Aurora DSQL Multi-Region
 *
 * In this example, we deploy a multi-region Aurora DSQL cluster and connect to both
 * clusters from a Lambda function.
 *
 * :::note
 * Multi-region with VPCs is not currently supported.
 * :::
 *
 * Create the cluster with a witness region and a peer region. The witness must differ
 * from both cluster regions.
 *
 * ```ts title="sst.config.ts"
 * const cluster = new sst.aws.Dsql("MultiRegion", {
 *   regions: {
 *     witness: "us-west-2",
 *     peer: "us-east-2",
 *   },
 * });
 * ```
 *
 * Connect to both clusters from your function using the DSQL connector.
 * Learn more about [DSQL Node.js connectors](https://docs.aws.amazon.com/aurora-dsql/latest/userguide/SECTION_Node-js-connectors.html).
 *
 * ```ts title="lambda.ts"
 * import { AuroraDSQLClient } from "@aws/aurora-dsql-node-postgres-connector";
 * import { Resource } from "sst";
 *
 * async function connectToCluster(endpoint: string) {
 *   const client = new AuroraDSQLClient({ host: endpoint, user: "admin" });
 *   await client.connect();
 *   return client;
 * }
 *
 * // Cluster in us-east-1
 * const usEast1 = await connectToCluster(Resource.MultiRegion.endpoint);
 *
 * // Cluster in us-east-2
 * const usEast2 = await connectToCluster(Resource.MultiRegion.peer.endpoint);
 * ```
 */

export default $config({
  app(input) {
    return {
      name: "aws-dsql-multiregion",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      providers: {
        aws: { region: "us-east-1" },
      },
    };
  },
  async run() {
    const cluster = new sst.aws.Dsql("MultiRegion", {
      regions: {
        witness: "us-west-2",
        peer: "us-east-2",
      },
    });

    const fn = new sst.aws.Function("MyFunction", {
      handler: "lambda.handler",
      link: [cluster],
      url: true,
    });

    return {
      endpoint: cluster.endpoint,
      region: cluster.region,
      peerEndpoint: cluster.peer.endpoint,
      peerRegion: cluster.peer.region,
    };
  },
});
