/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-hyperdrive",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
      providers: {
        planetscale: "1.0.0",
      },
    };
  },
  async run() {
    const organization = new sst.Secret("PLANETSCALE_ORGANIZATION");
    const databaseId = new sst.Secret("PLANETSCALE_DATABASE_ID");

    const database = planetscale.getDatabasePostgresOutput({
      id: databaseId.value,
      organization: organization.value,
    });

    const branch =
      $app.stage === "production"
        ? planetscale.getPostgresBranchOutput({
            id: database.defaultBranch,
            database: database.name,
            organization: database.organization,
          })
        : new planetscale.PostgresBranch("DatabaseBranch", {
            database: database.name,
            name: $app.stage,
            organization: database.organization,
            parentBranch: database.defaultBranch,
          });

    const role = new planetscale.PostgresBranchRole("DatabaseRole", {
      branch: branch.name,
      database: database.name,
      inheritedRoles: ["pg_read_all_data", "pg_write_all_data"],
      name: `${$app.name}-${$app.stage}`,
      organization: database.organization,
    });

    const hyperdrive = new sst.cloudflare.Hyperdrive("Database", {
      origin: {
        host: role.accessHostUrl,
        database: role.databaseName,
        user: role.username,
        password: role.password,
        port: 6432, // Use 5432 for direct connection instead of PgBouncer
        scheme: "postgresql",
      },
      caching: {
        disabled: true,
      },
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
