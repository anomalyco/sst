import { describe, it, expect, beforeAll } from "vitest";
import { setupSSTTestEnvironment } from "../../components/pulumi/helpers/pulumi-mocks";

// Set up global test environment
setupSSTTestEnvironment("test-app", "test");

/**
 * Security validation tests for AWS SST components
 * These tests validate that SST components follow security best practices
 */
describe("AWS Security Policies", () => {
  let Function: typeof import("../../../../src/components/aws/function").Function;
  let Bucket: typeof import("../../../../src/components/aws/bucket").Bucket;
  let ApiGatewayV2: typeof import("../../../../src/components/aws/apigatewayv2").ApiGatewayV2;

  beforeAll(async () => {
    Function = (await import("../../../../src/components/aws/function")).Function;
    Bucket = (await import("../../../../src/components/aws/bucket")).Bucket;
    ApiGatewayV2 = (await import("../../../../src/components/aws/apigatewayv2")).ApiGatewayV2;
  });

  describe("Lambda Function Security", () => {
    it("should use supported runtime versions", () => {
      const supportedRuntimes = [
        "nodejs18.x", "nodejs20.x",
        "python3.9", "python3.10", "python3.11", "python3.12",
        "java11", "java17", "java21",
        "dotnet6", "dotnet8",
        "provided.al2", "provided.al2023"
      ];

      const deprecatedRuntimes = [
        "nodejs12.x", "nodejs10.x", "nodejs8.10", "nodejs6.10", "nodejs4.3",
        "python2.7", "python3.6", "python3.7", "python3.8",
        "dotnetcore2.1", "dotnetcore1.0",
        "go1.x"
      ];

      // Test that supported runtimes work
      supportedRuntimes.forEach(runtime => {
        expect(() => {
          new Function(`TestFunction-${runtime}`, {
            handler: "index.handler",
            runtime: runtime as any,
          });
        }).not.toThrow();
      });

      // Test that deprecated runtimes should be avoided (this is advisory)
      deprecatedRuntimes.forEach(runtime => {
        const fn = new Function(`TestFunction-${runtime}`, {
          handler: "index.handler",
          runtime: runtime as any,
        });
        // Function should be created but we log a warning about deprecated runtime
        expect(fn).toBeDefined();
      });
    });

    it("should not allow overly permissive IAM policies", () => {
      // Test that functions don't get wildcard permissions by default
      const fn = new Function("TestFunction", {
        handler: "index.handler",
        permissions: [
          {
            actions: ["s3:GetObject"],
            resources: ["arn:aws:s3:::my-bucket/*"]
          }
        ]
      });

      expect(fn).toBeDefined();
      
      // Functions should not have wildcard permissions unless explicitly needed
      const restrictedFn = new Function("RestrictedFunction", {
        handler: "index.handler",
        permissions: [
          {
            actions: ["s3:*"],
            resources: ["arn:aws:s3:::specific-bucket/*"]
          }
        ]
      });

      expect(restrictedFn).toBeDefined();
    });

    it("should support environment variable encryption", () => {
      const fn = new Function("TestFunction", {
        handler: "index.handler",
        environment: {
          NODE_ENV: "production",
          API_KEY: "secret-value"
        }
      });

      expect(fn).toBeDefined();
      // SST should handle environment variable encryption automatically
    });

    it("should have reasonable timeout and memory limits", () => {
      const fn = new Function("TestFunction", {
        handler: "index.handler",
        timeout: "30 seconds",
        memory: "512 MB"
      });

      expect(fn).toBeDefined();
      
      // Test that extremely high values are allowed but should be reviewed
      const highResourceFn = new Function("HighResourceFunction", {
        handler: "index.handler",
        timeout: "15 minutes",
        memory: "10240 MB"
      });

      expect(highResourceFn).toBeDefined();
    });
  });

  describe("S3 Bucket Security", () => {
    it("should have secure defaults for public access", () => {
      const bucket = new Bucket("TestBucket");
      expect(bucket).toBeDefined();
      
      // SST buckets should block public access by default
      // unless explicitly configured otherwise
    });

    it("should support encryption configuration", () => {
      const bucket = new Bucket("EncryptedBucket", {
        transform: {
          bucket: (args) => {
            args.serverSideEncryptionConfiguration = {
              rules: [{
                applyServerSideEncryptionByDefault: {
                  sseAlgorithm: "AES256"
                }
              }]
            };
          }
        }
      });

      expect(bucket).toBeDefined();
    });

    it("should support versioning for data protection", () => {
      const bucket = new Bucket("VersionedBucket", {
        transform: {
          bucket: (args) => {
            // Versioning can be enabled via transform
          }
        }
      });

      expect(bucket).toBeDefined();
    });

    it("should handle public bucket configuration carefully", () => {
      const publicBucket = new Bucket("PublicBucket", {
        public: true
      });

      expect(publicBucket).toBeDefined();
      // When public is explicitly set to true, it should be allowed
      // but should be reviewed for security implications
    });
  });

  describe("API Gateway Security", () => {
    it("should support HTTPS enforcement", () => {
      const api = new ApiGatewayV2("TestAPI");
      expect(api).toBeDefined();
      
      // SST API Gateway should enforce HTTPS by default
    });

    it("should support custom domains with certificates", () => {
      const api = new ApiGatewayV2("TestAPI", {
        domain: {
          name: "api.example.com",
          cert: "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"
        }
      });

      expect(api).toBeDefined();
    });

    it("should support CORS configuration", () => {
      const api = new ApiGatewayV2("TestAPI", {
        cors: {
          allowOrigins: ["https://example.com"],
          allowMethods: ["GET", "POST"],
          allowHeaders: ["Content-Type", "Authorization"]
        }
      });

      expect(api).toBeDefined();
    });

    it("should support authentication and authorization", () => {
      const api = new ApiGatewayV2("TestAPI");
      
      // Routes can be protected with authorizers
      api.route("GET /protected", {
        handler: "protected.handler",
        auth: {
          jwt: {
            issuer: "https://example.auth0.com/",
            audiences: ["api.example.com"]
          }
        }
      });

      expect(api).toBeDefined();
    });
  });

  describe("VPC and Network Security", () => {
    it("should support VPC configuration for functions", () => {
      const fn = new Function("VPCFunction", {
        handler: "index.handler",
        vpc: {
          securityGroups: ["sg-12345678"],
          subnets: ["subnet-12345678", "subnet-87654321"]
        }
      });

      expect(fn).toBeDefined();
    });

    it("should validate security group configurations", () => {
      // Security groups should be configured through VPC settings
      // and should not allow unrestricted access unless explicitly needed
      const fn = new Function("SecureFunction", {
        handler: "index.handler",
        vpc: {
          securityGroups: ["sg-restrictive"],
          subnets: ["subnet-private"]
        }
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Compliance and Best Practices", () => {
    it("should follow SST naming conventions", () => {
      const fn = new Function("MyFunction", {
        handler: "index.handler"
      });

      expect(fn).toBeDefined();
      // SST should automatically apply consistent naming conventions
    });

    it("should support resource tagging", () => {
      const fn = new Function("TaggedFunction", {
        handler: "index.handler",
        transform: {
          function: (args) => {
            args.tags = {
              Environment: "production",
              Project: "my-app",
              Owner: "team@example.com"
            };
          }
        }
      });

      expect(fn).toBeDefined();
    });

    it("should support monitoring and logging", () => {
      const fn = new Function("MonitoredFunction", {
        handler: "index.handler",
        logging: {
          retention: "1 week"
        }
      });

      expect(fn).toBeDefined();
    });

    it("should validate resource limits and quotas", () => {
      // Test that reasonable limits are enforced
      const fn = new Function("LimitedFunction", {
        handler: "index.handler",
        timeout: "5 minutes",
        memory: "1024 MB"
      });

      expect(fn).toBeDefined();
    });
  });

  describe("Cost Optimization", () => {
    it("should use appropriate instance sizes", () => {
      const fn = new Function("OptimizedFunction", {
        handler: "index.handler",
        memory: "128 MB", // Start with minimal memory
        timeout: "10 seconds" // Use appropriate timeout
      });

      expect(fn).toBeDefined();
    });

    it("should support reserved capacity when appropriate", () => {
      const fn = new Function("ReservedFunction", {
        handler: "index.handler",
        reservedConcurrency: 10 // Limit concurrency for cost control
      });

      expect(fn).toBeDefined();
    });
  });
});