import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test incremental migration scenarios
 * Tests upgrading one component at a time and migration path validation
 */
describe("Incremental Migration Scenarios", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("Single Component Migration", () => {
    it("should migrate Auth component from v1 to v2", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { Auth: "1.0.0" };
        
        const auth = new MockAWSComponent("Auth", "aws:auth:component", {
          authenticator: { handler: "auth.handler" }
        });

        // Force upgrade required for major version change
        auth.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Auth v2 migration with OpenAuth integration",
          forceUpgrade: true
        });

        expect(auth).toBeDefined();
        expect(auth.args.authenticator.handler).toBe("auth.handler");
      });
    });

    it("should migrate VPC component from v1 to v2", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { Vpc: "1.0.0" };
        
        const vpc = new MockAWSComponent("Vpc", "aws:vpc:component", {
          az: 2
        });

        // Force upgrade required for major version change
        vpc.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "VPC v2 migration with network architecture changes",
          forceUpgrade: true
        });

        expect(vpc).toBeDefined();
        expect(vpc.args.az).toBe(2);
      });
    });

    it("should migrate Redis component from v1 to v2", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { Redis: "1.0.0" };
        
        const redis = new MockAWSComponent("Redis", "aws:redis:component", {
          engine: "redis"
        });

        // Force upgrade required for major version change
        redis.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Redis v2 migration with configuration changes",
          forceUpgrade: true
        });

        expect(redis).toBeDefined();
        expect(redis.args.engine).toBe("redis");
      });
    });
  });

  describe("Migration Path Validation", () => {
    it("should validate migration from v1.0.0 to v2.0.0", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { ValidationComponent: "1.0.0" };
        
        const component = new MockAWSComponent("ValidationComponent", "aws:test:component", {});
        
        // Major version migration should require force upgrade
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Major version migration",
            forceUpgrade: false
          });
        }).toThrow("Migration required");
      });
    });

    it("should prevent downgrade from v2.0.0 to v1.0.0", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { DowngradeComponent: "2.0.0" };
        
        const component = new MockAWSComponent("DowngradeComponent", "aws:test:component", {});
        
        // Downgrade should fail
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "1.0.0",
            message: "Attempted downgrade",
            forceUpgrade: false
          });
        }).toThrow();
      });
    });

    it("should handle patch version updates gracefully", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { PatchComponent: "1.0.0" };
        
        const component = new MockAWSComponent("PatchComponent", "aws:test:component", {});
        
        // Even patch version updates require force upgrade in SST
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "1.0.1",
            message: "Patch version update",
            forceUpgrade: false
          });
        }).toThrow();
        
        // But should work with force upgrade
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "1.0.1",
            message: "Patch version update",
            forceUpgrade: true
          });
        }).not.toThrow();
      });
    });
  });

  describe("State Preservation During Migration", () => {
    it("should preserve existing component state during migration", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { StatefulComponent: "1.0.0" };
        
        const component = new MockAWSComponent("StatefulComponent", "aws:function:component", {
          handler: "index.handler",
          environment: {
            EXISTING_VAR: "preserved_value"
          }
        });

        // Force upgrade for major version change
        component.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Migration with state preservation",
          forceUpgrade: true
        });

        expect(component.args.environment.EXISTING_VAR).toBe("preserved_value");
      });
    });

    it("should handle configuration changes during migration", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { ConfigComponent: "1.0.0" };
        
        const component = new MockAWSComponent("ConfigComponent", "aws:bucket:component", {
          cors: {
            allowOrigins: ["https://old-domain.com"]
          }
        });

        // Force upgrade for major version change
        component.registerVersion({
          old: "1.0.0",
          new: "2.0.0",
          message: "Configuration update migration",
          forceUpgrade: true
        });

        expect(component.args.cors.allowOrigins).toContain("https://old-domain.com");
      });
    });
  });

  describe("Mixed Version Environment Tests", () => {
    it("should handle mixed version environments", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { 
          Auth: "1.0.0",
          Function: "2.0.0",
          Bucket: "1.5.0"
        };
        
        const auth = new MockAWSComponent("MixedAuth", "aws:auth:component", {
          authenticator: { handler: "auth.handler" }
        });
        
        const fn = new MockAWSComponent("MixedFunction", "aws:function:component", {
          handler: "index.handler"
        });
        
        const bucket = new MockAWSComponent("MixedBucket", "aws:bucket:component", {});

        // Major version migration should require force upgrade
        expect(() => {
          auth.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Auth migration required",
            forceUpgrade: false
          });
        }).toThrow("Migration required");

        expect(auth).toBeDefined();
        expect(fn).toBeDefined();
        expect(bucket).toBeDefined();
      });
    });

    it("should handle dependency chain migrations", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { 
          Vpc: "1.0.0",
          Service: "1.0.0"
        };
        
        const vpc = new MockAWSComponent("ChainVpc", "aws:vpc:component", {
          az: 2
        });
        
        const service = new MockAWSComponent("ChainService", "aws:service:component", {
          vpc: "ChainVpc"
        });

        // Major version migrations should require force upgrade
        expect(() => {
          vpc.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "VPC migration affects dependent services",
            forceUpgrade: false
          });
        }).toThrow("Migration required");

        expect(() => {
          service.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Service migration after VPC update",
            forceUpgrade: false
          });
        }).toThrow("Migration required");

        expect(vpc).toBeDefined();
        expect(service).toBeDefined();
        expect(service.args.vpc).toBe("ChainVpc");
      });
    });
  });

  describe("Migration Rollback Prevention", () => {
    it("should prevent rollback to older versions", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { RollbackComponent: "2.0.0" };
        
        const component = new MockAWSComponent("RollbackComponent", "aws:test:component", {});
        
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "1.0.0",
            message: "Attempted rollback",
            forceUpgrade: false
          });
        }).toThrow();
      });
    });

    it("should allow same version registration", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { SameComponent: "2.0.0" };
        
        const component = new MockAWSComponent("SameComponent", "aws:test:component", {});
        
        expect(() => {
          component.registerVersion({
            old: "2.0.0",
            new: "2.0.0",
            message: "Same version",
            forceUpgrade: false
          });
        }).not.toThrow();
      });
    });
  });

  describe("Complex Migration Scenarios", () => {
    it("should handle multi-component application migration", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = {
          Auth: "1.0.0",
          Vpc: "1.0.0",
          Function: "1.0.0",
          Bucket: "1.0.0"
        };
        
        const auth = new MockAWSComponent("AppAuth", "aws:auth:component", {
          authenticator: { handler: "auth.handler" }
        });
        
        const vpc = new MockAWSComponent("AppVpc", "aws:vpc:component", {
          az: 2
        });
        
        const api = new MockAWSComponent("AppApi", "aws:function:component", {
          handler: "api.handler",
          vpc: "AppVpc"
        });
        
        const storage = new MockAWSComponent("AppStorage", "aws:bucket:component", {});

        // Major version migrations should require force upgrade
        expect(() => {
          auth.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Auth v2 migration",
            forceUpgrade: false
          });
        }).toThrow("Migration required");

        expect(() => {
          vpc.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "VPC v2 migration",
            forceUpgrade: false
          });
        }).toThrow("Migration required");

        expect(auth).toBeDefined();
        expect(vpc).toBeDefined();
        expect(api).toBeDefined();
        expect(storage).toBeDefined();
        expect(api.args.vpc).toBe("AppVpc");
      });
    });

    it("should handle migration with configuration updates", async () => {
      await withTestEnvironment(async (env) => {
        env.sst.version = { ConfigMigration: "1.0.0" };
        
        const component = new MockAWSComponent("ConfigMigration", "aws:function:component", {
          handler: "index.handler",
          runtime: "nodejs18.x",
          timeout: "30 seconds"
        });

        // Major version migration should require force upgrade
        expect(() => {
          component.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Migration includes runtime update to nodejs20.x",
            forceUpgrade: false
          });
        }).toThrow("Migration required");

        component.args.runtime = "nodejs20.x";
        expect(component.args.runtime).toBe("nodejs20.x");
        expect(component.args.timeout).toBe("30 seconds");
      });
    });
  });
});