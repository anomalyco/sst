import * as durable from "@aws/durable-execution-sdk-js";
import { aws } from "./client.js";

/**
 * The `workflow` SDK is a thin wrapper around the
 * [`@aws/durable-execution-sdk-js`](https://www.npmjs.com/package/@aws/durable-execution-sdk-js)
 * package and the AWS Lambda durable execution APIs.
 *
 * SST also adds a few helpers on top, including `ctx.stepWithRollback()`,
 * `ctx.rollbackAll()`, and `ctx.waitUntil()`.
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
 *       undo: async (error, result) => {
 *         await fetch(`https://example.com/orders/${result.orderId}`, {
 *           method: "DELETE",
 *         });
 *       },
 *     });
 *
 *     await ctx.step("charge-card", async () => {
 *       throw new Error("Card declined");
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
  export interface Context<
    TLogger extends durable.DurableLogger = durable.DurableLogger,
  > extends durable.DurableContext<TLogger> {
    /**
     * Execute a durable step and register a compensating rollback step if it succeeds.
     * If `run` throws, nothing is added to the rollback stack for that step.
     */
    stepWithRollback<TOutput>(
      name: string,
      handler: StepWithRollbackHandler<TOutput, TLogger>,
      config?: StepConfig<TOutput>,
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
  export type ExecutionStatus =
    | "RUNNING"
    | "SUCCEEDED"
    | "FAILED"
    | "TIMED_OUT"
    | "STOPPED";

  export interface Resource {
    /**
     * The name of the workflow function.
     */
    name: string;
    /**
     * The version or alias qualifier to invoke.
     *
     * Linked `sst.aws.Workflow` resources include this automatically.
     */
    qualifier: string;
  }

  export interface Options {
    /**
     * Configure the options for the [aws4fetch](https://github.com/mhart/aws4fetch)
     * [`AWSClient`](https://github.com/mhart/aws4fetch?tab=readme-ov-file#new-awsclientoptions) used internally by the SDK.
     */
    aws?: aws.Options;
  }

  export interface StartResponse {
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
  }

  export interface Execution {
    /**
     * The ARN of the durable execution.
     */
    arn: string;
    /**
     * The durable execution name.
     */
    name: string;
    /**
     * The ARN of the workflow function.
     */
    functionArn: string;
    /**
     * The current execution status.
     */
    status: ExecutionStatus;
    /**
     * When the execution started.
     */
    createdAt: Date;
    /**
     * When the execution ended, if it has finished.
     */
    endedAt?: Date;
  }

  export interface ListResponse {
    /**
     * The matching executions.
     */
    executions: Execution[];
  }

  export interface DescribeResponse extends Execution {
    /**
     * The version that started the execution.
     */
    version?: string;
  }

  export interface StopResponse {
    /**
     * The ARN of the durable execution.
     */
    arn: string;
    /**
     * The execution status after the stop call.
     */
    status: "STOPPED";
    /**
     * When the execution was stopped.
     */
    stoppedAt?: Date;
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
      Qualifier: resource.qualifier,
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

    return {
      arn: response.headers.get("X-Amz-Durable-Execution-Arn") ?? undefined,
      statusCode: response.status,
      version: response.headers.get("X-Amz-Executed-Version") ?? undefined,
    };
  }

  /**
   * List workflow executions.
   *
   * The SDK returns only the first page of results.
   */
  export async function list(
    resource: Resource,
    query: ListQuery,
    options?: Options,
  ): Promise<ListResponse> {
    const startedAfter = query.createdAt?.from?.toISOString();
    const startedBefore = query.createdAt?.to?.toISOString();
    const direction = query.createdAt?.order ?? "asc";
    const status = query.status;

    if (startedAfter && startedBefore && startedAfter > startedBefore) {
      throw new TypeError(
        "workflow.list createdAt.from must be before createdAt.to",
      );
    }
    if (direction !== "asc" && direction !== "desc") {
      throw new TypeError(
        `Unsupported workflow order direction '${direction}'`,
      );
    }

    const params = new URLSearchParams({
      MaxItems: String(workflowListPageSize),
      Qualifier: resource.qualifier,
    });

    if (Array.isArray(status)) {
      throw new TypeError("workflow.list status must be a single status");
    }
    if (status) params.append("Statuses", status);
    if (startedAfter) params.set("StartedAfter", startedAfter);
    if (startedBefore) params.set("StartedBefore", startedBefore);
    if (direction === "desc") params.set("ReverseOrder", "true");

    const response = await aws.fetch(
      "lambda",
      `/2025-12-01/functions/${encodeURIComponent(
        resource.name,
      )}/durable-executions?${params.toString()}`,
      {
        method: "GET",
      },
      options,
    );
    if (!response.ok) throw new ListError(response);

    const data = (await response.json()) as Partial<ListInvocationResponse>;
    const executions = Array.isArray(data.DurableExecutions)
      ? data.DurableExecutions
      : [];

    return {
      executions: executions.map(parseExecution),
    };
  }

  /**
   * Get the details for a single workflow execution.
   */
  export async function describe(
    arn: string,
    options?: Options,
  ): Promise<DescribeResponse> {
    const response = await aws.fetch(
      "lambda",
      `/2025-12-01/durable-executions/${encodeURIComponent(arn)}`,
      {
        method: "GET",
      },
      options,
    );
    if (!response.ok) throw new DescribeError(response);

    const data = (await response.json()) as Partial<DescribeInvocationResponse>;

    if (
      !data.DurableExecutionArn ||
      !data.DurableExecutionName ||
      !data.FunctionArn ||
      data.StartTimestamp === undefined ||
      data.Status === undefined
    ) {
      throw new DescribeError(response);
    }

    const execution = parseExecution(data as DescribeInvocationResponse);
    return {
      ...execution,
      version: data.Version,
    };
  }

  /**
   * Stop a running workflow execution.
   */
  export async function stop(
    arn: string,
    input?: StopInput,
    options?: Options,
  ): Promise<StopResponse> {
    const response = await aws.fetch(
      "lambda",
      `/2025-12-01/durable-executions/${encodeURIComponent(arn)}/stop`,
      {
        method: "POST",
        headers: input?.error
          ? {
              "Content-Type": "application/json",
            }
          : undefined,
        body:
          input?.error === undefined
            ? undefined
            : JSON.stringify(normalizeError(input.error)),
      },
      options,
    );
    if (!response.ok) throw new StopError(response);

    const data = (await response.json()) as StopInvocationResponse;
    return {
      arn,
      status: "STOPPED",
      stoppedAt:
        data.StopTimestamp === undefined
          ? undefined
          : parseTimestamp(data.StopTimestamp),
    };
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
  ): Promise<void> {
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
  ): Promise<void> {
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
  ): Promise<void> {
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
  }

  export class StartError extends Error {
    constructor(public readonly response: Response) {
      super("Failed to start workflow");
    }
  }

  export class ListError extends Error {
    constructor(public readonly response: Response) {
      super("Failed to list workflows");
    }
  }

  export class DescribeError extends Error {
    constructor(public readonly response: Response) {
      super("Failed to describe workflow");
    }
  }

  export class StopError extends Error {
    constructor(public readonly response: Response) {
      super("Failed to stop workflow");
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

interface DurableError {
  ErrorMessage?: string;
  ErrorType?: string;
  ErrorData?: string;
  StackTrace?: string[];
}

const workflowListPageSize = 1000;

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

function normalizeError(error: unknown): DurableError {
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

  if (error === undefined) {
    return {
      ErrorMessage: "Callback failed",
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

  if (error === null || typeof error !== "object") {
    return {
      ErrorMessage: String(error),
      ErrorType: "Error",
    };
  }

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

interface FailInput {
  /**
   * The error to reject the callback with. Supports an `Error`, a string,
   * or an object with camelCase fields like `message`, `type`, `data`, and `stack`.
   */
  error: unknown;
}

interface StopInput {
  /**
   * The error to reject the callback with. Supports an `Error`, a string,
   * or an object with camelCase fields like `message`, `type`, `data`, and `stack`.
   */
  error?: unknown;
}

interface ListQuery {
  status?: workflow.ExecutionStatus;
  createdAt?: {
    from?: Date;
    to?: Date;
    order?: "asc" | "desc";
  };
}

interface ListInvocationExecution {
  DurableExecutionArn: string;
  DurableExecutionName: string;
  FunctionArn: string;
  Status: workflow.ExecutionStatus;
  StartTimestamp: string | number;
  EndTimestamp?: string | number;
}

interface ListInvocationResponse {
  DurableExecutions: ListInvocationExecution[];
  NextMarker?: string;
}

interface DescribeInvocationResponse {
  DurableExecutionArn: string;
  DurableExecutionName: string;
  FunctionArn: string;
  InputPayload?: string;
  Result?: string;
  Error?: {
    ErrorMessage?: string;
    ErrorType?: string;
    ErrorData?: string;
    StackTrace?: string[];
  };
  StartTimestamp: string | number;
  Status: workflow.ExecutionStatus;
  EndTimestamp?: string | number;
  Version?: string;
  TraceHeader?: {
    XAmznTraceId?: string;
  };
}

interface StopInvocationResponse {
  StopTimestamp?: string | number;
}

function parseExecution(
  execution: ListInvocationExecution | DescribeInvocationResponse,
): workflow.Execution {
  return {
    arn: execution.DurableExecutionArn,
    name: execution.DurableExecutionName,
    functionArn: execution.FunctionArn,
    status: execution.Status,
    createdAt: parseTimestamp(execution.StartTimestamp),
    endedAt:
      execution.EndTimestamp === undefined
        ? undefined
        : parseTimestamp(execution.EndTimestamp),
  };
}

function parseTimestamp(timestamp: string | number): Date {
  const value =
    typeof timestamp === "number" ? timestamp : Number(timestamp);

  if (Number.isFinite(value)) {
    return new Date(value < 1_000_000_000_000 ? value * 1000 : value);
  }

  return new Date(timestamp);
}

function resolveWaitUntilDuration(until: Date): durable.Duration {
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
>(context: durable.DurableContext<TLogger>): workflow.Context<TLogger> {
  const wrapped = context as WrappedDurableContext<TLogger>;
  if (wrapped[rollbackStateSymbol]) return wrapped as workflow.Context<TLogger>;

  const rollbackState: RollbackState<TLogger> = { undoStack: [] };

  wrapped[rollbackStateSymbol] = rollbackState;

  Object.defineProperty(wrapped, "stepWithRollback", {
    configurable: true,
    enumerable: false,
    writable: true,
    value: function <TOutput>(
      name: string,
      handler: StepWithRollbackHandler<TOutput, TLogger>,
      config?: durable.StepConfig<TOutput>,
    ): durable.DurablePromise<TOutput> {
      const undoConfig =
        config?.retryStrategy || config?.semantics
          ? {
              retryStrategy: config.retryStrategy,
              semantics: config.semantics,
            }
          : undefined;

      return new durable.DurablePromise(async () => {
        const result = await context.step(name, handler.run, config);

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
          throw new workflow.RollbackError(rollbackStep.name, error, undoError);
        }
      }
    },
  });

  return wrapped as workflow.Context<TLogger>;
}
