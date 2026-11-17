export function createCanaryAlarms(
  name: string,
  opts: {
    fn: sst.aws.Function;
    alertsTopic: sst.aws.SnsTopic;
  },
) {
  const { fn, alertsTopic } = opts;

  // These alarms track a specific Lambda version via the ExecutedVersion
  // dimension. When we deploy a new version, we replace the alarm so it starts
  // fresh — no leftover state from the previous version. The old alarm is
  // deleted after the new one is created so the deployment is not interrupted
  // by the removal of the alarm mid-deploy.
  const pulumiOpts: $util.CustomResourceOptions = {
    replaceOnChanges: ["dimensions"],
    deleteBeforeReplace: false,
  };

  // Triggers if any errors occur in the deployed version within a 5-minute window.
  const errorAlarm = new aws.cloudwatch.MetricAlarm(
    `${name}ErrorAlarm`,
    {
      alarmActions: [alertsTopic.arn],
      namespace: "AWS/Lambda",
      metricName: "Errors",
      dimensions: {
        FunctionName: fn.name,
        Resource: getFunctionResource(fn.targetArn),
        ExecutedVersion: fn.nodes.function.version,
      },
      statistic: "Sum",
      period: 300,
      evaluationPeriods: 1,
      threshold: 1,
      comparisonOperator: "GreaterThanOrEqualToThreshold",
      treatMissingData: "notBreaching",
    },
    pulumiOpts,
  );

  // Triggers if average latency exceeds 2 seconds in a 5-minute window.
  const latencyAlarm = new aws.cloudwatch.MetricAlarm(
    `${name}LatencyAlarm`,
    {
      alarmActions: [alertsTopic.arn],
      namespace: "AWS/Lambda",
      metricName: "Duration",
      dimensions: {
        FunctionName: fn.name,
        Resource: getFunctionResource(fn.targetArn),
        ExecutedVersion: fn.nodes.function.version,
      },
      statistic: "Average",
      period: 300,
      evaluationPeriods: 1,
      threshold: 2000,
      comparisonOperator: "GreaterThanOrEqualToThreshold",
      treatMissingData: "notBreaching",
    },
    pulumiOpts,
  );

  return { errorAlarm, latencyAlarm };
}

/**
 * Extracts the `FunctionName:Alias` resource identifier from a Lambda alias ARN.
 * CloudWatch metrics use this format for the `Resource` dimension when tracking
 * a specific alias.
 *
 * For example, given `arn:aws:lambda:us-east-1:123456789:function:my-fn:live`,
 * this returns `my-fn:live`.
 */
function getFunctionResource(targetArn: $util.Input<string>) {
  return aws
    .getArnOutput({ arn: targetArn })
    .resource.apply((r) => r.split(":").slice(1).join(":"));
}
