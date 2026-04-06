import * as durable from "@aws/durable-execution-sdk-js";
import { aws } from "./client.js";

interface LambdaError {
  ErrorMessage?: string;
  ErrorType?: string;
  ErrorData?: string;
  StackTrace?: string[];
}

async function parseStartResponse(
  response: Response,
): Promise<workflow.StartResponse> {
  const payload = new Uint8Array(await response.arrayBuffer());
  const data: workflow.StartInvocationResponse = {
    StatusCode: response.status,
    FunctionError: response.headers.get("X-Amz-Function-Error") ?? undefined,
    LogResult: response.headers.get("X-Amz-Log-Result") ?? undefined,
    Payload: payload.byteLength ? payload : undefined,
    ExecutedVersion:
      response.headers.get("X-Amz-Executed-Version") ?? undefined,
    DurableExecutionArn:
      response.headers.get("X-Amz-Durable-Execution-Arn") ?? undefined,
  };

  return {
    arn: data.DurableExecutionArn,
    response: data,
    statusCode: data.StatusCode,
    version: data.ExecutedVersion,
  };
}

function serializeErrorData(input: unknown) {
  if (input === undefined) return undefined;
  if (typeof input === "string") return input;
  try {
    return JSON.stringify(input);
  } catch {
    return String(input);
  }
}

function normalizeStack(input: unknown) {
  if (typeof input === "string") {
    return input.split("\n").map((line) => line.trim());
  }
  if (Array.isArray(input)) return input.map(String);
  return undefined;
}

function normalizeError(error: workflow.ErrorInput): LambdaError {
  if (
    error === null ||
    error === undefined ||
    typeof error === "number" ||
    typeof error === "boolean" ||
    typeof error === "bigint"
  ) {
    return {
      ErrorMessage: error === undefined ? "Callback failed" : String(error),
      ErrorType: "Error",
    };
  }

  if (error instanceof Error) {
    const { message, name, stack, ...rest } = error as Error &
      Record<string, unknown>;
    return {
      ErrorMessage: message,
      ErrorType: name,
      ErrorData: Object.keys(rest).length
        ? serializeErrorData(rest)
        : undefined,
      StackTrace: normalizeStack(stack),
    };
  }

  if (typeof error === "string") {
    return {
      ErrorMessage: error,
      ErrorType: "Error",
    };
  }

  if (typeof error === "object") {
    const value = error as Record<string, unknown>;
    const { data, message, name, stack, type, ...rest } = value;

    const hasKnownFields =
      message !== undefined ||
      name !== undefined ||
      type !== undefined ||
      data !== undefined ||
      stack !== undefined;

    return {
      ErrorMessage: typeof message === "string" ? message : "Callback failed",
      ErrorType:
        typeof type === "string"
          ? type
          : typeof name === "string"
            ? name
            : "Error",
      ErrorData:
        data !== undefined
          ? serializeErrorData(data)
          : Object.keys(rest).length
            ? serializeErrorData(rest)
            : hasKnownFields
              ? undefined
              : serializeErrorData(error),
      StackTrace: normalizeStack(stack),
    };
  }

  return {
    ErrorMessage: String(error),
    ErrorType: "Error",
  };
}

const rollbackStateSymbol = Symbol("sst.workflow.rollback.state");

interface RollbackEntry<
  TLogger extends durable.DurableLogger = durable.DurableLogger,
> {
  name: string;
  execute(
    error: unknown,
    context: durable.DurableContext<TLogger>,
  ): Promise<void>;
}

interface RollbackState<
  TLogger extends durable.DurableLogger = durable.DurableLogger,
> {
  undoStack: RollbackEntry<TLogger>[];
}

type WrappedDurableContext<
  TLogger extends durable.DurableLogger = durable.DurableLogger,
> = durable.DurableContext<TLogger> & {
  [rollbackStateSymbol]?: RollbackState<TLogger>;
};

/**
 * The `workflow` SDK is available through the following.
 *
 * SST also adds a few helpers on top of the base AWS durable execution SDK,
 * including `ctx.stepWithRollback()`, `ctx.rollbackAll()`, and `ctx.waitUntil()`.
 *
 * @example
 * ```ts title="src/workflow.ts"
 * import { workflow } from "sst/aws/workflow";
 * ```
 *
 * @example
 * Use `stepWithRollback()` and `rollbackAll()` to register compensating actions.
 *
 * ```ts title="src/workflow.ts"
 * import { workflow } from "sst/aws/workflow";
 *
 * export const handler = workflow.handler(async (_event, ctx) => {
 *   try {
 *     const order = await ctx.stepWithRollback("create-order", {
 *       run: async () => ({ orderId: "order_123" }),
 *       undo: async (_error, result) => {
 *         await fetch(`https://example.com/orders/${result.orderId}`, {
 *           method: "DELETE",
 *         });
 *       },
 *     });
 *
 *     return order;
 *   } catch (error) {
 *     await ctx.rollbackAll(error);
 *     throw error;
 *   }
 * });
 * ```
 *
 * @example
 * Use `waitUntil()` when you already know the exact time the workflow should resume.
 *
 * ```ts title="src/workflow.ts"
 * import { workflow } from "sst/aws/workflow";
 *
 * export const handler = workflow.handler(
 *   async (_event, ctx) => {
 *     const resumeAt = new Date();
 *     resumeAt.setMinutes(resumeAt.getMinutes() + 10);
 *
 *     await ctx.waitUntil("wait-for-follow-up", resumeAt);
 *
 *     return ctx.step("send-follow-up", async () => {
 *       return { delivered: true };
 *     });
 *   },
 * );
 * ```
 */
export namespace workflow {
  interface StepWithRollbackHandler<
    TOutput = any,
    TLogger extends durable.DurableLogger = durable.DurableLogger,
  > {
    /**
     * The durable step to execute.
     */
    run: durable.StepFunc<TOutput, TLogger>;
    /**
     * Called during rollback with the original error, the step result, and step context.
     */
    undo: (
      error: unknown,
      value: TOutput,
      context: Parameters<durable.StepFunc<void, TLogger>>[0],
    ) => Promise<void>;
  }

  interface StepWithRollbackConfig<TOutput = any>
    extends StepConfig<TOutput> {
    /**
     * The config used for the rollback step. Defaults to inheriting the run step's retry strategy
     * and semantics.
     */
    undo?: "inherit" | durable.StepConfig<void>;
  }

  export interface Context<
    TLogger extends durable.DurableLogger = durable.DurableLogger,
  > extends durable.DurableContext<TLogger> {
    /**
     * Execute a durable step and register a compensating rollback step if it succeeds.
     */
    stepWithRollback<TOutput>(
      name: string,
      handler: StepWithRollbackHandler<TOutput, TLogger>,
      config?: StepWithRollbackConfig<TOutput>,
    ): durable.DurablePromise<TOutput>;
    /**
     * Wait until the provided time. Delays are rounded up to the nearest second.
     */
    waitUntil(name: string, until: Date): durable.DurablePromise<void>;
    /**
     * Execute all registered rollback steps in reverse order.
     */
    rollbackAll(error: unknown): Promise<void>;
  }

  export type Handler<
    TEvent = any,
    TResult = any,
    TLogger extends durable.DurableLogger = durable.DurableLogger,
  > = (event: TEvent, context: Context<TLogger>) => Promise<TResult>;
  export type Config = durable.DurableExecutionConfig;
  export type Duration = durable.Duration;
  export type StepConfig<TOutput = any> = durable.StepConfig<TOutput>;

  export interface Resource {
    /**
     * The name of the workflow function.
     */
    name: string;
  }

  export interface Options {
    /**
     * Configure the options for the [aws4fetch](https://github.com/mhart/aws4fetch)
     * [`AWSClient`](https://github.com/mhart/aws4fetch?tab=readme-ov-file#new-awsclientoptions) used internally by the SDK.
     */
    aws?: aws.Options;
  }

  interface StartInput<TPayload = unknown> {
    /**
     * The unique name for this workflow execution.
     */
    name: string;
    /**
     * The event payload passed to the workflow handler.
     */
    payload?: TPayload;
  }

  interface SucceedInput<TPayload = unknown> {
    /**
     * The payload to resolve the callback with.
     */
    payload?: TPayload;
  }

  interface StartInvocationResponse {
    /**
     * The HTTP status code from Lambda.
     */
    StatusCode: number;
    /**
     * The error returned by the function.
     */
    FunctionError?: string;
    /**
     * The execution logs for the invocation.
     */
    LogResult?: string;
    /**
     * The raw payload returned by Lambda.
     */
    Payload?: Uint8Array;
    /**
     * The function version that was executed.
     */
    ExecutedVersion?: string;
    /**
     * The ARN of the durable execution.
     */
    DurableExecutionArn?: string;
  }

  interface ErrorDetails {
    /**
     * Human-readable error message.
     */
    message?: string;
    /**
     * Error type or code.
     */
    type?: string;
    /**
     * Additional machine-readable error data.
     */
    data?: unknown;
    /**
     * Stack trace details.
     */
    stack?: string | string[];
    [key: string]: unknown;
  }

  type ErrorInput =
    | Error
    | string
    | ErrorDetails
    | Record<string, unknown>
    | number
    | boolean
    | bigint
    | null
    | undefined;

  interface FailInput {
    /**
     * The error to reject the callback with. Supports an `Error`, a string,
     * or an object with camelCase fields like `message`, `type`, `data`, and `stack`.
     */
    error: ErrorInput;
  }

  interface StartResponse {
    /**
     * The ARN of the durable execution.
     */
    arn?: string;
    /**
     * The HTTP status code from Lambda.
     */
    statusCode: number;
    /**
     * The function version that was executed.
     */
    version?: string;
    /**
     * The full response from the AWS Lambda Invoke API.
     */
    response: StartInvocationResponse;
  }

  function resolveRunConfig<TOutput>(
    config?: StepWithRollbackConfig<TOutput>,
  ): StepConfig<TOutput> | undefined {
    if (!config) return undefined;

    const runConfig: StepConfig<TOutput> = {};
    if (config.retryStrategy) runConfig.retryStrategy = config.retryStrategy;
    if (config.semantics) runConfig.semantics = config.semantics;
    if (config.serdes) runConfig.serdes = config.serdes;

    return Object.keys(runConfig).length ? runConfig : undefined;
  }

  function resolveUndoConfig<TOutput>(
    config?: StepWithRollbackConfig<TOutput>,
  ): durable.StepConfig<void> | undefined {
    if (!config) return undefined;
    if (config.undo && config.undo !== "inherit") return config.undo;

    const undoConfig: durable.StepConfig<void> = {};
    if (config.retryStrategy) undoConfig.retryStrategy = config.retryStrategy;
    if (config.semantics) undoConfig.semantics = config.semantics;

    return Object.keys(undoConfig).length ? undoConfig : undefined;
  }

  function resolveWaitUntilDuration(until: Date): Duration {
    const timestamp = until.getTime();
    if (!Number.isFinite(timestamp)) {
      throw new TypeError("waitUntil requires a valid Date");
    }

    return {
      seconds: Math.max(0, Math.ceil((timestamp - Date.now()) / 1000)),
    };
  }

  function withRollback<
    TLogger extends durable.DurableLogger = durable.DurableLogger,
  >(context: durable.DurableContext<TLogger>): Context<TLogger> {
    const wrapped = context as WrappedDurableContext<TLogger>;
    if (wrapped[rollbackStateSymbol]) return wrapped as Context<TLogger>;

    const rollbackState: RollbackState<TLogger> = { undoStack: [] };

    wrapped[rollbackStateSymbol] = rollbackState;

    Object.defineProperty(wrapped, "stepWithRollback", {
      configurable: true,
      enumerable: false,
      writable: true,
      value: function <TOutput>(
        name: string,
        handler: StepWithRollbackHandler<TOutput, TLogger>,
        config?: StepWithRollbackConfig<TOutput>,
      ): durable.DurablePromise<TOutput> {
        const runConfig = resolveRunConfig(config);
        const undoConfig = resolveUndoConfig(config);

        return new durable.DurablePromise(async () => {
          const result = await context.step(name, handler.run, runConfig);

          rollbackState.undoStack.push({
            name,
            execute: async (
              error: unknown,
              rollbackContext: durable.DurableContext<TLogger>,
            ) => {
              await rollbackContext.step(
                `Undo '${name}'`,
                (stepContext) => handler.undo(error, result, stepContext),
                undoConfig,
              );
            },
          });

          return result;
        });
      },
    });

    Object.defineProperty(wrapped, "waitUntil", {
      configurable: true,
      enumerable: false,
      writable: true,
      value: (name: string, until: Date): durable.DurablePromise<void> =>
        context.wait(name, resolveWaitUntilDuration(until)),
    });

    Object.defineProperty(wrapped, "rollbackAll", {
      configurable: true,
      enumerable: false,
      writable: true,
      value: async (error: unknown) => {
        while (rollbackState.undoStack.length > 0) {
          const rollbackStep = rollbackState.undoStack.pop();
          if (!rollbackStep) continue;

          try {
            await rollbackStep.execute(error, context);
          } catch (undoError) {
            throw new RollbackError(rollbackStep.name, error, undoError);
          }
        }
      },
    });

    return wrapped as Context<TLogger>;
  }

  /**
   * Create a durable workflow handler.
   *
   * @example
   * ```ts title="src/workflow.ts"
   * import { workflow } from "sst/aws/workflow";
   *
   * export const handler = workflow.handler(
   *   async (_event, ctx) => {
   *     const user = await ctx.step("load-user", async () => {
   *       return { id: "user_123", email: "alice@example.com" };
   *     });
   *
   *     await ctx.wait("pause-before-email", "1 minute");
   *
   *     return ctx.step("send-email", async () => {
   *       return { sent: true, userId: user.id };
   *     });
   *   },
   * );
   * ```
   */
  export function handler<
    TEvent = any,
    TResult = any,
    TLogger extends durable.DurableLogger = durable.DurableLogger,
  >(input: Handler<TEvent, TResult, TLogger>, config?: Config) {
    return durable.withDurableExecution(
      (event: TEvent, context: durable.DurableContext<TLogger>) =>
        input(event, withRollback(context)),
      config,
    );
  }

  /**
   * Start a new workflow execution.
   *
   * This is the equivalent to calling
   * [`Invoke`](https://docs.aws.amazon.com/lambda/latest/api/API_Invoke.html)
   * for a durable Lambda function, using the durable invocation flow described in
   * [Invoking durable Lambda functions](https://docs.aws.amazon.com/lambda/latest/dg/durable-invoking.html).
   */
  export async function start<TPayload = unknown>(
    resource: Resource,
    input: StartInput<TPayload>,
    options?: Options,
  ): Promise<StartResponse> {
    const query = new URLSearchParams({
      Qualifier: "$LATEST",
    });
    const response = await aws.fetch(
      "lambda",
      `/2015-03-31/functions/${encodeURIComponent(
        resource.name,
      )}/invocations?${query.toString()}`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Amz-Durable-Execution-Name": input.name,
          "X-Amz-Invocation-Type": "Event",
        },
        body:
          input.payload === undefined
            ? undefined
            : JSON.stringify(input.payload),
      },
      options,
    );
    if (!response.ok) throw new StartError(response);

    return parseStartResponse(response);
  }

  /**
   * Send a successful result for a pending workflow callback.
   *
   * This is the equivalent to calling
   * [`SendDurableExecutionCallbackSuccess`](https://docs.aws.amazon.com/lambda/latest/api/API_SendDurableExecutionCallbackSuccess.html).
   */
  export async function succeed<TPayload = unknown>(
    token: string,
    input: SucceedInput<TPayload> = {},
    options?: Options,
  ): Promise<Response> {
    const response = await aws.fetch(
      "lambda",
      `/2025-12-01/durable-execution-callbacks/${encodeURIComponent(
        token,
      )}/succeed`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body:
          input.payload === undefined
            ? undefined
            : JSON.stringify(input.payload),
      },
      options,
    );
    if (!response.ok) throw new SucceedError(response);

    return response;
  }

  /**
   * Send a failure result for a pending workflow callback.
   *
   * This is the equivalent to calling
   * [`SendDurableExecutionCallbackFailure`](https://docs.aws.amazon.com/lambda/latest/api/API_SendDurableExecutionCallbackFailure.html).
   */
  export async function fail(
    token: string,
    input: FailInput,
    options?: Options,
  ): Promise<Response> {
    const response = await aws.fetch(
      "lambda",
      `/2025-12-01/durable-execution-callbacks/${encodeURIComponent(
        token,
      )}/fail`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(normalizeError(input.error)),
      },
      options,
    );
    if (!response.ok) throw new FailError(response);

    return response;
  }

  /**
   * Send a heartbeat for a pending workflow callback.
   *
   * This is useful when the external system handling the callback is still doing
   * work and needs to prevent the callback from timing out.
   *
   * This is the equivalent to calling
   * [`SendDurableExecutionCallbackHeartbeat`](https://docs.aws.amazon.com/lambda/latest/api/API_SendDurableExecutionCallbackHeartbeat.html).
   */
  export async function heartbeat(
    token: string,
    options?: Options,
  ): Promise<Response> {
    const response = await aws.fetch(
      "lambda",
      `/2025-12-01/durable-execution-callbacks/${encodeURIComponent(
        token,
      )}/heartbeat`,
      {
        method: "POST",
      },
      options,
    );
    if (!response.ok) throw new HeartbeatError(response);

    return response;
  }

  export class StartError extends Error {
    constructor(public readonly response: Response) {
      super("Failed to start workflow");
    }
  }

  export class SucceedError extends Error {
    constructor(public readonly response: Response) {
      super("Failed to succeed workflow callback");
    }
  }

  export class FailError extends Error {
    constructor(public readonly response: Response) {
      super("Failed to fail workflow callback");
    }
  }

  export class HeartbeatError extends Error {
    constructor(public readonly response: Response) {
      super("Failed to heartbeat workflow callback");
    }
  }

  export class RollbackError extends Error {
    constructor(
      public readonly stepName: string,
      public readonly originalError: unknown,
      public readonly undoError: unknown,
    ) {
      super(
        `Failed to rollback workflow step '${stepName}': ${
          undoError instanceof Error ? undoError.message : String(undoError)
        }`,
      );
      this.name = "RollbackError";
    }
  }
}
