import * as aws from "@pulumi/aws";
import { PolicyPack, validateResourceOfType } from "@pulumi/policy";

new PolicyPack("iam-roles-policies", {
  policies: [
    {
      name: "iam-role-requires-permission-boundary",
      description: "IAM roles must have a permission boundary.",
      enforcementLevel: "mandatory",
      validateResource: validateResourceOfType(
        aws.iam.Role,
        (role, _args, reportViolation) => {
          if (!role.permissionsBoundary) {
            reportViolation(
              "IAM roles must have a permission boundary. " +
                "Permission boundaries are important for limiting the maximum permissions " +
                "that can be granted to an IAM role.",
            );
          }
        },
      ),
    },
  ],
});
