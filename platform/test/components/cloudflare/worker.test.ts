import { describe, expect, it } from "vitest";
import { output, type Input } from "@pulumi/pulumi";
import { Link } from "../../../src/components/link";
import { binding } from "../../../src/components/cloudflare/binding";
import { buildWorkerBinding } from "../../../src/components/cloudflare/helpers/worker-binding";

function link(definition: Link.Definition): Link.Linkable {
  return {
    urn: output("urn:pulumi:stack::project::Resource::Linked"),
    getSSTLink: () => definition,
  };
}

function resolve<T>(value: Input<T>): Promise<T> {
  return new Promise((resolve) => {
    output(value).apply((resolved) => {
      resolve(resolved as T);
      return resolved;
    });
  });
}

describe("Worker binding projection", () => {
  it("projects legacy version metadata into the provider shape", async () => {
    const result = buildWorkerBinding(
      link({
        properties: {},
        include: [binding({ type: "versionMetadataBindings", properties: {} })],
      }),
      "CF_VERSION_METADATA",
    );

    await expect(resolve(result)).resolves.toEqual({
      type: "version_metadata",
      name: "CF_VERSION_METADATA",
    });
  });

  it("copies arbitrary production binding fields and overwrites name", async () => {
    const result = buildWorkerBinding(
      link({
        properties: { ignored: true },
        include: [
          {
            type: "cloudflare.binding",
            binding: {
              type: "future_kind",
              name: "old-name",
              customField: "preserved",
            },
          },
          {
            type: "cloudflare.dev",
            config: { ai: { binding: "wrong-name" } },
          },
        ],
      }),
      output("FinalName"),
    );

    await expect(resolve(result)).resolves.toEqual({
      type: "future_kind",
      name: "FinalName",
      customField: "preserved",
    });
  });

  it("uses the ordinary resource fallback without reading dev config", async () => {
    const result = buildWorkerBinding(
      link({
        properties: { url: "https://example.com" },
        include: [
          {
            type: "cloudflare.dev",
            config: { ai: { binding: "should-not-be-used" } },
          },
        ],
      }),
      "External",
    );

    await expect(resolve(result)).resolves.toEqual({
      type: "secret_text",
      name: "SST_RESOURCE_External",
      text: JSON.stringify({ url: "https://example.com" }),
    });
  });
});
