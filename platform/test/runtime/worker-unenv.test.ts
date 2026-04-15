import { describe, expect, it } from "vitest";
import { getCloudflareUnenvConfig } from "../../src/runtime/worker/unenv";

describe("getCloudflareUnenvConfig", () => {
  it("keeps http shimmed before native compat", () => {
    const config = getCloudflareUnenvConfig({
      date: "2025-05-05",
      flags: ["nodejs_compat"],
    });

    expect(config.external).not.toContain("http");
    expect(config.external).not.toContain("node:http");
    expect(config.external).not.toContain("https");
    expect(config.external).not.toContain("node:https");
  });

  it("externalizes http client modules when enabled by flag", () => {
    const config = getCloudflareUnenvConfig({
      date: "2025-05-05",
      flags: ["nodejs_compat", "enable_nodejs_http_modules"],
    });

    expect(config.external).toContain("http");
    expect(config.external).toContain("node:http");
    expect(config.external).toContain("https");
    expect(config.external).toContain("node:https");
    expect(config.external).toContain("_http_agent");
    expect(config.external).toContain("node:_http_agent");
    expect(config.external).not.toContain("_http_server");
    expect(config.external).not.toContain("node:_http_server");
  });

  it("externalizes http server modules once the compat date enables them", () => {
    const config = getCloudflareUnenvConfig({
      date: "2025-09-01",
      flags: ["nodejs_compat"],
    });

    expect(config.external).toContain("_http_server");
    expect(config.external).toContain("node:_http_server");
  });

  it("lets disable flags win over compat date defaults", () => {
    const config = getCloudflareUnenvConfig({
      date: "2026-02-05",
      flags: ["nodejs_compat", "disable_nodejs_http_modules"],
    });

    expect(config.external).not.toContain("http");
    expect(config.external).not.toContain("node:http");
    expect(config.external).not.toContain("https");
    expect(config.external).not.toContain("node:https");
    expect(config.external).not.toContain("_http_server");
    expect(config.external).not.toContain("node:_http_server");
  });
});
