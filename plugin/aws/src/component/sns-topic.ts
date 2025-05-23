import { sns } from "@pulumi/aws";
import { Transform, transform } from "sst-plugin/internal/transform";
import * as sst from "sst-plugin";
import { AWSComponent } from "../component.js";

export interface SnsTopicArgs {
  /**
   * FIFO (First-In-First-Out) topics are designed to provide strict message ordering.
   *
   * :::caution
   * Changing a standard topic to a FIFO topic or the other way around will result in the destruction and recreation of the topic.
   * :::
   *
   * @default `false`
   * @example
   * ```js
   * {
   *   fifo: true
   * }
   * ```
   */
  fifo?: sst.Input<boolean>;
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the SNS Topic resource.
     */
    topic?: Transform<sns.TopicArgs>;
  };
}

export interface SnsTopicSubscriberArgs {
  /**
   * Filter the messages that'll be processed by the subscriber.
   *
   * If any single property in the filter doesn't match
   * an attribute assigned to the message, then the policy rejects the message.
   *
   * :::tip
   * Learn more about [subscription filter policies](https://docs.aws.amazon.com/sns/latest/dg/sns-subscription-filter-policies.html).
   * :::
   *
   * @example
   * For example, if your SNS Topic message contains this in a JSON format.
   * ```js
   * {
   *   store: "example_corp",
   *   event: "order-placed",
   *   customer_interests: [
   *      "soccer",
   *      "rugby",
   *      "hockey"
   *   ],
   *   price_usd: 210.75
   * }
   * ```
   *
   * Then this filter policy accepts the message.
   *
   * ```js
   * {
   *   filter: {
   *     store: ["example_corp"],
   *     event: [{"anything-but": "order_cancelled"}],
   *     customer_interests: [
   *        "rugby",
   *        "football",
   *        "baseball"
   *     ],
   *     price_usd: [{numeric: [">=", 100]}]
   *   }
   * }
   * ```
   */
  filter?: sst.Input<Record<string, any>>;
  /**
   * [Transform](/docs/components#transform) how this subscription creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the SNS Topic Subscription resource.
     */
    subscription?: Transform<sns.TopicSubscriptionArgs>;
  };
}

interface SnsTopicRef {
  ref: true;
  topicArn: sst.Input<string>;
}

/**
 * The `SnsTopic` component lets you add an [Amazon SNS Topic](https://docs.aws.amazon.com/sns/latest/dg/sns-create-topic.html) to your app.
 *
 * :::note
 * The difference between an `SnsTopic` and a `Queue` is that with a topic you can deliver messages to multiple subscribers.
 * :::
 *
 * @example
 *
 * #### Create a topic
 *
 * ```ts title="sst.config.ts"
 * const topic = new sst.aws.SnsTopic("MyTopic");
 * ```
 *
 * #### Make it a FIFO topic
 *
 * You can optionally make it a FIFO topic.
 *
 * ```ts {2} title="sst.config.ts"
 * new sst.aws.SnsTopic("MyTopic", {
 *   fifo: true
 * });
 * ```
 *
 * #### Add a subscriber
 *
 * ```ts title="sst.config.ts"
 * topic.subscribe("MySubscriber", "src/subscriber.handler");
 * ```
 *
 * #### Link the topic to a resource
 *
 * You can link the topic to other resources, like a function or your Next.js app.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Nextjs("MyWeb", {
 *   link: [topic]
 * });
 * ```
 *
 * Once linked, you can publish messages to the topic from your function code.
 *
 * ```ts title="app/page.tsx" {1,7}
 * import { Resource } from "sst";
 * import { SNSClient, PublishCommand } from "@aws-sdk/client-sns";
 *
 * const sns = new SNSClient({});
 *
 * await sns.send(new PublishCommand({
 *   TopicArn: Resource.MyTopic.arn,
 *   Message: "Hello from Next.js!"
 * }));
 * ```
 */
export class SnsTopic extends AWSComponent {
  private constructorName: string;
  private constructorOpts: sst.ComponentOptions;
  private topic: sns.Topic;

  /**
   * The ARN of the SNS Topic.
   */
  public get arn() {
    return this.topic.arn;
  }

  /**
   * The name of the SNS Topic.
   */
  public get name() {
    return this.topic.name;
  }

  constructor(
    name: string,
    args: SnsTopicArgs = {},
    opts: sst.ComponentOptions = {},
  ) {
    super(__pulumiType, name, args, opts);
    const self = this;
    this.constructorName = name;
    this.constructorOpts = opts;

    if (args && "ref" in args) {
      const ref = reference();
      this.topic = ref.topic;
      return;
    }

    const fifo = normalizeFifo();

    this.topic = createTopic();

    function reference() {
      const ref = args as SnsTopicRef;
      const topic = sns.Topic.get(`${name}Topic`, ref.topicArn, undefined, {
        parent: self,
      });

      return { topic };
    }

    function normalizeFifo() {
      return sst.output(args.fifo).apply((v) => v ?? false);
    }

    function createTopic() {
      return new sns.Topic(
        ...transform(
          args.transform?.topic,
          `${name}Topic`,
          {
            fifoTopic: fifo,
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
    return {
      /**
       * The Amazon SNS Topic.
       */
      topic: this.topic,
    };
  }

  /**
   * Reference an existing SNS topic with its topic ARN. This is useful when you create a
   * topic in one stage and want to share it in another stage. It avoids having to create
   * a new topic in the other stage.
   *
   * :::tip
   * You can use the `static get` method to share SNS topics across stages.
   * :::
   *
   * @param name The name of the component.
   * @param topicArn The ARN of the existing SNS Topic.
   * @param opts? Resource options.
   *
   * @example
   * Imagine you create a topic in the `dev` stage. And in your personal stage `frank`,
   * instead of creating a new topic, you want to share the topic from `dev`.
   *
   * ```ts title="sst.config.ts"
   * const topic = $app.stage === "frank"
   *   ? sst.aws.SnsTopic.get("MyTopic", "arn:aws:sns:us-east-1:123456789012:MyTopic")
   *   : new sst.aws.SnsTopic("MyTopic");
   * ```
   *
   * Here `arn:aws:sns:us-east-1:123456789012:MyTopic` is the ARN of the topic created in
   * the `dev` stage. You can find this by outputting the topic ARN in the `dev` stage.
   *
   * ```ts title="sst.config.ts"
   * return topic.arn;
   * ```
   */
  public static get(
    name: string,
    topicArn: sst.Input<string>,
    opts?: sst.ComponentOptions,
  ) {
    return new SnsTopic(
      name,
      {
        ref: true,
        topicArn,
      } as SnsTopicArgs,
      opts,
    );
  }
}

const __pulumiType = "sst:aws:SnsTopic";
// @ts-expect-error
SnsTopic.__pulumiType = __pulumiType;
