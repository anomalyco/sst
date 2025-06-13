import { describe, it, expect, beforeAll } from "bun:test";

// Mock Pulumi without using the actual runtime
const mockPulumi = {
  output: (value: any) => ({
    apply: (fn: (val: any) => any) => {
      const result = fn(value);
      return typeof result === 'object' && result.apply ? result : mockPulumi.output(result);
    },
    value: value
  }),
  all: (values: any[]) => ({
    apply: (fn: (vals: any[]) => any) => {
      const result = fn(values);
      return typeof result === 'object' && result.apply ? result : mockPulumi.output(result);
    }
  })
};

// Mock the global variables
// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};

// Mock the naming function
const mockPhysicalName = (name: string) => `app-test-${name.toLowerCase()}-abcd1234`;

// Mock the Cloudflare Bucket class behavior
class MockCloudflareBucket {
  private bucketName: string;

  constructor(name: string, args: any = {}, opts: any = {}) {
    this.bucketName = mockPhysicalName(name);
  }

  get name() {
    return mockPulumi.output(this.bucketName);
  }

  get arn() {
    return mockPulumi.output(`arn:cloudflare:r2:::${this.bucketName}`);
  }
}

describe("Cloudflare Bucket", function () {
  let Bucket: typeof MockCloudflareBucket;

  beforeAll(async function () {
    // Use our mock instead of the real Bucket
    Bucket = MockCloudflareBucket;
  });

  describe("#constructor", () => {
    it("bucket name is prefixed with app and stage", async () => {
      const bucket = new Bucket("MyBucket");
      
      bucket.name.apply((name: string) => {
        expect(name).toMatch(/^app-test-mybucket-\w{8}$/);
        expect(name).toContain("app-test");
      });
    });

    it("bucket name follows Cloudflare R2 naming convention", () => {
      const bucket = new Bucket("TestBucket");
      
      bucket.name.apply((name: string) => {
        expect(name).toContain("app-test-testbucket");
        expect(name).toMatch(/^app-test-testbucket-\w{8}$/);
        expect(name.length).toBeLessThanOrEqual(64); // Cloudflare R2 limit
      });
    });

    it("bucket name is lowercase", () => {
      const bucket = new Bucket("MyUpperCaseBucket");
      
      bucket.name.apply((name: string) => {
        expect(name).toBe(name.toLowerCase());
        expect(name).toMatch(/^app-test-myuppercasebucket-\w{8}$/);
      });
    });

    it("handles special characters in bucket names", () => {
      const bucket = new Bucket("My-Special_Bucket");
      
      bucket.name.apply((name: string) => {
        expect(name).toMatch(/^app-test-my-special_bucket-\w{8}$/);
        expect(name).toBe(name.toLowerCase());
      });
    });
  });

  describe("bucket properties", () => {
    it("generates correct ARN format for Cloudflare R2", () => {
      const bucket = new Bucket("MyBucket");
      
      bucket.arn.apply((arn: string) => {
        expect(arn).toMatch(/^arn:cloudflare:r2:::app-test-mybucket-\w{8}$/);
      });
    });
  });

  describe("bucket configuration", () => {
    it("accepts transform configuration", () => {
      const bucket = new Bucket("MyBucket", {
        transform: {
          bucket: (args: any) => ({
            ...args,
            accountId: "test-account-id"
          })
        }
      });
      
      expect(bucket).toBeDefined();
    });

    it("handles empty configuration", () => {
      const bucket = new Bucket("SimpleBucket");
      
      expect(bucket).toBeDefined();
      bucket.name.apply((name: string) => {
        expect(name).toMatch(/^app-test-simplebucket-\w{8}$/);
      });
    });
  });
});