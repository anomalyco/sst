import * as sst from "sst-plugin";
import { transform, Transform } from "sst-plugin/internal/transform";
import { AWSComponent } from "../component.js";
import { permission } from "../permission.js";
import { lambda } from "@pulumi/aws";
import { DynamoSubscriberArgs } from "./dynamo.js";
import { FunctionArgs } from "./function.js";
import { FunctionBuilder, functionBuilder } from "./util/function-builder.js";
import { arn } from "../arn.js";

export interface Args extends DynamoSubscriberArgs {
  /**
   * The DynamoDB table to use.
   */
  dynamo: sst.Input<{
    /**
     * The ARN of the stream.
     */
    streamArn: sst.Input<string>;
  }>;
  /**
   * The subscriber function.
   */
  subscriber: sst.Input<string | FunctionArgs>;
  /**
   * In early versions of SST, parent were forgotten to be set for resources in components.
   * This flag is used to disable the automatic setting of the parent to prevent breaking
   * changes.
   * @internal
   */
  disableParent?: boolean;
}

/**
 * The `DynamoLambdaSubscriber` component is internally used by the `Dynamo` component to
 * add stream subscriptions to [Amazon DynamoDB](https://aws.amazon.com/dynamodb/).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `Dynamo` component.
 */
export class DynamoLambdaSubscriber extends AWSComponent {
  private readonly fn: FunctionBuilder;
  private readonly eventSourceMapping: lambda.EventSourceMapping;

  constructor(name: string, args: Args, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const dynamo = sst.output(args.dynamo);
    const fn = createFunction();
    const eventSourceMapping = createEventSourceMapping();

    this.fn = fn;
    this.eventSourceMapping = eventSourceMapping;

    function createFunction() {
      return functionBuilder(
        `${name}Function`,
        args.subscriber,
        {
          description: `Subscribed to ${name}`,
          permissions: [
            {
              actions: [
                "dynamodb:DescribeStream",
                "dynamodb:GetRecords",
                "dynamodb:GetShardIterator",
                "dynamodb:ListStreams",
              ],
              resources: [dynamo.streamArn],
            },
          ],
        },
        undefined,
        { parent: self },
      );
    }

    function createEventSourceMapping() {
      return new lambda.EventSourceMapping(
        ...transform(
          args.transform?.eventSourceMapping,
          `${name}EventSourceMapping`,
          {
            eventSourceArn: dynamo.streamArn,
            functionName: fn.arn.apply(
              (item) => arn.parseFunction(item).functionName,
            ),
            filterCriteria: args.filters
              ? sst.output(args.filters).apply((filters) => ({
                  filters: filters.map((filter) => ({
                    pattern: JSON.stringify(filter),
                  })),
                }))
              : undefined,
            startingPosition: "LATEST",
          },
          { parent: args.disableParent ? undefined : self },
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
      eventSourceMapping: this.eventSourceMapping,
    };
  }
}

const __pulumiType = "sst:aws:DynamoLambdaSubscriber";
// @ts-expect-error
DynamoLambdaSubscriber.__pulumiType = __pulumiType;
