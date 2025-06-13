import { describe, it, expect } from "bun:test";
import { Remix } from "./remix";

describe("Remix", () => {
  it("should create a Remix component", () => {
    expect(() => {
      new Remix("TestRemix", {
        path: "./test-app"
      });
    }).not.toThrow();
  });

  it("should have correct component type", () => {
    const remix = new Remix("TestRemix", {
      path: "./test-app"
    });
    expect(remix.constructor.name).toBe("Remix");
  });

  it("should accept domain configuration", () => {
    expect(() => {
      new Remix("TestRemix", {
        path: "./test-app",
        domain: "example.com"
      });
    }).not.toThrow();
  });

  it("should accept environment variables", () => {
    expect(() => {
      new Remix("TestRemix", {
        path: "./test-app",
        environment: {
          NODE_ENV: "production",
          API_URL: "https://api.example.com"
        }
      });
    }).not.toThrow();
  });

  it("should accept build command configuration", () => {
    expect(() => {
      new Remix("TestRemix", {
        path: "./test-app",
        buildCommand: "bun run build"
      });
    }).not.toThrow();
  });

  it("should have url property", () => {
    const remix = new Remix("TestRemix", {
      path: "./test-app"
    });
    expect(remix.url).toBeDefined();
  });

  it("should have nodes property with server and assets", () => {
    const remix = new Remix("TestRemix", {
      path: "./test-app"
    });
    expect(remix.nodes).toBeDefined();
    expect(remix.nodes.server).toBeDefined();
    expect(remix.nodes.assets).toBeDefined();
  });

  it("should implement getSSTLink method", () => {
    const remix = new Remix("TestRemix", {
      path: "./test-app"
    });
    const link = remix.getSSTLink();
    expect(link).toBeDefined();
    expect(link.properties).toBeDefined();
    expect(link.properties.url).toBeDefined();
  });
});