import { ComponentResourceOptions, Input, output } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
import {
  DurationMinutes,
  DurationSeconds,
  toMilliseconds,
} from "../duration";
import { WorkerBuilder, workerBuilder } from "./helpers/worker-builder";
import { WorkerArgs } from "./worker";
import { DEFAULT_ACCOUNT_ID } from "./account-id";

export interface QueueWorkerSubscriberArgs {
  /**
   * The queue to use.
   */
  queue: Input<{
    /**
     * The ID of the queue.
     */
    id: Input<string>;
  }>;
  /**
   * The subscriber worker.
   */
  subscriber: Input<string | WorkerArgs>;
  /**
   * The dead letter queue to send messages that fail processing.
   *
   * When `dlq` is configured, `dlq.queue` is required.
   */
  dlq?: {
    /**
     * The name of the dead letter queue.
     */
    queue: Input<string>;
    /**
     * The number of times the main queue will retry the message before sending it to the dead-letter queue.
     * @default `3`
     */
    retry?: Input<number>;
    /**
     * The number of seconds to delay before making the message available for another attempt.
     * @default `0 seconds`
     */
    retryDelay?: Input<DurationSeconds>;
  };
  /**
   * Maximum number of concurrent consumers that may consume from this Queue.
   * @default `null`
   */
  maxConcurrency?: Input<number>;
  /**
   * The maximum number of messages to include in a batch.
   * @default `10`
   */
  batch?: {
    /**
     * The maximum number of events that will be processed together in a single invocation
     * of the consumer function.
     *
     * Value must be between 1 and 100.
     *
     * :::note
     * When `size` is set to a value greater than 10, `window` must be set to at least `1 second`.
     * :::
     *
     * @default `10`
     */
    size?: Input<number>;
    /**
     * The maximum amount of time to wait for collecting events before sending the batch to
     * the consumer function, even if the batch size hasn't been reached.
     *
     * Value must be between 0 seconds and 60 seconds.
     * @default `"5 seconds"`
     */
    window?: Input<DurationMinutes>;
  };
  /**
   * [Transform](/docs/components/#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Worker resource.
     */
    worker?: Transform<WorkerArgs>;
    /**
     * Transform the Consumer resource.
     */
    consumer?: Transform<cloudflare.QueueConsumerArgs>;
  };
}

/**
 * The `QueueWorkerSubscriber` component is internally used by the `Queue` component to
 * add a consumer to [Cloudflare Queues](https://developers.cloudflare.com/queues/).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by `Queue.subscribe()`.
 */
export class QueueWorkerSubscriber extends Component {
  private readonly _worker: WorkerBuilder;
  private readonly consumer: cloudflare.QueueConsumer;

  constructor(
    name: string,
    args: QueueWorkerSubscriberArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const queue = output(args.queue);
    const worker = createWorker();
    const batchSize = output(args.batch?.size ?? 10);
    const window = output(args.batch?.window ?? "5 seconds");
    const retryDelay = output(args.dlq?.retryDelay ?? "0 seconds");
    const consumer = createConsumer();

    this._worker = worker;
    this.consumer = consumer;

    function createWorker() {
      return workerBuilder(
        `${name}Function`,
        args.subscriber,
        args.transform?.worker,
        { parent: self },
      );
    }

    function createConsumer() {
      return new cloudflare.QueueConsumer(
        ...transform(
          args.transform?.consumer,
          `${name}Consumer`,
          {
            accountId: DEFAULT_ACCOUNT_ID,
            deadLetterQueue: args.dlq?.queue,
            queueId: queue.id,
            scriptName: worker.script.scriptName,
            settings: {
              batchSize,
              maxConcurrency: args.maxConcurrency,
              maxRetries: args.dlq?.retry,
              retryDelay: retryDelay.apply((v) => toMilliseconds(v)),
              maxWaitTimeMs: window.apply((v) => toMilliseconds(v)),
            },
            type: "worker",
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
       * The Worker that'll process messages from the queue.
       */
      get worker() {
        return self._worker.apply((worker) => worker.getWorker());
      },
      /**
       * The Cloudflare Queue Consumer.
       */
      consumer: this.consumer,
    };
  }
}

const __pulumiType = "sst:cloudflare:QueueWorkerSubscriber";
// @ts-expect-error
QueueWorkerSubscriber.__pulumiType = __pulumiType;
