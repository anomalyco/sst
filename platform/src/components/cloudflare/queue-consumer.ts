import { ComponentResourceOptions, Input } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
import { DEFAULT_ACCOUNT_ID } from "./account-id";
import { WorkerBuilder, workerBuilder } from "./helpers/worker-builder";
import { WorkerArgs } from "./worker";

export interface QueueConsumerArgs {
  deadLetterQueue?: Input<string>;
  queueId: Input<string>;
  /**
   * The worker that'll process messages from the queue.
   *
   * @example
   *
   * ```ts
   * {
   *   processor: "consumer.ts"
   * }
   * ```
   *
   * You can pass in the full worker props.
   *
   * ```ts
   * {
   *   processor: {
   *     handler: "consumer.ts",
   *     link: [bucket]
   *   }
   * }
   * ```
   */
  processor: Input<string | WorkerArgs>;
  settings?: cloudflare.QueueConsumerArgs["settings"];
  /**
   * [Transform](/docs/components/#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Consumer resource.
     */
    consumer?: Transform<cloudflare.QueueConsumerArgs>;
  };
}

/**
 * The `Queue Consumer` component lets you add queue consumers to your app using Cloudflare.
 * It uses [Cloudflare Queues](https://developers.cloudflare.com/queues/).
 *
 * @example
 * #### Minimal example
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
 * Create a queue and pass in the consumer worker.
 *
 * ```ts title="sst.config.ts"
 * const queue = new sst.cloudflare.Queue("MyQueue");
 *
 * new sst.cloudflare.QueueConsumer("MyConsumer", {
 *   queueId: queue.id,
 *   processor: "consumer.ts"
 * });
 * ```
 *
 * #### With environment bindings
 *
 * ```ts title="sst.config.ts"
 * const bucket = new sst.cloudflare.R2Bucket("MyBucket");
 * const queue = new sst.cloudflare.Queue("MyQueue");
 *
 * new sst.cloudflare.QueueConsumer("MyConsumer", {
 *   queueId: queue.id,
 *   processor: {
 *     handler: "consumer.ts",
 *     link: [bucket]
 *   }
 * });
 * ```
 */
export class QueueConsumer extends Component {
  private worker: WorkerBuilder;
  private consumer: cloudflare.QueueConsumer;

  constructor(
    name: string,
    args: QueueConsumerArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const parent = this;

    const worker = createWorker();
    const consumer = createConsumer();

    this.worker = worker;
    this.consumer = consumer;

    function createWorker() {
      return workerBuilder(`${name}Handler`, args.processor);
    }

    function createConsumer() {
      return new cloudflare.QueueConsumer(
        ...transform(
          args?.transform?.consumer,
          `${name}QueueConsumer`,
          {
            accountId: DEFAULT_ACCOUNT_ID,
            deadLetterQueue: args.deadLetterQueue,
            queueId: args.queueId,
            scriptName: worker.script.scriptName,
            settings: args.settings,
            type: "worker",
          },
          { parent },
        ),
      );
    }
  }

  /**
   * The generated id of the queue consumer
   */

  public get id() {
    return this.consumer.id;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Cloudflare Worker.
       */
      worker: this.worker.script,
      /**
       * The Cloudflare Queue Consumer.
       */
      consumer: this.consumer,
    };
  }
}

const __pulumiType = "sst:cloudflare:QueueConsumer";
// @ts-expect-error
QueueConsumer.__pulumiType = __pulumiType;
