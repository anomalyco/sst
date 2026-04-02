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
      ErrorData: Object.keys(rest).length ? serializeErrorData(rest) : undefined,
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
    const {
      data,
      message,
      name,
      stack,
      type,
      ...rest
    } = value;

    const hasKnownFields =
      message !== undefined ||
      name !== undefined ||
      type !== undefined ||
      data !== undefined ||
      stack !== undefined;

    return {
      ErrorMessage:
        typeof message === "string" ? message : "Callback failed",
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

/**
 * The `workflow` SDK is available through the following.
 *
 * @example
 * ```ts title="src/workflow.ts"
 * import { workflow } from "sst/aws/workflow";
 * ```
 */
export namespace workflow {
  export type Context<
    TLogger extends durable.DurableLogger = durable.DurableLogger,
  > = durable.DurableContext<TLogger>;
  export type Handler<
    TEvent = any,
    TResult = any,
    TLogger extends durable.DurableLogger = durable.DurableLogger,
  > = durable.DurableExecutionHandler<TEvent, TResult, TLogger>;
  export type Config = durable.DurableExecutionConfig;
  export type Duration = durable.Duration;
  export type StepConfig<TOutput = any> = durable.StepConfig<TOutput>;

  export interface Resource {
    /**
     * The name of the function.
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

  export interface StartInput<TPayload = unknown> {
    /**
     * The unique name for this workflow execution.
     */
    name: string;
    /**
     * The event payload passed to the workflow handler.
     */
    payload?: TPayload;
  }

  export interface SucceedInput<TPayload = unknown> {
    /**
     * The payload to resolve the callback with.
     */
    payload?: TPayload;
  }

  export interface StartInvocationResponse {
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

  export interface ErrorDetails {
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

  export type ErrorInput =
    | Error
    | string
    | ErrorDetails
    | Record<string, unknown>
    | number
    | boolean
    | bigint
    | null
    | undefined;

  export interface FailInput {
    /**
     * The error to reject the callback with. Supports an `Error`, a string,
     * or an object with camelCase fields like `message`, `type`, `data`, and `stack`.
     */
    error: ErrorInput;
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
    /**
     * The full response from the AWS Lambda Invoke API.
     */
    response: StartInvocationResponse;
  }

  export function handler<
    TEvent = any,
    TResult = any,
    TLogger extends durable.DurableLogger = durable.DurableLogger,
  >(input: Handler<TEvent, TResult, TLogger>, config?: Config) {
    return durable.withDurableExecution(input, config);
  }

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
          input.payload === undefined ? undefined : JSON.stringify(input.payload),
      },
      options,
    );
    if (!response.ok) throw new StartError(response);

    return parseStartResponse(response);
  }

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
          input.payload === undefined ? undefined : JSON.stringify(input.payload),
      },
      options,
    );
    if (!response.ok) throw new SucceedError(response);

    return response;
  }

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
}
