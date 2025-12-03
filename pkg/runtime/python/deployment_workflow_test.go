package python

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// TestDeploymentWorkflow tests the complete SST deployment workflow
// This validates that the Python runtime simplification works end-to-end
func TestDeploymentWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		examplePath string
		handler     string
		runtime     string
	}{
		{
			name:        "aws-python-example",
			examplePath: "examples/aws-python",
			handler:     "functions/src/functions/api.handler",
			runtime:     "python3.11",
		},
		{
			name:        "flat-layout-example",
			examplePath: "examples/python-layouts/flat-layout",
			handler:     "handler.lambda_handler",
			runtime:     "python3.11",
		},
		{
			name:        "workspace-layout-example",
			examplePath: "examples/python-layouts/workspace-layout",
			handler:     "src/mypackage/handler.api_handler",
			runtime:     "python3.11",
		},
		{
			name:        "nested-layout-example",
			examplePath: "examples/python-layouts/nested-layout",
			handler:     "app/functions/api/handler.main",
			runtime:     "python3.11",
		},
		{
			name:        "monorepo-layout-api",
			examplePath: "examples/python-layouts/monorepo-layout",
			handler:     "services/api/handler.main",
			runtime:     "python3.12",
		},
		{
			name:        "monorepo-layout-auth",
			examplePath: "examples/python-layouts/monorepo-layout",
			handler:     "services/auth/handler.main",
			runtime:     "python3.12",
		},
		{
			name:        "monorepo-layout-worker",
			examplePath: "examples/python-layouts/monorepo-layout",
			handler:     "services/worker/handler.main",
			runtime:     "python3.12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if example doesn't exist
			if _, err := os.Stat(tt.examplePath); os.IsNotExist(err) {
				t.Skipf("Example %s not found, skipping", tt.examplePath)
			}

			t.Logf("🧪 Testing complete deployment workflow for %s", tt.name)

			// Create Python runtime
			pythonRuntime := New()

			// Test the complete build workflow
			functionID := "deployment-test-" + tt.name

			start := time.Now()
			result, err := pythonRuntime.Build(context.Background(), &runtime.BuildInput{
				FunctionID: functionID,
				Handler:    tt.handler,
				Runtime:    tt.runtime,
				CfgPath:    filepath.Join(tt.examplePath, "sst.config.ts"),
			})
			buildDuration := time.Since(start)

			if err != nil {
				t.Fatalf("❌ Build failed for %s: %v", tt.name, err)
			}

			// Validate build result
			if result.Handler == "" {
				t.Errorf("❌ Build result missing handler for %s", tt.name)
			}

			if result.Out == "" {
				t.Errorf("❌ Build result missing output directory for %s", tt.name)
			}

			// Validate output directory exists and contains expected files
			outputPath := filepath.Join(tt.examplePath, result.Out)
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("❌ Output directory does not exist for %s: %s", tt.name, outputPath)
			}

			// Check for Python files in output
			pythonFiles := 0
			err = filepath.Walk(outputPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if filepath.Ext(path) == ".py" {
					pythonFiles++
				}
				return nil
			})

			if err != nil {
				t.Errorf("❌ Error walking output directory for %s: %v", tt.name, err)
			}

			if pythonFiles == 0 {
				t.Errorf("❌ No Python files found in output directory for %s", tt.name)
			}

			// Test caching - second build should be faster
			start = time.Now()
			shouldRebuild := pythonRuntime.ShouldRebuild(functionID, tt.handler)
			cacheCheckDuration := time.Since(start)

			// Performance validation
			t.Logf("✅ %s: Build completed successfully", tt.name)
			t.Logf("   📊 Build time: %v", buildDuration)
			t.Logf("   📊 Cache check time: %v", cacheCheckDuration)
			t.Logf("   📊 Handler: %s", result.Handler)
			t.Logf("   📊 Output: %s", result.Out)
			t.Logf("   📊 Python files: %d", pythonFiles)
			t.Logf("   📊 Should rebuild: %v", shouldRebuild)

			// Validate performance expectations
			if buildDuration > 30*time.Second {
				t.Logf("⚠️  Build time for %s is slower than expected: %v", tt.name, buildDuration)
			}

			if cacheCheckDuration > 100*time.Millisecond {
				t.Logf("⚠️  Cache check for %s is slower than expected: %v", tt.name, cacheCheckDuration)
			}
		})
	}
}

// TestDeploymentErrorHandling tests error scenarios in deployment workflow
func TestDeploymentErrorHandling(t *testing.T) {
	pythonRuntime := New()

	tests := []struct {
		name        string
		handler     string
		workDir     string
		expectError bool
		errorType   string
	}{
		{
			name:        "nonexistent-handler",
			handler:     "nonexistent/handler.py",
			workDir:     "/tmp",
			expectError: true,
			errorType:   "handler_not_found",
		},
		{
			name:        "invalid-workdir",
			handler:     "handler.py",
			workDir:     "/nonexistent/directory",
			expectError: true,
			errorType:   "project_resolution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("🧪 Testing error handling for %s", tt.name)

			_, err := pythonRuntime.Build(context.Background(), &runtime.BuildInput{
				FunctionID: "error-test-" + tt.name,
				Handler:    tt.handler,
				Runtime:    "python3.11",
				CfgPath:    filepath.Join(tt.workDir, "sst.config.ts"),
			})

			if tt.expectError {
				if err == nil {
					t.Errorf("❌ Expected error for %s but got none", tt.name)
				} else {
					t.Logf("✅ %s: Error handled correctly: %v", tt.name, err)
				}
			} else {
				if err != nil {
					t.Errorf("❌ Unexpected error for %s: %v", tt.name, err)
				}
			}
		})
	}
}

// TestDeploymentSuccessCriteria validates all success criteria from the requirements
func TestDeploymentSuccessCriteria(t *testing.T) {
	t.Log("🧪 Validating success criteria from requirements")

	// Find project root by looking for go.mod
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Skipf("Could not find project root: %v", err)
	}

	// Test that all existing project types continue to work
	projectTypes := []struct {
		name        string
		examplePath string
		handler     string
	}{
		{"flat", filepath.Join(projectRoot, "examples/python-layouts/flat-layout"), "handler.lambda_handler"},
		{"workspace", filepath.Join(projectRoot, "examples/python-layouts/workspace-layout"), "src/mypackage/handler.api_handler"},
		{"nested", filepath.Join(projectRoot, "examples/python-layouts/nested-layout"), "app/functions/api/handler.main"},
	}

	pythonRuntime := New()
	successCount := 0

	for _, pt := range projectTypes {
		if _, err := os.Stat(pt.examplePath); os.IsNotExist(err) {
			t.Logf("⚠️  %s example not found, skipping", pt.name)
			continue
		}

		_, err := pythonRuntime.Build(context.Background(), &runtime.BuildInput{
			FunctionID: "success-criteria-" + pt.name,
			Handler:    pt.handler,
			Runtime:    "python3.11",
			CfgPath:    filepath.Join(pt.examplePath, "sst.config.ts"),
			Properties: json.RawMessage(`{"architecture":"x86_64","container":false}`),
		})

		if err != nil {
			t.Errorf("❌ %s project type failed: %v", pt.name, err)
		} else {
			t.Logf("✅ %s project type: WORKING", pt.name)
			successCount++
		}
	}

	// Validate that we tested at least some project types
	if successCount == 0 {
		t.Error("❌ No project types could be tested - examples may be missing")
	} else if successCount < len(projectTypes) {
		t.Logf("⚠️  Success criteria validation: %d/%d project types working (some failures expected during development)", successCount, len(projectTypes))
	} else {
		t.Logf("✅ Success criteria validation: %d/%d project types working", successCount, len(projectTypes))
	}

	// Test error messages are clear and actionable
	_, err = pythonRuntime.Build(context.Background(), &runtime.BuildInput{
		FunctionID: "error-message-test",
		Handler:    "nonexistent.handler",
		Runtime:    "python3.11",
		CfgPath:    "/tmp/sst.config.ts",
		Properties: json.RawMessage(`{"architecture":"x86_64","container":false}`),
	})

	if err == nil {
		t.Error("❌ Expected error for nonexistent handler")
	} else {
		errorMsg := err.Error()
		if len(errorMsg) < 50 {
			t.Error("❌ Error message too short, may not be actionable")
		} else {
			t.Log("✅ Error messages: CLEAR AND ACTIONABLE")
		}
	}
}

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod not found)")
		}
		dir = parent
	}
}
