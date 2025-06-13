import { describe, it, expect } from "bun:test";
import { StaticSite } from "./static-site";

describe("StaticSite", () => {
  it("should create a StaticSite component", () => {
    expect(() => {
      new StaticSite("TestStaticSite", {
        path: "./test-site"
      });
    }).not.toThrow();
  });

  it("should have correct component type", () => {
    const site = new StaticSite("TestStaticSite", {
      path: "./test-site"
    });
    expect(site.constructor.name).toBe("StaticSite");
  });

  it("should accept domain configuration", () => {
    expect(() => {
      new StaticSite("TestStaticSite", {
        path: "./test-site",
        domain: "example.com"
      });
    }).not.toThrow();
  });

  it("should accept build configuration", () => {
    expect(() => {
      new StaticSite("TestStaticSite", {
        path: "./test-site",
        build: {
          command: "bun run build",
          output: "dist"
        }
      });
    }).not.toThrow();
  });

  it("should accept assets configuration", () => {
    expect(() => {
      new StaticSite("TestStaticSite", {
        path: "./test-site",
        assets: {
          textEncoding: "utf-8",
          fileOptions: [
            {
              files: "**/*.css",
              cacheControl: "max-age=31536000,public,immutable"
            }
          ]
        }
      });
    }).not.toThrow();
  });

  it("should have url property", () => {
    const site = new StaticSite("TestStaticSite", {
      path: "./test-site"
    });
    expect(site.url).toBeDefined();
  });

  it("should have nodes property with assets and router", () => {
    const site = new StaticSite("TestStaticSite", {
      path: "./test-site"
    });
    expect(site.nodes).toBeDefined();
    expect(site.nodes.assets).toBeDefined();
    expect(site.nodes.router).toBeDefined();
  });

  it("should implement getSSTLink method", () => {
    const site = new StaticSite("TestStaticSite", {
      path: "./test-site"
    });
    const link = site.getSSTLink();
    expect(link).toBeDefined();
    expect(link.properties).toBeDefined();
    expect(link.properties.url).toBeDefined();
  });
});