/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Cluster custom autoscaling
 *
 * In this example, we'll create a cluster that autoscales based on a custom
 * metric. In this case, the number of messages in a queue.
 *
 * We'll create a queue, and two functions that'll seed and purge the queue. We'll
 * also create two policies.
 *
 * One that scales it up.
 *
 * ```ts title="sst.config.ts"
 * const scaleUpPolicy = new aws.appautoscaling.Policy("ScaleUpPolicy", {
 *   serviceNamespace: service.nodes.autoScalingTarget.serviceNamespace,
 *   scalableDimension: service.nodes.autoScalingTarget.scalableDimension,
 *   resourceId: service.nodes.autoScalingTarget.resourceId,
 *   policyType: "StepScaling",
 *   stepScalingPolicyConfiguration: {
 *     adjustmentType: "ChangeInCapacity",
 *     cooldown: 5,
 *     stepAdjustments: [
 *       {
 *         metricIntervalLowerBound: "0",
 *         scalingAdjustment: 1,
 *       },
 *     ],
 *   },
 * });
 * ```
 *
 * And one that scales it down.
 *
 * ```ts title="sst.config.ts"
 * const scaleDownPolicy = new aws.appautoscaling.Policy("ScaleDownPolicy", {
 *   serviceNamespace: service.nodes.autoScalingTarget.serviceNamespace,
 *   scalableDimension: service.nodes.autoScalingTarget.scalableDimension,
 *   resourceId: service.nodes.autoScalingTarget.resourceId,
 *   policyType: "StepScaling",
 *   stepScalingPolicyConfiguration: {
 *     adjustmentType: "ChangeInCapacity",
 *     cooldown: 5,
 *     stepAdjustments: [
 *       {
 *         metricIntervalUpperBound: "0",
 *         scalingAdjustment: -1,
 *       },
 *     ],
 *   },
 * });
 * ```
 *
 * We'll add a CloudWatch metric alarm that triggers the scaling policies if the
 * queue depth exceeds 3 messages.
 *
 * ```ts title="sst.config.ts"
 * new aws.cloudwatch.MetricAlarm("QueueDepthAlarm", {
 *   comparisonOperator: "GreaterThanThreshold",
 *   evaluationPeriods: 1,
 *   metricName: "ApproximateNumberOfMessagesVisible",
 *   namespace: "AWS/SQS",
 *   period: 10,
 *   statistic: "Average",
 *   threshold: 3,
 *   dimensions: {
 *     QueueName: queue.nodes.queue.name,
 *   },
 *   alarmDescription: "Scale up when queue depth exceeds 3 messages",
 *   alarmActions: [scaleUpPolicy.arn],
 *   okActions: [scaleDownPolicy.arn],
 * });
 * ```
 *
 * To test this example, first deploy your app then:
 *
 * 1. Invoke the `MyQueueSeeder` URL. This will cause the service to scale up to 5
 *    instances in a few minutes.
 * 2. Then invoke the `MyQueuePurger` URL. This will cause the service to scale
 *    down to 1 instance in a few minutes.
 */
export default $config({
  app(input) {
    return {
      name: "aws-cluster-autoscaling",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const vpc = new sst.aws.Vpc("MyVpc");

    // Create a queue and two functions to seed and purge the queue
    const queue = new sst.aws.Queue("MyQueue");
    new sst.aws.Function("MyQueueSeeder", {
      handler: "queue.seeder",
      link: [queue],
      url: true,
    });
    new sst.aws.Function("MyQueuePurger", {
      handler: "queue.purger",
      link: [queue],
      url: true,
    });

    // Create a cluster and disable default scaling on CPU and memory utilization
    const cluster = new sst.aws.Cluster("MyCluster", { vpc });
    const service = new sst.aws.Service("MyService", {
      cluster,
      loadBalancer: {
        ports: [{ listen: "80/http" }],
      },
      scaling: {
        min: 1,
        max: 5,
        cpuUtilization: false,
        memoryUtilization: false,
      },
    });

    // Create a scale up policy that scales up by 1 instance at a time
    const scaleUpPolicy = new aws.appautoscaling.Policy("ScaleUpPolicy", {
      serviceNamespace: service.nodes.autoScalingTarget.serviceNamespace,
      scalableDimension: service.nodes.autoScalingTarget.scalableDimension,
      resourceId: service.nodes.autoScalingTarget.resourceId,
      policyType: "StepScaling",
      stepScalingPolicyConfiguration: {
        adjustmentType: "ChangeInCapacity",
        cooldown: 5,
        stepAdjustments: [
          {
            metricIntervalLowerBound: "0",
            scalingAdjustment: 1,
          },
        ],
      },
    });

    // Create a scale down policy that scales down by 1 instance at a time
    const scaleDownPolicy = new aws.appautoscaling.Policy("ScaleDownPolicy", {
      serviceNamespace: service.nodes.autoScalingTarget.serviceNamespace,
      scalableDimension: service.nodes.autoScalingTarget.scalableDimension,
      resourceId: service.nodes.autoScalingTarget.resourceId,
      policyType: "StepScaling",
      stepScalingPolicyConfiguration: {
        adjustmentType: "ChangeInCapacity",
        cooldown: 5,
        stepAdjustments: [
          {
            metricIntervalUpperBound: "0",
            scalingAdjustment: -1,
          },
        ],
      },
    });

    // Create an alarm that scales up when the queue depth exceeds 3 messages
    // and scales down when the queue depth is less than 3 messages
    new aws.cloudwatch.MetricAlarm("QueueDepthAlarm", {
      comparisonOperator: "GreaterThanThreshold",
      evaluationPeriods: 1,
      metricName: "ApproximateNumberOfMessagesVisible",
      namespace: "AWS/SQS",
      period: 10,
      statistic: "Average",
      threshold: 3,
      dimensions: {
        QueueName: queue.nodes.queue.name,
      },
      alarmDescription: "Scale up when queue depth exceeds 3 messages",
      alarmActions: [scaleUpPolicy.arn],
      okActions: [scaleDownPolicy.arn],
    });
  },
});
