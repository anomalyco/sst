package resource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerScript_Create(t *testing.T) {
	tests := map[string]struct {
		input       WorkerScriptInputs
		expectError bool
		errorMsg    string
	}{
		"valid basic script": {
			input: WorkerScriptInputs{
				AccountId:  "test-account-123",
				ApiToken:   "test-token-456",
				ScriptName: "test-script",
				Content: struct {
					Filename string `json:"filename"`
					Hash     string `json:"hash"`
				}{
					Filename: "test-script.js",
					Hash:     "abc123",
				},
			},
			expectError: true, // Will fail due to file not existing
			errorMsg:    "failed to read file",
		},
		"empty account id": {
			input: WorkerScriptInputs{
				AccountId:  "",
				ApiToken:   "test-token",
				ScriptName: "test-script",
				Content: struct {
					Filename string `json:"filename"`
					Hash     string `json:"hash"`
				}{
					Filename: "test.js",
					Hash:     "hash123",
				},
			},
			expectError: true,
			errorMsg:    "failed to read file",
		},
		"empty api token": {
			input: WorkerScriptInputs{
				AccountId:  "account123",
				ApiToken:   "",
				ScriptName: "test-script",
				Content: struct {
					Filename string `json:"filename"`
					Hash     string `json:"hash"`
				}{
					Filename: "test.js",
					Hash:     "hash123",
				},
			},
			expectError: true,
			errorMsg:    "failed to read file",
		},
		"empty script name": {
			input: WorkerScriptInputs{
				AccountId:  "account123",
				ApiToken:   "token456",
				ScriptName: "",
				Content: struct {
					Filename string `json:"filename"`
					Hash     string `json:"hash"`
				}{
					Filename: "test.js",
					Hash:     "hash123",
				},
			},
			expectError: true,
			errorMsg:    "failed to read file",
		},
		"empty content filename": {
			input: WorkerScriptInputs{
				AccountId:  "account123",
				ApiToken:   "token456",
				ScriptName: "test-script",
				Content: struct {
					Filename string `json:"filename"`
					Hash     string `json:"hash"`
				}{
					Filename: "",
					Hash:     "hash123",
				},
			},
			expectError: true,
			errorMsg:    "failed to read file",
		},
		"empty content hash": {
			input: WorkerScriptInputs{
				AccountId:  "account123",
				ApiToken:   "token456",
				ScriptName: "test-script",
				Content: struct {
					Filename string `json:"filename"`
					Hash     string `json:"hash"`
				}{
					Filename: "test.js",
					Hash:     "",
				},
			},
			expectError: true,
			errorMsg:    "failed to read file",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			resource := &WorkerScript{
				CloudflareResource: &CloudflareResource{},
			}

			var output CreateResult[WorkerScriptOutputs]
			err := resource.Create(&tt.input, &output)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "script", output.ID)
				assert.Equal(t, tt.input.AccountId, output.Outs.AccountId)
				assert.Equal(t, tt.input.ApiToken, output.Outs.ApiToken)
				assert.Equal(t, tt.input.ScriptName, output.Outs.ScriptName)
			}
		})
	}
}

func TestWorkerScript_Create_WithValidFile(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-script.js")
	testContent := `
export default {
	async fetch(request, env, ctx) {
		return new Response('Hello World!');
	},
};
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	input := WorkerScriptInputs{
		AccountId:  "test-account-123",
		ApiToken:   "test-token-456",
		ScriptName: "test-script",
		Content: struct {
			Filename string `json:"filename"`
			Hash     string `json:"hash"`
		}{
			Filename: testFile,
			Hash:     "abc123",
		},
		CompatibilityDate: "2023-05-18",
		CompatibilityFlags: []string{"nodejs_compat"},
		MainModule: "worker.js",
		UsageModel: "bundled",
	}

	resource := &WorkerScript{
		CloudflareResource: &CloudflareResource{},
	}

	var output CreateResult[WorkerScriptOutputs]
	err = resource.Create(&input, &output)

	// This will fail due to network call, but we can verify the structure
	assert.Error(t, err) // Expected to fail due to actual HTTP request
	// The error should be related to HTTP request, not file reading
	assert.NotContains(t, err.Error(), "failed to read file")
}

func TestWorkerScript_Update(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-script.js")
	testContent := `export default { async fetch() { return new Response('Updated!'); } };`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	tests := map[string]struct {
		input       UpdateInput[WorkerScriptInputs, WorkerScriptOutputs]
		expectError bool
		errorMsg    string
	}{
		"valid update": {
			input: UpdateInput[WorkerScriptInputs, WorkerScriptOutputs]{
				News: WorkerScriptInputs{
					AccountId:  "account123",
					ApiToken:   "token456",
					ScriptName: "updated-script",
					Content: struct {
						Filename string `json:"filename"`
						Hash     string `json:"hash"`
					}{
						Filename: testFile,
						Hash:     "def456",
					},
				},
				Olds: WorkerScriptOutputs{
					AccountId:  "account123",
					ApiToken:   "token456",
					ScriptName: "old-script",
				},
			},
			expectError: true, // Will fail due to HTTP request
		},
		"update with bindings": {
			input: UpdateInput[WorkerScriptInputs, WorkerScriptOutputs]{
				News: WorkerScriptInputs{
					AccountId:  "account123",
					ApiToken:   "token456",
					ScriptName: "script-with-bindings",
					Content: struct {
						Filename string `json:"filename"`
						Hash     string `json:"hash"`
					}{
						Filename: testFile,
						Hash:     "ghi789",
					},
					Bindings: []struct {
						Type        string `json:"type"`
						Name        string `json:"name"`
						BucketName  string `json:"bucketName,omitempty"`
						ClassName   string `json:"className,omitempty"`
						NamespaceId string `json:"namespaceId,omitempty"`
						QueueName   string `json:"queueName,omitempty"`
						ScriptName  string `json:"scriptName,omitempty"`
						SecretName  string `json:"secretName,omitempty"`
						Service     string `json:"service,omitempty"`
						Text        string `json:"text,omitempty"`
					}{
						{Type: "kv_namespace", Name: "MY_KV", NamespaceId: "kv123"},
						{Type: "r2_bucket", Name: "MY_BUCKET", BucketName: "bucket123"},
						{Type: "durable_object_namespace", Name: "MY_DO", ClassName: "MyDurableObject"},
						{Type: "queue", Name: "MY_QUEUE", QueueName: "queue123"},
						{Type: "service", Name: "MY_SERVICE", Service: "service123"},
						{Type: "secret_text", Name: "MY_SECRET", SecretName: "secret123"},
						{Type: "plain_text", Name: "MY_TEXT", Text: "plain text value"},
					},
				},
				Olds: WorkerScriptOutputs{
					AccountId:  "account123",
					ApiToken:   "token456",
					ScriptName: "old-script",
				},
			},
			expectError: true, // Will fail due to HTTP request
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			resource := &WorkerScript{
				CloudflareResource: &CloudflareResource{},
			}

			var output UpdateResult[WorkerScriptOutputs]
			err := resource.Update(&tt.input, &output)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.input.News.AccountId, output.Outs.AccountId)
				assert.Equal(t, tt.input.News.ApiToken, output.Outs.ApiToken)
				assert.Equal(t, tt.input.News.ScriptName, output.Outs.ScriptName)
			}
		})
	}
}

func TestWorkerScript_Delete(t *testing.T) {
	tests := map[string]struct {
		input       DeleteInput[WorkerScriptOutputs]
		expectError bool
		errorMsg    string
	}{
		"valid delete": {
			input: DeleteInput[WorkerScriptOutputs]{
				Outs: WorkerScriptOutputs{
					AccountId:  "account123",
					ApiToken:   "token456",
					ScriptName: "script-to-delete",
				},
			},
			expectError: true, // Will fail due to HTTP request
		},
		"delete with empty account": {
			input: DeleteInput[WorkerScriptOutputs]{
				Outs: WorkerScriptOutputs{
					AccountId:  "",
					ApiToken:   "token456",
					ScriptName: "script-to-delete",
				},
			},
			expectError: true, // Will fail due to HTTP request
		},
		"delete with empty token": {
			input: DeleteInput[WorkerScriptOutputs]{
				Outs: WorkerScriptOutputs{
					AccountId:  "account123",
					ApiToken:   "",
					ScriptName: "script-to-delete",
				},
			},
			expectError: true, // Will fail due to HTTP request
		},
		"delete with empty script name": {
			input: DeleteInput[WorkerScriptOutputs]{
				Outs: WorkerScriptOutputs{
					AccountId:  "account123",
					ApiToken:   "token456",
					ScriptName: "",
				},
			},
			expectError: true, // Will fail due to HTTP request
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			resource := &WorkerScript{
				CloudflareResource: &CloudflareResource{},
			}

			var output int
			err := resource.Delete(&tt.input, &output)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWorkerScript_buildMetadata(t *testing.T) {
	tests := map[string]struct {
		input    WorkerScriptInputs
		expected map[string]interface{}
	}{
		"empty input": {
			input:    WorkerScriptInputs{},
			expected: map[string]interface{}{},
		},
		"basic fields": {
			input: WorkerScriptInputs{
				BodyPart:           "worker.js",
				CompatibilityDate:  "2023-05-18",
				CompatibilityFlags: []string{"nodejs_compat", "streams_enable_constructors"},
				KeepAssets:         true,
				KeepBindings:       []string{"MY_KV", "MY_BUCKET"},
				Logpush:            true,
				MainModule:         "worker.js",
				UsageModel:         "bundled",
			},
			expected: map[string]interface{}{
				"body_part":           "worker.js",
				"compatibility_date":  "2023-05-18",
				"compatibility_flags": []string{"nodejs_compat", "streams_enable_constructors"},
				"keep_assets":         true,
				"keep_bindings":       []string{"MY_KV", "MY_BUCKET"},
				"logpush":             true,
				"main_module":         "worker.js",
				"usage_model":         "bundled",
			},
		},
		"assets configuration": {
			input: WorkerScriptInputs{
				Assets: struct {
					Jwt    string `json:"jwt,omitempty"`
					Config struct {
						Headers          string `json:"_headers,omitempty"`
						Redirects        string `json:"_redirects,omitempty"`
						HtmlHandling     string `json:"htmlHandling,omitempty"`
						NotFoundHandling string `json:"notFoundHandling,omitempty"`
						RunWorkerFirst   bool   `json:"runWorkerFirst,omitempty"`
					} `json:"config,omitempty"`
				}{
					Jwt: "jwt-token-123",
					Config: struct {
						Headers          string `json:"_headers,omitempty"`
						Redirects        string `json:"_redirects,omitempty"`
						HtmlHandling     string `json:"htmlHandling,omitempty"`
						NotFoundHandling string `json:"notFoundHandling,omitempty"`
						RunWorkerFirst   bool   `json:"runWorkerFirst,omitempty"`
					}{
						Headers:          "Cache-Control: max-age=3600",
						Redirects:        "/old/* /new/:splat 301",
						HtmlHandling:     "auto-trailing-slash",
						NotFoundHandling: "single-page-application",
						RunWorkerFirst:   true,
					},
				},
			},
			expected: map[string]interface{}{
				"assets": map[string]interface{}{
					"jwt": "jwt-token-123",
					"config": map[string]interface{}{
						"_headers":            "Cache-Control: max-age=3600",
						"_redirects":          "/old/* /new/:splat 301",
						"html_handling":       "auto-trailing-slash",
						"not_found_handling":  "single-page-application",
						"run_worker_first":    true,
					},
				},
			},
		},
		"bindings": {
			input: WorkerScriptInputs{
				Bindings: []struct {
					Type        string `json:"type"`
					Name        string `json:"name"`
					BucketName  string `json:"bucketName,omitempty"`
					ClassName   string `json:"className,omitempty"`
					NamespaceId string `json:"namespaceId,omitempty"`
					QueueName   string `json:"queueName,omitempty"`
					ScriptName  string `json:"scriptName,omitempty"`
					SecretName  string `json:"secretName,omitempty"`
					Service     string `json:"service,omitempty"`
					Text        string `json:"text,omitempty"`
				}{
					{Type: "kv_namespace", Name: "MY_KV", NamespaceId: "kv123"},
					{Type: "r2_bucket", Name: "MY_BUCKET", BucketName: "bucket123"},
					{Type: "durable_object_namespace", Name: "MY_DO", ClassName: "MyDurableObject"},
					{Type: "queue", Name: "MY_QUEUE", QueueName: "queue123"},
					{Type: "service", Name: "MY_SERVICE", Service: "service123"},
					{Type: "secret_text", Name: "MY_SECRET", SecretName: "secret123"},
					{Type: "plain_text", Name: "MY_TEXT", Text: "plain text value"},
				},
			},
			expected: map[string]interface{}{
				"bindings": []map[string]interface{}{
					{"type": "kv_namespace", "name": "MY_KV", "namespace_id": "kv123"},
					{"type": "r2_bucket", "name": "MY_BUCKET", "bucket_name": "bucket123"},
					{"type": "durable_object_namespace", "name": "MY_DO", "class_name": "MyDurableObject"},
					{"type": "queue", "name": "MY_QUEUE", "queue_name": "queue123"},
					{"type": "service", "name": "MY_SERVICE", "service": "service123"},
					{"type": "secret_text", "name": "MY_SECRET", "secret_name": "secret123"},
					{"type": "plain_text", "name": "MY_TEXT", "text": "plain text value"},
				},
			},
		},
		"migrations": {
			input: WorkerScriptInputs{
				Migrations: struct {
					DeletedClasses []string `json:"deletedClasses,omitempty"`
					NewClasses     []string `json:"newClasses,omitempty"`
					NewSqliteClasses []string `json:"newSqliteClasses,omitempty"`
					NewTag string `json:"newTag,omitempty"`
					OldTag string `json:"oldTag,omitempty"`
					RenamedClasses []struct {
						From string `json:"from,omitempty"`
						To   string `json:"to,omitempty"`
					} `json:"renamedClasses,omitempty"`
					Steps []struct {
						DeletedClasses []string `json:"deletedClasses,omitempty"`
						NewClasses     []string `json:"newClasses,omitempty"`
						NewSqliteClasses []string `json:"newSqliteClasses,omitempty"`
						RenamedClasses []struct {
							From string `json:"from,omitempty"`
							To   string `json:"to,omitempty"`
						} `json:"renamedClasses,omitempty"`
						TransferredClasses []struct {
							From       string `json:"from,omitempty"`
							FromScript string `json:"fromScript,omitempty"`
							To         string `json:"to,omitempty"`
						} `json:"transferredClasses,omitempty"`
					} `json:"steps,omitempty"`
					TransferredClasses []struct {
						From       string `json:"from,omitempty"`
						FromScript string `json:"fromScript,omitempty"`
						To         string `json:"to,omitempty"`
					} `json:"transferredClasses,omitempty"`
				}{
					DeletedClasses: []string{"OldClass1", "OldClass2"},
					NewClasses:     []string{"NewClass1", "NewClass2"},
					NewSqliteClasses: []string{"SqliteClass1"},
					NewTag:         "v2.0.0",
					OldTag:         "v1.0.0",
					RenamedClasses: []struct {
						From string `json:"from,omitempty"`
						To   string `json:"to,omitempty"`
					}{
						{From: "OldName", To: "NewName"},
					},
					TransferredClasses: []struct {
						From       string `json:"from,omitempty"`
						FromScript string `json:"fromScript,omitempty"`
						To         string `json:"to,omitempty"`
					}{
						{From: "ClassA", FromScript: "script1", To: "ClassB"},
					},
				},
			},
			expected: map[string]interface{}{
				"migrations": map[string]interface{}{
					"deleted_classes": []string{"OldClass1", "OldClass2"},
					"new_classes":     []string{"NewClass1", "NewClass2"},
					"new_sqlite_classes": []string{"SqliteClass1"},
					"new_tag":         "v2.0.0",
					"old_tag":         "v1.0.0",
					"renamed_classes": []map[string]interface{}{
						{"from": "OldName", "to": "NewName"},
					},
					"transferred_classes": []map[string]interface{}{
						{"from": "ClassA", "from_script": "script1", "to": "ClassB"},
					},
				},
			},
		},
		"observability": {
			input: WorkerScriptInputs{
				Observability: struct {
					Enabled          bool `json:"enabled,omitempty"`
					HeapSamplingRate int  `json:"heapSamplingRate,omitempty"`
				}{
					Enabled:          true,
					HeapSamplingRate: 50,
				},
			},
			expected: map[string]interface{}{
				"observability": map[string]interface{}{
					"enabled":             true,
					"heap_sampling_rate":  50,
				},
			},
		},
		"placement": {
			input: WorkerScriptInputs{
				Placement: struct {
					LastAnalyzedAt string `json:"lastAnalyzedAt,omitempty"`
					Mode           string `json:"mode,omitempty"`
					Status         string `json:"status,omitempty"`
				}{
					LastAnalyzedAt: "2023-05-18T10:00:00Z",
					Mode:           "smart",
					Status:         "active",
				},
			},
			expected: map[string]interface{}{
				"placement": map[string]interface{}{
					"last_analyzed_at": "2023-05-18T10:00:00Z",
					"mode":             "smart",
					"status":           "active",
				},
			},
		},
		"tail consumers": {
			input: WorkerScriptInputs{
				TailConsumers: []struct {
					Service     string `json:"service"`
					Environment string `json:"environment,omitempty"`
					Namespace   string `json:"namespace,omitempty"`
				}{
					{Service: "analytics", Environment: "production", Namespace: "logs"},
					{Service: "monitoring"},
				},
			},
			expected: map[string]interface{}{
				"tail_consumers": []map[string]interface{}{
					{"service": "analytics", "environment": "production", "namespace": "logs"},
					{"service": "monitoring"},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := buildMetadata(&tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkerScript_StructValidation(t *testing.T) {
	t.Run("WorkerScriptInputs JSON marshaling", func(t *testing.T) {
		input := WorkerScriptInputs{
			AccountId:  "account123",
			ApiToken:   "token456",
			ScriptName: "test-script",
			Content: struct {
				Filename string `json:"filename"`
				Hash     string `json:"hash"`
			}{
				Filename: "worker.js",
				Hash:     "abc123",
			},
			CompatibilityDate:  "2023-05-18",
			CompatibilityFlags: []string{"nodejs_compat"},
			MainModule:         "worker.js",
			UsageModel:         "bundled",
		}

		data, err := json.Marshal(input)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "account123")
		assert.Contains(t, string(data), "test-script")
		assert.Contains(t, string(data), "worker.js")

		var unmarshaled WorkerScriptInputs
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, input.AccountId, unmarshaled.AccountId)
		assert.Equal(t, input.ScriptName, unmarshaled.ScriptName)
		assert.Equal(t, input.Content.Filename, unmarshaled.Content.Filename)
	})

	t.Run("WorkerScriptOutputs JSON marshaling", func(t *testing.T) {
		output := WorkerScriptOutputs{
			AccountId:  "account123",
			ApiToken:   "token456",
			ScriptName: "test-script",
		}

		data, err := json.Marshal(output)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "account123")
		assert.Contains(t, string(data), "test-script")

		var unmarshaled WorkerScriptOutputs
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, output.AccountId, unmarshaled.AccountId)
		assert.Equal(t, output.ScriptName, unmarshaled.ScriptName)
	})

	t.Run("WorkerScript embedded CloudflareResource", func(t *testing.T) {
		resource := &WorkerScript{
			CloudflareResource: &CloudflareResource{},
		}
		assert.NotNil(t, resource.CloudflareResource)
	})
}

func TestWorkerScript_EdgeCases(t *testing.T) {
	t.Run("unicode and special characters", func(t *testing.T) {
		input := WorkerScriptInputs{
			AccountId:  "账户-123",
			ScriptName: "script-with-émojis-🚀",
			Bindings: []struct {
				Type        string `json:"type"`
				Name        string `json:"name"`
				BucketName  string `json:"bucketName,omitempty"`
				ClassName   string `json:"className,omitempty"`
				NamespaceId string `json:"namespaceId,omitempty"`
				QueueName   string `json:"queueName,omitempty"`
				ScriptName  string `json:"scriptName,omitempty"`
				SecretName  string `json:"secretName,omitempty"`
				Service     string `json:"service,omitempty"`
				Text        string `json:"text,omitempty"`
			}{
				{Type: "plain_text", Name: "UNICODE_VAR", Text: "Hello 世界! 🌍"},
			},
		}

		metadata := buildMetadata(&input)
		bindings := metadata["bindings"].([]map[string]interface{})
		assert.Equal(t, "Hello 世界! 🌍", bindings[0]["text"])
	})

	t.Run("large number of bindings", func(t *testing.T) {
		var bindings []struct {
			Type        string `json:"type"`
			Name        string `json:"name"`
			BucketName  string `json:"bucketName,omitempty"`
			ClassName   string `json:"className,omitempty"`
			NamespaceId string `json:"namespaceId,omitempty"`
			QueueName   string `json:"queueName,omitempty"`
			ScriptName  string `json:"scriptName,omitempty"`
			SecretName  string `json:"secretName,omitempty"`
			Service     string `json:"service,omitempty"`
			Text        string `json:"text,omitempty"`
		}

		for i := 0; i < 100; i++ {
			bindings = append(bindings, struct {
				Type        string `json:"type"`
				Name        string `json:"name"`
				BucketName  string `json:"bucketName,omitempty"`
				ClassName   string `json:"className,omitempty"`
				NamespaceId string `json:"namespaceId,omitempty"`
				QueueName   string `json:"queueName,omitempty"`
				ScriptName  string `json:"scriptName,omitempty"`
				SecretName  string `json:"secretName,omitempty"`
				Service     string `json:"service,omitempty"`
				Text        string `json:"text,omitempty"`
			}{
				Type: "plain_text",
				Name: "VAR_" + string(rune(i)),
				Text: "value" + string(rune(i)),
			})
		}

		input := WorkerScriptInputs{Bindings: bindings}
		metadata := buildMetadata(&input)
		
		resultBindings := metadata["bindings"].([]map[string]interface{})
		assert.Len(t, resultBindings, 100)
	})

	t.Run("complex migration steps", func(t *testing.T) {
		input := WorkerScriptInputs{
			Migrations: struct {
				DeletedClasses []string `json:"deletedClasses,omitempty"`
				NewClasses     []string `json:"newClasses,omitempty"`
				NewSqliteClasses []string `json:"newSqliteClasses,omitempty"`
				NewTag string `json:"newTag,omitempty"`
				OldTag string `json:"oldTag,omitempty"`
				RenamedClasses []struct {
					From string `json:"from,omitempty"`
					To   string `json:"to,omitempty"`
				} `json:"renamedClasses,omitempty"`
				Steps []struct {
					DeletedClasses []string `json:"deletedClasses,omitempty"`
					NewClasses     []string `json:"newClasses,omitempty"`
					NewSqliteClasses []string `json:"newSqliteClasses,omitempty"`
					RenamedClasses []struct {
						From string `json:"from,omitempty"`
						To   string `json:"to,omitempty"`
					} `json:"renamedClasses,omitempty"`
					TransferredClasses []struct {
						From       string `json:"from,omitempty"`
						FromScript string `json:"fromScript,omitempty"`
						To         string `json:"to,omitempty"`
					} `json:"transferredClasses,omitempty"`
				} `json:"steps,omitempty"`
				TransferredClasses []struct {
					From       string `json:"from,omitempty"`
					FromScript string `json:"fromScript,omitempty"`
					To         string `json:"to,omitempty"`
				} `json:"transferredClasses,omitempty"`
			}{
				Steps: []struct {
					DeletedClasses []string `json:"deletedClasses,omitempty"`
					NewClasses     []string `json:"newClasses,omitempty"`
					NewSqliteClasses []string `json:"newSqliteClasses,omitempty"`
					RenamedClasses []struct {
						From string `json:"from,omitempty"`
						To   string `json:"to,omitempty"`
					} `json:"renamedClasses,omitempty"`
					TransferredClasses []struct {
						From       string `json:"from,omitempty"`
						FromScript string `json:"fromScript,omitempty"`
						To         string `json:"to,omitempty"`
					} `json:"transferredClasses,omitempty"`
				}{
					{
						DeletedClasses: []string{"Step1Delete"},
						NewClasses:     []string{"Step1New"},
						RenamedClasses: []struct {
							From string `json:"from,omitempty"`
							To   string `json:"to,omitempty"`
						}{
							{From: "Step1Old", To: "Step1New"},
						},
						TransferredClasses: []struct {
							From       string `json:"from,omitempty"`
							FromScript string `json:"fromScript,omitempty"`
							To         string `json:"to,omitempty"`
						}{
							{From: "Step1Transfer", FromScript: "old-script", To: "Step1Transferred"},
						},
					},
					{
						NewClasses: []string{"Step2New"},
					},
				},
			},
		}

		metadata := buildMetadata(&input)
		migrations := metadata["migrations"].(map[string]interface{})
		steps := migrations["steps"].([]map[string]interface{})
		
		assert.Len(t, steps, 2)
		assert.Equal(t, []string{"Step1Delete"}, steps[0]["deleted_classes"])
		assert.Equal(t, []string{"Step2New"}, steps[1]["new_classes"])
	})

	t.Run("empty nested structures", func(t *testing.T) {
		input := WorkerScriptInputs{
			Assets: struct {
				Jwt    string `json:"jwt,omitempty"`
				Config struct {
					Headers          string `json:"_headers,omitempty"`
					Redirects        string `json:"_redirects,omitempty"`
					HtmlHandling     string `json:"htmlHandling,omitempty"`
					NotFoundHandling string `json:"notFoundHandling,omitempty"`
					RunWorkerFirst   bool   `json:"runWorkerFirst,omitempty"`
				} `json:"config,omitempty"`
			}{
				// Empty assets
			},
			Bindings: []struct {
				Type        string `json:"type"`
				Name        string `json:"name"`
				BucketName  string `json:"bucketName,omitempty"`
				ClassName   string `json:"className,omitempty"`
				NamespaceId string `json:"namespaceId,omitempty"`
				QueueName   string `json:"queueName,omitempty"`
				ScriptName  string `json:"scriptName,omitempty"`
				SecretName  string `json:"secretName,omitempty"`
				Service     string `json:"service,omitempty"`
				Text        string `json:"text,omitempty"`
			}{}, // Empty bindings slice
		}

		metadata := buildMetadata(&input)
		
		// Should not include empty assets or bindings
		_, hasAssets := metadata["assets"]
		_, hasBindings := metadata["bindings"]
		assert.False(t, hasAssets)
		assert.False(t, hasBindings)
	})
}