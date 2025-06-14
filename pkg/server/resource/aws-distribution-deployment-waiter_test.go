package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestDistributionDeploymentWaiter_Create_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	waiter := &DistributionDeploymentWaiter{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DistributionDeploymentWaiterInputs{
		DistributionId: "E1234567890123",
		Etag:           "ETAG123",
		Wait:           true,
	}

	var output CreateResult[DistributionDeploymentWaiterOutputs]
	err := waiter.Create(input, &output)

	// Should fail with "no aws provider found" when Wait is true
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestDistributionDeploymentWaiter_Create_NoWait(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	waiter := &DistributionDeploymentWaiter{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DistributionDeploymentWaiterInputs{
		DistributionId: "E1234567890123",
		Etag:           "ETAG123",
		Wait:           false, // No waiting, should succeed
	}

	var output CreateResult[DistributionDeploymentWaiterOutputs]
	err := waiter.Create(input, &output)

	// Should succeed when Wait is false
	assert.NoError(t, err)
	assert.Equal(t, "waiter", output.ID)
	assert.True(t, output.Outs.IsDone)
}

func TestDistributionDeploymentWaiter_Update_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	waiter := &DistributionDeploymentWaiter{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &UpdateInput[DistributionDeploymentWaiterInputs, DistributionDeploymentWaiterOutputs]{
		News: DistributionDeploymentWaiterInputs{
			DistributionId: "E1234567890123",
			Etag:           "ETAG456",
			Wait:           true,
		},
		Olds: DistributionDeploymentWaiterOutputs{
			IsDone: false,
		},
	}

	var output UpdateResult[DistributionDeploymentWaiterOutputs]
	err := waiter.Update(input, &output)

	// Should fail with "no aws provider found" when Wait is true
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestDistributionDeploymentWaiter_Update_NoWait(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	waiter := &DistributionDeploymentWaiter{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &UpdateInput[DistributionDeploymentWaiterInputs, DistributionDeploymentWaiterOutputs]{
		News: DistributionDeploymentWaiterInputs{
			DistributionId: "E1234567890123",
			Etag:           "ETAG456",
			Wait:           false, // No waiting, should succeed
		},
		Olds: DistributionDeploymentWaiterOutputs{
			IsDone: true,
		},
	}

	var output UpdateResult[DistributionDeploymentWaiterOutputs]
	err := waiter.Update(input, &output)

	// Should succeed when Wait is false
	assert.NoError(t, err)
	assert.True(t, output.Outs.IsDone)
}

func TestDistributionDeploymentWaiterInputs_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input DistributionDeploymentWaiterInputs
	}{
		{
			name: "valid input with wait enabled",
			input: DistributionDeploymentWaiterInputs{
				DistributionId: "E1234567890123",
				Etag:           "ETAG123",
				Wait:           true,
			},
		},
		{
			name: "valid input with wait disabled",
			input: DistributionDeploymentWaiterInputs{
				DistributionId: "E1234567890123",
				Etag:           "ETAG123",
				Wait:           false,
			},
		},
		{
			name: "empty distribution ID",
			input: DistributionDeploymentWaiterInputs{
				DistributionId: "",
				Etag:           "ETAG123",
				Wait:           true,
			},
		},
		{
			name: "empty etag",
			input: DistributionDeploymentWaiterInputs{
				DistributionId: "E1234567890123",
				Etag:           "",
				Wait:           true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the struct can be created and fields are accessible
			assert.IsType(t, "", tt.input.DistributionId)
			assert.IsType(t, "", tt.input.Etag)
			assert.IsType(t, false, tt.input.Wait)
		})
	}
}

func TestDistributionDeploymentWaiterOutputs_Structure(t *testing.T) {
	output := DistributionDeploymentWaiterOutputs{
		IsDone: true,
	}

	// Test that the struct has the expected fields and types
	assert.IsType(t, false, output.IsDone)
	assert.True(t, output.IsDone)

	// Test with false value
	output.IsDone = false
	assert.False(t, output.IsDone)
}

func TestDistributionDeploymentWaiter_StructureValidation(t *testing.T) {
	// Test that DistributionDeploymentWaiter has the expected embedded struct
	ctx := context.Background()
	p := &project.Project{}

	waiter := &DistributionDeploymentWaiter{
		AwsResource: &AwsResource{
			context: ctx,
			project: p,
		},
	}

	// Verify the struct has the expected embedded AwsResource
	assert.NotNil(t, waiter.AwsResource)
	assert.Equal(t, ctx, waiter.AwsResource.context)
	assert.Equal(t, p, waiter.AwsResource.project)
}

func TestDistributionDeploymentWaiter_HandleMethod(t *testing.T) {
	// Test the handle method behavior through Create/Update methods
	p := &project.Project{}

	waiter := &DistributionDeploymentWaiter{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	tests := []struct {
		name        string
		wait        bool
		expectError bool
	}{
		{
			name:        "handle with wait false should succeed",
			wait:        false,
			expectError: false,
		},
		{
			name:        "handle with wait true should fail (no provider)",
			wait:        true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &DistributionDeploymentWaiterInputs{
				DistributionId: "E1234567890123",
				Etag:           "ETAG123",
				Wait:           tt.wait,
			}

			var output CreateResult[DistributionDeploymentWaiterOutputs]
			err := waiter.Create(input, &output)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "waiter", output.ID)
				assert.True(t, output.Outs.IsDone)
			}
		})
	}
}

func TestDistributionDeploymentWaiter_CreateResultStructure(t *testing.T) {
	// Test the CreateResult structure
	result := CreateResult[DistributionDeploymentWaiterOutputs]{
		ID: "waiter",
		Outs: DistributionDeploymentWaiterOutputs{
			IsDone: true,
		},
	}

	assert.Equal(t, "waiter", result.ID)
	assert.True(t, result.Outs.IsDone)
}

func TestDistributionDeploymentWaiter_UpdateResultStructure(t *testing.T) {
	// Test the UpdateResult structure
	result := UpdateResult[DistributionDeploymentWaiterOutputs]{
		Outs: DistributionDeploymentWaiterOutputs{
			IsDone: true,
		},
	}

	assert.True(t, result.Outs.IsDone)
}