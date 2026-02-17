/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Policy Pack Validation
 *
 * You can use Pulumi Policy Packs to enforce compliance and security policies on your
 * infrastructure before deployment. Created policies with and enforcement level of "mandatory" will block the deployment.
 *
 * This example shows how to use the `--policy` flag with `sst diff` and `sst deploy` to
 * validate your infrastructure against a policy pack.
 *
 * Run the diff command with a policy pack to preview changes and check for violations:
 *
 * ```bash
 * sst diff --policy ./policy-pack --stage prod
 * ```
 *
 * Deploy with policy validation:
 *
 * ```bash
 * sst deploy --policy ./policy-pack --stage prod
 * ```
 *
 * To get started you can create a new policy pack for AWS using:
 *
 * ```bash
 * mkdir policy-pack
 * cd policy-pack
 * pulumi policy new aws-typescript
 * ```
 *
 * The example policy pack (check the full example) enforces that all IAM roles must have a permission boundary, blocking the deployment in this sst example.
 */
export default $config({
  app(input) {
    return {
      name: "aws-policy-pack",
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
