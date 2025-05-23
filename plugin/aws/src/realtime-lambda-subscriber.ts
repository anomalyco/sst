import * as sst from "sst-plugin";
import { Transform, transform } from "sst-plugin/internal/transform";
import { VisibleError } from "sst-plugin/error";
import { AWSComponent } from "./component.js";
import { lambda, iot } from "@pulumi/aws";
import { FunctionArgs } from "./function.js";
import { RealtimeSubscriberArgs } from "./realtime.js";
import { FunctionBuilder, functionBuilder } from "./util/function-builder.js";
import { arn } from "./util/arn.js";

export interface Args extends RealtimeSubscriberArgs {
  /**
   * The IoT WebSocket server to use.
   */
  iot: sst.Input<{
    /**
     * The name of the Realtime component.
     */
    name: sst.Input<string>;
  }>;
  /**
   * The subscriber function.
   */
  subscriber: sst.Input<string | FunctionArgs>;
}

/**
 * The `RealtimeLambdaSubscriber` component is internally used by the `Realtime` component
 * to add subscriptions to the [AWS IoT endpoint](https://docs.aws.amazon.com/iot/latest/developerguide/what-is-aws-iot.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `subscribe` method of the `Realtime` component.
 */
export class RealtimeLambdaSubscriber extends sst.Component {
  private readonly fn: FunctionBuilder;
  private readonly permission: lambda.Permission;
  private readonly rule: iot.TopicRule;

  constructor(name: string, args: Args, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const normalizedIot = sst.output(args.iot);
    const filter = sst.output(args.filter);
    const fn = createFunction();
    const rule = createRule();
    const permission = createPermission();

    this.fn = fn;
    this.permission = permission;
    this.rule = rule;

    function createFunction() {
      return functionBuilder(
        `${name}Handler`,
        args.subscriber,
        {
          description: sst.interpolate`Subscribed to ${normalizedIot.name} on ${filter}`,
        },
        undefined,
        { parent: self },
      );
    }

    function createRule() {
      return new iot.TopicRule(
        ...transform(
          args?.transform?.topicRule,
          `${name}Rule`,
          {
            sqlVersion: "2016-03-23",
            sql: sst.interpolate`SELECT * FROM '${filter}'`,
            enabled: true,
            lambdas: [{ functionArn: fn.arn }],
          },
          { parent: self },
        ),
      );
    }

    function createPermission() {
      return new lambda.Permission(
        `${name}Permission`,
        {
          action: "lambda:InvokeFunction",
          function: fn.arn.apply((val) => arn.parseFunction(val).functionName),
          principal: "iot.amazonaws.com",
          sourceArn: rule.arn,
        },
        { parent: self },
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
       * The IoT Topic rule.
       */
      rule: this.rule,
    };
  }
}

const __pulumiType = "sst:aws:RealtimeLambdaSubscriber";
// @ts-expect-error
RealtimeLambdaSubscriber.__pulumiType = __pulumiType;
