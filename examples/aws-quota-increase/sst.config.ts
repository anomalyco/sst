/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Quota Increase
 *
 * Use the Pulumi AWS provider to request an increase to an AWS service quota.
 * In this example, we increase the Lambda concurrent executions quota.
 */
export default $config({
  app(input) {
    return {
      name: "aws-quota-increase",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    new aws.servicequotas.ServiceQuota("LambdaConcurrentExecutions", {
      serviceCode: "lambda",
      quotaCode: "L-B99A9384",
      value: 2000,
    });
  },
});
