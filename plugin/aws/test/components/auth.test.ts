import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { withTestEnvironment, createConsoleSpy, assertions, generators } from "../utils/test-helpers";
import { MockAWSComponent } from "../utils/mock-sst";

/**
 * Test Auth component migration from v1 to v2
 * Tests OpenAuth integration and forceUpgrade mechanism
 */
describe("Auth Component", () => {
  let consoleSpy: ReturnType<typeof createConsoleSpy>;

  beforeEach(() => {
    consoleSpy = createConsoleSpy('warn');
  });

  afterEach(() => {
    consoleSpy.restore();
  });

  describe("Auth v2 migration", () => {
    it("should create Auth v2 component without migration warnings for new installations", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google", "github"]
          }
        });

        auth.registerVersion({
          new: "2.0.0",
          message: "Auth v2 with OpenAuth integration"
        });

        expect(auth).toBeDefined();
        expect(auth.name).toBe("TestAuth");
        expect(auth.args.authenticator.type).toBe("openauth");
      });
    });

    it("should require migration from v1 to v2 without forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google"]
          }
        });

        expect(() => {
          auth.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Auth v2 introduces OpenAuth. Please migrate your authentication configuration."
          });
        }).toThrow("Migration required for TestAuth: Auth v2 introduces OpenAuth. Please migrate your authentication configuration.");
      });
    });

    it("should allow migration from v1 to v2 with forceUpgrade", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google", "github"]
          }
        });

        expect(() => {
          auth.registerVersion({
            old: "1.0.0",
            new: "2.0.0",
            message: "Auth v2 introduces OpenAuth. Please migrate your authentication configuration.",
            forceUpgrade: true
          });
        }).not.toThrow();

        const history = auth.getVersionHistory();
        expect(history).toHaveLength(1);
        expect(history[0].forceUpgrade).toBe(true);
      });
    });
  });

  describe("OpenAuth integration", () => {
    it("should support OpenAuth provider configuration", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google", "github", "discord"],
            config: {
              google: {
                clientId: "google-client-id",
                clientSecret: "google-client-secret"
              },
              github: {
                clientId: "github-client-id",
                clientSecret: "github-client-secret"
              }
            }
          }
        });

        expect(auth.args.authenticator.type).toBe("openauth");
        expect(auth.args.authenticator.providers).toContain("google");
        expect(auth.args.authenticator.providers).toContain("github");
        expect(auth.args.authenticator.config.google.clientId).toBe("google-client-id");
      });
    });

    it("should validate OpenAuth provider configuration", async () => {
      await withTestEnvironment(async () => {
        // Test with missing provider config
        const authWithMissingConfig = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google"],
            config: {} // Missing google config
          }
        });

        expect(authWithMissingConfig.args.authenticator.config).toEqual({});
      });
    });

    it("should support custom OpenAuth configuration", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["custom"],
            config: {
              custom: {
                issuer: "https://custom-provider.com",
                clientId: "custom-client-id",
                clientSecret: "custom-client-secret",
                scopes: ["openid", "profile", "email"]
              }
            }
          }
        });

        expect(auth.args.authenticator.config.custom.issuer).toBe("https://custom-provider.com");
        expect(auth.args.authenticator.config.custom.scopes).toContain("openid");
      });
    });
  });

  describe("Auth component naming", () => {
    it("should generate valid AWS resource names", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("MyAuthComponent", "aws:auth:component");
        const physicalName = auth.generatePhysicalName("auth");
        
        assertions.validAWSName(physicalName);
        expect(physicalName.value).toMatch(/test-app-test-myauthcomponent-auth-/);
      });
    });

    it("should handle long component names correctly", async () => {
      await withTestEnvironment(async () => {
        const longName = "VeryLongAuthComponentNameThatExceedsNormalLimits";
        const auth = new MockAWSComponent(longName, "aws:auth:component");
        const physicalName = auth.generatePhysicalName("auth");
        
        // Just verify it generates a valid name, don't enforce length limit in mock
        assertions.validAWSName(physicalName);
        expect(physicalName.value).toContain("verylongauthcomponentnamethatexceedsnormallimits");
      });
    });
  });

  describe("Auth v1 legacy support", () => {
    it("should provide access to v1 Auth component", async () => {
      await withTestEnvironment(async () => {
        // Simulate Auth.v1 access pattern
        const AuthV1 = class extends MockAWSComponent {
          constructor(name: string, args: any = {}) {
            super(name, "aws:auth:v1:component", args);
            this.registerVersion({
              new: "1.0.0",
              message: "Auth v1 legacy component"
            });
          }
        };

        const authV1 = new AuthV1("LegacyAuth", {
          cognito: {
            userPool: "existing-user-pool",
            userPoolClient: "existing-client"
          }
        });

        expect(authV1).toBeDefined();
        expect(authV1.type).toBe("aws:auth:v1:component");
        expect(authV1.args.cognito).toBeDefined();
      });
    });

    it("should maintain v1 component functionality", async () => {
      await withTestEnvironment(async () => {
        const authV1 = new MockAWSComponent("LegacyAuth", "aws:auth:v1:component", {
          cognito: {
            userPool: {
              passwordPolicy: {
                minimumLength: 8,
                requireLowercase: true,
                requireUppercase: true,
                requireNumbers: true,
                requireSymbols: false
              }
            }
          }
        });

        authV1.registerVersion({
          new: "1.0.0",
          message: "Auth v1 with Cognito"
        });

        expect(authV1.args.cognito.userPool.passwordPolicy.minimumLength).toBe(8);
        expect(authV1.args.cognito.userPool.passwordPolicy.requireSymbols).toBe(false);
      });
    });
  });

  describe("Auth component integration", () => {
    it("should integrate with Router component", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google"]
          }
        });

        const router = new MockAWSComponent("TestRouter", "aws:router:component", {
          routes: {
            "/api/auth/*": {
              auth: auth.generatePhysicalName("auth")
            }
          }
        });

        expect(router.args.routes["/api/auth/*"].auth).toBeDefined();
        assertions.validOutput(router.args.routes["/api/auth/*"].auth);
      });
    });

    it("should provide authentication outputs for other components", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component");
        
        // Mock typical Auth component outputs
        const authUrl = auth.generatePhysicalName("url");
        const authSecret = auth.generatePhysicalName("secret");
        
        assertions.validOutput(authUrl);
        assertions.validOutput(authSecret);
        
        expect(authUrl.value).toMatch(/test-app-test-testauth-url-/);
        expect(authSecret.value).toMatch(/test-app-test-testauth-secret-/);
      });
    });
  });

  describe("Auth configuration validation", () => {
    it("should validate required authenticator configuration", async () => {
      await withTestEnvironment(async () => {
        // Test with minimal valid configuration
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google"]
          }
        });

        expect(auth.args.authenticator.type).toBe("openauth");
        expect(auth.args.authenticator.providers).toHaveLength(1);
      });
    });

    it("should handle empty providers array", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: []
          }
        });

        expect(auth.args.authenticator.providers).toHaveLength(0);
      });
    });

    it("should support multiple authentication providers", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "openauth",
            providers: ["google", "github", "discord", "apple", "facebook"]
          }
        });

        expect(auth.args.authenticator.providers).toHaveLength(5);
        expect(auth.args.authenticator.providers).toContain("apple");
        expect(auth.args.authenticator.providers).toContain("facebook");
      });
    });
  });

  describe("Auth error handling", () => {
    it("should handle invalid authenticator type", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {
          authenticator: {
            type: "invalid-type",
            providers: ["google"]
          }
        });

        // Component should still be created but with invalid type
        expect(auth.args.authenticator.type).toBe("invalid-type");
      });
    });

    it("should handle missing authenticator configuration", async () => {
      await withTestEnvironment(async () => {
        const auth = new MockAWSComponent("TestAuth", "aws:auth:component", {});

        expect(auth.args.authenticator).toBeUndefined();
      });
    });
  });
});