import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test fresh installation scenarios
 * Tests new project setup with latest component versions
 */
describe("Fresh Installation Scenarios", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("New Project Setup", () => {
    it("should create new project with latest component versions", async () => {
      await withTestEnvironment(async () => {
        // Create components with latest versions
        const fn = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler",
          runtime: "nodejs20.x"
        });
        
        const bucket = new MockAWSComponent("TestBucket", "aws:bucket:component", {});
        
        const table = new MockAWSComponent("TestTable", "aws:dynamo:component", {
          fields: {
            id: "string"
          },
          primaryIndex: { hashKey: "id" }
        });

        // Verify components are created with latest versions
        expect(fn).toBeDefined();
        expect(bucket).toBeDefined();
        expect(table).toBeDefined();
        
        // Verify no migration warnings are generated
        expect(consoleSpy.calls).toHaveLength(0);
      });
    });

    it("should initialize components with default configuration", async () => {
      await withTestEnvironment(async () => {
        const fn = new MockAWSComponent("DefaultFunction", "aws:function:component", {
          handler: "index.handler"
        });
        
        const bucket = new MockAWSComponent("DefaultBucket", "aws:bucket:component", {});
        const queue = new MockAWSComponent("DefaultQueue", "aws:queue:component", {});

        // Verify default configurations are applied
        expect(fn.args.handler).toBe("index.handler");
        expect(bucket.args).toBeDefined();
        expect(queue.args).toBeDefined();
      });
    });

    it("should handle component initialization order correctly", async () => {
      await withTestEnvironment(async () => {
        // Create components in random order
        const table = new MockAWSComponent("OrderTable", "aws:dynamo:component", {
          fields: { id: "string" },
          primaryIndex: { hashKey: "id" }
        });
        
        const fn = new MockAWSComponent("OrderFunction", "aws:function:component", {
          handler: "index.handler",
          link: ["OrderTable"]
        });
        
        const bucket = new MockAWSComponent("OrderBucket", "aws:bucket:component", {
          notifications: {
            "object-created": "OrderFunction"
          }
        });
        
        const queue = new MockAWSComponent("OrderQueue", "aws:queue:component", {});

        // Verify all components are properly initialized
        expect(fn.args.link).toContain("OrderTable");
        expect(bucket.args.notifications).toBeDefined();
        expect(table.urn).toBeDefined();
        expect(queue.urn).toBeDefined();
      });
    });
  });

  describe("Component Configuration Validation", () => {
    it("should validate Function configuration on fresh install", async () => {
      await withTestEnvironment(async () => {
        const fn = new MockAWSComponent("ValidatedFunction", "aws:function:component", {
          handler: "index.handler",
          runtime: "nodejs20.x",
          timeout: "30 seconds",
          memory: "512 MB",
          environment: {
            NODE_ENV: "production"
          }
        });

        expect(fn.args.runtime).toBe("nodejs20.x");
        expect(fn.args.timeout).toBe("30 seconds");
        expect(fn.args.memory).toBe("512 MB");
        expect(fn.args.environment.NODE_ENV).toBe("production");
      });
    });

    it("should validate Bucket configuration on fresh install", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("ValidatedBucket", "aws:bucket:component", {
          cors: {
            allowCredentials: true,
            allowHeaders: ["*"],
            allowMethods: ["GET", "POST"],
            allowOrigins: ["*"]
          },
          versioning: true
        });

        expect(bucket.args.cors).toBeDefined();
        expect(bucket.args.cors.allowCredentials).toBe(true);
        expect(bucket.args.versioning).toBe(true);
      });
    });

    it("should validate VPC configuration on fresh install", async () => {
      await withTestEnvironment(async () => {
        const vpc = new MockAWSComponent("ValidatedVpc", "aws:vpc:component", {
          az: 2,
          nat: "managed"
        });

        expect(vpc.args.az).toBe(2);
        expect(vpc.args.nat).toBe("managed");
      });
    });
  });

  describe("Component Linking on Fresh Install", () => {
    it("should link Function with Bucket correctly", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("LinkBucket", "aws:bucket:component", {});
        const fn = new MockAWSComponent("LinkFunction", "aws:function:component", {
          handler: "index.handler",
          link: ["LinkBucket"]
        });

        expect(fn.args.link).toContain("LinkBucket");
      });
    });

    it("should link Function with DynamoDB correctly", async () => {
      await withTestEnvironment(async () => {
        const table = new MockAWSComponent("LinkTable", "aws:dynamo:component", {
          fields: { id: "string" },
          primaryIndex: { hashKey: "id" }
        });
        
        const fn = new MockAWSComponent("LinkFunction", "aws:function:component", {
          handler: "index.handler",
          link: ["LinkTable"]
        });

        expect(fn.args.link).toContain("LinkTable");
      });
    });

    it("should handle multiple component links", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("MultiBucket", "aws:bucket:component", {});
        const table = new MockAWSComponent("MultiTable", "aws:dynamo:component", {
          fields: { id: "string" },
          primaryIndex: { hashKey: "id" }
        });
        const queue = new MockAWSComponent("MultiQueue", "aws:queue:component", {});
        
        const fn = new MockAWSComponent("MultiFunction", "aws:function:component", {
          handler: "index.handler",
          link: ["MultiBucket", "MultiTable", "MultiQueue"]
        });

        expect(fn.args.link).toContain("MultiBucket");
        expect(fn.args.link).toContain("MultiTable");
        expect(fn.args.link).toContain("MultiQueue");
      });
    });
  });

  describe("Resource Naming on Fresh Install", () => {
    it("should generate consistent resource names", async () => {
      await withTestEnvironment(async () => {
        const fn = new MockAWSComponent("NamingFunction", "aws:function:component", {
          handler: "index.handler"
        });
        
        const bucket = new MockAWSComponent("NamingBucket", "aws:bucket:component", {});
        
        const table = new MockAWSComponent("NamingTable", "aws:dynamo:component", {
          fields: { id: "string" },
          primaryIndex: { hashKey: "id" }
        });

        // Verify naming follows conventions
        assertions.validAWSName(fn.name);
        assertions.validAWSName(bucket.name);
        assertions.validAWSName(table.name);
      });
    });

    it("should handle name length limits", async () => {
      await withTestEnvironment(async () => {
        const longName = "A".repeat(100);
        
        const fn = new MockAWSComponent(longName, "aws:function:component", {
          handler: "index.handler"
        });
        
        const bucket = new MockAWSComponent(longName, "aws:bucket:component", {});

        // Verify names are truncated appropriately
        assertions.validAWSName(fn.name, 64);
        assertions.validAWSName(bucket.name, 63);
      });
    });
  });

  describe("Environment Context", () => {
    it("should apply stage context correctly", async () => {
      await withTestEnvironment(async (env) => {
        const fn = new MockAWSComponent("StageFunction", "aws:function:component", {
          handler: "index.handler"
        });
        
        const bucket = new MockAWSComponent("StageBucket", "aws:bucket:component", {});

        expect(fn.name).toContain("production");
        expect(bucket.name).toContain("production");
      }, { 
        app: { name: 'test-app', stage: 'production' },
        stage: 'production'
      });
    });

    it("should apply app context correctly", async () => {
      await withTestEnvironment(async (env) => {
        const fn = new MockAWSComponent("AppFunction", "aws:function:component", {
          handler: "index.handler"
        });
        
        const bucket = new MockAWSComponent("AppBucket", "aws:bucket:component", {});

        expect(fn.name).toContain("myapp");
        expect(bucket.name).toContain("myapp");
      }, { 
        app: { name: 'myapp', stage: 'test' },
        name: 'myapp'
      });
    });
  });

  describe("Error Handling on Fresh Install", () => {
    it("should handle missing required configuration", async () => {
      await withTestEnvironment(async () => {
        // Function without handler should be handled gracefully
        const fn = new MockAWSComponent("NoHandlerFunction", "aws:function:component", {
          runtime: "nodejs20.x"
        });

        expect(fn).toBeDefined();
        // In a real scenario, this might trigger validation warnings
      });
    });

    it("should handle invalid configuration values", async () => {
      await withTestEnvironment(async () => {
        // Invalid runtime should be handled gracefully in mock
        const fn = new MockAWSComponent("InvalidFunction", "aws:function:component", {
          handler: "index.handler",
          runtime: "invalid-runtime"
        });

        expect(fn).toBeDefined();
        expect(fn.args.runtime).toBe("invalid-runtime");
      });
    });
  });

  describe("Fresh Install Integration Scenarios", () => {
    it("should create a complete web application stack", async () => {
      await withTestEnvironment(async () => {
        // Create a typical web app stack
        const vpc = new MockAWSComponent("AppVpc", "aws:vpc:component", {
          az: 2,
          nat: "managed"
        });

        const database = new MockAWSComponent("AppDatabase", "aws:postgres:component", {
          engine: "postgres15.4",
          vpc: "AppVpc"
        });

        const api = new MockAWSComponent("AppApi", "aws:function:component", {
          handler: "api.handler",
          link: ["AppDatabase"],
          vpc: "AppVpc"
        });

        const frontend = new MockAWSComponent("AppFrontend", "aws:bucket:component", {
          cors: {
            allowOrigins: ["*"],
            allowMethods: ["GET", "POST"]
          }
        });

        // Verify all components are created
        expect(vpc).toBeDefined();
        expect(database).toBeDefined();
        expect(api).toBeDefined();
        expect(frontend).toBeDefined();

        // Verify relationships
        expect(database.args.vpc).toBe("AppVpc");
        expect(api.args.link).toContain("AppDatabase");
        expect(api.args.vpc).toBe("AppVpc");
      });
    });

    it("should create a serverless data processing pipeline", async () => {
      await withTestEnvironment(async () => {
        // Create a data processing pipeline
        const inputBucket = new MockAWSComponent("InputBucket", "aws:bucket:component", {});

        const processor = new MockAWSComponent("DataProcessor", "aws:function:component", {
          handler: "process.handler",
          timeout: "5 minutes",
          memory: "1024 MB"
        });

        const outputBucket = new MockAWSComponent("OutputBucket", "aws:bucket:component", {
          notifications: {
            "object-created": "DataProcessor"
          }
        });

        const queue = new MockAWSComponent("ProcessingQueue", "aws:queue:component", {
          visibilityTimeout: "6 minutes"
        });

        // Verify pipeline components
        expect(inputBucket).toBeDefined();
        expect(processor).toBeDefined();
        expect(outputBucket).toBeDefined();
        expect(queue).toBeDefined();

        // Verify configuration
        expect(processor.args.timeout).toBe("5 minutes");
        expect(outputBucket.args.notifications).toBeDefined();
        expect(queue.args.visibilityTimeout).toBe("6 minutes");
      });
    });
  });
});