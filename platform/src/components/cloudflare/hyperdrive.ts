import { ComponentResourceOptions } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
import { Link } from "../link";
import { binding } from "./binding";
import { DEFAULT_ACCOUNT_ID } from "./account-id";

export interface HyperdriveArgs
  extends Omit<cloudflare.HyperdriveConfigArgs, "accountId" | "name"> {
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
 * Set `origin.scheme` to `"postgres"`, `"postgresql"`, or `"mysql"`.
 *
 * @example
 *
 * #### PostgreSQL example
 *
 * ```ts title="sst.config.ts"
 * const hyperdrive = new sst.cloudflare.Hyperdrive('PostgresDatabase', {
 *   origin: {
 *     database: 'app',
 *     host: 'db.example.com',
 *     password: 'secret',
 *     scheme: 'postgres',
 *     user: 'postgres',
 *   },
 * })
 * ```
 *
 * #### MySQL example
 *
 * ```ts title="sst.config.ts"
 * const hyperdrive = new sst.cloudflare.Hyperdrive('MySQLDatabase', {
 *   origin: {
 *     database: 'app',
 *     host: 'db.example.com',
 *     password: 'secret',
 *     scheme: 'mysql',
 *     user: 'root',
 *   },
 * })
 * ```
 *
 * #### Link to a worker
 *
 * You can link Hyperdrive to a worker.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.cloudflare.Worker('MyWorker', {
 *   handler: './index.ts',
 *   link: [hyperdrive],
 *   url: true,
 * })
 * ```
 *
 * Once linked, you can use the SDK to access the Hyperdrive binding in your worker.
 *
 * ```ts title="index.ts" {3}
 * import postgres from "postgres"
 * import { Resource } from "sst/resource"
 *
 * const sql = postgres(Resource.PostgresDatabase.connectionString)
 * ```
 *
 * For MySQL, use a MySQL client.
 *
 * ```ts title="index.ts" {3}
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

    const parent = this;

    this.hyperdrive = new cloudflare.HyperdriveConfig(
      ...transform(
        args.transform?.hyperdrive,
        `${name}Hyperdrive`,
        {
          accountId: DEFAULT_ACCOUNT_ID,
          caching: args.caching,
          mtls: args.mtls,
          name: "",
          origin: args.origin,
          originConnectionLimit: args.originConnectionLimit,
        },
        { parent },
      ),
    );
  }

  /**
   * When you link Hyperdrive to a worker, the Hyperdrive binding will be available in the
   * worker and you can use its `connectionString` to connect with a PostgreSQL or MySQL client.
   *
   * @example
   * ```ts title="index.ts" {3}
   * import postgres from 'postgres'
   * import { Resource } from 'sst'
   *
   * const sql = postgres(Resource.PostgresDatabase.connectionString)
   * ```
   *
   * For MySQL:
   *
   * ```ts title="index.ts" {3}
   * import mysql from 'mysql2/promise'
   * import { Resource } from 'sst'
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
}

const __pulumiType = "sst:cloudflare:Hyperdrive";
// @ts-expect-error
Hyperdrive.__pulumiType = __pulumiType;
