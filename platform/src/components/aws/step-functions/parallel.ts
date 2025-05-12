import { Input } from "../../input";
import {
  CatchArgs,
  Failable,
  Nextable,
  RetryArgs,
  State,
  StateArgs,
} from "./state";

export interface ParallelArgs extends StateArgs {
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
}

/**
 * The `Parallel` state is internally used by the `StepFunctions` component to add a [Parallel
 * workflow state](https://docs.aws.amazon.com/step-functions/latest/dg/state-parallel.html)
 * to a state machine.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `parallel` method of the `StepFunctions` component.
 */
export class Parallel extends State implements Nextable, Failable {
  private branches: State[] = [];

  constructor(protected args: ParallelArgs) {
    super(args);
  }

  /**
   * Add a branch to the `Parallel` state. Each branch runs concurrently.
   *
   * @param branch The state to add as a branch.
   */
  public branch(branch: State) {
    const head = branch.getHead();
    this.branches.push(head);
    this.addChildGraph(head);
    return this;
  }

  /**
   * Add a next state to the `Parallel` state. If all branches complete successfully,
   * continue execution with the given state.
   *
   * @param state The state to transition to.
   */
  public next<T extends State>(state: T): T {
    return this.addNext(state);
  }

  /**
   * Add retry behavior to the `Parallel` state. If the state fails with any of the
   * specified errors, retry execution using the specified parameters.
   *
   * @param args Optional retry properties to customize retry behavior.
   */
  public retry(args?: RetryArgs) {
    return this.addRetry(args);
  }

  /**
   * Add catch behavior to the `Parallel` state. If the state fails with any of the
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
    return [
      ...this.branches.flatMap((b) => b.getPermissions()),
      ...super.getPermissions(),
    ];
  }

  /**
   * Serialize the state into JSON state definition.
   */
  protected toJSON() {
    if (this.branches.length === 0) {
      throw new Error(
        `The "${this.name}" Parallel state must have at least one branch.`,
      );
    }

    return {
      Type: "Parallel",
      Branches: this.branches.map((b) => {
        return {
          StartAt: b.name,
          States: b.serialize(),
        };
      }),
      ...super.toJSON(),
    };
  }
}
