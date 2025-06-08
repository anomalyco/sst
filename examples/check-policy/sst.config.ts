/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Check SST stack against compliance policies
 *
 * You can use it using the command:
 * `sst diff --policy /path/to/policy-pack`
 * This will help you view if the stack is compliant with the policies of the org that the stack belongs to.
 * You also have the option to use the below command:
 * `sst deploy --policy /path/to/policy-pack`
 * 
 * This will prevent the stack from being deployed if it is not compliant with the policies.
 */
export default $config({
  app(input) {
    return {
      name: "check-policy-example",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {

    const role = new aws.iam.Role("RoleWithoutBoundary", {
      assumeRolePolicy: aws.iam.assumeRolePolicyForPrincipal({
        Service: "lambda.amazonaws.com",
      }),
    });

    new aws.iam.RolePolicy("S3GetItemPolicy", {
      role: role.id,
      policy: aws.iam.getPolicyDocumentOutput({
        statements: [
          {
            actions: ["s3:GetObject"],
            resources: ["*"],
          },
        ],
      }).json,
    });

    const fn = new sst.aws.Function("MyFunction", {
      handler: "index.handler",
      role: role.arn,
      url: true,
    });

    return {
      url: fn.url,
    };
  },
});