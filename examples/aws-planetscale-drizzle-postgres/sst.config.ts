/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS PlanetScale Drizzle Postgres
 *
 * In this example, we use PlanetScale Postgres with a branch-per-stage pattern. Every stage
 * gets its own database branch — so each PR can have an isolated database.
 *
 * ```ts title="sst.config.ts"
 * const db = planetscale.getDatabasePostgresOutput({
 *   id: "mydb",
 *   organization: "myorg",
 * });
 *
 * const branch =
 *   $app.stage === "production"
 *     ? planetscale.getPostgresBranchOutput({
 *         id: db.defaultBranch,
 *         organization: db.organization,
 *         database: db.name,
 *       })
 *     : new planetscale.PostgresBranch("DatabaseBranch", {
 *         database: db.name,
 *         organization: db.organization,
 *         name: $app.stage,
 *         parentBranch: db.defaultBranch,
 *       });
 * ```
 *
 * We then create a role and wrap it in a `Linkable` to link it to a function.
 *
 * ```ts title="sst.config.ts" {3}
 * new sst.aws.Function("Api", {
 *   handler: "src/api.handler",
 *   link: [database],
 *   url: true,
 * });
 * ```
 *
 * You can push your Drizzle schema changes to PlanetScale with:
 *
 * ```bash
 * bun run db:push
 * ```
 *
 * In the function we use [Drizzle ORM](https://orm.drizzle.team) with the
 * [`Resource`](/docs/reference/sdk/#resource) helper.
 *
 * ```ts title="src/drizzle.ts"
 * import { drizzle } from "drizzle-orm/postgres-js";
 * import { Resource } from "sst";
 * import postgres from "postgres";
 *
 * const client = postgres({
 *   host: Resource.Database.host,
 *   username: Resource.Database.username,
 *   password: Resource.Database.password,
 *   database: Resource.Database.database,
 * });
 *
 * export const db = drizzle(client);
 * ```
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-planetscale-drizzle-postgres",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      providers: {
        planetscale: "1.0.0",
      },
    };
  },
  async run() {
    const db = planetscale.getDatabasePostgresOutput({
      id: "mydb",
      organization: "myorg",
    });

    const branch =
      $app.stage === "production"
        ? planetscale.getPostgresBranchOutput({
            id: db.defaultBranch,
            organization: db.organization,
            database: db.name,
          })
        : new planetscale.PostgresBranch("DatabaseBranch", {
            database: db.name,
            organization: db.organization,
            name: $app.stage,
            parentBranch: db.defaultBranch,
          });

    const role = new planetscale.PostgresBranchRole("DatabaseRole", {
      database: db.name,
      organization: db.organization,
      branch: branch.name,
      name: `${$app.name}-${$app.stage}`,
      inheritedRoles: [
        "pg_read_all_data",
        "pg_write_all_data",
        "postgres", // Only needed for pushing schema changes
      ],
    });

    const database = new sst.Linkable("Database", {
      properties: {
        host: role.accessHostUrl,
        username: role.username,
        password: role.password,
        database: role.databaseName,
        port: 6432, // Use 5432 for direct connection instead of PgBouncer
      },
    });

    const api = new sst.aws.Function("Api", {
      handler: "src/api.handler",
      link: [database],
      url: true,
    });

    return {
      url: api.url,
    };
  },
});
