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
      const arn =
        args.type === "aws:wafv2/webAcl:WebAcl"
          ? "arn:aws:wafv2:us-east-1:123456789012:global/webacl/test/abc-123"
          : `arn:aws:mock:us-east-1:123456789012:${args.name}`;
      return {
        id: args.inputs.name + "_id",
        state: { ...args.inputs, arn },
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
  webAcl: "aws:wafv2/webAcl:WebAcl",
  logGroup: "aws:cloudwatch/logGroup:LogGroup",
  loggingConfig:
    "aws:wafv2/webAclLoggingConfiguration:WebAclLoggingConfiguration",
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

describe("Router WAF Logging", function () {
  let Router: typeof import("./../../src/components/aws/router").Router;

  beforeAll(async function () {
    Router = (await import("./../../src/components/aws/router")).Router;
  });

  beforeEach(function () {
    createdResources = [];
  });

  describe("no WAF configured", () => {
    it("does not create WAF or logging resources", async () => {
      new Router("NoWaf");
      await settle();
      expect(findResources(TYPES.webAcl)).toHaveLength(0);
      expect(findResources(TYPES.logGroup)).toHaveLength(0);
      expect(findResources(TYPES.loggingConfig)).toHaveLength(0);
    });
  });

  describe("WAF without logging", () => {
    it("creates WebAcl but no logging resources", async () => {
      new Router("WafNoLog", { waf: true });
      await settle();
      expect(findResources(TYPES.webAcl)).toHaveLength(1);
      expect(findResources(TYPES.logGroup)).toHaveLength(0);
      expect(findResources(TYPES.loggingConfig)).toHaveLength(0);
    });
  });

  describe("WAF with logging: true", () => {
    it("creates WebAcl, LogGroup, and LoggingConfiguration", async () => {
      new Router("WafLogTrue", { waf: { logging: true } });
      await settle();
      expect(findResources(TYPES.webAcl)).toHaveLength(1);
      expect(findResources(TYPES.logGroup)).toHaveLength(1);
      expect(findResources(TYPES.loggingConfig)).toHaveLength(1);
    });

    it("LogGroup name starts with aws-waf-logs-", async () => {
      new Router("WafLogName", { waf: { logging: true } });
      await settle();
      const logGroups = findResources(TYPES.logGroup);
      expect(logGroups[0].inputs.name).toMatch(/^aws-waf-logs-/);
    });

    it("LogGroup retention defaults to 30 days (1 month)", async () => {
      new Router("WafLogRet", { waf: { logging: true } });
      await settle();
      const logGroups = findResources(TYPES.logGroup);
      expect(logGroups[0].inputs.retentionInDays).toBe(30);
    });

    it("applies default PII redaction (queryString + cookie/authorization)", async () => {
      new Router("WafLogRedact", { waf: { logging: true } });
      await settle();
      const configs = findResources(TYPES.loggingConfig);
      expect(configs).toHaveLength(1);
      const redacted = configs[0].inputs.redactedFields;
      expect(redacted).toBeDefined();
      expect(redacted).toContainEqual({ queryString: {} });
      expect(redacted).toContainEqual({
        singleHeader: { name: "cookie" },
      });
      expect(redacted).toContainEqual({
        singleHeader: { name: "authorization" },
      });
    });
  });

  describe("logging filter", () => {
    it("include: 'blocked' sets DROP default with KEEP for BLOCK", async () => {
      new Router("WafFiltered", {
        waf: { logging: { include: "blocked" } },
      });
      await settle();
      const configs = findResources(TYPES.loggingConfig);
      expect(configs).toHaveLength(1);
      const filter = configs[0].inputs.loggingFilter;
      expect(filter).toBeDefined();
      expect(filter.defaultBehavior).toBe("DROP");
      expect(filter.filters[0].behavior).toBe("KEEP");
      expect(filter.filters[0].conditions[0].actionCondition.action).toBe(
        "BLOCK",
      );
    });

    it("include: 'all' does not set a logging filter", async () => {
      new Router("WafAll", {
        waf: { logging: { include: "all" } },
      });
      await settle();
      const configs = findResources(TYPES.loggingConfig);
      expect(configs).toHaveLength(1);
      expect(configs[0].inputs.loggingFilter).toBeUndefined();
    });
  });

  describe("retention", () => {
    it("custom retention is applied", async () => {
      new Router("WafRet3m", {
        waf: { logging: { retention: "3 months" } },
      });
      await settle();
      const logGroups = findResources(TYPES.logGroup);
      expect(logGroups[0].inputs.retentionInDays).toBe(90);
    });
  });

  describe("redact", () => {
    it("redact: false disables all redaction", async () => {
      new Router("WafNoRedact", {
        waf: { logging: { redact: false } },
      });
      await settle();
      const configs = findResources(TYPES.loggingConfig);
      expect(configs).toHaveLength(1);
      expect(configs[0].inputs.redactedFields).toBeUndefined();
    });

    it("custom redact fields are applied", async () => {
      new Router("WafCustomRedact", {
        waf: {
          logging: {
            redact: {
              uriPath: true,
              method: true,
              headers: ["x-api-key"],
            },
          },
        },
      });
      await settle();
      const configs = findResources(TYPES.loggingConfig);
      expect(configs).toHaveLength(1);
      const redacted = configs[0].inputs.redactedFields;
      expect(redacted).toContainEqual({ method: {} });
      expect(redacted).toContainEqual({ uriPath: {} });
      expect(redacted).toContainEqual({
        singleHeader: { name: "x-api-key" },
      });
      // queryString not set, should not be in the list
      expect(redacted).not.toContainEqual({ queryString: {} });
    });
  });
});
