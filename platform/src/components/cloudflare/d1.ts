import { ComponentResourceOptions } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
import { Link } from "../link";
import { binding } from "./binding";
import { DEFAULT_ACCOUNT_ID } from ".";

export interface D1Args {
  /**
   * [Transform](/docs/components/#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the D1 resource.
     */
    database?: Transform<cloudflare.D1DatabaseArgs>;
  };
}

interface D1Ref {
  ref: boolean;
  database: cloudflare.D1Database;
}

/**
 * The `D1` component lets you add a [Cloudflare D1 database](https://developers.cloudflare.com/d1/) to
 * your app.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * const db = new sst.cloudflare.D1("MyDatabase");
 * ```
 *
 * #### Link to a worker
 *
 * You can link the db to a worker.
 *
 * ```ts {3} title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "./index.ts",
 *   link: [db],
 *   url: true
 * });
 * ```
 *
 * Once linked, you can use the SDK to interact with the db.
 *
 * ```ts title="index.ts" {1} "Resource.MyDatabase.prepare"
 * import { Resource } from "sst";
 *
 * await Resource.MyDatabase.prepare(
 *   "SELECT id FROM todo ORDER BY id DESC LIMIT 1",
 * ).first();
 * ```
 */
export class D1 extends Component implements Link.Linkable {
  private database: cloudflare.D1Database;

  constructor(name: string, args?: D1Args, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    if (args && "ref" in args) {
      const ref = args as D1Ref;
      this.database = ref.database;
      return;
    }

    const parent = this;

    const db = createDB();

    this.database = db;

    function createDB() {
      return new cloudflare.D1Database(
        ...transform(
          args?.transform?.database,
          `${name}Database`,
          {
            name: "",
            accountId: DEFAULT_ACCOUNT_ID,
          },
          { parent },
        ),
      );
    }
  }

  /**
   * When you link a D1 database, the database will be available to the worker and you can
   * query it using its [API methods](https://developers.cloudflare.com/d1/build-with-d1/d1-client-api/).
   *
   * @example
   * ```ts title="index.ts" {1} "Resource.MyDatabase.prepare"
   * import { Resource } from "sst";
   *
   * await Resource.MyDatabase.prepare(
   *   "SELECT id FROM todo ORDER BY id DESC LIMIT 1",
   * ).first();
   * ```
   *
   * @internal
   */
  getSSTLink() {
    return {
      properties: {
        databaseId: this.databaseId,
      },
      include: [
        binding({
          type: "d1DatabaseBindings",
          properties: {
            id: this.databaseId,
          },
        }),
      ],
    };
  }

  /**
   * The generated ID of the D1 database.
   */
  public get databaseId() {
    // Pulumi returns "accountId/databaseId" for imported databases
    return this.database.id.apply((id) =>
      id.includes("/") ? id.split("/")[1] : id,
    );
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Cloudflare D1 database.
       */
      database: this.database,
    };
  }

  /**
   * Reference an existing D1 Database with the given database ID. This is
   * useful when you create a D1 in one stage and want to share it in another.
   * It avoids having to create a new D1 Database in the other stage.
   *
   * :::tip
   * You can use the `static get` method to share D1 Databases across stages.
   * :::
   *
   * @param name The name of the component.
   * @param databaseId The database ID of the existing D1 Database.
   *
   * @example
   * Imagine you create a D1 Database in the `dev` stage. And in your personal
   * stage `giorgio`, instead of creating a new database, you want to share the
   * same database from `dev`.
   *
   * ```ts title="sst.config.ts"
   * const d1 = $app.stage === "giorgio"
   *   ? sst.cloudflare.D1.get("MyD1", "my-database-id")
   *   : new sst.cloudflare.D1("MyD1");
   * ```
   *
   * Here `my-database-id` is the ID of the D1 Database created in the `dev`
   * stage. You can find it by outputting the D1 Database in the `dev` stage.
   *
   * ```ts title="sst.config.ts"
   * return {
   *   d1
   * };
   * ```
   */
  public static get(
    name: string,
    databaseId: string,
    opts?: ComponentResourceOptions,
  ) {
    const database = cloudflare.D1Database.get(
      `${name}Database`,
      `${DEFAULT_ACCOUNT_ID}/${databaseId}`,
      undefined,
      opts,
    );
    return new D1(name, {
      ref: true,
      database,
    } as D1Args);
  }
}

const __pulumiType = "sst:cloudflare:D1";
// @ts-expect-error
D1.__pulumiType = __pulumiType;
