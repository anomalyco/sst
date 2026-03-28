import { ComponentResourceOptions, Output } from "@pulumi/pulumi";
import { Component, transform, Transform } from "../component";
import { Link } from "../link";
import { dsql } from "@pulumi/aws";
import { permission } from "./permission";
import { useProvider } from "./helpers/provider";
import { Vpc } from "./vpc";

import type { Input } from "../input";

import * as aws from "@pulumi/aws";

export interface DsqlArgs {
  /**
   * Configure multi-region cluster peering.
   *
   * Creates a primary cluster in the current region and a peer cluster in another region,
   * linked via a witness region. The witness must differ from both cluster regions.
   *
   * See https://docs.aws.amazon.com/aurora-dsql/latest/userguide/what-is-aurora-dsql.html#region-availability
   *
   * @example
   * ```ts
   * const cluster = new sst.aws.Dsql("MyCluster", {
   * regions: {
   * witness: "us-west-2",
   * peer: { region: "us-east-2" }
   * }
   * });
   * ```
   */
  regions?: {
    /** The witness region. Must differ from both cluster regions. */
    witness: Input<string>;
    peer: {
      /** The AWS region for the peer cluster. */
      region: Input<string>;
    };
  };

  /**
   * Enable automatic backups for the cluster using AWS Backup.
   * Retains daily backups for 35 days. If multi-region is enabled, identical backup resources
   * are created in both regions.
   * * Set to `false` to explicitly disable backups.
   *
   * @default true
   * @example
   * ```ts title="sst.config.ts"
   * const cluster = new sst.aws.Dsql("MyCluster", {
   * backup: false
   * });
   * ```
   */
  backup?: boolean;

  /**
   * :::note
   * Currently only single region VPC is supported. Multi region coming soon.
   * :::
   *
   * Create AWS PrivateLink interface endpoints in a VPC for private connectivity.
   *
   * Two endpoint types are supported:
   * - **Management** — control plane ops (create, get, update, delete clusters).
   * Service: `com.amazonaws.{region}.dsql`
   * - **Connection** — PostgreSQL client connections.
   * Service name is cluster-specific and resolved automatically.
   *
   * :::note
   * The VPC endpoint security group allows inbound on port 5432 from within the VPC CIDR.
   * Your Lambda must be in the same VPC. The Lambda's default security group allows all
   * outbound traffic, so it can reach the endpoint on port 5432 without any extra config.
   * :::
   *
   *
   * @example
   * ```ts title="sst.config.ts"
   * const vpc = new sst.aws.Vpc("MyVpc");
   *
   * const cluster = new sst.aws.Dsql("MyCluster", {
   * vpc: {
   * instance: vpc,
   * endpoints: {
   * management: true,
   * connection: true,
   * }
   * }
   * });
   * ```
   */
  vpc?:
    | Vpc
    | {
        instance: Vpc;
        endpoints?: {
          /** @default `false` */
          management?: Input<boolean>;
          /** @default `false` */
          connection?: Input<boolean>;
        };
      };

  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    cluster?: Transform<dsql.ClusterArgs>;
    peerCluster?: Transform<dsql.ClusterArgs>;
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
 * #### Multi-region cluster
 *
 * ```ts title="sst.config.ts"
 * const cluster = new sst.aws.Dsql("MyCluster", {
 * regions: {
 * witness: "us-west-2",
 * peer: { region: "us-east-2" }
 * }
 * });
 * ```
 *
 * #### With private VPC endpoints
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 *
 * const cluster = new sst.aws.Dsql("MyCluster", {
 * vpc: {
 * instance: vpc,
 * endpoints: { connection: true }
 * }
 * });
 * ```
 *
 * #### Link to a function
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Function("MyFunction", {
 * handler: "src/lambda.handler",
 * link: [cluster]
 * });
 * ```
 *
 */

export class Dsql extends Component implements Link.Linkable {
  private cluster: dsql.Cluster;
  private peerCluster: dsql.Cluster | undefined;
  private connectionEndpoint: aws.ec2.VpcEndpoint | undefined;

  constructor(
    name: string,
    args: DsqlArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    if (args && "ref" in args) {
      const ref = args as unknown as DsqlRef;
      this.cluster = ref.cluster;
      this.peerCluster = ref.peerCluster;
      return;
    }

    const parent = this;
    const multiRegion = args.regions;

    const peerProvider = multiRegion
      ? useProvider(multiRegion.peer.region as aws.Region)
      : undefined;

    const primaryCluster = createPrimaryCluster();
    this.cluster = primaryCluster;

    const createdPeerCluster = multiRegion ? createPeerCluster() : undefined;
    this.peerCluster = createdPeerCluster;

    if (multiRegion && createdPeerCluster && peerProvider) {
      createPeerings(createdPeerCluster, peerProvider);
    }

    const backupEnabled = args.backup !== false;

    if (backupEnabled) {
      createBackupSetup(primaryCluster, undefined, "");

      if (multiRegion && createdPeerCluster && peerProvider) {
        createBackupSetup(createdPeerCluster, peerProvider, "Peer");
      }
    }

    function createBackupSetup(
      targetCluster: dsql.Cluster,
      provider?: aws.Provider,
      suffix: string = "",
    ) {
      const role = new aws.iam.Role(
        `${name}BackupRole${suffix}`,
        {
          assumeRolePolicy: aws.iam.assumeRolePolicyForPrincipal({
            Service: "backup.amazonaws.com",
          }),
          managedPolicyArns: [
            "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForBackup",
          ],
        },
        { parent, provider },
      );

      const vault = new aws.backup.Vault(
        `${name}BackupVault${suffix}`,
        {},
        { parent, provider },
      );

      const plan = new aws.backup.Plan(
        `${name}BackupPlan${suffix}`,
        {
          rules: [
            {
              ruleName: "Daily",
              targetVaultName: vault.name,
              schedule: "cron(0 12 * * ? *)",
              lifecycle: { deleteAfter: 35 },
            },
          ],
        },
        { parent, provider },
      );

      new aws.backup.Selection(
        `${name}BackupSelection${suffix}`,
        {
          planId: plan.id,
          iamRoleArn: role.arn,
          resources: [targetCluster.arn],
        },
        { parent, provider },
      );
    }

    function createPrimaryCluster() {
      return new dsql.Cluster(
        ...transform(
          args.transform?.cluster,
          `${name}Cluster`,
          {
            multiRegionProperties: multiRegion
              ? { witnessRegion: multiRegion.witness }
              : undefined,
            tags: { Name: name },
          },
          { parent },
        ),
      );
    }

    function createPeerCluster() {
      return new dsql.Cluster(
        ...transform(
          args.transform?.peerCluster,
          `${name}PeerCluster`,
          {
            multiRegionProperties: {
              witnessRegion: multiRegion!.witness,
            },
            tags: { Name: `${name}Peer` },
          },
          { parent, provider: peerProvider },
        ),
      );
    }

    function createPeerings(peer: dsql.Cluster, peerProv: aws.Provider) {
      new dsql.ClusterPeering(
        `${name}PrimaryPeering`,
        {
          identifier: primaryCluster.identifier,
          clusters: [peer.arn],
          witnessRegion: primaryCluster.multiRegionProperties.apply(
            (p) => p?.witnessRegion!,
          ),
        },
        { parent },
      );

      new dsql.ClusterPeering(
        `${name}PeerPeering`,
        {
          identifier: peer.identifier,
          clusters: [primaryCluster.arn],
          witnessRegion: peer.multiRegionProperties.apply(
            (p) => p?.witnessRegion!,
          ),
        },
        { parent, provider: peerProv },
      );
    }

    const vpc = normalizeVpc();

    function normalizeVpc() {
      if (!args.vpc) return undefined;

      if (args.vpc instanceof Vpc) {
        return {
          instance: args.vpc,
          endpoints: {
            management: false as Input<boolean>,
            connection: false as Input<boolean>,
          },
        };
      }

      return {
        instance: args.vpc.instance,
        endpoints: {
          management: args.vpc.endpoints?.management ?? false,
          connection: args.vpc.endpoints?.connection ?? false,
        },
      };
    }

    if (vpc) {
      const managementEnabled = vpc.endpoints?.management ?? false;
      const connectionEnabled = vpc.endpoints?.connection ?? false;

      const endpointSg = new aws.ec2.SecurityGroup(
        `${name}EndpointSg`,
        {
          vpcId: vpc.instance.id,
          description: "Allow PostgreSQL access to DSQL VPC endpoints",
          ingress: [
            {
              protocol: "tcp",
              fromPort: 5432,
              toPort: 5432,
              cidrBlocks: [vpc.instance.nodes.vpc.cidrBlock],
            },
          ],
          egress: [
            {
              protocol: "-1",
              fromPort: 0,
              toPort: 0,
              cidrBlocks: ["0.0.0.0/0"],
            },
          ],
          tags: { Name: `${name}Endpoint` },
        },
        { parent },
      );

      if (managementEnabled) {
        new aws.ec2.VpcEndpoint(
          `${name}ManagementEndpoint`,
          {
            vpcId: vpc.instance.id,
            serviceName: primaryCluster.arn.apply((arn) => {
              const region = arn.split(":")[3];
              return `com.amazonaws.${region}.dsql`;
            }),
            vpcEndpointType: "Interface",
            subnetIds: vpc.instance.privateSubnets,
            privateDnsEnabled: true,
            tags: { Name: `${name}Management` },
            securityGroupIds: [endpointSg.id],
          },
          { parent },
        );
      }

      if (connectionEnabled) {
        this.connectionEndpoint = new aws.ec2.VpcEndpoint(
          `${name}ConnectionEndpoint`,
          {
            vpcId: vpc.instance.id,
            serviceName: primaryCluster.vpcEndpointServiceName,
            vpcEndpointType: "Interface",
            subnetIds: vpc.instance.privateSubnets,
            privateDnsEnabled: true,
            tags: { Name: `${name}Connection` },
            securityGroupIds: [endpointSg.id],
          },
          { parent },
        );
      }
    }
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
    return clusterArn.apply((arn) => arn.split(":")[3]);
  }

  private static privateEndpointFromVpcEndpoint(
    cluster: dsql.Cluster,
    vpcEndpoint: aws.ec2.VpcEndpoint,
  ): Output<string> {
    return cluster.arn.apply((arn) => {
      const clusterId = arn.split(":")[5].split("/")[1];
      return vpcEndpoint.dnsEntries.apply((dnsEntries) => {
        const wildcardEntry = dnsEntries.find(
          (e) => e.dnsName?.startsWith("*."),
        );
        const privateDnsName = wildcardEntry?.dnsName ?? dnsEntries[0]?.dnsName;
        return privateDnsName!.replace("*", clusterId);
      });
    });
  }

  /** The ARN of the primary cluster. */
  public get arn() {
    return this.cluster.arn;
  }

  /** The ARN of the peer cluster (multi-region only). */
  public get peerArn(): Output<string> | undefined {
    return this.peerCluster?.arn;
  }

  /** The identifier of the primary cluster. */
  public get identifier() {
    return this.cluster.identifier;
  }

  /** The identifier of the peer cluster (multi-region only). */
  public get peerIdentifier(): Output<string> | undefined {
    return this.peerCluster?.identifier;
  }

  /**
   * The public endpoint of the primary cluster.
   * Format: `{identifier}.dsql.{region}.on.aws`
   */
  public get publicEndpoint() {
    return Dsql.publicEndpointFromArn(this.cluster.arn);
  }

  /** The public endpoint of the peer cluster (multi-region only). */
  public get peerPublicEndpoint(): Output<string> | undefined {
    return this.peerCluster
      ? Dsql.publicEndpointFromArn(this.peerCluster.arn)
      : undefined;
  }

  /** The region of the primary cluster. */
  public get region() {
    return Dsql.regionFromArn(this.cluster.arn);
  }

  /** The region of the peer cluster (multi-region only). */
  public get peerRegion(): Output<string> | undefined {
    return this.peerCluster
      ? Dsql.regionFromArn(this.peerCluster.arn)
      : undefined;
  }

  public get nodes() {
    return {
      cluster: this.cluster,
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
   * ```ts title="sst.config.ts"
   * const cluster = $app.stage === "frank"
   * ? sst.aws.Dsql.get("MyCluster", "app-dev-mycluster")
   * : new sst.aws.Dsql("MyCluster");
   * ```
   */
  public static get(
    name: string,
    args: {
      identifier: Input<string>;
      peerIdentifier?: Input<string>;
      peerRegion: Input<string>;
    },
    opts?: ComponentResourceOptions,
  ) {
    const cluster = dsql.Cluster.get(
      `${name}Cluster`,
      args.identifier,
      undefined,
      opts,
    );

    let peerCluster: dsql.Cluster | undefined;

    if (args.peerIdentifier && args.peerRegion) {
      const peerProvider = useProvider(args.peerRegion as aws.Region);
      peerCluster = dsql.Cluster.get(
        `${name}PeerCluster`,
        args.peerIdentifier,
        undefined,
        { ...opts, provider: peerProvider },
      );
    }

    return new Dsql(name, { ref: true, cluster, peerCluster } as DsqlArgs);
  }

  /** @internal */
  public getSSTLink() {
    const primaryEndpoint = this.connectionEndpoint
      ? Dsql.privateEndpointFromVpcEndpoint(
          this.cluster,
          this.connectionEndpoint,
        )
      : this.publicEndpoint;

    const peerEndpoint = this.peerCluster
      ? Dsql.publicEndpointFromArn(this.peerCluster.arn)
      : undefined;

    return {
      properties: {
        region: this.region,
        endpoint: primaryEndpoint,
        peer: {
          region: this.peerCluster
            ? Dsql.regionFromArn(this.peerCluster.arn)
            : undefined,
          endpoint: peerEndpoint,
        },
      },
      include: [
        permission({
          actions: ["dsql:DbConnect", "dsql:DbConnectAdmin", "dsql:GetCluster"],
          resources: this.peerCluster
            ? [this.arn, this.peerCluster.arn]
            : [this.arn],
        }),
      ],
    };
  }
}

const __pulumiType = "sst:aws:Dsql";
// @ts-expect-error
Dsql.__pulumiType = __pulumiType;
