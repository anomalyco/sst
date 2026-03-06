package python

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

// TestDeploymentWorkflow tests the complete build workflow for key project structures.
// We test flat (simplest) and monorepo (most complex) to cover the main code paths
// without running 10+ uv builds.
func TestDeploymentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Skipf("Could not find project root: %v", err)
	}

	tests := []struct {
		name        string
		examplePath string
		handler     string
		runtime     string
	}{
		{
			name:        "flat-layout",
			examplePath: "examples/python-layouts/flat-layout",
			handler:     "handler.lambda_handler",
			runtime:     "python3.11",
		},
		{
			name:        "monorepo-layout-worker",
			examplePath: "examples/python-layouts/monorepo-layout",
			handler:     "services/worker/handler.main",
			runtime:     "python3.12",
		},
	}

	pythonRuntime := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exampleAbsPath := filepath.Join(projectRoot, tt.examplePath)
			if _, err := os.Stat(exampleAbsPath); os.IsNotExist(err) {
				t.Skipf("Example %s not found", exampleAbsPath)
			}

			result, err := pythonRuntime.Build(context.Background(), &runtime.BuildInput{
				FunctionID: "deployment-test-" + tt.name,
				Handler:    tt.handler,
				Runtime:    tt.runtime,
				Properties: json.RawMessage(`{"architecture": "x86_64", "container": false}`),
				CfgPath:    filepath.Join(exampleAbsPath, "sst.config.ts"),
			})
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			if result.Handler == "" {
				t.Error("Build result missing handler")
			}
			if result.Out == "" {
				t.Error("Build result missing output directory")
			}

			// Check output has Python files
			pythonFiles := 0
			filepath.Walk(result.Out, func(path string, info os.FileInfo, err error) error {
				if err == nil && filepath.Ext(path) == ".py" {
					pythonFiles++
				}
				return nil
			})
			if pythonFiles == 0 {
				t.Error("No Python files in output directory")
			}
		})
	}
}

// TestDeploymentErrorHandling tests error scenarios in deployment workflow
func TestDeploymentErrorHandling(t *testing.T) {
	pythonRuntime := New()

	t.Run("nonexistent handler", func(t *testing.T) {
		_, err := pythonRuntime.Build(context.Background(), &runtime.BuildInput{
			FunctionID: "error-test",
			Handler:    "nonexistent/handler.py",
			Runtime:    "python3.11",
			CfgPath:    "/tmp/sst.config.ts",
		})
		if err == nil {
			t.Error("Expected error for nonexistent handler")
		}
	})

	t.Run("invalid workdir", func(t *testing.T) {
		_, err := pythonRuntime.Build(context.Background(), &runtime.BuildInput{
			FunctionID: "error-test-2",
			Handler:    "handler.py",
			Runtime:    "python3.11",
			CfgPath:    "/nonexistent/directory/sst.config.ts",
		})
		if err == nil {
			t.Error("Expected error for invalid workdir")
		}
	})
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
