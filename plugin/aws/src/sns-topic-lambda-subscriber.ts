import * as sst from "sst-plugin";
import { transform } from "sst-plugin/internal/transform";
import { lambda, sns } from "@pulumi/aws";
import { FunctionArgs } from "./function";
import { SnsTopicSubscriberArgs } from "./sns-topic";
import { FunctionBuilder, functionBuilder } from "./util/function-builder";

export interface Args extends SnsTopicSubscriberArgs {
  /**
   * The Topic to use.
   */
  topic: sst.Input<{
    /**
     * The ARN of the Topic.
     */
    arn: sst.Input<string>;
  }>;
  /**
   * The subscriber function.
   */
  subscriber: sst.Input<string | FunctionArgs>;
}

/**
 * The `SnsTopicLambdaSubscriber` component is internally used by the `SnsTopic` component
 * to add subscriptions to your [Amazon SNS Topic](https://docs.aws.amazon.com/sns/latest/dg/sns-create-topic.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `SnsTopic` component.
 */
export class SnsTopicLambdaSubscriber extends sst.Component {
  private readonly fn: FunctionBuilder;
  private readonly permission: lambda.Permission;
  private readonly subscription: sns.TopicSubscription;

  constructor(name: string, args: Args, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const topic = sst.output(args.topic);
    const fn = createFunction();
    const permission = createPermission();
    const subscription = createSubscription();

    this.fn = fn;
    this.permission = permission;
    this.subscription = subscription;

    function createFunction() {
      return functionBuilder(
        `${name}Function`,
        args.subscriber,
        {
          description: `Subscribed to ${name}`,
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
          principal: "sns.amazonaws.com",
          sourceArn: topic.arn,
        },
        { parent: self },
      );
    }

    function createSubscription() {
      return new sns.TopicSubscription(
        ...transform(
          args?.transform?.subscription,
          `${name}Subscription`,
          {
            topic: topic.arn,
            protocol: "lambda",
            endpoint: fn.arn,
            filterPolicy: args.filter && sst.json.stringify(args.filter),
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
       * The SNS Topic subscription.
       */
      subscription: this.subscription,
    };
  }
}

const __pulumiType = "sst:aws:SnsTopicLambdaSubscriber";
// @ts-expect-error
SnsTopicLambdaSubscriber.__pulumiType = __pulumiType;
