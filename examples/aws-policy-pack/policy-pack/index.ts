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
              "Permission boundaries are important for limiting the maximum permissions that can be granted to an IAM role."
            );
          }
        },
      ),
    },
    {
      name: "iam-role-policy-no-wildcard-resources",
      description:
        "IAM role policies should avoid wildcard resources for better security.",
      enforcementLevel: "advisory",
      validateResource: validateResourceOfType(
        aws.iam.RolePolicy,
        (policy, _args, reportViolation) => {
          if (policy.policy && typeof policy.policy === "string") {
            try {
              const policyDoc = JSON.parse(policy.policy);
              const statements = policyDoc.Statement || [];
              for (const statement of statements) {
                const resources = Array.isArray(statement.Resource)
                  ? statement.Resource
                  : [statement.Resource];
                if (resources.includes("*")) {
                  reportViolation(
                    "IAM role policies should not use wildcard (*) for resources. Specify explicit resource ARNs to follow the principle of least privilege."
                  );
                  break;
                }
              }
            } catch (e) {}
          }
        },
      ),
    },
  ],
});
