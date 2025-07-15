import { ComponentResourceOptions, Output } from "@pulumi/pulumi";
import { Component, Transform } from "../component";
import { Link } from "../link";
import type { Input } from "../input";
import { dsql } from "@pulumi/aws";
import { permission } from "./permission";

export interface DsqlArgs {
  /**
   * Enable deletion protection for the cluster.
   *
   * When enabled, the cluster cannot be deleted.
   *
   * @default `false`
   * @example
   * ```js
   * {
   *   deletionProtection: true
   * }
   * ```
   */
  deletionProtection?: Input<boolean>;

  /**
   * The ARN of the AWS KMS key for encryption.
   *
   * @default `"AWS_OWNED_KMS_KEY"`
   * @example
   * ```js
   * {
   *   kmsEncryptionKey: "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
   * }
   * ```
   */
  kmsEncryptionKey?: Input<string>;

  /**
   * Witness region for multi-region cluster preparation.
   *
   * When specified, creates a cluster ready for multi-region peering.
   * Must be in the same region set as the cluster region.
   * See https://docs.aws.amazon.com/aurora-dsql/latest/userguide/what-is-aurora-dsql.html#region-availability
   *
   * @example
   * ```js
   * {
   *   witnessRegion: "us-west-2"
   * }
   * ```
   */
  witnessRegion?: Input<string>;

  /**
   * Tags to associate with the cluster.
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

  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the DSQL Cluster resource.
     */
    cluster?: Transform<dsql.ClusterArgs>;
  };
  /**
   * @internal
   */
  ref?: boolean;
  /**
   * @internal
   */
  cluster?: dsql.Cluster;
}

/**
 * The `Dsql` component lets you add an [Amazon Aurora DSQL](https://aws.amazon.com/rds/aurora/dsql/) cluster to your app.
 *
 * Aurora DSQL is a serverless, distributed relational database optimized for transactional workloads.
 * It provides PostgreSQL compatibility with virtually unlimited scale and active-active high availability.
 *
 * @example
 *
 * #### Create a single-region cluster
 *
 * ```ts title="sst.config.ts"
 * const cluster = new sst.aws.Dsql("MyCluster");
 * ```
 *
 * #### Create clusters in multiple regions
 *
 * For multi-region setup, create separate clusters and use DsqlPeering:
 *
 * ```ts title="sst.config.ts"
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
 * const peering = new sst.aws.DsqlPeering("Peering", {
 *   primaryCluster: primary,
 *   peerCluster: peer,
 *   witnessRegion: witnessRegion
 * });
 * ```
 *
 * #### Link to a function
 *
 * You can link the cluster to other resources, like a function.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("MyFunction", {
 *   handler: "src/lambda.handler",
 *   link: [cluster]
 * });
 * ```
 *
 * Once linked, you can connect to it from your function code using the AWS SDK.
 *
 * ```ts title="src/lambda.ts"
 * import { Resource } from "sst";
 * import { DsqlSigner } from "@aws-sdk/dsql-signer";
 * import { Client } from "pg";
 *
 * const signer = new DsqlSigner({
 *   region: Resource.MyCluster.region,
 *   hostname: Resource.MyCluster.publicEndpoint,
 * });
 *
 * const token = await signer.getDbConnectAdminAuthToken();
 *
 * const client = new Client({
 *   host: Resource.MyCluster.publicEndpoint,
 *   port: 5432,
 *   database: "postgres",
 *   user: "admin",
 *   password: token,
 *   ssl: true
 * });
 * ```
 */

export class Dsql extends Component implements Link.Linkable {
  private cluster: dsql.Cluster;

  constructor(
    name: string,
    args: DsqlArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    if (args.ref && args.cluster) {
      this.cluster = args.cluster;
      return;
    }

    this.cluster = new dsql.Cluster(
      `${name}Cluster`,
      {
        deletionProtectionEnabled: args.deletionProtection ?? false,
        kmsEncryptionKey: args.kmsEncryptionKey ?? "AWS_OWNED_KMS_KEY",
        multiRegionProperties: args.witnessRegion
          ? {
              witnessRegion: args.witnessRegion,
            }
          : undefined,
        tags: {
          Name: `${$app.name}-${$app.stage}-${name}`,
          ...(args.tags ?? {}),
        },
        ...(args.transform?.cluster || {}),
      },
      {
        parent: this,
      },
    );
  }

  private static publicEndpointFromArn(clusterArn: Output<string>) {
    return clusterArn.apply((arn) => {
      const parts = arn.split(":");
      const region = parts[3];
      const clusterId = parts[5].split("/")[1];
      return `${clusterId}.dsql.${region}.on.aws`;
    });
  }

  private static regionFromArn(clusterArn: Output<string>) {
    return clusterArn.apply((arn) => {
      const parts = arn.split(":");
      const region = parts[3];
      return region;
    });
  }

  /**
   * The ARN of the DSQL Cluster.
   */
  public get arn() {
    return this.cluster.arn;
  }

  /**
   * The identifier of the DSQL Cluster.
   */
  public get identifier() {
    return this.cluster.identifier;
  }

  /**
   * The public endpoint of the DSQL Cluster for PostgreSQL connections.
   *
   * This returns the full cluster endpoint in the format:
   * {identifier}.dsql.{region}.on.aws
   *
   * Use this endpoint for direct PostgreSQL connections from your applications.
   */
  public get publicEndpoint() {
    return Dsql.publicEndpointFromArn(this.cluster.arn);
  }

  /**
   * The region of the DSQL Cluster.
   */
  public get region() {
    return Dsql.regionFromArn(this.cluster.arn);
  }

  /**
   * The VPC endpoint service name for the DSQL Cluster.
   *
   * This is used for creating VPC endpoints for private connectivity.
   * Use this when you want to connect to DSQL through a VPC endpoint
   * instead of over the public internet.
   */
  public get vpcEndpointServiceName() {
    return this.cluster.vpcEndpointServiceName;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Amazon Aurora DSQL Cluster.
       */
      cluster: this.cluster,
    };
  }

  /**
   * Reference an existing DSQL Cluster with the given cluster identifier. This is useful when you
   * create a cluster in one stage and want to share it in another stage. It avoids having to
   * create a new cluster in the other stage.
   *
   * :::tip
   * You can use the `static get` method to share a cluster across stages.
   * :::
   *
   * @param name The name of the component.
   * @param identifier The identifier of the DSQL Cluster.
   * @param opts? Resource options.
   *
   * @example
   * Imagine you create a cluster in the `dev` stage. And in your personal stage `frank`,
   * instead of creating a new cluster, you want to share the cluster from `dev`.
   *
   * ```ts title="sst.config.ts"
   * const cluster = $app.stage === "frank"
   *   ? sst.aws.Dsql.get("MyCluster", "app-dev-mycluster")
   *   : new sst.aws.Dsql("MyCluster");
   * ```
   *
   * Here `app-dev-mycluster` is the identifier of the DSQL Cluster created in the `dev` stage.
   * You can find this by outputting the cluster identifier in the `dev` stage.
   *
   * ```ts title="sst.config.ts"
   * return {
   *   cluster: cluster.identifier
   * };
   * ```
   */
  public static get(
    name: string,
    identifier: Input<string>,
    opts?: ComponentResourceOptions,
  ) {
    const cluster = dsql.Cluster.get(
      `${name}Cluster`,
      identifier,
      undefined,
      opts,
    );

    return new Dsql(
      name,
      {
        ref: true,
        cluster: cluster,
      },
      opts,
    );
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        arn: this.arn,
        identifier: this.identifier,
        publicEndpoint: this.publicEndpoint,
        vpcEndpointServiceName: this.vpcEndpointServiceName,
        region: this.region,
      },
      include: [
        permission({
          actions: ["dsql:DbConnect", "dsql:DbConnectAdmin", "dsql:GetCluster"],
          resources: [this.arn],
        }),
      ],
    };
  }
}

const __pulumiType = "sst:aws:Dsql";
// @ts-expect-error
Dsql.__pulumiType = __pulumiType;
