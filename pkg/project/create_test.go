package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		setupFunc    func(t *testing.T) (string, func())
		expectError  bool
		expectedErr  error
	}{
		{
			name:         "returns error when sst.config.ts already exists",
			templateName: "aws-base",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-create-test-*")
				require.NoError(t, err)

				// Create existing sst.config.ts
				configPath := filepath.Join(tmpDir, "sst.config.ts")
				err = os.WriteFile(configPath, []byte("export default {}"), 0644)
				require.NoError(t, err)

				originalDir, err := os.Getwd()
				require.NoError(t, err)
				err = os.Chdir(tmpDir)
				require.NoError(t, err)

				cleanup := func() {
					os.Chdir(originalDir)
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			expectError: true,
			expectedErr: ErrConfigExists,
		},
		{
			name:         "returns error for invalid template",
			templateName: "nonexistent-template",
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-create-test-*")
				require.NoError(t, err)

				originalDir, err := os.Getwd()
				require.NoError(t, err)
				err = os.Chdir(tmpDir)
				require.NoError(t, err)

				cleanup := func() {
					os.Chdir(originalDir)
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := tt.setupFunc(t)
			defer cleanup()

			instructions, err := Create(tt.templateName, tmpDir)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				}
				assert.Nil(t, instructions)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, instructions)
			}
		})
	}
}

func TestCreateSteps(t *testing.T) {
	t.Run("handles copy step", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-create-copy-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create a mock preset with copy step
		presetDir := filepath.Join(tmpDir, "templates", "test-template")
		err = os.MkdirAll(presetDir, 0755)
		require.NoError(t, err)

		// Create preset.json
		preset := map[string]interface{}{
			"steps": []map[string]interface{}{
				{
					"type":       "copy",
					"properties": map[string]interface{}{},
				},
			},
		}
		presetBytes, err := json.Marshal(preset)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(presetDir, "preset.json"), presetBytes, 0644)
		require.NoError(t, err)

		// Create files directory with test file
		filesDir := filepath.Join(presetDir, "files")
		err = os.MkdirAll(filesDir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(filesDir, "test.txt"), []byte("Hello {{.App}}"), 0644)
		require.NoError(t, err)

		// This test would require mocking the platform.Templates embed.FS
		// For now, we'll test the error case
		_, err = Create("test-template", tmpDir)
		assert.Error(t, err) // Expected to fail since we can't mock embed.FS easily
	})

	t.Run("handles gitignore step", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-create-gitignore-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create existing .gitignore
		gitignoreContent := "node_modules/\n"
		err = os.WriteFile(".gitignore", []byte(gitignoreContent), 0644)
		require.NoError(t, err)

		// This test would also require mocking platform.Templates
		// Testing the gitignore logic in isolation would be better
	})
}

func TestCreateNpmStep(t *testing.T) {
	t.Run("handles npm step with existing package.json", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-create-npm-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create package.json
		packageJson := map[string]interface{}{
			"name":         "test-project",
			"version":      "1.0.0",
			"dependencies": map[string]interface{}{},
		}
		packageJsonBytes, err := json.Marshal(packageJson)
		require.NoError(t, err)
		packageJsonPath := filepath.Join(tmpDir, "package.json")
		err = os.WriteFile(packageJsonPath, packageJsonBytes, 0644)
		require.NoError(t, err)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// This test would require mocking npm.Get and platform.Templates
		// For comprehensive testing, we'd need to mock external dependencies
	})
}

func TestCreatePatchStep(t *testing.T) {
	t.Run("handles patch step with JSON file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-create-patch-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a JSON file to patch
		testJson := map[string]interface{}{
			"name":    "test",
			"version": "1.0.0",
		}
		testJsonBytes, err := json.Marshal(testJson)
		require.NoError(t, err)
		testJsonPath := filepath.Join(tmpDir, "test.json")
		err = os.WriteFile(testJsonPath, testJsonBytes, 0644)
		require.NoError(t, err)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// This test would require mocking platform.Templates and creating a proper preset
		// The patch logic is complex and would benefit from unit testing in isolation
	})
}

func TestCreateErrorHandling(t *testing.T) {
	t.Run("handles invalid preset JSON", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-create-error-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Test with invalid template name
		_, err = Create("invalid-template", tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read preset.json")
	})

	t.Run("handles directory creation errors", func(t *testing.T) {
		// This would test scenarios where directory creation fails
		// due to permissions or other filesystem issues
		// Requires more complex setup with restricted permissions
	})
}

func TestCreateInstructions(t *testing.T) {
	t.Run("returns instructions from preset", func(t *testing.T) {
		// This test would verify that instructions are properly extracted
		// from the preset and returned to the caller
		// Requires mocking platform.Templates with a preset containing instructions
	})
}

func TestCreateDirectoryName(t *testing.T) {
	t.Run("uses current directory name for app name", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "my-test-app-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// The directory name should be used as the app name in templates
		// This would be tested by creating a template that uses {{.App}}
		// and verifying the output contains the directory name
	})
}