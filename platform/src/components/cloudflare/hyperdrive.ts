import { ComponentResourceOptions, Input, output } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
import { Link } from "../link";
import { binding } from "./binding";
import { DEFAULT_ACCOUNT_ID } from "./account-id";
import { DurationHours, toSeconds } from "../duration";

export interface HyperdriveGetArgs {
  /**
   * The ID of the existing Hyperdrive config.
   */
  hyperdriveId: string;
}

interface HyperdriveRef {
  ref: true;
  hyperdrive: cloudflare.HyperdriveConfig;
}

export interface HyperdriveArgs {
  /**
   * Configure caching for SQL queries sent through Hyperdrive.
   *
   * :::tip
   * Caching is enabled by default. Pass `false` to disable it.
   * :::
   *
   * @example
   * Disable caching.
   * ```js
   * {
   *   caching: false
   * }
   * ```
   *
   * Customize cache durations.
   * ```js
   * {
   *   caching: {
   *     maxAge: "30 minutes",
   *     staleWhileRevalidate: "30 seconds"
   *   }
   * }
   * ```
   */
  caching?: Input<
    | false
    | {
        /**
         * The maximum duration items should persist in the cache. Can be up to 1 hour.
         *
         * @default `"60 seconds"`
         *
         * @example
         * ```js
         * {
         *   caching: {
         *     maxAge: "30 minutes"
         *   }
         * }
         * ```
         */
        maxAge?: Input<DurationHours>;
        /**
         * The duration the cache may serve a stale response while it's being revalidated.
         *
         * @default `"15 seconds"`
         *
         * @example
         * ```js
         * {
         *   caching: {
         *     staleWhileRevalidate: "30 seconds"
         *   }
         * }
         * ```
         */
        staleWhileRevalidate?: Input<DurationHours>;
      }
  >;
  /**
   * Configure mTLS authentication when connecting to the origin database.
   */
  mtls?: Input<{
    /**
     * Define CA certificate ID obtained after uploading CA cert.
     */
    caCertificateId?: Input<string>;
    /**
     * Define mTLS certificate ID obtained after uploading client cert.
     */
    mtlsCertificateId?: Input<string>;
    /**
     * Set SSL mode to 'require', 'verify-ca', or 'verify-full' to verify the CA.
     */
    sslmode?: Input<string>;
  }>;
  /**
   * The (soft) maximum number of connections the Hyperdrive is allowed to make to the origin database.
   */
  connectionLimit?: Input<number>;
  /**
   * The connection details for the origin database Hyperdrive connects to.
   */
  origin: Input<{
    /**
     * Defines the Client ID of the Access token to use when connecting to the origin database.
     */
    accessClientId?: Input<string>;
    /**
     * Defines the Client Secret of the Access Token to use when connecting to the origin database. The API never returns this write-only value.
     */
    accessClientSecret?: Input<string>;
    /**
     * Set the name of your origin database.
     */
    database: Input<string>;
    /**
     * Defines the host (hostname or IP) of your origin database.
     */
    host: Input<string>;
    /**
     * Set the password needed to access your origin database. The API never returns this write-only value.
     */
    password: Input<string>;
    /**
     * Defines the port of your origin database. Defaults to 5432 for PostgreSQL or 3306 for MySQL if not specified.
     */
    port?: Input<number>;
    /**
     * Specifies the URL scheme used to connect to your origin database.
     */
    scheme: Input<"postgres" | "mysql">;
    /**
     * Set the user of your origin database.
     */
    user: Input<string>;
  }>;
  /**
   * [Transform](/docs/components/#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Hyperdrive config resource.
     */
    hyperdrive?: Transform<cloudflare.HyperdriveConfigArgs>;
  };
}

/**
 * The `Hyperdrive` component lets you add a [Cloudflare Hyperdrive](https://developers.cloudflare.com/hyperdrive/) to
 * your app.
 *
 * Hyperdrive can connect Workers to PostgreSQL and MySQL databases.
 * Set `origin.scheme` to `"postgres"` or `"mysql"`.
 *
 * @example
 *
 * #### PostgreSQL example
 *
 * ```ts title="sst.config.ts"
 * const hyperdrive = new sst.cloudflare.Hyperdrive("PostgresDatabase", {
 *   origin: {
 *     database: "app",
 *     host: "db.example.com",
 *     password: "secret",
 *     scheme: "postgres",
 *     user: "postgres",
 *   },
 * })
 * ```
 *
 * [Check out the PlanetScale](/docs/examples/#cloudflare-hyperdrive-planetscale)
 * or the [AWS RDS Postgres](/docs/examples/#cloudflare-hyperdrive-with-aws-postgres) examples for a complete guide.
 *
 * #### MySQL example
 *
 * ```ts title="sst.config.ts"
 * const hyperdrive = new sst.cloudflare.Hyperdrive("MySQLDatabase", {
 *   origin: {
 *     database: "app",
 *     host: "db.example.com",
 *     password: "secret",
 *     scheme: "mysql",
 *     user: "root",
 *   },
 * })
 * ```
 *
 * #### Link to a worker
 *
 * You can link Hyperdrive to a worker.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "./index.ts",
 *   link: [hyperdrive],
 *   url: true,
 * })
 * ```
 *
 * Once linked, you can use the SDK to access the Hyperdrive binding in your worker.
 *
 * ```ts title="index.ts" {4}
 * import postgres from "postgres"
 * import { Resource } from "sst/resource"
 *
 * const sql = postgres(Resource.PostgresDatabase.connectionString)
 * ```
 *
 * It also works with MySQL:
 *
 * ```ts title="index.ts" {4}
 * import mysql from "mysql2/promise"
 * import { Resource } from "sst/resource"
 *
 * const db = await mysql.createConnection(Resource.MySQLDatabase.connectionString)
 * ```
 */
export class Hyperdrive extends Component implements Link.Linkable {
  private hyperdrive: cloudflare.HyperdriveConfig;

  constructor(
    name: string,
    args: HyperdriveArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    if (args && "ref" in args) {
      const ref = args as unknown as HyperdriveRef;
      this.hyperdrive = ref.hyperdrive;
      return;
    }

    const parent = this;

    const origin = output(args.origin);
    const caching = normalizeCaching();

    this.hyperdrive = new cloudflare.HyperdriveConfig(
      ...transform(
        args.transform?.hyperdrive,
        `${name}Hyperdrive`,
        {
          accountId: DEFAULT_ACCOUNT_ID,
          caching,
          mtls: args.mtls,
          name: "",
          origin,
          originConnectionLimit: args.connectionLimit,
        },
        { parent },
      ),
    );

    function normalizeCaching() {
      if (args.caching === undefined) return undefined;
      return output(args.caching).apply((c) => {
        if (c === false) return { disabled: true };
        return {
          maxAge: c.maxAge ? toSeconds(c.maxAge) : undefined,
          staleWhileRevalidate: c.staleWhileRevalidate
            ? toSeconds(c.staleWhileRevalidate)
            : undefined,
        };
      });
    }
  }

  /**
   * When you link Hyperdrive to a worker, the Hyperdrive binding will be available in the
   * worker and you can use its `connectionString` to connect with a PostgreSQL or MySQL client.
   *
   * @example
   * ```ts title="index.ts" {3}
   * import postgres from "postgres"
   * import { Resource } from "sst"
   *
   * const sql = postgres(Resource.PostgresDatabase.connectionString)
   * ```
   *
   * For MySQL:
   *
   * ```ts title="index.ts" {3}
   * import mysql from "mysql2/promise"
   * import { Resource } from "sst"
   *
   * const db = await mysql.createConnection(Resource.MySQLDatabase.connectionString)
   * ```
   *
   * @internal
   */
  public getSSTLink() {
    return {
      properties: {
        id: this.id,
      },
      include: [
        binding({
          type: "hyperdriveBindings",
          properties: {
            id: this.id,
          },
        }),
      ],
    };
  }

  /**
   * The generated ID of the Hyperdrive config.
   */
  public get id() {
    // Pulumi returns "accountId/hyperdriveId" for imported resources.
    return this.hyperdrive.id.apply((id) =>
      id.includes("/") ? id.split("/")[1] : id,
    );
  }

  /**
   * The generated name of the Hyperdrive config.
   */
  public get name() {
    return this.hyperdrive.name;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Cloudflare Hyperdrive config.
       */
      hyperdrive: this.hyperdrive,
    };
  }

  /**
   * Reference an existing Hyperdrive config with the given ID. This is useful when you
   * create a Hyperdrive config in one stage and want to share it in another.
   *
   * :::tip
   * You can use the `static get` method to share Hyperdrive configs across stages.
   * :::
   *
   * @param name The name of the component.
   * @param args The arguments to get the Hyperdrive config.
   * @param opts Resource options.
   *
   * @example
   * Imagine you create a Hyperdrive config in the `dev` stage. And in your personal stage
   * `frank`, instead of creating a new one, you want to share the same one from `dev`.
   *
   * ```ts title="sst.config.ts"
   * const hyperdrive = $app.stage === "frank"
   *   ? sst.cloudflare.Hyperdrive.get("MyDatabase", {
   *       hyperdriveId: "a1b2c3d4e5f6",
   *     })
   *   : new sst.cloudflare.Hyperdrive("MyDatabase", { ... });
   * ```
   */
  public static get(
    name: string,
    args: HyperdriveGetArgs,
    opts?: ComponentResourceOptions,
  ) {
    const hyperdrive = cloudflare.HyperdriveConfig.get(
      `${name}Hyperdrive`,
      `${DEFAULT_ACCOUNT_ID}/${args.hyperdriveId}`,
      undefined,
      opts,
    );
    return new Hyperdrive(
      name,
      {
        ref: true,
        hyperdrive,
      } as unknown as HyperdriveArgs,
      opts,
    );
  }
}

const __pulumiType = "sst:cloudflare:Hyperdrive";
// @ts-expect-error
Hyperdrive.__pulumiType = __pulumiType;
