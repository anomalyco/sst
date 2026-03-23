/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-postgres-shared-pg",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc");

    // Alternative: create a standalone parameter group and reference it
    //
    // const sharedParameterGroup = new aws.rds.ParameterGroup(
    //   "SharedParameterGroup",
    //   {
    //     name: "shared-parameter-group",
    //     family: "postgres17",
    //     parameters: [
    //       { name: "rds.force_ssl", value: "0" },
    //       { name: "log_min_duration_statement", value: "1000" },
    //     ],
    //   },
    // );
    //
    // Then use in each Postgres:
    //   transform: {
    //     instance: { parameterGroupName: sharedParameterGroup.name },
    //   }

    // First database with custom parameters
    const db1 = new sst.aws.Postgres("Database1", {
      vpc,
      transform: {
        parameterGroup: {
          parameters: [
            {
              name: "rds.force_ssl",
              value: "0",
            },
            {
              name: "rds.logical_replication",
              value: "1",
              applyMethod: "pending-reboot",
            },
            {
              name: "log_min_duration_statement",
              value: "1000",
            },
          ],
        },
      },
    });

    // Second database reuses db1's parameter group
    const db2 = new sst.aws.Postgres("Database2", {
      vpc,
      transform: {
        instance: {
          parameterGroupName: db1.nodes.instance.parameterGroupName,
        },
      },
    });

    return {
      db1Host: db1.host,
      db2Host: db2.host,
    };
  },
});
