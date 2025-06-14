package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sst/sst/v3/test/integration/pulumi/helpers"
)

func TestAPIGatewayV2Deployment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_AWS_ACCOUNT_ID") == "" {
		t.Skip("Skipping AWS integration test - SST_TEST_AWS_ACCOUNT_ID not set")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	ctx := context.Background()
	testStage := fmt.Sprintf("api-v2-%d", time.Now().Unix())
	
	// Create test project with API Gateway v2 (HTTP API)
	projectDir, err := helpers.CreateTestProject("aws-api-v2", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-api-v2",
      region: "us-east-1",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const api = new sst.aws.ApiGatewayV2("TestAPI", {
        routes: {
          "GET /": "index.handler",
          "POST /users": "users.create",
          "GET /users/{id}": "users.get",
          "PUT /users/{id}": "users.update",
          "DELETE /users/{id}": "users.delete",
        },
        cors: {
          allowCredentials: true,
          allowHeaders: ["content-type", "authorization"],
          allowMethods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
          allowOrigins: ["*"],
        },
      });

      return {
        apiUrl: api.url,
        apiId: api.id,
        apiName: api.name,
      };
    });
  },
};`,
		"index.js": `
export const handler = async (event) => {
  console.log("Root handler called:", JSON.stringify(event, null, 2));
  
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      message: "Hello from API Gateway v2!",
      path: event.rawPath,
      method: event.requestContext.http.method,
      timestamp: new Date().toISOString(),
    }),
  };
};`,
		"users.js": `
export const create = async (event) => {
  console.log("Create user:", JSON.stringify(event, null, 2));
  
  const body = event.body ? JSON.parse(event.body) : {};
  
  return {
    statusCode: 201,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      message: "User created",
      user: {
        id: "user-123",
        name: body.name || "Test User",
        email: body.email || "test@example.com",
      },
      timestamp: new Date().toISOString(),
    }),
  };
};

export const get = async (event) => {
  console.log("Get user:", JSON.stringify(event, null, 2));
  
  const userId = event.pathParameters?.id || "unknown";
  
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      message: "User retrieved",
      user: {
        id: userId,
        name: "Test User",
        email: "test@example.com",
        createdAt: "2024-01-01T00:00:00Z",
      },
      timestamp: new Date().toISOString(),
    }),
  };
};

export const update = async (event) => {
  console.log("Update user:", JSON.stringify(event, null, 2));
  
  const userId = event.pathParameters?.id || "unknown";
  const body = event.body ? JSON.parse(event.body) : {};
  
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      message: "User updated",
      user: {
        id: userId,
        name: body.name || "Updated User",
        email: body.email || "updated@example.com",
        updatedAt: new Date().toISOString(),
      },
      timestamp: new Date().toISOString(),
    }),
  };
};

export const delete = async (event) => {
  console.log("Delete user:", JSON.stringify(event, null, 2));
  
  const userId = event.pathParameters?.id || "unknown";
  
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      message: "User deleted",
      userId: userId,
      timestamp: new Date().toISOString(),
    }),
  };
};`,
		"package.json": `{
  "name": "test-api-v2",
  "version": "1.0.0",
  "type": "module",
  "dependencies": {}
}`,
	})
	require.NoError(t, err)
	defer helpers.CleanupTestProject(projectDir)

	// Deploy the project
	err = helpers.DeployProject(ctx, projectDir, testStage)
	require.NoError(t, err)
	defer helpers.RemoveProject(ctx, projectDir, testStage)

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, testStage)
	require.NoError(t, err)

	apiUrl, ok := outputs["apiUrl"].(string)
	require.True(t, ok, "apiUrl output should be a string")
	require.NotEmpty(t, apiUrl, "apiUrl should not be empty")
	require.True(t, strings.HasPrefix(apiUrl, "https://"), "API URL should use HTTPS")

	apiId, ok := outputs["apiId"].(string)
	require.True(t, ok, "apiId output should be a string")
	require.NotEmpty(t, apiId, "apiId should not be empty")

	// Test HTTP endpoints
	t.Run("TestRootEndpoint", func(t *testing.T) {
		resp, err := http.Get(apiUrl)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "Hello from API Gateway v2!", response["message"])
		assert.Equal(t, "/", response["path"])
		assert.Equal(t, "GET", response["method"])
		assert.NotEmpty(t, response["timestamp"])
	})

	t.Run("TestCreateUser", func(t *testing.T) {
		userPayload := map[string]string{
			"name":  "John Doe",
			"email": "john@example.com",
		}
		payloadBytes, err := json.Marshal(userPayload)
		require.NoError(t, err)

		resp, err := http.Post(apiUrl+"/users", "application/json", strings.NewReader(string(payloadBytes)))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "User created", response["message"])
		
		user, ok := response["user"].(map[string]interface{})
		require.True(t, ok, "user should be an object")
		assert.Equal(t, "user-123", user["id"])
		assert.Equal(t, "John Doe", user["name"])
		assert.Equal(t, "john@example.com", user["email"])
	})

	t.Run("TestGetUser", func(t *testing.T) {
		resp, err := http.Get(apiUrl + "/users/123")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "User retrieved", response["message"])
		
		user, ok := response["user"].(map[string]interface{})
		require.True(t, ok, "user should be an object")
		assert.Equal(t, "123", user["id"])
		assert.Equal(t, "Test User", user["name"])
		assert.NotEmpty(t, user["createdAt"])
	})

	t.Run("TestUpdateUser", func(t *testing.T) {
		updatePayload := map[string]string{
			"name":  "Jane Doe",
			"email": "jane@example.com",
		}
		payloadBytes, err := json.Marshal(updatePayload)
		require.NoError(t, err)

		req, err := http.NewRequest("PUT", apiUrl+"/users/123", strings.NewReader(string(payloadBytes)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "User updated", response["message"])
		
		user, ok := response["user"].(map[string]interface{})
		require.True(t, ok, "user should be an object")
		assert.Equal(t, "123", user["id"])
		assert.Equal(t, "Jane Doe", user["name"])
		assert.Equal(t, "jane@example.com", user["email"])
		assert.NotEmpty(t, user["updatedAt"])
	})

	t.Run("TestDeleteUser", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", apiUrl+"/users/123", nil)
		require.NoError(t, err)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "User deleted", response["message"])
		assert.Equal(t, "123", response["userId"])
	})

	t.Run("TestCORSHeaders", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", apiUrl, nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "content-type,authorization")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// CORS preflight should return 200 or 204
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent)
		
		// Check CORS headers
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "content-type")
	})

	// Test API Gateway configuration
	t.Run("ValidateAPIConfiguration", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		apiGatewayV2Client := apigatewayv2.NewFromConfig(cfg)

		// Get API details
		apiDetails, err := apiGatewayV2Client.GetApi(ctx, &apigatewayv2.GetApiInput{
			ApiId: aws.String(apiId),
		})
		require.NoError(t, err)

		// Validate API configuration
		assert.Equal(t, "HTTP", string(apiDetails.ProtocolType))
		assert.NotNil(t, apiDetails.ApiEndpoint)
		assert.True(t, strings.HasPrefix(*apiDetails.ApiEndpoint, "https://"))

		// Get routes
		routes, err := apiGatewayV2Client.GetRoutes(ctx, &apigatewayv2.GetRoutesInput{
			ApiId: aws.String(apiId),
		})
		require.NoError(t, err)

		// Validate routes exist
		assert.GreaterOrEqual(t, len(routes.Items), 5, "Should have at least 5 routes")

		routeKeys := make([]string, len(routes.Items))
		for i, route := range routes.Items {
			routeKeys[i] = *route.RouteKey
		}

		// Check expected routes
		expectedRoutes := []string{"GET /", "POST /users", "GET /users/{id}", "PUT /users/{id}", "DELETE /users/{id}"}
		for _, expectedRoute := range expectedRoutes {
			assert.Contains(t, routeKeys, expectedRoute, "Route %s should exist", expectedRoute)
		}

		// Note: CORS configuration is typically handled at the integration level
		// For now, we'll validate that the API has the expected structure
		t.Logf("API has %d routes configured", len(routes.Items))
	})
}

func TestAPIGatewayV1Deployment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_AWS_ACCOUNT_ID") == "" {
		t.Skip("Skipping AWS integration test - SST_TEST_AWS_ACCOUNT_ID not set")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	ctx := context.Background()
	testStage := fmt.Sprintf("api-v1-%d", time.Now().Unix())
	
	// Create test project with API Gateway v1 (REST API)
	projectDir, err := helpers.CreateTestProject("aws-api-v1", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-api-v1",
      region: "us-east-1",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const api = new sst.aws.ApiGatewayV1("TestAPIv1", {
        routes: {
          "GET /": "index.handler",
          "GET /health": "health.handler",
          "POST /webhook": "webhook.handler",
        },
      });

      return {
        apiUrl: api.url,
        apiId: api.id,
        apiName: api.name,
      };
    });
  },
};`,
		"index.js": `
export const handler = async (event, context) => {
  console.log("API v1 root handler:", JSON.stringify(event, null, 2));
  
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      message: "Hello from API Gateway v1!",
      path: event.path,
      httpMethod: event.httpMethod,
      requestId: context.awsRequestId,
      timestamp: new Date().toISOString(),
    }),
  };
};`,
		"health.js": `
export const handler = async (event, context) => {
  console.log("Health check:", JSON.stringify(event, null, 2));
  
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      status: "healthy",
      timestamp: new Date().toISOString(),
      requestId: context.awsRequestId,
      version: "1.0.0",
    }),
  };
};`,
		"webhook.js": `
export const handler = async (event, context) => {
  console.log("Webhook handler:", JSON.stringify(event, null, 2));
  
  const body = event.body ? JSON.parse(event.body) : {};
  
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      message: "Webhook received",
      received: body,
      requestId: context.awsRequestId,
      timestamp: new Date().toISOString(),
    }),
  };
};`,
		"package.json": `{
  "name": "test-api-v1",
  "version": "1.0.0",
  "type": "module",
  "dependencies": {}
}`,
	})
	require.NoError(t, err)
	defer helpers.CleanupTestProject(projectDir)

	// Deploy the project
	err = helpers.DeployProject(ctx, projectDir, testStage)
	require.NoError(t, err)
	defer helpers.RemoveProject(ctx, projectDir, testStage)

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, testStage)
	require.NoError(t, err)

	apiUrl, ok := outputs["apiUrl"].(string)
	require.True(t, ok, "apiUrl output should be a string")
	require.NotEmpty(t, apiUrl, "apiUrl should not be empty")

	apiId, ok := outputs["apiId"].(string)
	require.True(t, ok, "apiId output should be a string")
	require.NotEmpty(t, apiId, "apiId should not be empty")

	// Test HTTP endpoints
	t.Run("TestV1RootEndpoint", func(t *testing.T) {
		resp, err := http.Get(apiUrl)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "Hello from API Gateway v1!", response["message"])
		assert.Equal(t, "/", response["path"])
		assert.Equal(t, "GET", response["httpMethod"])
		assert.NotEmpty(t, response["requestId"])
	})

	t.Run("TestHealthEndpoint", func(t *testing.T) {
		resp, err := http.Get(apiUrl + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, "1.0.0", response["version"])
		assert.NotEmpty(t, response["requestId"])
	})

	t.Run("TestWebhookEndpoint", func(t *testing.T) {
		webhookPayload := map[string]interface{}{
			"event": "user.created",
			"data": map[string]string{
				"userId": "123",
				"email":  "test@example.com",
			},
		}
		payloadBytes, err := json.Marshal(webhookPayload)
		require.NoError(t, err)

		resp, err := http.Post(apiUrl+"/webhook", "application/json", strings.NewReader(string(payloadBytes)))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "Webhook received", response["message"])
		assert.NotNil(t, response["received"])
		assert.NotEmpty(t, response["requestId"])
	})

	// Test API Gateway v1 configuration
	t.Run("ValidateAPIv1Configuration", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		apiGatewayClient := apigateway.NewFromConfig(cfg)

		// Get API details
		apiDetails, err := apiGatewayClient.GetRestApi(ctx, &apigateway.GetRestApiInput{
			RestApiId: aws.String(apiId),
		})
		require.NoError(t, err)

		// Validate API configuration
		assert.NotNil(t, apiDetails.Name)
		assert.NotEmpty(t, *apiDetails.Name)

		// Get resources
		resources, err := apiGatewayClient.GetResources(ctx, &apigateway.GetResourcesInput{
			RestApiId: aws.String(apiId),
		})
		require.NoError(t, err)

		// Should have at least root resource plus our defined resources
		assert.GreaterOrEqual(t, len(resources.Items), 3, "Should have at least 3 resources")

		// Check for expected paths
		paths := make([]string, 0)
		for _, resource := range resources.Items {
			if resource.PathPart != nil {
				paths = append(paths, *resource.PathPart)
			}
		}

		// Should contain our defined paths
		assert.Contains(t, paths, "health")
		assert.Contains(t, paths, "webhook")
	})
}

func TestAPIWithAuthentication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_AWS_ACCOUNT_ID") == "" {
		t.Skip("Skipping AWS integration test - SST_TEST_AWS_ACCOUNT_ID not set")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	ctx := context.Background()
	testStage := fmt.Sprintf("api-auth-%d", time.Now().Unix())
	
	// Create test project with API Gateway and authentication
	projectDir, err := helpers.CreateTestProject("aws-api-auth", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-api-auth",
      region: "us-east-1",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const api = new sst.aws.ApiGatewayV2("TestAPIAuth", {
        routes: {
          "GET /public": "public.handler",
          "GET /protected": {
            function: "protected.handler",
            auth: {
              jwt: {
                issuer: "https://example.com",
                audience: ["api"],
              },
            },
          },
        },
      });

      return {
        apiUrl: api.url,
        apiId: api.id,
      };
    });
  },
};`,
		"public.js": `
export const handler = async (event) => {
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      message: "This is a public endpoint",
      timestamp: new Date().toISOString(),
    }),
  };
};`,
		"protected.js": `
export const handler = async (event) => {
  console.log("Protected endpoint called:", JSON.stringify(event, null, 2));
  
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      message: "This is a protected endpoint",
      user: event.requestContext?.authorizer?.jwt?.claims || null,
      timestamp: new Date().toISOString(),
    }),
  };
};`,
		"package.json": `{
  "name": "test-api-auth",
  "version": "1.0.0",
  "type": "module",
  "dependencies": {}
}`,
	})
	require.NoError(t, err)
	defer helpers.CleanupTestProject(projectDir)

	// Deploy the project
	err = helpers.DeployProject(ctx, projectDir, testStage)
	require.NoError(t, err)
	defer helpers.RemoveProject(ctx, projectDir, testStage)

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, testStage)
	require.NoError(t, err)

	apiUrl, ok := outputs["apiUrl"].(string)
	require.True(t, ok, "apiUrl output should be a string")
	require.NotEmpty(t, apiUrl, "apiUrl should not be empty")

	// Test public endpoint (should work without authentication)
	t.Run("TestPublicEndpoint", func(t *testing.T) {
		resp, err := http.Get(apiUrl + "/public")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.Equal(t, "This is a public endpoint", response["message"])
	})

	// Test protected endpoint (should fail without authentication)
	t.Run("TestProtectedEndpointWithoutAuth", func(t *testing.T) {
		resp, err := http.Get(apiUrl + "/protected")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test protected endpoint with invalid token (should fail)
	t.Run("TestProtectedEndpointWithInvalidAuth", func(t *testing.T) {
		req, err := http.NewRequest("GET", apiUrl+"/protected", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid-token")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 401 or 403
		assert.True(t, resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden)
	})
}