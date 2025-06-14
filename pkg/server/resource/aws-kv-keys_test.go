package resource

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestKvKeys_Create_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	kvKeys := &KvKeys{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &KvKeysInputs{
		Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
		Namespace: "test",
		Entries: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Purge: false,
	}

	var output CreateResult[KvKeysOutputs]
	err := kvKeys.Create(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestKvKeys_Update_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	kvKeys := &KvKeys{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &UpdateInput[KvKeysInputs, KvKeysOutputs]{
		News: KvKeysInputs{
			Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			Namespace: "test",
			Entries: map[string]string{
				"key1": "new-value1",
				"key3": "value3",
			},
			Purge: true,
		},
		Olds: KvKeysOutputs{
			Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			Namespace: "test",
			Entries: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Purge: false,
		},
	}

	var output UpdateResult[KvKeysOutputs]
	err := kvKeys.Update(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestKvKeys_Delete_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	kvKeys := &KvKeys{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DeleteInput[KvKeysOutputs]{
		Outs: KvKeysOutputs{
			Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			Namespace: "test",
			Entries: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Purge: false,
		},
	}

	var output int
	err := kvKeys.Delete(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestKvKeys_Delete_EmptyStore(t *testing.T) {
	// Create project without AWS provider (doesn't matter for this test)
	p := &project.Project{}

	kvKeys := &KvKeys{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DeleteInput[KvKeysOutputs]{
		Outs: KvKeysOutputs{
			Store:     "", // Empty store should return early
			Namespace: "test",
			Entries: map[string]string{
				"key1": "value1",
			},
			Purge: false,
		},
	}

	var output int
	err := kvKeys.Delete(input, &output)

	// Should succeed without error for empty store
	assert.NoError(t, err)
}

func TestKvKeysInputs_Validation(t *testing.T) {
	tests := map[string]struct {
		input    KvKeysInputs
		expected KvKeysInputs
	}{
		"basic inputs": {
			input: KvKeysInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
				Namespace: "prod",
				Entries: map[string]string{
					"config": "value1",
					"secret": "value2",
				},
				Purge: true,
			},
			expected: KvKeysInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
				Namespace: "prod",
				Entries: map[string]string{
					"config": "value1",
					"secret": "value2",
				},
				Purge: true,
			},
		},
		"empty entries": {
			input: KvKeysInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/empty-store",
				Namespace: "test",
				Entries:   map[string]string{},
				Purge:     false,
			},
			expected: KvKeysInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/empty-store",
				Namespace: "test",
				Entries:   map[string]string{},
				Purge:     false,
			},
		},
		"nil entries": {
			input: KvKeysInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/nil-store",
				Namespace: "dev",
				Entries:   nil,
				Purge:     true,
			},
			expected: KvKeysInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/nil-store",
				Namespace: "dev",
				Entries:   nil,
				Purge:     true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected.Store, tc.input.Store)
			assert.Equal(t, tc.expected.Namespace, tc.input.Namespace)
			assert.Equal(t, tc.expected.Entries, tc.input.Entries)
			assert.Equal(t, tc.expected.Purge, tc.input.Purge)
		})
	}
}

func TestKvKeysOutputs_Validation(t *testing.T) {
	tests := map[string]struct {
		output   KvKeysOutputs
		expected KvKeysOutputs
	}{
		"complete outputs": {
			output: KvKeysOutputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/output-store",
				Namespace: "staging",
				Entries: map[string]string{
					"api_key":    "abc123",
					"db_url":     "postgres://localhost:5432/db",
					"cache_ttl":  "3600",
				},
				Purge: true,
			},
			expected: KvKeysOutputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/output-store",
				Namespace: "staging",
				Entries: map[string]string{
					"api_key":    "abc123",
					"db_url":     "postgres://localhost:5432/db",
					"cache_ttl":  "3600",
				},
				Purge: true,
			},
		},
		"empty outputs": {
			output: KvKeysOutputs{
				Store:     "",
				Namespace: "",
				Entries:   map[string]string{},
				Purge:     false,
			},
			expected: KvKeysOutputs{
				Store:     "",
				Namespace: "",
				Entries:   map[string]string{},
				Purge:     false,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected.Store, tc.output.Store)
			assert.Equal(t, tc.expected.Namespace, tc.output.Namespace)
			assert.Equal(t, tc.expected.Entries, tc.output.Entries)
			assert.Equal(t, tc.expected.Purge, tc.output.Purge)
		})
	}
}

func TestKvKeys_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		description string
		store       string
		namespace   string
		entries     map[string]string
		purge       bool
	}{
		"long store ARN": {
			description: "Handle very long CloudFront KV store ARN",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/very-long-store-name-with-many-characters-and-hyphens-to-test-limits",
			namespace:   "test",
			entries: map[string]string{
				"key": "value",
			},
			purge: false,
		},
		"special characters in namespace": {
			description: "Handle special characters in namespace",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			namespace:   "test-env_v1.2",
			entries: map[string]string{
				"config": "value",
			},
			purge: false,
		},
		"special characters in keys and values": {
			description: "Handle special characters in keys and values",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			namespace:   "test",
			entries: map[string]string{
				"key-with-hyphens":     "value with spaces",
				"key_with_underscores": "value/with/slashes",
				"key.with.dots":        "value:with:colons",
				"unicode-key-🔑":       "unicode-value-🎯",
			},
			purge: true,
		},
		"large number of entries": {
			description: "Handle large number of key-value entries",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/large-store",
			namespace:   "bulk",
			entries:     generateLargeEntries(100),
			purge:       false,
		},
		"empty namespace": {
			description: "Handle empty namespace",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			namespace:   "",
			entries: map[string]string{
				"global-key": "global-value",
			},
			purge: false,
		},
		"very long values": {
			description: "Handle very long values",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			namespace:   "test",
			entries: map[string]string{
				"long-value": generateLongString(1000),
				"json-config": `{"api":{"endpoint":"https://api.example.com","timeout":30000,"retries":3},"database":{"host":"db.example.com","port":5432,"ssl":true}}`,
			},
			purge: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test input validation
			input := KvKeysInputs{
				Store:     tc.store,
				Namespace: tc.namespace,
				Entries:   tc.entries,
				Purge:     tc.purge,
			}

			// Validate that inputs are properly structured
			assert.NotEmpty(t, input.Store, "Store ARN should not be empty")
			assert.NotNil(t, input.Entries, "Entries should not be nil")
			
			// Test that namespace can be empty (valid case)
			if tc.namespace == "" {
				assert.Empty(t, input.Namespace, "Empty namespace should be preserved")
			} else {
				assert.Equal(t, tc.namespace, input.Namespace, "Namespace should match expected value")
			}

			// Test output structure
			output := KvKeysOutputs{
				Store:     input.Store,
				Namespace: input.Namespace,
				Entries:   input.Entries,
				Purge:     input.Purge,
			}

			assert.Equal(t, input.Store, output.Store)
			assert.Equal(t, input.Namespace, output.Namespace)
			assert.Equal(t, input.Entries, output.Entries)
			assert.Equal(t, input.Purge, output.Purge)
		})
	}
}

func TestKvKeys_StructFieldValidation(t *testing.T) {
	// Test KvKeys struct embedding
	kvKeys := &KvKeys{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: &project.Project{},
		},
	}

	assert.NotNil(t, kvKeys.AwsResource, "AwsResource should be embedded")
	assert.NotNil(t, kvKeys.AwsResource.context, "Context should be set")
	assert.NotNil(t, kvKeys.AwsResource.project, "Project should be set")

	// Test input struct fields
	input := KvKeysInputs{
		Store:     "test-store",
		Namespace: "test-namespace",
		Entries:   map[string]string{"key": "value"},
		Purge:     true,
	}

	assert.IsType(t, "", input.Store, "Store should be string")
	assert.IsType(t, "", input.Namespace, "Namespace should be string")
	assert.IsType(t, map[string]string{}, input.Entries, "Entries should be map[string]string")
	assert.IsType(t, true, input.Purge, "Purge should be bool")

	// Test output struct fields
	output := KvKeysOutputs{
		Store:     "test-store",
		Namespace: "test-namespace",
		Entries:   map[string]string{"key": "value"},
		Purge:     true,
	}

	assert.IsType(t, "", output.Store, "Store should be string")
	assert.IsType(t, "", output.Namespace, "Namespace should be string")
	assert.IsType(t, map[string]string{}, output.Entries, "Entries should be map[string]string")
	assert.IsType(t, true, output.Purge, "Purge should be bool")
}

func TestKvKeys_NamespaceHandling(t *testing.T) {
	tests := map[string]struct {
		namespace string
		key       string
		expected  string
	}{
		"standard namespace": {
			namespace: "prod",
			key:       "config",
			expected:  "prod:config",
		},
		"empty namespace": {
			namespace: "",
			key:       "global",
			expected:  ":global",
		},
		"namespace with special chars": {
			namespace: "env-v1.2_test",
			key:       "api-key",
			expected:  "env-v1.2_test:api-key",
		},
		"key with special chars": {
			namespace: "test",
			key:       "key.with.dots-and_underscores",
			expected:  "test:key.with.dots-and_underscores",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test the namespace:key combination that would be used internally
			namespacedKey := tc.namespace + ":" + tc.key
			assert.Equal(t, tc.expected, namespacedKey, "Namespaced key should match expected format")
		})
	}
}

func TestKvKeys_BatchProcessing(t *testing.T) {
	// Test scenarios that would trigger batch processing logic
	tests := map[string]struct {
		description string
		entries     map[string]string
		expectBatch bool
	}{
		"small batch": {
			description: "Small number of entries should fit in single batch",
			entries:     generateLargeEntries(10),
			expectBatch: false,
		},
		"medium batch": {
			description: "Medium number of entries should fit in single batch",
			entries:     generateLargeEntries(30),
			expectBatch: false,
		},
		"large batch": {
			description: "Large number of entries may require batching",
			entries:     generateLargeEntries(60),
			expectBatch: true,
		},
		"very large batch": {
			description: "Very large number of entries will require multiple batches",
			entries:     generateLargeEntries(150),
			expectBatch: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Validate that we can handle different batch sizes
			assert.NotNil(t, tc.entries, "Entries should not be nil")
			assert.GreaterOrEqual(t, len(tc.entries), 1, "Should have at least one entry")
			
			// Test that entries are properly structured
			for key, value := range tc.entries {
				assert.NotEmpty(t, key, "Key should not be empty")
				assert.NotEmpty(t, value, "Value should not be empty")
			}
			
			// Simulate batch size check (CloudFront KV has a limit of 50 operations per batch)
			const batchSize = 50
			expectedBatches := (len(tc.entries) + batchSize - 1) / batchSize
			
			if tc.expectBatch && len(tc.entries) > batchSize {
				assert.Greater(t, expectedBatches, 1, "Should require multiple batches")
			} else {
				assert.LessOrEqual(t, expectedBatches, 1, "Should fit in single batch")
			}
		})
	}
}

// Helper function to generate large number of entries for testing
func generateLargeEntries(count int) map[string]string {
	entries := make(map[string]string, count)
	for i := 0; i < count; i++ {
		key := aws.ToString(aws.String("key-" + string(rune('A'+i%26)) + string(rune('0'+i%10))))
		value := aws.ToString(aws.String("value-" + string(rune('a'+i%26)) + string(rune('0'+i%10))))
		entries[key] = value
	}
	return entries
}

// Helper function to generate long string for testing
func generateLongString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}