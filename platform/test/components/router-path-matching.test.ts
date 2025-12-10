import { describe, it, expect } from "vitest";

/**
 * Tests for Router path matching logic
 *
 * The Router CloudFront function uses path matching to route requests.
 * This test verifies the path segment boundary logic to ensure routes
 * only match at proper path boundaries.
 */
describe("Router path matching", () => {
  /**
   * Simulates the path matching logic from the CloudFront function
   * This is extracted from platform/src/components/aws/router.ts
   */
  function pathMatches(requestUri: string, routePath: string): boolean {
    return (
      requestUri.startsWith(routePath) &&
      (requestUri === routePath ||
        requestUri[routePath.length] === "/" ||
        routePath === "/")
    );
  }

  describe("exact matches", () => {
    it("should match exact path", () => {
      expect(pathMatches("/api", "/api")).toBe(true);
      expect(pathMatches("/travel-plan", "/travel-plan")).toBe(true);
      expect(pathMatches("/", "/")).toBe(true);
    });
  });

  describe("path segment matches", () => {
    it("should match when followed by slash", () => {
      expect(pathMatches("/api/", "/api")).toBe(true);
      expect(pathMatches("/api/users", "/api")).toBe(true);
      expect(pathMatches("/api/users/123", "/api")).toBe(true);
    });

    it("should match nested paths", () => {
      expect(pathMatches("/travel-plan/abc", "/travel-plan")).toBe(true);
      expect(pathMatches("/travel-plan/abc/def", "/travel-plan")).toBe(true);
      expect(pathMatches("/uploads/file.txt", "/uploads")).toBe(true);
    });
  });

  describe("root path catch-all", () => {
    it("should match any path for root route", () => {
      expect(pathMatches("/", "/")).toBe(true);
      expect(pathMatches("/anything", "/")).toBe(true);
      expect(pathMatches("/foo/bar", "/")).toBe(true);
      expect(pathMatches("/api", "/")).toBe(true);
    });
  });

  describe("non-matches - path boundaries", () => {
    it("should NOT match when path continues without slash", () => {
      expect(pathMatches("/api-docs", "/api")).toBe(false);
      expect(pathMatches("/apiv2", "/api")).toBe(false);
      expect(pathMatches("/travel-plans", "/travel-plan")).toBe(false);
      expect(pathMatches("/travel-planning", "/travel-plan")).toBe(false);
    });

    it("should NOT match different paths", () => {
      expect(pathMatches("/users", "/api")).toBe(false);
      expect(pathMatches("/v2/api", "/api")).toBe(false);
      expect(pathMatches("/files", "/uploads")).toBe(false);
    });

    it("should NOT match shorter paths", () => {
      expect(pathMatches("/ap", "/api")).toBe(false);
      expect(pathMatches("/a", "/api")).toBe(false);
    });
  });

  describe("edge cases", () => {
    it("should handle trailing slashes correctly", () => {
      expect(pathMatches("/api/", "/api")).toBe(true);
      expect(pathMatches("/api/users", "/api/")).toBe(true);
    });

    it("should handle special characters in paths", () => {
      expect(pathMatches("/api/users-list", "/api")).toBe(true);
      expect(pathMatches("/api_v2", "/api")).toBe(false);
      expect(pathMatches("/api.json", "/api")).toBe(false);
    });

    it("should handle deeply nested paths", () => {
      expect(pathMatches("/a/b/c/d/e/f", "/a")).toBe(true);
      expect(pathMatches("/a/b/c/d/e/f", "/a/b")).toBe(true);
      expect(pathMatches("/a/b/c/d/e/f", "/a/b/c")).toBe(true);
    });
  });

  describe("real-world scenarios", () => {
    it("should correctly route travel-plan vs travel-plans", () => {
      // This is the bug that was fixed
      expect(pathMatches("/travel-plan/abc123", "/travel-plan")).toBe(true);
      expect(pathMatches("/travel-plans", "/travel-plan")).toBe(false);
      expect(pathMatches("/travel-plans/123", "/travel-plan")).toBe(false);
    });

    it("should correctly route API paths", () => {
      expect(pathMatches("/v1/users", "/v1")).toBe(true);
      expect(pathMatches("/v1-beta", "/v1")).toBe(false);
      expect(pathMatches("/v1-beta/users", "/v1")).toBe(false);
    });

    it("should correctly route file uploads", () => {
      expect(pathMatches("/uploads/file.pdf", "/uploads")).toBe(true);
      expect(pathMatches("/uploads-backup", "/uploads")).toBe(false);
      expect(pathMatches("/uploads-backup/file.pdf", "/uploads")).toBe(false);
    });

    it("should handle multiple similar routes", () => {
      // When you have /api and /api-docs as different routes
      // /api/users should only match /api
      expect(pathMatches("/api/users", "/api")).toBe(true);
      expect(pathMatches("/api/users", "/api-docs")).toBe(false);

      // /api-docs/intro should only match /api-docs
      expect(pathMatches("/api-docs/intro", "/api")).toBe(false);
      expect(pathMatches("/api-docs/intro", "/api-docs")).toBe(true);
    });
  });

  describe("priority testing (longest match)", () => {
    it("should support longest path matching", () => {
      // When multiple routes could match, the Router picks the longest
      // These tests verify our logic supports that behavior
      const uri = "/api/v2/users";

      expect(pathMatches(uri, "/api")).toBe(true);
      expect(pathMatches(uri, "/api/v2")).toBe(true);
      expect(pathMatches(uri, "/api/v2/users")).toBe(true);
      expect(pathMatches(uri, "/api/v3")).toBe(false);
    });
  });
});
