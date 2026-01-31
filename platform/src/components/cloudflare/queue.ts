import { ComponentResourceOptions, Input, output } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
import { Link } from "../link";
import { binding } from "./binding";
import { DEFAULT_ACCOUNT_ID } from "./account-id";
import { WorkerBuilder, workerBuilder } from "./helpers/worker-builder";
import { WorkerArgs } from "./worker";

export interface QueueArgs {
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
   * The dead letter queue to send messages that fail processing.
   */
  deadLetterQueue?: Input<string>;
  /**
   * Consumer settings like batch size, retries, and visibility timeout.
   */
  settings?: cloudflare.QueueConsumerArgs["settings"];
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

  constructor(name: string, args?: QueueArgs, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const parent = this;
    this.constructorName = name;

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
   * Configure consumer settings.
   *
   * ```ts title="sst.config.ts"
   * queue.subscribe("consumer.ts", {
   *   settings: {
   *     batchSize: 10,
   *     maxRetries: 3,
   *   },
   * });
   * ```
   *
   * Configure a dead letter queue.
   *
   * ```ts title="sst.config.ts"
   * const dlq = new sst.cloudflare.Queue("DeadLetterQueue");
   *
   * queue.subscribe("consumer.ts", {
   *   deadLetterQueue: dlq.id,
   * });
   * ```
   */
  public subscribe(
    subscriber: Input<string | WorkerArgs>,
    args?: QueueSubscribeArgs,
    opts?: ComponentResourceOptions,
  ) {
    if (this.isSubscribed) {
      throw new Error(
        `Cannot subscribe to the "${this.constructorName}" queue multiple times. A Cloudflare Queue can only have one consumer.`,
      );
    }
    this.isSubscribed = true;

    const parent = this;
    const name = this.constructorName;

    return output(subscriber).apply((subscriber) => {
      const worker = workerBuilder(
        `${name}Subscriber`,
        subscriber,
        args?.transform?.worker,
        { parent, ...opts },
      );

      const consumer = new cloudflare.QueueConsumer(
        ...transform(
          args?.transform?.consumer,
          `${name}Consumer`,
          {
            accountId: DEFAULT_ACCOUNT_ID,
            deadLetterQueue: args?.deadLetterQueue,
            queueId: this.queue.id,
            scriptName: worker.script.scriptName,
            settings: args?.settings,
            type: "worker",
          },
          { parent, ...opts },
        ),
      );

      return { worker, consumer };
    });
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
