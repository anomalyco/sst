import * as sst from "sst-plugin";
import { Transform, transform } from "sst-plugin/internal/transform";
import { VisibleError } from "sst-plugin/error";
import { lambda } from "@pulumi/aws";
import { FunctionArgs } from "./function.js";
import { KinesisStreamLambdaSubscriberArgs } from "./kinesis-stream.js";
import { FunctionBuilder, functionBuilder } from "./util/function-builder.js";
import { AWSComponent } from "../component.js";
import { arn } from "../arn.js";

export interface Args extends KinesisStreamLambdaSubscriberArgs {
  /**
   * The Kinesis stream to use.
   */
  stream: sst.Input<{
    /**
     * The ARN of the stream.
     */
    arn: sst.Input<string>;
  }>;
  /**
   * The subscriber function.
   */
  subscriber: sst.Input<string | FunctionArgs>;
}

/**
 * The `KinesisStreamLambdaSubscriber` component is internally used by the `KinesisStream` component to
 * add a consumer to [Amazon Kinesis Data Streams](https://docs.aws.amazon.com/streams/latest/dev/introduction.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `KinesisStream` component.
 */
export class KinesisStreamLambdaSubscriber extends AWSComponent {
  private readonly fn: FunctionBuilder;
  private readonly eventSourceMapping: lambda.EventSourceMapping;
  constructor(name: string, args: Args, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const stream = sst.output(args.stream);
    const fn = createFunction();
    const eventSourceMapping = createEventSourceMapping();

    this.fn = fn;
    this.eventSourceMapping = eventSourceMapping;

    function createFunction() {
      return sst.output(args.subscriber).apply((subscriber) => {
        return functionBuilder(
          `${name}Function`,
          subscriber,
          {
            description: `Subscribed to ${name}`,
            permissions: [
              {
                actions: [
                  "kinesis:DescribeStream",
                  "kinesis:DescribeStreamSummary",
                  "kinesis:GetRecords",
                  "kinesis:GetShardIterator",
                  "kinesis:ListShards",
                  "kinesis:ListStreams",
                  "kinesis:SubscribeToShard",
                ],
                resources: [stream.arn],
              },
            ],
          },
          undefined,
          { parent: self },
        );
      });
    }

    function createEventSourceMapping() {
      return new lambda.EventSourceMapping(
        ...transform(
          args.transform?.eventSourceMapping,
          `${name}EventSourceMapping`,
          {
            eventSourceArn: stream.arn,
            functionName: fn.arn.apply(
              (item) => arn.parseFunction(item).functionName,
            ),
            startingPosition: "LATEST",
            filterCriteria: args.filters && {
              filters: sst.output(args.filters).apply((filters) =>
                filters.map((filter) => ({
                  pattern: JSON.stringify(filter),
                })),
              ),
            },
          },
          { parent: self },
        ),
      );
    }
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    const self = this;
    return {
      /**
       * The Lambda function that'll be notified.
       */
      get function() {
        return self.fn.apply((fn) => fn.getFunction());
      },
      /**
       * The Lambda event source mapping.
       */
      eventSourceMapping: self.eventSourceMapping,
    };
  }
}

const __pulumiType = "sst:aws:KinesisStreamLambdaSubscriber";
// @ts-expect-error
KinesisStreamLambdaSubscriber.__pulumiType = __pulumiType;
