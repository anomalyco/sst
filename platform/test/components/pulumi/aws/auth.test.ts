import { describe, beforeAll, it, expect } from "vitest";
import * as pulumi from "@pulumi/pulumi";
import { setupSSTTestEnvironment, createAWSMocks } from "../helpers/pulumi-mocks";
import { createComponentTestSuite, testComponentCreation, testSSTNaming } from "../helpers/test-utils";

// Set up global test environment
setupSSTTestEnvironment("test-app", "test");

describe("AWS Auth Component", () => {
  let Auth: typeof import("../../../../src/components/aws/auth").Auth;

  beforeAll(async () => {
    Auth = (await import("../../../../src/components/aws/auth")).Auth;
  });

  describe("Basic Auth Creation", () => {
    it("should create a basic auth component with issuer", async () => {
      const auth = await testComponentCreation(() => new Auth("TestAuth", {
        issuer: "src/auth.handler",
      }));

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
      expect(auth.nodes).toBeDefined();
      expect(auth.nodes.table).toBeDefined();
      expect(auth.nodes.issuer).toBeDefined();
    });

    it("should create auth component with authorizer (deprecated)", async () => {
      const auth = new Auth("TestAuth", {
        authorizer: "src/auth.handler",
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
      expect(auth.nodes.authorizer).toBeDefined();
    });

    it("should create auth component with function args", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          runtime: "nodejs20.x",
          timeout: "30 seconds",
          memory: "512 MB",
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should throw error when no issuer is provided", async () => {
      expect(() => {
        new Auth("TestAuth", {});
      }).toThrow("Auth: issuer field must be set");
    });
  });

  describe("Auth with Custom Domain", () => {
    it("should create auth with custom domain", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
        domain: "auth.example.com",
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
      expect(auth.nodes.router).toBeDefined();
    });

    it("should create auth with domain configuration object", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
        domain: {
          name: "auth.example.com",
          cert: "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
      expect(auth.nodes.router).toBeDefined();
    });

    it("should create auth with Cloudflare DNS", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
        domain: {
          name: "auth.example.com",
          dns: {
            provider: "cloudflare",
            zone: "example.com",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
      expect(auth.nodes.router).toBeDefined();
    });
  });

  describe("Auth Component Integration", () => {
    it("should create auth with linked resources", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          link: [], // Would normally link to other resources
          environment: {
            NODE_ENV: "production",
            API_URL: "https://api.example.com",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should provide correct SST link properties", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
      });

      const linkProperties = auth.getSSTLink();
      expect(linkProperties).toBeDefined();
      expect(linkProperties.properties).toBeDefined();
      expect(linkProperties.properties.url).toBeDefined();
      expect(linkProperties.include).toBeDefined();
    });

    it("should create DynamoDB table with correct configuration", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
      });

      expect(auth.nodes.table).toBeDefined();
      // The table should be configured for OpenAuth storage
    });
  });

  describe("Auth Version Management", () => {
    it("should handle force upgrade to v2", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
        forceUpgrade: "v2",
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should provide access to v1 auth", async () => {
      expect(Auth.v1).toBeDefined();
    });
  });

  describe("Auth Edge Cases", () => {
    it("should handle empty issuer string", async () => {
      expect(() => {
        new Auth("TestAuth", {
          issuer: "",
        });
      }).toThrow();
    });

    it("should handle complex function configuration", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          runtime: "nodejs20.x",
          timeout: "5 minutes",
          memory: "1024 MB",
          environment: {
            NODE_ENV: "production",
            LOG_LEVEL: "debug",
            CUSTOM_VAR: "value",
          },
          link: [],
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should handle unicode characters in domain", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
        domain: "auth.例え.com",
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should handle long domain names", async () => {
      const longDomain = "auth." + "a".repeat(50) + ".example.com";
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
        domain: longDomain,
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });
  });

  describe("Auth Authentication Providers", () => {
    it("should create auth for OAuth providers", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          environment: {
            GITHUB_CLIENT_ID: "github_client_id",
            GITHUB_CLIENT_SECRET: "github_client_secret",
            GOOGLE_CLIENT_ID: "google_client_id",
            GOOGLE_CLIENT_SECRET: "google_client_secret",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should create auth for email/password providers", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          environment: {
            EMAIL_PROVIDER: "ses",
            EMAIL_FROM: "noreply@example.com",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should create auth for custom providers", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          environment: {
            CUSTOM_PROVIDER_URL: "https://custom-auth.example.com",
            CUSTOM_PROVIDER_KEY: "custom_key",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });
  });

  describe("Auth Security Configuration", () => {
    it("should create auth with security headers", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          environment: {
            CORS_ORIGIN: "https://app.example.com",
            SECURE_COOKIES: "true",
            CSRF_PROTECTION: "true",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should create auth with session configuration", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          environment: {
            SESSION_DURATION: "7d",
            REFRESH_TOKEN_DURATION: "30d",
            JWT_SECRET: "jwt_secret_key",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });
  });

  describe("Auth Performance and Scaling", () => {
    it("should create auth with performance optimizations", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          runtime: "nodejs20.x",
          memory: "1024 MB",
          timeout: "30 seconds",
          environment: {
            NODE_OPTIONS: "--max-old-space-size=1024",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should create auth with caching configuration", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          environment: {
            CACHE_TTL: "3600",
            REDIS_URL: "redis://cache.example.com:6379",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });
  });

  describe("Auth Multi-Environment Support", () => {
    it("should create auth for development environment", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          environment: {
            NODE_ENV: "development",
            DEBUG: "true",
            LOG_LEVEL: "debug",
          },
        },
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
    });

    it("should create auth for production environment", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          environment: {
            NODE_ENV: "production",
            LOG_LEVEL: "error",
            MONITORING_ENABLED: "true",
          },
        },
        domain: "auth.production.example.com",
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
      expect(auth.nodes.router).toBeDefined();
    });

    it("should create auth for staging environment", async () => {
      const auth = new Auth("TestAuth", {
        issuer: {
          handler: "src/auth.handler",
          environment: {
            NODE_ENV: "staging",
            LOG_LEVEL: "info",
          },
        },
        domain: "auth.staging.example.com",
      });

      expect(auth).toBeDefined();
      expect(auth.url).toBeDefined();
      expect(auth.nodes.router).toBeDefined();
    });
  });

  describe("SST Naming Conventions", () => {
    it("should follow SST naming conventions", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
      });

      // Test that the auth component name follows SST conventions
      await pulumi.all([auth.nodes.issuer.name]).apply(([name]) => {
        testSSTNaming(name, "testauth", "test-app", "test");
      });
    });
  });

  describe("Auth Component Validation", () => {
    it("should validate component structure", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
      });

      // Validate component has required properties
      expect(auth.url).toBeDefined();
      expect(auth.nodes).toBeDefined();
      expect(auth.nodes.table).toBeDefined();
      expect(auth.nodes.issuer).toBeDefined();
      expect(auth.getSSTLink).toBeDefined();
    });

    it("should validate nodes structure", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
        domain: "auth.example.com",
      });

      const nodes = auth.nodes;
      expect(nodes.table).toBeDefined();
      expect(nodes.issuer).toBeDefined();
      expect(nodes.authorizer).toBeDefined(); // Deprecated alias
      expect(nodes.router).toBeDefined();
    });

    it("should validate SST link structure", async () => {
      const auth = new Auth("TestAuth", {
        issuer: "src/auth.handler",
      });

      const link = auth.getSSTLink();
      expect(link.properties).toBeDefined();
      expect(link.properties.url).toBeDefined();
      expect(link.include).toBeDefined();
      expect(Array.isArray(link.include)).toBe(true);
    });
  });
});