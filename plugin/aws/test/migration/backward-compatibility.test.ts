import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy } from "../utils/test-helpers";
import { MockAWSComponent, MockComponentFactory } from "../utils/mock-sst";

describe("Backward Compatibility", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;
  let componentFactory: MockComponentFactory;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
    componentFactory = new MockComponentFactory();
  });

  afterEach(() => {
    consoleSpy.restore();
    componentFactory.clear();
  });

  describe("v1 component access via .v1 property", () => {
    it("should provide access to v1 components through static property", async () => {
      await withTestEnvironment(async () => {
        // Mock a component class with v1 property
        class MockComponentV2 extends MockAWSComponent {
          static v1 = class MockComponentV1 extends MockAWSComponent {
            constructor(name: string, args: any = {}) {
              super(name, "aws:test:component:v1", args);
              this.registerVersion({
                new: "1.0.0",
                message: "V1 component"
              });
            }
          };

          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v2", args);
            this.registerVersion({
              new: "2.0.0",
              old: args.oldVersion,
              message: "V2 component with breaking changes",
              forceUpgrade: args.forceUpgrade
            });
          }
        }

        // Test v1 access
        expect(MockComponentV2.v1).toBeDefined();
        expect(typeof MockComponentV2.v1).toBe("function");

        // Create v1 instance
        const v1Component = new MockComponentV2.v1("TestV1Component");
        expect(v1Component).toBeDefined();
        expect(v1Component.type).toBe("aws:test:component:v1");
      });
    });

    it("should allow v1 components to coexist with v2 components", async () => {
      await withTestEnvironment(async () => {
        class MockComponentV2 extends MockAWSComponent {
          static v1 = class MockComponentV1 extends MockAWSComponent {
            constructor(name: string, args: any = {}) {
              super(name, "aws:test:component:v1", args);
            }
          };

          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v2", args);
          }
        }

        // Create both v1 and v2 instances
        const v1Component = new MockComponentV2.v1("TestV1");
        const v2Component = new MockComponentV2("TestV2");

        expect(v1Component.type).toBe("aws:test:component:v1");
        expect(v2Component.type).toBe("aws:test:component:v2");
        expect(v1Component.name).toMatch(/test-app-test-testv1-/);
        expect(v2Component.name).toMatch(/test-app-test-testv2-/);
      });
    });
  });

  describe("legacy component functionality", () => {
    it("should maintain v1 component functionality", async () => {
      await withTestEnvironment(async () => {
        class MockComponentV1 extends MockAWSComponent {
          constructor(name: string, args: MockComponentArgs = {}) {
            super(name, "MockComponentV1", args);
          }

          public legacyMethod(): string {
            return "legacy functionality";
          }

          public generateLegacyOutput() {
            return this.generatePhysicalName("legacy");
          }
        }

        class MockComponentV2 extends MockAWSComponent {
          static v1 = MockComponentV1;

          constructor(name: string, args: MockComponentArgs = {}) {
            super(name, "MockComponentV2", args);
          }

          public newMethod(): string {
            return "new functionality";
          }
        }

        const v1Component = new MockComponentV2.v1("TestLegacy");
        
        // Test legacy methods still work
        expect(v1Component.legacyMethod()).toBe("legacy functionality");
        expect(v1Component.generateLegacyOutput()).toBeDefined();
        expect(v1Component.generateLegacyOutput().value).toContain("legacy");
      });
    });

    it("should handle v1 component arguments correctly", async () => {
      await withTestEnvironment(async () => {
        class MockComponentV1 extends MockAWSComponent {
          constructor(name: string, args: { legacyOption?: string } = {}) {
            super(name, "aws:test:component:v1", args);
          }

          getLegacyOption(): string | undefined {
            return this.args.legacyOption;
          }
        }

        class MockComponentV2 extends MockAWSComponent {
          static v1 = MockComponentV1;
        }

        const v1Component = new MockComponentV2.v1("TestArgs", {
          legacyOption: "legacy-value"
        });

        expect(v1Component.getLegacyOption()).toBe("legacy-value");
      });
    });
  });

  describe("mixed version scenarios", () => {
    it("should handle projects with mixed v1 and v2 components", async () => {
      await withTestEnvironment(async () => {
        class MockComponentV1 extends MockAWSComponent {
          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v1", args);
          }
        }

        class MockComponentV2 extends MockAWSComponent {
          static v1 = MockComponentV1;

          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v2", args);
          }
        }

        // Create a mixed scenario
        const components = [
          new MockComponentV2.v1("LegacyComponent1"),
          new MockComponentV2("NewComponent1"),
          new MockComponentV2.v1("LegacyComponent2"),
          new MockComponentV2("NewComponent2")
        ];

        const v1Components = components.filter(c => c.type.includes("v1"));
        const v2Components = components.filter(c => c.type.includes("v2"));

        expect(v1Components).toHaveLength(2);
        expect(v2Components).toHaveLength(2);
      });
    });

    it("should handle component dependencies between versions", async () => {
      await withTestEnvironment(async () => {
        class MockBucketV1 extends MockAWSComponent {
          constructor(name: string, args: any = {}) {
            super(name, "aws:s3:bucket:v1", args);
          }

          getBucketName() {
            return this.generatePhysicalName("bucket");
          }
        }

        class MockFunctionV2 extends MockAWSComponent {
          constructor(name: string, args: { bucket?: MockBucketV1 } = {}) {
            super(name, "aws:lambda:function:v2", args);
          }

          getBucketReference() {
            return this.args.bucket?.getBucketName();
          }
        }

        class MockBucketV2 extends MockAWSComponent {
          static v1 = MockBucketV1;
        }

        // Create v1 bucket and v2 function
        const bucket = new MockBucketV2.v1("TestBucket");
        const func = new MockFunctionV2("TestFunction", { bucket });

        expect(func.getBucketReference()).toBeDefined();
        expect(func.getBucketReference()?.value).toContain("bucket");
      });
    });
  });

  describe("version compatibility warnings", () => {
    it("should warn when using deprecated v1 components", async () => {
      await withTestEnvironment(async () => {
        class MockComponentV1 extends MockAWSComponent {
          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v1", args);
            console.warn(`Component ${name} is using deprecated v1 API. Consider upgrading to v2.`);
          }
        }

        class MockComponentV2 extends MockAWSComponent {
          static v1 = MockComponentV1;
        }

        new MockComponentV2.v1("DeprecatedComponent");

        expect(consoleSpy.calls.length).toBeGreaterThan(0);
        expect(consoleSpy.calls[0][0]).toContain("deprecated v1 API");
      });
    });

    it("should provide migration guidance for v1 components", async () => {
      await withTestEnvironment(async () => {
        class MockComponentV1 extends MockAWSComponent {
          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v1", args);
            console.warn(
              `Component ${name} is using v1. ` +
              `To migrate to v2, update your configuration and set forceUpgrade: true.`
            );
          }
        }

        class MockComponentV2 extends MockAWSComponent {
          static v1 = MockComponentV1;
        }

        new MockComponentV2.v1("ComponentNeedingMigration");

        expect(consoleSpy.calls.length).toBeGreaterThan(0);
        expect(consoleSpy.calls[0][0]).toContain("forceUpgrade: true");
      });
    });
  });

  describe("v1 component isolation", () => {
    it("should isolate v1 component behavior from v2", async () => {
      await withTestEnvironment(async () => {
        class MockComponentV1 extends MockAWSComponent {
          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v1", args);
          }

          getVersion(): string {
            return "1.0.0";
          }
        }

        class MockComponentV2 extends MockAWSComponent {
          static v1 = MockComponentV1;

          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v2", args);
          }

          getVersion(): string {
            return "2.0.0";
          }
        }

        const v1Component = new MockComponentV2.v1("TestV1");
        const v2Component = new MockComponentV2("TestV2");

        expect(v1Component.getVersion()).toBe("1.0.0");
        expect(v2Component.getVersion()).toBe("2.0.0");
      });
    });

    it("should prevent v1 components from accessing v2 features", async () => {
      await withTestEnvironment(async () => {
        class MockComponentV1 extends MockAWSComponent {
          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v1", args);
          }
        }

        class MockComponentV2 extends MockAWSComponent {
          static v1 = MockComponentV1;

          constructor(name: string, args: any = {}) {
            super(name, "aws:test:component:v2", args);
          }

          newV2Feature(): string {
            return "v2 only feature";
          }
        }

        const v1Component = new MockComponentV2.v1("TestV1");
        
        // v1 component should not have v2 methods
        expect((v1Component as any).newV2Feature).toBeUndefined();
      });
    });
  });
});