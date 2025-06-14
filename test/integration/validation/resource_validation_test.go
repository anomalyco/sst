package validation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceValidation tests that deployed resources match SST configuration
func TestResourceValidation(t *testing.T) {
	// Skip if AWS credentials not configured
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("AWS credentials not configured, skipping resource validation test")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	// Create test project
	projectDir := createTestProject(t, "resource-validation")
	defer cleanupTestProject(t, projectDir)

	// Deploy the project
	stackName := deployTestProject(t, projectDir, "resource-validation-test")
	defer removeTestProject(t, projectDir, "resource-validation-test")

	// Validate Lambda function
	t.Run("Lambda Function Validation", func(t *testing.T) {
		lambdaClient := lambda.NewFromConfig(cfg)
		
		// Find Lambda function by name pattern
		functionName := findLambdaFunction(t, lambdaClient, stackName)
		require.NotEmpty(t, functionName)
		
		// Get function configuration
		resp, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)
		
		config := resp.Configuration
		assert.Equal(t, "nodejs20.x", string(config.Runtime))
		assert.Equal(t, int32(1024), *config.MemorySize)
		assert.Equal(t, int32(10), *config.Timeout)
		
		// Validate environment variables
		assert.NotNil(t, config.Environment)
		assert.NotNil(t, config.Environment.Variables)
		
		// Check for SST-specific environment variables
		envVars := config.Environment.Variables
		assert.Contains(t, envVars, "SST_APP")
		assert.Contains(t, envVars, "SST_STAGE")
		assert.Equal(t, "resource-validation", envVars["SST_APP"])
		assert.Equal(t, "resource-validation-test", envVars["SST_STAGE"])
		
		// Validate function tags
		tagsResp, err := lambdaClient.ListTags(ctx, &lambda.ListTagsInput{
			Resource: config.FunctionArn,
		})
		require.NoError(t, err)
		
		tags := tagsResp.Tags
		assert.Equal(t, "resource-validation", tags["sst:app"])
		assert.Equal(t, "resource-validation-test", tags["sst:stage"])
	})

	// Validate S3 bucket
	t.Run("S3 Bucket Validation", func(t *testing.T) {
		s3Client := s3.NewFromConfig(cfg)
		
		// Find S3 bucket by name pattern
		bucketName := findS3Bucket(t, s3Client, stackName)
		require.NotEmpty(t, bucketName)
		
		// Check bucket exists
		_, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		require.NoError(t, err)
		
		// Validate bucket encryption
		encResp, err := s3Client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
			Bucket: aws.String(bucketName),
		})
		require.NoError(t, err)
		
		rules := encResp.ServerSideEncryptionConfiguration.Rules
		require.Len(t, rules, 1)
		assert.Equal(t, "AES256", string(rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm))
		
		// Validate bucket public access block
		pabResp, err := s3Client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
			Bucket: aws.String(bucketName),
		})
		require.NoError(t, err)
		
		pab := pabResp.PublicAccessBlockConfiguration
		assert.True(t, *pab.BlockPublicAcls)
		assert.True(t, *pab.BlockPublicPolicy)
		assert.True(t, *pab.IgnorePublicAcls)
		assert.True(t, *pab.RestrictPublicBuckets)
		
		// Validate bucket tags
		tagsResp, err := s3Client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
			Bucket: aws.String(bucketName),
		})
		require.NoError(t, err)
		
		tagMap := make(map[string]string)
		for _, tag := range tagsResp.TagSet {
			tagMap[*tag.Key] = *tag.Value
		}
		
		assert.Equal(t, "resource-validation", tagMap["sst:app"])
		assert.Equal(t, "resource-validation-test", tagMap["sst:stage"])
	})
}

// TestResourceProperties tests that resource properties match configuration
func TestResourceProperties(t *testing.T) {
	// Skip if AWS credentials not configured
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("AWS credentials not configured, skipping resource properties test")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	// Create test project with specific configurations
	projectDir := createTestProjectWithConfig(t, "resource-properties")
	defer cleanupTestProject(t, projectDir)

	// Deploy the project
	stackName := deployTestProject(t, projectDir, "resource-properties-test")
	defer removeTestProject(t, projectDir, "resource-properties-test")

	// Test Lambda function properties
	t.Run("Lambda Function Properties", func(t *testing.T) {
		lambdaClient := lambda.NewFromConfig(cfg)
		
		functionName := findLambdaFunction(t, lambdaClient, stackName)
		require.NotEmpty(t, functionName)
		
		resp, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)
		
		config := resp.Configuration
		
		// Test custom memory size
		assert.Equal(t, int32(512), *config.MemorySize)
		
		// Test custom timeout
		assert.Equal(t, int32(30), *config.Timeout)
		
		// Test custom environment variables
		envVars := config.Environment.Variables
		assert.Equal(t, "production", envVars["NODE_ENV"])
		assert.Equal(t, "custom-value", envVars["CUSTOM_VAR"])
		
		// Test VPC configuration if specified
		if config.VpcConfig != nil {
			assert.NotEmpty(t, config.VpcConfig.SubnetIds)
			assert.NotEmpty(t, config.VpcConfig.SecurityGroupIds)
		}
	})

	// Test S3 bucket properties
	t.Run("S3 Bucket Properties", func(t *testing.T) {
		s3Client := s3.NewFromConfig(cfg)
		
		bucketName := findS3Bucket(t, s3Client, stackName)
		require.NotEmpty(t, bucketName)
		
		// Test versioning configuration
		versionResp, err := s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
			Bucket: aws.String(bucketName),
		})
		require.NoError(t, err)
		assert.Equal(t, "Enabled", string(versionResp.Status))
		
		// Test lifecycle configuration
		lifecycleResp, err := s3Client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
			Bucket: aws.String(bucketName),
		})
		require.NoError(t, err)
		
		rules := lifecycleResp.Rules
		require.Len(t, rules, 1)
		assert.Equal(t, "Enabled", string(rules[0].Status))
		assert.Equal(t, int32(30), *rules[0].Expiration.Days)
	})
}

// TestResourceDependencies tests that resource dependencies are correctly established
func TestResourceDependencies(t *testing.T) {
	// Skip if AWS credentials not configured
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("AWS credentials not configured, skipping resource dependencies test")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	// Create test project with dependencies
	projectDir := createTestProjectWithDependencies(t, "resource-dependencies")
	defer cleanupTestProject(t, projectDir)

	// Deploy the project
	stackName := deployTestProject(t, projectDir, "resource-dependencies-test")
	defer removeTestProject(t, projectDir, "resource-dependencies-test")

	t.Run("Lambda Function Dependencies", func(t *testing.T) {
		lambdaClient := lambda.NewFromConfig(cfg)
		s3Client := s3.NewFromConfig(cfg)
		
		functionName := findLambdaFunction(t, lambdaClient, stackName)
		bucketName := findS3Bucket(t, s3Client, stackName)
		
		require.NotEmpty(t, functionName)
		require.NotEmpty(t, bucketName)
		
		// Get function configuration
		resp, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)
		
		// Check that function has access to S3 bucket via environment variable
		envVars := resp.Configuration.Environment.Variables
		assert.Contains(t, envVars, "BUCKET_NAME")
		assert.Equal(t, bucketName, envVars["BUCKET_NAME"])
		
		// Test function can access the bucket
		policy, err := lambdaClient.GetPolicy(ctx, &lambda.GetPolicyInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)
		
		// Policy should contain S3 permissions
		assert.Contains(t, *policy.Policy, "s3:GetObject")
		assert.Contains(t, *policy.Policy, "s3:PutObject")
		assert.Contains(t, *policy.Policy, bucketName)
	})
}

// Helper functions

func createTestProject(t *testing.T, appName string) string {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, appName)
	
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)
	
	// Create basic SST configuration
	sstConfig := fmt.Sprintf(`/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "%s",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Create a simple Lambda function
    const api = new sst.aws.Function("TestFunction", {
      handler: "index.handler",
      runtime: "nodejs20.x",
      memory: "1024 MB",
      timeout: "10 seconds",
    });

    // Create a simple S3 bucket
    const bucket = new sst.aws.Bucket("TestBucket", {
      public: false,
    });

    return {
      api: api.url,
      bucket: bucket.name,
    };
  },
});`, appName)
	
	err = os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644)
	require.NoError(t, err)
	
	// Create Lambda handler
	handler := `export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Hello from Lambda!" }),
  };
};`
	
	err = os.WriteFile(filepath.Join(projectDir, "index.ts"), []byte(handler), 0644)
	require.NoError(t, err)
	
	// Create package.json
	packageJson := fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "sst dev",
    "deploy": "sst deploy"
  },
  "dependencies": {
    "sst": "latest"
  }
}`, appName)
	
	err = os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(packageJson), 0644)
	require.NoError(t, err)
	
	return projectDir
}

func createTestProjectWithConfig(t *testing.T, appName string) string {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, appName)
	
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)
	
	// Create SST configuration with custom properties
	sstConfig := fmt.Sprintf(`/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "%s",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Create Lambda function with custom configuration
    const api = new sst.aws.Function("TestFunction", {
      handler: "index.handler",
      runtime: "nodejs20.x",
      memory: "512 MB",
      timeout: "30 seconds",
      environment: {
        NODE_ENV: "production",
        CUSTOM_VAR: "custom-value",
      },
    });

    // Create S3 bucket with versioning and lifecycle
    const bucket = new sst.aws.Bucket("TestBucket", {
      public: false,
      versioning: true,
      transform: {
        bucket: {
          lifecycleConfiguration: {
            rules: [{
              id: "DeleteOldVersions",
              status: "Enabled",
              expiration: {
                days: 30,
              },
            }],
          },
        },
      },
    });

    return {
      api: api.url,
      bucket: bucket.name,
    };
  },
});`, appName)
	
	err = os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644)
	require.NoError(t, err)
	
	// Create Lambda handler
	handler := `export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({ 
      message: "Hello from Lambda!",
      env: process.env.NODE_ENV,
      custom: process.env.CUSTOM_VAR,
    }),
  };
};`
	
	err = os.WriteFile(filepath.Join(projectDir, "index.ts"), []byte(handler), 0644)
	require.NoError(t, err)
	
	// Create package.json
	packageJson := fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "sst dev",
    "deploy": "sst deploy"
  },
  "dependencies": {
    "sst": "latest"
  }
}`, appName)
	
	err = os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(packageJson), 0644)
	require.NoError(t, err)
	
	return projectDir
}

func createTestProjectWithDependencies(t *testing.T, appName string) string {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, appName)
	
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)
	
	// Create SST configuration with resource dependencies
	sstConfig := fmt.Sprintf(`/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "%s",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Create S3 bucket first
    const bucket = new sst.aws.Bucket("TestBucket", {
      public: false,
    });

    // Create Lambda function that depends on the bucket
    const api = new sst.aws.Function("TestFunction", {
      handler: "index.handler",
      runtime: "nodejs20.x",
      memory: "1024 MB",
      timeout: "10 seconds",
      environment: {
        BUCKET_NAME: bucket.name,
      },
      permissions: [
        {
          actions: ["s3:GetObject", "s3:PutObject"],
          resources: [bucket.arn + "/*"],
        },
      ],
    });

    return {
      api: api.url,
      bucket: bucket.name,
    };
  },
});`, appName)
	
	err = os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644)
	require.NoError(t, err)
	
	// Create Lambda handler that uses S3
	handler := `import { S3Client, GetObjectCommand, PutObjectCommand } from "@aws-sdk/client-s3";

const s3 = new S3Client({});

export const handler = async (event) => {
  const bucketName = process.env.BUCKET_NAME;
  
  try {
    // Test S3 access
    await s3.send(new PutObjectCommand({
      Bucket: bucketName,
      Key: "test.txt",
      Body: "Hello from Lambda!",
    }));
    
    return {
      statusCode: 200,
      body: JSON.stringify({ 
        message: "Successfully accessed S3 bucket",
        bucket: bucketName,
      }),
    };
  } catch (error) {
    return {
      statusCode: 500,
      body: JSON.stringify({ 
        message: "Failed to access S3 bucket",
        error: error.message,
      }),
    };
  }
};`
	
	err = os.WriteFile(filepath.Join(projectDir, "index.ts"), []byte(handler), 0644)
	require.NoError(t, err)
	
	// Create package.json with AWS SDK
	packageJson := fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "sst dev",
    "deploy": "sst deploy"
  },
  "dependencies": {
    "sst": "latest",
    "@aws-sdk/client-s3": "^3.0.0"
  }
}`, appName)
	
	err = os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(packageJson), 0644)
	require.NoError(t, err)
	
	return projectDir
}

func deployTestProject(t *testing.T, projectDir, stage string) string {
	// Change to project directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	
	err = os.Chdir(projectDir)
	require.NoError(t, err)
	
	// Install dependencies
	err = runCommand("npm", "install")
	require.NoError(t, err)
	
	// Deploy the project
	err = runCommand("npx", "sst", "deploy", "--stage", stage)
	require.NoError(t, err)
	
	// Return stack name
	return fmt.Sprintf("%s-%s", filepath.Base(projectDir), stage)
}

func removeTestProject(t *testing.T, projectDir, stage string) {
	// Change to project directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	
	err = os.Chdir(projectDir)
	require.NoError(t, err)
	
	// Remove the project
	err = runCommand("npx", "sst", "remove", "--stage", stage)
	if err != nil {
		t.Logf("Warning: Failed to remove project: %v", err)
	}
}

func cleanupTestProject(t *testing.T, projectDir string) {
	err := os.RemoveAll(projectDir)
	if err != nil {
		t.Logf("Warning: Failed to cleanup project directory: %v", err)
	}
}

func findLambdaFunction(t *testing.T, client *lambda.Client, stackName string) string {
	ctx := context.Background()
	
	// List all functions
	resp, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	require.NoError(t, err)
	
	// Find function that belongs to our stack
	for _, function := range resp.Functions {
		if strings.Contains(*function.FunctionName, stackName) ||
		   strings.Contains(*function.FunctionName, "TestFunction") {
			return *function.FunctionName
		}
	}
	
	return ""
}

func findS3Bucket(t *testing.T, client *s3.Client, stackName string) string {
	ctx := context.Background()
	
	// List all buckets
	resp, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.NoError(t, err)
	
	// Find bucket that belongs to our stack
	for _, bucket := range resp.Buckets {
		if strings.Contains(*bucket.Name, stackName) ||
		   strings.Contains(*bucket.Name, "testbucket") {
			return *bucket.Name
		}
	}
	
	return ""
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}