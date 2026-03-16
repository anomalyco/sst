import { all, ComponentResourceOptions, Output } from "@pulumi/pulumi";
import { Component, transform, Transform } from "../component";
import { Link } from "../link";
import { dsql, ec2, Region } from "@pulumi/aws";
import { permission } from "./permission";
import { useProvider } from "./helpers/provider";
import { Vpc } from "./vpc";
import {
  parseDsqlPublicEndpoint,
  parseDsqlPrivateEndpoint,
} from "./helpers/arn";
import type { Input } from "../input";
import { VisibleError } from "../error";

export interface DsqlArgs {
  /**
   * Configure multi-region cluster peering.
   *
   * Creates a cluster in the current region and a peer cluster in another region,
   * linked via a witness region. The witness must differ from both cluster regions.
   *
   * Learn more about [AWS DSQL regions](https://docs.aws.amazon.com/aurora-dsql/latest/userguide/what-is-aurora-dsql.html#region-availability).
   *
   * @example
   *
   * ```ts
 * const cluster = new sst.aws.Dsql("MyCluster", {
 *   regions: {
 *     witness: "us-west-2",
 *     peer: "us-east-2"
 *   }
 * });
 * ```
 */
  regions?: {
    /** The witness region. Must differ from both cluster regions. */
    witness: Input<string>;
    /** The AWS region for the peer cluster. */
    peer: Input<string>;
  };

  /**
   *
   * Create AWS PrivateLink interface endpoints in a VPC for private connectivity.
   * This allows lambdas placed inside a VPC without NAT gateways to connect to the DSQL instance.
   *
   * :::note
   * Currently only single region VPC is supported.
   * :::
   *
   * @example
   *
   * ```ts title="sst.config.ts"
   * const myVpc = new sst.aws.Vpc("MyVpc");
   *
   * const cluster = new sst.aws.Dsql("MyCluster", {
   *   vpc: myVpc
   * });
   * ```
   *
   * #### Customize VPC endpoints
   *
   * ```ts title="sst.config.ts"
   * const myVpc = new sst.aws.Vpc("MyVpc");
   *
   * const cluster = new sst.aws.Dsql("MyCluster", {
   *   vpc: {
   *     instance: vpc,
   *     endpoints: {
   *       management: true,
   *       connection: true,
   *     }
   *   }
   * });
   * ```
   */
  vpc?:
    | Vpc
    | {
        instance: Vpc;
        endpoints?: {
          /**
           * Endpoint for control plane ops (create, get, update, delete clusters).
           *
           * @default `false`
           */
          management?: boolean;
          /**
           * Endpoint for PostgreSQL client connections.
           *
           * @default `true`
           */
          connection?: boolean;
        };
      };

  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the DSQL cluster resource.
     */
    cluster?: Transform<dsql.ClusterArgs>;
    /**
     * Transform the peer DSQL cluster resource.
     */
    peerCluster?: Transform<dsql.ClusterArgs>;
    /**
     * Transform the EC2 security group resource for the DSQL VPC endpoints.
     */
    endpointSecurityGroup?: Transform<ec2.SecurityGroupArgs>;
    /**
     * Transform the EC2 VPC endpoint resource for DSQL management operations.
     */
    managementEndpoint?: Transform<ec2.VpcEndpointArgs>;
    /**
     * Transform the EC2 VPC endpoint resource for DSQL connections.
     */
    connectionEndpoint?: Transform<ec2.VpcEndpointArgs>;
  };
}

interface DsqlRef {
  ref: boolean;
  cluster: dsql.Cluster;
  peerCluster?: dsql.Cluster;
}

/**
 * The `Dsql` component lets you add an [Amazon Aurora DSQL](https://aws.amazon.com/rds/aurora/dsql/) cluster to your app.
 *
 * @example
 *
 * #### Single-region cluster
 *
 * ```ts title="sst.config.ts"
 * const cluster = new sst.aws.Dsql("MyCluster");
 * ```
 *
 * Once linked, you can connect to it from your function code.
 *
 * ```ts title="src/lambda.ts"
 * import { Resource } from "sst";
 * import { AuroraDSQLClient } from "@aws/aurora-dsql-node-postgres-connector";
 *
 * const client = new AuroraDSQLClient({
 *   host: Resource.MyCluster.endpoint,
 *   user: "admin",
 * });
 *
 * await client.connect();
 * const result = await client.query("SELECT NOW() as now");
 * await client.end();
 * ```
 *
 * #### Multi-region cluster
 *
 * ```ts title="sst.config.ts"
 * const cluster = new sst.aws.Dsql("MyCluster", {
 *   regions: {
 *     witness: "us-west-2",
 *     peer: "us-east-2"
 *   }
 * });
 * ```
 *
 * [Check out the full example](/docs/examples/#aws-dsql-multiregion).
 *
 * #### With private VPC endpoints
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 *
 * const cluster = new sst.aws.Dsql("MyCluster", {
 *   vpc: {
 *     instance: vpc,
 *     endpoints: { connection: true }
 *   }
 * });
 * ```
 *
 * [Check out the full example](/docs/examples/#aws-dsql-vpc).
 *
 * #### Link to a function
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("MyFunction", {
 *   handler: "src/lambda.handler",
 *   link: [cluster]
 * });
 * ```
 *
 * You can also use Drizzle ORM to query your DSQL cluster.
 * [Check out the Drizzle example](/docs/examples/#aws-dsql-drizzle).
 *
 * ---
 *
 * ### Cost
 *
 * Aurora DSQL is serverless and uses a pay-per-use pricing model. You are charged for
 * database activity measured in _Distributed Processing Units_ (DPUs) at $8 per million
 * DPUs, and storage at $0.33 per GB-month. When idle, usage scales to zero and you incur
 * no DPU charges.
 *
 * There is a free tier of 100,000 DPUs and 1 GB of storage per month.
 *
 * For example, a single-region cluster averaging 1.3M DPUs per month with 15 GB of storage
 * costs roughly 1.3 x $8 + 15 x $0.33 or **$15 per month**.
 *
 * Check out the [Aurora DSQL pricing](https://aws.amazon.com/rds/aurora/dsql/pricing/) for more details.
 *
 */

export class Dsql extends Component implements Link.Linkable {
  private cluster: dsql.Cluster;
  private peerCluster: dsql.Cluster | undefined;
  private connectionEndpoint: ec2.VpcEndpoint | undefined;
  private constructorName: string;

  constructor(
    name: string,
    args: DsqlArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);
    this.constructorName = name;

    if (args && "ref" in args) {
      const ref = args as unknown as DsqlRef;
      this.cluster = ref.cluster;
      this.peerCluster = ref.peerCluster;
      return;
    }

    const parent = this;
    const regions = args.regions;

    if (regions && args.vpc)
      throw new VisibleError(
        `Cannot use "vpc" with multi-region "regions". VPC endpoints are only supported for single-region clusters.`,
      );

    const vpc = normalizeVpc();

    const cluster = createCluster();
    const peerCluster = createPeerCluster();
    const endpoints = createVpcEndpoints();

    this.cluster = cluster;
    this.peerCluster = peerCluster;
    this.connectionEndpoint = endpoints?.connection;

    function createCluster() {
      return new dsql.Cluster(
        ...transform(
          args.transform?.cluster,
          `${name}Cluster`,
          {
            multiRegionProperties: regions
              ? { witnessRegion: regions.witness }
              : undefined,
          },
          { parent },
        ),
      );
    }

    function createPeerCluster() {
      if (!regions) return;

      const peerProvider = useProvider(regions.peer as Region);

      const peerCluster = new dsql.Cluster(
        ...transform(
          args.transform?.peerCluster,
          `${name}PeerCluster`,
          {
            multiRegionProperties: {
              witnessRegion: regions.witness,
            },
          },
          { parent, provider: peerProvider },
        ),
      );

      // DSQL requires both clusters to declare each other — two-way handshake.
      new dsql.ClusterPeering(
        `${name}Peering1`,
        {
          identifier: cluster.identifier,
          clusters: [peerCluster.arn],
          witnessRegion: regions.witness,
        },
        { parent },
      );

      new dsql.ClusterPeering(
        `${name}Peering2`,
        {
          identifier: peerCluster.identifier,
          clusters: [cluster.arn],
          witnessRegion: regions.witness,
        },
        { parent, provider: peerProvider },
      );

      return peerCluster;
    }

    function normalizeVpc() {
      if (!args.vpc) return undefined;

      if (args.vpc instanceof Vpc) {
        return {
          instance: args.vpc,
          endpoints: {
            management: false,
            connection: true,
          },
        };
      }

      return {
        instance: args.vpc.instance,
        endpoints: {
          management: args.vpc.endpoints?.management ?? false,
          connection: args.vpc.endpoints?.connection ?? true,
        },
      };
    }

    function createVpcEndpoints() {
      if (!vpc) return;

      const endpointSecurityGroup = new ec2.SecurityGroup(
        ...transform(
          args.transform?.endpointSecurityGroup,
          `${name}DsqlEndpointSecurityGroup`,
          {
            vpcId: vpc.instance.id,
            description: "Allow DSQL access to VPC endpoints",
            ingress: [
              ...(vpc.endpoints.management
                ? [
                    {
                      protocol: "tcp",
                      fromPort: 443,
                      toPort: 443,
                      cidrBlocks: [vpc.instance.nodes.vpc.cidrBlock],
                    },
                  ]
                : []),
              ...(vpc.endpoints.connection
                ? [
                    {
                      protocol: "tcp",
                      fromPort: 5432,
                      toPort: 5432,
                      cidrBlocks: [vpc.instance.nodes.vpc.cidrBlock],
                    },
                  ]
                : []),
            ],
            egress: [
              {
                protocol: "-1",
                fromPort: 0,
                toPort: 0,
                cidrBlocks: ["0.0.0.0/0"],
              },
            ],
          },
          { parent },
        ),
      );

      let management, connection;

      if (vpc.endpoints.management) {
        management = new ec2.VpcEndpoint(
          ...transform(
            args.transform?.managementEndpoint,
            `${name}ManagementEndpoint`,
            {
              vpcId: vpc.instance.id,
              serviceName: cluster.arn.apply((arn) => {
                const region = arn.split(":")[3];
                return `com.amazonaws.${region}.dsql`;
              }),
              vpcEndpointType: "Interface",
              subnetIds: vpc.instance.privateSubnets,
              privateDnsEnabled: true,
              securityGroupIds: [endpointSecurityGroup.id],
            },
            { parent },
          ),
        );
      }

      if (vpc.endpoints.connection) {
        connection = new ec2.VpcEndpoint(
          ...transform(
            args.transform?.connectionEndpoint,
            `${name}ConnectionEndpoint`,
            {
              vpcId: vpc.instance.id,
              serviceName: cluster.vpcEndpointServiceName,
              vpcEndpointType: "Interface",
              subnetIds: vpc.instance.privateSubnets,
              privateDnsEnabled: true,
              securityGroupIds: [endpointSecurityGroup.id],
            },
            { parent },
          ),
        );
      }

      return { connection, management };
    }
  }

  /** The region of the cluster. */
  public get region() {
    return this.cluster.region;
  }

  /** The endpoint of the cluster. */
  public get endpoint() {
    // Use the private VPC endpoint hostname when available so linked functions
    // inside the VPC don't route through the public IP.
    return all([this.cluster.arn, this.connectionEndpoint?.dnsEntries]).apply(
      ([arn, dns]) => {
        if (!dns) {
          return parseDsqlPublicEndpoint(arn);
        }
        return parseDsqlPrivateEndpoint(arn, dns);
      },
    );
  }

  /**
   * The peer cluster info. Only available for multi-region clusters.
   *
   * @example
   * ```ts title="sst.config.ts"
   * const cluster = new sst.aws.Dsql("MyCluster", {
   *   regions: { peer: "us-east-2" },
   * });
   *
   * return {
   *   peerRegion: cluster.peer.region,
   *   peerEndpoint: cluster.peer.endpoint,
   * };
   * ```
   */
  public get peer() {
    if (!this.peerCluster)
      throw new VisibleError(
        `Cannot access "peer" on "${this.constructorName}" because it is a single-region cluster. Set "regions.peer" to enable multi-region.`,
      );
    const peerCluster = this.peerCluster;
    return {
      /** The region of the peer cluster. */
      region: peerCluster.region,
      /** The endpoint of the peer cluster. */
      endpoint: peerCluster.arn.apply(parseDsqlPublicEndpoint),
    };
  }

  /** The underlying [resources](/docs/components/#nodes) this component creates. */
  public get nodes() {
    return {
      /** The DSQL cluster. */
      cluster: this.cluster,
      /** The peer DSQL cluster (multi-region only). */
      peerCluster: this.peerCluster,
    };
  }

  /**
   * Reference an existing DSQL cluster by identifier. Useful for sharing a cluster
   * across stages without creating a new one.
   *
   * :::tip
   * You can use the `static get` method to share a cluster across stages.
   * :::
   *
   * @example
   *
   * #### Single-region cluster
   *
   * ```ts title="sst.config.ts"
   * const cluster = $app.stage === "frank"
   *   ? sst.aws.Dsql.get("MyCluster", { id: "kzttrvbdg4k2o5ze2m2rrwdj7u" })
   *   : new sst.aws.Dsql("MyCluster");
   * ```
   * #### Multi-region cluster
   *
   * ```ts title="sst.config.ts"
   * const cluster = sst.aws.Dsql.get("MyCluster", {
   *   id: "app-dev-mycluster",
   *   peer: {
   *     id: "kzttrvbdg4k2o5ze2m2rrwdj7u",
   *     region: "us-east-2",
   *   }
   * });
   * ```
   */
  public static get(
    name: string,
    args: {
      id: Input<string>;
      peer?: {
        id: string;
        region: string;
      };
    },
    opts?: ComponentResourceOptions,
  ) {
    const cluster = dsql.Cluster.get(
      `${name}Cluster`,
      args.id,
      undefined,
      opts,
    );

    const peerCluster = args.peer
      ? dsql.Cluster.get(`${name}PeerCluster`, args.peer.id, undefined, {
          ...opts,
          provider: useProvider(args.peer.region as Region),
        })
      : undefined;

    return new Dsql(
      name,
      { ref: true, cluster, peerCluster } satisfies DsqlRef as unknown as DsqlArgs,
      opts,
    );
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        region: this.region,
        endpoint: this.endpoint,
        peer: this.peerCluster
          ? {
              region: this.peerCluster.region,
              endpoint: this.peerCluster.arn.apply(parseDsqlPublicEndpoint),
            }
          : undefined,
      },
      include: [
        permission({
          actions: ["dsql:DbConnect", "dsql:DbConnectAdmin", "dsql:GetCluster"],
          resources: this.peerCluster
            ? [this.cluster.arn, this.peerCluster.arn]
            : [this.cluster.arn],
        }),
      ],
    };
  }
}

const __pulumiType = "sst:aws:Dsql";
// @ts-expect-error
Dsql.__pulumiType = __pulumiType;
