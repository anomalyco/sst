/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS PlanetScale Drizzle
 *
 * In this example, we use PlanetScale with a branch-per-stage pattern. Every stage gets its own
 * database branch — so each PR can have an isolated database.
 *
 * ```ts title="sst.config.ts"
 * const db = planetscale.getDatabaseOutput({
 *   name: "mydb",
 *   organization: "myorg",
 * });
 *
 * const branch =
 *   $app.stage === "production"
 *     ? planetscale.getBranchOutput({
 *         name: "production",
 *         organization: db.organization,
 *         database: db.name,
 *       })
 *     : new planetscale.Branch("DatabaseBranch", {
 *         database: db.name,
 *         organization: db.organization,
 *         name: $app.stage,
 *         parentBranch: "production",
 *       });
 * ```
 *
 * We then create a password and wrap it in a `Linkable` to link it to a function.
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
 * import { drizzle } from "drizzle-orm/planetscale-serverless";
 * import { Resource } from "sst";
 *
 * export const db = drizzle({
 *   connection: {
 *     host: Resource.Database.host,
 *     username: Resource.Database.username,
 *     password: Resource.Database.password,
 *   },
 * });
 * ```
 *
 */
export default $config({
  app(input) {
    return {
      name: "aws-planetscale-drizzle",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      providers: {
        planetscale: "0.4.1",
      },
    };
  },
  async run() {
    const db = planetscale.getDatabaseOutput({
      name: "mydb",
      organization: "myorg",
    });

    const branch =
      $app.stage === "production"
        ? planetscale.getBranchOutput({
            name: "production",
            organization: db.organization,
            database: db.name,
          })
        : new planetscale.Branch("DatabaseBranch", {
            database: db.name,
            organization: db.organization,
            name: $app.stage,
            parentBranch: "production",
          });

    const password = new planetscale.Password("DatabasePassword", {
      database: db.name,
      organization: db.organization,
      branch: branch.name,
      role: "admin",
      name: `${$app.name}-${$app.stage}`,
    });

    const database = new sst.Linkable("Database", {
      properties: {
        host: password.accessHostUrl,
        username: password.username,
        password: password.plaintext,
        database: db.name,
        port: 3306,
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
