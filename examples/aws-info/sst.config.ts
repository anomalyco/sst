/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## Current AWS account
 *
 * You can use the `aws.getXXXXOutput()` provider functions to get info about the current
 * AWS account.
 * Learn more about [provider functions](/docs/providers/#functions).
 */
export default $config({
  app(input) {
    return {
      name: "aws-info",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    return {
      region: aws.getRegionOutput().region,
      account: aws.getCallerIdentityOutput({}).accountId,
    };
  },
});

