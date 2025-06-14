package server

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates new registry successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			
			assert.NotNil(t, registry)
			assert.NotNil(t, registry.types)
			assert.Equal(t, 0, len(registry.types))
		})
	}
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "registers string value",
			value: "test",
		},
		{
			name:  "registers int value",
			value: 42,
		},
		{
			name:  "registers struct value",
			value: struct{ Name string }{Name: "test"},
		},
		{
			name:  "registers nil value",
			value: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			
			// Register should not panic
			assert.NotPanics(t, func() {
				registry.Register(tt.value)
			})
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	tests := []struct {
		name         string
		registerName string
		registerType reflect.Type
		getName      string
		expectFound  bool
	}{
		{
			name:         "gets existing type",
			registerName: "string",
			registerType: reflect.TypeOf(""),
			getName:      "string",
			expectFound:  true,
		},
		{
			name:         "gets non-existing type",
			registerName: "string",
			registerType: reflect.TypeOf(""),
			getName:      "int",
			expectFound:  false,
		},
		{
			name:         "gets from empty registry",
			registerName: "",
			registerType: nil,
			getName:      "string",
			expectFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			
			// Register type if specified
			if tt.registerName != "" && tt.registerType != nil {
				registry.types[tt.registerName] = tt.registerType
			}
			
			// Get type
			resultType, found := registry.Get(tt.getName)
			
			assert.Equal(t, tt.expectFound, found)
			if tt.expectFound {
				assert.Equal(t, tt.registerType, resultType)
			} else {
				assert.Nil(t, resultType)
			}
		})
	}
}

func TestResolveServerFile(t *testing.T) {
	tests := []struct {
		name     string
		cfgPath  string
		stage    string
		expected string
	}{
		{
			name:     "resolves server file with dev stage",
			cfgPath:  "/path/to/config",
			stage:    "dev",
			expected: "dev.server",
		},
		{
			name:     "resolves server file with production stage",
			cfgPath:  "/path/to/config",
			stage:    "production",
			expected: "production.server",
		},
		{
			name:     "resolves server file with custom stage",
			cfgPath:  "/custom/path",
			stage:    "staging",
			expected: "staging.server",
		},
		{
			name:     "handles empty stage",
			cfgPath:  "/path/to/config",
			stage:    "",
			expected: ".server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveServerFile(tt.cfgPath, tt.stage)
			
			assert.Contains(t, result, tt.expected)
			assert.True(t, filepath.IsAbs(result) || filepath.Base(result) == tt.expected)
		})
	}
}

func TestDiscover(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, string, func())
		envVar      string
		expectError bool
		expectedErr error
	}{
		{
			name: "discovers server from environment variable",
			setupFunc: func(t *testing.T) (string, string, func()) {
				return "/path/to/config", "dev", func() {}
			},
			envVar:      "http://localhost:8080",
			expectError: false,
		},
		{
			name: "discovers server from file",
			setupFunc: func(t *testing.T) (string, string, func()) {
				// Create temporary directory
				tmpDir, err := os.MkdirTemp("", "sst-server-test-*")
				require.NoError(t, err)
				
				// Create .sst directory (ResolveWorkingDir adds .sst to the path)
				sstDir := filepath.Join(tmpDir, ".sst")
				err = os.MkdirAll(sstDir, 0755)
				require.NoError(t, err)
				
				// Create server file in .sst directory
				serverFile := filepath.Join(sstDir, "dev.server")
				err = os.WriteFile(serverFile, []byte("http://localhost:9000"), 0644)
				require.NoError(t, err)
				
				cleanup := func() {
					os.RemoveAll(tmpDir)
				}
				
				// Return the config path (tmpDir), not the .sst directory
				return filepath.Join(tmpDir, "sst.config.ts"), "dev", cleanup
			},
			envVar:      "",
			expectError: false,
		},
		{
			name: "returns error when server file not found",
			setupFunc: func(t *testing.T) (string, string, func()) {
				// Create temporary directory without server file
				tmpDir, err := os.MkdirTemp("", "sst-server-test-*")
				require.NoError(t, err)
				
				cleanup := func() {
					os.RemoveAll(tmpDir)
				}
				
				return tmpDir, "dev", cleanup
			},
			envVar:      "",
			expectError: true,
			expectedErr: ErrServerNotFound,
		},
		{
			name: "returns error when server file is corrupted",
			setupFunc: func(t *testing.T) (string, string, func()) {
				// Create temporary directory
				tmpDir, err := os.MkdirTemp("", "sst-server-test-*")
				require.NoError(t, err)
				
				// Create server file with restricted permissions
				serverFile := filepath.Join(tmpDir, "dev.server")
				err = os.WriteFile(serverFile, []byte("http://localhost:9000"), 0000)
				require.NoError(t, err)
				
				cleanup := func() {
					os.Chmod(serverFile, 0644) // Restore permissions for cleanup
					os.RemoveAll(tmpDir)
				}
				
				return tmpDir, "dev", cleanup
			},
			envVar:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgPath, stage, cleanup := tt.setupFunc(t)
			defer cleanup()
			
			// Set environment variable if specified
			if tt.envVar != "" {
				oldEnv := os.Getenv("SST_SERVER")
				os.Setenv("SST_SERVER", tt.envVar)
				defer os.Setenv("SST_SERVER", oldEnv)
			} else {
				// Ensure environment variable is not set
				oldEnv := os.Getenv("SST_SERVER")
				os.Unsetenv("SST_SERVER")
				defer os.Setenv("SST_SERVER", oldEnv)
			}
			
			result, err := Discover(cfgPath, stage)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				}
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				if tt.envVar != "" {
					assert.Equal(t, tt.envVar, result)
				} else {
					assert.NotEmpty(t, result)
					assert.Contains(t, result, "localhost")
				}
			}
		})
	}
}

func TestDiscover_EnvironmentVariablePriority(t *testing.T) {
	// Create temporary directory with server file
	tmpDir, err := os.MkdirTemp("", "sst-server-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	// Create server file
	serverFile := filepath.Join(tmpDir, "dev.server")
	err = os.WriteFile(serverFile, []byte("http://localhost:9000"), 0644)
	require.NoError(t, err)
	
	// Set environment variable
	envValue := "http://localhost:8080"
	oldEnv := os.Getenv("SST_SERVER")
	os.Setenv("SST_SERVER", envValue)
	defer os.Setenv("SST_SERVER", oldEnv)
	
	// Discover should return environment variable value, not file content
	result, err := Discover(tmpDir, "dev")
	
	assert.NoError(t, err)
	assert.Equal(t, envValue, result)
}

func TestDiscover_FileContent(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		expected    string
	}{
		{
			name:        "reads simple URL",
			fileContent: "http://localhost:8080",
			expected:    "http://localhost:8080",
		},
		{
			name:        "reads URL with trailing newline",
			fileContent: "http://localhost:8080\n",
			expected:    "http://localhost:8080\n",
		},
		{
			name:        "reads empty file",
			fileContent: "",
			expected:    "",
		},
		{
			name:        "reads URL with port",
			fileContent: "http://127.0.0.1:9999",
			expected:    "http://127.0.0.1:9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "sst-server-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)
			
			// Create .sst directory (ResolveWorkingDir adds .sst to the path)
			sstDir := filepath.Join(tmpDir, ".sst")
			err = os.MkdirAll(sstDir, 0755)
			require.NoError(t, err)
			
			// Create server file with specific content in .sst directory
			serverFile := filepath.Join(sstDir, "test.server")
			err = os.WriteFile(serverFile, []byte(tt.fileContent), 0644)
			require.NoError(t, err)
			
			// Ensure no environment variable is set
			oldEnv := os.Getenv("SST_SERVER")
			os.Unsetenv("SST_SERVER")
			defer os.Setenv("SST_SERVER", oldEnv)
			
			// Use config path (tmpDir + config file), not the .sst directory
			configPath := filepath.Join(tmpDir, "sst.config.ts")
			result, err := Discover(configPath, "test")
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrServerNotFound(t *testing.T) {
	assert.NotNil(t, ErrServerNotFound)
	assert.Equal(t, "server not found", ErrServerNotFound.Error())
}