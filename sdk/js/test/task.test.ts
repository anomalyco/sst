import { afterEach, beforeEach, describe, expect, it } from "bun:test";
import { task } from "../src/aws/task.ts";

const resource: task.Resource = {
  cluster: "arn:cluster",
  taskDefinition: "arn:task-definition",
  subnets: ["subnet-1"],
  securityGroups: ["sg-1"],
  assignPublicIp: false,
  containers: ["app"],
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/x-amz-json-1.1" },
    status,
  });
}

describe("task client errors", () => {
  const originalFetch = globalThis.fetch;
  const originalEnv = { ...process.env };

  beforeEach(() => {
    process.env.AWS_ACCESS_KEY_ID = "test";
    process.env.AWS_SECRET_ACCESS_KEY = "test";
    process.env.AWS_REGION = "us-east-1";
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    process.env = { ...originalEnv };
  });

  function respondWith(response: () => Response) {
    globalThis.fetch = (async () => response()) as unknown as typeof fetch;
  }

  it("exposes the ECS failure reason when RunTask returns no tasks", async () => {
    respondWith(() =>
      jsonResponse({
        tasks: [],
        failures: [
          { arn: "arn:x", reason: "Capacity is unavailable at this time" },
        ],
      }),
    );

    const err = await task.run(resource).catch((e) => e);
    expect(err).toBeInstanceOf(task.RunError);
    expect(err.message).toBe(
      "Failed to run task: Capacity is unavailable at this time",
    );
    expect(err.status).toBe(200);
    expect(err.failures).toEqual([
      { arn: "arn:x", reason: "Capacity is unavailable at this time" },
    ]);
  });

  it("exposes the AWS error type and message on an HTTP error", async () => {
    respondWith(() =>
      jsonResponse(
        { __type: "com.amazon#ThrottlingException", message: "Rate exceeded" },
        400,
      ),
    );

    const err = await task.run(resource).catch((e) => e);
    expect(err.message).toBe("Failed to run task: Rate exceeded");
    expect(err.status).toBe(400);
    expect(err.errorType).toBe("ThrottlingException");
    expect(err.body).toEqual({
      __type: "com.amazon#ThrottlingException",
      message: "Rate exceeded",
    });
  });

  it("keeps the plain message when the body is not JSON", async () => {
    respondWith(
      () => new Response("<html>bad gateway</html>", { status: 403 }),
    );

    const err = await task.run(resource).catch((e) => e);
    expect(err.message).toBe("Failed to run task");
    expect(err.status).toBe(403);
    expect(err.body).toBeUndefined();
    expect(err.failures).toBeUndefined();
  });

  it("applies the same details to describe and stop", async () => {
    respondWith(() =>
      jsonResponse({
        tasks: [],
        failures: [{ arn: "arn:t", reason: "MISSING" }],
      }),
    );
    const described = await task.describe(resource, "arn:t").catch((e) => e);
    expect(described).toBeInstanceOf(task.DescribeError);
    expect(described.message).toBe("Failed to describe task: MISSING");

    respondWith(() => jsonResponse({ message: "Task not found" }, 400));
    const stopped = await task.stop(resource, "arn:t").catch((e) => e);
    expect(stopped).toBeInstanceOf(task.StopError);
    expect(stopped.message).toBe("Failed to stop task: Task not found");
  });
});
