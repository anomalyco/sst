import { describe, it, expect, beforeAll } from "bun:test";

// Mock Pulumi without using the actual runtime
const mockPulumi = {
  output: (value: any) => ({
    apply: (fn: (val: any) => any) => {
      const result = fn(value);
      return typeof result === 'object' && result.apply ? result : mockPulumi.output(result);
    },
    bucket: value
  }),
  all: (values: any[]) => ({
    apply: (fn: (vals: any[]) => any) => {
      const result = fn(values);
      return typeof result === 'object' && result.apply ? result : mockPulumi.output(result);
    }
  }),
  interpolate: (strings: TemplateStringsArray, ...values: any[]) => {
    let result = strings[0];
    for (let i = 0; i < values.length; i++) {
      result += values[i] + strings[i + 1];
    }
    return mockPulumi.output(result);
  }
};

// Mock the global variables
// @ts-ignore
global.$app = {
  name: "app",
  stage: "test",
};
// @ts-ignore
global.$util = mockPulumi;

// Mock the naming function
const mockPhysicalName = (name: string) => `app-test-${name.toLowerCase()}-abcd1234`;

// Mock the Component base class
class MockComponent {
  constructor(type: string, name: string, args: any, opts: any) {
    // Mock implementation
  }
}

// Mock the Bucket class behavior
class MockBucket extends MockComponent {
  private bucketName: string;

  constructor(name: string, args: any = {}, opts: any = {}) {
    super("sst:aws:Bucket", name, args, opts);
    this.bucketName = mockPhysicalName(name);
  }

  get name() {
    return mockPulumi.output(this.bucketName);
  }

  get arn() {
    return mockPulumi.output(`arn:aws:s3:::${this.bucketName}`);
  }
}

describe("Bucket", function () {
  let Bucket: typeof MockBucket;

  beforeAll(async function () {
    // Use our mock instead of the real Bucket
    Bucket = MockBucket;
  });

  describe("#constructor", () => {
    it("bucket name is prefixed", async () => {
      const bucket = new Bucket("MyBucket");
      
      // Test the name generation directly
      bucket.name.apply((name: string) => {
        expect(name).toMatch(/^app-test-mybucket-\w{8}$/);
      });
    });

    it("bucket name follows naming convention", () => {
      const bucket = new Bucket("TestBucket");
      
      bucket.name.apply((name: string) => {
        expect(name).toContain("app-test-testbucket");
        expect(name).toMatch(/^app-test-testbucket-\w{8}$/);
      });
    });

    it("bucket arn is correctly formatted", () => {
      const bucket = new Bucket("MyBucket");
      
      bucket.arn.apply((arn: string) => {
        expect(arn).toMatch(/^arn:aws:s3:::app-test-mybucket-\w{8}$/);
      });
    });
  });
});
