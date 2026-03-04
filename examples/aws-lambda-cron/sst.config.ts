/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "aws-lambda-cron",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const queue = new sst.aws.Queue("MyDLQ");

    const cron = new sst.aws.CronV2("MyCron", {
      schedule: "rate(1 minute)",
      function: "cron.handler",
      timezone: "America/New_York",
      retries: 3,
      dlq: queue.arn,
    });

    return {
      function: cron.nodes.function.name,
      dlq: queue.arn,
    };
  },
});
