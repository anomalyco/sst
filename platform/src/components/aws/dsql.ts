import { ComponentResourceOptions, Output, all, output } from "@pulumi/pulumi";
import { Component, Transform } from "../component";
import { Link } from "../link";
import type { Input } from "../input";
import { Region, Provider, dsql } from "@pulumi/aws";
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
   * Multi-region properties for the DSQL Cluster.
   *
   * Setting this creates a multi-region cluster with active-active replication.
   * Multi-region clusters provide 99.999% availability and allow concurrent
   * reads/writes from multiple regions with strong consistency.
   *
   * @example
   * ```js
   * {
   *   multiRegion: {
   *     witnessRegion: "us-west-2"
   *   }
   * }
   * ```
   */
  multiRegion?: Input<{
    /**
     * Witness region for multi-region clusters.
     *
     * Must be in the same region set as the primary and peer region.
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
    witnessRegion: Input<string>;

    /**
     * Peer cluster region to for multi-region clusters.
     * Must be in the same Region group as the primary and witness region.
     * See https://docs.aws.amazon.com/aurora-dsql/latest/userguide/what-is-aurora-dsql.html#region-availability
     *
     * @example
     * ```js
     * {
     *   region: "us-east-2"
     * }
     * ```
     */
    peerRegion: Input<Region>;

    /**
     * The ARN of the AWS KMS key for encryption on the peer cluster.
     * Must be in the same region as the peer cluster.
     *
     * @default `"AWS_OWNED_KMS_KEY"`
     * @example
     * ```js
     * {
     *   kmsEncryptionKey: "arn:aws:kms:us-east-2:123456789012:key/12345678-1234-1234-1234-123456789012"
     * }
     * ```
     */
    peerKmsEncryptionKey?: Input<string>;
  }>;

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
  clusters?: Input<dsql.Cluster[]>;
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
 * #### Create a multi-region cluster
 *
 * Assuming default region is `us-east-1`
 *
 * ```ts title="sst.config.ts"
 * const cluster = new sst.aws.Dsql("MyCluster", {
 *   multiRegion: {
 *     witnessRegion: "us-west-2"
 *     peerRegion: "us-east-2"
 *   }
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
  private clusters: Output<Array<dsql.Cluster>>;

  constructor(
    name: string,
    args: DsqlArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    if (args.ref && args.clusters) {
      this.clusters = output(args.clusters);
      return;
    }

    const clusters = createClusters();
    this.clusters = clusters;

    function createClusters() {
      return all([
        args.deletionProtection,
        args.kmsEncryptionKey,
        args.multiRegion,
        args.tags,
      ]).apply(([deletionProtection, kmsEncryptionKey, multiRegion, tags]) => {
        let clusters = new Array<dsql.Cluster>();

        clusters.push(
          new dsql.Cluster(`${name}Cluster`, {
            deletionProtectionEnabled: deletionProtection ?? false,
            kmsEncryptionKey: kmsEncryptionKey ?? "AWS_OWNED_KMS_KEY",
            multiRegionProperties: multiRegion
              ? {
                  witnessRegion: multiRegion.witnessRegion,
                }
              : undefined,
            tags: {
              Name: `${$app.name}-${$app.stage}-${name}`,
              ...(tags ?? {}),
            },
          }),
        );

        if (multiRegion) {
          const peerRegion = new Provider("peerRegion", {
            region: multiRegion.peerRegion,
          });

          clusters.push(
            new dsql.Cluster(
              `${name}PeerCluster`,
              {
                deletionProtectionEnabled: deletionProtection ?? false,
                kmsEncryptionKey:
                  multiRegion.peerKmsEncryptionKey ?? "AWS_OWNED_KMS_KEY",
                multiRegionProperties: {
                  witnessRegion: multiRegion.witnessRegion,
                },

                tags: {
                  Name: `${$app.name}-${$app.stage}-${name}`,
                  ...(tags ?? {}),
                },
              },
              { provider: peerRegion },
            ),
          );

          const cluster1Peering = new dsql.ClusterPeering(
            `${name}PeeringSide1`,
            {
              identifier: clusters[0].identifier,
              clusters: [clusters[1].arn],
              witnessRegion: multiRegion.witnessRegion,
            },
          );

          const cluster2Peering = new dsql.ClusterPeering(
            `${name}PeeringSide2`,
            {
              identifier: clusters[1].identifier,
              clusters: [clusters[0].arn],
              witnessRegion: multiRegion.witnessRegion,
            },
            { provider: peerRegion },
          );
        }

        return clusters;
      });
    }
  }

  private static identifierFromArn(clusterArn: Output<string>) {
    return clusterArn.apply((arn) => {
      const parts = arn.split(":");
      const clusterId = parts[5].split("/")[1];
      return clusterId;
    });
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
    return this.clusters[0].arn;
  }

  /**
   * The ARN of the peer DSQL Cluster.
   */
  public get peerArn() {
    return this.clusters.apply((c) => {
      if (c.length > 1) {
        return c[1].arn;
      }
    });
  }

  /**
   * The identifier of the DSQL Cluster.
   */
  public get identifier() {
    return this.clusters[0].identifier;
  }

  /**
   * The identifier of the peer DSQL Cluster.
   */
  public get peerIdentifier() {
    return this.clusters.apply((c) => {
      if (c.length > 1) {
        return c[1].identifier;
      }
    });
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
    return this.clusters[0].apply((c) => Dsql.publicEndpointFromArn(c.arn));
  }

  /**
   * The public endpoint of the peer DSQL Cluster for PostgreSQL connections.
   *
   * This returns the full cluster endpoint in the format:
   * {identifier}.dsql.{region}.on.aws
   *
   * Use this endpoint for direct PostgreSQL connections from your applications.
   */
  public get peerPublicEndpoint() {
    return this.clusters.apply((c) => {
      if (c.length > 1) {
        return Dsql.publicEndpointFromArn(c[1].arn);
      }
    });
  }

  /**
   * The region of the DSQL Cluster.
   */
  public get region() {
    return this.clusters[0].apply((c) => Dsql.regionFromArn(c.arn));
  }

  /**
   * The region of the peer DSQL Cluster.
   */
  public get peerRegion() {
    return this.clusters.apply((c) => {
      if (c.length > 1) {
        return Dsql.regionFromArn(c[1].arn);
      }
    });
  }

  /**
   * The VPC endpoint service name for the DSQL Cluster.
   *
   * This is used for creating VPC endpoints for private connectivity.
   * Use this when you want to connect to DSQL through a VPC endpoint
   * instead of over the public internet.
   */
  public get vpcEndpointServiceName() {
    return this.clusters[0].apply((c) => c.vpcEndpointServiceName);
  }

  /**
   * The VPC endpoint service name for the peer DSQL Cluster.
   *
   * This is used for creating VPC endpoints for private connectivity.
   * Use this when you want to connect to DSQL through a VPC endpoint
   * instead of over the public internet.
   */
  public get peerVpcEndpointServiceName() {
    return this.clusters.apply((c) => {
      if (c.length > 1) {
        return c[1].vpcEndpointServiceName;
      }
    });
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Amazon Aurora DSQL Cluster(s).
       */
      cluster: this.clusters[0],
      peerCluster: this.clusters[1] || null,
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
    const cluster1 = dsql.Cluster.get(
      `${name}Cluster`,
      identifier,
      undefined,
      opts,
    );

    const clusters = all([cluster1.multiRegionProperties, cluster1.arn]).apply(
      ([multiRegionProps, cluster1Arn]) => {
        const result: dsql.Cluster[] = [cluster1];
        if ((multiRegionProps?.clusters?.length ?? 0) > 0 && multiRegionProps) {
          const peerArn = multiRegionProps.clusters.find(
            (arn) => arn !== cluster1Arn,
          );
          if (peerArn) {
            const peerIdentifier = Dsql.identifierFromArn(output(peerArn));
            const cluster2 = dsql.Cluster.get(
              `${name}PeerCluster`,
              peerIdentifier,
              undefined,
              opts,
            );
            result.push(cluster2);
          }
        }
        return result;
      },
    );

    return new Dsql(
      name,
      {
        ref: true,
        clusters: clusters,
      },
      opts,
    );
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        arn: this.arn,
        peerArn: this.peerArn,
        identifier: this.identifier,
        peerIdentifier: this.peerIdentifier,
        publicEndpoint: this.publicEndpoint,
        peerPublicEndpoint: this.peerPublicEndpoint,
        vpcEndpointServiceName: this.vpcEndpointServiceName,
        peerVpcEndpointServiceName: this.peerVpcEndpointServiceName,
        region: this.region,
        peerRegion: this.peerRegion,
      },
      include: [
        permission({
          actions: ["dsql:DbConnect", "dsql:DbConnectAdmin", "dsql:GetCluster"],
          resources: all([this.peerArn, this.arn]).apply(([peerArn, arn]) =>
            peerArn ? [arn, peerArn] : [arn],
          ),
        }),
      ],
    };
  }
}

const __pulumiType = "sst:aws:Dsql";
// @ts-expect-error
Dsql.__pulumiType = __pulumiType;
