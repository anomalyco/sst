import { VisibleError } from "sst-plugin/error";

export namespace arn {
  export function parseFunction(arn: string) {
    // arn:aws:lambda:region:account-id:function:function-name
    const functionName = arn.split(":")[6];
    if (!arn.startsWith("arn:") || !functionName)
      throw new VisibleError(
        `The provided ARN "${arn}" is not a Lambda function ARN.`,
      );
    return { functionName };
  }

  export function parseBucket(arn: string) {
    // arn:aws:s3:::bucket-name
    const bucketName = arn.split(":")[5];
    if (!arn.startsWith("arn:") || !bucketName)
      throw new VisibleError(
        `The provided ARN "${arn}" is not an S3 bucket ARN.`,
      );
    return { bucketName };
  }

  export function parseTopic(arn: string) {
    // arn:aws:sns:region:account-id:topic-name
    const topicName = arn.split(":")[5];
    if (!arn.startsWith("arn:") || !topicName)
      throw new VisibleError(
        `The provided ARN "${arn}" is not an SNS Topic ARN.`,
      );
    return { topicName };
  }

  export function parseQueue(arn: string) {
    // arn:aws:sqs:region:account-id:queue-name
    const [arnStr, , , region, accountId, queueName] = arn.split(":");
    if (arnStr !== "arn" || !queueName)
      throw new VisibleError(
        `The provided ARN "${arn}" is not an SQS Queue ARN.`,
      );
    return {
      queueName,
      queueUrl: `https://sqs.${region}.amazonaws.com/${accountId}/${queueName}`,
    };
  }

  export function parseDynamo(arn: string) {
    // arn:aws:dynamodb:region:account-id:table/table-name
    const tableName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !tableName)
      throw new VisibleError(
        `The provided ARN "${arn}" is not a DynamoDB table ARN.`,
      );
    return { tableName };
  }

  export function parseDynamoStream(streamArn: string) {
    // ie. "arn:aws:dynamodb:us-east-1:112233445566:table/MyTable/stream/2024-02-25T23:17:55.264"
    const parts = streamArn.split(":");
    const tableName = parts[5]?.split("/")[1];
    if (parts[0] !== "arn" || parts[2] !== "dynamodb" || !tableName)
      throw new VisibleError(
        `The provided ARN "${streamArn}" is not a DynamoDB stream ARN.`,
      );
    return { tableName };
  }

  export function parseKinesisStream(streamArn: string) {
    // ie. "arn:aws:kinesis:us-east-1:123456789012:stream/MyStream";
    const parts = streamArn.split(":");
    const streamName = parts[5]?.split("/")[1];
    if (parts[0] !== "arn" || parts[2] !== "kinesis" || !streamName)
      throw new VisibleError(
        `The provided ARN "${streamArn}" is not a Kinesis stream ARN.`,
      );
    return { streamName };
  }

  export function parseEventBus(arn: string) {
    // arn:aws:events:region:account-id:event-bus/bus-name
    const busName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !busName)
      throw new VisibleError(
        `The provided ARN "${arn}" is not a EventBridge event bus ARN.`,
      );
    return { busName };
  }

  export function parseRole(arn: string) {
    // arn:aws:iam::123456789012:role/MyRole
    const roleName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !roleName)
      throw new VisibleError(
        `The provided ARN "${arn}" is not an IAM role ARN.`,
      );
    return { roleName };
  }

  export function parseElasticSearch(arn: string) {
    // arn:aws:es:region:account-id:domain/domain-name
    const tableName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !tableName)
      throw new VisibleError(
        `The provided ARN "${arn}" is not a ElasticSearch domain ARN.`,
      );
    return { tableName };
  }

  export function parseOpenSearch(arn: string) {
    // arn:aws:opensearch:region:account-id:domain/domain-name
    const tableName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !tableName)
      throw new VisibleError(
        `The provided ARN "${arn}" is not a OpenSearch domain ARN.`,
      );
    return { tableName };
  }
}
