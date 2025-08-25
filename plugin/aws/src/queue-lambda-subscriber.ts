import * as sst from "sst-plugin";
import { Transform, transform } from "sst-plugin/internal/transform";
import { VisibleError } from "sst-plugin/error";
import { AWSComponent } from "./component.js";
import { QueueSubscriberArgs } from "./queue.js";
import { lambda } from "@pulumi/aws";
import { toSeconds } from "./util/duration.js";
import { FunctionArgs } from "./function.js";
import { FunctionBuilder, functionBuilder } from "./util/function-builder.js";
import { arn } from "./util/arn.js";

export interface Args extends QueueSubscriberArgs {
  /**
   * The queue to use.
   */
  queue: sst.Input<{
    /**
     * The ARN of the queue.
     */
    arn: sst.Input<string>;
  }>;
  /**
   * The subscriber function.
   */
  subscriber: sst.Input<string | FunctionArgs>;
}

/**
 * The `QueueLambdaSubscriber` component is internally used by the `Queue` component to
 * add a consumer to [Amazon SQS](https://aws.amazon.com/sqs/).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `Queue` component.
 */
export class QueueLambdaSubscriber extends AWSComponent {
  private readonly fn: FunctionBuilder;
  private readonly eventSourceMapping: lambda.EventSourceMapping;

  constructor(name: string, args: Args, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const queue = sst.output(args.queue);
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
                "sqs:ChangeMessageVisibility",
                "sqs:DeleteMessage",
                "sqs:GetQueueAttributes",
                "sqs:GetQueueUrl",
                "sqs:ReceiveMessage",
              ],
              resources: [queue.arn],
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
            functionResponseTypes: sst
              .output(args.batch)
              .apply((batch) =>
                batch?.partialResponses ? ["ReportBatchItemFailures"] : [],
              ),
            batchSize: sst
              .output(args.batch)
              .apply((batch) => batch?.size ?? 10),
            maximumBatchingWindowInSeconds: sst
              .output(args.batch)
              .apply((batch) => (batch?.window ? toSeconds(batch.window) : 0)),
            eventSourceArn: queue.arn,
            functionName: fn.arn.apply(
              (item) => arn.parseFunction(item).functionName,
            ),
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
      eventSourceMapping: this.eventSourceMapping,
    };
  }
}

const __pulumiType = "sst:aws:QueueLambdaSubscriber";
// @ts-expect-error
QueueLambdaSubscriber.__pulumiType = __pulumiType;
