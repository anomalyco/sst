/// <reference path="./.sst/platform/config.d.ts" />
/**
 * ## AWS EFS with SurrealDB
 *
 * We use the SurrealDB docker image to run a server in a container and use EFS as the file
 * system.
 *
 * ```ts title="sst.config.ts"
 * const server = new sst.aws.Service("MyService", {
 *   cluster,
 *   architecture: "arm64",
 *   image: "surrealdb/surrealdb:v2.0.2",
 *   // ...
 *   volumes: [
 *     { efs, path: "/data" },
 *   ],
 * });
 * ```
 *
 * We then connect to the server from a Lambda function.
 *
 * ```js title="index.ts"
 * const endpoint = `http://${Resource.MyConfig.host}:${Resource.MyConfig.port}`;
 *
 * const db = new Surreal();
 * await db.connect(endpoint);
 * ```
 *
 * This uses the SurrealDB client to connect to the server.
 *
 * :::note
 * Given the performance of EFS, it's not recommended to use it for databases.
 * :::
 *
 * This example is for demonstration purposes only. It's not recommended to use
 * EFS for databases in production.
 */
export default $config({
  app(input) {
    return {
      name: "aws-efs-surrealdb",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
      providers: { "@pulumi/random": "4.16.7" },
    };
  },
  async run() {
    const { RandomPassword } = await import("@pulumi/random");

    // SurrealDB Credentials
    const PORT = 8080;
    const NAMESPACE = "test";
    const DATABASE = "test";
    const USERNAME = "root";
    const PASSWORD = new RandomPassword("Password", {
      length: 32,
    }).result;

    // NAT Gateways are required for Lambda functions
    const vpc = new sst.aws.Vpc("MyVpc", { nat: "managed" });

    // Store SurrealDB data in EFS
    const efs = new sst.aws.Efs("MyEfs", { vpc });

    // Run SurrealDB server in a container
    const cluster = new sst.aws.Cluster("MyCluster", { vpc });
    const server = new sst.aws.Service("MyService", {
      cluster,
      architecture: "arm64",
      image: "surrealdb/surrealdb:v2.0.2",
      command: [
        "start",
        "--bind",
        $interpolate`0.0.0.0:${PORT}`,
        "--log",
        "info",
        "--user",
        USERNAME,
        "--pass",
        PASSWORD,
        "surrealkv://data/data.skv",
        "--allow-scripting",
      ],
      volumes: [{ efs, path: "/data" }],
    });

    // Lambda client to connect to SurrealDB
    const config = new sst.Linkable("MyConfig", {
      properties: {
        username: USERNAME,
        password: PASSWORD,
        namespace: NAMESPACE,
        database: DATABASE,
        port: PORT,
        host: server.service,
      },
    });

    new sst.aws.Function("MyApp", {
      handler: "index.handler",
      link: [config],
      url: true,
      vpc,
    });
  },
});
