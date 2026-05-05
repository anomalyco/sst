/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Cloudflare Hyperdrive PlanetScale
 *
 * Connect a Cloudflare Worker to a PlanetScale Postgres database through
 * Cloudflare Hyperdrive.
 *
 */
export default $config({
  app(input) {
    return {
      name: "cloudflare-hyperdrive-planetscale",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
      providers: {
        planetscale: "1.0.0",
      },
    };
  },
  async run() {
    const db = planetscale.getDatabaseVitessOutput({
      id: "mydb",
      organization: "myorg",
    });

    const branch =
      $app.stage === "production"
        ? planetscale.getPostgresBranchOutput({
            id: db.defaultBranch,
            database: db.name,
            organization: db.organization,
          })
        : new planetscale.PostgresBranch("DatabaseBranch", {
            database: db.name,
            name: $app.stage,
            organization: db.organization,
            parentBranch: db.defaultBranch,
          });

    const role = new planetscale.PostgresBranchRole("DatabaseRole", {
      branch: branch.name,
      database: db.name,
      inheritedRoles: ["pg_read_all_data", "pg_write_all_data"],
      name: `${$app.name}-${$app.stage}`,
      organization: db.organization,
    });

    const hyperdrive = new sst.cloudflare.Hyperdrive("Database", {
      origin: {
        host: role.accessHostUrl,
        database: role.databaseName,
        user: role.username,
        password: role.password,
        port: 6432, // Use 5432 for direct connection instead of PgBouncer
        scheme: "postgres",
      },
      caching: false,
    });

    const worker = new sst.cloudflare.Worker("Worker", {
      handler: "./worker.ts",
      link: [hyperdrive],
      url: true,
    });

    return {
      url: worker.url,
    };
  },
});
