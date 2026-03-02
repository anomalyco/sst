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

    const cron = new sst.aws.Cron("MyCron", {
      schedule: "rate(1 minute)",
      function: "cron.handler",
      retry: {
        attempts: 3,
        maxAge: "1 hour",
      },
      dlq: queue.arn,
    });

    return {
      function: cron.nodes.function.name,
      dlq: queue.arn,
    };
  },
});
