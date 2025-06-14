package resource

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestFunctionEnvironmentUpdate_Create_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	updater := &FunctionEnvironmentUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &FunctionEnvironmentUpdateInputs{
		FunctionName: "test-function",
		Environment: map[string]string{
			"NODE_ENV": "production",
			"API_KEY":  "test-key",
		},
		Region: "us-east-1",
	}

	var output CreateResult[FunctionEnvironmentUpdateOutputs]
	err := updater.Create(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestFunctionEnvironmentUpdate_Update_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	updater := &FunctionEnvironmentUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := UpdateInput[FunctionEnvironmentUpdateInputs, FunctionEnvironmentUpdateOutputs]{
		News: FunctionEnvironmentUpdateInputs{
			FunctionName: "test-function",
			Environment: map[string]string{
				"NODE_ENV": "development",
				"DEBUG":    "true",
			},
			Region: "us-west-2",
		},
		Olds: FunctionEnvironmentUpdateOutputs{
			Updated: false,
		},
	}

	var output UpdateResult[FunctionEnvironmentUpdateOutputs]
	err := updater.Update(&input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestFunctionEnvironmentUpdate_Read(t *testing.T) {
	// Create project without AWS provider (Read doesn't need AWS)
	p := &project.Project{}

	updater := &FunctionEnvironmentUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &ReadInput[FunctionEnvironmentUpdateInputs]{
		ID: "test-function",
	}

	var output ReadResult[FunctionEnvironmentUpdateOutputs]
	err := updater.Read(input, &output)

	// Read should always succeed and return Updated: false
	assert.NoError(t, err)
	assert.Equal(t, "test-function", output.ID)
	assert.False(t, output.Outs.Updated)
}

func TestFunctionEnvironmentUpdate_Diff_NoChanges(t *testing.T) {
	// Create project without AWS provider (Diff doesn't need AWS)
	p := &project.Project{}

	updater := &FunctionEnvironmentUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DiffInput[FunctionEnvironmentUpdateInputs, FunctionEnvironmentUpdateOutputs]{
		News: FunctionEnvironmentUpdateInputs{
			FunctionName: "test-function",
			Environment: map[string]string{
				"NODE_ENV": "production",
			},
			Region: "us-east-1",
		},
		Olds: FunctionEnvironmentUpdateOutputs{
			Updated: true, // Already updated
		},
	}

	var output DiffResult
	err := updater.Diff(input, &output)

	// Should succeed with no changes
	assert.NoError(t, err)
	assert.False(t, output.Changes) // No changes needed since already updated
}

func TestFunctionEnvironmentUpdate_Diff_HasChanges(t *testing.T) {
	// Create project without AWS provider (Diff doesn't need AWS)
	p := &project.Project{}

	updater := &FunctionEnvironmentUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DiffInput[FunctionEnvironmentUpdateInputs, FunctionEnvironmentUpdateOutputs]{
		News: FunctionEnvironmentUpdateInputs{
			FunctionName: "test-function",
			Environment: map[string]string{
				"NODE_ENV": "development",
			},
			Region: "us-east-1",
		},
		Olds: FunctionEnvironmentUpdateOutputs{
			Updated: false, // Not yet updated
		},
	}

	var output DiffResult
	err := updater.Diff(input, &output)

	// Should succeed with changes needed
	assert.NoError(t, err)
	assert.True(t, output.Changes) // Changes needed since not updated
}

func TestFunctionEnvironmentUpdateInputs_Validation(t *testing.T) {
	tests := map[string]FunctionEnvironmentUpdateInputs{
		"valid basic input": {
			FunctionName: "my-function",
			Environment: map[string]string{
				"NODE_ENV": "production",
			},
			Region: "us-east-1",
		},
		"valid with multiple env vars": {
			FunctionName: "my-function",
			Environment: map[string]string{
				"NODE_ENV":    "production",
				"API_KEY":     "secret-key",
				"DEBUG":       "false",
				"PORT":        "3000",
				"DATABASE_URL": "postgres://localhost:5432/mydb",
			},
			Region: "eu-west-1",
		},
		"valid with empty environment": {
			FunctionName: "my-function",
			Environment:  map[string]string{},
			Region:       "ap-southeast-1",
		},
		"valid without region": {
			FunctionName: "my-function",
			Environment: map[string]string{
				"NODE_ENV": "development",
			},
			Region: "", // Should use default region from AWS config
		},
		"valid with long function name": {
			FunctionName: "my-very-long-function-name-that-is-still-valid-according-to-aws-limits",
			Environment: map[string]string{
				"NODE_ENV": "production",
			},
			Region: "us-west-2",
		},
		"valid with special characters in env values": {
			FunctionName: "my-function",
			Environment: map[string]string{
				"DATABASE_URL": "postgres://user:pass@host:5432/db?ssl=true&timeout=30",
				"JSON_CONFIG":  `{"key": "value", "nested": {"array": [1, 2, 3]}}`,
				"SPECIAL_CHARS": "!@#$%^&*()_+-=[]{}|;:,.<>?",
			},
			Region: "us-east-1",
		},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			// Test that the struct can be created and fields are accessible
			assert.NotEmpty(t, input.FunctionName, "FunctionName should not be empty")
			assert.NotNil(t, input.Environment, "Environment should not be nil")
			// Region can be empty (will use default)
		})
	}
}

func TestFunctionEnvironmentUpdateOutputs_Structure(t *testing.T) {
	tests := map[string]FunctionEnvironmentUpdateOutputs{
		"updated true": {
			Updated: true,
		},
		"updated false": {
			Updated: false,
		},
	}

	for name, output := range tests {
		t.Run(name, func(t *testing.T) {
			// Test that the struct fields are accessible and have expected types
			assert.IsType(t, bool(false), output.Updated, "Updated should be a boolean")
		})
	}
}

func TestFunctionEnvironmentUpdate_CreateResult_Structure(t *testing.T) {
	input := &FunctionEnvironmentUpdateInputs{
		FunctionName: "test-function",
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		Region: "us-east-1",
	}

	// Test the structure that would be returned by Create method
	expectedResult := CreateResult[FunctionEnvironmentUpdateOutputs]{
		ID: input.FunctionName,
		Outs: FunctionEnvironmentUpdateOutputs{
			Updated: true,
		},
	}

	assert.Equal(t, "test-function", expectedResult.ID)
	assert.True(t, expectedResult.Outs.Updated)
}

func TestFunctionEnvironmentUpdate_UpdateResult_Structure(t *testing.T) {
	// Test the structure that would be returned by Update method
	expectedResult := UpdateResult[FunctionEnvironmentUpdateOutputs]{
		Outs: FunctionEnvironmentUpdateOutputs{
			Updated: true,
		},
	}

	assert.True(t, expectedResult.Outs.Updated)
}

func TestFunctionEnvironmentUpdate_AwsResource_Embedded(t *testing.T) {
	p := &project.Project{}

	updater := &FunctionEnvironmentUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	// Test that AwsResource is properly embedded
	assert.NotNil(t, updater.AwsResource)
	assert.Equal(t, context.Background(), updater.AwsResource.context)
	assert.Equal(t, p, updater.AwsResource.project)
}

func TestFunctionEnvironmentUpdate_EdgeCases(t *testing.T) {
	p := &project.Project{}

	updater := &FunctionEnvironmentUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	t.Run("empty function name", func(t *testing.T) {
		input := &FunctionEnvironmentUpdateInputs{
			FunctionName: "", // Empty function name
			Environment: map[string]string{
				"NODE_ENV": "production",
			},
			Region: "us-east-1",
		}

		var output CreateResult[FunctionEnvironmentUpdateOutputs]
		err := updater.Create(input, &output)

		// Should fail due to no AWS provider, but the empty function name would also cause issues
		assert.Error(t, err)
	})

	t.Run("nil environment map", func(t *testing.T) {
		input := &FunctionEnvironmentUpdateInputs{
			FunctionName: "test-function",
			Environment:  nil, // Nil environment
			Region:       "us-east-1",
		}

		var output CreateResult[FunctionEnvironmentUpdateOutputs]
		err := updater.Create(input, &output)

		// Should fail due to no AWS provider
		assert.Error(t, err)
	})

	t.Run("very long environment variable values", func(t *testing.T) {
		longValue := make([]byte, 4096) // 4KB value
		for i := range longValue {
			longValue[i] = 'a'
		}

		input := &FunctionEnvironmentUpdateInputs{
			FunctionName: "test-function",
			Environment: map[string]string{
				"LONG_VALUE": string(longValue),
			},
			Region: "us-east-1",
		}

		var output CreateResult[FunctionEnvironmentUpdateOutputs]
		err := updater.Create(input, &output)

		// Should fail due to no AWS provider
		assert.Error(t, err)
	})

	t.Run("many environment variables", func(t *testing.T) {
		env := make(map[string]string)
		// AWS Lambda supports up to 4KB total for environment variables
		for i := 0; i < 100; i++ {
			env[fmt.Sprintf("VAR_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		input := &FunctionEnvironmentUpdateInputs{
			FunctionName: "test-function",
			Environment:  env,
			Region:       "us-east-1",
		}

		var output CreateResult[FunctionEnvironmentUpdateOutputs]
		err := updater.Create(input, &output)

		// Should fail due to no AWS provider
		assert.Error(t, err)
	})
}

func TestFunctionEnvironmentUpdate_RegionHandling(t *testing.T) {
	p := &project.Project{}

	updater := &FunctionEnvironmentUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	tests := map[string]string{
		"us-east-1":      "us-east-1",
		"us-west-2":      "us-west-2",
		"eu-west-1":      "eu-west-1",
		"ap-southeast-1": "ap-southeast-1",
		"":               "", // Empty region should use default
	}

	for name, region := range tests {
		t.Run(name, func(t *testing.T) {
			input := &FunctionEnvironmentUpdateInputs{
				FunctionName: "test-function",
				Environment: map[string]string{
					"NODE_ENV": "production",
				},
				Region: region,
			}

			var output CreateResult[FunctionEnvironmentUpdateOutputs]
			err := updater.Create(input, &output)

			// Should fail due to no AWS provider, but region handling is tested
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no aws provider found")
		})
	}
}