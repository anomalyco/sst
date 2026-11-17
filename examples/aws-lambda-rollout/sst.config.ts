/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Lambda Rollout
 *
 * Deploys a Lambda function with a canary rollout. Each deploy publishes a new version
 * and uses CodeDeploy to gradually shift traffic — 10% for 10 minutes, then 100%.
 *
 * CloudWatch alarms monitor the error rate and latency during the rollout. If either
 * alarm fires, CodeDeploy automatically rolls back to the previous version.
 *
 * An SNS topic sends notifications on failures, rollbacks, and stops.
 */
export default $config({
  app(input) {
    return {
      name: "aws-lambda-rollout",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    const { createTopics } = await import("./infra/topics");
    const { createCanaryAlarms } = await import("./infra/alarms");

    const { alertsTopic } = createTopics({
      // email: EMAIL,
    });

    const fn = new sst.aws.Function("Function", {
      handler: "src/api.handler",
      rollout: { latestUrl: true },
      url: true,
      // Rollout only runs when function code changes. Set to false to deploy
      // actual code since sst dev deploys a stub that never changes.
      dev: false,
    });

    const { errorAlarm: canaryErrorAlarm, latencyAlarm: canaryLatencyAlarm } =
      createCanaryAlarms("Function", {
        fn,
        alertsTopic,
      });

    fn.addRollout({
      type: "canary",
      percentage: 10,
      duration: "10 minutes",
      wait: true,
      alarms: [canaryErrorAlarm.name, canaryLatencyAlarm.name],
      notifications: [
        {
          name: "Alerts",
          events: ["failure", "rollback", "stop"],
          topic: alertsTopic.arn,
        },
      ],
    });

    $util
      .all([
        fn.url,
        fn.nodes.function.version,
        fn.nodes.rolloutDeployment?.apply(
          (deployment) => deployment?.deploymentId,
        ),
      ])
      .apply(async ([url, version, deploymentId]) => {
        // wait for CodeDeploy to update the lambda alias
        await new Promise((r) => setTimeout(r, 10_000));

        console.log(
          `\nDeployed version ${version} (deployment: ${deploymentId})\n`,
        );
        const { loadTest } = await import("./scripts/test");
        await loadTest(url);
      });

    return {
      url: fn.url,
      latestUrl: fn.latestUrl,
    };
  },
});
