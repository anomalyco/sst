/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS MySQL local
 *
 * In this example, we connect to a locally running MySQL instance for dev. While
 * on deploy, we use RDS.
 *
 * We use the [`docker run`](https://docs.docker.com/reference/cli/docker/container/run/) CLI
 * to start a local container with MySQL. You don't have to use Docker, you can use
 * any other way to run MySQL locally.
 *
 * ```bash
 * docker run \
 *   --rm \
 *   -p 3306:3306 \
 *   -v $(pwd)/.sst/storage/mysql:/var/lib/mysql/data \
 *   -e MYSQL_ROOT_PASSWORD=password \
 *   -e MYSQL_DATABASE=local \
 *   mysql:8.0
 * ```
 *
 * The data is saved to the `.sst/storage` directory. So if you restart the dev server, the
 * data will still be there.
 *
 * We then configure the `dev` property of the `Mysql` component with the settings for the
 * local MySQL instance.
 *
 * ```ts title="sst.config.ts"
 * dev: {
 *   username: "root",
 *   password: "password",
 *   database: "local",
 *   host: "localhost",
 *   port: 3306,
 * }
 * ```
 *
 * By providing the `dev` prop for Mysql, SST will use the local MySQL instance and
 * not deploy a new RDS database when running `sst dev`.
 *
 * It also allows us to access the database through a Resource `link` without having to
 * conditionally check if we are running locally.
 *
 * ```ts title="index.ts"
 * const pool = new Pool({
 *   host: Resource.MyDatabase.host,
 *   port: Resource.MyDatabase.port,
 *   user: Resource.MyDatabase.username,
 *   password: Resource.MyDatabase.password,
 *   database: Resource.MyDatabase.database,
 * });
 * ```
 *
 * The above will work in both `sst dev` and `sst deploy`.
 */
export default $config({
  app(input) {
    return {
      name: "aws-mysql-local",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc", { nat: "ec2" });

    const mysql = new sst.aws.Mysql("MyDatabase", {
      dev: {
        username: "root",
        password: "password",
        database: "local",
        host: "localhost",
        port: 3306,
      },
      vpc,
    });

    new sst.aws.Function("MyFunction", {
      vpc,
      url: true,
      link: [mysql],
      handler: "index.handler",
    });
  },
});
