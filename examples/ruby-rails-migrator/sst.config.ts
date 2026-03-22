/// <reference path="../../.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "ruby-rails-migrator",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc");
    const rds = new sst.aws.Postgres("MyDatabase", { vpc });

    const migrator = new sst.aws.Function("MyMigrator", {
      vpc,
      handler: "handler.handler",
      runtime: "ruby3.3",
      link: [rds],
      environment: {
        DB_HOST: rds.host,
        DB_NAME: rds.database,
        DB_USER: rds.username,
        DB_PASSWORD: rds.password,
        RAILS_ENV: "production",
      },
    });

    return {
      migrator: migrator.name,
    };
  },
});
