import { describe, it, expect, beforeEach } from "bun:test";
import { CloudflareComponent } from "./component";

// Mock sst.version to simulate different version scenarios
const mockVersion: Record<string, number> = {};

// Mock sst module - removed unused mock

describe("Cloudflare Component Migration System", () => {
  beforeEach(() => {
    // Clear version mock before each test
    Object.keys(mockVersion).forEach(key => delete mockVersion[key]);
  });

  it("should register version for new component", () => {
    class TestComponent extends CloudflareComponent {
      constructor(name: string) {
        super("test:Component", name, {});
        
        this.registerVersion({
          new: 1,
          old: mockVersion[name],
        });
      }
    }

    expect(() => new TestComponent("TestComponent")).not.toThrow();
  });

  it("should handle version upgrade without forceUpgrade", () => {
    mockVersion["TestComponent"] = 1;

    class TestComponent extends CloudflareComponent {
      constructor(name: string) {
        super("test:Component", name, {});
        
        this.registerVersion({
          new: 2,
          old: mockVersion[name],
          message: "Breaking changes in version 2",
        });
      }
    }

    expect(() => new TestComponent("TestComponent")).toThrow("Breaking changes in version 2");
  });

  it("should allow version upgrade with forceUpgrade", () => {
    mockVersion["TestComponent"] = 1;

    class TestComponent extends CloudflareComponent {
      constructor(name: string) {
        super("test:Component", name, {});
        
        this.registerVersion({
          new: 2,
          old: mockVersion[name],
          message: "Breaking changes in version 2",
          forceUpgrade: "v2",
        });
      }
    }

    expect(() => new TestComponent("TestComponent")).not.toThrow();
  });

  it("should reject invalid forceUpgrade value", () => {
    mockVersion["TestComponent"] = 1;

    class TestComponent extends CloudflareComponent {
      constructor(name: string) {
        super("test:Component", name, {});
        
        this.registerVersion({
          new: 2,
          old: mockVersion[name],
          message: "Breaking changes in version 2",
          forceUpgrade: "v1", // Wrong version
        });
      }
    }

    expect(() => new TestComponent("TestComponent")).toThrow(
      'The value of "forceUpgrade" does not match the version of "test.Component" component'
    );
  });

  it("should prevent version downgrade", () => {
    mockVersion["TestComponent"] = 2;

    class TestComponent extends CloudflareComponent {
      constructor(name: string) {
        super("test:Component", name, {});
        
        this.registerVersion({
          new: 1,
          old: mockVersion[name],
        });
      }
    }

    expect(() => new TestComponent("TestComponent")).toThrow(
      'It seems you are trying to use an older version of "test.Component"'
    );
  });

  it("should handle same version gracefully", () => {
    mockVersion["TestComponent"] = 1;

    class TestComponent extends CloudflareComponent {
      constructor(name: string) {
        super("test:Component", name, {});
        
        this.registerVersion({
          new: 1,
          old: mockVersion[name],
        });
      }
    }

    expect(() => new TestComponent("TestComponent")).not.toThrow();
  });
});