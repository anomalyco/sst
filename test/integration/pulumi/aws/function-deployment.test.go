package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sst/sst/v3/test/integration/pulumi/helpers"
)

func TestFunctionDeploymentNodeJS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_AWS_ACCOUNT_ID") == "" {
		t.Skip("Skipping AWS integration test - SST_TEST_AWS_ACCOUNT_ID not set")
	}

	ctx := context.Background()
	testStage := fmt.Sprintf("func-nodejs-%d", time.Now().Unix())
	
	// Create test project
	projectDir, err := helpers.CreateTestProject("aws-function-nodejs", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-function-nodejs",
      region: "us-east-1",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const fn = new sst.aws.Function("TestFunction", {
        handler: "index.handler",
        runtime: "nodejs20.x",
        environment: {
          TEST_VAR: "test-value",
          STAGE: stack.stage,
        },
      });

      return {
        functionName: fn.name,
        functionArn: fn.arn,
      };
    });
  },
};`,
		"index.js": `
export const handler = async (event) => {
  console.log("Event:", JSON.stringify(event, null, 2));
  console.log("Environment:", process.env.TEST_VAR);
  console.log("Stage:", process.env.STAGE);
  
  return {
    statusCode: 200,
    body: JSON.stringify({
      message: "Hello from Node.js Lambda!",
      environment: process.env.TEST_VAR,
      stage: process.env.STAGE,
      timestamp: new Date().toISOString(),
    }),
  };
};`,
		"package.json": `{
  "name": "test-function-nodejs",
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

	functionName, ok := outputs["functionName"].(string)
	require.True(t, ok, "functionName output should be a string")
	require.NotEmpty(t, functionName, "functionName should not be empty")

	functionArn, ok := outputs["functionArn"].(string)
	require.True(t, ok, "functionArn output should be a string")
	require.NotEmpty(t, functionArn, "functionArn should not be empty")

	// Test function invocation
	t.Run("InvokeFunction", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		lambdaClient := lambda.NewFromConfig(cfg)

		// Invoke the function
		payload := map[string]interface{}{
			"test": "data",
			"timestamp": time.Now().Unix(),
		}
		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		result, err := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
			FunctionName: aws.String(functionName),
			Payload:      payloadBytes,
		})
		require.NoError(t, err)
		assert.Nil(t, result.FunctionError, "Function should not have errors")

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal(result.Payload, &response)
		require.NoError(t, err)

		assert.Equal(t, float64(200), response["statusCode"])
		
		bodyStr, ok := response["body"].(string)
		require.True(t, ok, "body should be a string")
		
		var body map[string]interface{}
		err = json.Unmarshal([]byte(bodyStr), &body)
		require.NoError(t, err)

		assert.Equal(t, "Hello from Node.js Lambda!", body["message"])
		assert.Equal(t, "test-value", body["environment"])
		assert.Equal(t, testStage, body["stage"])
		assert.NotEmpty(t, body["timestamp"])
	})

	// Test function configuration
	t.Run("ValidateFunctionConfiguration", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		lambdaClient := lambda.NewFromConfig(cfg)

		// Get function configuration
		funcConfig, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)

		// Validate runtime
		assert.Equal(t, types.RuntimeNodejs20x, funcConfig.Configuration.Runtime)

		// Validate environment variables
		env := funcConfig.Configuration.Environment
		require.NotNil(t, env)
		require.NotNil(t, env.Variables)
		
		assert.Equal(t, "test-value", env.Variables["TEST_VAR"])
		assert.Equal(t, testStage, env.Variables["STAGE"])

		// Validate handler
		assert.Equal(t, "index.handler", *funcConfig.Configuration.Handler)

		// Validate timeout (should be default)
		assert.Equal(t, int32(3), *funcConfig.Configuration.Timeout)
	})
}

func TestFunctionDeploymentPython(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_AWS_ACCOUNT_ID") == "" {
		t.Skip("Skipping AWS integration test - SST_TEST_AWS_ACCOUNT_ID not set")
	}

	ctx := context.Background()
	testStage := fmt.Sprintf("func-python-%d", time.Now().Unix())
	
	// Create test project
	projectDir, err := helpers.CreateTestProject("aws-function-python", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-function-python",
      region: "us-east-1",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const fn = new sst.aws.Function("TestFunction", {
        handler: "lambda_function.lambda_handler",
        runtime: "python3.11",
        timeout: "30 seconds",
        memory: "256 MB",
        environment: {
          PYTHON_VAR: "python-test",
          STAGE: stack.stage,
        },
      });

      return {
        functionName: fn.name,
        functionArn: fn.arn,
      };
    });
  },
};`,
		"lambda_function.py": `
import json
import os
from datetime import datetime

def lambda_handler(event, context):
    print(f"Event: {json.dumps(event)}")
    print(f"Environment: {os.environ.get('PYTHON_VAR')}")
    print(f"Stage: {os.environ.get('STAGE')}")
    
    return {
        'statusCode': 200,
        'body': json.dumps({
            'message': 'Hello from Python Lambda!',
            'environment': os.environ.get('PYTHON_VAR'),
            'stage': os.environ.get('STAGE'),
            'timestamp': datetime.now().isoformat(),
            'python_version': f"{context.aws_request_id[:8]}"
        })
    }
`,
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

	functionName, ok := outputs["functionName"].(string)
	require.True(t, ok, "functionName output should be a string")

	// Test function invocation
	t.Run("InvokePythonFunction", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		lambdaClient := lambda.NewFromConfig(cfg)

		// Invoke the function
		payload := map[string]interface{}{
			"test": "python-data",
			"number": 42,
		}
		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		result, err := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
			FunctionName: aws.String(functionName),
			Payload:      payloadBytes,
		})
		require.NoError(t, err)
		assert.Nil(t, result.FunctionError, "Function should not have errors")

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal(result.Payload, &response)
		require.NoError(t, err)

		assert.Equal(t, float64(200), response["statusCode"])
		
		bodyStr, ok := response["body"].(string)
		require.True(t, ok, "body should be a string")
		
		var body map[string]interface{}
		err = json.Unmarshal([]byte(bodyStr), &body)
		require.NoError(t, err)

		assert.Equal(t, "Hello from Python Lambda!", body["message"])
		assert.Equal(t, "python-test", body["environment"])
		assert.Equal(t, testStage, body["stage"])
		assert.NotEmpty(t, body["timestamp"])
	})

	// Test function configuration
	t.Run("ValidatePythonFunctionConfiguration", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		lambdaClient := lambda.NewFromConfig(cfg)

		// Get function configuration
		funcConfig, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)

		// Validate runtime
		assert.Equal(t, types.RuntimePython311, funcConfig.Configuration.Runtime)

		// Validate timeout (30 seconds)
		assert.Equal(t, int32(30), *funcConfig.Configuration.Timeout)

		// Validate memory (256 MB)
		assert.Equal(t, int32(256), *funcConfig.Configuration.MemorySize)

		// Validate environment variables
		env := funcConfig.Configuration.Environment
		require.NotNil(t, env)
		require.NotNil(t, env.Variables)
		
		assert.Equal(t, "python-test", env.Variables["PYTHON_VAR"])
		assert.Equal(t, testStage, env.Variables["STAGE"])

		// Validate handler
		assert.Equal(t, "lambda_function.lambda_handler", *funcConfig.Configuration.Handler)
	})
}

func TestFunctionDeploymentGo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_AWS_ACCOUNT_ID") == "" {
		t.Skip("Skipping AWS integration test - SST_TEST_AWS_ACCOUNT_ID not set")
	}

	ctx := context.Background()
	testStage := fmt.Sprintf("func-go-%d", time.Now().Unix())
	
	// Create test project
	projectDir, err := helpers.CreateTestProject("aws-function-go", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-function-go",
      region: "us-east-1",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const fn = new sst.aws.Function("TestFunction", {
        handler: "bootstrap",
        runtime: "provided.al2023",
        timeout: "15 seconds",
        memory: "512 MB",
        environment: {
          GO_VAR: "go-test",
          STAGE: stack.stage,
        },
      });

      return {
        functionName: fn.name,
        functionArn: fn.arn,
      };
    });
  },
};`,
		"main.go": `
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

type Event struct {
	Test   string ` + "`json:\"test\"`" + `
	Number int    ` + "`json:\"number\"`" + `
}

type Response struct {
	StatusCode int    ` + "`json:\"statusCode\"`" + `
	Body       string ` + "`json:\"body\"`" + `
}

type ResponseBody struct {
	Message     string ` + "`json:\"message\"`" + `
	Environment string ` + "`json:\"environment\"`" + `
	Stage       string ` + "`json:\"stage\"`" + `
	Timestamp   string ` + "`json:\"timestamp\"`" + `
}

func HandleRequest(ctx context.Context, event Event) (Response, error) {
	fmt.Printf("Event: %+v\n", event)
	fmt.Printf("Environment: %s\n", os.Getenv("GO_VAR"))
	fmt.Printf("Stage: %s\n", os.Getenv("STAGE"))

	responseBody := ResponseBody{
		Message:     "Hello from Go Lambda!",
		Environment: os.Getenv("GO_VAR"),
		Stage:       os.Getenv("STAGE"),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	bodyBytes, err := json.Marshal(responseBody)
	if err != nil {
		return Response{}, err
	}

	return Response{
		StatusCode: 200,
		Body:       string(bodyBytes),
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
`,
		"go.mod": `module test-function-go

go 1.21

require github.com/aws/aws-lambda-go v1.41.0
`,
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

	functionName, ok := outputs["functionName"].(string)
	require.True(t, ok, "functionName output should be a string")

	// Test function invocation
	t.Run("InvokeGoFunction", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		lambdaClient := lambda.NewFromConfig(cfg)

		// Invoke the function
		payload := map[string]interface{}{
			"test":   "go-data",
			"number": 123,
		}
		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		result, err := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
			FunctionName: aws.String(functionName),
			Payload:      payloadBytes,
		})
		require.NoError(t, err)
		assert.Nil(t, result.FunctionError, "Function should not have errors")

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal(result.Payload, &response)
		require.NoError(t, err)

		assert.Equal(t, float64(200), response["statusCode"])
		
		bodyStr, ok := response["body"].(string)
		require.True(t, ok, "body should be a string")
		
		var body map[string]interface{}
		err = json.Unmarshal([]byte(bodyStr), &body)
		require.NoError(t, err)

		assert.Equal(t, "Hello from Go Lambda!", body["message"])
		assert.Equal(t, "go-test", body["environment"])
		assert.Equal(t, testStage, body["stage"])
		assert.NotEmpty(t, body["timestamp"])
	})

	// Test function configuration
	t.Run("ValidateGoFunctionConfiguration", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		lambdaClient := lambda.NewFromConfig(cfg)

		// Get function configuration
		funcConfig, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)

		// Validate runtime
		assert.Equal(t, types.RuntimeProvidedal2023, funcConfig.Configuration.Runtime)

		// Validate timeout (15 seconds)
		assert.Equal(t, int32(15), *funcConfig.Configuration.Timeout)

		// Validate memory (512 MB)
		assert.Equal(t, int32(512), *funcConfig.Configuration.MemorySize)

		// Validate environment variables
		env := funcConfig.Configuration.Environment
		require.NotNil(t, env)
		require.NotNil(t, env.Variables)
		
		assert.Equal(t, "go-test", env.Variables["GO_VAR"])
		assert.Equal(t, testStage, env.Variables["STAGE"])

		// Validate handler
		assert.Equal(t, "bootstrap", *funcConfig.Configuration.Handler)
	})
}

func TestFunctionUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("SST_TEST_AWS_ACCOUNT_ID") == "" {
		t.Skip("Skipping AWS integration test - SST_TEST_AWS_ACCOUNT_ID not set")
	}

	ctx := context.Background()
	testStage := fmt.Sprintf("func-update-%d", time.Now().Unix())
	
	// Create initial test project
	projectDir, err := helpers.CreateTestProject("aws-function-update", map[string]string{
		"sst.config.ts": `
export default {
  config() {
    return {
      name: "test-function-update",
      region: "us-east-1",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const fn = new sst.aws.Function("TestFunction", {
        handler: "index.handler",
        runtime: "nodejs20.x",
        environment: {
          VERSION: "v1",
        },
      });

      return {
        functionName: fn.name,
      };
    });
  },
};`,
		"index.js": `
export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({
      message: "Version 1",
      version: process.env.VERSION,
    }),
  };
};`,
		"package.json": `{
  "name": "test-function-update",
  "version": "1.0.0",
  "type": "module"
}`,
	})
	require.NoError(t, err)
	defer helpers.CleanupTestProject(projectDir)

	// Deploy initial version
	err = helpers.DeployProject(ctx, projectDir, testStage)
	require.NoError(t, err)
	defer helpers.RemoveProject(ctx, projectDir, testStage)

	// Get initial outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, testStage)
	require.NoError(t, err)

	functionName, ok := outputs["functionName"].(string)
	require.True(t, ok, "functionName output should be a string")

	// Test initial version
	t.Run("TestInitialVersion", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		lambdaClient := lambda.NewFromConfig(cfg)

		result, err := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
			FunctionName: aws.String(functionName),
			Payload:      []byte("{}"),
		})
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(result.Payload, &response)
		require.NoError(t, err)

		bodyStr := response["body"].(string)
		var body map[string]interface{}
		err = json.Unmarshal([]byte(bodyStr), &body)
		require.NoError(t, err)

		assert.Equal(t, "Version 1", body["message"])
		assert.Equal(t, "v1", body["version"])
	})

	// Update the function code and environment
	err = helpers.UpdateTestProjectFile(projectDir, "index.js", `
export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({
      message: "Version 2 - Updated!",
      version: process.env.VERSION,
      updated: true,
    }),
  };
};`)
	require.NoError(t, err)

	err = helpers.UpdateTestProjectFile(projectDir, "sst.config.ts", `
export default {
  config() {
    return {
      name: "test-function-update",
      region: "us-east-1",
    };
  },
  stacks(app) {
    app.stack(function MyStack({ stack }) {
      const fn = new sst.aws.Function("TestFunction", {
        handler: "index.handler",
        runtime: "nodejs20.x",
        environment: {
          VERSION: "v2",
          UPDATED: "true",
        },
      });

      return {
        functionName: fn.name,
      };
    });
  },
};`)
	require.NoError(t, err)

	// Deploy updated version
	err = helpers.DeployProject(ctx, projectDir, testStage)
	require.NoError(t, err)

	// Test updated version
	t.Run("TestUpdatedVersion", func(t *testing.T) {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
		require.NoError(t, err)

		lambdaClient := lambda.NewFromConfig(cfg)

		// Wait a moment for the update to propagate
		time.Sleep(5 * time.Second)

		result, err := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
			FunctionName: aws.String(functionName),
			Payload:      []byte("{}"),
		})
		require.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(result.Payload, &response)
		require.NoError(t, err)

		bodyStr := response["body"].(string)
		var body map[string]interface{}
		err = json.Unmarshal([]byte(bodyStr), &body)
		require.NoError(t, err)

		assert.Equal(t, "Version 2 - Updated!", body["message"])
		assert.Equal(t, "v2", body["version"])
		assert.Equal(t, true, body["updated"])

		// Verify environment variables were updated
		funcConfig, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)

		env := funcConfig.Configuration.Environment
		require.NotNil(t, env)
		require.NotNil(t, env.Variables)
		
		assert.Equal(t, "v2", env.Variables["VERSION"])
		assert.Equal(t, "true", env.Variables["UPDATED"])
	})
}