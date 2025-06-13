import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

describe("Version Handling", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("registerVersion functionality", () => {
    it("should register version without throwing when no old version exists", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            new: "2.0.0",
            message: "Initial version registration"
          });
        }).not.toThrow();
      });
    });

    it("should register version without throwing when versions match", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            new: "2.0.0",
            old: "2.0.0",
            message: "Same version registration"
          });
        }).not.toThrow();
      });
    });

    it("should throw when version mismatch without forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            new: "2.0.0",
            old: "1.0.0",
            message: "Breaking change requires migration"
          });
        }).toThrow(/Migration required for.*testcomponent.*: Breaking change requires migration/);
      });
    });

    it("should not throw when version mismatch with forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            new: "2.0.0",
            old: "1.0.0",
            message: "Breaking change requires migration",
            forceUpgrade: true
          });
        }).not.toThrow();
      });
    });
  });

  describe("version comparison logic", () => {
    it("should handle semantic version comparisons", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        const testCases = [
          { old: "1.0.0", new: "2.0.0", shouldThrow: true },
          { old: "1.5.0", new: "2.0.0", shouldThrow: true },
          { old: "2.0.0", new: "2.0.0", shouldThrow: false },
          { old: "2.0.0", new: "2.0.1", shouldThrow: true },
          { old: "2.1.0", new: "2.0.0", shouldThrow: true }
        ];

        testCases.forEach(({ old, new: newVersion, shouldThrow }, index) => {
          if (shouldThrow) {
            expect(() => {
              component.registerVersion({
                old,
                new: newVersion,
                message: `Test case ${index + 1}`
              });
            }).toThrow();
          } else {
            expect(() => {
              component.registerVersion({
                old,
                new: newVersion,
                message: `Test case ${index + 1}`
              });
            }).not.toThrow();
          }
        });
      });
    });

    it("should handle non-semantic version strings", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            old: "v1",
            new: "v2",
            message: "Simple version change"
          });
        }).toThrow();
        
        expect(() => {
          component.registerVersion({
            old: "v1",
            new: "v2",
            message: "Simple version change",
            forceUpgrade: true
          });
        }).not.toThrow();
      });
    });
  });

  describe("migration message generation", () => {
    it("should include component name in error message", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("MyTestComponent", "aws:test:component");
        
        try {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Custom migration message"
          });
        } catch (error) {
          expect(error.message).toContain("mytestcomponent");
          expect(error.message).toContain("Custom migration message");
        }
      });
    });

    it("should preserve custom migration messages", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        const customMessage = "Please update your configuration to use the new API";
        
        try {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: customMessage
          });
        } catch (error) {
          expect(error.message).toContain(customMessage);
        }
      });
    });
  });

  describe("version history tracking", () => {
    it("should track version registration history", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        // Register multiple versions
        component.registerVersion({
          new: "1.0.0",
          message: "Initial version"
        });
        
        try {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Major update",
            forceUpgrade: true
          });
        } catch (error) {
          // Expected to not throw due to forceUpgrade
        }
        
        const history = component.getVersionHistory();
        expect(history).toHaveLength(2);
        expect(history[0].new).toBe("1.0.0");
        expect(history[1].new).toBe("2.0.0");
        expect(history[1].forceUpgrade).toBe(true);
      });
    });
  });

  describe("edge cases", () => {
    it("should handle undefined old version", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            old: undefined,
            new: "2.0.0",
            message: "No old version"
          });
        }).not.toThrow();
      });
    });

    it("should handle empty version strings", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        // Empty string should be treated as a version mismatch and throw
        expect(() => {
          component.registerVersion({
            old: "",
            new: "2.0.0",
            message: "Empty old version"
          });
        }).toThrow();
      });
    });

    it("should handle null forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const component = new MockAWSComponent("TestComponent", "aws:test:component");
        
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Null forceUpgrade",
            forceUpgrade: undefined
          });
        }).toThrow();
      });
    });
  });
});