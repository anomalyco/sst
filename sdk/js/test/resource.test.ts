import { describe, expect, it } from "bun:test";
import {
  createResource,
  loadResourceEnvironment,
} from "../src/resource/shared.ts";

describe("resource environment", () => {
  it("loads SST_RESOURCE string values", () => {
    const name = "TestStringResource";
    const resource = createResource(() => {});

    loadResourceEnvironment({
      SST_RESOURCE_App: JSON.stringify({ name: "app", stage: "dev" }),
      [`SST_RESOURCE_${name}`]: JSON.stringify({ value: "linked" }),
    });

    expect(resource.App).toEqual({ name: "app", stage: "dev" });
    expect((resource as any)[name]).toEqual({ value: "linked" });
    expect(`SST_RESOURCE_${name}` in resource).toBe(false);
  });

  it("ignores plain string values", () => {
    const resource = createResource(() => {});
    const plain = "TestPlainString";
    const json = "TestPlainJsonString";

    loadResourceEnvironment({
      [plain]: "hello",
      [json]: JSON.stringify({ value: "not-linked" }),
    });

    expect(plain in resource).toBe(false);
    expect(json in resource).toBe(false);
  });

  it("loads non-string bindings directly", () => {
    const resource = createResource(() => {});
    const binding = "TestBinding";
    const value = { get: () => "ok" };

    loadResourceEnvironment({
      [binding]: value,
    });

    expect((resource as any)[binding]).toBe(value);
  });

  it("loads resources once through createResource", () => {
    const name = "TestLoadedOnce";
    let loads = 0;
    const resource = createResource(() => {
      loads += 1;
      loadResourceEnvironment({
        [`SST_RESOURCE_${name}`]: JSON.stringify({ value: "once" }),
      });
    });

    expect((resource as any)[name]).toEqual({ value: "once" });
    expect((resource as any)[name]).toEqual({ value: "once" });
    expect(loads).toBe(1);
  });
});
