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
        id: `${args.name}_id`,
        state: {
          ...args.inputs,
          id: `${args.name}_id`,
          arn: `arn:aws:apigateway:us-east-1::/apis/${args.name}`,
          apiEndpoint: `https://${args.name}.execute-api.us-east-1.amazonaws.com`,
          executionArn: `arn:aws:execute-api:us-east-1:123456789012:${args.name}`,
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

const API_TYPE = "aws:apigatewayv2/api:Api";

function findApis() {
  return createdResources.filter((r) => r.type === API_TYPE);
}

async function settle() {
  for (let i = 0; i < 50; i++) {
    await new Promise((resolve) => setImmediate(resolve));
  }
}

describe("ApiGatewayV2 CORS", function () {
  let ApiGatewayV2: typeof import("./../../src/components/aws/apigatewayv2").ApiGatewayV2;

  beforeAll(async function () {
    ApiGatewayV2 = (await import("./../../src/components/aws/apigatewayv2"))
      .ApiGatewayV2;
  });

  beforeEach(function () {
    createdResources = [];
  });

  it("enables wildcard CORS by default", async () => {
    new ApiGatewayV2("DefaultCors");
    await settle();

    const apis = findApis();
    expect(apis).toHaveLength(1);
    expect(apis[0].inputs.corsConfiguration).toEqual({
      allowHeaders: ["*"],
      allowMethods: ["*"],
      allowOrigins: ["*"],
    });
  });

  it("omits corsConfiguration when cors is false", async () => {
    new ApiGatewayV2("CorsDisabled", { cors: false });
    await settle();

    const apis = findApis();
    expect(apis).toHaveLength(1);
    expect(apis[0].inputs.corsConfiguration).toBeUndefined();
  });

  it("applies custom CORS settings", async () => {
    new ApiGatewayV2("CustomCors", {
      cors: {
        allowMethods: ["GET", "POST"],
        allowOrigins: ["https://example.com"],
      },
    });
    await settle();

    const apis = findApis();
    expect(apis).toHaveLength(1);
    expect(apis[0].inputs.corsConfiguration).toEqual({
      allowHeaders: ["*"],
      allowMethods: ["GET", "POST"],
      allowOrigins: ["https://example.com"],
    });
  });
});
