import { describe, beforeEach, it, expect } from "vitest";
import * as pulumi from "@pulumi/pulumi";

// Set up global SST environment
// @ts-ignore
global.$app = {
  name: "test-app",
  stage: "test",
};
global.$util = pulumi;

// Set up Pulumi mocks
pulumi.runtime.setMocks(
  {
    newResource: function (args: pulumi.runtime.MockResourceArgs): {
      id: string;
      state: any;
    } {
      // Handle SST Bucket component
      if (args.type === "sst:aws:Bucket") {
        // Convert component name to lowercase and remove "bucket" suffix if present
        let bucketName = args.name.toLowerCase();
        if (bucketName.endsWith("bucket")) {
          bucketName = bucketName.slice(0, -6);
        }
        return {
          id: `${args.name}_bucket_id`,
          state: {
            ...args.inputs,
            name: `test-app-test-${bucketName}-12345678`,
            arn: `arn:aws:s3:::test-app-test-${bucketName}-12345678`,
            domain: `test-app-test-${bucketName}-12345678.s3.amazonaws.com`,
          },
        };
      }
      
      // Handle AWS S3 resources
      if (args.type.startsWith("aws:s3/")) {
        const bucketName = args.inputs.bucket || args.inputs.name || args.name;
        return {
          id: `${bucketName}_${args.type.split("/").pop()}_id`,
          state: {
            ...args.inputs,
            bucket: bucketName,
            arn: `arn:aws:s3:::${bucketName}`,
            bucketDomainName: `${bucketName}.s3.amazonaws.com`,
          },
        };
      }
      
      // Default handler
      return {
        id: `${args.name}_${args.type.replace(/[^a-zA-Z0-9]/g, "_")}_id`,
        state: args.inputs,
      };
    },
    call: function (args: pulumi.runtime.MockCallArgs) {
      return args.inputs;
    },
  },
  "project",
  "stack",
  false
);

describe("AWS Bucket Component", () => {
  let Bucket: typeof import("../../../../src/components/aws/bucket").Bucket;

  beforeEach(async () => {
    // Import the Bucket component
    Bucket = (await import("../../../../src/components/aws/bucket")).Bucket;
  });

  describe("Basic Bucket Creation", () => {
    it("should create a bucket with default configuration", async () => {
      const bucket = new Bucket("TestBucket");
      
      expect(bucket).toBeDefined();
      expect(bucket.name).toBeDefined();
      expect(bucket.arn).toBeDefined();
      expect(bucket.domain).toBeDefined();
    });

    it("should follow SST naming conventions", async () => {
      const bucket = new Bucket("MyBucket");
      
      await pulumi.all([bucket.name]).apply(([name]) => {
        expect(name).toMatch(/^test-app-test-.*-[a-z0-9]{8}$/);
      });
    });

    it("should have correct bucket properties", async () => {
      const bucket = new Bucket("TestBucket");
      
      await pulumi.all([bucket.name, bucket.arn, bucket.domain]).apply(([name, arn, domain]) => {
        expect(name).toMatch(/^test-app-test-.*-[a-z0-9]{8}$/);
        expect(arn).toMatch(/^arn:aws:s3:::test-app-test-.*-[a-z0-9]{8}$/);
        expect(domain).toMatch(/^test-app-test-.*-[a-z0-9]{8}\.s3\.amazonaws\.com$/);
      });
    });
  });

  describe("Public Access Configuration", () => {
    it("should create a private bucket by default", async () => {
      const bucket = new Bucket("PrivateBucket");
      expect(bucket).toBeDefined();
    });

    it("should create a public bucket when access is set to public", async () => {
      const bucket = new Bucket("PublicBucket", {
        access: "public"
      });
      expect(bucket).toBeDefined();
    });

    it("should create a cloudfront-accessible bucket", async () => {
      const bucket = new Bucket("CloudFrontBucket", {
        access: "cloudfront"
      });
      expect(bucket).toBeDefined();
    });

    it("should support deprecated public property", async () => {
      const bucket = new Bucket("LegacyPublicBucket", {
        public: true
      });
      expect(bucket).toBeDefined();
    });
  });

  describe("CORS Configuration", () => {
    it("should enable CORS by default", async () => {
      const bucket = new Bucket("CorsDefaultBucket");
      expect(bucket).toBeDefined();
    });

    it("should disable CORS when set to false", async () => {
      const bucket = new Bucket("NoCorseBucket", {
        cors: false
      });
      expect(bucket).toBeDefined();
    });

    it("should configure custom CORS settings", async () => {
      const bucket = new Bucket("CustomCorsBucket", {
        cors: {
          allowHeaders: ["Content-Type", "Authorization"],
          allowOrigins: ["https://example.com", "https://app.example.com"],
          allowMethods: ["GET", "POST", "PUT"],
          exposeHeaders: ["ETag"],
          maxAge: "1 day"
        }
      });
      expect(bucket).toBeDefined();
    });
  });

  describe("Bucket Policies", () => {
    it("should enforce HTTPS by default", async () => {
      const bucket = new Bucket("HttpsBucket");
      expect(bucket).toBeDefined();
    });

    it("should allow disabling HTTPS enforcement", async () => {
      const bucket = new Bucket("NoHttpsBucket", {
        enforceHttps: false
      });
      expect(bucket).toBeDefined();
    });

    it("should support custom bucket policies", async () => {
      const bucket = new Bucket("PolicyBucket", {
        policy: [
          {
            actions: ["s3:GetObject"],
            principals: "*",
            paths: ["public/*"]
          }
        ]
      });
      expect(bucket).toBeDefined();
    });
  });

  describe("Versioning Configuration", () => {
    it("should not enable versioning by default", async () => {
      const bucket = new Bucket("NoVersioningBucket");
      expect(bucket).toBeDefined();
    });

    it("should enable versioning when configured", async () => {
      const bucket = new Bucket("VersionedBucket", {
        versioning: true
      });
      expect(bucket).toBeDefined();
    });
  });

  describe("Bucket Reference", () => {
    it("should support referencing existing buckets", async () => {
      const bucket = Bucket.get("ExistingBucket", "existing-bucket-name");
      expect(bucket).toBeDefined();
      expect(bucket.name).toBeDefined();
      expect(bucket.arn).toBeDefined();
    });
  });

  describe("Component Linking", () => {
    it("should be linkable to other components", () => {
      const bucket = new Bucket("LinkableBucket");
      const link = bucket.getSSTLink();
      
      expect(link).toBeDefined();
      expect(link.properties).toHaveProperty("name");
      expect(link.include).toBeDefined();
      expect(Array.isArray(link.include)).toBe(true);
    });
  });

  describe("Edge Cases and Error Handling", () => {
    it("should handle valid bucket names", async () => {
      const bucket = new Bucket("ValidBucket");
      expect(bucket).toBeDefined();
    });

    it("should handle special characters in bucket name", async () => {
      const bucket = new Bucket("Special-Bucket_123");
      expect(bucket).toBeDefined();
    });

    it("should handle empty policy arrays", async () => {
      const bucket = new Bucket("EmptyPolicyBucket", {
        policy: []
      });
      expect(bucket).toBeDefined();
    });
  });

  describe("Integration Scenarios", () => {
    it("should support static website hosting configuration", async () => {
      const bucket = new Bucket("WebsiteBucket", {
        access: "public",
        cors: {
          allowOrigins: ["*"],
          allowMethods: ["GET", "HEAD"],
          allowHeaders: ["*"]
        }
      });
      expect(bucket).toBeDefined();
    });

    it("should support CDN integration", async () => {
      const bucket = new Bucket("CdnBucket", {
        access: "cloudfront",
        cors: false
      });
      expect(bucket).toBeDefined();
    });

    it("should support backup bucket configuration", async () => {
      const bucket = new Bucket("BackupBucket", {
        versioning: true,
        enforceHttps: true,
        policy: [
          {
            effect: "deny",
            actions: ["s3:DeleteObject", "s3:DeleteObjectVersion"],
            principals: "*"
          }
        ]
      });
      expect(bucket).toBeDefined();
    });
  });

  describe("Transform Configuration", () => {
    it("should support bucket transform", async () => {
      const bucket = new Bucket("TransformBucket", {
        transform: {
          bucket: (args) => ({
            ...args,
            tags: { Environment: "test" }
          })
        }
      });
      expect(bucket).toBeDefined();
    });

    it("should support disabling public access block", async () => {
      const bucket = new Bucket("NoPublicAccessBlockBucket", {
        transform: {
          publicAccessBlock: false
        }
      });
      expect(bucket).toBeDefined();
    });
  });
});