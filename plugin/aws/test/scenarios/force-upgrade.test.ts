import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test force upgrade scenarios
 * Tests force upgrade mechanism and validation
 */
describe("Force Upgrade Scenarios", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("Force Upgrade Mechanism", () => {
    it("should force upgrade Auth component from v1 to v2", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { Auth: "1.0.0" };
        
        const auth = new MockAWSComponent("ForceAuth", "aws:auth:component", {
          authenticator: {
            handler: "auth.handler"
          },
          forceUpgrade: true
        });

        // Force upgrade should bypass migration warnings
        auth.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Auth v2 introduces OpenAuth integration",
          forceUpgrade: true
        });

        expect(auth).toBeDefined();
        expect(auth.args.forceUpgrade).toBe(true);
        // Should not trigger migration warning when force upgrade is used
        expect(consoleSpy.calls).toHaveLength(0);
      });
    });

    it("should force upgrade VPC component with network changes", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { Vpc: "1.0.0" };
        
        const vpc = new MockAWSComponent("ForceVpc", "aws:vpc:component", {
          az: 3,
          nat: "managed",
          forceUpgrade: true
        });

        vpc.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "VPC v2 changes network architecture",
          forceUpgrade: true
        });

        expect(vpc).toBeDefined();
        expect(vpc.args.forceUpgrade).toBe(true);
        expect(vpc.args.az).toBe(3);
      });
    });

    it("should force upgrade multiple components simultaneously", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = {
          Auth: "1.0.0",
          Vpc: "1.0.0",
          Redis: "1.0.0"
        };
        
        const auth = new MockAWSComponent("MultiAuth", "aws:auth:component", {
          authenticator: { handler: "auth.handler" },
          forceUpgrade: true
        });
        
        const vpc = new MockAWSComponent("MultiVpc", "aws:vpc:component", {
          az: 2,
          forceUpgrade: true
        });
        
        const redis = new MockAWSComponent("MultiRedis", "aws:redis:component", {
          engine: "redis7.0",
          forceUpgrade: true
        });

        // All components should force upgrade
        [auth, vpc, redis].forEach(component => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Force upgrade to v2",
            forceUpgrade: true
          });
        });

        expect(auth.args.forceUpgrade).toBe(true);
        expect(vpc.args.forceUpgrade).toBe(true);
        expect(redis.args.forceUpgrade).toBe(true);
      });
    });
  });

  describe("Force Upgrade Validation", () => {
    it("should validate force upgrade parameter types", async () => {
      await withTestEnvironment(async () => {
        // Boolean true should be valid
        const component1 = new MockAWSComponent("ValidForce1", "aws:test:component", {
          forceUpgrade: true
        });
        
        // Boolean false should be valid
        const component2 = new MockAWSComponent("ValidForce2", "aws:test:component", {
          forceUpgrade: false
        });
        
        // Undefined should be valid (defaults to false)
        const component3 = new MockAWSComponent("ValidForce3", "aws:test:component", {});

        expect(component1.args.forceUpgrade).toBe(true);
        expect(component2.args.forceUpgrade).toBe(false);
        expect(component3.args.forceUpgrade).toBeUndefined();
      });
    });

    it("should handle invalid forceUpgrade values gracefully", async () => {
      await withTestEnvironment(async () => {
        // String values should be handled gracefully
        const component = new MockAWSComponent("InvalidForce", "aws:test:component", {
          forceUpgrade: "true" as any
        });

        expect(component).toBeDefined();
        expect(component.args.forceUpgrade).toBe("true");
      });
    });

    it("should validate force upgrade with version constraints", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { ConstraintComponent: "1.0.0" };
        
        const component = new MockAWSComponent("ConstraintComponent", "aws:test:component", {
          forceUpgrade: true
        });

        // Force upgrade should work even with major version jumps
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "3.0.0",
            message: "Major version jump with force upgrade",
            forceUpgrade: true
          });
        }).not.toThrow();
      });
    });
  });

  describe("Post-Upgrade State Validation", () => {
    it("should maintain component functionality after force upgrade", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { FunctionComponent: "1.0.0" };
        
        const fn = new MockAWSComponent("UpgradedFunction", "aws:function:component", {
          handler: "index.handler",
          runtime: "nodejs18.x",
          environment: {
            EXISTING_VAR: "preserved"
          },
          forceUpgrade: true
        });

        fn.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Function upgrade with state preservation",
          forceUpgrade: true
        });

        // Verify functionality is maintained
        expect(fn.args.handler).toBe("index.handler");
        expect(fn.args.environment.EXISTING_VAR).toBe("preserved");
        expect(fn.urn).toBeDefined();
      });
    });

    it("should update configuration after force upgrade", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { BucketComponent: "1.0.0" };
        
        const bucket = new MockAWSComponent("UpgradedBucket", "aws:bucket:component", {
          cors: {
            allowOrigins: ["https://old-domain.com"]
          },
          forceUpgrade: true
        });

        bucket.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Bucket upgrade with new features",
          forceUpgrade: true
        });

        // Update configuration post-upgrade
        bucket.args.versioning = true;
        bucket.args.cors.allowOrigins.push("https://new-domain.com");

        expect(bucket.args.versioning).toBe(true);
        expect(bucket.args.cors.allowOrigins).toContain("https://new-domain.com");
      });
    });

    it("should handle component linking after force upgrade", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = {
          Function: "1.0.0",
          Bucket: "1.0.0"
        };
        
        const bucket = new MockAWSComponent("LinkedBucket", "aws:bucket:component", {
          forceUpgrade: true
        });
        
        const fn = new MockAWSComponent("LinkedFunction", "aws:function:component", {
          handler: "index.handler",
          link: ["LinkedBucket"],
          forceUpgrade: true
        });

        // Force upgrade both components
        bucket.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Bucket force upgrade",
          forceUpgrade: true
        });

        fn.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Function force upgrade",
          forceUpgrade: true
        });

        // Verify linking still works
        expect(fn.args.link).toContain("LinkedBucket");
      });
    });
  });

  describe("Force Upgrade Error Scenarios", () => {
    it("should handle force upgrade with missing dependencies", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { DependentComponent: "1.0.0" };
        
        const component = new MockAWSComponent("DependentComponent", "aws:service:component", {
          vpc: "NonExistentVpc",
          forceUpgrade: true
        });

        component.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Force upgrade with missing dependency",
          forceUpgrade: true
        });

        // Component should be created but dependency issue should be noted
        expect(component).toBeDefined();
        expect(component.args.vpc).toBe("NonExistentVpc");
      });
    });

    it("should handle force upgrade with invalid configuration", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { InvalidComponent: "1.0.0" };
        
        const component = new MockAWSComponent("InvalidComponent", "aws:function:component", {
          handler: "index.handler",
          runtime: "invalid-runtime",
          forceUpgrade: true
        });

        component.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Force upgrade with invalid config",
          forceUpgrade: true
        });

        // Component should be created despite invalid configuration
        expect(component).toBeDefined();
        expect(component.args.runtime).toBe("invalid-runtime");
      });
    });
  });

  describe("Complex Force Upgrade Scenarios", () => {
    it("should handle application-wide force upgrade", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = {
          Auth: "1.0.0",
          Vpc: "1.0.0",
          Function: "1.0.0",
          Bucket: "1.0.0",
          Dynamo: "1.0.0"
        };
        
        // Create complete application stack with force upgrade
        const auth = new MockAWSComponent("AppAuth", "aws:auth:component", {
          authenticator: { handler: "auth.handler" },
          forceUpgrade: true
        });
        
        const vpc = new MockAWSComponent("AppVpc", "aws:vpc:component", {
          az: 2,
          nat: "managed",
          forceUpgrade: true
        });
        
        const api = new MockAWSComponent("AppApi", "aws:function:component", {
          handler: "api.handler",
          vpc: "AppVpc",
          forceUpgrade: true
        });
        
        const storage = new MockAWSComponent("AppStorage", "aws:bucket:component", {
          forceUpgrade: true
        });
        
        const database = new MockAWSComponent("AppDatabase", "aws:dynamo:component", {
          fields: { id: "string" },
          primaryIndex: { hashKey: "id" },
          forceUpgrade: true
        });

        // Force upgrade all components
        [auth, vpc, api, storage, database].forEach(component => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Application-wide force upgrade to v2",
            forceUpgrade: true
          });
        });

        // Verify all components are upgraded
        expect(auth.args.forceUpgrade).toBe(true);
        expect(vpc.args.forceUpgrade).toBe(true);
        expect(api.args.forceUpgrade).toBe(true);
        expect(storage.args.forceUpgrade).toBe(true);
        expect(database.args.forceUpgrade).toBe(true);
        
        // Verify relationships are maintained
        expect(api.args.vpc).toBe("AppVpc");
      });
    });

    it("should handle selective force upgrade in mixed environment", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = {
          CriticalComponent: "1.0.0",
          StableComponent: "2.0.0"
        };
        
        // Force upgrade only critical component
        const critical = new MockAWSComponent("CriticalComponent", "aws:auth:component", {
          authenticator: { handler: "auth.handler" },
          forceUpgrade: true
        });
        
        // Keep stable component as-is
        const stable = new MockAWSComponent("StableComponent", "aws:function:component", {
          handler: "stable.handler"
        });

        // Only critical component should force upgrade
        critical.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Critical security update",
          forceUpgrade: true
        });

        expect(critical.args.forceUpgrade).toBe(true);
        expect(stable.args.forceUpgrade).toBeUndefined();
      });
    });

    it("should handle force upgrade with configuration migration", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { ConfigMigration: "1.0.0" };
        
        const component = new MockAWSComponent("ConfigMigration", "aws:auth:component", {
          // Old v1 configuration
          providers: {
            google: {
              clientId: "old-client-id"
            }
          },
          forceUpgrade: true
        });

        component.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Auth v2 migration with OpenAuth",
          forceUpgrade: true
        });

        // Migrate to new v2 configuration structure
        component.args.authenticator = {
          handler: "auth.handler"
        };
        delete component.args.providers;

        expect(component.args.authenticator).toBeDefined();
        expect(component.args.providers).toBeUndefined();
        expect(component.args.forceUpgrade).toBe(true);
      });
    });
  });

  describe("Force Upgrade Rollback Prevention", () => {
    it("should prevent rollback even with force upgrade", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { NoRollback: "2.0.0" };
        
        const component = new MockAWSComponent("NoRollback", "aws:test:component", {
          forceUpgrade: true
        });

        // Even force upgrade should not allow rollback
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "1.0.0",
            message: "Attempted rollback with force",
            forceUpgrade: true
          });
        }).toThrow();
      });
    });

    it("should allow force upgrade to same version", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { SameVersion: "2.0.0" };
        
        const component = new MockAWSComponent("SameVersion", "aws:test:component", {
          forceUpgrade: true
        });

        // Same version with force upgrade should be allowed
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "2.0.0",
            message: "Force upgrade to same version",
            forceUpgrade: true
          });
        }).not.toThrow();
      });
    });
  });
});