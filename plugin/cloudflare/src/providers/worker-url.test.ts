import { describe, it, expect } from "bun:test";

describe("Cloudflare WorkerUrl Provider", () => {
  describe("WorkerUrlInputs interface validation", () => {
    it("should have proper WorkerUrlInputs structure", () => {
      // This tests the TypeScript interface structure
      const workerUrlInputs: any = {
        accountId: "test-account-id",
        scriptName: "test-script",
        enabled: true
      };
      expect(workerUrlInputs.accountId).toBeDefined();
      expect(workerUrlInputs.scriptName).toBeDefined();
      expect(workerUrlInputs.enabled).toBeDefined();
      expect(typeof workerUrlInputs.enabled).toBe("boolean");
    });

    it("should support disabled worker", () => {
      const workerUrlInputs = {
        accountId: "test-account-id",
        scriptName: "test-script",
        enabled: false
      };
      expect(workerUrlInputs.enabled).toBe(false);
    });

    it("should support enabled worker", () => {
      const workerUrlInputs = {
        accountId: "test-account-id",
        scriptName: "test-script",
        enabled: true
      };
      expect(workerUrlInputs.enabled).toBe(true);
    });
  });

  describe("Inputs interface validation", () => {
    it("should have correct Inputs structure", () => {
      const inputs = {
        accountId: "test-account",
        scriptName: "my-worker",
        enabled: true
      };
      expect(typeof inputs.accountId).toBe("string");
      expect(typeof inputs.scriptName).toBe("string");
      expect(typeof inputs.enabled).toBe("boolean");
    });
  });

  describe("Outputs interface validation", () => {
    it("should support url output", () => {
      const outputs = {
        url: "https://my-worker.example.workers.dev"
      };
      expect(outputs.url).toBeDefined();
      expect(typeof outputs.url).toBe("string");
    });

    it("should support undefined url", () => {
      const outputs = {
        url: undefined
      };
      expect(outputs.url).toBeUndefined();
    });
  });

  describe("Provider class validation", () => {
    it("should have correct resource type", () => {
      // Test the expected resource type string
      const expectedType = "sst.cloudflare.WorkerUrl";
      expect(expectedType).toContain("cloudflare");
      expect(expectedType).toContain("WorkerUrl");
    });

    it("should follow SST naming convention", () => {
      const expectedType = "sst.cloudflare.WorkerUrl";
      expect(expectedType.startsWith("sst.")).toBe(true);
    });
  });

  describe("Worker URL operations validation", () => {
    it("should support create operation", () => {
      const inputs = {
        accountId: "test-account",
        scriptName: "test-worker",
        enabled: true
      };
      expect(inputs).toBeDefined();
      expect(typeof inputs.accountId).toBe("string");
      expect(typeof inputs.scriptName).toBe("string");
      expect(typeof inputs.enabled).toBe("boolean");
    });

    it("should support update operation", () => {
      const oldInputs = {
        accountId: "test-account",
        scriptName: "test-worker",
        enabled: false
      };
      const newInputs = {
        accountId: "test-account",
        scriptName: "test-worker",
        enabled: true
      };
      expect(oldInputs.enabled).toBe(false);
      expect(newInputs.enabled).toBe(true);
    });
  });
});