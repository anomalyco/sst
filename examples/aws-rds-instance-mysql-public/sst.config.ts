/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS RDS MySQL public
 *
 * Create a publicly accessible MySQL RDS instance with a security group that
 * allows external connections.
 */
export default $config({
  app(input) {
    return {
      name: "aws-rds-instance-mysql-public",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const MYSQL_PORT = 3306;
    const ALL_IPS = '0.0.0.0/0';
    const publicSecurityGroup = new aws.ec2.SecurityGroup(
      'MyPublicSecurityGroup',
      {
        ingress: [
          {
            // Expose to public connection. Remove if not needed
            protocol: 'tcp',
            fromPort: MYSQL_PORT,
            toPort: MYSQL_PORT,
            cidrBlocks: [ALL_IPS],
          },
        ],
      },
    );

    const identifier = 'my-db-instance';

    const database = new aws.rds.Instance(
      'MyDbInstanceMySQL',
      {
        identifier,
        engine: 'mysql',
        // free-tier
        instanceClass: 'db.t3.micro',
        allocatedStorage: 20, // free-tier 20GB
        // credentials
        username: 'dev-user',
        password: 'dev-password',
        dbName: 'dev-database',
        // settings
        tags: { Name: identifier },
        skipFinalSnapshot: true,

        // allow public access
        vpcSecurityGroupIds: [publicSecurityGroup.id],
        publiclyAccessible: true,
      },
    );

    return {
      Database: database.address
    };
  },
});
