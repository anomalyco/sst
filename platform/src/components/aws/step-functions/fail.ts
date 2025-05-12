import { Input } from "../../input";
import { State, StateArgs } from "./state";

export interface FailArgs extends StateArgs {
  /**
   * A custom string that describes the cause of the error.
   * Alternatively, you can specify a JSONata expression that evaluates to a string.
   */
  cause?: Input<string>;
  /**
   * An error name that you can provide to perform error handling using `retry` or `catch`.
   * Alternatively, you can specify a JSONata expression that evaluates to a string.
   */
  error?: Input<string>;
}

/**
 * The `Fail` state is internally used by the `StepFunctions` component to add a [Fail
 * workflow state](https://docs.aws.amazon.com/step-functions/latest/dg/state-fail.html)
 * to a state machine.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `fail` method of the `StepFunctions` component.
 */
export class Fail extends State {
  constructor(protected args: FailArgs) {
    super(args);
  }

  /**
   * Serialize the state into JSON state definition.
   */
  protected toJSON() {
    return {
      Type: "Fail",
      Error: this.args.error,
      Cause: this.args.cause,
      ...super.toJSON(),
      End: undefined,
    };
  }
}
