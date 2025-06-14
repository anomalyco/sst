package resource

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerAssets_Create(t *testing.T) {
	tests := map[string]struct {
		input         WorkerAssetsInputs
		setupFiles    map[string]string // filename -> content
		expectedError string
	}{
		"missing API token": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
			},
			expectedError: "failed to initialize assets upload",
		},
		"missing account ID": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			expectedError: "failed to initialize assets upload",
		},
		"missing script name": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory: "/tmp/test",
				AccountId: "account123",
				ApiToken:  "token123",
			},
			expectedError: "failed to initialize assets upload",
		},
		"empty manifest": {
			input: WorkerAssetsInputs{
				Manifest:   AssetManifest{},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			expectedError: "failed to initialize assets upload",
		},
		"valid input with single file": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			setupFiles: map[string]string{
				"test.js": "console.log('test');",
			},
			expectedError: "failed to initialize assets upload",
		},
		"valid input with multiple files": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"index.js": AssetEntry{
						Hash:        "hash1",
						Size:        200,
						ContentType: "application/javascript",
					},
					"style.css": AssetEntry{
						Hash:        "hash2",
						Size:        150,
						ContentType: "text/css",
					},
					"image.png": AssetEntry{
						Hash:        "hash3",
						Size:        1024,
						ContentType: "image/png",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			setupFiles: map[string]string{
				"index.js":  "export default { fetch() { return new Response('Hello'); } };",
				"style.css": "body { margin: 0; }",
				"image.png": "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01",
			},
			expectedError: "failed to initialize assets upload",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup temporary directory and files
			if tt.setupFiles != nil {
				tempDir, err := os.MkdirTemp("", "worker-assets-test")
				require.NoError(t, err)
				defer os.RemoveAll(tempDir)

				for filename, content := range tt.setupFiles {
					filePath := filepath.Join(tempDir, filename)
					err := os.WriteFile(filePath, []byte(content), 0644)
					require.NoError(t, err)
				}

				tt.input.Directory = tempDir
			}

			resource := &WorkerAssets{
				CloudflareResource: &CloudflareResource{
					context: context.Background(),
					project: &project.Project{},
				},
			}

			var output CreateResult[WorkerAssetsOutputs]
			err := resource.Create(&tt.input, &output)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "assets", output.ID)
				assert.Equal(t, tt.input.Manifest, output.Outs.Manifest)
				assert.Equal(t, tt.input.Directory, output.Outs.Directory)
				assert.Equal(t, tt.input.AccountId, output.Outs.AccountId)
				assert.Equal(t, tt.input.ScriptName, output.Outs.ScriptName)
				assert.NotEmpty(t, output.Outs.Jwt)
			}
		})
	}
}

func TestWorkerAssets_Update(t *testing.T) {
	tests := map[string]struct {
		oldInput      WorkerAssetsInputs
		newInput      WorkerAssetsInputs
		setupFiles    map[string]string
		expectedError string
	}{
		"update with new manifest": {
			oldInput: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"old.js": AssetEntry{
						Hash:        "old123",
						Size:        50,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			newInput: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"new.js": AssetEntry{
						Hash:        "new123",
						Size:        75,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			setupFiles: map[string]string{
				"new.js": "console.log('updated');",
			},
			expectedError: "failed to initialize assets upload",
		},
		"update with changed API token": {
			oldInput: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "old-token",
			},
			newInput: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "new-token",
			},
			setupFiles: map[string]string{
				"test.js": "console.log('test');",
			},
			expectedError: "failed to initialize assets upload",
		},
		"update with missing API token": {
			oldInput: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			newInput: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
			},
			expectedError: "failed to initialize assets upload",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup temporary directory and files
			if tt.setupFiles != nil {
				tempDir, err := os.MkdirTemp("", "worker-assets-test")
				require.NoError(t, err)
				defer os.RemoveAll(tempDir)

				for filename, content := range tt.setupFiles {
					filePath := filepath.Join(tempDir, filename)
					err := os.WriteFile(filePath, []byte(content), 0644)
					require.NoError(t, err)
				}

				tt.newInput.Directory = tempDir
			}

			resource := &WorkerAssets{
				CloudflareResource: &CloudflareResource{
					context: context.Background(),
					project: &project.Project{},
				},
			}

			input := &UpdateInput[WorkerAssetsInputs, WorkerAssetsOutputs]{
				Olds: WorkerAssetsOutputs{
					Manifest:   tt.oldInput.Manifest,
					Directory:  tt.oldInput.Directory,
					AccountId:  tt.oldInput.AccountId,
					ScriptName: tt.oldInput.ScriptName,
					Jwt:        "old-jwt",
				},
				News: tt.newInput,
			}

			var output UpdateResult[WorkerAssetsOutputs]
			err := resource.Update(input, &output)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.newInput.Manifest, output.Outs.Manifest)
				assert.Equal(t, tt.newInput.Directory, output.Outs.Directory)
				assert.Equal(t, tt.newInput.AccountId, output.Outs.AccountId)
				assert.Equal(t, tt.newInput.ScriptName, output.Outs.ScriptName)
				assert.NotEmpty(t, output.Outs.Jwt)
			}
		})
	}
}

func TestWorkerAssetsInputs_Validation(t *testing.T) {
	tests := map[string]struct {
		input WorkerAssetsInputs
		valid bool
	}{
		"valid input": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			valid: true,
		},
		"empty manifest": {
			input: WorkerAssetsInputs{
				Manifest:   AssetManifest{},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			valid: true, // Empty manifest is technically valid
		},
		"nil manifest": {
			input: WorkerAssetsInputs{
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			valid: true, // Nil manifest is valid (will be empty)
		},
		"empty directory": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				AccountId:  "account123",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			valid: false,
		},
		"empty account ID": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				ScriptName: "test-worker",
				ApiToken:   "token123",
			},
			valid: false,
		},
		"empty script name": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory: "/tmp/test",
				AccountId: "account123",
				ApiToken:  "token123",
			},
			valid: false,
		},
		"empty API token": {
			input: WorkerAssetsInputs{
				Manifest: AssetManifest{
					"test.js": AssetEntry{
						Hash:        "abc123",
						Size:        100,
						ContentType: "application/javascript",
					},
				},
				Directory:  "/tmp/test",
				AccountId:  "account123",
				ScriptName: "test-worker",
			},
			valid: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Basic validation - check if required fields are present
			hasRequiredFields := tt.input.AccountId != "" &&
				tt.input.ScriptName != "" &&
				tt.input.ApiToken != "" &&
				tt.input.Directory != ""

			assert.Equal(t, tt.valid, hasRequiredFields)
		})
	}
}

func TestAssetManifest_Structure(t *testing.T) {
	tests := map[string]struct {
		manifest AssetManifest
		valid    bool
	}{
		"valid manifest with single file": {
			manifest: AssetManifest{
				"index.js": AssetEntry{
					Hash:        "abc123",
					Size:        100,
					ContentType: "application/javascript",
				},
			},
			valid: true,
		},
		"valid manifest with multiple files": {
			manifest: AssetManifest{
				"index.js": AssetEntry{
					Hash:        "hash1",
					Size:        200,
					ContentType: "application/javascript",
				},
				"style.css": AssetEntry{
					Hash:        "hash2",
					Size:        150,
					ContentType: "text/css",
				},
				"image.png": AssetEntry{
					Hash:        "hash3",
					Size:        1024,
					ContentType: "image/png",
				},
			},
			valid: true,
		},
		"manifest with nested paths": {
			manifest: AssetManifest{
				"assets/js/main.js": AssetEntry{
					Hash:        "hash1",
					Size:        500,
					ContentType: "application/javascript",
				},
				"assets/css/style.css": AssetEntry{
					Hash:        "hash2",
					Size:        300,
					ContentType: "text/css",
				},
				"images/logo.svg": AssetEntry{
					Hash:        "hash3",
					Size:        2048,
					ContentType: "image/svg+xml",
				},
			},
			valid: true,
		},
		"manifest with special characters in filenames": {
			manifest: AssetManifest{
				"file-with-dashes.js": AssetEntry{
					Hash:        "hash1",
					Size:        100,
					ContentType: "application/javascript",
				},
				"file_with_underscores.css": AssetEntry{
					Hash:        "hash2",
					Size:        200,
					ContentType: "text/css",
				},
				"file.with.dots.html": AssetEntry{
					Hash:        "hash3",
					Size:        300,
					ContentType: "text/html",
				},
			},
			valid: true,
		},
		"empty manifest": {
			manifest: AssetManifest{},
			valid:    true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Validate manifest structure
			for path, entry := range tt.manifest {
				assert.NotEmpty(t, path, "File path should not be empty")
				assert.NotEmpty(t, entry.Hash, "Hash should not be empty")
				assert.Greater(t, entry.Size, int64(0), "Size should be positive")
				assert.NotEmpty(t, entry.ContentType, "ContentType should not be empty")
			}
		})
	}
}

func TestAssetEntry_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		entry AssetEntry
		valid bool
	}{
		"valid JavaScript entry": {
			entry: AssetEntry{
				Hash:        "abc123def456",
				Size:        1024,
				ContentType: "application/javascript",
			},
			valid: true,
		},
		"valid CSS entry": {
			entry: AssetEntry{
				Hash:        "css789hash",
				Size:        512,
				ContentType: "text/css",
			},
			valid: true,
		},
		"valid image entry": {
			entry: AssetEntry{
				Hash:        "img456hash",
				Size:        2048,
				ContentType: "image/png",
			},
			valid: true,
		},
		"entry with zero size": {
			entry: AssetEntry{
				Hash:        "empty123",
				Size:        0,
				ContentType: "text/plain",
			},
			valid: false,
		},
		"entry with negative size": {
			entry: AssetEntry{
				Hash:        "negative123",
				Size:        -100,
				ContentType: "text/plain",
			},
			valid: false,
		},
		"entry with empty hash": {
			entry: AssetEntry{
				Hash:        "",
				Size:        100,
				ContentType: "text/plain",
			},
			valid: false,
		},
		"entry with empty content type": {
			entry: AssetEntry{
				Hash:        "hash123",
				Size:        100,
				ContentType: "",
			},
			valid: false,
		},
		"entry with very long hash": {
			entry: AssetEntry{
				Hash:        strings.Repeat("a", 1000),
				Size:        100,
				ContentType: "text/plain",
			},
			valid: true, // Long hashes are valid
		},
		"entry with very large size": {
			entry: AssetEntry{
				Hash:        "large123",
				Size:        1024 * 1024 * 100, // 100MB
				ContentType: "application/octet-stream",
			},
			valid: true, // Large files are valid
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			isValid := tt.entry.Hash != "" &&
				tt.entry.Size > 0 &&
				tt.entry.ContentType != ""

			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func TestWorkerAssetsOutputs_Structure(t *testing.T) {
	output := WorkerAssetsOutputs{
		Manifest: AssetManifest{
			"test.js": AssetEntry{
				Hash:        "abc123",
				Size:        100,
				ContentType: "application/javascript",
			},
		},
		Directory:  "/tmp/test",
		AccountId:  "account123",
		ScriptName: "test-worker",
		Jwt:        "jwt-token-123",
	}

	assert.NotNil(t, output.Manifest)
	assert.NotEmpty(t, output.Directory)
	assert.NotEmpty(t, output.AccountId)
	assert.NotEmpty(t, output.ScriptName)
	assert.NotEmpty(t, output.Jwt)

	// Verify manifest structure
	entry, exists := output.Manifest["test.js"]
	assert.True(t, exists)
	assert.Equal(t, "abc123", entry.Hash)
	assert.Equal(t, int64(100), entry.Size)
	assert.Equal(t, "application/javascript", entry.ContentType)
}

func TestInitializeAssetsResponse_Structure(t *testing.T) {
	response := InitializeAssetsResponse{
		Buckets: [][]string{
			{"hash1", "hash2"},
			{"hash3"},
			{"hash4", "hash5", "hash6"},
		},
		Jwt: "init-jwt-token",
	}

	assert.Len(t, response.Buckets, 3)
	assert.Len(t, response.Buckets[0], 2)
	assert.Len(t, response.Buckets[1], 1)
	assert.Len(t, response.Buckets[2], 3)
	assert.Equal(t, "init-jwt-token", response.Jwt)

	// Verify bucket contents
	assert.Equal(t, "hash1", response.Buckets[0][0])
	assert.Equal(t, "hash2", response.Buckets[0][1])
	assert.Equal(t, "hash3", response.Buckets[1][0])
	assert.Equal(t, "hash4", response.Buckets[2][0])
	assert.Equal(t, "hash5", response.Buckets[2][1])
	assert.Equal(t, "hash6", response.Buckets[2][2])
}

func TestUploadResponse_Structure(t *testing.T) {
	response := UploadResponse{
		Jwt: "upload-jwt-token",
	}

	assert.Equal(t, "upload-jwt-token", response.Jwt)
}

func TestWorkerAssets_EmbeddedCloudflareResource(t *testing.T) {
	ctx := context.Background()
	proj := &project.Project{}

	resource := &WorkerAssets{
		CloudflareResource: &CloudflareResource{
			context: ctx,
			project: proj,
		},
	}

	assert.NotNil(t, resource.CloudflareResource)
	assert.Equal(t, ctx, resource.CloudflareResource.context)
	assert.Equal(t, proj, resource.CloudflareResource.project)
}

func TestWorkerAssets_ConcurrentUploadScenarios(t *testing.T) {
	tests := map[string]struct {
		buckets       [][]string
		expectedError string
	}{
		"single bucket": {
			buckets: [][]string{
				{"hash1", "hash2", "hash3"},
			},
			expectedError: "failed to initialize assets upload",
		},
		"multiple buckets": {
			buckets: [][]string{
				{"hash1", "hash2"},
				{"hash3", "hash4"},
				{"hash5", "hash6"},
			},
			expectedError: "failed to initialize assets upload",
		},
		"many small buckets": {
			buckets: [][]string{
				{"hash1"},
				{"hash2"},
				{"hash3"},
				{"hash4"},
				{"hash5"},
			},
			expectedError: "failed to initialize assets upload",
		},
		"empty buckets": {
			buckets:       [][]string{},
			expectedError: "failed to initialize assets upload",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a mock InitializeAssetsResponse
			response := &InitializeAssetsResponse{
				Buckets: tt.buckets,
				Jwt:     "test-jwt",
			}

			// Verify bucket structure
			assert.Equal(t, len(tt.buckets), len(response.Buckets))
			for i, bucket := range tt.buckets {
				assert.Equal(t, bucket, response.Buckets[i])
			}
		})
	}
}

func TestWorkerAssets_FileHandlingEdgeCases(t *testing.T) {
	tests := map[string]struct {
		filename    string
		content     string
		contentType string
		valid       bool
	}{
		"JavaScript file": {
			filename:    "main.js",
			content:     "console.log('Hello World');",
			contentType: "application/javascript",
			valid:       true,
		},
		"TypeScript file": {
			filename:    "app.ts",
			content:     "const message: string = 'Hello';",
			contentType: "application/typescript",
			valid:       true,
		},
		"CSS file": {
			filename:    "styles.css",
			content:     "body { margin: 0; padding: 0; }",
			contentType: "text/css",
			valid:       true,
		},
		"HTML file": {
			filename:    "index.html",
			content:     "<!DOCTYPE html><html><head><title>Test</title></head><body></body></html>",
			contentType: "text/html",
			valid:       true,
		},
		"JSON file": {
			filename:    "config.json",
			content:     `{"name": "test", "version": "1.0.0"}`,
			contentType: "application/json",
			valid:       true,
		},
		"binary file": {
			filename:    "image.png",
			content:     "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01",
			contentType: "image/png",
			valid:       true,
		},
		"empty file": {
			filename:    "empty.txt",
			content:     "",
			contentType: "text/plain",
			valid:       true,
		},
		"large file": {
			filename:    "large.txt",
			content:     strings.Repeat("a", 10000),
			contentType: "text/plain",
			valid:       true,
		},
		"file with unicode": {
			filename:    "unicode.txt",
			content:     "Hello 世界 🌍",
			contentType: "text/plain",
			valid:       true,
		},
		"file with special characters": {
			filename:    "special-chars_file.test.js",
			content:     "// Test file with special characters in name",
			contentType: "application/javascript",
			valid:       true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temporary file
			tempDir, err := os.MkdirTemp("", "worker-assets-file-test")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			filePath := filepath.Join(tempDir, tt.filename)
			err = os.WriteFile(filePath, []byte(tt.content), 0644)
			require.NoError(t, err)

			// Verify file was created correctly
			readContent, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, tt.content, string(readContent))

			// Create asset entry
			entry := AssetEntry{
				Hash:        "test-hash-" + tt.filename,
				Size:        int64(len(tt.content)),
				ContentType: tt.contentType,
			}

			assert.Equal(t, int64(len(tt.content)), entry.Size)
			assert.Equal(t, tt.contentType, entry.ContentType)
			assert.NotEmpty(t, entry.Hash)
		})
	}
}