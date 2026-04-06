import { afterAll, beforeAll, describe, expect, it } from "bun:test";
import {
  ExecutionStatus,
  LocalDurableTestRunner,
} from "@aws/durable-execution-sdk-js-testing";
import { workflow } from "../src/aws/workflow.ts";

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
    expect(execution.getInvocations().length).toBeGreaterThan(1);
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

  it("executes rollback steps durably with retry support", async () => {
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
            undo: {
              retryStrategy: (_error, attempt) => ({
                shouldRetry: attempt < 2,
                delay: { seconds: 1 },
              }),
            },
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
