import { describe, it, expect } from "vitest";
import { CF_ROUTER_INJECTION } from "../../src/components/aws/router";

describe("minify", () => {
  it("CF_ROUTER_INJECTION fits within 10KB limit", () => {
    expect(CF_ROUTER_INJECTION.length).toBeLessThan(10240);
  });
});
