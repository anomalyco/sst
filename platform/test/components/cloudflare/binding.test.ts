import * as cloudflare from "@pulumi/cloudflare";
import { output, type Input } from "@pulumi/pulumi";
import type { Unstable_RawConfig as RawConfig } from "wrangler";
import { describe, expect, it } from "vitest";
import {
  binding,
  type CloudflareBinding,
} from "../../../src/components/cloudflare/binding";

describe("Cloudflare worker binding type baseline", () => {
  const productionBindings = [
    { type: "ai" },
    { type: "kv_namespace", namespaceId: "namespace-id" },
    { type: "r2_bucket", bucketName: "bucket-name" },
    { type: "d1", id: "database-id" },
    { type: "queue", queueName: "queue-name" },
    { type: "durable_object_namespace", className: "Counter" },
    { type: "service", service: "service-name" },
    { type: "hyperdrive", id: "hyperdrive-id" },
    {
      type: "workflow",
      workflowName: "workflow-name",
      className: "Workflow",
      scriptName: "script-name",
    },
    {
      type: "ratelimit",
      namespaceId: "1001",
      simple: { limit: 100, period: 60 },
    },
  ] satisfies CloudflareBinding[];

  const developmentConfig = {
    ai: { binding: "AI", remote: true },
    kv_namespaces: [{ binding: "KV", id: "namespace-id", remote: true }],
    r2_buckets: [
      { binding: "BUCKET", bucket_name: "bucket-name", remote: true },
    ],
    d1_databases: [
      { binding: "DATABASE", database_id: "database-id", remote: true },
    ],
    queues: {
      producers: [{ binding: "QUEUE", queue: "queue-name", remote: true }],
    },
    durable_objects: {
      bindings: [{ name: "COUNTER", class_name: "Counter" }],
    },
    services: [{ binding: "SERVICE", service: "service-name", remote: true }],
    hyperdrive: [{ binding: "HYPERDRIVE", id: "hyperdrive-id" }],
    workflows: [
      {
        binding: "WORKFLOW",
        name: "workflow-name",
        class_name: "Workflow",
        script_name: "script-name",
        remote: true,
      },
    ],
    ratelimits: [
      {
        name: "RATE_LIMIT",
        namespace_id: "1001",
        simple: { limit: 100, period: 60 },
      },
    ],
  } satisfies Partial<RawConfig>;

  it("aligns all first-party production and development shapes", () => {
    expect(productionBindings).toHaveLength(10);
    expect(developmentConfig.ratelimits).toHaveLength(1);
    expect("rate_limits" in developmentConfig).toBe(false);
  });

  it("aligns Pulumi WorkersScriptBinding with Wrangler ratelimits", () => {
    const binding: cloudflare.types.input.WorkersScriptBinding = {
      type: "ratelimit",
      name: "Limiter",
      namespaceId: "namespace-id",
      simple: {
        limit: 100,
        period: 60,
      },
    };
    const bindings: Input<
      Input<cloudflare.types.input.WorkersScriptBinding>[]
    > = [binding];
    const config: Partial<RawConfig> = {
      ratelimits: [
        {
          name: "Limiter",
          namespace_id: "namespace-id",
          simple: {
            limit: 100,
            period: 60,
          },
        },
      ],
    };

    expect(bindings).toHaveLength(1);
    expect(config.ratelimits?.[0]?.name).toBe("Limiter");
  });

  it("preserves arbitrary WorkersScriptBinding fields and omits name", () => {
    const input: CloudflareBinding = {
      type: "custom-kind",
      id: "resource-id",
      customField: "preserved",
    };

    expect(binding(input)).toEqual({
      type: "cloudflare.binding",
      binding: input,
    });
  });

  it("converts legacy SST bindings to provider-shaped bindings", () => {
    const cases = [
      ["aiBindings", "ai", {}],
      ["plainTextBindings", "plain_text", { text: "value" }],
      ["secretTextBindings", "secret_text", { text: "secret" }],
      ["queueBindings", "queue", { queueName: "queue" }],
      ["serviceBindings", "service", { service: "worker" }],
      [
        "durableObjectNamespaceBindings",
        "durable_object_namespace",
        { className: "Counter" },
      ],
      ["kvNamespaceBindings", "kv_namespace", { namespaceId: "namespace" }],
      ["d1DatabaseBindings", "d1", { id: "database" }],
      ["r2BucketBindings", "r2_bucket", { bucketName: "bucket" }],
      ["hyperdriveBindings", "hyperdrive", { id: "hyperdrive" }],
      ["versionMetadataBindings", "version_metadata", {}],
      [
        "workflowBindings",
        "workflow",
        {
          workflowName: "workflow",
          className: "Workflow",
          scriptName: "worker",
        },
      ],
      [
        "rateLimitBindings",
        "ratelimit",
        { namespaceId: "1001", simple: { limit: 100, period: 60 } },
      ],
    ] as const;

    for (const [legacyType, providerType, properties] of cases) {
      expect(binding({ type: legacyType, properties }).binding).toEqual({
        type: providerType,
        ...properties,
      });
    }
  });

  it("accepts every provider binding kind without a local kind union", () => {
    const input: CloudflareBinding = {
      type: "new-provider-kind",
      name: undefined as never,
    };

    expect(binding(input).binding.type).toBe("new-provider-kind");
  });

  it("does not treat inherited object properties as legacy binding kinds", () => {
    const input: CloudflareBinding = {
      type: "constructor",
      customField: "preserved",
    };

    expect(binding(input).binding).toBe(input);
  });

  it("does not allow the Worker-owned name in the public binding type", () => {
    // @ts-expect-error Worker assigns the binding name at the final projection.
    const input: CloudflareBinding = { type: "ai", name: "Ai" };
    expect(input.name).toBe("Ai");
  });

  it("accepts Pulumi inputs for provider binding kinds", () => {
    const input: CloudflareBinding = { type: output("version_metadata") };
    expect(binding(input).binding).toBe(input);
  });
});
