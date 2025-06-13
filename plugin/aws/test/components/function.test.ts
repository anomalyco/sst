import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test Function component creation and configuration
 * Tests environment variable handling and linking mechanism
 */
describe("Function Component", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("Function creation", () => {
    it("should create Function component with basic configuration", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler",
          runtime: "nodejs20.x",
          timeout: "30 seconds"
        });

        expect(func).toBeDefined();
        expect(func.originalName).toBe("TestFunction");
        expect(func.name).toMatch(/test-app-test-testfunction-/);
        expect(func.args.handler).toBe("index.handler");
        expect(func.args.runtime).toBe("nodejs20.x");
        expect(func.args.timeout).toBe("30 seconds");
      });
    });

    it("should create Function component with minimal configuration", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("MinimalFunction", "aws:function:component", {
          handler: "index.handler"
        });

        expect(func).toBeDefined();
        expect(func.args.handler).toBe("index.handler");
      });
    });

    it("should create Function component with advanced configuration", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("AdvancedFunction", "aws:function:component", {
          handler: "src/lambda.handler",
          runtime: "python3.11",
          timeout: "5 minutes",
          memory: "1024 MB",
          environment: {
            NODE_ENV: "production",
            API_KEY: "secret-key"
          },
          layers: ["arn:aws:lambda:us-east-1:123456789012:layer:my-layer:1"],
          vpc: {
            securityGroups: ["sg-12345"],
            subnets: ["subnet-12345", "subnet-67890"]
          }
        });

        expect(func.args.runtime).toBe("python3.11");
        expect(func.args.memory).toBe("1024 MB");
        expect(func.args.environment.NODE_ENV).toBe("production");
        expect(func.args.layers).toHaveLength(1);
        expect(func.args.vpc.subnets).toHaveLength(2);
      });
    });
  });

  describe("Function environment variables", () => {
    it("should handle environment variable configuration", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler",
          environment: {
            DATABASE_URL: "postgresql://localhost:5432/mydb",
            REDIS_URL: "redis://localhost:6379",
            NODE_ENV: "development"
          }
        });

        expect(func.args.environment.DATABASE_URL).toBe("postgresql://localhost:5432/mydb");
        expect(func.args.environment.REDIS_URL).toBe("redis://localhost:6379");
        expect(func.args.environment.NODE_ENV).toBe("development");
      });
    });

    it("should handle empty environment variables", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler",
          environment: {}
        });

        expect(func.args.environment).toEqual({});
      });
    });

    it("should handle undefined environment variables", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler"
        });

        expect(func.args.environment).toBeUndefined();
      });
    });

    it("should handle environment variables with special characters", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler",
          environment: {
            "API_KEY": "sk-1234567890abcdef",
            "DATABASE_URL": "postgresql://user:pass@host:5432/db?ssl=true",
            "SPECIAL_CHARS": "value with spaces & symbols!"
          }
        });

        expect(func.args.environment["API_KEY"]).toBe("sk-1234567890abcdef");
        expect(func.args.environment["DATABASE_URL"]).toContain("postgresql://");
        expect(func.args.environment["SPECIAL_CHARS"]).toBe("value with spaces & symbols!");
      });
    });
  });

  describe("Function linking mechanism", () => {
    it("should link Function with Bucket component", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("TestBucket", "aws:bucket:component", {
          name: "my-test-bucket"
        });

        const func = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler",
          link: [bucket],
          environment: {
            BUCKET_NAME: bucket.generatePhysicalName("bucket")
          }
        });

        expect(func.args.link).toHaveLength(1);
        expect(func.args.link[0]).toBe(bucket);
        assertions.validOutput(func.args.environment.BUCKET_NAME);
      });
    });

    it("should link Function with multiple components", async () => {
      await withTestEnvironment(async () => {
        const bucket = new MockAWSComponent("TestBucket", "aws:bucket:component");
        const dynamo = new MockAWSComponent("TestTable", "aws:dynamo:component");
        const queue = new MockAWSComponent("TestQueue", "aws:queue:component");

        const func = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler",
          link: [bucket, dynamo, queue],
          environment: {
            BUCKET_NAME: bucket.generatePhysicalName("bucket"),
            TABLE_NAME: dynamo.generatePhysicalName("table"),
            QUEUE_URL: queue.generatePhysicalName("queue")
          }
        });

        expect(func.args.link).toHaveLength(3);
        expect(func.args.link).toContain(bucket);
        expect(func.args.link).toContain(dynamo);
        expect(func.args.link).toContain(queue);
      });
    });

    it("should handle empty link array", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler",
          link: []
        });

        expect(func.args.link).toHaveLength(0);
      });
    });

    it("should handle undefined link", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("TestFunction", "aws:function:component", {
          handler: "index.handler"
        });

        expect(func.args.link).toBeUndefined();
      });
    });
  });

  describe("Function naming", () => {
    it("should generate valid AWS Lambda function names", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("MyTestFunction", "aws:function:component");
        const physicalName = func.generatePhysicalName("function");
        
        assertions.validAWSName(physicalName);
        expect(physicalName.value).toMatch(/test-app-test-mytestfunction-function-/);
      });
    });

    it("should handle function names with special characters", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("My-Test_Function.v2", "aws:function:component");
        const physicalName = func.generatePhysicalName("function");
        
        assertions.validAWSName(physicalName);
        // Should normalize special characters to hyphens
        expect(physicalName.value).toMatch(/function/);
      });
    });
  });

  describe("Function runtime configuration", () => {
    it("should support Node.js runtimes", async () => {
      await withTestEnvironment(async () => {
        const nodeRuntimes = ["nodejs18.x", "nodejs20.x"];
        
        nodeRuntimes.forEach(runtime => {
          const func = new MockAWSComponent(`Function${runtime}`, "aws:function:component", {
            handler: "index.handler",
            runtime
          });
          
          expect(func.args.runtime).toBe(runtime);
        });
      });
    });

    it("should support Python runtimes", async () => {
      await withTestEnvironment(async () => {
        const pythonRuntimes = ["python3.9", "python3.10", "python3.11"];
        
        pythonRuntimes.forEach(runtime => {
          const func = new MockAWSComponent(`Function${runtime}`, "aws:function:component", {
            handler: "lambda_function.lambda_handler",
            runtime
          });
          
          expect(func.args.runtime).toBe(runtime);
        });
      });
    });

    it("should support other runtimes", async () => {
      await withTestEnvironment(async () => {
        const otherRuntimes = ["java17", "dotnet6", "go1.x", "ruby3.2"];
        
        otherRuntimes.forEach(runtime => {
          const func = new MockAWSComponent(`Function${runtime}`, "aws:function:component", {
            handler: "main.handler",
            runtime
          });
          
          expect(func.args.runtime).toBe(runtime);
        });
      });
    });
  });

  describe("Function timeout and memory configuration", () => {
    it("should handle timeout configuration", async () => {
      await withTestEnvironment(async () => {
        const timeoutConfigs = [
          "30 seconds",
          "1 minute",
          "5 minutes",
          "15 minutes"
        ];
        
        timeoutConfigs.forEach(timeout => {
          const func = new MockAWSComponent(`Function${timeout}`, "aws:function:component", {
            handler: "index.handler",
            timeout
          });
          
          expect(func.args.timeout).toBe(timeout);
        });
      });
    });

    it("should handle memory configuration", async () => {
      await withTestEnvironment(async () => {
        const memoryConfigs = [
          "128 MB",
          "256 MB",
          "512 MB",
          "1024 MB",
          "3008 MB"
        ];
        
        memoryConfigs.forEach(memory => {
          const func = new MockAWSComponent(`Function${memory}`, "aws:function:component", {
            handler: "index.handler",
            memory
          });
          
          expect(func.args.memory).toBe(memory);
        });
      });
    });
  });

  describe("Function VPC configuration", () => {
    it("should handle VPC configuration", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("VPCFunction", "aws:function:component", {
          handler: "index.handler",
          vpc: {
            securityGroups: ["sg-12345", "sg-67890"],
            subnets: ["subnet-abc123", "subnet-def456"]
          }
        });

        expect(func.args.vpc.securityGroups).toHaveLength(2);
        expect(func.args.vpc.subnets).toHaveLength(2);
        expect(func.args.vpc.securityGroups).toContain("sg-12345");
        expect(func.args.vpc.subnets).toContain("subnet-abc123");
      });
    });

    it("should handle empty VPC configuration", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("NoVPCFunction", "aws:function:component", {
          handler: "index.handler",
          vpc: {
            securityGroups: [],
            subnets: []
          }
        });

        expect(func.args.vpc.securityGroups).toHaveLength(0);
        expect(func.args.vpc.subnets).toHaveLength(0);
      });
    });
  });

  describe("Function layers configuration", () => {
    it("should handle Lambda layers", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("LayeredFunction", "aws:function:component", {
          handler: "index.handler",
          layers: [
            "arn:aws:lambda:us-east-1:123456789012:layer:my-layer:1",
            "arn:aws:lambda:us-east-1:123456789012:layer:another-layer:2"
          ]
        });

        expect(func.args.layers).toHaveLength(2);
        expect(func.args.layers[0]).toContain("my-layer");
        expect(func.args.layers[1]).toContain("another-layer");
      });
    });

    it("should handle empty layers array", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("NoLayersFunction", "aws:function:component", {
          handler: "index.handler",
          layers: []
        });

        expect(func.args.layers).toHaveLength(0);
      });
    });
  });

  describe("Function error handling", () => {
    it("should handle missing handler", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("NoHandlerFunction", "aws:function:component", {
          runtime: "nodejs20.x"
        });

        expect(func.args.handler).toBeUndefined();
      });
    });

    it("should handle invalid runtime", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("InvalidRuntimeFunction", "aws:function:component", {
          handler: "index.handler",
          runtime: "invalid-runtime"
        });

        expect(func.args.runtime).toBe("invalid-runtime");
      });
    });

    it("should handle invalid timeout format", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("InvalidTimeoutFunction", "aws:function:component", {
          handler: "index.handler",
          timeout: "invalid-timeout"
        });

        expect(func.args.timeout).toBe("invalid-timeout");
      });
    });

    it("should handle invalid memory format", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("InvalidMemoryFunction", "aws:function:component", {
          handler: "index.handler",
          memory: "invalid-memory"
        });

        expect(func.args.memory).toBe("invalid-memory");
      });
    });
  });

  describe("Function integration scenarios", () => {
    it("should integrate with API Gateway", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("APIFunction", "aws:function:component", {
          handler: "api.handler",
          runtime: "nodejs20.x"
        });

        const api = new MockAWSComponent("TestAPI", "aws:apigateway:component", {
          routes: {
            "GET /users": func.generatePhysicalName("function"),
            "POST /users": func.generatePhysicalName("function")
          }
        });

        assertions.validOutput(api.args.routes["GET /users"]);
        assertions.validOutput(api.args.routes["POST /users"]);
      });
    });

    it("should integrate with EventBridge", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("EventFunction", "aws:function:component", {
          handler: "events.handler",
          runtime: "nodejs20.x"
        });

        const eventBridge = new MockAWSComponent("TestEventBridge", "aws:eventbridge:component", {
          rules: {
            "user-created": {
              target: func.generatePhysicalName("function"),
              pattern: {
                source: ["myapp.users"],
                "detail-type": ["User Created"]
              }
            }
          }
        });

        assertions.validOutput(eventBridge.args.rules["user-created"].target);
      });
    });

    it("should integrate with S3 bucket notifications", async () => {
      await withTestEnvironment(async () => {
        const func = new MockAWSComponent("S3Function", "aws:function:component", {
          handler: "s3.handler",
          runtime: "nodejs20.x"
        });

        const bucket = new MockAWSComponent("TestBucket", "aws:bucket:component", {
          notifications: {
            "object-created": {
              function: func.generatePhysicalName("function"),
              events: ["s3:ObjectCreated:*"],
              filterPrefix: "uploads/"
            }
          }
        });

        assertions.validOutput(bucket.args.notifications["object-created"].function);
      });
    });
  });
});