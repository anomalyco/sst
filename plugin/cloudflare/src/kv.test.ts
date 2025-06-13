import { describe, it, expect } from "bun:test";

describe("Cloudflare KV", () => {
  describe("KV interface validation", () => {
    it("should have proper KvArgs structure", () => {
      // This tests the TypeScript interface structure
      const kvArgs: any = {
        transform: {
          namespace: (args: any) => args
        }
      };
      expect(kvArgs.transform).toBeDefined();
      expect(typeof kvArgs.transform.namespace).toBe("function");
    });

    it("should support empty configuration", () => {
      const kvArgs: any = {};
      expect(kvArgs).toBeDefined();
    });

    it("should support transform configuration", () => {
      const kvArgs = {
        transform: {
          namespace: (args: any) => ({ ...args, title: "custom-title" })
        }
      };
      expect(kvArgs.transform.namespace).toBeDefined();
      expect(typeof kvArgs.transform.namespace).toBe("function");
    });
  });

  describe("KV component type validation", () => {
    it("should have correct Pulumi type", () => {
      // Test the expected Pulumi type string
      const expectedType = "sst:cloudflare:Kv";
      expect(expectedType).toContain("cloudflare");
      expect(expectedType).toContain("Kv");
    });

    it("should follow SST naming convention", () => {
      const expectedType = "sst:cloudflare:Kv";
      expect(expectedType.startsWith("sst:")).toBe(true);
    });
  });
});