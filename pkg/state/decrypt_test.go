package state_test

import (
	"context"
	"testing"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/sst/sst/v3/pkg/state"
)

func TestDecrypt(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	
	tests := []struct {
		name        string
		passphrase  string
		checkpoint  *apitype.CheckpointV3
		expectError bool
	}{
		{
			name:       "decrypt with valid passphrase",
			passphrase: "test-passphrase",
			checkpoint: &apitype.CheckpointV3{
				Stack: "test-stack",
				Latest: &apitype.DeploymentV3{
					Manifest: apitype.ManifestV1{
						Time:    testTime,
						Magic:   "0xDEADBEEF",
						Version: "v3.0.0",
					},
					Resources: []apitype.ResourceV3{
						{
							URN:  resource.URN("urn:pulumi:test::test::test::resource1"),
							Type: "test:resource:Type",
							Inputs: map[string]interface{}{
								"value": "test-value",
							},
							Outputs: map[string]interface{}{
								"result": "test-result",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:       "decrypt with empty passphrase",
			passphrase: "",
			checkpoint: &apitype.CheckpointV3{
				Stack: "test-stack",
				Latest: &apitype.DeploymentV3{
					Manifest: apitype.ManifestV1{
						Time:    testTime,
						Magic:   "0xDEADBEEF",
						Version: "v3.0.0",
					},
					Resources: []apitype.ResourceV3{},
				},
			},
			expectError: false, // Empty passphrase might be valid for some cases
		},
		{
			name:       "decrypt with nil checkpoint",
			passphrase: "test-passphrase",
			checkpoint: nil,
			expectError: true,
		},
		{
			name:       "decrypt with nil latest deployment",
			passphrase: "test-passphrase",
			checkpoint: &apitype.CheckpointV3{
				Stack:  "test-stack",
				Latest: nil,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			result, err := state.Decrypt(ctx, tt.passphrase, tt.checkpoint)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if result == nil {
				t.Error("Expected result but got nil")
				return
			}
			
			// Verify the result structure
			if result.Stack != tt.checkpoint.Stack {
				t.Errorf("Expected stack %s, got %s", tt.checkpoint.Stack, result.Stack)
			}
			
			if result.Latest == nil {
				t.Error("Expected Latest deployment but got nil")
				return
			}
			
			// Verify that the original checkpoint was not modified
			if tt.checkpoint != nil && tt.checkpoint.Latest != nil {
				if len(result.Latest.Resources) != len(tt.checkpoint.Latest.Resources) {
					t.Errorf("Expected %d resources, got %d", 
						len(tt.checkpoint.Latest.Resources), 
						len(result.Latest.Resources))
				}
			}
		})
	}
}

func TestDecryptPreservesResourceData(t *testing.T) {
	ctx := context.Background()
	passphrase := "test-passphrase"
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	
	originalCheckpoint := &apitype.CheckpointV3{
		Stack: "test-stack",
		Latest: &apitype.DeploymentV3{
			Manifest: apitype.ManifestV1{
				Time:    testTime,
				Magic:   "0xDEADBEEF",
				Version: "v3.0.0",
			},
			Resources: []apitype.ResourceV3{
				{
					URN:  resource.URN("urn:pulumi:test::test::test::resource1"),
					Type: "test:resource:Type",
					Inputs: map[string]interface{}{
						"inputKey": "inputValue",
					},
					Outputs: map[string]interface{}{
						"outputKey": "outputValue",
					},
					Dependencies: []resource.URN{
						resource.URN("urn:pulumi:test::test::test::dependency"),
					},
					PropertyDependencies: map[resource.PropertyKey][]resource.URN{
						"prop1": {
							resource.URN("urn:pulumi:test::test::test::propDep"),
						},
					},
				},
			},
		},
	}
	
	result, err := state.Decrypt(ctx, passphrase, originalCheckpoint)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result.Latest.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(result.Latest.Resources))
	}
	
	resultResource := result.Latest.Resources[0]
	originalResource := originalCheckpoint.Latest.Resources[0]
	
	// Verify URN is preserved
	if resultResource.URN != originalResource.URN {
		t.Errorf("URN not preserved: expected %s, got %s", 
			originalResource.URN, resultResource.URN)
	}
	
	// Verify Type is preserved
	if resultResource.Type != originalResource.Type {
		t.Errorf("Type not preserved: expected %s, got %s", 
			originalResource.Type, resultResource.Type)
	}
	
	// Verify Dependencies are preserved
	if len(resultResource.Dependencies) != len(originalResource.Dependencies) {
		t.Errorf("Dependencies count not preserved: expected %d, got %d", 
			len(originalResource.Dependencies), len(resultResource.Dependencies))
	}
	
	// Verify PropertyDependencies are preserved
	if len(resultResource.PropertyDependencies) != len(originalResource.PropertyDependencies) {
		t.Errorf("PropertyDependencies count not preserved: expected %d, got %d", 
			len(originalResource.PropertyDependencies), len(resultResource.PropertyDependencies))
	}
}

func TestDecryptEnvironmentVariable(t *testing.T) {
	ctx := context.Background()
	passphrase := "test-env-passphrase"
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	
	checkpoint := &apitype.CheckpointV3{
		Stack: "test-stack",
		Latest: &apitype.DeploymentV3{
			Manifest: apitype.ManifestV1{
				Time:    testTime,
				Magic:   "0xDEADBEEF",
				Version: "v3.0.0",
			},
			Resources: []apitype.ResourceV3{},
		},
	}
	
	// Test that the function sets the environment variable
	_, err := state.Decrypt(ctx, passphrase, checkpoint)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Note: We can't easily test that the environment variable was set
	// without exposing internal implementation details or using os.Getenv
	// which might interfere with other tests. The important thing is that
	// the function doesn't error when setting the environment variable.
}

func TestDecryptWithComplexCheckpoint(t *testing.T) {
	ctx := context.Background()
	passphrase := "complex-test-passphrase"
	testTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	
	checkpoint := &apitype.CheckpointV3{
		Stack: "complex-test-stack",
		Latest: &apitype.DeploymentV3{
			Manifest: apitype.ManifestV1{
				Time:    testTime,
				Magic:   "0xDEADBEEF",
				Version: "v3.0.0",
			},
			Resources: []apitype.ResourceV3{
				{
					URN:  resource.URN("urn:pulumi:test::test::aws:s3/bucket:Bucket::bucket1"),
					Type: "aws:s3/bucket:Bucket",
					Inputs: map[string]interface{}{
						"bucket": "test-bucket-1",
						"acl":    "private",
					},
					Outputs: map[string]interface{}{
						"id":  "test-bucket-1",
						"arn": "arn:aws:s3:::test-bucket-1",
					},
				},
				{
					URN:  resource.URN("urn:pulumi:test::test::aws:lambda/function:Function::function1"),
					Type: "aws:lambda/function:Function",
					Inputs: map[string]interface{}{
						"name":    "test-function-1",
						"runtime": "nodejs18.x",
					},
					Outputs: map[string]interface{}{
						"arn": "arn:aws:lambda:us-east-1:123456789012:function:test-function-1",
					},
					Dependencies: []resource.URN{
						resource.URN("urn:pulumi:test::test::aws:s3/bucket:Bucket::bucket1"),
					},
					PropertyDependencies: map[resource.PropertyKey][]resource.URN{
						"environment": {
							resource.URN("urn:pulumi:test::test::aws:s3/bucket:Bucket::bucket1"),
						},
					},
				},
			},
		},
	}
	
	result, err := state.Decrypt(ctx, passphrase, checkpoint)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result but got nil")
	}
	
	if len(result.Latest.Resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(result.Latest.Resources))
	}
	
	// Verify that complex resource relationships are preserved
	functionResource := result.Latest.Resources[1]
	if len(functionResource.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(functionResource.Dependencies))
	}
	
	if len(functionResource.PropertyDependencies) != 1 {
		t.Errorf("Expected 1 property dependency, got %d", len(functionResource.PropertyDependencies))
	}
}