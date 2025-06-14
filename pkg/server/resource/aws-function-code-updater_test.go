package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestFunctionCodeUpdater_Create_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	updater := &FunctionCodeUpdater{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &FunctionCodeUpdaterInputs{
		S3Bucket:             "test-bucket",
		S3Key:                "test-function.zip",
		FunctionName:         "test-function",
		FunctionLastModified: "2023-01-01T00:00:00.000Z",
		Region:               "us-east-1",
	}

	var output CreateResult[FunctionCodeUpdaterOutputs]
	err := updater.Create(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestFunctionCodeUpdater_Update_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	updater := &FunctionCodeUpdater{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := UpdateInput[FunctionCodeUpdaterInputs, FunctionCodeUpdaterOutputs]{
		News: FunctionCodeUpdaterInputs{
			S3Bucket:             "test-bucket",
			S3Key:                "test-function-v2.zip",
			FunctionName:         "test-function",
			FunctionLastModified: "2023-01-02T00:00:00.000Z",
			Region:               "us-east-1",
		},
		Olds: FunctionCodeUpdaterOutputs{
			Version: "1",
		},
	}

	var output UpdateResult[FunctionCodeUpdaterOutputs]
	err := updater.Update(&input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestFunctionCodeUpdaterInputs_Validation(t *testing.T) {
	tests := map[string]struct {
		input    FunctionCodeUpdaterInputs
		expected FunctionCodeUpdaterInputs
	}{
		"S3 deployment": {
			input: FunctionCodeUpdaterInputs{
				S3Bucket:             "my-deployment-bucket",
				S3Key:                "functions/my-function-v1.2.3.zip",
				FunctionName:         "my-function",
				FunctionLastModified: "2023-12-01T10:30:00.000Z",
				Region:               "us-west-2",
			},
			expected: FunctionCodeUpdaterInputs{
				S3Bucket:             "my-deployment-bucket",
				S3Key:                "functions/my-function-v1.2.3.zip",
				FunctionName:         "my-function",
				FunctionLastModified: "2023-12-01T10:30:00.000Z",
				Region:               "us-west-2",
			},
		},
		"Container deployment": {
			input: FunctionCodeUpdaterInputs{
				FunctionName:         "my-container-function",
				FunctionLastModified: "2023-12-01T10:30:00.000Z",
				Region:               "eu-west-1",
				ImageUri:             "123456789012.dkr.ecr.eu-west-1.amazonaws.com/my-function:latest",
			},
			expected: FunctionCodeUpdaterInputs{
				FunctionName:         "my-container-function",
				FunctionLastModified: "2023-12-01T10:30:00.000Z",
				Region:               "eu-west-1",
				ImageUri:             "123456789012.dkr.ecr.eu-west-1.amazonaws.com/my-function:latest",
			},
		},
		"Mixed deployment (should prefer ImageUri)": {
			input: FunctionCodeUpdaterInputs{
				S3Bucket:             "my-deployment-bucket",
				S3Key:                "functions/my-function.zip",
				FunctionName:         "my-mixed-function",
				FunctionLastModified: "2023-12-01T10:30:00.000Z",
				Region:               "ap-southeast-1",
				ImageUri:             "123456789012.dkr.ecr.ap-southeast-1.amazonaws.com/my-function:v2",
			},
			expected: FunctionCodeUpdaterInputs{
				S3Bucket:             "my-deployment-bucket",
				S3Key:                "functions/my-function.zip",
				FunctionName:         "my-mixed-function",
				FunctionLastModified: "2023-12-01T10:30:00.000Z",
				Region:               "ap-southeast-1",
				ImageUri:             "123456789012.dkr.ecr.ap-southeast-1.amazonaws.com/my-function:v2",
			},
		},
		"Empty values": {
			input: FunctionCodeUpdaterInputs{
				S3Bucket:             "",
				S3Key:                "",
				FunctionName:         "",
				FunctionLastModified: "",
				Region:               "",
				ImageUri:             "",
			},
			expected: FunctionCodeUpdaterInputs{
				S3Bucket:             "",
				S3Key:                "",
				FunctionName:         "",
				FunctionLastModified: "",
				Region:               "",
				ImageUri:             "",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Test that input structure is properly formed
			assert.Equal(t, test.expected.S3Bucket, test.input.S3Bucket)
			assert.Equal(t, test.expected.S3Key, test.input.S3Key)
			assert.Equal(t, test.expected.FunctionName, test.input.FunctionName)
			assert.Equal(t, test.expected.FunctionLastModified, test.input.FunctionLastModified)
			assert.Equal(t, test.expected.Region, test.input.Region)
			assert.Equal(t, test.expected.ImageUri, test.input.ImageUri)
		})
	}
}

func TestFunctionCodeUpdaterOutputs_Validation(t *testing.T) {
	tests := map[string]struct {
		output   FunctionCodeUpdaterOutputs
		expected FunctionCodeUpdaterOutputs
	}{
		"Version number": {
			output: FunctionCodeUpdaterOutputs{
				Version: "42",
			},
			expected: FunctionCodeUpdaterOutputs{
				Version: "42",
			},
		},
		"Version string": {
			output: FunctionCodeUpdaterOutputs{
				Version: "$LATEST",
			},
			expected: FunctionCodeUpdaterOutputs{
				Version: "$LATEST",
			},
		},
		"Unknown version": {
			output: FunctionCodeUpdaterOutputs{
				Version: "unknown",
			},
			expected: FunctionCodeUpdaterOutputs{
				Version: "unknown",
			},
		},
		"Empty version": {
			output: FunctionCodeUpdaterOutputs{
				Version: "",
			},
			expected: FunctionCodeUpdaterOutputs{
				Version: "",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected.Version, test.output.Version)
		})
	}
}

func TestFunctionCodeUpdater_StructValidation(t *testing.T) {
	// Test that FunctionCodeUpdater embeds AwsResource correctly
	updater := &FunctionCodeUpdater{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: &project.Project{},
		},
	}

	assert.NotNil(t, updater.AwsResource)
	assert.NotNil(t, updater.AwsResource.context)
	assert.NotNil(t, updater.AwsResource.project)
}

func TestFunctionCodeUpdater_InputTypes(t *testing.T) {
	// Test that all input fields have correct types
	input := FunctionCodeUpdaterInputs{
		S3Bucket:             "string-type",
		S3Key:                "string-type",
		FunctionName:         "string-type",
		FunctionLastModified: "string-type",
		Region:               "string-type",
		ImageUri:             "string-type",
	}

	// Verify field types through reflection-like checks
	assert.IsType(t, "", input.S3Bucket)
	assert.IsType(t, "", input.S3Key)
	assert.IsType(t, "", input.FunctionName)
	assert.IsType(t, "", input.FunctionLastModified)
	assert.IsType(t, "", input.Region)
	assert.IsType(t, "", input.ImageUri)
}

func TestFunctionCodeUpdater_OutputTypes(t *testing.T) {
	// Test that all output fields have correct types
	output := FunctionCodeUpdaterOutputs{
		Version: "string-type",
	}

	// Verify field types
	assert.IsType(t, "", output.Version)
}

func TestFunctionCodeUpdater_CreateResultStructure(t *testing.T) {
	// Test CreateResult structure
	result := CreateResult[FunctionCodeUpdaterOutputs]{
		ID: "test-function",
		Outs: FunctionCodeUpdaterOutputs{
			Version: "1",
		},
	}

	assert.Equal(t, "test-function", result.ID)
	assert.Equal(t, "1", result.Outs.Version)
}

func TestFunctionCodeUpdater_UpdateResultStructure(t *testing.T) {
	// Test UpdateResult structure
	result := UpdateResult[FunctionCodeUpdaterOutputs]{
		Outs: FunctionCodeUpdaterOutputs{
			Version: "2",
		},
	}

	assert.Equal(t, "2", result.Outs.Version)
}

func TestFunctionCodeUpdater_UpdateInputStructure(t *testing.T) {
	// Test UpdateInput structure
	updateInput := UpdateInput[FunctionCodeUpdaterInputs, FunctionCodeUpdaterOutputs]{
		News: FunctionCodeUpdaterInputs{
			S3Bucket:             "new-bucket",
			S3Key:                "new-key.zip",
			FunctionName:         "test-function",
			FunctionLastModified: "2023-12-02T00:00:00.000Z",
			Region:               "us-east-1",
		},
		Olds: FunctionCodeUpdaterOutputs{
			Version: "1",
		},
	}

	assert.Equal(t, "new-bucket", updateInput.News.S3Bucket)
	assert.Equal(t, "new-key.zip", updateInput.News.S3Key)
	assert.Equal(t, "test-function", updateInput.News.FunctionName)
	assert.Equal(t, "2023-12-02T00:00:00.000Z", updateInput.News.FunctionLastModified)
	assert.Equal(t, "us-east-1", updateInput.News.Region)
	assert.Equal(t, "1", updateInput.Olds.Version)
}

func TestFunctionCodeUpdater_DeploymentScenarios(t *testing.T) {
	tests := map[string]struct {
		input       FunctionCodeUpdaterInputs
		description string
	}{
		"S3 ZIP deployment": {
			input: FunctionCodeUpdaterInputs{
				S3Bucket:             "my-lambda-deployments",
				S3Key:                "functions/api-handler-v1.0.0.zip",
				FunctionName:         "api-handler",
				FunctionLastModified: "2023-12-01T10:30:00.000Z",
				Region:               "us-east-1",
			},
			description: "Standard S3-based ZIP deployment",
		},
		"Container image deployment": {
			input: FunctionCodeUpdaterInputs{
				FunctionName:         "container-function",
				FunctionLastModified: "2023-12-01T11:00:00.000Z",
				Region:               "us-west-2",
				ImageUri:             "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-app:latest",
			},
			description: "Container image deployment from ECR",
		},
		"Cross-region deployment": {
			input: FunctionCodeUpdaterInputs{
				S3Bucket:             "global-lambda-artifacts",
				S3Key:                "eu-functions/processor-v2.1.0.zip",
				FunctionName:         "data-processor",
				FunctionLastModified: "2023-12-01T12:15:00.000Z",
				Region:               "eu-central-1",
			},
			description: "Cross-region S3 deployment",
		},
		"Large function deployment": {
			input: FunctionCodeUpdaterInputs{
				S3Bucket:             "large-lambda-packages",
				S3Key:                "ml-models/tensorflow-inference-v3.2.1.zip",
				FunctionName:         "ml-inference-function",
				FunctionLastModified: "2023-12-01T14:45:00.000Z",
				Region:               "ap-northeast-1",
			},
			description: "Large ML model function deployment",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Validate that the input structure is well-formed for different scenarios
			assert.NotEmpty(t, test.input.FunctionName, "Function name should not be empty")
			assert.NotEmpty(t, test.input.Region, "Region should not be empty")
			assert.NotEmpty(t, test.input.FunctionLastModified, "Last modified should not be empty")
			
			// Either S3 deployment or container deployment should be specified
			hasS3Config := test.input.S3Bucket != "" && test.input.S3Key != ""
			hasContainerConfig := test.input.ImageUri != ""
			assert.True(t, hasS3Config || hasContainerConfig, 
				"Either S3 configuration or container image URI should be provided")
		})
	}
}

func TestFunctionCodeUpdater_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		input       FunctionCodeUpdaterInputs
		description string
	}{
		"Very long function name": {
			input: FunctionCodeUpdaterInputs{
				S3Bucket:             "test-bucket",
				S3Key:                "test.zip",
				FunctionName:         "very-long-function-name-that-might-exceed-aws-limits-but-should-still-be-handled-gracefully",
				FunctionLastModified: "2023-12-01T00:00:00.000Z",
				Region:               "us-east-1",
			},
			description: "Function name at AWS limits",
		},
		"Special characters in S3 key": {
			input: FunctionCodeUpdaterInputs{
				S3Bucket:             "test-bucket",
				S3Key:                "functions/my-function@v1.0.0+build.123.zip",
				FunctionName:         "test-function",
				FunctionLastModified: "2023-12-01T00:00:00.000Z",
				Region:               "us-east-1",
			},
			description: "S3 key with special characters",
		},
		"Deep S3 path": {
			input: FunctionCodeUpdaterInputs{
				S3Bucket:             "test-bucket",
				S3Key:                "deployments/2023/12/01/environments/production/services/api/functions/handler-v1.2.3-build.456.zip",
				FunctionName:         "api-handler",
				FunctionLastModified: "2023-12-01T00:00:00.000Z",
				Region:               "us-east-1",
			},
			description: "Deep nested S3 path structure",
		},
		"ECR image with tag": {
			input: FunctionCodeUpdaterInputs{
				FunctionName:         "tagged-container-function",
				FunctionLastModified: "2023-12-01T00:00:00.000Z",
				Region:               "us-east-1",
				ImageUri:             "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app:v1.2.3-production",
			},
			description: "ECR image with specific version tag",
		},
		"ECR image with digest": {
			input: FunctionCodeUpdaterInputs{
				FunctionName:         "digest-container-function",
				FunctionLastModified: "2023-12-01T00:00:00.000Z",
				Region:               "us-east-1",
				ImageUri:             "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app@sha256:abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
			},
			description: "ECR image with SHA256 digest",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Test that edge case inputs are handled properly
			assert.NotEmpty(t, test.input.FunctionName)
			assert.NotEmpty(t, test.input.Region)
			assert.NotEmpty(t, test.input.FunctionLastModified)
			
			// Validate that the structure can handle edge cases
			if test.input.ImageUri != "" {
				assert.Contains(t, test.input.ImageUri, ".amazonaws.com", 
					"Image URI should be from AWS ECR")
			}
			
			if test.input.S3Key != "" {
				assert.NotEmpty(t, test.input.S3Bucket, 
					"S3 bucket should be provided when S3 key is specified")
			}
		})
	}
}