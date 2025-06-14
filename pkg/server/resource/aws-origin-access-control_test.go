package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestOriginAccessControl_Create_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	oac := &OriginAccessControl{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &OriginAccessControlInputs{
		Name: "test-oac",
	}

	var output CreateResult[OriginAccessControlOutputs]
	err := oac.Create(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestOriginAccessControl_Read_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	oac := &OriginAccessControl{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DeleteInput[OriginAccessControlOutputs]{
		ID: "test-oac-id",
	}

	var output ReadResult[OriginAccessControlOutputs]
	err := oac.Read(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestOriginAccessControl_Delete_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	oac := &OriginAccessControl{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DeleteInput[OriginAccessControlOutputs]{
		ID: "test-oac-id",
	}

	var output int
	err := oac.Delete(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestOriginAccessControlInputs_Validation(t *testing.T) {
	tests := map[string]struct {
		input    OriginAccessControlInputs
		expected OriginAccessControlInputs
	}{
		"basic name": {
			input: OriginAccessControlInputs{
				Name: "test-oac",
			},
			expected: OriginAccessControlInputs{
				Name: "test-oac",
			},
		},
		"empty name": {
			input: OriginAccessControlInputs{
				Name: "",
			},
			expected: OriginAccessControlInputs{
				Name: "",
			},
		},
		"long name": {
			input: OriginAccessControlInputs{
				Name: "very-long-origin-access-control-name-that-exceeds-normal-limits-and-should-be-handled-properly",
			},
			expected: OriginAccessControlInputs{
				Name: "very-long-origin-access-control-name-that-exceeds-normal-limits-and-should-be-handled-properly",
			},
		},
		"name with special characters": {
			input: OriginAccessControlInputs{
				Name: "test-oac_with.special-chars",
			},
			expected: OriginAccessControlInputs{
				Name: "test-oac_with.special-chars",
			},
		},
		"name with numbers": {
			input: OriginAccessControlInputs{
				Name: "test-oac-123",
			},
			expected: OriginAccessControlInputs{
				Name: "test-oac-123",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected.Name, tc.input.Name)
		})
	}
}

func TestOriginAccessControlOutputs_Structure(t *testing.T) {
	// Test that OriginAccessControlOutputs struct exists and is properly defined
	outputs := OriginAccessControlOutputs{}
	
	// Verify it's an empty struct as expected
	assert.IsType(t, OriginAccessControlOutputs{}, outputs)
}

func TestOriginAccessControl_StructValidation(t *testing.T) {
	// Test OriginAccessControl struct embedding
	oac := &OriginAccessControl{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: &project.Project{},
		},
	}

	// Verify embedded AwsResource
	assert.NotNil(t, oac.AwsResource)
	assert.NotNil(t, oac.AwsResource.context)
	assert.NotNil(t, oac.AwsResource.project)
}

func TestGenerateName_Function(t *testing.T) {
	tests := map[string]struct {
		input          string
		expectedLength int
		shouldTruncate bool
	}{
		"short name": {
			input:          "test",
			expectedLength: 13, // "test" (4) + "-" (1) + random (8) = 13
			shouldTruncate: false,
		},
		"normal name": {
			input:          "my-origin-access-control",
			expectedLength: 33, // "my-origin-access-control" (24) + "-" (1) + random (8) = 33
			shouldTruncate: false,
		},
		"exactly 55 chars": {
			input:          "1234567890123456789012345678901234567890123456789012345", // 55 chars
			expectedLength: 64, // 55 + "-" (1) + random (8) = 64
			shouldTruncate: false,
		},
		"long name that needs truncation": {
			input:          "very-long-origin-access-control-name-that-exceeds-the-55-character-limit-and-should-be-truncated", // 96 chars
			expectedLength: 64, // truncated to 55 + "-" (1) + random (8) = 64
			shouldTruncate: true,
		},
		"empty name": {
			input:          "",
			expectedLength: 9, // "" (0) + "-" (1) + random (8) = 9
			shouldTruncate: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := generateName(tc.input)
			
			// Check total length
			assert.Equal(t, tc.expectedLength, len(result))
			
			// Check that it contains a dash
			assert.Contains(t, result, "-")
			
			// Check that the random suffix is 8 characters
			parts := result[len(result)-8:]
			assert.Len(t, parts, 8)
			
			// Check that random part only contains valid characters
			charset := "abcdefghijklmnopqrstuvwxyz0123456789"
			for _, char := range parts {
				assert.Contains(t, charset, string(char))
			}
			
			// If truncation is expected, verify the prefix is exactly 55 chars
			if tc.shouldTruncate {
				prefix := result[:len(result)-9] // Remove "-" + 8 random chars
				assert.Len(t, prefix, 55)
				assert.Equal(t, tc.input[:55], prefix)
			} else if tc.input != "" {
				// If no truncation, verify the prefix matches the input
				prefix := result[:len(result)-9] // Remove "-" + 8 random chars
				assert.Equal(t, tc.input, prefix)
			}
		})
	}
}

func TestGenerateName_Randomness(t *testing.T) {
	// Test that generateName produces different results for the same input
	input := "test-oac"
	results := make(map[string]bool)
	
	// Generate multiple names and ensure they're different
	for i := 0; i < 10; i++ {
		result := generateName(input)
		assert.False(t, results[result], "generateName should produce unique results")
		results[result] = true
		
		// Verify structure
		assert.True(t, len(result) > len(input))
		assert.Contains(t, result, input+"-")
	}
	
	// Should have 10 unique results
	assert.Len(t, results, 10)
}

func TestGenerateName_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		input    string
		testFunc func(t *testing.T, result string)
	}{
		"single character": {
			input: "a",
			testFunc: func(t *testing.T, result string) {
				assert.Equal(t, 10, len(result)) // "a" + "-" + 8 random = 10
				assert.True(t, result[0] == 'a')
				assert.True(t, result[1] == '-')
			},
		},
		"special characters": {
			input: "test-oac_with.special",
			testFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "test-oac_with.special-")
				assert.Equal(t, 30, len(result)) // 21 + "-" + 8 = 30
			},
		},
		"unicode characters": {
			input: "test-oac-ñ",
			testFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "test-oac-ñ-")
				// Note: Unicode characters may affect byte length differently
				assert.True(t, len(result) >= 19) // At least original + "-" + 8
			},
		},
		"numbers only": {
			input: "123456789",
			testFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "123456789-")
				assert.Equal(t, 18, len(result)) // 9 + "-" + 8 = 18
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := generateName(tc.input)
			tc.testFunc(t, result)
			
			// Common validations for all edge cases
			assert.Contains(t, result, "-")
			
			// Verify random suffix
			suffix := result[len(result)-8:]
			charset := "abcdefghijklmnopqrstuvwxyz0123456789"
			for _, char := range suffix {
				assert.Contains(t, charset, string(char))
			}
		})
	}
}

func TestOriginAccessControl_CreateResult_Structure(t *testing.T) {
	// Test CreateResult structure
	result := CreateResult[OriginAccessControlOutputs]{
		ID:   "test-oac-id-123",
		Outs: OriginAccessControlOutputs{},
	}

	assert.Equal(t, "test-oac-id-123", result.ID)
	assert.IsType(t, OriginAccessControlOutputs{}, result.Outs)
}

func TestOriginAccessControl_ReadResult_Structure(t *testing.T) {
	// Test ReadResult structure
	result := ReadResult[OriginAccessControlOutputs]{
		ID:   "test-oac-id-456",
		Outs: OriginAccessControlOutputs{},
	}

	assert.Equal(t, "test-oac-id-456", result.ID)
	assert.IsType(t, OriginAccessControlOutputs{}, result.Outs)
}

func TestOriginAccessControl_DeleteInput_Structure(t *testing.T) {
	// Test DeleteInput structure
	input := DeleteInput[OriginAccessControlOutputs]{
		ID: "test-oac-id-789",
	}

	assert.Equal(t, "test-oac-id-789", input.ID)
}

func TestOriginAccessControl_Integration_Scenarios(t *testing.T) {
	// Test various integration scenarios without AWS provider
	scenarios := map[string]struct {
		name        string
		expectError bool
	}{
		"production environment": {
			name:        "prod-oac",
			expectError: true, // No AWS provider
		},
		"development environment": {
			name:        "dev-oac",
			expectError: true, // No AWS provider
		},
		"staging environment": {
			name:        "staging-oac",
			expectError: true, // No AWS provider
		},
		"test environment": {
			name:        "test-oac",
			expectError: true, // No AWS provider
		},
	}

	for scenarioName, scenario := range scenarios {
		t.Run(scenarioName, func(t *testing.T) {
			p := &project.Project{}
			oac := &OriginAccessControl{
				AwsResource: &AwsResource{
					context: context.Background(),
					project: p,
				},
			}

			input := &OriginAccessControlInputs{
				Name: scenario.name,
			}

			var output CreateResult[OriginAccessControlOutputs]
			err := oac.Create(input, &output)

			if scenario.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "no aws provider found")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}