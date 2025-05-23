import { lambda, s3 } from "@pulumi/aws";
import * as sst from "sst-plugin";
import { transform, Transform } from "sst-plugin/internal/transform";
import { BucketSubscriberArgs } from "./bucket.js";
import { FunctionArgs } from "./function.js";
import { FunctionBuilder, functionBuilder } from "./util/function-builder.js";

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
   * The subscriber function.
   */
  subscriber: sst.Input<string | FunctionArgs>;
}

/**
 * The `BucketLambdaSubscriber` component is internally used by the `Bucket` component to
 * add bucket notifications to [AWS S3 Bucket](https://aws.amazon.com/s3/).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `Bucket` component.
 */
export class BucketLambdaSubscriber extends sst.Component {
  private readonly fn: FunctionBuilder;
  private readonly permission: lambda.Permission;
  private readonly notification: s3.BucketNotification;

  constructor(name: string, args: Args, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;
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

    const fn = createFunction();
    const permission = createPermission();
    const notification = createNotification();

    this.fn = fn;
    this.permission = permission;
    this.notification = notification;

    function createFunction() {
      return functionBuilder(
        `${name}Function`,
        args.subscriber,
        {
          description: events.apply((events) =>
            events.length < 5
              ? `Subscribed to ${name} on ${events.join(", ")}`
              : `Subscribed to ${name} on ${events
                  .slice(0, 3)
                  .join(", ")}, and ${events.length - 3} more events`,
          ),
        },
        undefined,
        { parent: self },
      );
    }

    function createPermission() {
      return new lambda.Permission(
        `${name}Permission`,
        {
          action: "lambda:InvokeFunction",
          function: fn.arn,
          principal: "s3.amazonaws.com",
          sourceArn: bucket.arn,
        },
        { parent: self },
      );
    }

    function createNotification() {
      return new s3.BucketNotification(
        ...transform(
          args.transform?.notification,
          `${name}Notification`,
          {
            bucket: bucket.name,
            lambdaFunctions: [
              {
                id: sst.interpolate`Notification${args.subscriberId}`,
                lambdaFunctionArn: fn.arn,
                events,
                filterPrefix: args.filterPrefix,
                filterSuffix: args.filterSuffix,
              },
            ],
          },
          { parent: self, dependsOn: [permission] },
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
       * The Lambda permission.
       */
      permission: this.permission,
      /**
       * The S3 bucket notification.
       */
      notification: this.notification,
    };
  }
}

const __pulumiType = "sst:aws:BucketLambdaSubscriber";
// @ts-expect-error
BucketLambdaSubscriber.__pulumiType = __pulumiType;
