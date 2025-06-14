package aws

import (
	"fmt"
	"testing"

	"github.com/sst/sst/v3/test/integration/pulumi/helpers"
	"github.com/stretchr/testify/require"
)

// TestBasicAWSDeployment tests deploying a simple SST project to real AWS
func TestBasicAWSDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	helpers.RunIntegrationTest(t, "basic-aws", func(t *testing.T, config *helpers.PulumiIntegrationTestConfig, projectDir string) {
		// Deploy the project
		result, err := helpers.DeployWithPolicies(t, projectDir, config.TestStage, config.PolicyPackPath)
		require.NoError(t, err, "Failed to deploy project")
		require.NotNil(t, result, "Deploy result should not be nil")

		// Validate deployment
		validators := []helpers.ResourceValidator{
			helpers.ValidateBucketExists("bucket"),
		}

		err = helpers.ValidateDeployment(t, "test-"+config.TestStage, validators, projectDir)
		require.NoError(t, err, "Failed to validate deployment")

		t.Logf("Basic AWS deployment test completed successfully")
	})
}

// TestFunctionDeployment tests deploying a Lambda function with SST
func TestFunctionDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	helpers.RunIntegrationTest(t, "function", func(t *testing.T, config *helpers.PulumiIntegrationTestConfig, projectDir string) {
		// Create a more complex SST config with a function
		sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "test-function-app",
      removal: input?.stage === "pulumi-integration-test" ? "remove" : "retain",
      home: "aws",
    };
  },
  async run() {
    // Create a simple function
    const fn = new sst.aws.Function("TestFunction", {
      handler: "index.handler",
      runtime: "nodejs20.x",
      code: {
        zipFile: "exports.handler = async (event) => { return { statusCode: 200, body: 'Hello World' }; };"
      }
    });
    
    const bucket = new sst.aws.Bucket("TestBucket");
    
    return {
      function: fn.name,
      bucket: bucket.name,
    };
  },
});`

		// Write the config
		err := helpers.WriteFile(projectDir, "sst.config.ts", sstConfig)
		require.NoError(t, err, "Failed to write SST config")

		// Deploy the project
		result, err := helpers.DeployWithPolicies(t, projectDir, config.TestStage, config.PolicyPackPath)
		require.NoError(t, err, "Failed to deploy project")
		require.NotNil(t, result, "Deploy result should not be nil")

		// Validate deployment
		validators := []helpers.ResourceValidator{
			helpers.ValidateBucketExists("bucket"),
			helpers.ValidateFunctionExists("function"),
		}

		err = helpers.ValidateDeployment(t, "test-"+config.TestStage, validators, projectDir)
		require.NoError(t, err, "Failed to validate deployment")

		t.Logf("Function deployment test completed successfully")
	})
}

// TestBucketDeployment tests deploying an S3 bucket with SST
func TestBucketDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Clean up test artifacts at the end
	defer helpers.CleanupTestArtifacts()

	helpers.RunIntegrationTest(t, "bucket", func(t *testing.T, config *helpers.PulumiIntegrationTestConfig, projectDir string) {
		// Create SST config with bucket configuration
		sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "test-bucket-app",
      removal: input?.stage === "pulumi-integration-test" ? "remove" : "retain",
      home: "aws",
    };
  },
  async run() {
    // Create bucket with public access
    const bucket = new sst.aws.Bucket("TestBucket", {
      public: true,
      cors: {
        allowCredentials: false,
        allowHeaders: ["*"],
        allowMethods: ["GET", "POST", "PUT", "DELETE"],
        allowOrigins: ["*"],
        exposeHeaders: [],
        maxAge: "1 day"
      }
    });
    
    return {
      bucket: bucket.name,
      bucketDomain: bucket.domain,
    };
  },
});`

		// Write the config
		err := helpers.WriteFile(projectDir, "sst.config.ts", sstConfig)
		require.NoError(t, err, "Failed to write SST config")

		// Deploy the project
		result, err := helpers.DeployWithPolicies(t, projectDir, config.TestStage, config.PolicyPackPath)
		require.NoError(t, err, "Failed to deploy project")
		require.NotNil(t, result, "Deploy result should not be nil")

		// Validate deployment
		validators := []helpers.ResourceValidator{
			helpers.ValidateBucketExists("bucket"),
			func(t *testing.T, outputs helpers.OutputMap) error {
				// Validate bucket domain exists
				domainOutput, exists := outputs["bucketDomain"]
				if !exists {
					return fmt.Errorf("bucket domain output not found")
				}
				
				if domainOutput == nil {
					return fmt.Errorf("bucket domain output is nil")
				}
				
				domain, ok := domainOutput.(string)
				if !ok {
					return fmt.Errorf("bucket domain is not a string")
				}
				
				if domain == "" {
					return fmt.Errorf("bucket domain is empty")
				}
				
				t.Logf("Validated bucket domain: %s", domain)
				return nil
			},
		}

		err = helpers.ValidateDeployment(t, "test-"+config.TestStage, validators, projectDir)
		require.NoError(t, err, "Failed to validate deployment")

		t.Logf("Bucket deployment test completed successfully")
	})
}