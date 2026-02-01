import { ComponentResourceOptions, Input, output } from "@pulumi/pulumi";
import * as cloudflare from "@pulumi/cloudflare";
import { Component, Transform, transform } from "../component";
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
 * The `QueueWorkerSubscriber` component is internally used by the `Queue` component to
 * add a consumer to [Cloudflare Queues](https://developers.cloudflare.com/queues/).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `Queue` component.
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
            deadLetterQueue: args.deadLetterQueue,
            queueId: queue.id,
            scriptName: worker.script.scriptName,
            settings: args.settings,
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
