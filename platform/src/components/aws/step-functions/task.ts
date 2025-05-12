import { output } from "@pulumi/pulumi";
import { Duration, toSeconds } from "../../duration";
import { Input } from "../../input";
import { Function, FunctionPermissionArgs } from "../function";
import {
  CatchArgs,
  Failable,
  isJSONata,
  JSONata,
  Nextable,
  RetryArgs,
  State,
  StateArgs,
} from "./state";
import { SnsTopic } from "../sns-topic";
import { Queue } from "../queue";
import { Task as ServiceTask } from "../task";
import { Bus } from "../bus";

export interface TaskBaseArgs extends StateArgs {
  /**
   * Specifies a target role the state machine's execution role must assume before invoking the specified resource.
   * See [Task state's Credentials field](https://docs.aws.amazon.com/step-functions/latest/dg/state-task.html#task-state-example-credentials) examples.
   *
   * @example
   *
   * ```ts
   * {
   *   role: "arn:aws:iam::123456789012:role/MyRole"
   * }
   * ```
   */
  role?: Input<string>;
  /**
   * Specifies the maximum time a task can run before it times out with the
   * `States.Timeout` error and fails. Alternatively, you can specify a JSONata
   * expression that evaluates to a number in seconds.
   *
   * @default `"60 seconds"` for HTTP tasks, `"99999999 seconds"` for all other tasks.
   */
  timeout?: Input<JSONata | Duration>;
}

export interface TaskArgs extends TaskBaseArgs {
  resource: Input<string>;
  /**
   * Used to pass information to the API actions of connected resources. Values can include JSONata expressions. For more information, see [Transforming data with JSONata in Step Functions](https://docs.aws.amazon.com/step-functions/latest/dg/transforming-data.html).
   *
   * @example
   *
   * ```ts
   * {
   *   arguments: {
   *     product: "{% $states.input.order.product %}",
   *     count: 32
   *   }
   * }
   * ```
   */
  arguments?: Input<Record<string, Input<any>>>;
  permissions?: FunctionPermissionArgs[];
}

/**
 * The `Task` state is internally used by the `StepFunctions` component to add a [Task
 * workflow state](https://docs.aws.amazon.com/step-functions/latest/dg/state-task.html)
 * to a state machine.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by task-specific methods of the `StepFunctions` component
 * such as `lambdaInvoke`, `snsPublish`, `sqsSendMessage`, and more.
 */
export class Task extends State implements Nextable, Failable {
  constructor(protected args: TaskArgs) {
    super(args);
  }

  /**
   * Add a next state to the `Task` state. If the state completes successfully,
   * continue execution with the given state.
   *
   * @param state The state to transition to.
   */
  public next<T extends State>(state: T): T {
    return this.addNext(state);
  }

  /**
   * Add retry behavior to the `Task` state. If the state fails with any of the
   * specified errors, retry execution using the specified parameters.
   *
   * @param args Optional retry properties to customize retry behavior.
   */
  public retry(args?: RetryArgs) {
    return this.addRetry(args);
  }

  /**
   * Add catch behavior to the `Task` state. If the state fails with any of the
   * specified errors, continue execution with the given state.
   *
   * @param state The state to transition to on error.
   * @param args Optional catch properties to customize error handling.
   */
  public catch(state: State, args: CatchArgs = {}) {
    return this.addCatch(state, args);
  }

  /**
   * @internal
   */
  public getPermissions() {
    return [...(this.args.permissions || []), ...super.getPermissions()];
  }

  /**
   * Serialize the state into JSON state definition.
   */
  protected toJSON() {
    return {
      Type: "Task",
      ...super.toJSON(),
      Resource: this.args.resource,
      Credentials: this.args.role && {
        RoleArn: this.args.role,
      },
      Timeout: this.args.timeout
        ? output(this.args.timeout).apply((t) =>
            isJSONata(t) ? t : toSeconds(t as Duration),
          )
        : undefined,
      Arguments: this.args.arguments,
    };
  }
}

export interface LambdaInvokeArgs extends TaskBaseArgs {
  /**
   * The `Function` component to invoke.
   */
  function: Function;
  /**
   * The payload to send to the Lambda function.
   */
  payload?: Record<string, Input<unknown>>;
}

export interface SnsPublishArgs extends TaskBaseArgs {
  /**
   * The `SnsTopic` component to publish the message to.
   */
  topic: SnsTopic;
  /**
   * The message to send to the SNS topic.
   */
  message: Input<string>;
  /**
   * The message attributes to send to the SNS topic.
   */
  messageAttributes?: Input<Record<string, Input<string>>>;
  /**
   * The message deduplication ID to send to the SNS topic.
   */
  messageDeduplicationId?: Input<string>;
  /**
   * The message group ID to send to the SNS topic.
   */
  messageGroupId?: Input<string>;
  /**
   * The subject of the message to send to the SNS topic.
   */
  subject?: Input<string>;
}

export interface SqsSendMessageArgs extends TaskBaseArgs {
  /**
   * The `Queue` component to send the message to.
   */
  queue: Queue;
  /**
   * The message body to send to the SQS queue.
   */
  messageBody: Input<string>;
  /**
   * The message attributes to send to the SQS queue.
   */
  messageAttributes?: Input<Record<string, Input<string>>>;
  /**
   * The message deduplication ID to send to the SQS queue.
   */
  messageDeduplicationId?: Input<string>;
  /**
   * The message group ID to send to the SQS queue.
   */
  messageGroupId?: Input<string>;
}

export interface EcsRunTaskArgs extends TaskBaseArgs {
  /**
   * The `Task` component to run.
   */
  task: ServiceTask;
  /**
   * The environment variables to apply to the ECS task.
   * @example
   *
   * ```ts
   * {
   *   environment: {
   *     MY_ENV: "{% $states.input.foo %}",
   *   },
   * }
   * ```
   */
  environment?: Input<Record<string, Input<string>>>;
}

export interface EventBridgePutEventsArgs extends TaskBaseArgs {
  /**
   * The events to put to the EventBridge event bus.
   *
   * @example
   * ```ts
   * {
   *   events: [
   *     {
   *       bus: myBus,
   *       source: "my-application",
   *       detailType: "order-created",
   *       detail: {
   *         orderId: "{% $states.input.orderId %}",
   *         customerId: "{% $states.input.customer.id %}",
   *         items: "{% $states.input.items %}"
   *       }
   *     }
   *   ]
   * }
   * ```
   */
  events: {
    /**
     * The `Bus` component to send the event to.
     */
    bus: Bus;
    /**
     * The source of the event, which identifies the service or component that generated it.
     * Can be a static string or a JSONata expression.
     */
    source?: Input<string>;
    /**
     * The detail type of the event, which helps subscribers filter and route events.
     * Can be a static string or a JSONata expression.
     */
    detailType?: Input<string>;
    /**
     * The event payload containing the event details as a JSON object.
     * Values can include JSONata expressions for dynamic content.
     */
    detail?: Input<Record<string, Input<unknown>>>;
  }[];
}
