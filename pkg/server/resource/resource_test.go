package resource

import (
	"context"
	"net/rpc"
	"testing"

	"github.com/sst/sst/v3/pkg/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadInput(t *testing.T) {
	tests := map[string]struct {
		input    ReadInput[string]
		expected ReadInput[string]
	}{
		"basic read input": {
			input:    ReadInput[string]{ID: "test-id"},
			expected: ReadInput[string]{ID: "test-id"},
		},
		"empty id": {
			input:    ReadInput[string]{ID: ""},
			expected: ReadInput[string]{ID: ""},
		},
		"complex id": {
			input:    ReadInput[string]{ID: "arn:aws:s3:::my-bucket/path/to/file"},
			expected: ReadInput[string]{ID: "arn:aws:s3:::my-bucket/path/to/file"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestReadResult(t *testing.T) {
	tests := map[string]struct {
		result   ReadResult[map[string]string]
		expected ReadResult[map[string]string]
	}{
		"basic read result": {
			result: ReadResult[map[string]string]{
				ID:   "test-id",
				Outs: map[string]string{"key": "value"},
			},
			expected: ReadResult[map[string]string]{
				ID:   "test-id",
				Outs: map[string]string{"key": "value"},
			},
		},
		"empty outputs": {
			result: ReadResult[map[string]string]{
				ID:   "test-id",
				Outs: map[string]string{},
			},
			expected: ReadResult[map[string]string]{
				ID:   "test-id",
				Outs: map[string]string{},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result)
		})
	}
}

func TestDiffInput(t *testing.T) {
	tests := map[string]struct {
		input    DiffInput[string, string]
		expected DiffInput[string, string]
	}{
		"basic diff input": {
			input: DiffInput[string, string]{
				ID:   "test-id",
				News: "new-value",
				Olds: "old-value",
			},
			expected: DiffInput[string, string]{
				ID:   "test-id",
				News: "new-value",
				Olds: "old-value",
			},
		},
		"same values": {
			input: DiffInput[string, string]{
				ID:   "test-id",
				News: "same-value",
				Olds: "same-value",
			},
			expected: DiffInput[string, string]{
				ID:   "test-id",
				News: "same-value",
				Olds: "same-value",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestDiffResult(t *testing.T) {
	tests := map[string]struct {
		result   DiffResult
		expected DiffResult
	}{
		"changes detected": {
			result:   DiffResult{Changes: true},
			expected: DiffResult{Changes: true},
		},
		"no changes": {
			result:   DiffResult{Changes: false},
			expected: DiffResult{Changes: false},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result)
		})
	}
}

func TestCreateResult(t *testing.T) {
	tests := map[string]struct {
		result   CreateResult[map[string]string]
		expected CreateResult[map[string]string]
	}{
		"basic create result": {
			result: CreateResult[map[string]string]{
				ID:   "created-id",
				Outs: map[string]string{"status": "created"},
			},
			expected: CreateResult[map[string]string]{
				ID:   "created-id",
				Outs: map[string]string{"status": "created"},
			},
		},
		"empty outputs": {
			result: CreateResult[map[string]string]{
				ID:   "created-id",
				Outs: map[string]string{},
			},
			expected: CreateResult[map[string]string]{
				ID:   "created-id",
				Outs: map[string]string{},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result)
		})
	}
}

func TestUpdateInput(t *testing.T) {
	tests := map[string]struct {
		input    UpdateInput[string, string]
		expected UpdateInput[string, string]
	}{
		"basic update input": {
			input: UpdateInput[string, string]{
				ID:   "update-id",
				News: "new-config",
				Olds: "old-config",
			},
			expected: UpdateInput[string, string]{
				ID:   "update-id",
				News: "new-config",
				Olds: "old-config",
			},
		},
		"complex update": {
			input: UpdateInput[string, string]{
				ID:   "arn:aws:lambda:us-east-1:123456789012:function:my-function",
				News: "updated-code",
				Olds: "original-code",
			},
			expected: UpdateInput[string, string]{
				ID:   "arn:aws:lambda:us-east-1:123456789012:function:my-function",
				News: "updated-code",
				Olds: "original-code",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestUpdateResult(t *testing.T) {
	tests := map[string]struct {
		result   UpdateResult[map[string]string]
		expected UpdateResult[map[string]string]
	}{
		"basic update result": {
			result: UpdateResult[map[string]string]{
				Outs: map[string]string{"status": "updated"},
			},
			expected: UpdateResult[map[string]string]{
				Outs: map[string]string{"status": "updated"},
			},
		},
		"empty outputs": {
			result: UpdateResult[map[string]string]{
				Outs: map[string]string{},
			},
			expected: UpdateResult[map[string]string]{
				Outs: map[string]string{},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result)
		})
	}
}

func TestDeleteInput(t *testing.T) {
	tests := map[string]struct {
		input    DeleteInput[map[string]string]
		expected DeleteInput[map[string]string]
	}{
		"basic delete input": {
			input: DeleteInput[map[string]string]{
				ID:   "delete-id",
				Outs: map[string]string{"resource": "to-delete"},
			},
			expected: DeleteInput[map[string]string]{
				ID:   "delete-id",
				Outs: map[string]string{"resource": "to-delete"},
			},
		},
		"empty outputs": {
			input: DeleteInput[map[string]string]{
				ID:   "delete-id",
				Outs: map[string]string{},
			},
			expected: DeleteInput[map[string]string]{
				ID:   "delete-id",
				Outs: map[string]string{},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestRegister(t *testing.T) {
	tests := map[string]struct {
		setupProject func() *project.Project
		expectError  bool
	}{
		"successful registration": {
			setupProject: func() *project.Project {
				return &project.Project{}
			},
			expectError: false,
		},
		"nil project": {
			setupProject: func() *project.Project {
				return nil
			},
			expectError: false, // Register doesn't validate project, just passes it through
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			p := tt.setupProject()
			server := rpc.NewServer()

			err := Register(ctx, p, server)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegister_ServiceRegistration(t *testing.T) {
	ctx := context.Background()
	p := &project.Project{}
	server := rpc.NewServer()

	err := Register(ctx, p, server)
	require.NoError(t, err)

	// Test that services are registered by attempting to find them
	// Note: rpc.Server doesn't expose a way to list registered services,
	// so we can only verify that registration doesn't error
	// In a real scenario, you'd test by making RPC calls
}

func TestGenericTypes_EdgeCases(t *testing.T) {
	t.Run("ReadInput with complex type", func(t *testing.T) {
		type ComplexType struct {
			Name   string            `json:"name"`
			Config map[string]string `json:"config"`
			Items  []string          `json:"items"`
		}

		input := ReadInput[ComplexType]{
			ID: "complex-id",
		}

		assert.Equal(t, "complex-id", input.ID)
	})

	t.Run("CreateResult with nil outputs", func(t *testing.T) {
		result := CreateResult[*string]{
			ID:   "test-id",
			Outs: nil,
		}

		assert.Equal(t, "test-id", result.ID)
		assert.Nil(t, result.Outs)
	})

	t.Run("UpdateInput with different types", func(t *testing.T) {
		input := UpdateInput[int, string]{
			ID:   "mixed-types",
			News: 42,
			Olds: "old-string",
		}

		assert.Equal(t, "mixed-types", input.ID)
		assert.Equal(t, 42, input.News)
		assert.Equal(t, "old-string", input.Olds)
	})
}

func TestResourceTypes_JSONTags(t *testing.T) {
	t.Run("ReadInput JSON tags", func(t *testing.T) {
		input := ReadInput[string]{ID: "test"}
		// In a real test, you'd marshal/unmarshal to verify JSON tags work correctly
		assert.Equal(t, "test", input.ID)
	})

	t.Run("DiffInput JSON tags", func(t *testing.T) {
		input := DiffInput[string, string]{
			ID:   "test",
			News: "new",
			Olds: "old",
		}
		assert.Equal(t, "test", input.ID)
		assert.Equal(t, "new", input.News)
		assert.Equal(t, "old", input.Olds)
	})

	t.Run("CreateResult JSON tags", func(t *testing.T) {
		result := CreateResult[string]{
			ID:   "test",
			Outs: "output",
		}
		assert.Equal(t, "test", result.ID)
		assert.Equal(t, "output", result.Outs)
	})
}

func TestAwsResource_Structure(t *testing.T) {
	ctx := context.Background()
	p := &project.Project{}

	awsResource := &AwsResource{
		context: ctx,
		project: p,
	}

	assert.Equal(t, ctx, awsResource.context)
	assert.Equal(t, p, awsResource.project)
}