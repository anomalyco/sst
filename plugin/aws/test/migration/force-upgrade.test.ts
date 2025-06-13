import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

describe("Force Upgrade Handling", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("forceUpgrade parameter handling", () => {
    it("should bypass migration errors when forceUpgrade is true", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Breaking change requires migration",
            forceUpgrade: true
          });
        }).not.toThrow();
      });
    });

    it("should throw migration errors when forceUpgrade is false", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Breaking change requires migration",
            forceUpgrade: false
          });
        }).toThrow(/Migration required for.*testcomponent.*: Breaking change requires migration/);
      });
    });

    it("should throw migration errors when forceUpgrade is undefined", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Breaking change requires migration"
            // forceUpgrade is undefined
          });
        }).toThrow(/Migration required for.*testcomponent.*: Breaking change requires migration/);
      });
    });
  });

  describe("upgrade validation", () => {
    it("should validate that forceUpgrade is boolean when provided", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        // Test with valid boolean values
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Test with true",
            forceUpgrade: true
          });
        }).not.toThrow();
        
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Test with false",
            forceUpgrade: false
          });
        }).toThrow();
      });
    });

    it("should handle multiple version upgrades with forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        // First upgrade
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "First major upgrade",
            forceUpgrade: true
          });
        }).not.toThrow();
        
        // Second upgrade
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "3.0.0",
            message: "Second major upgrade",
            forceUpgrade: true
          });
        }).not.toThrow();
        
        const history = component.getVersionHistory();
        expect(history).toHaveLength(2);
        expect(history.every(entry => entry.forceUpgrade === true)).toBe(true);
      });
    });
  });

  describe("invalid forceUpgrade values", () => {
    it("should handle string values for forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        // In JavaScript, non-empty strings are truthy
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Test with string 'true'",
            forceUpgrade: "true" as any
          });
        }).not.toThrow();
        
        // Empty string is falsy
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Test with empty string",
            forceUpgrade: "" as any
          });
        }).toThrow();
      });
    });

    it("should handle numeric values for forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        // Non-zero numbers are truthy
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Test with number 1",
            forceUpgrade: 1 as any
          });
        }).not.toThrow();
        
        // Zero is falsy
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Test with number 0",
            forceUpgrade: 0 as any
          });
        }).toThrow();
      });
    });

    it("should handle null and undefined for forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        // null is falsy
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Test with null",
            forceUpgrade: null as any
          });
        }).toThrow();
        
        // undefined is falsy
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Test with undefined",
            forceUpgrade: undefined
          });
        }).toThrow();
      });
    });
  });

  describe("forceUpgrade with same versions", () => {
    it("should not require forceUpgrade when versions are the same", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "2.0.0",
            message: "Same version, no upgrade needed"
          });
        }).not.toThrow();
        
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "2.0.0",
            message: "Same version with forceUpgrade false",
            forceUpgrade: false
          });
        }).not.toThrow();
        
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "2.0.0",
            message: "Same version with forceUpgrade true",
            forceUpgrade: true
          });
        }).not.toThrow();
      });
    });
  });

  describe("forceUpgrade tracking", () => {
    it("should track forceUpgrade flag in version history", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        // Register version without forceUpgrade
        component.registerVersion({
          new: "1.0.0",
          message: "Initial version"
        });
        
        // Register version with forceUpgrade
        component.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Forced upgrade",
          forceUpgrade: true
        });
        
        const history = component.getVersionHistory();
        expect(history).toHaveLength(2);
        expect(history[0].forceUpgrade).toBeUndefined();
        expect(history[1].forceUpgrade).toBe(true);
      });
    });
  });

  describe("complex upgrade scenarios", () => {
    it("should handle mixed upgrade scenarios", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        // Start with initial version
        component.registerVersion({
          new: "1.0.0",
          message: "Initial version"
        });
        
        // Force upgrade to v2
        component.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Major breaking changes",
          forceUpgrade: true
        });
        
        // Try to upgrade to v3 without force (should throw)
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "3.0.0",
            message: "Another major change"
          });
        }).toThrow();
        
        // Force upgrade to v3
        component.registerVersion({
          old: "2.0.0",
          new: "3.0.0",
          message: "Another major change",
          forceUpgrade: true
        });
        
        const history = component.getVersionHistory();
        expect(history).toHaveLength(4);
        expect(history.filter(entry => entry.forceUpgrade === true)).toHaveLength(2);
      });
    });
  });
});