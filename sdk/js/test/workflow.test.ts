import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it } from "bun:test";
import {
  ExecutionStatus,
  LocalDurableTestRunner,
} from "@aws/durable-execution-sdk-js-testing";
import { aws } from "../src/aws/client.ts";
import { workflow } from "../src/aws/workflow.ts";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: {
      "Content-Type": "application/json",
    },
    status: 200,
    ...init,
  });
}

describe("workflow client", () => {
  const originalFetch = aws.fetch;

  beforeEach(() => {
    aws.fetch = originalFetch;
  });

  afterEach(() => {
    aws.fetch = originalFetch;
  });

  it("starts a workflow execution", async () => {
    aws.fetch = async (service, path, init) => {
      expect(service).toBe("lambda");
      expect(path).toBe(
        "/2015-03-31/functions/my-workflow/invocations?Qualifier=live",
      );
      expect(init.method).toBe("POST");
      expect(init.headers).toEqual({
        "Content-Type": "application/json",
        "X-Amz-Durable-Execution-Name": "job-123",
        "X-Amz-Invocation-Type": "Event",
      });
      expect(JSON.parse(String(init.body))).toEqual({ hello: "world" });

      return new Response(undefined, {
        status: 202,
        headers: {
          "X-Amz-Durable-Execution-Arn": "arn:workflow:live/job-123",
          "X-Amz-Executed-Version": "12",
        },
      });
    };

    const result = await workflow.start(
      { name: "my-workflow", qualifier: "live" },
      {
        name: "job-123",
        payload: { hello: "world" },
      },
    );

    expect(result).toEqual({
      arn: "arn:workflow:live/job-123",
      statusCode: 202,
      version: "12",
    });
    expect("response" in result).toBe(false);
  });

  it("lists first-page executions", async () => {
    const requests: {
      service: string;
      path: string;
      init: RequestInit;
    }[] = [];

    aws.fetch = async (service, path, init) => {
      requests.push({ service, path, init });
      const url = new URL(`https://example.com${path}`);
      const status = url.searchParams.getAll("Statuses")[0];

      if (status === "FAILED") {
        return jsonResponse({
          DurableExecutions: [
            {
              DurableExecutionArn: "arn:failed-2",
              DurableExecutionName: "failed-2",
              FunctionArn: "arn:function:live",
              Status: "FAILED",
              StartTimestamp: "2026-04-03T00:00:00.000Z",
              EndTimestamp: "2026-04-03T00:05:00.000Z",
            },
            {
              DurableExecutionArn: "arn:failed-1",
              DurableExecutionName: "failed-1",
              FunctionArn: "arn:function:live",
              Status: "FAILED",
              StartTimestamp: "2026-04-02T00:00:00.000Z",
              EndTimestamp: "2026-04-02T00:05:00.000Z",
            },
          ],
          NextMarker: "ignored-page-2",
        });
      }

      throw new Error(`Unexpected request: ${path}`);
    };

    const result = await workflow.list(
      { name: "my-workflow", qualifier: "live" },
      {
        status: "FAILED",
        createdAt: {
          from: new Date("2026-04-01T00:00:00.000Z"),
          to: new Date("2026-04-04T00:00:00.000Z"),
          order: "desc",
        },
      },
    );

    expect(requests).toHaveLength(1);
    expect(requests.map((request) => request.service)).toEqual(["lambda"]);

    const firstUrl = new URL(`https://example.com${requests[0].path}`);
    expect(firstUrl.searchParams.get("Qualifier")).toBe("live");
    expect(firstUrl.searchParams.get("MaxItems")).toBe("1000");
    expect(firstUrl.searchParams.get("StartedAfter")).toBe(
      "2026-04-01T00:00:00.000Z",
    );
    expect(firstUrl.searchParams.get("StartedBefore")).toBe(
      "2026-04-04T00:00:00.000Z",
    );
    expect(firstUrl.searchParams.get("ReverseOrder")).toBe("true");
    expect(firstUrl.searchParams.getAll("Statuses")).toEqual(["FAILED"]);
    expect(requests[0].init.method).toBe("GET");

    expect(firstUrl.searchParams.get("Marker")).toBeNull();

    expect(result.executions.map((execution) => execution.arn)).toEqual([
      "arn:failed-2",
      "arn:failed-1",
    ]);
    expect(result.executions[0]?.status).toBe("FAILED");
    expect(result.executions[0]?.endedAt?.toISOString()).toBe(
      "2026-04-03T00:05:00.000Z",
    );
    expect("response" in result).toBe(false);
  });

  it("rejects multiple statuses", async () => {
    let calls = 0;

    aws.fetch = async () => {
      calls += 1;
      return jsonResponse({ DurableExecutions: [] });
    };

    await expect(
      workflow.list(
        { name: "my-workflow", qualifier: "live" },
        { status: ["RUNNING", "FAILED"] as never },
      ),
    ).rejects.toThrow("workflow.list status must be a single status");

    expect(calls).toBe(0);
  });

  it("describes a workflow execution", async () => {
    aws.fetch = async (service, path, init) => {
      expect(service).toBe("lambda");
      expect(path).toBe(
        "/2025-12-01/durable-executions/arn%3Aworkflow%3Alive%2Fexecution-123",
      );
      expect(init.method).toBe("GET");

      return jsonResponse({
        DurableExecutionArn: "arn:workflow:live/execution-123",
        DurableExecutionName: "execution-123",
        FunctionArn: "arn:function:live",
        Status: "RUNNING",
        StartTimestamp: 1775556000,
        Version: "12",
      });
    };

    const result = await workflow.describe("arn:workflow:live/execution-123");

    expect(result.arn).toBe("arn:workflow:live/execution-123");
    expect(result.name).toBe("execution-123");
    expect(result.status).toBe("RUNNING");
    expect(result.createdAt.toISOString()).toBe("2026-04-07T10:00:00.000Z");
    expect(result.version).toBe("12");
    expect("response" in result).toBe(false);
  });

  it("stops a workflow execution with a normalized error payload", async () => {
    aws.fetch = async (service, path, init) => {
      expect(service).toBe("lambda");
      expect(path).toBe(
        "/2025-12-01/durable-executions/arn%3Aworkflow%3Alive%2Fexecution-123/stop",
      );
      expect(init.method).toBe("POST");
      expect(init.headers).toEqual({
        "Content-Type": "application/json",
      });
      expect(JSON.parse(String(init.body))).toEqual({
        ErrorMessage: "Cancelled by user",
        ErrorType: "Cancelled",
        ErrorData: JSON.stringify({ reason: "user" }),
      });

      return jsonResponse({
        StopTimestamp: "1775559600",
      });
    };

    const result = await workflow.stop("arn:workflow:live/execution-123", {
      error: {
        message: "Cancelled by user",
        type: "Cancelled",
        data: { reason: "user" },
      },
    });

    expect(result).toEqual({
      arn: "arn:workflow:live/execution-123",
      status: "STOPPED",
      stoppedAt: new Date("2026-04-07T11:00:00.000Z"),
    });
    expect("response" in result).toBe(false);
  });

  it("succeeds a workflow callback", async () => {
    aws.fetch = async (service, path, init) => {
      expect(service).toBe("lambda");
      expect(path).toBe(
        "/2025-12-01/durable-execution-callbacks/callback-token/succeed",
      );
      expect(init.method).toBe("POST");
      expect(init.headers).toEqual({
        "Content-Type": "application/json",
      });
      expect(JSON.parse(String(init.body))).toEqual({ ok: true });

      return new Response(undefined, { status: 200 });
    };

    const result = await workflow.succeed("callback-token", {
      payload: { ok: true },
    });

    expect(result).toBeUndefined();
  });

  it("fails a workflow callback", async () => {
    aws.fetch = async (service, path, init) => {
      expect(service).toBe("lambda");
      expect(path).toBe(
        "/2025-12-01/durable-execution-callbacks/callback-token/fail",
      );
      expect(init.method).toBe("POST");
      expect(init.headers).toEqual({
        "Content-Type": "application/json",
      });
      expect(JSON.parse(String(init.body))).toEqual({
        ErrorMessage: "boom",
        ErrorType: "Error",
      });

      return new Response(undefined, { status: 200 });
    };

    const result = await workflow.fail("callback-token", {
      error: "boom",
    });

    expect(result).toBeUndefined();
  });

  it("heartbeats a workflow callback", async () => {
    aws.fetch = async (service, path, init) => {
      expect(service).toBe("lambda");
      expect(path).toBe(
        "/2025-12-01/durable-execution-callbacks/callback-token/heartbeat",
      );
      expect(init.method).toBe("POST");

      return new Response(undefined, { status: 200 });
    };

    const result = await workflow.heartbeat("callback-token");

    expect(result).toBeUndefined();
  });
});

describe("workflow rollback runner", () => {
  beforeAll(async () => {
    await LocalDurableTestRunner.setupTestEnvironment({ skipTime: true });
  });

  afterAll(async () => {
    await LocalDurableTestRunner.teardownTestEnvironment();
  });

  it("waitUntil wraps a named wait", async () => {
    const fixedNow = 1_700_000_000_000;
    const originalNow = Date.now;
    Date.now = () => fixedNow;

    try {
      const handler = workflow.handler(async (_event, ctx) => {
        await ctx.waitUntil("pause", new Date(fixedNow + 1_500));
      });

      const runner = new LocalDurableTestRunner({ handlerFunction: handler });
      const execution = await runner.run();

      expect(execution.getStatus()).toBe(ExecutionStatus.SUCCEEDED);

      const wait = await runner.getOperation("pause").waitForData();
      expect(wait.getWaitDetails()?.waitSeconds).toBe(2);
    } finally {
      Date.now = originalNow;
    }
  });

  it("does not register rollback when run fails", async () => {
    const calls: string[] = [];

    const handler = workflow.handler(async (_event, ctx) => {
      try {
        await ctx.stepWithRollback(
          "step-a",
          {
            run: async () => {
              calls.push("run:step-a");
              throw new Error("boom");
            },
            undo: async () => {
              calls.push("undo:step-a");
            },
          },
          {
            retryStrategy: () => ({ shouldRetry: false }),
          },
        );
      } catch (error) {
        await ctx.rollbackAll(error);
        throw error;
      }
    });

    const runner = new LocalDurableTestRunner({ handlerFunction: handler });
    const execution = await runner.run();

    expect(execution.getStatus()).toBe(ExecutionStatus.FAILED);
    expect(calls).toEqual(["run:step-a"]);
  });

  it("rebuilds rollback stack after wait replay", async () => {
    const calls: string[] = [];

    const handler = workflow.handler(async (_event, ctx) => {
      try {
        await ctx.stepWithRollback("step-a", {
          run: async () => {
            calls.push("run:step-a");
            return "result-a";
          },
          undo: async (_error, value) => {
            calls.push(`undo:${value}`);
          },
        });

        await ctx.wait("pause", { seconds: 1 });

        await ctx.stepWithRollback("step-b", {
          run: async () => {
            calls.push("run:step-b");
            return "result-b";
          },
          undo: async (_error, value) => {
            calls.push(`undo:${value}`);
          },
        });

        await ctx.step("fail", async () => {
          throw new Error("boom");
        });
      } catch (error) {
        await ctx.rollbackAll(error);
        throw error;
      }
    });

    const runner = new LocalDurableTestRunner({ handlerFunction: handler });
    const execution = await runner.run();

    expect(execution.getStatus()).toBe(ExecutionStatus.FAILED);
    expect(calls).toEqual([
      "run:step-a",
      "run:step-b",
      "undo:result-b",
      "undo:result-a",
    ]);

    const wait = await runner.getOperation("pause").waitForData();
    expect(wait.getWaitDetails()?.waitSeconds).toBe(1);
    expect(runner.getOperation("Undo 'step-b'").getStatus()).toBeDefined();
    expect(runner.getOperation("Undo 'step-a'").getStatus()).toBeDefined();
  });

  it("inherits step retry config for rollback steps", async () => {
    const calls: string[] = [];
    let undoAttempts = 0;

    const handler = workflow.handler(async (_event, ctx) => {
      try {
        await ctx.stepWithRollback(
          "step-a",
          {
            run: async () => {
              calls.push("run:step-a");
              return "result-a";
            },
            undo: async () => {
              undoAttempts++;
              calls.push(`undo-attempt:${undoAttempts}`);
              if (undoAttempts === 1) {
                throw new Error("retry undo once");
              }
            },
          },
          {
            retryStrategy: (_error, attempt) => ({
              shouldRetry: attempt < 2,
              delay: { seconds: 1 },
            }),
          },
        );

        await ctx.step("fail", async () => {
          throw new Error("boom");
        });
      } catch (error) {
        await ctx.rollbackAll(error);
        throw error;
      }
    });

    const runner = new LocalDurableTestRunner({ handlerFunction: handler });
    const execution = await runner.run();

    expect(execution.getStatus()).toBe(ExecutionStatus.FAILED);
    expect(calls).toEqual([
      "run:step-a",
      "undo-attempt:1",
      "undo-attempt:2",
    ]);
    expect(runner.getOperation("Undo 'step-a'").getStepDetails()?.attempt).toBe(2);
  });

  it("stops rollback on undo failure", async () => {
    const calls: string[] = [];

    const handler = workflow.handler(async (_event, ctx) => {
      try {
        await ctx.stepWithRollback("step-a", {
          run: async () => {
            calls.push("run:step-a");
            return "result-a";
          },
          undo: async () => {
            calls.push("undo:step-a");
          },
        });

        await ctx.stepWithRollback("step-b", {
          run: async () => {
            calls.push("run:step-b");
            return "result-b";
          },
          undo: async () => {
            calls.push("undo:step-b");
            throw new Error("undo failed");
          },
        });

        await ctx.step("fail", async () => {
          throw new Error("boom");
        });
      } catch (error) {
        await ctx.rollbackAll(error);
        throw error;
      }
    });

    const runner = new LocalDurableTestRunner({ handlerFunction: handler });
    const execution = await runner.run();

    expect(execution.getStatus()).toBe(ExecutionStatus.FAILED);
    expect(calls.slice(0, 2)).toEqual(["run:step-a", "run:step-b"]);
    expect(calls.slice(2).every((call) => call === "undo:step-b")).toBe(true);
    expect(calls).not.toContain("undo:step-a");
    expect(execution.getError().errorType).toBe("RollbackError");
    expect(execution.getError().errorMessage).toContain("step-b");
  });

  it("allows rollbackAll to be called twice", async () => {
    const calls: string[] = [];

    const handler = workflow.handler(async (_event, ctx) => {
      try {
        await ctx.stepWithRollback("step-a", {
          run: async () => {
            calls.push("run:step-a");
            return "result-a";
          },
          undo: async () => {
            calls.push("undo:step-a");
          },
        });

        await ctx.step("fail", async () => {
          throw new Error("boom");
        });
      } catch (error) {
        await ctx.rollbackAll(error);
        await ctx.rollbackAll(error);
        throw error;
      }
    });

    const runner = new LocalDurableTestRunner({ handlerFunction: handler });
    const execution = await runner.run();

    expect(execution.getStatus()).toBe(ExecutionStatus.FAILED);
    expect(calls).toEqual(["run:step-a", "undo:step-a"]);
  });
});
