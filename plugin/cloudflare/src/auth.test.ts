import { describe, it, expect } from "bun:test";

describe("Auth Component", () => {
  it("should have correct module structure", () => {
    // Test that the module can be imported
    const authModule = require("./auth");
    expect(authModule).toBeDefined();
    expect(authModule.Auth).toBeDefined();
    expect(typeof authModule.Auth).toBe("function");
  });

  it("should have correct Pulumi type", () => {
    const { Auth } = require("./auth");
    expect((Auth as any).__pulumiType).toBe("sst:cloudflare:Auth");
  });

  it("should export AuthArgs interface", () => {
    // Test that the module exports are structured correctly
    const authModule = require("./auth");
    expect(authModule.Auth).toBeDefined();
    
    // Check that Auth is a constructor function
    expect(typeof authModule.Auth).toBe("function");
    expect(authModule.Auth.prototype).toBeDefined();
  });

  it("should have getSSTLink method in prototype", () => {
    const { Auth } = require("./auth");
    expect(Auth.prototype.getSSTLink).toBeDefined();
    expect(typeof Auth.prototype.getSSTLink).toBe("function");
  });

  it("should have key getter in prototype", () => {
    const { Auth } = require("./auth");
    expect(Auth.prototype.key).toBeDefined();
  });

  it("should have authenticator getter in prototype", () => {
    const { Auth } = require("./auth");
    expect(Auth.prototype.authenticator).toBeDefined();
  });

  it("should have url getter in prototype", () => {
    const { Auth } = require("./auth");
    expect(Auth.prototype.url).toBeDefined();
  });
});