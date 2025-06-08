# SST Policy Check Example

This example demonstrates how to use Pulumi policy packs with SST to enforce infrastructure policies.

## Structure

- `policy-pack/`: Contains a Pulumi policy pack that enforces IAM roles to have permission boundaries
- `./`: Contains the example SST application that creates an IAM role without a permission boundary


## How It Works

The policy pack defines a mandatory policy that checks if IAM roles have a permission boundary. The AWS stack creates an IAM role without a permission boundary, which will trigger a policy violation when you run `sst diff` or `sst deploy` with the `--policy` flag.

## Running the Example

### Prerequisites

1. Make sure you have SST installed
2. Configure your AWS credentials

### Steps

1. First, build the policy pack (this step is critical):

```bash
cd examples/check-policy/policy-pack
npm install
npm run build
cd ../../..
```

> **Important**: The policy pack must be built before use. If you skip this step, you'll get TypeScript compilation errors when trying to run the policy checks.

2. Navigate to the aws-stack directory and install dependencies:

```bash
cd examples/check-policy/
npm install
```

3. Try to deploy the stack with the policy pack:

```bash
sst deploy --policy ./policy-pack --stage dev
```

**Expected Result**: The deployment will fail with a non-zero error code preventing you from deploying infrastructure that doesn't comply with your policies. It will display a message similar to:

```
✕  Policy Violations Detected

The deployment would violate policy constraints.
Review the policy violations and update your infrastructure accordingly.
```

**Note:** The policy violation detection is working correctly (preventing non-compliant infrastructure from being deployed), but the specific error message might not be displayed due to how policy violations are currently handled in SST v3 with Pulumi policy packs.

Note: You can also check for policy violation while performing an `sst diff`:

```bash
sst diff --policy ./policy-pack --stage dev
```

**Expected Result**: The command will fail with a non-zero exit code. It will display a message similar to:

```
✕  Failed: Policy Violations    

POLICY VIOLATION
preview failed
This deployment was blocked by a policy violation.
```

This indicates that the policy violation was detected (causing the non-zero exit code), but the specific error message about the IAM role requiring a permission boundary might not be displayed.

## Fixing the Policy Violation

To fix the policy violation, you need to add a permission boundary to the IAM role in the `sst.config.ts` file. Here's how to modify the file:

1. Open the `sst.config.ts` file in the aws-stack directory
2. Add a permission boundary policy before creating the role:

```typescript
// Create a permission boundary policy
const permissionsBoundary = new aws.iam.Policy("MyPermissionsBoundary", {
  policy: aws.iam.getPolicyDocumentOutput({
    statements: [
      {
        actions: ["s3:GetObject"],
        resources: ["*"],
      },
    ],
  }).json,
});

// Update the role to include the permission boundary
const role = new aws.iam.Role("RoleWithoutBoundary", {
  assumeRolePolicy: aws.iam.assumeRolePolicyForPrincipal({
    Service: "lambda.amazonaws.com",
  }),
  permissionsBoundary: permissionsBoundary.arn, // Add this line to fix the policy violation
});
```

3. Run the diff command again to verify the policy violation is fixed:

```bash
sst diff --policy ./policy-pack --stage dev
```

Now the diff should complete successfully without any policy violations.

## Why Use Policy Packs?

Policy packs help you enforce organizational policies and best practices across your infrastructure. They can prevent common security issues, ensure compliance with regulations, and maintain consistency across your infrastructure.

## Troubleshooting

### TypeScript Compilation Errors

If you see errors like:

```
error: [runtime] Running program '...' failed with an unhandled exception:
TSError: ⨯ Unable to compile TypeScript:
index.ts(2,52): error TS2307: Cannot find module '@pulumi/policy' or its corresponding type declarations.
```

This means the policy pack hasn't been properly built. Make sure to:

1. Navigate to the policy-pack directory
2. Run `npm install` to install dependencies
3. Run `npm run build` to compile the TypeScript code
4. Try running the `sst diff` or `sst deploy` command again

### Policy Configuration Errors

If you see a "Policy Configuration Error" message, check:

1. The path to your policy pack is correct
2. The policy pack has been built successfully
3. The policy pack's `PulumiPolicy.yaml` file is valid

### Policy Violations

If you see a "Policy Violations Detected" message, this means your infrastructure doesn't comply with the policies defined in the policy pack. Review the policy violations and update your infrastructure accordingly.