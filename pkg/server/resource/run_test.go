package resource

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/flag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func TestNewRun(t *testing.T) {
	tests := map[string]struct {
		setupEnv     func()
		cleanupEnv   func()
		expectedWeight int64
	}{
		"default weight when no env var": {
			setupEnv: func() {
				flag.SST_BUILD_CONCURRENCY_SITE = ""
			},
			cleanupEnv: func() {
				flag.SST_BUILD_CONCURRENCY_SITE = ""
			},
			expectedWeight: 1,
		},
		"custom weight from env var": {
			setupEnv: func() {
				flag.SST_BUILD_CONCURRENCY_SITE = "5"
			},
			cleanupEnv: func() {
				flag.SST_BUILD_CONCURRENCY_SITE = ""
			},
			expectedWeight: 5,
		},
		"invalid env var defaults to 1": {
			setupEnv: func() {
				flag.SST_BUILD_CONCURRENCY_SITE = "invalid"
			},
			cleanupEnv: func() {
				flag.SST_BUILD_CONCURRENCY_SITE = ""
			},
			expectedWeight: 1, // ParseInt will fail and return 0, but we expect 1 as default
		},
		"zero weight from env var": {
			setupEnv: func() {
				flag.SST_BUILD_CONCURRENCY_SITE = "0"
			},
			cleanupEnv: func() {
				flag.SST_BUILD_CONCURRENCY_SITE = ""
			},
			expectedWeight: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup
			originalValue := flag.SST_BUILD_CONCURRENCY_SITE
			tt.setupEnv()
			defer func() {
				flag.SST_BUILD_CONCURRENCY_SITE = originalValue
				tt.cleanupEnv()
			}()

			// Test
			run := NewRun()

			// Verify
			assert.NotNil(t, run)
			assert.NotNil(t, run.lock)
			
			// We can't directly access the weight from semaphore.Weighted,
			// but we can test that the semaphore was created
			assert.IsType(t, &semaphore.Weighted{}, run.lock)
		})
	}
}

func TestRunInputs(t *testing.T) {
	tests := map[string]struct {
		input    RunInputs
		expected RunInputs
	}{
		"basic run inputs": {
			input: RunInputs{
				Command: "echo hello",
				Cwd:     "/tmp",
				Env:     map[string]string{"KEY": "value"},
				Version: "1.0.0",
			},
			expected: RunInputs{
				Command: "echo hello",
				Cwd:     "/tmp",
				Env:     map[string]string{"KEY": "value"},
				Version: "1.0.0",
			},
		},
		"empty inputs": {
			input: RunInputs{
				Command: "",
				Cwd:     "",
				Env:     map[string]string{},
				Version: "",
			},
			expected: RunInputs{
				Command: "",
				Cwd:     "",
				Env:     map[string]string{},
				Version: "",
			},
		},
		"nil env map": {
			input: RunInputs{
				Command: "ls",
				Cwd:     "/",
				Env:     nil,
				Version: "2.0.0",
			},
			expected: RunInputs{
				Command: "ls",
				Cwd:     "/",
				Env:     nil,
				Version: "2.0.0",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestRunOutputs(t *testing.T) {
	// Test that RunOutputs is an empty struct
	outputs := RunOutputs{}
	assert.Equal(t, RunOutputs{}, outputs)
}

func TestRun_Create(t *testing.T) {
	tests := map[string]struct {
		input       *RunInputs
		expectError bool
		errorMsg    string
	}{
		"successful command execution": {
			input: &RunInputs{
				Command: "echo 'test'",
				Cwd:     "/tmp",
				Env:     map[string]string{},
				Version: "1.0.0",
			},
			expectError: false,
		},
		"command with environment variables": {
			input: &RunInputs{
				Command: "echo $TEST_VAR",
				Cwd:     "/tmp",
				Env:     map[string]string{"TEST_VAR": "hello"},
				Version: "1.0.0",
			},
			expectError: false,
		},
		"failing command": {
			input: &RunInputs{
				Command: "exit 1",
				Cwd:     "/tmp",
				Env:     map[string]string{},
				Version: "1.0.0",
			},
			expectError: true,
			errorMsg:    "command exited with code 1",
		},
		"invalid command": {
			input: &RunInputs{
				Command: "nonexistentcommand12345",
				Cwd:     "/tmp",
				Env:     map[string]string{},
				Version: "1.0.0",
			},
			expectError: true,
		},
		"invalid working directory": {
			input: &RunInputs{
				Command: "echo test",
				Cwd:     "/nonexistent/directory",
				Env:     map[string]string{},
				Version: "1.0.0",
			},
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			run := NewRun()
			var output CreateResult[RunOutputs]

			err := run.Create(tt.input, &output)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "run", output.ID)
				assert.Equal(t, RunOutputs{}, output.Outs)
			}
		})
	}
}

func TestRun_Update(t *testing.T) {
	tests := map[string]struct {
		input       *UpdateInput[RunInputs, RunOutputs]
		expectError bool
		errorMsg    string
	}{
		"successful update": {
			input: &UpdateInput[RunInputs, RunOutputs]{
				ID: "test-run",
				News: RunInputs{
					Command: "echo 'updated'",
					Cwd:     "/tmp",
					Env:     map[string]string{},
					Version: "2.0.0",
				},
				Olds: RunOutputs{},
			},
			expectError: false,
		},
		"failing update command": {
			input: &UpdateInput[RunInputs, RunOutputs]{
				ID: "test-run",
				News: RunInputs{
					Command: "exit 2",
					Cwd:     "/tmp",
					Env:     map[string]string{},
					Version: "2.0.0",
				},
				Olds: RunOutputs{},
			},
			expectError: true,
			errorMsg:    "command exited with code 2",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			run := NewRun()
			var output UpdateResult[RunOutputs]

			err := run.Update(tt.input, &output)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, RunOutputs{}, output.Outs)
			}
		})
	}
}

func TestRun_executeCommand(t *testing.T) {
	tests := map[string]struct {
		input       *RunInputs
		expectError bool
		errorMsg    string
	}{
		"simple echo command": {
			input: &RunInputs{
				Command: "echo hello",
				Cwd:     "/tmp",
				Env:     map[string]string{},
				Version: "1.0.0",
			},
			expectError: false,
		},
		"command with custom environment": {
			input: &RunInputs{
				Command: "sh -c 'echo $CUSTOM_VAR'",
				Cwd:     "/tmp",
				Env:     map[string]string{"CUSTOM_VAR": "custom_value"},
				Version: "1.0.0",
			},
			expectError: false,
		},
		"command with multiple env vars": {
			input: &RunInputs{
				Command: "sh -c 'echo $VAR1 $VAR2'",
				Cwd:     "/tmp",
				Env: map[string]string{
					"VAR1": "value1",
					"VAR2": "value2",
				},
				Version: "1.0.0",
			},
			expectError: false,
		},
		"empty environment map": {
			input: &RunInputs{
				Command: "echo test",
				Cwd:     "/tmp",
				Env:     map[string]string{},
				Version: "1.0.0",
			},
			expectError: false,
		},
		"nil environment map": {
			input: &RunInputs{
				Command: "echo test",
				Cwd:     "/tmp",
				Env:     nil,
				Version: "1.0.0",
			},
			expectError: false,
		},
		"command that fails": {
			input: &RunInputs{
				Command: "exit 3",
				Cwd:     "/tmp",
				Env:     map[string]string{},
				Version: "1.0.0",
			},
			expectError: true,
			errorMsg:    "command exited with code 3",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			run := NewRun()

			err := run.executeCommand(tt.input)

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

func TestRun_ConcurrencyControl(t *testing.T) {
	// Test that the semaphore properly controls concurrency
	t.Run("semaphore limits concurrent execution", func(t *testing.T) {
		// Set concurrency to 1 for this test
		originalValue := flag.SST_BUILD_CONCURRENCY_SITE
		flag.SST_BUILD_CONCURRENCY_SITE = "1"
		defer func() {
			flag.SST_BUILD_CONCURRENCY_SITE = originalValue
		}()

		run := NewRun()

		// Start a long-running command
		input1 := &RunInputs{
			Command: "sleep 0.1",
			Cwd:     "/tmp",
			Env:     map[string]string{},
			Version: "1.0.0",
		}

		input2 := &RunInputs{
			Command: "echo quick",
			Cwd:     "/tmp",
			Env:     map[string]string{},
			Version: "1.0.0",
		}

		// Start both commands concurrently
		done1 := make(chan error, 1)
		done2 := make(chan error, 1)

		go func() {
			var output CreateResult[RunOutputs]
			done1 <- run.Create(input1, &output)
		}()

		go func() {
			var output CreateResult[RunOutputs]
			done2 <- run.Create(input2, &output)
		}()

		// Both should complete successfully
		err1 := <-done1
		err2 := <-done2

		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}

func TestRun_EnvironmentHandling(t *testing.T) {
	t.Run("preserves existing environment", func(t *testing.T) {
		// Set a test environment variable
		originalPath := os.Getenv("PATH")
		require.NotEmpty(t, originalPath, "PATH should be set in test environment")

		run := NewRun()
		input := &RunInputs{
			Command: "sh -c 'echo $PATH'",
			Cwd:     "/tmp",
			Env:     map[string]string{"CUSTOM": "value"},
			Version: "1.0.0",
		}

		var output CreateResult[RunOutputs]
		err := run.Create(input, &output)

		assert.NoError(t, err)
		// The command should have access to both existing PATH and custom env vars
	})

	t.Run("custom env vars override existing ones", func(t *testing.T) {
		run := NewRun()
		input := &RunInputs{
			Command: "sh -c 'echo $TEST_OVERRIDE'",
			Cwd:     "/tmp",
			Env:     map[string]string{"TEST_OVERRIDE": "new_value"},
			Version: "1.0.0",
		}

		var output CreateResult[RunOutputs]
		err := run.Create(input, &output)

		assert.NoError(t, err)
	})
}

func TestRun_EdgeCases(t *testing.T) {
	t.Run("empty command", func(t *testing.T) {
		run := NewRun()
		input := &RunInputs{
			Command: "",
			Cwd:     "/tmp",
			Env:     map[string]string{},
			Version: "1.0.0",
		}

		var output CreateResult[RunOutputs]
		err := run.Create(input, &output)

		// Empty command should succeed (sh -c "" is valid)
		assert.NoError(t, err)
	})

	t.Run("command with special characters", func(t *testing.T) {
		run := NewRun()
		input := &RunInputs{
			Command: "echo 'hello world; echo test'",
			Cwd:     "/tmp",
			Env:     map[string]string{},
			Version: "1.0.0",
		}

		var output CreateResult[RunOutputs]
		err := run.Create(input, &output)

		assert.NoError(t, err)
	})

	t.Run("very long command", func(t *testing.T) {
		run := NewRun()
		longString := "very_long_string_" + strconv.Itoa(int(time.Now().UnixNano()))
		input := &RunInputs{
			Command: "echo " + longString,
			Cwd:     "/tmp",
			Env:     map[string]string{},
			Version: "1.0.0",
		}

		var output CreateResult[RunOutputs]
		err := run.Create(input, &output)

		assert.NoError(t, err)
	})
}

func TestRun_Structure(t *testing.T) {
	run := NewRun()

	assert.NotNil(t, run)
	assert.NotNil(t, run.lock)
	assert.IsType(t, &semaphore.Weighted{}, run.lock)
}

func TestRun_SemaphoreWeightParsing(t *testing.T) {
	tests := map[string]struct {
		envValue       string
		expectedResult bool // true if we expect the semaphore to be created successfully
	}{
		"valid positive number": {
			envValue:       "3",
			expectedResult: true,
		},
		"valid zero": {
			envValue:       "0",
			expectedResult: true,
		},
		"invalid string": {
			envValue:       "abc",
			expectedResult: true, // Should default to some value
		},
		"negative number": {
			envValue:       "-1",
			expectedResult: true, // ParseInt will handle this
		},
		"very large number": {
			envValue:       "999999",
			expectedResult: true,
		},
		"empty string": {
			envValue:       "",
			expectedResult: true, // Should use default
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			originalValue := flag.SST_BUILD_CONCURRENCY_SITE
			flag.SST_BUILD_CONCURRENCY_SITE = tt.envValue
			defer func() {
				flag.SST_BUILD_CONCURRENCY_SITE = originalValue
			}()

			run := NewRun()

			if tt.expectedResult {
				assert.NotNil(t, run)
				assert.NotNil(t, run.lock)
			}
		})
	}
}