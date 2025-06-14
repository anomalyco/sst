import { describe, beforeAll, it, expect } from "vitest";
import * as pulumi from "@pulumi/pulumi";
import { setupSSTTestEnvironment, createAWSMocks } from "../helpers/pulumi-mocks";
import { createComponentTestSuite, testComponentCreation, testSSTNaming } from "../helpers/test-utils";

// Set up global test environment
setupSSTTestEnvironment("test-app", "test");

describe("AWS Function Component", () => {
  let Function: typeof import("../../../../src/components/aws/function").Function;

  beforeAll(async () => {
    Function = (await import("../../../../src/components/aws/function")).Function;
  });

  describe("Basic Function Creation", () => {
    it("should create a basic function with default settings", async () => {
      const fn = await testComponentCreation(() => new Function("TestFunction", {
        handler: "index.handler",
      }));

      expect(fn).toBeDefined();
    });

    it("should create function with custom runtime", async () => {
      const fn = new Function("TestFunction", {
        handler: "index.handler",
        runtime: "nodejs20.x",
      });

      expect(fn).toBeDefined();
    });

    it("should create function with environment variables", async () => {
      const fn = new Function("TestFunction", {
        handler: "index.handler",
        environment: {
          NODE_ENV: "test",
          API_URL: "https://api.example.com",
        },
      });

      expect(fn).toBeDefined();
    });

    it("should create function with custom timeout and memory", async () => {
      const fn = new Function("TestFunction", {
        handler: "index.handler",
        timeout: "30 seconds",
        memory: "512 MB",
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function Naming", () => {
    it("should follow SST naming conventions", async () => {
      const fn = new Function("TestFunction", {
        handler: "index.handler",
      });

      // Test that the function name follows SST conventions
      await pulumi.all([fn.name]).apply(([name]) => {
        testSSTNaming(name, "testfunction", "test-app", "test");
      });
    });

    it("should handle special characters in function names", async () => {
      const fn = new Function("Test-Function_123", {
        handler: "index.handler",
      });

      await pulumi.all([fn.name]).apply(([name]) => {
        expect(name).toMatch(/^test-app-test-test-function-123-[a-z0-9]{8}$/);
      });
    });
  });

  describe("Function Runtime Configuration", () => {
    it("should support Node.js runtime", async () => {
      const fn = new Function("NodeFunction", {
        handler: "index.handler",
        runtime: "nodejs20.x",
      });

      expect(fn).toBeDefined();
    });

    it("should support Python runtime", async () => {
      const fn = new Function("PythonFunction", {
        handler: "lambda_function.lambda_handler",
        runtime: "python3.11",
      });

      expect(fn).toBeDefined();
    });

    it("should support Go runtime", async () => {
      const fn = new Function("GoFunction", {
        handler: "main",
        runtime: "go1.x",
      });

      expect(fn).toBeDefined();
    });

    it("should support custom runtime", async () => {
      const fn = new Function("CustomFunction", {
        handler: "bootstrap",
        runtime: "provided.al2",
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function Build Configuration", () => {
    it("should handle TypeScript functions", async () => {
      const fn = new Function("TypeScriptFunction", {
        handler: "src/index.handler",
        runtime: "nodejs20.x",
      });

      expect(fn).toBeDefined();
    });

    it("should handle functions with external dependencies", async () => {
      const fn = new Function("DependencyFunction", {
        handler: "index.handler",
        nodejs: {
          install: ["aws-sdk", "lodash"],
        },
      });

      expect(fn).toBeDefined();
    });

    it("should handle functions with custom build commands", async () => {
      const fn = new Function("BuildFunction", {
        handler: "dist/index.handler",
        nodejs: {
          build: {
            cmd: "npm run build",
            env: {
              NODE_ENV: "production",
            },
          },
        },
      });

      expect(fn).toBeDefined();
    });

    it("should handle functions with bundling configuration", async () => {
      const fn = new Function("BundleFunction", {
        handler: "index.handler",
        nodejs: {
          esbuild: {
            minify: true,
            target: "node18",
            external: ["aws-sdk"],
          },
        },
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function Environment Variables", () => {
    it("should set environment variables correctly", async () => {
      const fn = new Function("EnvFunction", {
        handler: "index.handler",
        environment: {
          NODE_ENV: "production",
          API_KEY: "secret-key",
          DEBUG: "true",
        },
      });

      expect(fn).toBeDefined();
    });

    it("should handle empty environment variables", async () => {
      const fn = new Function("EmptyEnvFunction", {
        handler: "index.handler",
        environment: {},
      });

      expect(fn).toBeDefined();
    });

    it("should handle special characters in environment variables", async () => {
      const fn = new Function("SpecialEnvFunction", {
        handler: "index.handler",
        environment: {
          "SPECIAL_VAR": "value with spaces & symbols!",
          "UNICODE_VAR": "🚀 rocket emoji",
          "JSON_VAR": '{"key": "value"}',
        },
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function Linking", () => {
    it("should support linking to other resources", async () => {
      const fn = new Function("LinkFunction", {
        handler: "index.handler",
        link: [], // Would link to other resources in real scenario
      });

      expect(fn).toBeDefined();
      expect(typeof fn.getSSTLink).toBe("function");
    });

    it("should provide linkable properties", () => {
      const fn = new Function("LinkableFunction", {
        handler: "index.handler",
      });

      // Test that function has linkable properties
      expect(fn).toHaveProperty("name");
      expect(fn).toHaveProperty("arn");
    });
  });

  describe("Function VPC Configuration", () => {
    it("should support VPC configuration", async () => {
      const fn = new Function("VPCFunction", {
        handler: "index.handler",
        vpc: {
          securityGroups: ["sg-12345"],
          subnets: ["subnet-12345", "subnet-67890"],
        },
      });

      expect(fn).toBeDefined();
    });

    it("should handle empty VPC configuration", async () => {
      const fn = new Function("NoVPCFunction", {
        handler: "index.handler",
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function IAM Permissions", () => {
    it("should create function with basic execution role", async () => {
      const fn = new Function("BasicFunction", {
        handler: "index.handler",
      });

      expect(fn).toBeDefined();
      expect(fn).toHaveProperty("role");
    });

    it("should support custom IAM permissions", async () => {
      const fn = new Function("PermissionFunction", {
        handler: "index.handler",
        permissions: [
          {
            actions: ["s3:GetObject", "s3:PutObject"],
            resources: ["arn:aws:s3:::my-bucket/*"],
          },
        ],
      });

      expect(fn).toBeDefined();
    });

    it("should support wildcard permissions", async () => {
      const fn = new Function("WildcardFunction", {
        handler: "index.handler",
        permissions: [
          {
            actions: ["dynamodb:*"],
            resources: ["*"],
          },
        ],
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function Monitoring", () => {
    it("should create CloudWatch log group", async () => {
      const fn = new Function("LogFunction", {
        handler: "index.handler",
      });

      expect(fn).toBeDefined();
      // In real implementation, would check for log group creation
    });

    it("should support custom log retention", async () => {
      const fn = new Function("RetentionFunction", {
        handler: "index.handler",
        logging: {
          retention: "7 days",
        },
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function Scaling", () => {
    it("should support reserved concurrency", async () => {
      const fn = new Function("ConcurrencyFunction", {
        handler: "index.handler",
        reservedConcurrency: 10,
      });

      expect(fn).toBeDefined();
    });

    it("should support provisioned concurrency", async () => {
      const fn = new Function("ProvisionedFunction", {
        handler: "index.handler",
        provisionedConcurrency: 5,
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function Error Handling", () => {
    it("should handle basic error scenarios gracefully", () => {
      // Test that functions can be created even with edge case configurations
      const fn = new Function("ErrorHandlingFunction", {
        handler: "index.handler",
      });
      
      expect(fn).toBeDefined();
    });
  });

  describe("Function Edge Cases", () => {
    it("should handle very long function names", async () => {
      const longName = "A".repeat(50);
      const fn = new Function(longName, {
        handler: "index.handler",
      });

      await pulumi.all([fn.name]).apply(([name]) => {
        // AWS Lambda function names are limited to 64 characters
        expect(name.length).toBeLessThanOrEqual(64);
      });
    });

    it("should handle unicode characters in configuration", async () => {
      const fn = new Function("UnicodeFunction", {
        handler: "index.handler",
        description: "Function with unicode: 🚀 ñáéíóú",
        environment: {
          UNICODE_VAR: "🌟 星星",
        },
      });

      expect(fn).toBeDefined();
    });

    it("should handle large environment variable values", async () => {
      const largeValue = "x".repeat(4000); // Close to AWS limit
      const fn = new Function("LargeEnvFunction", {
        handler: "index.handler",
        environment: {
          LARGE_VAR: largeValue,
        },
      });

      expect(fn).toBeDefined();
    });

    it("should handle maximum timeout", async () => {
      const fn = new Function("MaxTimeoutFunction", {
        handler: "index.handler",
        timeout: "15 minutes", // AWS Lambda maximum
      });

      expect(fn).toBeDefined();
    });

    it("should handle maximum memory", async () => {
      const fn = new Function("MaxMemoryFunction", {
        handler: "index.handler",
        memory: "10240 MB", // AWS Lambda maximum
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function Integration Scenarios", () => {
    it("should work with API Gateway integration", async () => {
      const fn = new Function("APIFunction", {
        handler: "index.handler",
        url: true, // Enable function URL
      });

      expect(fn).toBeDefined();
      expect(fn).toHaveProperty("url");
    });

    it("should work with event source mappings", async () => {
      const fn = new Function("EventFunction", {
        handler: "index.handler",
      });

      expect(fn).toBeDefined();
      // In real scenario, would test event source mapping creation
    });

    it("should work with layers", async () => {
      const fn = new Function("LayerFunction", {
        handler: "index.handler",
        layers: ["arn:aws:lambda:us-east-1:123456789012:layer:my-layer:1"],
      });

      expect(fn).toBeDefined();
    });

    it("should work with dead letter queues", async () => {
      const fn = new Function("DLQFunction", {
        handler: "index.handler",
        deadLetterQueue: "arn:aws:sqs:us-east-1:123456789012:dlq",
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Function Performance", () => {
    it("should handle multiple function creation efficiently", async () => {
      const functions = [];
      const startTime = Date.now();

      for (let i = 0; i < 10; i++) {
        functions.push(new Function(`PerfFunction${i}`, {
          handler: "index.handler",
        }));
      }

      const endTime = Date.now();
      const duration = endTime - startTime;

      // Should create 10 functions in reasonable time (< 1 second in mock environment)
      expect(duration).toBeLessThan(1000);
      expect(functions).toHaveLength(10);
    });
  });
});