package resource

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestAwsResource_config(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() *project.Project
		expectError bool
		errorMsg    string
	}{
		{
			name: "returns error when no AWS provider found",
			setupFunc: func() *project.Project {
				return &project.Project{}
			},
			expectError: true,
			errorMsg:    "no aws provider found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			p := tt.setupFunc()
			awsResource := &AwsResource{
				context: ctx,
				project: p,
			}

			config, err := awsResource.config()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, aws.Config{}, config)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, aws.Config{}, config)
			}
		})
	}
}

func TestAwsResource_initialization(t *testing.T) {
	ctx := context.Background()
	p := &project.Project{}
	
	awsResource := &AwsResource{
		context: ctx,
		project: p,
	}

	assert.NotNil(t, awsResource.context)
	assert.NotNil(t, awsResource.project)
	assert.Equal(t, ctx, awsResource.context)
	assert.Equal(t, p, awsResource.project)
}

func TestAwsResource_contextHandling(t *testing.T) {
	// Test with different context types
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "background context",
			ctx:  context.Background(),
		},
		{
			name: "context with value",
			ctx:  context.WithValue(context.Background(), "key", "value"),
		},
		{
			name: "context with timeout",
			ctx: func() context.Context {
				ctx, _ := context.WithCancel(context.Background())
				return ctx
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &project.Project{}

			awsResource := &AwsResource{
				context: tt.ctx,
				project: p,
			}

			assert.Equal(t, tt.ctx, awsResource.context)
			
			// Verify config returns error when no provider is set
			_, err := awsResource.config()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no aws provider found")
		})
	}
}

func TestAwsResource_configErrorHandling(t *testing.T) {
	ctx := context.Background()
	p := &project.Project{}

	awsResource := &AwsResource{
		context: ctx,
		project: p,
	}

	// Test that config method properly handles missing provider
	config, err := awsResource.config()
	assert.Error(t, err)
	assert.Equal(t, "no aws provider found", err.Error())
	assert.Equal(t, aws.Config{}, config)
}

func TestAwsResource_structureValidation(t *testing.T) {
	// Test that AwsResource has the expected fields
	ctx := context.Background()
	p := &project.Project{}

	awsResource := &AwsResource{
		context: ctx,
		project: p,
	}

	// Verify the struct has the expected fields and types
	assert.Implements(t, (*context.Context)(nil), awsResource.context)
	assert.IsType(t, (*project.Project)(nil), awsResource.project)
}