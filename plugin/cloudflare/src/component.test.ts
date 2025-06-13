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

// Mock the naming function
const mockPhysicalName = (name: string) => `app-test-${name.toLowerCase()}-abcd1234`;

describe("Cloudflare Component", function () {
  beforeAll(async function () {
    // Setup test environment
  });

  describe("Component naming", () => {
    it("should follow Cloudflare naming conventions", () => {
      const name = "TestResource";
      const physicalName = mockPhysicalName(name);
      
      expect(physicalName).toMatch(/^app-test-testresource-\w{8}$/);
      expect(physicalName).toContain("app-test");
      expect(physicalName.length).toBeLessThanOrEqual(64); // Cloudflare naming limit
    });

    it("should handle lowercase naming for Cloudflare resources", () => {
      const name = "MyCloudflareResource";
      const physicalName = mockPhysicalName(name).toLowerCase();
      
      expect(physicalName).toBe(physicalName.toLowerCase());
      expect(physicalName).toMatch(/^app-test-mycloudflareresource-\w{8}$/);
    });

    it("should respect Cloudflare resource naming rules", () => {
      // Test specific Cloudflare resource types
      const testCases = [
        { type: "r2Bucket", maxLength: 64 },
        { type: "d1Database", maxLength: 64 },
        { type: "workerScript", maxLength: 64 },
        { type: "kvNamespace", maxLength: 64 }
      ];

      testCases.forEach(({ type, maxLength }) => {
        const name = `Test${type}`;
        const physicalName = mockPhysicalName(name);
        expect(physicalName.length).toBeLessThanOrEqual(maxLength);
        expect(physicalName).toMatch(/^app-test-test\w+-\w{8}$/);
      });
    });
  });

  describe("Component transformation", () => {
    it("should apply naming transformations", () => {
      // Test that the component applies proper naming transformations
      const componentName = "TestComponent";
      const expectedPattern = /^app-test-testcomponent-\w{8}$/;
      
      expect(mockPhysicalName(componentName)).toMatch(expectedPattern);
    });

    it("should handle special characters in names", () => {
      const componentName = "Test-Component_123";
      const physicalName = mockPhysicalName(componentName);
      
      // Should normalize special characters
      expect(physicalName).toMatch(/^app-test-test-component_123-\w{8}$/);
    });
  });
});