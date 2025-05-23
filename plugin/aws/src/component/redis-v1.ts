import * as sst from "sst-plugin";
import { Transform, transform } from "sst-plugin/internal/transform";
import { VisibleError } from "sst-plugin/error";
import { AWSComponent } from "../component.js";
import { elasticache, secretsmanager } from "@pulumi/aws";
import { RandomPassword } from "@pulumi/random";
import { Vpc } from "./vpc.js";

export interface RedisArgs {
  /**
   * The Redis engine to use. The following engines are supported:
   *
   * - `"redis"`: The open-source version of Redis.
   * - `"valkey"`: [Valkey](https://valkey.io/) is a Redis-compatible in-memory key-value store.
   *
   * @default `"redis"`
   */
  engine?: sst.Input<"redis" | "valkey">;
  /**
   * The version of Redis.
   *
   * The default is `"7.1"` for the `"redis"` engine and `"7.2"` for the `"valkey"` engine.
   *
   * Check out the [supported versions](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/supported-engine-versions.html).
   *
   * @default `"7.1"` for Redis, `"7.2"` for Valkey
   * @example
   * ```js
   * {
   *   version: "6.2"
   * }
   * ```
   */
  version?: sst.Input<string>;
  /**
   * The type of instance to use for the nodes of the Redis cluster. Check out the [supported instance types](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/CacheNodes.SupportedTypes.html).
   *
   * @default `"t4g.micro"`
   * @example
   * ```js
   * {
   *   instance: "m7g.xlarge"
   * }
   * ```
   */
  instance?: sst.Input<string>;
  /**
   * The number of nodes to use for the Redis cluster.
   *
   * @default `1`
   * @example
   * ```js
   * {
   *   nodes: 4
   * }
   * ```
   */
  nodes?: sst.Input<number>;
  /**
   * The VPC to use for the Redis cluster.
   *
   * @example
   * Create a VPC component.
   *
   * ```js
   * const myVpc = new sst.aws.Vpc("MyVpc");
   * ```
   *
   * And pass it in.
   *
   * ```js
   * {
   *   vpc: myVpc
   * }
   * ```
   *
   * Or pass in a custom VPC configuration.
   *
   * ```js
   * {
   *   vpc: {
   *     subnets: ["subnet-0db7376a7ad4db5fd ", "subnet-06fc7ee8319b2c0ce"],
   *     securityGroups: ["sg-0399348378a4c256c"]
   *   }
   * }
   * ```
   */
  vpc:
    | Vpc
    | sst.Input<{
        /**
         * A list of subnet IDs in the VPC to deploy the Redis cluster in.
         */
        subnets: sst.Input<sst.Input<string>[]>;
        /**
         * A list of VPC security group IDs.
         */
        securityGroups: sst.Input<sst.Input<string>[]>;
      }>;
  /**
   * Configure how this component works in `sst dev`.
   *
   * By default, your Redis cluster is deployed in `sst dev`. But if you want to instead
   * connect to a locally running Redis server, you can configure the `dev` prop.
   *
   * :::note
   * By default, this creates a new Redis ElastiCache cluster even in `sst dev`.
   * :::
   *
   * This will skip deploying a Redis ElastiCache cluster and link to the locally running Redis
   * server instead.
   *
   * @example
   *
   * Setting the `dev` prop also means that any linked resources will connect to the right
   * Redis instance both in `sst dev` and `sst deploy`.
   *
   * ```ts
   * {
   *   dev: {
   *     host: "localhost",
   *     port: 6379
   *   }
   * }
   * ```
   */
  dev?: {
    /**
     * The host of the local Redis server to connect to when running in dev.
     * @default `"localhost"`
     */
    host?: sst.Input<string>;
    /**
     * The port of the local Redis server when running in dev.
     * @default `6379`
     */
    port?: sst.Input<number>;
    /**
     * The username of the local Redis server to connect to when running in dev.
     * @default `"default"`
     */
    username?: sst.Input<string>;
    /**
     * The password of the local Redis server to connect to when running in dev.
     * @default No password
     */
    password?: sst.Input<string>;
  };
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Redis subnet group.
     */
    subnetGroup?: Transform<elasticache.SubnetGroupArgs>;
    /**
     * Transform the Redis cluster.
     */
    cluster?: Transform<elasticache.ReplicationGroupArgs>;
  };
}

interface RedisRef {
  ref: boolean;
  cluster: elasticache.ReplicationGroup;
  authToken: sst.Output<string>;
}

/**
 * The `Redis` component lets you add a Redis cluster to your app using
 * [Amazon ElastiCache](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/WhatIs.html).
 *
 * For existing usage, rename `sst.aws.Redis` to `sst.aws.Redis.v1`. For new Redis, use
 * the latest [`Redis`](/docs/component/aws/redis) component instead.
 *
 * :::caution
 * This component has been deprecated.
 * :::
 *
 * What changed:
 * - In this version, the Redis/Valkey cluster uses the default parameter group, which
 * cannot be customized.
 * - In the new version, the cluster now creates a custom parameter group. This allows
 * you to customize the parameters via the `transform` prop.
 *
 * @example
 *
 * #### Create the cluster
 *
 * ```js title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const redis = new sst.aws.Redis.v1("MyRedis", { vpc });
 * ```
 *
 * #### Link to a resource
 *
 * You can link your cluster to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [redis],
 *   vpc
 * });
 * ```
 *
 * Once linked, you can connect to it from your function code.
 *
 * ```ts title="app/page.tsx" {1,6,7,12,13}
 * import { Resource } from "sst";
 * import { Cluster } from "ioredis";
 *
 * const client = new Cluster(
 *   [{
 *     host: Resource.MyRedis.host,
 *     port: Resource.MyRedis.port
 *   }],
 *   {
 *     redisOptions: {
 *       tls: { checkServerIdentity: () => undefined },
 *       username: Resource.MyRedis.username,
 *       password: Resource.MyRedis.password
 *     }
 *   }
 * );
 * ```
 *
 * #### Running locally
 *
 * By default, your Redis cluster is deployed in `sst dev`. But let's say you are running Redis
 * locally.
 *
 * ```bash
 * docker run \
 *   --rm \
 *   -p 6379:6379 \
 *   -v $(pwd)/.sst/storage/redis:/data \
 *   redis:latest
 * ```
 *
 * You can connect to it in `sst dev` by configuring the `dev` prop.
 *
 * ```ts title="sst.config.ts" {3-6}
 * const redis = new sst.aws.Redis.v1("MyRedis", {
 *   vpc,
 *   dev: {
 *     host: "localhost",
 *     port: 6379
 *   }
 * });
 * ```
 *
 * This will skip deploying a Redis ElastiCache cluster and link to the locally running Redis
 * server instead. [Check out the full example](/docs/examples/#aws-redis-local).
 *
 * ---
 *
 * ### Cost
 *
 * By default this component uses _On-demand nodes_ with a single `cache.t4g.micro` instance.
 *
 * The default `redis` engine costs $0.016 per hour. That works out to $0.016 x 24 x 30 or **$12 per month**.
 *
 * If the `valkey` engine is used, the cost is $0.0128 per hour. That works out to $0.0128 x 24 x 30 or **$9 per month**.
 *
 * Adjust this for the `instance` type and number of `nodes` you are using.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [ElastiCache pricing](https://aws.amazon.com/elasticache/pricing/) for more details.
 */
export class Redis extends AWSComponent implements sst.Linkable {
  private cluster?: elasticache.ReplicationGroup;
  private _authToken?: sst.Output<string>;
  private dev?: {
    enabled: boolean;
    host: sst.Output<string>;
    port: sst.Output<number>;
    username: sst.Output<string>;
    password?: sst.Output<string>;
  };

  constructor(name: string, args: RedisArgs, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    if (args && "ref" in args) {
      const ref = args as unknown as RedisRef;
      this.cluster = ref.cluster;
      this._authToken = ref.authToken;
      return;
    }

    const parent = this;
    const engine = sst.output(args.engine).apply((v) => v ?? "redis");
    const version = sst
      .resolve([engine, args.version])
      .apply(([engine, v]) => v ?? (engine === "redis" ? "7.1" : "7.2"));
    const instance = sst.output(args.instance).apply((v) => v ?? "t4g.micro");
    const nodes = sst.output(args.nodes).apply((v) => v ?? 1);
    const vpc = normalizeVpc();

    const dev = registerDev();
    if (dev?.enabled) {
      this.dev = dev;
      return;
    }

    const { authToken, secret } = createAuthToken();
    const subnetGroup = createSubnetGroup();
    const cluster = createCluster();

    this.cluster = cluster;
    this._authToken = authToken;

    function registerDev() {
      if (!args.dev) return undefined;

      const dev = {
        enabled: sst.dev,
        host: sst.output(args.dev.host ?? "localhost"),
        port: sst.output(args.dev.port ?? 6379),
        username: sst.output(args.dev.username ?? "default"),
        password: args.dev.password ? sst.output(args.dev.password) : undefined,
      };

      new DevCommand(`${name}Dev`, {
        dev: {
          title: name,
          autostart: true,
          command: `sst print-and-not-quit`,
        },
        environment: {
          SST_DEV_COMMAND_MESSAGE: sst.interpolate`Make sure your local Redis server is using:

  username: "${dev.username}"
  password: ${
    dev.password ? `"${dev.password}"` : "\x1b[38;5;8m[no password]\x1b[0m"
  }

Listening on "${dev.host}:${dev.port}"...`,
        },
      });

      return dev;
    }

    function normalizeVpc() {
      // "vpc" is a Vpc component
      if (args.vpc instanceof Vpc) {
        return sst.output({
          subnets: args.vpc.privateSubnets,
          securityGroups: args.vpc.securityGroups,
        });
      }

      // "vpc" is object
      return sst.output(args.vpc);
    }

    function createAuthToken() {
      const authToken = new RandomPassword(
        `${name}AuthToken`,
        {
          length: 32,
          special: true,
          overrideSpecial: "!&#$^<>-",
        },
        { parent },
      ).result;

      const secret = new secretsmanager.Secret(
        `${name}ProxySecret`,
        {
          recoveryWindowInDays: 0,
        },
        { parent },
      );

      new secretsmanager.SecretVersion(
        `${name}ProxySecretVersion`,
        {
          secretId: secret.id,
          secretString: sst.json.stringify({ authToken }),
        },
        { parent },
      );

      return { secret, authToken };
    }

    function createSubnetGroup() {
      return new elasticache.SubnetGroup(
        ...transform(
          args.transform?.subnetGroup,
          `${name}SubnetGroup`,
          {
            description: "Managed by SST",
            subnetIds: vpc.subnets,
          },
          { parent },
        ),
      );
    }

    function createCluster() {
      return new elasticache.ReplicationGroup(
        ...transform(
          args.transform?.cluster,
          `${name}Cluster`,
          {
            description: "Managed by SST",
            engine,
            engineVersion: version,
            nodeType: sst.interpolate`cache.${instance}`,
            dataTieringEnabled: instance.apply((v) => v.startsWith("r6gd.")),
            port: 6379,
            automaticFailoverEnabled: true,
            clusterMode: "enabled",
            numNodeGroups: nodes,
            replicasPerNodeGroup: 0,
            multiAzEnabled: false,
            atRestEncryptionEnabled: true,
            transitEncryptionEnabled: true,
            transitEncryptionMode: "required",
            authToken,
            subnetGroupName: subnetGroup.name,
            securityGroupIds: vpc.securityGroups,
            tags: {
              "sst:auth-token-ref": secret.id,
            },
          },
          { parent },
        ),
      );
    }
  }

  /**
   * The ID of the Redis cluster.
   */
  public get clusterID() {
    return this.dev ? sst.output("placeholder") : this.cluster!.id;
  }

  /**
   * The username to connect to the Redis cluster.
   */
  public get username() {
    return this.dev ? this.dev.username : sst.output("default");
  }

  /**
   * The password to connect to the Redis cluster.
   */
  public get password() {
    return this.dev ? this.dev.password ?? sst.output("") : this._authToken;
  }

  /**
   * The host to connect to the Redis cluster.
   */
  public get host() {
    return this.dev
      ? this.dev.host
      : this.cluster!.configurationEndpointAddress;
  }

  /**
   * The port to connect to the Redis cluster.
   */
  public get port() {
    return this.dev ? this.dev.port : this.cluster!.port.apply((v) => v!);
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    const _this = this;
    return {
      /**
       * The ElastiCache Redis cluster.
       */
      get cluster() {
        if (_this.dev)
          throw new VisibleError("Cannot access `nodes.cluster` in dev mode.");
        return _this.cluster!;
      },
    };
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        host: this.host,
        port: this.port,
        username: this.username,
        password: this.password,
      },
    };
  }

  /**
   * Reference an existing Redis cluster with the given cluster name. This is useful when you
   * create a Redis cluster in one stage and want to share it in another. It avoids having to
   * create a new Redis cluster in the other stage.
   *
   * :::tip
   * You can use the `static get` method to share Redis clusters across stages.
   * :::
   *
   * @param name The name of the component.
   * @param clusterID The id of the existing Redis cluster.
   * @param opts? Resource options.
   *
   * @example
   * Imagine you create a cluster in the `dev` stage. And in your personal stage `frank`,
   * instead of creating a new cluster, you want to share the same cluster from `dev`.
   *
   * ```ts title="sst.config.ts"
   * const redis = $app.stage === "frank"
   *   ? sst.aws.Redis.v1.get("MyRedis", "app-dev-myredis")
   *   : new sst.aws.Redis.v1("MyRedis");
   * ```
   *
   * Here `app-dev-myredis` is the ID of the cluster created in the `dev` stage.
   * You can find this by outputting the cluster ID in the `dev` stage.
   *
   * ```ts title="sst.config.ts"
   * return {
   *   cluster: redis.clusterID
   * };
   * ```
   */
  public static get(
    name: string,
    clusterID: sst.Input<string>,
    opts?: sst.ComponentOptions,
  ) {
    const cluster = elasticache.ReplicationGroup.get(
      `${name}Cluster`,
      clusterID,
      undefined,
      opts,
    );
    const secret = cluster.tags.apply((tags) =>
      tags?.["sst:auth-token-ref"]
        ? secretsmanager.getSecretVersionOutput(
            {
              secretId: tags["sst:auth-token-ref"],
            },
            opts,
          )
        : sst.output(undefined),
    );
    const authToken = secret.apply((v) => {
      if (!v)
        throw new VisibleError(`Failed to get auth token for Redis ${name}.`);
      return JSON.parse(v.secretString).authToken as string;
    });

    return new Redis(name, {
      ref: true,
      cluster,
      authToken,
    } as unknown as RedisArgs);
  }
}

const __pulumiType = "sst:aws:Redis";
// @ts-expect-error
Redis.__pulumiType = __pulumiType;
