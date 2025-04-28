/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-mysql",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // NAT Gateways are required for Lambda functions
    const vpc = new sst.aws.Vpc("MyVpc", { bastion: true });
    const mysql = new sst.aws.Mysql("MyDatabase", {
      vpc,
    });
    const app = new sst.aws.Function("MyApp", {
      handler: "index.handler",
      url: true,
      link: [mysql],
    });

    return {
      app: app.url,
      host: mysql.host,
      port: mysql.port,
      username: mysql.username,
      password: mysql.password,
      database: mysql.database,
    };
  },
});
