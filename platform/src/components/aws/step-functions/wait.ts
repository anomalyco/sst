import { output } from "@pulumi/pulumi";
import { Duration, toSeconds } from "../../duration";
import { Input } from "../../input";
import { isJSONata, JSONata, Nextable, State, StateArgs } from "./state";

export interface WaitArgs extends StateArgs {
  /**
   * Specify the amount of time to wait before beginning the state specified in the Next
   * field. Alternatively, you can specify a JSONata expression that evaluates to a number
   * in seconds.
   *
   * Must be between 0 and 99999999.
   * @example
   * ```ts
   * {
   *   time: "10 seconds"
   * }
   * ```
   */
  time?: Input<JSONata | Duration>;
  /**
   * The timestamp to wait for.
   *
   * Timestamps must conform to the RFC3339 profile of ISO 8601, with the further
   * restrictions that an uppercase T must separate the date and time portions, and an
   * uppercase Z must denote that a numeric time zone offset is not present.
   *
   * Alternatively, you can use a JSONata expression to evaluate to a timestamp that
   * conforms to the above format.
   *
   * @example
   * ```ts
   * {
   *   timestamp: "2026-01-01T00:00:00Z",
   * }
   * ```
   */
  timestamp?: Input<string>;
}

/**
 * The `Wait` state is internally used by the `StepFunctions` component to add a [Wait
 * workflow state](https://docs.aws.amazon.com/step-functions/latest/dg/state-wait.html)
 * to a state machine.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `wait` method of the `StepFunctions` component.
 */
export class Wait extends State implements Nextable {
  constructor(protected args: WaitArgs) {
    super(args);
  }

  /**
   * Add a next state to the `Wait` state. After the wait completes,
   * continue execution with the given state.
   *
   * @param state The state to transition to.
   */
  public next<T extends State>(state: T): T {
    return this.addNext(state);
  }

  /**
   * Serialize the state into JSON state definition.
   */
  protected toJSON() {
    return {
      Type: "Wait",
      Seconds: this.args.time
        ? output(this.args.time).apply((t) =>
            isJSONata(t) ? t : toSeconds(t as Duration),
          )
        : undefined,
      Timestamp: this.args.timestamp,
      ...super.toJSON(),
    };
  }
}
