import fs from "fs";
import os from "os";
import path from "path";
import { describe, beforeAll, beforeEach, it, expect, vi } from "vitest";
import * as pulumi from "@pulumi/pulumi";

// Suppress Pulumi "Trace events are unavailable" errors in test environment
process.on("unhandledRejection", (err: any) => {
  if (err?.code === "ERR_TRACE_EVENTS_UNAVAILABLE") return;
  throw err;
});

const work = fs.mkdtempSync(path.join(os.tmpdir(), "sst-function-alias-"));
const bundle = path.join(work, "bundle");
fs.mkdirSync(bundle);
fs.writeFileSync(path.join(bundle, "index.mjs"), "export const handler = 1;");

// The function is built by the CLI over RPC, and the bootstrap bucket is looked
// up in the account. Neither exists here.
vi.mock("../../src/components/rpc/rpc.js", async (importOriginal) => {
  const { rpc } = await importOriginal<any>();
  return {
    rpc: {
      ...rpc,
      call: async (method: string) =>
        method === "Runtime.Build"
          ? {
              handler: "index.handler",
              out: bundle,
              errors: [],
              sourcemaps: [],
            }
          : {},
    },
  };
});
vi.mock("../../src/components/aws/helpers/bootstrap.js", () => ({
  bootstrap: {
    forRegion: async () => ({ asset: "bootstrap-bucket" }),
  },
}));

// Dynamic providers can't serialize their code in this environment. Record
// what the function writes to the Router's key-value store instead.
const kvKeys = vi.hoisted(() => [] as { name: string; args: any }[]);
vi.mock("../../src/components/aws/providers/kv-keys.js", () => ({
  KvKeys: class {
    constructor(name: string, args: any) {
      kvKeys.push({ name, args });
    }
  },
}));
vi.mock("../../src/components/aws/providers/kv-routes-update.js", () => ({
  KvRoutesUpdate: class {},
}));

// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};
// @ts-ignore
global.$cli = {
  paths: { work, platform: work },
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
    newResource: function (args: pulumi.runtime.MockResourceArgs): {
      id: string;
      state: any;
    } {
      createdResources.push({
        type: args.type,
        name: args.name,
        inputs: args.inputs,
      });
      const arn = `arn:aws:mock:us-east-1:123456789012:${args.name}`;
      const state: any = { ...args.inputs, arn };
      if (args.type === TYPES.function) {
        state.version = args.inputs.publish ? "7" : "$LATEST";
        state.qualifiedArn = `${arn}:${state.version}`;
      }
      if (args.type === TYPES.functionUrl) {
        state.functionUrl = args.inputs.qualifier
          ? `https://${args.inputs.qualifier}-url.lambda-url.us-east-1.on.aws/`
          : "https://latest-url.lambda-url.us-east-1.on.aws/";
      }
      return {
        id: args.inputs.name + "_id",
        state,
      };
    },
    call: function (args: pulumi.runtime.MockCallArgs) {
      return {
        region: "us-east-1",
        partition: "aws",
        json: "{}",
        ...args.inputs,
      };
    },
  },
  "project",
  "stack",
  false,
);

const TYPES = {
  function: "aws:lambda/function:Function",
  alias: "aws:lambda/alias:Alias",
  functionUrl: "aws:lambda/functionUrl:FunctionUrl",
  permission: "aws:lambda/permission:Permission",
};

function findResources(type: string) {
  return createdResources.filter((r) => r.type === type);
}

function permissions() {
  return findResources(TYPES.permission).map((r) => ({
    name: r.name,
    action: r.inputs.action,
    principal: r.inputs.principal,
    qualifier: r.inputs.qualifier,
  }));
}

// Building the function zips its bundle on disk, so wait until no new
// resources have been registered for a while rather than for a fixed number
// of ticks.
async function settle() {
  let count = -1;
  let quiet = 0;
  while (quiet < 10) {
    await new Promise((resolve) => setTimeout(resolve, 10));
    quiet = createdResources.length === count ? quiet + 1 : 0;
    count = createdResources.length;
  }
}

function resolve<T>(value: pulumi.Output<T>) {
  return new Promise<T>((done) => value.apply(done));
}

describe("Function versioning alias", function () {
  let Function: typeof import("./../../src/components/aws/function").Function;
  let Router: typeof import("./../../src/components/aws/router").Router;

  beforeAll(async function () {
    Function = (await import("./../../src/components/aws/function")).Function;
    Router = (await import("./../../src/components/aws/router")).Router;
  });

  beforeEach(function () {
    createdResources = [];
    kvKeys.length = 0;
  });

  describe("without an alias", () => {
    it("keeps the function URL and its permissions on $LATEST", async () => {
      new Function("NoAlias", {
        handler: "index.handler",
        url: true,
        versioning: true,
      });
      await settle();

      expect(findResources(TYPES.alias)).toHaveLength(0);
      expect(findResources(TYPES.function)[0].inputs.publish).toBe(true);
      expect(findResources(TYPES.functionUrl)[0].inputs.qualifier).toBe(
        undefined,
      );
      expect(permissions()).toEqual([
        {
          name: "NoAliasPublicFunctionUrlAccess",
          action: "lambda:InvokeFunctionUrl",
          principal: "*",
          qualifier: undefined,
        },
        {
          name: "NoAliasInvokeFunction",
          action: "lambda:InvokeFunction",
          principal: "*",
          qualifier: undefined,
        },
      ]);
    });

    it("leaves aliasName unset", async () => {
      const fn = new Function("NoAliasName", { handler: "index.handler" });
      await settle();

      expect(await resolve(fn.aliasName)).toBe(undefined);
    });
  });

  describe("with an alias", () => {
    it("publishes and points the alias at the published version", async () => {
      new Function("Live", {
        handler: "index.handler",
        versioning: { alias: "live" },
      });
      await settle();

      expect(findResources(TYPES.function)[0].inputs.publish).toBe(true);
      const aliases = findResources(TYPES.alias);
      expect(aliases).toHaveLength(1);
      expect(aliases[0].name).toBe("LiveAlias");
      expect(aliases[0].inputs).toMatchObject({
        name: "live",
        functionName: "arn:aws:mock:us-east-1:123456789012:LiveFunction",
        functionVersion: "7",
      });
    });

    it("creates the function URL on the alias", async () => {
      const fn = new Function("LiveUrl", {
        handler: "index.handler",
        url: true,
        versioning: { alias: "live" },
      });
      await settle();

      expect(findResources(TYPES.functionUrl)[0].inputs.qualifier).toBe("live");
      expect(await resolve(fn.url)).toBe(
        "https://live-url.lambda-url.us-east-1.on.aws/",
      );
      expect(await resolve(fn.aliasName)).toBe("live");
    });

    it("grants public access on the alias", async () => {
      new Function("LivePublic", {
        handler: "index.handler",
        url: true,
        versioning: { alias: "live" },
      });
      await settle();

      expect(permissions()).toEqual([
        {
          name: "LivePublicPublicFunctionUrlAccess",
          action: "lambda:InvokeFunctionUrl",
          principal: "*",
          qualifier: "live",
        },
        {
          name: "LivePublicInvokeFunction",
          action: "lambda:InvokeFunction",
          principal: "*",
          qualifier: "live",
        },
      ]);
    });

    it("routes the Router to the alias URL and grants CloudFront access on the alias", async () => {
      const router = new Router("OacRouter", { protection: "oac" });
      new Function("LiveRouted", {
        handler: "index.handler",
        url: { router: { instance: router } },
        versioning: { alias: "live" },
      });
      await settle();

      expect(
        permissions().filter((p) => p.name.startsWith("LiveRouted")),
      ).toEqual([
        {
          name: "LiveRoutedCloudFrontFunctionUrlAccess",
          action: "lambda:InvokeFunctionUrl",
          principal: "cloudfront.amazonaws.com",
          qualifier: "live",
        },
        {
          name: "LiveRoutedCloudFrontInvokeFunction",
          action: "lambda:InvokeFunction",
          principal: "cloudfront.amazonaws.com",
          qualifier: "live",
        },
      ]);

      const routeKey = kvKeys.find((k) => k.name === "LiveRoutedRouteKey");
      const entries = await resolve(pulumi.output(routeKey!.args.entries));
      expect(JSON.parse(entries.metadata).host).toBe(
        "live-url.lambda-url.us-east-1.on.aws",
      );
    });

    it("lets transform.alias change the alias and its options", async () => {
      let ignoreChanges: string[] | undefined;
      new Function(
        "LiveTransform",
        {
          handler: "index.handler",
          versioning: { alias: "live" },
          transform: {
            alias: (args, opts) => {
              args.description = "Moved by the rollback tool";
              opts.ignoreChanges = ["functionVersion"];
            },
          },
        },
        {
          transformations: [
            ({ type, opts }) => {
              if (type === TYPES.alias) ignoreChanges = opts.ignoreChanges;
              return undefined;
            },
          ],
        },
      );
      await settle();

      expect(findResources(TYPES.alias)[0].inputs.description).toBe(
        "Moved by the rollback tool",
      );
      expect(ignoreChanges).toEqual(["functionVersion"]);
    });

    it("lets linked resources invoke the alias", async () => {
      const fn = new Function("LiveLink", {
        handler: "index.handler",
        versioning: { alias: "live" },
      });
      await settle();

      const [include] = fn.getSSTLink().include;
      expect(await resolve(pulumi.output(include.resources))).toEqual([
        "arn:aws:mock:us-east-1:123456789012:LiveLinkFunction",
        "arn:aws:mock:us-east-1:123456789012:LiveLinkAlias",
      ]);
    });
  });
});
