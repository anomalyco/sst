package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, func())
		expectError bool
		errorMsg    string
	}{
		{
			name: "finds sst.config.ts in current directory",
			setupFunc: func(t *testing.T) (string, func()) {
				// Create temporary directory
				tmpDir, err := os.MkdirTemp("", "sst-test-*")
				require.NoError(t, err)
				
				// Create sst.config.ts file
				configPath := filepath.Join(tmpDir, "sst.config.ts")
				err = os.WriteFile(configPath, []byte("export default {}"), 0644)
				require.NoError(t, err)
				
				// Change to temp directory
				originalDir, err := os.Getwd()
				require.NoError(t, err)
				err = os.Chdir(tmpDir)
				require.NoError(t, err)
				
				cleanup := func() {
					os.Chdir(originalDir)
					os.RemoveAll(tmpDir)
				}
				
				return configPath, cleanup
			},
			expectError: false,
		},
		{
			name: "finds sst.config.ts in parent directory",
			setupFunc: func(t *testing.T) (string, func()) {
				// Create temporary directory structure
				tmpDir, err := os.MkdirTemp("", "sst-test-*")
				require.NoError(t, err)
				
				subDir := filepath.Join(tmpDir, "subdir")
				err = os.MkdirAll(subDir, 0755)
				require.NoError(t, err)
				
				// Create sst.config.ts in parent
				configPath := filepath.Join(tmpDir, "sst.config.ts")
				err = os.WriteFile(configPath, []byte("export default {}"), 0644)
				require.NoError(t, err)
				
				// Change to subdirectory
				originalDir, err := os.Getwd()
				require.NoError(t, err)
				err = os.Chdir(subDir)
				require.NoError(t, err)
				
				cleanup := func() {
					os.Chdir(originalDir)
					os.RemoveAll(tmpDir)
				}
				
				return configPath, cleanup
			},
			expectError: false,
		},
		{
			name: "returns error when sst.config.ts not found",
			setupFunc: func(t *testing.T) (string, func()) {
				// Create temporary directory without config file
				tmpDir, err := os.MkdirTemp("", "sst-test-*")
				require.NoError(t, err)
				
				// Change to temp directory
				originalDir, err := os.Getwd()
				require.NoError(t, err)
				err = os.Chdir(tmpDir)
				require.NoError(t, err)
				
				cleanup := func() {
					os.Chdir(originalDir)
					os.RemoveAll(tmpDir)
				}
				
				return "", cleanup
			},
			expectError: true,
			errorMsg:    "File 'sst.config.ts' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedPath, cleanup := tt.setupFunc(t)
			defer cleanup()

			result, err := Discover()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				// Use filepath.EvalSymlinks to resolve any symlinks (like /private on macOS)
				expectedResolved, _ := filepath.EvalSymlinks(expectedPath)
				resultResolved, _ := filepath.EvalSymlinks(result)
				assert.Equal(t, expectedResolved, resultResolved)
				
				// Verify .sst directory was created
				sstDir := ResolveWorkingDir(result)
				assert.DirExists(t, sstDir)
			}
		})
	}
}

func TestResolveWorkingDir(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		expected   string
	}{
		{
			name:       "resolves working directory from config path",
			configPath: "/path/to/project/sst.config.ts",
			expected:   "/path/to/project/.sst",
		},
		{
			name:       "handles nested config path",
			configPath: "/path/to/nested/project/sst.config.ts",
			expected:   "/path/to/nested/project/.sst",
		},
		{
			name:       "handles root config path",
			configPath: "/sst.config.ts",
			expected:   "/.sst",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveWorkingDir(tt.configPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolvePlatformDir(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		expected   string
	}{
		{
			name:       "resolves platform directory from config path",
			configPath: "/path/to/project/sst.config.ts",
			expected:   "/path/to/project/.sst/platform",
		},
		{
			name:       "handles nested config path",
			configPath: "/path/to/nested/project/sst.config.ts",
			expected:   "/path/to/nested/project/.sst/platform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolvePlatformDir(tt.configPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveLogDir(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		expected   string
	}{
		{
			name:       "resolves log directory from config path",
			configPath: "/path/to/project/sst.config.ts",
			expected:   "/path/to/project/.sst/log",
		},
		{
			name:       "handles nested config path",
			configPath: "/path/to/nested/project/sst.config.ts",
			expected:   "/path/to/nested/project/.sst/log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveLogDir(tt.configPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProjectValidation(t *testing.T) {
	tests := []struct {
		name        string
		stage       string
		expectError bool
		expectedErr error
	}{
		{
			name:        "valid stage name with letters and numbers",
			stage:       "dev123",
			expectError: false,
		},
		{
			name:        "valid stage name with hyphens",
			stage:       "dev-staging",
			expectError: false,
		},
		{
			name:        "invalid stage name with underscore",
			stage:       "dev_staging",
			expectError: true,
			expectedErr: ErrInvalidStageName,
		},
		{
			name:        "invalid stage name with special characters",
			stage:       "dev@staging",
			expectError: true,
			expectedErr: ErrInvalidStageName,
		},
		{
			name:        "invalid stage name with spaces",
			stage:       "dev staging",
			expectError: true,
			expectedErr: ErrInvalidStageName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal config for testing validation
			tmpDir, err := os.MkdirTemp("", "sst-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			configPath := filepath.Join(tmpDir, "sst.config.ts")
			
			config := &ProjectConfig{
				Version: "dev",
				Stage:   tt.stage,
				Config:  configPath,
			}

			// Test stage validation by calling New
			// Note: This will fail at config evaluation, but we're testing the stage validation part
			_, err = New(config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
			} else {
				// For valid stages, we expect a different error (config evaluation failure)
				// not the stage validation error
				if err != nil {
					assert.NotEqual(t, ErrInvalidStageName, err)
				}
			}
		})
	}
}

func TestProjectPaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sst-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "sst.config.ts")
	
	// Create a project instance with minimal setup
	project := &Project{
		root:   tmpDir,
		config: configPath,
	}

	t.Run("PathRoot returns correct root path", func(t *testing.T) {
		assert.Equal(t, tmpDir, project.PathRoot())
	})

	t.Run("PathConfig returns correct config path", func(t *testing.T) {
		assert.Equal(t, configPath, project.PathConfig())
	})

	t.Run("PathWorkingDir returns correct working directory", func(t *testing.T) {
		expected := filepath.Join(tmpDir, ".sst")
		assert.Equal(t, expected, project.PathWorkingDir())
	})

	t.Run("PathPlatformDir returns correct platform directory", func(t *testing.T) {
		expected := filepath.Join(tmpDir, ".sst", "platform")
		assert.Equal(t, expected, project.PathPlatformDir())
	})

	t.Run("PathLog returns correct log paths", func(t *testing.T) {
		// Test log directory
		expectedDir := filepath.Join(tmpDir, ".sst", "log")
		assert.Equal(t, expectedDir, project.PathLog(""))

		// Test specific log file
		expectedFile := filepath.Join(tmpDir, ".sst", "log", "test.log")
		assert.Equal(t, expectedFile, project.PathLog("test"))
	})
}

func TestErrorTypes(t *testing.T) {
	t.Run("ErrVersionMismatch error message", func(t *testing.T) {
		err := &ErrVersionMismatch{
			Needed:   "1.0.0",
			Received: "2.0.0",
		}
		assert.Equal(t, "ErrorVersionMismatch", err.Error())
	})

	t.Run("ErrBuildFailed error message", func(t *testing.T) {
		err := &ErrBuildFailed{
			msg: "Build failed with errors",
		}
		assert.Equal(t, "Build failed with errors", err.Error())
	})
}