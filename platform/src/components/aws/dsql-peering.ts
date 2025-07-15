import { ComponentResourceOptions, Input, Output, all } from "@pulumi/pulumi";
import { Component } from "../component";
import type { Dsql } from "./dsql";
import { DsqlPeeringProvider } from "./providers/dsql-peering";
import { Region } from "@pulumi/aws";

export interface DsqlPeeringArgs {
  /**
   * The primary DSQL cluster to peer.
   */
  primaryCluster: Dsql;

  /**
   * The peer DSQL cluster to peer with the primary.
   */
  peerCluster: Dsql;

  /**
   * Witness region for multi-region clusters.
   *
   * Must be in the same region set as the primary and peer regions.
   * See https://docs.aws.amazon.com/aurora-dsql/latest/userguide/what-is-aurora-dsql.html#region-availability
   * The witness region is used for consensus in the distributed system.
   *
   * @example
   * ```js
   * {
   *   witnessRegion: "us-west-2"
   * }
   * ```
   */
  witnessRegion: Input<Region>;

  /**
   * Tags to associate with the peering.
   *
   * @example
   * ```js
   * {
   *   tags: {
   *     Environment: "production",
   *     Team: "backend"
   *   }
   * }
   * ```
   */
  tags?: Input<Record<string, Input<string>>>;
}

/**
 * The `DsqlPeering` component lets you create a multi-region peering connection between
 * two [Amazon Aurora DSQL](https://aws.amazon.com/rds/aurora/dsql/) clusters.
 *
 * This component coordinates the peering setup and ensures proper deletion order to
 * avoid deadlocks during cluster deletion. The peering creation waits for both clusters
 * to reach ACTIVE status before completing, ensuring the multi-region setup is fully ready.
 *
 * @example
 *
 * #### Create clusters and peer them
 *
 * ```ts title="sst.config.ts"
 * // Create clusters in different regions
 * const witnessRegion = "us-west-2";
 *
 * const primary = new sst.aws.Dsql("Primary", {
 *   deletionProtection: false, // Required for deletion
 *   witnessRegion: witnessRegion // Required for multi-region
 * });
 *
 * const peerProvider = new aws.Provider("PeerRegion", { region: "us-east-2" });
 * const peer = new sst.aws.Dsql("Peer", {
 *   deletionProtection: false, // Required for deletion
 *   witnessRegion: witnessRegion // Required for multi-region
 * }, { provider: peerProvider });
 *
 * // Create peering between them
 * const peering = new sst.aws.DsqlPeering("Peering", {
 *   primaryCluster: primary,
 *   peerCluster: peer,
 *   witnessRegion: witnessRegion
 * });
 * ```
 *
 * :::note
 * Both clusters must have the same `witnessRegion` set during creation and `deletionProtection: false`
 * for the peering deletion to work properly. The peering component will not automatically disable
 * deletion protection for security reasons.
 * :::
 *
 * #### Link to a function
 *
 * You can link the clusters to other resources, like a function.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("MyFunction", {
 *   handler: "src/lambda.handler",
 *   link: [primary, peer]
 * });
 * ```
 *
 * Once linked, you can connect to either cluster from your function code.
 *
 * ```ts title="src/lambda.ts"
 * import { Resource } from "sst";
 * import { DsqlSigner } from "@aws-sdk/dsql-signer";
 * import { Client } from "pg";
 *
 * // Connect to primary cluster
 * const signer = new DsqlSigner({
 *   region: Resource.Primary.region,
 *   hostname: Resource.Primary.publicEndpoint,
 * });
 *
 * const token = await signer.getDbConnectAdminAuthToken();
 *
 * const client = new Client({
 *   host: Resource.Primary.publicEndpoint,
 *   port: 5432,
 *   database: "postgres",
 *   user: "admin",
 *   password: token,
 *   ssl: true
 * });
 * ```
 */
export class DsqlPeering extends Component {
  private peeringProvider: Output<DsqlPeeringProvider>;

  constructor(
    name: string,
    args: DsqlPeeringArgs,
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    this.peeringProvider = all([
      args.primaryCluster.arn,
      args.peerCluster.arn,
      args.primaryCluster.identifier,
      args.peerCluster.identifier,
      args.primaryCluster.region,
      args.peerCluster.region,
      args.witnessRegion,
      args.tags,
    ]).apply(
      ([
        primaryClusterArn,
        peerClusterArn,
        primaryClusterIdentifier,
        peerClusterIdentifier,
        primaryRegion,
        peerRegion,
        witnessRegion,
        tags,
      ]) => {
        return new DsqlPeeringProvider(
          `${name}Provider`,
          {
            primaryClusterArn,
            peerClusterArn,
            primaryClusterIdentifier,
            peerClusterIdentifier,
            primaryRegion,
            peerRegion,
            witnessRegion,
            tags,
          },
          {
            parent: this,
            dependsOn: [args.primaryCluster, args.peerCluster],
          },
        );
      },
    );
  }

  /**
   * The ARN of the primary DSQL cluster.
   */
  public get primaryClusterArn() {
    return this.peeringProvider.apply((p) => p.primaryClusterArn);
  }

  /**
   * The ARN of the peer DSQL cluster.
   */
  public get peerClusterArn() {
    return this.peeringProvider.apply((p) => p.peerClusterArn);
  }

  /**
   * The witness region used for the peering.
   */
  public get witnessRegion() {
    return this.peeringProvider.apply((p) => p.witnessRegion);
  }

  /**
   * The unique identifier for this peering connection.
   */
  public get peeringId() {
    return this.peeringProvider.apply((p) => p.peeringId);
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The DSQL peering provider resource.
       */
      peeringProvider: this.peeringProvider,
    };
  }
}

const __pulumiType = "sst:aws:DsqlPeering";
// @ts-expect-error
DsqlPeering.__pulumiType = __pulumiType;
