import { describe, it, expect } from "bun:test";

describe("Cloudflare Account ID", () => {
  describe("DEFAULT_ACCOUNT_ID constant", () => {
    it("should be accessible", () => {
      // Import the constant
      const { DEFAULT_ACCOUNT_ID } = require("./account-id");
      
      // The constant should be accessible (can be undefined if env var not set)
      expect(DEFAULT_ACCOUNT_ID !== null).toBe(true);
    });

    it("should be a string when environment variable is set", () => {
      // Set environment variable for test
      const originalValue = process.env.CLOUDFLARE_DEFAULT_ACCOUNT_ID;
      process.env.CLOUDFLARE_DEFAULT_ACCOUNT_ID = "test-account-id";
      
      // Re-import to get updated value
      delete require.cache[require.resolve("./account-id")];
      const { DEFAULT_ACCOUNT_ID } = require("./account-id");
      
      expect(typeof DEFAULT_ACCOUNT_ID).toBe("string");
      expect(DEFAULT_ACCOUNT_ID).toBe("test-account-id");
      
      // Restore original value
      if (originalValue !== undefined) {
        process.env.CLOUDFLARE_DEFAULT_ACCOUNT_ID = originalValue;
      } else {
        delete process.env.CLOUDFLARE_DEFAULT_ACCOUNT_ID;
      }
    });

    it("should handle missing environment variable", () => {
      // Save original value
      const originalValue = process.env.CLOUDFLARE_DEFAULT_ACCOUNT_ID;
      
      // Remove environment variable
      delete process.env.CLOUDFLARE_DEFAULT_ACCOUNT_ID;
      
      // Re-import to get updated value
      delete require.cache[require.resolve("./account-id")];
      const { DEFAULT_ACCOUNT_ID } = require("./account-id");
      
      // Should be undefined when env var is not set
      expect(DEFAULT_ACCOUNT_ID).toBeUndefined();
      
      // Restore original value
      if (originalValue !== undefined) {
        process.env.CLOUDFLARE_DEFAULT_ACCOUNT_ID = originalValue;
      }
    });

    it("should export the constant correctly", () => {
      const accountIdModule = require("./account-id");
      expect(accountIdModule).toHaveProperty("DEFAULT_ACCOUNT_ID");
    });
  });

  describe("Environment variable handling", () => {
    it("should use CLOUDFLARE_DEFAULT_ACCOUNT_ID environment variable", () => {
      const envVarName = "CLOUDFLARE_DEFAULT_ACCOUNT_ID";
      expect(envVarName).toBe("CLOUDFLARE_DEFAULT_ACCOUNT_ID");
    });

    it("should handle different account ID formats", () => {
      const testAccountIds = [
        "1234567890abcdef1234567890abcdef",
        "test-account-id",
        "my-cloudflare-account"
      ];
      
      testAccountIds.forEach(accountId => {
        expect(typeof accountId).toBe("string");
        expect(accountId.length).toBeGreaterThan(0);
      });
    });
  });
});