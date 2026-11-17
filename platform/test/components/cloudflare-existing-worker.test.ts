import * as pulumi from "@pulumi/pulumi";
import type * as cloudflare from "@pulumi/cloudflare";
import { describe, expect, it } from "vitest";
import type { Worker } from "../../src/components/cloudflare/worker";

// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
  providers: {},
};

pulumi.runtime.setMocks(
  {
    newResource: function (args: pulumi.runtime.MockResourceArgs): {
      id: string;
      state: any;
    } {
      return {
        id: args.name + "_id",
        state: args.inputs,
      };
    },
    call: function (args: pulumi.runtime.MockCallArgs) {
      return args.inputs;
    },
  },
  "project",
  "stack",
  false,
);

async function resolveOutput<T>(value: pulumi.Output<T>) {
  return await new Promise<T>((resolve) => {
    value.apply((resolved) => {
      resolve(resolved);
      return resolved;
    });
  });
}

function createMockWorker() {
  const script = {
    scriptName: pulumi.output("existing-worker"),
  } as cloudflare.WorkerScript;
  return import("../../src/components/cloudflare/worker").then(({ Worker }) => {
    const worker = Object.create(Worker.prototype) as Worker;
    Object.defineProperty(worker, "nodes", { value: { worker: script } });
    return worker;
  });
}

describe("Cloudflare existing Worker targets", () => {
  it("attaches a queue consumer to an existing Worker", async () => {
    const { QueueWorkerSubscriber } = await import(
      "../../src/components/cloudflare/queue-worker-subscriber"
    );
    const worker = await createMockWorker();

    const subscriber = new QueueWorkerSubscriber("Subscriber", {
      queue: { id: "queue-id" },
      subscriber: worker,
    });

    const scriptName = await resolveOutput(subscriber.nodes.consumer.scriptName);

    expect(scriptName).toBe("existing-worker");
  });

  it("rejects worker transforms when subscribing an existing Worker", async () => {
    const { QueueWorkerSubscriber } = await import(
      "../../src/components/cloudflare/queue-worker-subscriber"
    );
    const worker = await createMockWorker();

    expect(
      () =>
        new QueueWorkerSubscriber("SubscriberWithTransform", {
          queue: { id: "queue-id" },
          subscriber: worker,
          transform: {
            worker: () => undefined,
          },
        }),
    ).toThrow(/already created/);
  });

  it("attaches a cron trigger to an existing Worker", async () => {
    const { Cron } = await import("../../src/components/cloudflare/cron");
    const worker = await createMockWorker();

    const cron = new Cron("Cron", {
      worker,
      schedules: ["*/5 * * * *"],
    });

    const trigger = await resolveOutput(cron.nodes.trigger);
    const scriptName = await resolveOutput(trigger.scriptName);

    expect(scriptName).toBe("existing-worker");
  });
});
