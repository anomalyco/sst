import { sqs, s3 } from "@pulumi/aws";
import { ComponentResourceOptions } from "@pulumi/pulumi";
import * as sst from "sst-plugin";
import { transform, Transform } from "sst-plugin/internal/transform";
import { BucketSubscriberArgs } from "./bucket";
import { Queue } from "./queue";

export interface Args extends BucketSubscriberArgs {
  /**
   * The bucket to use.
   */
  bucket: sst.Input<{
    /**
     * The name of the bucket.
     */
    name: sst.Input<string>;
    /**
     * The ARN of the bucket.
     */
    arn: sst.Input<string>;
  }>;
  /**
   * The subscriber ID.
   */
  subscriberId: sst.Input<string>;
  /**
   * The ARN of the SQS Queue.
   */
  queue: sst.Input<string>;
}

/**
 * The `BucketQueueSubscriber` component is internally used by the `Bucket` component
 * to add subscriptions to your [AWS S3 Bucket](https://aws.amazon.com/s3/).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribeQueue` method of the `Bucket` component.
 */
export class BucketQueueSubscriber extends sst.Component {
  private readonly policy: sqs.QueuePolicy;
  private readonly notification: s3.BucketNotification;

  constructor(name: string, args: Args, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const queueArn = sst.output(args.queue);
    const bucket = sst.output(args.bucket);
    const events = args.events
      ? sst.output(args.events)
      : sst.output([
          "s3:ObjectCreated:*",
          "s3:ObjectRemoved:*",
          "s3:ObjectRestore:*",
          "s3:ReducedRedundancyLostObject",
          "s3:Replication:*",
          "s3:LifecycleExpiration:*",
          "s3:LifecycleTransition",
          "s3:IntelligentTiering",
          "s3:ObjectTagging:*",
          "s3:ObjectAcl:Put",
        ]);
    const policy = createPolicy();
    const notification = createNotification();

    this.policy = policy;
    this.notification = notification;

    function createPolicy() {
      return Queue.createPolicy(`${name}Policy`, queueArn);
    }

    function createNotification() {
      return new s3.BucketNotification(
        ...transform(
          args.transform?.notification,
          `${name}Notification`,
          {
            bucket: bucket.name,
            queues: [
              {
                id: sst.interpolate`Notification${args.subscriberId}`,
                queueArn,
                events,
                filterPrefix: args.filterPrefix,
                filterSuffix: args.filterSuffix,
              },
            ],
          },
          { parent: self, dependsOn: [policy] },
        ),
      );
    }
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The SQS Queue policy.
       */
      policy: this.policy,
      /**
       * The S3 Bucket notification.
       */
      notification: this.notification,
    };
  }
}

const __pulumiType = "sst:aws:BucketQueueSubscriber";
// @ts-expect-error
BucketQueueSubscriber.__pulumiType = __pulumiType;
