import { randomBytes } from "crypto";
import { Duration, toSeconds } from "../../duration";
import { Input } from "../../input";
import { FunctionPermissionArgs } from "../function";

export type JSONata = `{% ${string} %}`;

export function isJSONata(value: string) {
  return value.startsWith("{%") && value.endsWith("%}");
}

type DefaultError =
  | "States.ALL"
  | "States.DataLimitExceeded"
  | "States.ExceedToleratedFailureThreshold"
  | "States.HeartbeatTimeout"
  | "States.Http.Socket"
  | "States.IntrinsicFailure"
  | "States.ItemReaderFailed"
  | "States.NoChoiceMatched"
  | "States.ParameterPathFailure"
  | "States.Permissions"
  | "States.ResultPathMatchFailure"
  | "States.ResultWriterFailed"
  | "States.Runtime"
  | "States.TaskFailed"
  | "States.Timeout";

/**
 * @internal
 */
export interface Nextable {
  next: (state: State) => State;
}

/**
 * @internal
 */
export interface Failable {
  retry: (props?: RetryArgs) => State;
  catch: (state: State, props?: CatchArgs) => State;
}

export type RetryArgs = {
  /**
   * The errors that are being retried.
   *
   * @default ["States.ALL"]
   */
  errors?: string[];
  /**
   * The interval between retries in seconds.
   *
   * @default "1 second"
   */
  interval?: Duration;
  /**
   * The maximum number of retries.
   *
   * @default 3
   */
  maxAttempts?: number;
  /**
   * The backoff rate.
   *
   * @default 2
   */
  backoffRate?: number;
};

export type CatchArgs = {
  /**
   * The errors that are being caught.
   *
   * @default ["States.ALL"]
   */
  errors?: string[];
};

export interface StateArgs {
  /**
   * The name of the state.
   */
  name: string;
  /**
   * A comment to describe the state.
   */
  comment?: Input<string>;
  /**
   * Specify and transform output from the state. When specified, the value overrides
   * the state output default.
   *
   * The output field accepts any JSON value (object, array, string, number, boolean, null).
   * Alternatively, you can pass in a JSONata expression directly.
   *
   * For more information, see [Transforming data with JSONata in Step Functions](https://docs.aws.amazon.com/step-functions/latest/dg/transforming-data.html).
   */
  output?: Input<JSONata | Record<string, any>>;
  /**
   * Used to store variables. The Assign field accepts a JSON object with key/value
   * pairs that define variable names and their assigned values. Alternatively, you can
   * pass in a JSONata expression directly.
   *
   * For more information, see [Passing data between states with variables](https://docs.aws.amazon.com/step-functions/latest/dg/workflow-variables.html).
   *
   * @example
   *
   * Provide a JSON object with variable names and values.
   *
   * ```ts
   * {
   *   assign: {
   *     productName: "product1",
   *     count: 42,
   *     available: true,
   *   }
   * }
   * ```
   *
   * Assign values from state input and result using JSONata expressions.
   *
   * ```ts
   *   {
   *     assign: {
   *       product: "{% $states.input.order.product %}",
   *       currentPrice: "{% $states.result.Payload.current_price %}"
   *     }
   *   }
   * ```
   */
  assign?: Record<string, any>;
}

/**
 * The `State` class is the base class for all states in a state machine.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 */
export abstract class State {
  protected _parentGraphState?: State; // only used for Parallel, Map
  protected _childGraphStates: State[] = []; // only used for Parallel, Map
  protected _prevState?: State;
  protected _nextState?: State;
  protected _retries?: RetryArgs[];
  protected _catches?: { next: State; props: CatchArgs }[];

  constructor(protected args: StateArgs) {}

  protected addChildGraph<T extends State>(state: T): T {
    if (state._parentGraphState)
      throw new Error(
        `Cannot reuse the "${state.name}" state. States cannot be reused in Map or Parallel branches.`,
      );

    this._childGraphStates.push(state);
    state._parentGraphState = this;
    return state;
  }

  protected addNext<T extends State>(state: T): T {
    if (this._nextState)
      throw new Error(
        `The "${this.name}" state already has a next state. States cannot have multiple next states.`,
      );

    this._nextState = state;
    state._prevState = this;
    return state;
  }

  protected addRetry(args?: RetryArgs) {
    this._retries = this._retries || [];
    this._retries.push({
      errors: ["States.ALL"],
      backoffRate: 2,
      interval: "1 second",
      maxAttempts: 3,
      ...args,
    });
    return this;
  }

  protected addCatch(state: State, args: CatchArgs = {}) {
    this._catches = this._catches || [];
    this._catches.push({
      next: state,
      props: {
        errors: args.errors ?? ["States.ALL"],
      },
    });
    return this;
  }

  /**
   * @internal
   */
  public get name() {
    return this.args.name;
  }

  /**
   * @internal
   */
  public getRoot(): State {
    return (
      this._prevState?.getRoot() ?? this._parentGraphState?.getRoot() ?? this
    );
  }

  /**
   * @internal
   */
  public getHead(): State {
    return this._prevState?.getHead() ?? this;
  }

  /**
   * Assert that the state name is unique.
   * @internal
   */
  public assertStateNameUnique(states: Map<string, State> = new Map()) {
    const existing = states.get(this.name);
    if (existing && existing !== this)
      throw new Error(
        `Multiple states with the same name "${this.name}". State names must be unique.`,
      );

    states.set(this.name, this);

    this._nextState?.assertStateNameUnique(states);
    this._catches?.forEach((c) => c.next.assertStateNameUnique(states));
    this._childGraphStates.forEach((c) => c.assertStateNameUnique(states));
  }

  /**
   * Assert that the state is not reused.
   * @internal
   */
  public assertStateNotReused(
    states: Map<State, string> = new Map(),
    graphId: string = "main",
  ) {
    const existing = states.get(this);
    if (existing && existing !== graphId)
      throw new Error(
        `Cannot reuse the "${this.name}" state. States cannot be reused in Map or Parallel branches.`,
      );

    states.set(this, graphId);

    this._nextState?.assertStateNotReused(states, graphId);
    this._catches?.forEach((c) => c.next.assertStateNotReused(states, graphId));
    this._childGraphStates.forEach((c) => {
      const childGraphId = randomBytes(16).toString("hex");
      c.assertStateNotReused(states, childGraphId);
    });
  }

  /**
   * Get the permissions required for the state.
   * @internal
   */
  public getPermissions(): FunctionPermissionArgs[] {
    return [
      ...(this._nextState?.getPermissions() || []),
      ...(this._catches || []).flatMap((c) => c.next.getPermissions()),
    ];
  }

  /**
   * Serialize the state into JSON state definition.
   * @internal
   */
  public serialize(): Record<string, any> {
    return {
      [this.name]: this.toJSON(),
      ...this._nextState?.serialize(),
      ...this._catches?.reduce(
        (acc, c) => ({ ...acc, ...c.next.serialize() }),
        {},
      ),
    };
  }

  protected toJSON(): Record<string, any> {
    return {
      QueryLanguage: "JSONata",
      Comment: this.args.comment,
      Output: this.args.output,
      Assign: this.args.assign,
      ...(this._nextState ? { Next: this._nextState.name } : { End: true }),
      Retry: this._retries?.map((r) => ({
        ErrorEquals: r.errors,
        IntervalSeconds: toSeconds(r.interval!),
        MaxAttempts: r.maxAttempts,
        BackoffRate: r.backoffRate,
      })),
      Catch: this._catches?.map((c) => ({
        ErrorEquals: c.props.errors,
        Next: c.next.name,
      })),
    };
  }
}
