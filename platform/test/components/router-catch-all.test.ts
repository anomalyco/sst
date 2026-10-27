import { describe, beforeAll, beforeEach, it, expect } from "vitest";
import * as pulumi from "@pulumi/pulumi";

// Suppress Pulumi "Trace events are unavailable" errors in test environment
process.on("unhandledRejection", (err: any) => {
  if (err?.code === "ERR_TRACE_EVENTS_UNAVAILABLE") return;
  throw err;
});

// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};
global.$util = pulumi;

interface CreatedResource {
  type: string;
  name: string;
  inputs: any;
}

let createdResources: CreatedResource[] = [];

pulumi.runtime.setMocks(
  {
    newResource: function (args: pulumi.runtime.MockResourceArgs): {
      id: string;
      state: any;
    } {
      createdResources.push({
        type: args.type,
        name: args.name,
        inputs: args.inputs,
      });
      return {
        id: args.inputs.name + "_id",
        state: {
          ...args.inputs,
          arn: `arn:aws:mock:us-east-1:123456789012:${args.name}`,
        },
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

const TYPES = {
  cfFunction: "aws:cloudfront/function:Function",
};

function findResources(type: string) {
  return createdResources.filter((r) => r.type === type);
}

// Flush event loop to let Pulumi .apply() chains settle without wall-clock delays
async function settle() {
  for (let i = 0; i < 50; i++) {
    await new Promise((resolve) => setImmediate(resolve));
  }
}

describe("Router catch-all route", function () {
  let Router: typeof import("./../../src/components/aws/router").Router;

  beforeAll(async function () {
    Router = (await import("./../../src/components/aws/router")).Router;
  });

  beforeEach(function () {
    createdResources = [];
  });

  it("returns 404 at the edge when no /* route is defined", async () => {
    new Router("NoCatchAll", {
      routes: { "/api/*": "https://example.com" },
    });
    await settle();

    const code = findResources(TYPES.cfFunction)
      .map((fn) => fn.inputs.code as string)
      .join("\n");
    expect(code).toContain("statusCode: 404");
  });

  it("does not inject a 404 when a /* route is defined", async () => {
    new Router("WithCatchAll", {
      routes: {
        "/api/*": "https://example.com",
        "/*": "https://example.org",
      },
    });
    await settle();

    const code = findResources(TYPES.cfFunction)
      .map((fn) => fn.inputs.code as string)
      .join("\n");
    expect(code).not.toContain("statusCode: 404");
  });
});
