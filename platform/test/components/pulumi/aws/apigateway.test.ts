import { describe, beforeAll, it, expect } from "vitest";
import * as pulumi from "@pulumi/pulumi";
import { setupSSTTestEnvironment, createAWSMocks } from "../helpers/pulumi-mocks";
import { createComponentTestSuite, testComponentCreation } from "../helpers/test-utils";

// Set up global test environment
setupSSTTestEnvironment("test-app", "test");

describe("AWS API Gateway Components", () => {
  let ApiGatewayV2: typeof import("../../../../src/components/aws/apigatewayv2").ApiGatewayV2;
  let ApiGatewayV1: typeof import("../../../../src/components/aws/apigatewayv1").ApiGatewayV1;

  beforeAll(async () => {
    ApiGatewayV2 = (await import("../../../../src/components/aws/apigatewayv2")).ApiGatewayV2;
    ApiGatewayV1 = (await import("../../../../src/components/aws/apigatewayv1")).ApiGatewayV1;
  });

  describe("ApiGatewayV2 Component", () => {
    describe("Basic API Creation", () => {
      it("should create a basic API with default settings", async () => {
        const api = await testComponentCreation(() => new ApiGatewayV2("TestApi"));
        expect(api).toBeDefined();
      });

      it("should create API with custom domain", async () => {
        const api = new ApiGatewayV2("TestApi", {
          domain: "api.example.com",
        });
        expect(api).toBeDefined();
      });

      it("should create API with custom domain object", async () => {
        const api = new ApiGatewayV2("TestApi", {
          domain: {
            name: "api.example.com",
            cert: "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
          },
        });
        expect(api).toBeDefined();
      });

      it("should create API with CORS configuration", async () => {
        const api = new ApiGatewayV2("TestApi", {
          cors: {
            allowCredentials: true,
            allowHeaders: ["Content-Type", "Authorization"],
            allowMethods: ["GET", "POST", "PUT", "DELETE"],
            allowOrigins: ["https://example.com", "https://app.example.com"],
            exposeHeaders: ["X-Custom-Header"],
            maxAge: "1 day",
          },
        });
        expect(api).toBeDefined();
      });

      it("should create API with simple CORS enabled", async () => {
        const api = new ApiGatewayV2("TestApi", {
          cors: true,
        });
        expect(api).toBeDefined();
      });

      it("should create API with access log configuration", async () => {
        const api = new ApiGatewayV2("TestApi", {
          accessLog: {
            retention: "forever",
          },
        });
        expect(api).toBeDefined();
      });

      it("should create API with VPC configuration", async () => {
        const api = new ApiGatewayV2("TestApi", {
          vpc: {
            securityGroups: ["sg-12345678"],
            subnets: ["subnet-12345678", "subnet-87654321"],
          },
        });
        expect(api).toBeDefined();
      });
    });

    describe("Route Management", () => {
      it("should add basic Lambda route", async () => {
        const api = new ApiGatewayV2("TestApi");
        const route = api.route("GET /", "src/handler.main");
        expect(route).toBeDefined();
      });

      it("should add route with custom handler configuration", async () => {
        const api = new ApiGatewayV2("TestApi");
        const route = api.route("POST /users", {
          handler: "src/users.create",
          memory: "512 MB",
          timeout: "30 seconds",
          environment: {
            NODE_ENV: "production",
          },
        });
        expect(route).toBeDefined();
      });

      it("should add route with IAM authentication", async () => {
        const api = new ApiGatewayV2("TestApi");
        const route = api.route("GET /protected", "src/protected.handler", {
          auth: {
            iam: true,
          },
        });
        expect(route).toBeDefined();
      });

      it("should add route with JWT authentication", async () => {
        const api = new ApiGatewayV2("TestApi", {
          auth: {
            jwt: {
              issuer: "https://auth.example.com",
              audiences: ["api.example.com"],
            },
          },
        });
        const route = api.route("GET /jwt-protected", "src/jwt.handler", {
          auth: {
            jwt: true,
          },
        });
        expect(route).toBeDefined();
      });

      it("should add route with Lambda authorizer", async () => {
        const api = new ApiGatewayV2("TestApi", {
          auth: {
            lambda: {
              function: "src/authorizer.handler",
            },
          },
        });
        const route = api.route("GET /lambda-auth", "src/lambda-auth.handler", {
          auth: {
            lambda: true,
          },
        });
        expect(route).toBeDefined();
      });

      it("should add parameterized routes", async () => {
        const api = new ApiGatewayV2("TestApi");
        const route1 = api.route("GET /users/{id}", "src/users.get");
        const route2 = api.route("PUT /users/{id}/posts/{postId}", "src/posts.update");
        expect(route1).toBeDefined();
        expect(route2).toBeDefined();
      });

      it("should add greedy routes", async () => {
        const api = new ApiGatewayV2("TestApi");
        const route = api.route("ANY /{proxy+}", "src/proxy.handler");
        expect(route).toBeDefined();
      });

      it("should add default route", async () => {
        const api = new ApiGatewayV2("TestApi");
        const route = api.route("$default", "src/default.handler");
        expect(route).toBeDefined();
      });

      it("should add URL route", async () => {
        const api = new ApiGatewayV2("TestApi");
        const route = api.routeUrl("GET /external", "https://api.external.com");
        expect(route).toBeDefined();
      });

      it("should add private route with VPC", async () => {
        const api = new ApiGatewayV2("TestApi", {
          vpc: {
            securityGroups: ["sg-12345678"],
            subnets: ["subnet-12345678"],
          },
        });
        const route = api.routePrivate(
          "GET /private",
          "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188"
        );
        expect(route).toBeDefined();
      });
    });

    describe("Transform Configuration", () => {
      it("should apply transform to API resource", async () => {
        const api = new ApiGatewayV2("TestApi", {
          transform: {
            api: {
              description: "Custom API description",
            },
          },
        });
        expect(api).toBeDefined();
      });

      it("should apply transform to stage resource", async () => {
        const api = new ApiGatewayV2("TestApi", {
          transform: {
            stage: {
              throttleSettings: {
                rateLimit: 1000,
                burstLimit: 2000,
              },
            },
          },
        });
        expect(api).toBeDefined();
      });

      it("should apply transform to route handlers", async () => {
        const api = new ApiGatewayV2("TestApi", {
          transform: {
            route: {
              handler: (args) => {
                args.memory = args.memory || "1024 MB";
                args.timeout = args.timeout || "30 seconds";
              },
            },
          },
        });
        const route = api.route("GET /", "src/handler.main");
        expect(route).toBeDefined();
      });
    });

    describe("Edge Cases and Error Handling", () => {
      it("should handle empty route configuration", async () => {
        const api = new ApiGatewayV2("TestApi", {});
        expect(api).toBeDefined();
      });

      it("should handle complex domain configuration", async () => {
        const api = new ApiGatewayV2("TestApi", {
          domain: {
            name: "api.example.com",
            cert: "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
            dns: false,
          },
        });
        expect(api).toBeDefined();
      });

      it("should handle multiple HTTP methods", async () => {
        const api = new ApiGatewayV2("TestApi");
        const methods = ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "ANY"];
        
        methods.forEach((method, index) => {
          const route = api.route(`${method} /test${index}`, "src/handler.main");
          expect(route).toBeDefined();
        });
      });

      it("should handle complex CORS configuration", async () => {
        const api = new ApiGatewayV2("TestApi", {
          cors: {
            allowCredentials: false,
            allowHeaders: ["*"],
            allowMethods: ["*"],
            allowOrigins: ["*"],
            exposeHeaders: ["X-Request-ID", "X-Response-Time"],
            maxAge: "7 days",
          },
        });
        expect(api).toBeDefined();
      });

      it("should handle long route paths", async () => {
        const api = new ApiGatewayV2("TestApi");
        const longPath = "/api/v1/organizations/{orgId}/projects/{projectId}/environments/{envId}/deployments/{deploymentId}/logs";
        const route = api.route(`GET ${longPath}`, "src/logs.handler");
        expect(route).toBeDefined();
      });

      it("should handle special characters in route paths", async () => {
        const api = new ApiGatewayV2("TestApi");
        const route = api.route("GET /api/v1/search", "src/search.handler");
        expect(route).toBeDefined();
      });

      it("should handle unicode in domain names", async () => {
        const api = new ApiGatewayV2("TestApi", {
          domain: "api.测试.com",
        });
        expect(api).toBeDefined();
      });
    });

    describe("Integration Scenarios", () => {
      it("should create API with multiple routes and auth", async () => {
        const api = new ApiGatewayV2("TestApi", {
          domain: "api.example.com",
          cors: true,
          auth: {
            jwt: {
              issuer: "https://auth.example.com",
              audiences: ["api.example.com"],
            },
          },
        });

        // Public routes
        api.route("GET /health", "src/health.handler");
        api.route("POST /webhook", "src/webhook.handler");

        // Protected routes
        api.route("GET /users", "src/users.list", { auth: { jwt: true } });
        api.route("POST /users", "src/users.create", { auth: { jwt: true } });
        api.route("GET /users/{id}", "src/users.get", { auth: { jwt: true } });

        // Admin routes
        api.route("DELETE /users/{id}", "src/users.delete", { auth: { iam: true } });

        expect(api).toBeDefined();
      });

      it("should create API with VPC and private routes", async () => {
        const api = new ApiGatewayV2("TestApi", {
          vpc: {
            securityGroups: ["sg-12345678", "sg-87654321"],
            subnets: ["subnet-12345678", "subnet-87654321", "subnet-11111111"],
          },
        });

        // Public Lambda routes
        api.route("GET /public", "src/public.handler");

        // Private routes to internal services
        api.routePrivate(
          "GET /internal",
          "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/internal-lb/50dc6c495c0c9188"
        );

        expect(api).toBeDefined();
      });

      it("should create API with comprehensive configuration", async () => {
        const api = new ApiGatewayV2("TestApi", {
          domain: {
            name: "api.example.com",
            cert: "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
          },
          cors: {
            allowCredentials: true,
            allowHeaders: ["Content-Type", "Authorization", "X-API-Key"],
            allowMethods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
            allowOrigins: ["https://app.example.com", "https://admin.example.com"],
            maxAge: "1 day",
          },
          accessLog: {
            retention: "3 months",
          },
          auth: {
            jwt: {
              issuer: "https://auth.example.com",
              audiences: ["api.example.com"],
              identitySource: "$request.header.Authorization",
            },
          },
          transform: {
            api: {
              description: "Production API Gateway",
            },
            stage: {
              throttleSettings: {
                rateLimit: 10000,
                burstLimit: 20000,
              },
            },
            route: {
              handler: (args) => {
                args.environment = {
                  ...args.environment,
                  NODE_ENV: "production",
                  LOG_LEVEL: "info",
                };
              },
            },
          },
        });

        expect(api).toBeDefined();
      });
    });

    describe("SST Naming Conventions", () => {
      it("should follow SST naming patterns", async () => {
        const api = new ApiGatewayV2("MyTestApi");
        expect(api).toBeDefined();
        expect(api.constructorName).toBe("MyTestApi");
      });

      it("should handle long component names", async () => {
        const api = new ApiGatewayV2("VeryLongApiGatewayComponentNameForTesting");
        expect(api).toBeDefined();
      });

      it("should handle names with numbers", async () => {
        const api = new ApiGatewayV2("ApiV2Gateway123");
        expect(api).toBeDefined();
      });
    });
  });

  describe("ApiGatewayV1 Component", () => {
    describe("Basic API Creation", () => {
      it("should create a basic REST API with default settings", async () => {
        const api = await testComponentCreation(() => new ApiGatewayV1("TestRestApi"));
        expect(api).toBeDefined();
      });

      it("should create REST API with custom domain", async () => {
        const api = new ApiGatewayV1("TestRestApi", {
          domain: "api.example.com",
        });
        expect(api).toBeDefined();
      });

      it("should create REST API with CORS configuration", async () => {
        const api = new ApiGatewayV1("TestRestApi", {
          cors: {
            allowCredentials: true,
            allowHeaders: ["Content-Type", "Authorization"],
            allowMethods: ["GET", "POST", "PUT", "DELETE"],
            allowOrigins: ["https://example.com"],
            exposeHeaders: ["X-Custom-Header"],
            maxAge: "1 day",
          },
        });
        expect(api).toBeDefined();
      });

      it("should create REST API with access log configuration", async () => {
        const api = new ApiGatewayV1("TestRestApi", {
          accessLog: {
            retention: "forever",
          },
        });
        expect(api).toBeDefined();
      });
    });

    describe("Route Management", () => {
      it("should add basic Lambda route", async () => {
        const api = new ApiGatewayV1("TestRestApi");
        const route = api.route("GET /", "src/handler.main");
        expect(route).toBeDefined();
      });

      it("should add route with custom handler configuration", async () => {
        const api = new ApiGatewayV1("TestRestApi");
        const route = api.route("POST /users", {
          handler: "src/users.create",
          memory: "512 MB",
          timeout: "30 seconds",
        });
        expect(route).toBeDefined();
      });

      it("should add route with IAM authentication", async () => {
        const api = new ApiGatewayV1("TestRestApi");
        const route = api.route("GET /protected", "src/protected.handler", {
          auth: {
            iam: true,
          },
        });
        expect(route).toBeDefined();
      });

      it("should add parameterized routes", async () => {
        const api = new ApiGatewayV1("TestRestApi");
        const route1 = api.route("GET /users/{id}", "src/users.get");
        const route2 = api.route("PUT /users/{id}/posts/{postId}", "src/posts.update");
        expect(route1).toBeDefined();
        expect(route2).toBeDefined();
      });

      it("should add greedy routes", async () => {
        const api = new ApiGatewayV1("TestRestApi");
        const route = api.route("ANY /{proxy+}", "src/proxy.handler");
        expect(route).toBeDefined();
      });
    });

    describe("Transform Configuration", () => {
      it("should apply transform to API resource", async () => {
        const api = new ApiGatewayV1("TestRestApi", {
          transform: {
            api: {
              description: "Custom REST API description",
            },
          },
        });
        expect(api).toBeDefined();
      });

      it("should apply transform to deployment resource", async () => {
        const api = new ApiGatewayV1("TestRestApi", {
          transform: {
            deployment: {
              description: "Custom deployment",
            },
          },
        });
        expect(api).toBeDefined();
      });
    });

    describe("Edge Cases and Error Handling", () => {
      it("should handle empty configuration", async () => {
        const api = new ApiGatewayV1("TestRestApi", {});
        expect(api).toBeDefined();
      });

      it("should handle complex domain configuration", async () => {
        const api = new ApiGatewayV1("TestRestApi", {
          domain: {
            name: "api.example.com",
            cert: "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
            dns: false,
          },
        });
        expect(api).toBeDefined();
      });

      it("should handle multiple HTTP methods", async () => {
        const api = new ApiGatewayV1("TestRestApi");
        const methods = ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "ANY"];
        
        methods.forEach((method, index) => {
          const route = api.route(`${method} /test${index}`, "src/handler.main");
          expect(route).toBeDefined();
        });
      });
    });

    describe("Integration Scenarios", () => {
      it("should create REST API with multiple routes and auth", async () => {
        const api = new ApiGatewayV1("TestRestApi", {
          domain: "api.example.com",
          cors: true,
        });

        // Public routes
        api.route("GET /health", "src/health.handler");
        api.route("POST /webhook", "src/webhook.handler");

        // Protected routes
        api.route("GET /users", "src/users.list", { auth: { iam: true } });
        api.route("POST /users", "src/users.create", { auth: { iam: true } });

        expect(api).toBeDefined();
      });

      it("should create REST API with comprehensive configuration", async () => {
        const api = new ApiGatewayV1("TestRestApi", {
          domain: {
            name: "api.example.com",
            cert: "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
          },
          cors: {
            allowCredentials: true,
            allowHeaders: ["Content-Type", "Authorization"],
            allowMethods: ["GET", "POST", "PUT", "DELETE"],
            allowOrigins: ["https://app.example.com"],
            maxAge: "1 day",
          },
          accessLog: {
            retention: "3 months",
          },
          transform: {
            api: {
              description: "Production REST API Gateway",
            },
          },
        });

        expect(api).toBeDefined();
      });
    });

    describe("SST Naming Conventions", () => {
      it("should follow SST naming patterns", async () => {
        const api = new ApiGatewayV1("MyTestRestApi");
        expect(api).toBeDefined();
      });
    });
  });

  describe("Component Comparison", () => {
    it("should create both V1 and V2 APIs with similar configurations", async () => {
      const apiV1 = new ApiGatewayV1("TestRestApi", {
        domain: "rest.example.com",
        cors: true,
      });

      const apiV2 = new ApiGatewayV2("TestHttpApi", {
        domain: "http.example.com",
        cors: true,
      });

      expect(apiV1).toBeDefined();
      expect(apiV2).toBeDefined();
    });

    it("should handle different route patterns between V1 and V2", async () => {
      const apiV1 = new ApiGatewayV1("TestRestApi");
      const apiV2 = new ApiGatewayV2("TestHttpApi");

      // Both should support basic routes
      const routeV1 = apiV1.route("GET /users", "src/users.handler");
      const routeV2 = apiV2.route("GET /users", "src/users.handler");

      expect(routeV1).toBeDefined();
      expect(routeV2).toBeDefined();
    });
  });
});