package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCli_Discover(t *testing.T) {
	tests := []struct {
		name        string
		configFlag  string
		setupFunc   func() (string, func())
		expectError bool
		errorMsg    string
	}{
		{
			name:       "with explicit config flag - valid file",
			configFlag: "sst.config.ts",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-test")
				require.NoError(t, err)
				
				configFile := filepath.Join(tmpDir, "sst.config.ts")
				err = os.WriteFile(configFile, []byte("export default {}"), 0644)
				require.NoError(t, err)
				
				// Change to temp dir
				originalDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				
				return configFile, func() {
					os.Chdir(originalDir)
					os.RemoveAll(tmpDir)
				}
			},
			expectError: false,
		},
		{
			name:       "with explicit config flag - nonexistent file",
			configFlag: "nonexistent.config.ts",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-test")
				require.NoError(t, err)
				
				originalDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				
				return "", func() {
					os.Chdir(originalDir)
					os.RemoveAll(tmpDir)
				}
			},
			expectError: true,
			errorMsg:    "Could not find",
		},
		{
			name:       "discover config automatically - success",
			configFlag: "",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-test")
				require.NoError(t, err)
				
				configFile := filepath.Join(tmpDir, "sst.config.ts")
				err = os.WriteFile(configFile, []byte("export default {}"), 0644)
				require.NoError(t, err)
				
				originalDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				
				return configFile, func() {
					os.Chdir(originalDir)
					os.RemoveAll(tmpDir)
				}
			},
			expectError: false,
		},
		{
			name:       "discover config automatically - not found",
			configFlag: "",
			setupFunc: func() (string, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-test")
				require.NoError(t, err)
				
				originalDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				
				return "", func() {
					os.Chdir(originalDir)
					os.RemoveAll(tmpDir)
				}
			},
			expectError: true,
			errorMsg:    "Could not find sst.config.ts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedPath, cleanup := tt.setupFunc()
			defer cleanup()

			cli := &Cli{
				flags: map[string]interface{}{},
			}

			if tt.configFlag != "" {
				configFlag := tt.configFlag
				cli.flags["config"] = &configFlag
			} else {
				emptyConfig := ""
				cli.flags["config"] = &emptyConfig
			}

			result, err := cli.Discover()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				if expectedPath != "" {
					// Use filepath.EvalSymlinks to resolve any symlinks for comparison
					expectedAbs, _ := filepath.EvalSymlinks(expectedPath)
					resultAbs, _ := filepath.EvalSymlinks(result)
					assert.Equal(t, expectedAbs, resultAbs)
				} else {
					// For auto-discovery, just check that we got a valid path
					assert.NotEmpty(t, result)
					assert.True(t, filepath.IsAbs(result))
				}
			}
		})
	}
}

func TestCli_configureLog(t *testing.T) {
	tests := []struct {
		name           string
		printLogsFlag  bool
		expectMultiple bool
	}{
		{
			name:           "print logs disabled",
			printLogsFlag:  false,
			expectMultiple: false,
		},
		{
			name:           "print logs enabled",
			printLogsFlag:  true,
			expectMultiple: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Cli{
				flags: map[string]interface{}{
					"print-logs": &tt.printLogsFlag,
				},
			}

			// This test mainly ensures configureLog doesn't panic
			// and can be called without errors
			assert.NotPanics(t, func() {
				cli.configureLog()
			})
		})
	}
}

func TestCli_InitProject_ConfigDiscoveryError(t *testing.T) {
	// Test case where config discovery fails
	tmpDir, err := os.MkdirTemp("", "sst-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	cli := &Cli{
		version: "test-version",
		flags: map[string]interface{}{
			"config": func() *string { s := ""; return &s }(),
		},
	}

	_, err = cli.InitProject()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Could not find sst.config.ts")
}

func TestCli_InitProject_StageError(t *testing.T) {
	// Test case where stage resolution fails
	tmpDir, err := os.MkdirTemp("", "sst-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create an invalid config file that will cause stage issues
	configFile := filepath.Join(tmpDir, "sst.config.ts")
	err = os.WriteFile(configFile, []byte("invalid config content"), 0644)
	require.NoError(t, err)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	cli := &Cli{
		version: "test-version",
		flags: map[string]interface{}{
			"config": func() *string { s := ""; return &s }(),
			"stage":  func() *string { s := ""; return &s }(),
		},
	}

	// Mock environment to ensure no stage is found
	originalStage := os.Getenv("SST_STAGE")
	defer func() {
		if originalStage != "" {
			os.Setenv("SST_STAGE", originalStage)
		} else {
			os.Unsetenv("SST_STAGE")
		}
	}()
	os.Unsetenv("SST_STAGE")

	_, err = cli.InitProject()
	// This should fail due to invalid config or stage issues
	assert.Error(t, err)
}

func TestLogFileInitialization(t *testing.T) {
	// Test that the global logFile variable is initialized
	assert.NotNil(t, logFile)
	
	// Test that we can write to it
	_, err := logFile.WriteString("test log entry\n")
	assert.NoError(t, err)
}

func TestCli_String_Config(t *testing.T) {
	configValue := "/path/to/config"
	cli := &Cli{
		flags: map[string]interface{}{
			"config": &configValue,
		},
	}

	result := cli.String("config")
	assert.Equal(t, "/path/to/config", result)
}

func TestCli_Bool_PrintLogs(t *testing.T) {
	printLogs := true
	cli := &Cli{
		flags: map[string]interface{}{
			"print-logs": &printLogs,
		},
	}

	result := cli.Bool("print-logs")
	assert.True(t, result)
}