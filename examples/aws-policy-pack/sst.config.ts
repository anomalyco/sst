/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-policy-pack-example",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const role = new aws.iam.Role("ExampleRoleWithBoundary", {
      assumeRolePolicy: aws.iam.assumeRolePolicyForPrincipal({
        Service: "lambda.amazonaws.com",
      }),
      // To make this compliant with the policy example, uncomment the following line:
      // permissionsBoundary: "arn:aws:iam::aws:policy/PowerUserAccess",
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
  },
});
