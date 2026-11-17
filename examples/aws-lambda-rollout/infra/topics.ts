export function createTopics(opts?: { email?: string }) {
  const alertsTopic = new sst.aws.SnsTopic("Alerts");

  allowCloudWatchPublish("Alerts", alertsTopic);

  if (opts?.email) {
    subscribeEmail("Alerts", alertsTopic, opts.email);
  }

  return { alertsTopic };
}

function subscribeEmail(name: string, topic: sst.aws.SnsTopic, email: string) {
  new aws.sns.TopicSubscription(`${name}Email`, {
    topic: topic.arn,
    protocol: "email",
    endpoint: email,
  });
}

function allowCloudWatchPublish(name: string, topic: sst.aws.SnsTopic) {
  new aws.sns.TopicPolicy(`${name}CloudWatchPolicy`, {
    arn: topic.arn,
    policy: aws.iam.getPolicyDocumentOutput({
      statements: [
        {
          actions: ["sns:Publish"],
          principals: [
            { type: "Service", identifiers: ["cloudwatch.amazonaws.com"] },
          ],
          resources: [topic.arn],
        },
      ],
    }).json,
  });
}
