import * as aws from "@pulumi/aws";
import { Input } from "@pulumi/pulumi";

export interface Chainable {
  name: string;
  _prev?: Chainable;
  next: (state: Chainable) => Chainable;
  toJSON: () => object;
  serialize: () => object;
  createPermissions: (role: aws.iam.Role, namePrefix: string) => void;
  firstNode: () => Chainable;
  serializeToDefinition: () => object;
}

function isChainable(obj: unknown): obj is Chainable {
  return (
    typeof obj === "object" && obj !== null && "name" in obj && "next" in obj
  );
}

export enum TaskState {
  ALL = "States.ALL",
}

type ErrorEqual = TaskState | string;

type RetryProps = {
  ErrorEquals: ErrorEqual[];
  IntervalSeconds?: number;
  MaxAttempts?: number;
  BackoffRate?: number;
};
const defaultRetry: RetryProps = {
  ErrorEquals: [TaskState.ALL],
  BackoffRate: 2,
  IntervalSeconds: 1,
  MaxAttempts: 3,
};
export interface Retriable {
  addRetry: (retry?: RetryProps) => Chainable;
}

type CatchProps = {
  ErrorEquals: ErrorEqual[];
  Next: Chainable;
};
export interface Catchable {
  addCatch: (c?: CatchProps) => Chainable;
}

export interface StateBaseParams {
  ResultPath?: Input<string | null>;
  ResultSelector?: Input<object>;
  OutputPath?: Input<string>;
  Comment?: Input<string>;
  InputPath?: Input<string>;
}

export class StateBase implements Chainable {
  private _forceStart?: boolean;
  public _prev?: Chainable;
  public _nextName?: string;
  protected _retries?: RetryProps[];
  protected _catches?: CatchProps[];

  markAsStart() {
    this._forceStart = true;
  }

  addRetry(retry?: Partial<RetryProps>) {
    this._retries = this._retries || [];
    this._retries.push({ ...defaultRetry, ...retry });
    return this;
  }

  addCatch(c: CatchProps | Chainable) {
    this._catches = this._catches || [];
    this._catches.push(
      isChainable(c) ? { ErrorEquals: [TaskState.ALL], Next: c } : c,
    );
    return this;
  }

  constructor(
    public name: string,
    protected params: StateBaseParams = {},
  ) {}

  public next(state: Chainable) {
    this._nextName = state.name;
    state._prev = this;
    return state;
  }

  serialize() {
    const obj: Record<string, unknown> = {
      ...this._prev?.serialize(),
      [this.name]: this.toJSON(),
    };
    for (const c of this._catches || []) {
      obj[c.Next.name] = c.Next.toJSON();
    }
    return obj;
  }

  firstNode(): Chainable {
    if (this._forceStart) return this;
    if (this._prev) {
      return this._prev.firstNode();
    }
    return this;
  }

  serializeToDefinition() {
    return {
      StartAt: this.firstNode().name,
      States: {
        ...this.serialize(),
      },
    };
  }

  toJSON() {
    return {
      ...(this._nextName ? { Next: this._nextName } : { End: true }),
      Retry: this._retries,
      Catch: this._catches?.map((c) => ({
        ErrorEquals: c.ErrorEquals,
        Next: c.Next.name,
      })),
    };
  }

  public createPermissions(role: aws.iam.Role, prefix: string) {
    this._prev?.createPermissions(role, prefix);
    this._catches?.forEach((c) => c.Next.createPermissions(role, prefix));
  }
}
