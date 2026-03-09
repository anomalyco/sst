import { ComponentResourceOptions, Input } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
import { Link } from "../link";
import { binding } from "./binding";
import { DEFAULT_ACCOUNT_ID } from "./account-id";
import { WorkerArgs } from "./worker";
import { VisibleError } from "../error";
import { QueueWorkerSubscriber } from "./queue-worker-subscriber";
import { DurationMinutes, DurationSeconds } from "../duration";

export interface QueueArgs {
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
   * [Transform](/docs/components/#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Queue resource.
     */
    queue?: Transform<cloudflare.QueueArgs>;
  };
}

export interface QueueSubscribeArgs {
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
     * @example
     * Set batch size to 1. This will process events individually.
     * ```js
     * {
     *   batch: {
     *     size: 1
     *   }
     * }
     * ```
     */
    size?: Input<number>;
    /**
     * The maximum amount of time to wait for collecting events before sending the batch to
     * the consumer function, even if the batch size hasn't been reached.
     *
     * Value must be between 0 seconds and 60 seconds.
     * @default `"5 seconds"`
     * @example
     * ```js
     * {
     *   batch: {
     *     window: "5 seconds"
     *   }
     * }
     * ```
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
 * The `Queue` component lets you add a [Cloudflare Queue](https://developers.cloudflare.com/queues/) to
 * your app.
 *
 * @example
 * #### Create a Queue
 *
 * ```ts title="sst.config.ts"
 * const queue = new sst.cloudflare.Queue("MyQueue");
 * ```
 *
 * #### Subscribe to the Queue
 *
 * Create a worker file that exposes a default handler for queue messages:
 *
 * ```ts title="consumer.ts"
 * export default {
 *   async queue(batch, env) {
 *     for (const message of batch.messages) {
 *       console.log("Processing message:", message.body);
 *     }
 *   },
 * };
 * ```
 *
 * Subscribe to the queue with a consumer worker.
 *
 * ```ts title="sst.config.ts"
 * queue.subscribe("consumer.ts");
 * ```
 *
 * #### Link to the Queue
 *
 * You can link other workers to the queue.
 *
 * ```ts title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "producer.ts",
 *   link: [queue],
 *   url: true,
 * });
 * ```
 *
 * #### Subscribe with full worker props
 *
 * ```ts title="sst.config.ts"
 * const bucket = new sst.cloudflare.Bucket("MyBucket");
 *
 * queue.subscribe({
 *   handler: "consumer.ts",
 *   link: [bucket],
 * });
 * ```
 */
export class Queue extends Component implements Link.Linkable {
  private queue: cloudflare.Queue;
  private isSubscribed = false;
  private constructorName: string;
  private constructorArgs?: QueueArgs;
  private constructorOpts?: ComponentResourceOptions;

  constructor(name: string, args?: QueueArgs, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const parent = this;
    this.constructorName = name;
    this.constructorArgs = args;
    this.constructorOpts = opts;

    const queue = create();

    this.queue = queue;

    function create() {
      return new cloudflare.Queue(
        ...transform(
          args?.transform?.queue,
          `${name}Queue`,
          {
            queueName: "",
            accountId: DEFAULT_ACCOUNT_ID,
          },
          { parent },
        ),
      );
    }
  }

  /**
   * Subscribe to the queue with a worker.
   *
   * @param subscriber The worker that'll process messages from the queue.
   * @param args Configure the subscription.
   * @param opts Component resource options.
   *
   * @example
   *
   * Subscribe to the queue with a worker file.
   *
   * ```ts title="sst.config.ts"
   * queue.subscribe("consumer.ts");
   * ```
   *
   * Pass in full worker props.
   *
   * ```ts title="sst.config.ts"
   * const bucket = new sst.cloudflare.Bucket("MyBucket");
   *
   * queue.subscribe({
   *   handler: "consumer.ts",
   *   link: [bucket],
   * });
   * ```
   *
   * Configure batch settings.
   *
   * ```ts title="sst.config.ts"
   * queue.subscribe("consumer.ts", {
   *   batch: {
   *     size: 10,
   *     window: "20 seconds",
   *   },
   * });
   * ```
   */
  public subscribe(
    subscriber: Input<string | WorkerArgs>,
    args?: QueueSubscribeArgs,
    opts?: ComponentResourceOptions,
  ) {
    if (this.isSubscribed) {
      throw new VisibleError(
        `Cannot subscribe to the "${this.constructorName}" queue multiple times. A Cloudflare Queue can only have one consumer.`,
      );
    }

    this.isSubscribed = true;

    const parent = this;
    const name = this.constructorName;

    return new QueueWorkerSubscriber(
      `${name}Subscriber`,
      {
        queue: { id: this.queue.id },
        subscriber,
        dlq: this.constructorArgs?.dlq,
        maxConcurrency: this.constructorArgs?.maxConcurrency,
        batch: args?.batch,
        transform: args?.transform,
      },
      { ...opts, provider: this.constructorOpts?.provider },
    );
  }

  getSSTLink() {
    return {
      properties: {},
      include: [
        binding({
          type: "queueBindings",
          properties: {
            queueName: this.queue.queueName,
          },
        }),
      ],
    };
  }

  /**
   * The generated id of the queue
   */
  public get id() {
    return this.queue.id;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Cloudflare queue.
       */
      queue: this.queue,
    };
  }
}

const __pulumiType = "sst:cloudflare:Queue";
// @ts-expect-error
Queue.__pulumiType = __pulumiType;
