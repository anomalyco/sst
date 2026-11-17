import * as pulumi from "@pulumi/pulumi";
import { describe, expect, it } from "vitest";
import { Task } from "../../src/components/aws/step-functions/task";

async function resolveOutputs<T extends readonly unknown[]>(values: T) {
  return await new Promise<T>((resolve, reject) => {
    pulumi.all(values as unknown as pulumi.Input<unknown>[]).apply((result) => {
      try {
        resolve(result as unknown as T);
      } catch (error) {
        reject(error);
      }
    });
  });
}

describe("Step Functions Task", () => {
  it("serializes duration and JSONata timeouts as TimeoutSeconds", async () => {
    const duration = new Task({
      name: "Duration",
      resource: "arn:aws:states:::lambda:invoke",
      timeout: "2 minutes",
    }).serialize().Duration;
    const jsonata = new Task({
      name: "JSONata",
      resource: "arn:aws:states:::lambda:invoke",
      timeout: "{% $states.input.timeout %}",
    }).serialize().JSONata;

    const [durationTimeout, jsonataTimeout] = await resolveOutputs([
      duration.TimeoutSeconds,
      jsonata.TimeoutSeconds,
    ] as const);

    expect(durationTimeout).toBe(120);
    expect(jsonataTimeout).toBe("{% $states.input.timeout %}");
    expect(duration).not.toHaveProperty("Timeout");
    expect(jsonata).not.toHaveProperty("Timeout");
  });
});
