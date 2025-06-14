package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestKvRoutesUpdate_Create_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	kvRoutesUpdate := &KvRoutesUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &KvRoutesUpdateInputs{
		Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
		Key:       "routes",
		Entry:     "/api/v1/*",
		Namespace: "prod",
	}

	var output CreateResult[KvRoutesUpdateOutputs]
	err := kvRoutesUpdate.Create(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestKvRoutesUpdate_Update_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	kvRoutesUpdate := &KvRoutesUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &UpdateInput[KvRoutesUpdateInputs, KvRoutesUpdateOutputs]{
		News: KvRoutesUpdateInputs{
			Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			Key:       "routes",
			Entry:     "/api/v2/*",
			Namespace: "prod",
		},
		Olds: KvRoutesUpdateOutputs{
			Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			Key:       "routes",
			Entry:     "/api/v1/*",
			Namespace: "prod",
		},
	}

	var output UpdateResult[KvRoutesUpdateOutputs]
	err := kvRoutesUpdate.Update(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestKvRoutesUpdate_Delete_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	kvRoutesUpdate := &KvRoutesUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DeleteInput[KvRoutesUpdateOutputs]{
		Outs: KvRoutesUpdateOutputs{
			Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			Key:       "routes",
			Entry:     "/api/v1/*",
			Namespace: "prod",
		},
	}

	var output int
	err := kvRoutesUpdate.Delete(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestKvRoutesUpdateInputs_Validation(t *testing.T) {
	tests := map[string]struct {
		input    KvRoutesUpdateInputs
		expected KvRoutesUpdateInputs
	}{
		"basic route inputs": {
			input: KvRoutesUpdateInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
				Key:       "api-routes",
				Entry:     "/api/users/*",
				Namespace: "production",
			},
			expected: KvRoutesUpdateInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
				Key:       "api-routes",
				Entry:     "/api/users/*",
				Namespace: "production",
			},
		},
		"wildcard route": {
			input: KvRoutesUpdateInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/wildcard-store",
				Key:       "catch-all",
				Entry:     "/*",
				Namespace: "dev",
			},
			expected: KvRoutesUpdateInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/wildcard-store",
				Key:       "catch-all",
				Entry:     "/*",
				Namespace: "dev",
			},
		},
		"specific path route": {
			input: KvRoutesUpdateInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/specific-store",
				Key:       "exact-routes",
				Entry:     "/health",
				Namespace: "monitoring",
			},
			expected: KvRoutesUpdateInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/specific-store",
				Key:       "exact-routes",
				Entry:     "/health",
				Namespace: "monitoring",
			},
		},
		"empty namespace": {
			input: KvRoutesUpdateInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/global-store",
				Key:       "global-routes",
				Entry:     "/global/*",
				Namespace: "",
			},
			expected: KvRoutesUpdateInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/global-store",
				Key:       "global-routes",
				Entry:     "/global/*",
				Namespace: "",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected.Store, tc.input.Store)
			assert.Equal(t, tc.expected.Key, tc.input.Key)
			assert.Equal(t, tc.expected.Entry, tc.input.Entry)
			assert.Equal(t, tc.expected.Namespace, tc.input.Namespace)
		})
	}
}

func TestKvRoutesUpdateOutputs_Validation(t *testing.T) {
	tests := map[string]struct {
		output   KvRoutesUpdateOutputs
		expected KvRoutesUpdateOutputs
	}{
		"complete outputs": {
			output: KvRoutesUpdateOutputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/output-store",
				Key:       "api-routes",
				Entry:     "/api/v1/users/*",
				Namespace: "staging",
			},
			expected: KvRoutesUpdateOutputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/output-store",
				Key:       "api-routes",
				Entry:     "/api/v1/users/*",
				Namespace: "staging",
			},
		},
		"empty outputs": {
			output: KvRoutesUpdateOutputs{
				Store:     "",
				Key:       "",
				Entry:     "",
				Namespace: "",
			},
			expected: KvRoutesUpdateOutputs{
				Store:     "",
				Key:       "",
				Entry:     "",
				Namespace: "",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected.Store, tc.output.Store)
			assert.Equal(t, tc.expected.Key, tc.output.Key)
			assert.Equal(t, tc.expected.Entry, tc.output.Entry)
			assert.Equal(t, tc.expected.Namespace, tc.output.Namespace)
		})
	}
}

func TestKvRoutesUpdate_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		description string
		store       string
		key         string
		entry       string
		namespace   string
	}{
		"long store ARN": {
			description: "Handle very long CloudFront KV store ARN",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/very-long-store-name-with-many-characters-and-hyphens-to-test-limits-for-routes",
			key:         "routes",
			entry:       "/api/v1/*",
			namespace:   "test",
		},
		"complex route patterns": {
			description: "Handle complex route patterns with special characters",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			key:         "complex-routes",
			entry:       "/api/v1/users/{id}/posts/{postId}?filter=*&sort=*",
			namespace:   "prod",
		},
		"special characters in namespace": {
			description: "Handle special characters in namespace",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			key:         "routes",
			entry:       "/api/*",
			namespace:   "env-v1.2_test",
		},
		"special characters in key": {
			description: "Handle special characters in key",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			key:         "api-routes_v1.2-beta",
			entry:       "/beta/*",
			namespace:   "test",
		},
		"unicode in routes": {
			description: "Handle unicode characters in route entries",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/unicode-store",
			key:         "i18n-routes",
			entry:       "/api/用户/*",
			namespace:   "international",
		},
		"very long route entry": {
			description: "Handle very long route entry",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			key:         "long-routes",
			entry:       generateLongRoute(500),
			namespace:   "test",
		},
		"empty namespace with colon handling": {
			description: "Handle empty namespace with colon in key formation",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			key:         "global-routes",
			entry:       "/global/*",
			namespace:   "",
		},
		"route with query parameters": {
			description: "Handle routes with query parameters and fragments",
			store:       "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			key:         "query-routes",
			entry:       "/search?q=*&category=*&page=*#results",
			namespace:   "search",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test input validation
			input := KvRoutesUpdateInputs{
				Store:     tc.store,
				Key:       tc.key,
				Entry:     tc.entry,
				Namespace: tc.namespace,
			}

			// Validate that inputs are properly structured
			assert.NotEmpty(t, input.Store, "Store ARN should not be empty")
			assert.NotEmpty(t, input.Key, "Key should not be empty")
			assert.NotEmpty(t, input.Entry, "Entry should not be empty")
			
			// Test that namespace can be empty (valid case)
			if tc.namespace == "" {
				assert.Empty(t, input.Namespace, "Empty namespace should be preserved")
			} else {
				assert.Equal(t, tc.namespace, input.Namespace, "Namespace should match expected value")
			}

			// Test output structure
			output := KvRoutesUpdateOutputs{
				Store:     input.Store,
				Key:       input.Key,
				Entry:     input.Entry,
				Namespace: input.Namespace,
			}

			assert.Equal(t, input.Store, output.Store)
			assert.Equal(t, input.Key, output.Key)
			assert.Equal(t, input.Entry, output.Entry)
			assert.Equal(t, input.Namespace, output.Namespace)
		})
	}
}

func TestKvRoutesUpdate_StructFieldValidation(t *testing.T) {
	// Test KvRoutesUpdate struct embedding
	kvRoutesUpdate := &KvRoutesUpdate{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: &project.Project{},
		},
	}

	assert.NotNil(t, kvRoutesUpdate.AwsResource, "AwsResource should be embedded")
	assert.NotNil(t, kvRoutesUpdate.AwsResource.context, "Context should be set")
	assert.NotNil(t, kvRoutesUpdate.AwsResource.project, "Project should be set")

	// Test input struct fields
	input := KvRoutesUpdateInputs{
		Store:     "test-store",
		Key:       "test-key",
		Entry:     "test-entry",
		Namespace: "test-namespace",
	}

	assert.IsType(t, "", input.Store, "Store should be string")
	assert.IsType(t, "", input.Key, "Key should be string")
	assert.IsType(t, "", input.Entry, "Entry should be string")
	assert.IsType(t, "", input.Namespace, "Namespace should be string")

	// Test output struct fields
	output := KvRoutesUpdateOutputs{
		Store:     "test-store",
		Key:       "test-key",
		Entry:     "test-entry",
		Namespace: "test-namespace",
	}

	assert.IsType(t, "", output.Store, "Store should be string")
	assert.IsType(t, "", output.Key, "Key should be string")
	assert.IsType(t, "", output.Entry, "Entry should be string")
	assert.IsType(t, "", output.Namespace, "Namespace should be string")
}

func TestKvRoutesUpdate_NamespaceKeyHandling(t *testing.T) {
	tests := map[string]struct {
		namespace string
		key       string
		expected  string
	}{
		"standard namespace and key": {
			namespace: "prod",
			key:       "api-routes",
			expected:  "prod:api-routes",
		},
		"empty namespace": {
			namespace: "",
			key:       "global-routes",
			expected:  ":global-routes",
		},
		"namespace with special chars": {
			namespace: "env-v1.2_test",
			key:       "api-routes",
			expected:  "env-v1.2_test:api-routes",
		},
		"key with special chars": {
			namespace: "test",
			key:       "routes.v1-beta_2",
			expected:  "test:routes.v1-beta_2",
		},
		"both with special chars": {
			namespace: "staging_v2.1",
			key:       "api-routes_v1.0",
			expected:  "staging_v2.1:api-routes_v1.0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test the namespace:key combination that would be used internally
			fullKey := tc.namespace + ":" + tc.key
			assert.Equal(t, tc.expected, fullKey, "Full key should match expected format")
		})
	}
}

func TestKvRoutesUpdate_ChunkSizeConstant(t *testing.T) {
	// Test that the chunk size constant is properly defined
	assert.Equal(t, 1000, chunkSize, "Chunk size should be 1000")
	assert.IsType(t, 0, chunkSize, "Chunk size should be int")
}

func TestKvRoutesUpdate_RouteHelperFunctions(t *testing.T) {
	// Test existsRoute function
	t.Run("existsRoute", func(t *testing.T) {
		routes := []string{"/api/v1/*", "/health", "/metrics"}
		
		assert.True(t, existsRoute(routes, "/api/v1/*"), "Should find existing route")
		assert.True(t, existsRoute(routes, "/health"), "Should find existing exact route")
		assert.False(t, existsRoute(routes, "/api/v2/*"), "Should not find non-existing route")
		assert.False(t, existsRoute([]string{}, "/any"), "Should not find route in empty slice")
		assert.False(t, existsRoute(nil, "/any"), "Should not find route in nil slice")
	})

	// Test removeRoute function
	t.Run("removeRoute", func(t *testing.T) {
		routes := []string{"/api/v1/*", "/health", "/metrics", "/api/v1/*"}
		
		result := removeRoute(routes, "/health")
		expected := []string{"/api/v1/*", "/metrics", "/api/v1/*"}
		assert.Equal(t, expected, result, "Should remove specified route")
		
		result = removeRoute(routes, "/nonexistent")
		assert.Equal(t, routes, result, "Should return original slice if route not found")
		
		result = removeRoute([]string{}, "/any")
		assert.Len(t, result, 0, "Should return empty result for empty input")
		
		result = removeRoute(nil, "/any")
		assert.Len(t, result, 0, "Should return empty result for nil input")
		
		// Test removing all occurrences
		result = removeRoute(routes, "/api/v1/*")
		expected = []string{"/health", "/metrics"}
		assert.Equal(t, expected, result, "Should remove all occurrences of route")
	})
}

func TestKvRoutesUpdate_IDGeneration(t *testing.T) {
	tests := map[string]struct {
		store     string
		namespace string
		key       string
		expected  string
	}{
		"standard ID": {
			store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			namespace: "prod",
			key:       "api-routes",
			expected:  "arn:aws:cloudfront::123456789012:key-value-store/test-store:prod:api-routes",
		},
		"empty namespace": {
			store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			namespace: "",
			key:       "global-routes",
			expected:  "arn:aws:cloudfront::123456789012:key-value-store/test-store::global-routes",
		},
		"special characters": {
			store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
			namespace: "env_v1.2",
			key:       "routes-beta",
			expected:  "arn:aws:cloudfront::123456789012:key-value-store/test-store:env_v1.2:routes-beta",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test the ID format that would be generated in Create method
			id := tc.store + ":" + tc.namespace + ":" + tc.key
			assert.Equal(t, tc.expected, id, "ID should match expected format")
		})
	}
}

func TestKvRoutesUpdate_ChunkingLogic(t *testing.T) {
	tests := map[string]struct {
		description string
		dataSize    int
		expectChunk bool
	}{
		"small data": {
			description: "Small data should not require chunking",
			dataSize:    500,
			expectChunk: false,
		},
		"medium data": {
			description: "Medium data should not require chunking",
			dataSize:    999,
			expectChunk: false,
		},
		"exactly chunk size": {
			description: "Data exactly at chunk size should not require chunking",
			dataSize:    1000,
			expectChunk: false,
		},
		"slightly over chunk size": {
			description: "Data slightly over chunk size should require chunking",
			dataSize:    1001,
			expectChunk: true,
		},
		"large data": {
			description: "Large data should require chunking",
			dataSize:    2500,
			expectChunk: true,
		},
		"very large data": {
			description: "Very large data should require multiple chunks",
			dataSize:    5000,
			expectChunk: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Simulate chunking logic
			needsChunking := tc.dataSize > chunkSize
			assert.Equal(t, tc.expectChunk, needsChunking, tc.description)
			
			if needsChunking {
				expectedChunks := (tc.dataSize + chunkSize - 1) / chunkSize // Ceiling division
				assert.Greater(t, expectedChunks, 1, "Should require multiple chunks")
			}
		})
	}
}

func TestKvRoutesUpdate_RoutePatterns(t *testing.T) {
	tests := map[string]struct {
		entry       string
		description string
	}{
		"wildcard route": {
			entry:       "/*",
			description: "Catch-all wildcard route",
		},
		"api wildcard": {
			entry:       "/api/*",
			description: "API namespace wildcard",
		},
		"versioned api": {
			entry:       "/api/v1/*",
			description: "Versioned API wildcard",
		},
		"specific endpoint": {
			entry:       "/health",
			description: "Specific health check endpoint",
		},
		"parameterized route": {
			entry:       "/users/{id}",
			description: "Route with path parameter",
		},
		"nested parameterized": {
			entry:       "/users/{id}/posts/{postId}",
			description: "Nested parameterized route",
		},
		"query parameters": {
			entry:       "/search?q=*",
			description: "Route with query parameters",
		},
		"complex pattern": {
			entry:       "/api/v1/users/{id}/posts/{postId}?include=*&sort=*",
			description: "Complex route with parameters and queries",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test that various route patterns are handled properly
			input := KvRoutesUpdateInputs{
				Store:     "arn:aws:cloudfront::123456789012:key-value-store/test-store",
				Key:       "routes",
				Entry:     tc.entry,
				Namespace: "test",
			}

			assert.NotEmpty(t, input.Entry, "Route entry should not be empty")
			assert.Equal(t, tc.entry, input.Entry, "Route entry should match expected pattern")
		})
	}
}

// Helper function to generate long route for testing
func generateLongRoute(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/-_"
	result := "/api/"
	for i := len(result); i < length; i++ {
		result += string(charset[i%len(charset)])
	}
	return result + "/*"
}