package aws

import (
	"fmt"
	"testing"

	"github.com/sst/sst/v3/test/integration/pulumi/helpers"
	"github.com/stretchr/testify/require"
)

// TestFullStackDeployment tests deploying a complete SST application with API + Frontend + Database
func TestFullStackDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	helpers.RunIntegrationTest(t, "full-stack", func(t *testing.T, config *helpers.PulumiIntegrationTestConfig, projectDir string) {
		// Create a comprehensive SST config with multiple components
		sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "test-full-stack",
      removal: input?.stage === "pulumi-integration-test" ? "remove" : "retain",
      home: "aws",
    };
  },
  async run() {
    // Database layer
    const table = new sst.aws.Dynamo("UserTable", {
      fields: {
        userId: "string",
        email: "string",
      },
      primaryIndex: { hashKey: "userId" },
      globalIndexes: {
        EmailIndex: { hashKey: "email" },
      },
    });

    // Storage layer
    const bucket = new sst.aws.Bucket("AssetsBucket", {
      public: true,
      cors: {
        allowCredentials: true,
        allowHeaders: ["*"],
        allowMethods: ["GET", "POST", "PUT", "DELETE"],
        allowOrigins: ["*"],
      },
    });

    // API layer
    const api = new sst.aws.ApiGatewayV2("Api", {
      cors: {
        allowCredentials: true,
        allowHeaders: ["*"],
        allowMethods: ["*"],
        allowOrigins: ["*"],
      },
    });

    // Lambda functions
    const getUserFunction = new sst.aws.Function("GetUser", {
      handler: "functions/getUser.handler",
      runtime: "nodejs20.x",
      environment: {
        TABLE_NAME: table.name,
      },
      link: [table],
    });

    const createUserFunction = new sst.aws.Function("CreateUser", {
      handler: "functions/createUser.handler", 
      runtime: "nodejs20.x",
      environment: {
        TABLE_NAME: table.name,
        BUCKET_NAME: bucket.name,
      },
      link: [table, bucket],
    });

    const uploadFunction = new sst.aws.Function("Upload", {
      handler: "functions/upload.handler",
      runtime: "nodejs20.x", 
      environment: {
        BUCKET_NAME: bucket.name,
      },
      link: [bucket],
    });

    // API routes
    api.route("GET /users/{id}", getUserFunction.arn);
    api.route("POST /users", createUserFunction.arn);
    api.route("POST /upload", uploadFunction.arn);
    api.route("GET /health", {
      handler: "functions/health.handler",
      runtime: "nodejs20.x",
    });

    // Frontend (Static Site)
    const frontend = new sst.aws.StaticSite("Frontend", {
      build: {
        command: "npm run build",
        output: "dist",
      },
      environment: {
        VITE_API_URL: api.url,
        VITE_BUCKET_NAME: bucket.name,
      },
    });

    return {
      api: api.url,
      frontend: frontend.url,
      bucket: bucket.name,
      table: table.name,
    };
  },
});`

		// Write SST config
		err := helpers.WriteFile(projectDir, "sst.config.ts", sstConfig)
		require.NoError(t, err, "Failed to write SST config")

		// Create package.json for the project
		packageJson := `{
  "name": "test-full-stack",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "build": "echo 'Building frontend...' && mkdir -p dist && echo '<html><body><h1>Test App</h1></body></html>' > dist/index.html",
    "dev": "sst dev",
    "deploy": "sst deploy"
  },
  "dependencies": {
    "sst": "latest"
  }
}`

		err = helpers.WriteFile(projectDir, "package.json", packageJson)
		require.NoError(t, err, "Failed to write package.json")

		// Create Lambda function handlers
		functionsDir := fmt.Sprintf("%s/functions", projectDir)
		err = helpers.CreateDirectory(functionsDir)
		require.NoError(t, err, "Failed to create functions directory")

		// Health check handler
		healthHandler := `export const handler = async (event) => {
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
    },
    body: JSON.stringify({
      status: "healthy",
      timestamp: new Date().toISOString(),
      stage: process.env.SST_STAGE,
    }),
  };
};`

		err = helpers.WriteFile(projectDir, "functions/health.js", healthHandler)
		require.NoError(t, err, "Failed to write health handler")

		// Get user handler
		getUserHandler := `import { DynamoDBClient } from "@aws-sdk/client-dynamodb";
import { DynamoDBDocumentClient, GetCommand } from "@aws-sdk/lib-dynamodb";

const client = new DynamoDBClient({});
const docClient = DynamoDBDocumentClient.from(client);

export const handler = async (event) => {
  try {
    const userId = event.pathParameters?.id;
    
    if (!userId) {
      return {
        statusCode: 400,
        headers: {
          "Content-Type": "application/json",
          "Access-Control-Allow-Origin": "*",
        },
        body: JSON.stringify({ error: "User ID is required" }),
      };
    }

    const command = new GetCommand({
      TableName: process.env.TABLE_NAME,
      Key: { userId },
    });

    const result = await docClient.send(command);
    
    if (!result.Item) {
      return {
        statusCode: 404,
        headers: {
          "Content-Type": "application/json", 
          "Access-Control-Allow-Origin": "*",
        },
        body: JSON.stringify({ error: "User not found" }),
      };
    }

    return {
      statusCode: 200,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
      },
      body: JSON.stringify(result.Item),
    };
  } catch (error) {
    console.error("Error getting user:", error);
    return {
      statusCode: 500,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
      },
      body: JSON.stringify({ error: "Internal server error" }),
    };
  }
};`

		err = helpers.WriteFile(projectDir, "functions/getUser.js", getUserHandler)
		require.NoError(t, err, "Failed to write getUser handler")

		// Create user handler
		createUserHandler := `import { DynamoDBClient } from "@aws-sdk/client-dynamodb";
import { DynamoDBDocumentClient, PutCommand } from "@aws-sdk/lib-dynamodb";
import { randomUUID } from "crypto";

const client = new DynamoDBClient({});
const docClient = DynamoDBDocumentClient.from(client);

export const handler = async (event) => {
  try {
    const body = JSON.parse(event.body || "{}");
    const { email, name } = body;
    
    if (!email || !name) {
      return {
        statusCode: 400,
        headers: {
          "Content-Type": "application/json",
          "Access-Control-Allow-Origin": "*",
        },
        body: JSON.stringify({ error: "Email and name are required" }),
      };
    }

    const userId = randomUUID();
    const user = {
      userId,
      email,
      name,
      createdAt: new Date().toISOString(),
    };

    const command = new PutCommand({
      TableName: process.env.TABLE_NAME,
      Item: user,
    });

    await docClient.send(command);

    return {
      statusCode: 201,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
      },
      body: JSON.stringify(user),
    };
  } catch (error) {
    console.error("Error creating user:", error);
    return {
      statusCode: 500,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
      },
      body: JSON.stringify({ error: "Internal server error" }),
    };
  }
};`

		err = helpers.WriteFile(projectDir, "functions/createUser.js", createUserHandler)
		require.NoError(t, err, "Failed to write createUser handler")

		// Upload handler
		uploadHandler := `import { S3Client, PutObjectCommand } from "@aws-sdk/client-s3";
import { getSignedUrl } from "@aws-sdk/s3-request-presigner";

const s3Client = new S3Client({});

export const handler = async (event) => {
  try {
    const body = JSON.parse(event.body || "{}");
    const { fileName, contentType } = body;
    
    if (!fileName) {
      return {
        statusCode: 400,
        headers: {
          "Content-Type": "application/json",
          "Access-Control-Allow-Origin": "*",
        },
        body: JSON.stringify({ error: "File name is required" }),
      };
    }

    const key = ` + "`uploads/${Date.now()}-${fileName}`" + `;
    
    const command = new PutObjectCommand({
      Bucket: process.env.BUCKET_NAME,
      Key: key,
      ContentType: contentType || "application/octet-stream",
    });

    const signedUrl = await getSignedUrl(s3Client, command, { expiresIn: 3600 });

    return {
      statusCode: 200,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
      },
      body: JSON.stringify({
        uploadUrl: signedUrl,
        key,
      }),
    };
  } catch (error) {
    console.error("Error generating upload URL:", error);
    return {
      statusCode: 500,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
      },
      body: JSON.stringify({ error: "Internal server error" }),
    };
  }
};`

		err = helpers.WriteFile(projectDir, "functions/upload.js", uploadHandler)
		require.NoError(t, err, "Failed to write upload handler")

		// Deploy the project
		result, err := helpers.DeployWithPolicies(t, projectDir, config.TestStage, config.PolicyPackPath)
		require.NoError(t, err, "Failed to deploy full-stack project")
		require.NotNil(t, result, "Deploy result should not be nil")

		// Validate deployment with comprehensive checks
		validators := []helpers.ResourceValidator{
			helpers.ValidateBucketExists("AssetsBucket"),
			helpers.ValidateDynamoTableExists("UserTable"),
			helpers.ValidateApiGatewayExists("Api"),
			helpers.ValidateFunctionExists("GetUser"),
			helpers.ValidateFunctionExists("CreateUser"),
			helpers.ValidateFunctionExists("Upload"),
			helpers.ValidateStaticSiteExists("Frontend"),
		}

		err = helpers.ValidateDeployment(t, "test-"+config.TestStage, validators, projectDir)
		require.NoError(t, err, "Failed to validate full-stack deployment")

		t.Logf("Full-stack deployment test completed successfully")
	})
}

// TestEndToEndFunctionality tests the deployed application's end-to-end functionality
func TestEndToEndFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	helpers.RunIntegrationTest(t, "e2e-functionality", func(t *testing.T, config *helpers.PulumiIntegrationTestConfig, projectDir string) {
		// First deploy the full stack (reuse the config from above)
		sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "test-e2e-functionality",
      removal: input?.stage === "pulumi-integration-test" ? "remove" : "retain",
      home: "aws",
    };
  },
  async run() {
    const table = new sst.aws.Dynamo("UserTable", {
      fields: {
        userId: "string",
        email: "string",
      },
      primaryIndex: { hashKey: "userId" },
    });

    const bucket = new sst.aws.Bucket("AssetsBucket", {
      public: true,
    });

    const api = new sst.aws.ApiGatewayV2("Api");

    api.route("GET /health", {
      handler: "functions/health.handler",
      runtime: "nodejs20.x",
    });

    api.route("POST /users", {
      handler: "functions/createUser.handler",
      runtime: "nodejs20.x",
      environment: {
        TABLE_NAME: table.name,
      },
      link: [table],
    });

    api.route("GET /users/{id}", {
      handler: "functions/getUser.handler",
      runtime: "nodejs20.x",
      environment: {
        TABLE_NAME: table.name,
      },
      link: [table],
    });

    return {
      api: api.url,
      bucket: bucket.name,
      table: table.name,
    };
  },
});`

		// Write configuration and handlers (simplified for E2E test)
		err := helpers.WriteFile(projectDir, "sst.config.ts", sstConfig)
		require.NoError(t, err, "Failed to write SST config")

		// Create minimal package.json
		packageJson := `{
  "name": "test-e2e-functionality",
  "version": "1.0.0",
  "type": "module",
  "dependencies": {
    "sst": "latest"
  }
}`

		err = helpers.WriteFile(projectDir, "package.json", packageJson)
		require.NoError(t, err, "Failed to write package.json")

		// Create functions directory and handlers
		functionsDir := fmt.Sprintf("%s/functions", projectDir)
		err = helpers.CreateDirectory(functionsDir)
		require.NoError(t, err, "Failed to create functions directory")

		// Simple health handler for E2E testing
		healthHandler := `export const handler = async (event) => {
  return {
    statusCode: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ status: "healthy", timestamp: new Date().toISOString() }),
  };
};`

		err = helpers.WriteFile(projectDir, "functions/health.js", healthHandler)
		require.NoError(t, err, "Failed to write health handler")

		// Deploy the project
		result, err := helpers.DeployWithPolicies(t, projectDir, config.TestStage, config.PolicyPackPath)
		require.NoError(t, err, "Failed to deploy E2E test project")
		require.NotNil(t, result, "Deploy result should not be nil")

		// Test end-to-end functionality
		err = helpers.TestEndToEndFunctionality(t, result, config)
		require.NoError(t, err, "End-to-end functionality test failed")

		t.Logf("End-to-end functionality test completed successfully")
	})
}

// TestServiceToServiceCommunication tests communication between different services
func TestServiceToServiceCommunication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	helpers.RunIntegrationTest(t, "service-communication", func(t *testing.T, config *helpers.PulumiIntegrationTestConfig, projectDir string) {
		// Create SST config with multiple interconnected services
		sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "test-service-communication",
      removal: input?.stage === "pulumi-integration-test" ? "remove" : "retain",
      home: "aws",
    };
  },
  async run() {
    // Shared resources
    const table = new sst.aws.Dynamo("SharedTable", {
      fields: {
        id: "string",
        type: "string",
      },
      primaryIndex: { hashKey: "id", rangeKey: "type" },
    });

    const queue = new sst.aws.Queue("ProcessingQueue");

    // Service A - Producer
    const producerFunction = new sst.aws.Function("Producer", {
      handler: "functions/producer.handler",
      runtime: "nodejs20.x",
      environment: {
        QUEUE_URL: queue.url,
        TABLE_NAME: table.name,
      },
      link: [queue, table],
    });

    // Service B - Consumer
    const consumerFunction = new sst.aws.Function("Consumer", {
      handler: "functions/consumer.handler",
      runtime: "nodejs20.x",
      environment: {
        TABLE_NAME: table.name,
      },
      link: [table],
    });

    // Connect queue to consumer
    queue.subscribe(consumerFunction.arn);

    // API to trigger the workflow
    const api = new sst.aws.ApiGatewayV2("WorkflowApi");
    api.route("POST /trigger", producerFunction.arn);

    return {
      api: api.url,
      queue: queue.url,
      table: table.name,
    };
  },
});`

		err := helpers.WriteFile(projectDir, "sst.config.ts", sstConfig)
		require.NoError(t, err, "Failed to write SST config")

		// Create package.json
		packageJson := `{
  "name": "test-service-communication",
  "version": "1.0.0",
  "type": "module",
  "dependencies": {
    "sst": "latest"
  }
}`

		err = helpers.WriteFile(projectDir, "package.json", packageJson)
		require.NoError(t, err, "Failed to write package.json")

		// Create functions directory
		functionsDir := fmt.Sprintf("%s/functions", projectDir)
		err = helpers.CreateDirectory(functionsDir)
		require.NoError(t, err, "Failed to create functions directory")

		// Producer function (sends messages to queue)
		producerHandler := `import { SQSClient, SendMessageCommand } from "@aws-sdk/client-sqs";
import { DynamoDBClient } from "@aws-sdk/client-dynamodb";
import { DynamoDBDocumentClient, PutCommand } from "@aws-sdk/lib-dynamodb";
import { randomUUID } from "crypto";

const sqsClient = new SQSClient({});
const dynamoClient = new DynamoDBClient({});
const docClient = DynamoDBDocumentClient.from(dynamoClient);

export const handler = async (event) => {
  try {
    const body = JSON.parse(event.body || "{}");
    const { message } = body;
    
    const id = randomUUID();
    const timestamp = new Date().toISOString();
    
    // Store in DynamoDB
    await docClient.send(new PutCommand({
      TableName: process.env.TABLE_NAME,
      Item: {
        id,
        type: "produced",
        message,
        timestamp,
        status: "pending",
      },
    }));
    
    // Send to queue
    await sqsClient.send(new SendMessageCommand({
      QueueUrl: process.env.QUEUE_URL,
      MessageBody: JSON.stringify({ id, message, timestamp }),
    }));
    
    return {
      statusCode: 200,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id, status: "queued" }),
    };
  } catch (error) {
    console.error("Producer error:", error);
    return {
      statusCode: 500,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ error: "Internal server error" }),
    };
  }
};`

		err = helpers.WriteFile(projectDir, "functions/producer.js", producerHandler)
		require.NoError(t, err, "Failed to write producer handler")

		// Consumer function (processes messages from queue)
		consumerHandler := `import { DynamoDBClient } from "@aws-sdk/client-dynamodb";
import { DynamoDBDocumentClient, UpdateCommand } from "@aws-sdk/lib-dynamodb";

const dynamoClient = new DynamoDBClient({});
const docClient = DynamoDBDocumentClient.from(dynamoClient);

export const handler = async (event) => {
  try {
    for (const record of event.Records) {
      const { id, message, timestamp } = JSON.parse(record.body);
      
      // Process the message (simulate some work)
      const processedMessage = message.toUpperCase();
      
      // Update DynamoDB record
      await docClient.send(new UpdateCommand({
        TableName: process.env.TABLE_NAME,
        Key: { id, type: "produced" },
        UpdateExpression: "SET #status = :status, processedMessage = :processedMessage, processedAt = :processedAt",
        ExpressionAttributeNames: {
          "#status": "status",
        },
        ExpressionAttributeValues: {
          ":status": "processed",
          ":processedMessage": processedMessage,
          ":processedAt": new Date().toISOString(),
        },
      }));
      
      console.log(` + "`Processed message ${id}: ${processedMessage}`" + `);
    }
    
    return { statusCode: 200 };
  } catch (error) {
    console.error("Consumer error:", error);
    throw error;
  }
};`

		err = helpers.WriteFile(projectDir, "functions/consumer.js", consumerHandler)
		require.NoError(t, err, "Failed to write consumer handler")

		// Deploy the project
		result, err := helpers.DeployWithPolicies(t, projectDir, config.TestStage, config.PolicyPackPath)
		require.NoError(t, err, "Failed to deploy service communication test project")
		require.NotNil(t, result, "Deploy result should not be nil")

		// Validate deployment
		validators := []helpers.ResourceValidator{
			helpers.ValidateDynamoTableExists("SharedTable"),
			helpers.ValidateQueueExists("ProcessingQueue"),
			helpers.ValidateFunctionExists("Producer"),
			helpers.ValidateFunctionExists("Consumer"),
			helpers.ValidateApiGatewayExists("WorkflowApi"),
		}

		err = helpers.ValidateDeployment(t, "test-"+config.TestStage, validators, projectDir)
		require.NoError(t, err, "Failed to validate service communication deployment")

		// Test service-to-service communication
		err = helpers.TestServiceCommunication(t, result, config)
		require.NoError(t, err, "Service-to-service communication test failed")

		t.Logf("Service-to-service communication test completed successfully")
	})
}

// TestCompleteApplicationLifecycle tests the complete application lifecycle
func TestCompleteApplicationLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	helpers.RunIntegrationTest(t, "app-lifecycle", func(t *testing.T, config *helpers.PulumiIntegrationTestConfig, projectDir string) {
		// Test complete lifecycle: deploy -> update -> rollback -> remove
		
		// Phase 1: Initial deployment
		t.Log("Phase 1: Initial deployment")
		initialConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "test-app-lifecycle",
      removal: input?.stage === "pulumi-integration-test" ? "remove" : "retain",
      home: "aws",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("TestBucket", {
      public: false,
    });

    const api = new sst.aws.ApiGatewayV2("TestApi");
    
    api.route("GET /v1/health", {
      handler: "functions/health.handler",
      runtime: "nodejs20.x",
      environment: {
        VERSION: "1.0.0",
      },
    });

    return {
      api: api.url,
      bucket: bucket.name,
      version: "1.0.0",
    };
  },
});`

		err := helpers.WriteFile(projectDir, "sst.config.ts", initialConfig)
		require.NoError(t, err, "Failed to write initial SST config")

		// Create package.json
		packageJson := `{
  "name": "test-app-lifecycle",
  "version": "1.0.0",
  "type": "module",
  "dependencies": {
    "sst": "latest"
  }
}`

		err = helpers.WriteFile(projectDir, "package.json", packageJson)
		require.NoError(t, err, "Failed to write package.json")

		// Create functions directory and initial handler
		functionsDir := fmt.Sprintf("%s/functions", projectDir)
		err = helpers.CreateDirectory(functionsDir)
		require.NoError(t, err, "Failed to create functions directory")

		healthHandler := `export const handler = async (event) => {
  return {
    statusCode: 200,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ 
      status: "healthy", 
      version: process.env.VERSION || "unknown",
      timestamp: new Date().toISOString() 
    }),
  };
};`

		err = helpers.WriteFile(projectDir, "functions/health.js", healthHandler)
		require.NoError(t, err, "Failed to write health handler")

		// Deploy initial version
		result1, err := helpers.DeployWithPolicies(t, projectDir, config.TestStage, config.PolicyPackPath)
		require.NoError(t, err, "Failed to deploy initial version")
		require.NotNil(t, result1, "Initial deploy result should not be nil")

		// Phase 2: Update deployment
		t.Log("Phase 2: Update deployment")
		updatedConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "test-app-lifecycle",
      removal: input?.stage === "pulumi-integration-test" ? "remove" : "retain",
      home: "aws",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("TestBucket", {
      public: true, // Changed from false to true
    });

    const table = new sst.aws.Dynamo("NewTable", { // Added new resource
      fields: {
        id: "string",
      },
      primaryIndex: { hashKey: "id" },
    });

    const api = new sst.aws.ApiGatewayV2("TestApi");
    
    api.route("GET /v1/health", {
      handler: "functions/health.handler",
      runtime: "nodejs20.x",
      environment: {
        VERSION: "2.0.0", // Updated version
        TABLE_NAME: table.name,
      },
      link: [table],
    });

    api.route("GET /v2/health", { // Added new route
      handler: "functions/health.handler",
      runtime: "nodejs20.x",
      environment: {
        VERSION: "2.0.0",
      },
    });

    return {
      api: api.url,
      bucket: bucket.name,
      table: table.name,
      version: "2.0.0",
    };
  },
});`

		err = helpers.WriteFile(projectDir, "sst.config.ts", updatedConfig)
		require.NoError(t, err, "Failed to write updated SST config")

		// Deploy updated version
		result2, err := helpers.DeployWithPolicies(t, projectDir, config.TestStage, config.PolicyPackPath)
		require.NoError(t, err, "Failed to deploy updated version")
		require.NotNil(t, result2, "Updated deploy result should not be nil")

		// Phase 3: Validate update
		t.Log("Phase 3: Validate update")
		validators := []helpers.ResourceValidator{
			helpers.ValidateBucketExists("TestBucket"),
			helpers.ValidateDynamoTableExists("NewTable"),
			helpers.ValidateApiGatewayExists("TestApi"),
			helpers.ValidateFunctionExists("health"),
		}

		err = helpers.ValidateDeployment(t, "test-"+config.TestStage, validators, projectDir)
		require.NoError(t, err, "Failed to validate updated deployment")

		// Phase 4: Test application lifecycle completion
		t.Log("Phase 4: Application lifecycle test completed")
		err = helpers.TestApplicationLifecycle(t, result2, config)
		require.NoError(t, err, "Application lifecycle test failed")

		t.Logf("Complete application lifecycle test completed successfully")
	})
}