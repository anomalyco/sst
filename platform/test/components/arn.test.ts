import { describe, it, expect } from "vitest";
import {
  parseFunctionArn,
  splitQualifiedFunctionArn,
  parseBucketArn,
  parseTopicArn,
  parseQueueArn,
  parseDynamoArn,
  parseDynamoStreamArn,
  parseKinesisStreamArn,
  parseEventBusArn,
  parseRoleArn,
  parseLambdaEdgeArn,
  parseElasticSearch,
  parseOpenSearch,
} from "../../src/components/aws/helpers/arn";

describe("parseFunctionArn", () => {
  it("parses unqualified ARN", () => {
    expect(
      parseFunctionArn(
        "arn:aws:lambda:us-east-1:123456789012:function:my-func",
      ),
    ).toEqual({ functionName: "my-func" });
  });

  it("parses qualified ARN (returns function name without qualifier)", () => {
    expect(
      parseFunctionArn(
        "arn:aws:lambda:us-east-1:123456789012:function:my-func:live",
      ),
    ).toEqual({ functionName: "my-func" });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseFunctionArn("not-an-arn")).toThrow();
  });
});

describe("splitQualifiedFunctionArn", () => {
  it("returns unqualified ARN as-is with no qualifier", () => {
    const arn = "arn:aws:lambda:us-east-1:123456789012:function:my-func";
    expect(splitQualifiedFunctionArn(arn)).toEqual({
      unqualifiedArn: arn,
      qualifier: undefined,
    });
  });

  it("splits qualified ARN with alias", () => {
    expect(
      splitQualifiedFunctionArn(
        "arn:aws:lambda:us-east-1:123456789012:function:my-func:live",
      ),
    ).toEqual({
      unqualifiedArn:
        "arn:aws:lambda:us-east-1:123456789012:function:my-func",
      qualifier: "live",
    });
  });

  it("splits qualified ARN with version number", () => {
    expect(
      splitQualifiedFunctionArn(
        "arn:aws:lambda:us-east-1:123456789012:function:my-func:42",
      ),
    ).toEqual({
      unqualifiedArn:
        "arn:aws:lambda:us-east-1:123456789012:function:my-func",
      qualifier: "42",
    });
  });

  it("splits qualified ARN with $LATEST", () => {
    expect(
      splitQualifiedFunctionArn(
        "arn:aws:lambda:us-east-1:123456789012:function:my-func:$LATEST",
      ),
    ).toEqual({
      unqualifiedArn:
        "arn:aws:lambda:us-east-1:123456789012:function:my-func",
      qualifier: "$LATEST",
    });
  });
});

describe("parseBucketArn", () => {
  it("parses valid S3 ARN", () => {
    expect(parseBucketArn("arn:aws:s3:::my-bucket")).toEqual({
      bucketName: "my-bucket",
    });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseBucketArn("not-an-arn")).toThrow();
  });
});

describe("parseTopicArn", () => {
  it("parses valid SNS ARN", () => {
    expect(
      parseTopicArn("arn:aws:sns:us-east-1:123456789012:my-topic"),
    ).toEqual({ topicName: "my-topic" });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseTopicArn("not-an-arn")).toThrow();
  });
});

describe("parseQueueArn", () => {
  it("parses valid SQS ARN", () => {
    const result = parseQueueArn(
      "arn:aws:sqs:us-east-1:123456789012:my-queue",
    );
    expect(result.queueName).toBe("my-queue");
    expect(result.queueUrl).toBe(
      "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
    );
  });

  it("throws on invalid ARN", () => {
    expect(() => parseQueueArn("not-an-arn")).toThrow();
  });
});

describe("parseDynamoArn", () => {
  it("parses valid DynamoDB ARN", () => {
    expect(
      parseDynamoArn(
        "arn:aws:dynamodb:us-east-1:123456789012:table/MyTable",
      ),
    ).toEqual({ tableName: "MyTable" });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseDynamoArn("not-an-arn")).toThrow();
  });
});

describe("parseDynamoStreamArn", () => {
  it("parses valid DynamoDB stream ARN", () => {
    expect(
      parseDynamoStreamArn(
        "arn:aws:dynamodb:us-east-1:123456789012:table/MyTable/stream/2024-02-25T23:17:55.264",
      ),
    ).toEqual({ tableName: "MyTable" });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseDynamoStreamArn("not-an-arn")).toThrow();
  });

  it("throws on wrong service", () => {
    expect(() =>
      parseDynamoStreamArn("arn:aws:kinesis:us-east-1:123456789012:stream/s"),
    ).toThrow();
  });
});

describe("parseKinesisStreamArn", () => {
  it("parses valid Kinesis stream ARN", () => {
    expect(
      parseKinesisStreamArn(
        "arn:aws:kinesis:us-east-1:123456789012:stream/MyStream",
      ),
    ).toEqual({ streamName: "MyStream" });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseKinesisStreamArn("not-an-arn")).toThrow();
  });

  it("throws on wrong service", () => {
    expect(() =>
      parseKinesisStreamArn(
        "arn:aws:dynamodb:us-east-1:123456789012:table/MyTable",
      ),
    ).toThrow();
  });
});

describe("parseEventBusArn", () => {
  it("parses valid EventBridge ARN", () => {
    expect(
      parseEventBusArn(
        "arn:aws:events:us-east-1:123456789012:event-bus/my-bus",
      ),
    ).toEqual({ busName: "my-bus" });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseEventBusArn("not-an-arn")).toThrow();
  });
});

describe("parseRoleArn", () => {
  it("parses valid IAM role ARN", () => {
    expect(
      parseRoleArn("arn:aws:iam::123456789012:role/MyRole"),
    ).toEqual({ roleName: "MyRole" });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseRoleArn("not-an-arn")).toThrow();
  });
});

describe("parseLambdaEdgeArn", () => {
  it("parses valid Lambda@Edge ARN", () => {
    expect(
      parseLambdaEdgeArn(
        "arn:aws:lambda:us-east-1:123456789012:function:my-func:1",
      ),
    ).toEqual({ functionName: "my-func", region: "us-east-1", version: "1" });
  });

  it("throws on wrong region", () => {
    expect(() =>
      parseLambdaEdgeArn(
        "arn:aws:lambda:eu-west-1:123456789012:function:my-func:1",
      ),
    ).toThrow(/us-east-1/);
  });

  it("throws on unqualified ARN (no version)", () => {
    expect(() =>
      parseLambdaEdgeArn(
        "arn:aws:lambda:us-east-1:123456789012:function:my-func",
      ),
    ).toThrow(/qualified/);
  });

  it("throws on $LATEST", () => {
    expect(() =>
      parseLambdaEdgeArn(
        "arn:aws:lambda:us-east-1:123456789012:function:my-func:$LATEST",
      ),
    ).toThrow(/qualified/);
  });
});

describe("parseElasticSearch", () => {
  it("parses valid ElasticSearch domain ARN", () => {
    expect(
      parseElasticSearch(
        "arn:aws:es:us-east-1:123456789012:domain/my-domain",
      ),
    ).toEqual({ tableName: "my-domain" });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseElasticSearch("not-an-arn")).toThrow();
  });
});

describe("parseOpenSearch", () => {
  it("parses valid OpenSearch domain ARN", () => {
    expect(
      parseOpenSearch(
        "arn:aws:opensearch:us-east-1:123456789012:domain/my-domain",
      ),
    ).toEqual({ tableName: "my-domain" });
  });

  it("throws on invalid ARN", () => {
    expect(() => parseOpenSearch("not-an-arn")).toThrow();
  });
});
