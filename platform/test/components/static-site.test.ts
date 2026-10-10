import fs from "fs";
import os from "os";
import path from "path";
import { describe, beforeAll, beforeEach, it, expect, vi } from "vitest";
import * as pulumi from "@pulumi/pulumi";

// The real OriginAccessControl is a dynamic provider backed by the CLI, which
// can't be serialized inside a Vitest worker thread.
vi.mock(
  "../../src/components/aws/providers/origin-access-control",
  async () => {
    const { CustomResource } = await import("@pulumi/pulumi");
    return {
      OriginAccessControl: class extends CustomResource {
        constructor(name: string, args: any, opts?: any) {
          super("sst:aws:OriginAccessControl", name, args, opts);
        }
      },
    };
  },
);

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
// @ts-ignore
global.$dev = false;
global.$util = pulumi;

interface CreatedResource {
  type: string;
  name: string;
  inputs: any;
}

let createdResources: CreatedResource[] = [];

pulumi.runtime.setMocks(
  {
    newResource: function (args: pulumi.runtime.MockResourceArgs) {
      createdResources.push({
        type: args.type,
        name: args.name,
        inputs: args.inputs,
      });
      return {
        id: args.name + "_id",
        state: {
          ...args.inputs,
          arn: `arn:aws:mock:us-east-1:123456789012:${args.name}`,
          bucketRegionalDomainName: `${args.name}.s3.us-east-1.amazonaws.com`,
        },
      };
    },
    call: function (args: pulumi.runtime.MockCallArgs) {
      if (args.token === "aws:index/getRegion:getRegion")
        return { name: "us-east-1", region: "us-east-1" };
      return args.inputs;
    },
  },
  "project",
  "stack",
  false,
);

// Flush event loop to let Pulumi .apply() chains settle without wall-clock delays
async function settle() {
  for (let i = 0; i < 50; i++) {
    await new Promise((resolve) => setImmediate(resolve));
  }
}

function findDistribution(siteName: string) {
  return createdResources.find(
    (r) =>
      r.type === "aws:cloudfront/distribution:Distribution" &&
      r.name.startsWith(siteName),
  )!;
}

describe("StaticSite", function () {
  let StaticSite: typeof import("./../../src/components/aws/static-site").StaticSite;
  let sitePath: string;

  beforeAll(async function () {
    StaticSite = (await import("./../../src/components/aws/static-site"))
      .StaticSite;
    sitePath = fs.mkdtempSync(path.join(os.tmpdir(), "sst-static-site-"));
    fs.writeFileSync(path.join(sitePath, "index.html"), "<h1>index</h1>");
    fs.writeFileSync(path.join(sitePath, "404.html"), "<h1>404</h1>");
  });

  beforeEach(function () {
    createdResources = [];
  });

  it("uses the assets bucket as the default origin", async () => {
    new StaticSite("Site", { path: sitePath, errorPage: "/404.html" });
    await settle();

    const { origins, customErrorResponses } = findDistribution("Site").inputs;
    expect(origins).toEqual([
      {
        originId: "default",
        domainName: "SiteAssetsBucket.s3.us-east-1.amazonaws.com",
        originAccessControlId: "SiteS3AccessControl_id",
      },
    ]);
    expect(customErrorResponses.map((r: any) => r.responsePagePath)).toEqual([
      "/404.html",
      "/404.html",
    ]);
  });

  it("points the error page at the assets path", async () => {
    new StaticSite("PrefixedSite", {
      path: sitePath,
      errorPage: "404.html",
      assets: { path: "/web/" },
    });
    await settle();

    const { customErrorResponses } = findDistribution("PrefixedSite").inputs;
    expect(customErrorResponses.map((r: any) => r.responsePagePath)).toEqual([
      "/web/404.html",
      "/web/404.html",
    ]);
  });
});
