import { Input } from "../../input";
import {
  CatchArgs,
  Failable,
  JSONata,
  Nextable,
  RetryArgs,
  State,
  StateArgs,
} from "./state";

export interface MapArgs extends StateArgs {
  /**
   * A JSON array or a JSONata expression that must evaluate to an array.
   *
   * Learn more about [Items](https://docs.aws.amazon.com/step-functions/latest/dg/input-output-map.html#input-output-map-items).
   *
   * @example
   * Specify an array of items to process.
   *
   * ```ts
   * {
   *   items: ["item1", "item2", "item3"],
   * }
   * ```
   *
   * Alternatively, specify a JSONata expression to evaluate to an array.
   *
   * ```ts
   * {
   *   items: "{% $states.input.items %}",
   * }
   * ```
   */
  items?: Input<JSONata | any[]>;
  /**
   * Overrides the values of the input array items before they're passed on to each Map
   * state iteration.
   *
   * Learn more about [ItemSelector](https://docs.aws.amazon.com/step-functions/latest/dg/input-output-itemselector.html).
   */
  itemSelector?: Input<Record<string, Input<any>>>;
  /**
   * Specifies an integer value or a JSONata expression that evaluates to an integer. This
   * provides the upper bound on the number of Map state iterations that can run in
   * parallel.
   *
   * The default value is 0, which places no limit on concurrency. Step Functions invokes
   * iterations as concurrently as possible.
   *
   * @default `0`
   * @example
   * Limit the Map state to 10 concurrent iterations running at one time.
   * ```ts
   * {
   *   maxConcurrency: 10,
   * }
   * ```
   */
  maxConcurrency?: Input<JSONata | number>;
  /**
   * The state to execute for each item in the array.
   */
  processor: State;
}

/**
 * The `Map` state is internally used by the `StepFunctions` component to add a [Map
 * workflow state](https://docs.aws.amazon.com/step-functions/latest/dg/state-map.html)
 * to a state machine.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `map` method of the `StepFunctions` component.
 */
export class Map extends State implements Nextable, Failable {
  private processor: State;

  constructor(protected args: MapArgs) {
    super(args);
    this.processor = args.processor.getHead();
    this.addChildGraph(this.processor);
  }

  /**
   * Add a next state to the `Map` state. If the state completes successfully,
   * continue execution with the given state.
   *
   * @param state The state to transition to.
   */
  public next<T extends State>(state: T): T {
    return this.addNext(state);
  }

  /**
   * Add retry behavior to the `Map` state. If the state fails with any of the
   * specified errors, retry execution using the specified parameters.
   *
   * @param args Optional retry properties to customize retry behavior.
   */
  public retry(args?: RetryArgs) {
    return this.addRetry(args);
  }

  /**
   * Add catch behavior to the `Map` state. If the state fails with any of the
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
    return [...this.processor.getPermissions(), ...super.getPermissions()];
  }

  /**
   * Serialize the state into JSON state definition.
   */
  protected toJSON() {
    return {
      Type: "Map",
      Items: this.args.items,
      ItemSelector: this.args.itemSelector,
      ItemProcessor: {
        ProcessorConfig: {
          Mode: "INLINE",
        },
        StartAt: this.processor.name,
        States: this.processor.serialize(),
      },
      MaxConcurrency: this.args.maxConcurrency,
      ...super.toJSON(),
    };
  }
}
