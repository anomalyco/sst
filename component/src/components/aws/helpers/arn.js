"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.parseFunctionArn = parseFunctionArn;
exports.parseBucketArn = parseBucketArn;
exports.parseTopicArn = parseTopicArn;
exports.parseQueueArn = parseQueueArn;
exports.parseDynamoArn = parseDynamoArn;
exports.parseDynamoStreamArn = parseDynamoStreamArn;
exports.parseKinesisStreamArn = parseKinesisStreamArn;
exports.parseEventBusArn = parseEventBusArn;
exports.parseRoleArn = parseRoleArn;
exports.parseElasticSearch = parseElasticSearch;
exports.parseOpenSearch = parseOpenSearch;
var error_1 = require("../../error");
function parseFunctionArn(arn) {
    // arn:aws:lambda:region:account-id:function:function-name
    var functionName = arn.split(":")[6];
    if (!arn.startsWith("arn:") || !functionName)
        throw new error_1.VisibleError("The provided ARN \"".concat(arn, "\" is not a Lambda function ARN."));
    return { functionName: functionName };
}
function parseBucketArn(arn) {
    // arn:aws:s3:::bucket-name
    var bucketName = arn.split(":")[5];
    if (!arn.startsWith("arn:") || !bucketName)
        throw new error_1.VisibleError("The provided ARN \"".concat(arn, "\" is not an S3 bucket ARN."));
    return { bucketName: bucketName };
}
function parseTopicArn(arn) {
    // arn:aws:sns:region:account-id:topic-name
    var topicName = arn.split(":")[5];
    if (!arn.startsWith("arn:") || !topicName)
        throw new error_1.VisibleError("The provided ARN \"".concat(arn, "\" is not an SNS Topic ARN."));
    return { topicName: topicName };
}
function parseQueueArn(arn) {
    // arn:aws:sqs:region:account-id:queue-name
    var _a = arn.split(":"), arnStr = _a[0], region = _a[3], accountId = _a[4], queueName = _a[5];
    if (arnStr !== "arn" || !queueName)
        throw new error_1.VisibleError("The provided ARN \"".concat(arn, "\" is not an SQS Queue ARN."));
    return {
        queueName: queueName,
        queueUrl: "https://sqs.".concat(region, ".amazonaws.com/").concat(accountId, "/").concat(queueName),
    };
}
function parseDynamoArn(arn) {
    // arn:aws:dynamodb:region:account-id:table/table-name
    var tableName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !tableName)
        throw new error_1.VisibleError("The provided ARN \"".concat(arn, "\" is not a DynamoDB table ARN."));
    return { tableName: tableName };
}
function parseDynamoStreamArn(streamArn) {
    var _a;
    // ie. "arn:aws:dynamodb:us-east-1:112233445566:table/MyTable/stream/2024-02-25T23:17:55.264"
    var parts = streamArn.split(":");
    var tableName = (_a = parts[5]) === null || _a === void 0 ? void 0 : _a.split("/")[1];
    if (parts[0] !== "arn" || parts[2] !== "dynamodb" || !tableName)
        throw new error_1.VisibleError("The provided ARN \"".concat(streamArn, "\" is not a DynamoDB stream ARN."));
    return { tableName: tableName };
}
function parseKinesisStreamArn(streamArn) {
    var _a;
    // ie. "arn:aws:kinesis:us-east-1:123456789012:stream/MyStream";
    var parts = streamArn.split(":");
    var streamName = (_a = parts[5]) === null || _a === void 0 ? void 0 : _a.split("/")[1];
    if (parts[0] !== "arn" || parts[2] !== "kinesis" || !streamName)
        throw new error_1.VisibleError("The provided ARN \"".concat(streamArn, "\" is not a Kinesis stream ARN."));
    return { streamName: streamName };
}
function parseEventBusArn(arn) {
    // arn:aws:events:region:account-id:event-bus/bus-name
    var busName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !busName)
        throw new error_1.VisibleError("The provided ARN \"".concat(arn, "\" is not a EventBridge event bus ARN."));
    return { busName: busName };
}
function parseRoleArn(arn) {
    // arn:aws:iam::123456789012:role/MyRole
    var roleName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !roleName)
        throw new error_1.VisibleError("The provided ARN \"".concat(arn, "\" is not an IAM role ARN."));
    return { roleName: roleName };
}
function parseElasticSearch(arn) {
    // arn:aws:es:region:account-id:domain/domain-name
    var tableName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !tableName)
        throw new error_1.VisibleError("The provided ARN \"".concat(arn, "\" is not a ElasticSearch domain ARN."));
    return { tableName: tableName };
}
function parseOpenSearch(arn) {
    // arn:aws:opensearch:region:account-id:domain/domain-name
    var tableName = arn.split("/")[1];
    if (!arn.startsWith("arn:") || !tableName)
        throw new error_1.VisibleError("The provided ARN \"".concat(arn, "\" is not a OpenSearch domain ARN."));
    return { tableName: tableName };
}
