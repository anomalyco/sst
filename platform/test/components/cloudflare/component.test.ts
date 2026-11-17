import { describe, expect, it } from "vitest";
import type { ComponentResourceOptions } from "@pulumi/pulumi";
import { CloudflareComponent } from "../../../src/components/cloudflare/component";
import type { CloudflareBinding } from "../../../src/components/cloudflare/binding";
import type { DeepInput } from "../../../src/components/cloudflare/helpers/wrangler-config";
import type { Unstable_RawConfig as RawConfig } from "wrangler";

class TestCloudflareComponent extends CloudflareComponent {
  protected readonly type = "import(\"workers-types\").Test";
  protected readonly binding?: CloudflareBinding;
  protected devConfig?: DeepInput<Partial<RawConfig>>;
  private readonly definition;

  constructor(
    name: string,
    definition: { properties: Record<string, unknown>; include?: any[] },
    options: {
      binding?: CloudflareBinding;
      devConfig?: DeepInput<Partial<RawConfig>>;
    } = {},
    opts?: ComponentResourceOptions,
  ) {
    super("sst:test:CloudflareComponent", name, {}, opts);
    this.definition = definition;
    this.binding = options.binding;
    this.devConfig = options.devConfig;
  }

  protected getLinkDefinition() {
    return this.definition;
  }
}

describe("CloudflareComponent", () => {
  it("appends binding, type, and dev includes in protocol order", () => {
    const component = new TestCloudflareComponent(
      "Test",
      { properties: { value: "one" }, include: [{ type: "existing" }] },
      {
        binding: { type: "custom", id: "id" },
        devConfig: { queues: { producers: [{ binding: "__sst_link_name__" }] } },
      },
    );

    expect(component.getSSTLink()).toEqual({
      properties: { value: "one" },
      include: [
        { type: "existing" },
        { type: "cloudflare.binding", binding: { type: "custom", id: "id" } },
        { type: "typescript.type", value: "import(\"workers-types\").Test" },
        {
          type: "cloudflare.dev",
          config: { queues: { producers: [{ binding: "__sst_link_name__" }] } },
        },
      ],
    });
  });

  it("adds only the explicit type for components without a binding", () => {
    const component = new TestCloudflareComponent("Test", {
      properties: { url: "https://example.com" },
    });

    expect(component.getSSTLink()).toEqual({
      properties: { url: "https://example.com" },
      include: [
        { type: "typescript.type", value: "import(\"workers-types\").Test" },
      ],
    });
  });

  it("does not mutate the subclass definition or include values", () => {
    const include = [{ type: "existing" }];
    const definition = { properties: { value: "one" }, include };
    const component = new TestCloudflareComponent("Test", definition, {
      binding: { type: "custom" },
    });

    const first = component.getSSTLink();
    const second = component.getSSTLink();

    expect(definition).toEqual({ properties: { value: "one" }, include });
    expect(first).not.toBe(second);
    expect(first.include).not.toBe(second.include);
    expect(first.include).not.toBe(include);
  });
});
