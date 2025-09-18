package python

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

// TestBackwardCompatibility verifies that existing project patterns continue to work
// after the runtime simplification changes. This is a comprehensive regression test
// suite that covers all the major project layouts that users have been using.
func TestBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name           string
		description    string
		setupProject   func(t *testing.T, projectDir string) (handlerPath, expectedHandler string)
		expectSuccess  bool
		validateOutput func(t *testing.T, output *runtime.BuildOutput)
	}{
		{
			name:        "flat_layout_project",
			description: "Flat layout with handler.py in root directory",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create handler.py in root
				handlerContent := `
import json

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from flat layout'})
    }
`
				handlerPath := filepath.Join(projectDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				// Create requirements.txt
				requirementsContent := "requests>=2.31.0\n"
				requirementsPath := filepath.Join(projectDir, "requirements.txt")
				if err := os.WriteFile(requirementsPath, []byte(requirementsContent), 0644); err != nil {
					t.Fatalf("Failed to create requirements.txt: %v", err)
				}

				return "handler.py", "handler.lambda_handler"
			},
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				if !strings.Contains(output.Handler, "handler") {
					t.Errorf("Expected handler to reference 'handler' module, got: %s", output.Handler)
				}
			},
		},
		{
			name:        "workspace_layout_project",
			description: "Workspace layout with src/package structure",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "workspace-example"
version = "0.1.0"
dependencies = ["requests>=2.31.0"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["src/mypackage"]
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create src/mypackage directory
				packageDir := filepath.Join(projectDir, "src", "mypackage")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}

				// Create __init__.py
				initPath := filepath.Join(packageDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				// Create handler
				handlerContent := `
import json

def api_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from workspace layout'})
    }
`
				handlerPath := filepath.Join(packageDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "src/mypackage/handler.py", "src/mypackage/handler.api_handler"
			},
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				if !strings.Contains(output.Handler, "mypackage.handler") {
					t.Errorf("Expected handler to reference 'mypackage.handler' module, got: %s", output.Handler)
				}
			},
		},
		{
			name:        "nested_layout_project",
			description: "Nested layout with app/functions structure",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "nested-example"
version = "0.1.0"
dependencies = ["fastapi>=0.104.0"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["app"]
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create nested directory structure
				handlerDir := filepath.Join(projectDir, "app", "functions", "api")
				if err := os.MkdirAll(handlerDir, 0755); err != nil {
					t.Fatalf("Failed to create handler directory: %v", err)
				}

				// Create __init__.py files
				initPaths := []string{
					filepath.Join(projectDir, "app", "__init__.py"),
					filepath.Join(projectDir, "app", "functions", "__init__.py"),
					filepath.Join(handlerDir, "__init__.py"),
				}
				for _, initPath := range initPaths {
					if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handler
				handlerContent := `
import json

def main(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from nested layout'})
    }
`
				handlerPath := filepath.Join(handlerDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "app/functions/api/handler.py", "app/functions/api/handler.main"
			},
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				if !strings.Contains(output.Handler, "app.functions.api.handler") {
					t.Errorf("Expected handler to reference 'app.functions.api.handler' module, got: %s", output.Handler)
				}
			},
		},
		{
			name:        "poetry_project",
			description: "Poetry project with pyproject.toml configuration",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create pyproject.toml with Poetry configuration
				pyprojectContent := `
[tool.poetry]
name = "poetry-project"
version = "0.1.0"
description = "A Poetry project"

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.31.0"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create handler
				handlerContent := `
import json

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from Poetry project'})
    }
`
				handlerPath := filepath.Join(projectDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "handler.py", "handler.lambda_handler"
			},
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				if !strings.Contains(output.Handler, "handler") {
					t.Errorf("Expected handler to reference 'handler' module, got: %s", output.Handler)
				}
			},
		},
		{
			name:        "monorepo_workspace_project",
			description: "Monorepo with multiple services and uv workspace",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create root pyproject.toml with workspace configuration
				pyprojectContent := `
[project]
name = "monorepo"
version = "0.1.0"

[tool.uv.workspace]
members = ["services/*"]
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create service directory
				serviceDir := filepath.Join(projectDir, "services", "user-service")
				if err := os.MkdirAll(serviceDir, 0755); err != nil {
					t.Fatalf("Failed to create service directory: %v", err)
				}

				// Create service pyproject.toml
				servicePyprojectContent := `
[project]
name = "user-service"
version = "0.1.0"
dependencies = ["boto3>=1.34.0"]
`
				servicePyprojectPath := filepath.Join(serviceDir, "pyproject.toml")
				if err := os.WriteFile(servicePyprojectPath, []byte(servicePyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create service pyproject.toml: %v", err)
				}

				// Create handler
				handlerContent := `
import json

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from monorepo service'})
    }
`
				handlerPath := filepath.Join(serviceDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "services/user-service/handler.py", "services/user-service/handler.lambda_handler"
			},
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				if !strings.Contains(output.Handler, "handler") {
					t.Errorf("Expected handler to reference 'handler' module, got: %s", output.Handler)
				}
			},
		},
		{
			name:        "aws_python_example_structure",
			description: "Structure matching the aws-python example",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create root pyproject.toml
				pyprojectContent := `
[project]
name = "aws-python"
version = "0.1.0"

[tool.uv.workspace]
members = ["core", "functions"]
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create core package
				coreDir := filepath.Join(projectDir, "core")
				if err := os.MkdirAll(coreDir, 0755); err != nil {
					t.Fatalf("Failed to create core directory: %v", err)
				}

				corePyprojectContent := `
[project]
name = "core"
version = "0.1.0"
dependencies = []
`
				corePyprojectPath := filepath.Join(coreDir, "pyproject.toml")
				if err := os.WriteFile(corePyprojectPath, []byte(corePyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create core pyproject.toml: %v", err)
				}

				// Create functions package
				functionsDir := filepath.Join(projectDir, "functions")
				if err := os.MkdirAll(functionsDir, 0755); err != nil {
					t.Fatalf("Failed to create functions directory: %v", err)
				}

				functionsPyprojectContent := `
[project]
name = "functions"
version = "0.1.0"
dependencies = ["core"]
`
				functionsPyprojectPath := filepath.Join(functionsDir, "pyproject.toml")
				if err := os.WriteFile(functionsPyprojectPath, []byte(functionsPyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create functions pyproject.toml: %v", err)
				}

				// Create functions source structure
				functionsSrcDir := filepath.Join(functionsDir, "src", "functions")
				if err := os.MkdirAll(functionsSrcDir, 0755); err != nil {
					t.Fatalf("Failed to create functions src directory: %v", err)
				}

				// Create __init__.py
				initPath := filepath.Join(functionsSrcDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				// Create handler
				handlerContent := `
import json

def handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from aws-python example'})
    }
`
				handlerPath := filepath.Join(functionsSrcDir, "api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "functions/src/functions/api.py", "functions/src/functions/api.handler"
			},
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				if !strings.Contains(output.Handler, "functions.api") {
					t.Errorf("Expected handler to reference 'functions.api' module, got: %s", output.Handler)
				}
			},
		},
		{
			name:        "fastapi_example_structure",
			description: "Structure matching the aws-fastapi example",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create root pyproject.toml
				pyprojectContent := `
[project]
name = "aws-fastapi"
version = "0.1.0"

[tool.uv.workspace]
members = ["core", "functions"]
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create functions package
				functionsDir := filepath.Join(projectDir, "functions")
				if err := os.MkdirAll(functionsDir, 0755); err != nil {
					t.Fatalf("Failed to create functions directory: %v", err)
				}

				functionsPyprojectContent := `
[project]
name = "functions"
version = "0.1.0"
dependencies = ["fastapi>=0.104.0", "core"]
`
				functionsPyprojectPath := filepath.Join(functionsDir, "pyproject.toml")
				if err := os.WriteFile(functionsPyprojectPath, []byte(functionsPyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create functions pyproject.toml: %v", err)
				}

				// Create functions source structure
				functionsSrcDir := filepath.Join(functionsDir, "src", "functions")
				if err := os.MkdirAll(functionsSrcDir, 0755); err != nil {
					t.Fatalf("Failed to create functions src directory: %v", err)
				}

				// Create __init__.py
				initPath := filepath.Join(functionsSrcDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				// Create handler
				handlerContent := `
import json
from fastapi import FastAPI

app = FastAPI()

def handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from FastAPI example'})
    }
`
				handlerPath := filepath.Join(functionsSrcDir, "api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "functions/src/functions/api.py", "functions/src/functions/api.handler"
			},
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				if !strings.Contains(output.Handler, "functions.api") {
					t.Errorf("Expected handler to reference 'functions.api' module, got: %s", output.Handler)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project directory: %v", err)
			}

			// Setup project structure
			_, expectedHandler := tt.setupProject(t, projectDir)

			// Test ProjectResolver can handle this structure
			resolver := NewProjectResolver(projectDir)
			projectInfo, err := resolver.ResolveHandler(expectedHandler)

			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("Failed to resolve handler for %s: %v", tt.description, err)
				}

				// Verify basic properties
				if projectInfo.ProjectRoot != projectDir {
					t.Errorf("Expected project root %s, got %s", projectDir, projectInfo.ProjectRoot)
				}

				if len(projectInfo.PythonPath) == 0 {
					t.Error("Expected non-empty Python path")
				}

				// Test that we can build the project
				pipeline, err := NewBuildPipeline(BuildPipelineConfig{
					ProjectRoot:             projectDir,
					ArtifactDir:             filepath.Join(tempDir, "artifacts"),
					EnableCaching:           false,
					EnableProgressReporting: false,
				})
				if err != nil {
					t.Fatalf("Failed to create build pipeline: %v", err)
				}

				// Create build input
				input := createBuildInputForBackwardCompatibilityTest(t, tempDir, expectedHandler)

				// Execute build
				ctx := context.Background()
				output, err := pipeline.Build(ctx, input)

				if err != nil {
					t.Fatalf("Build failed for %s: %v", tt.description, err)
				}

				if output == nil {
					t.Fatal("Expected build output but got nil")
				}

				// Verify basic output properties
				if output.Out == "" {
					t.Error("Expected output directory to be set")
				}

				if output.Handler == "" {
					t.Error("Expected handler to be set")
				}

				// Verify output directory exists
				if _, err := os.Stat(output.Out); os.IsNotExist(err) {
					t.Error("Output directory was not created")
				}

				// Run custom validation
				if tt.validateOutput != nil {
					tt.validateOutput(t, output)
				}

				t.Logf("✅ %s: Build successful - handler=%s, out=%s", tt.description, output.Handler, output.Out)
			} else {
				if err == nil {
					t.Errorf("Expected build to fail for %s but it succeeded", tt.description)
				}
				t.Logf("❌ %s: Build failed as expected - %v", tt.description, err)
			}
		})
	}
}

// TestExistingExampleProjects tests that the actual example projects in the repository
// continue to work with the simplified runtime
func TestExistingExampleProjects(t *testing.T) {
	// Get the repository root
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skipf("Skipping example project tests: %v", err)
	}

	exampleTests := []struct {
		name         string
		examplePath  string
		handlerSpecs []struct {
			handler        string
			expectSuccess  bool
			expectedModule string
		}
	}{
		{
			name:        "python-layouts-flat",
			examplePath: "examples/python-layouts/flat-layout",
			handlerSpecs: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"handler.main", true, "handler"},
				{"handler.worker", true, "handler"},
			},
		},
		{
			name:        "python-layouts-workspace",
			examplePath: "examples/python-layouts/workspace-layout",
			handlerSpecs: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"src/mypackage/handler.api_handler", true, "mypackage.handler"},
				{"src/mypackage/handler.worker_handler", true, "mypackage.handler"},
			},
		},
		{
			name:        "python-layouts-nested",
			examplePath: "examples/python-layouts/nested-layout",
			handlerSpecs: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"app/functions/api/handler.main", true, "app.functions.api.handler"},
				{"app/functions/worker/handler.main", true, "app.functions.worker.handler"},
				{"app/functions/auth/handler.main", true, "app.functions.auth.handler"},
			},
		},
		{
			name:        "aws-python",
			examplePath: "examples/aws-python",
			handlerSpecs: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"functions/src/functions/api.handler", true, "functions.api"},
			},
		},
		{
			name:        "aws-fastapi",
			examplePath: "examples/aws-fastapi",
			handlerSpecs: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"functions/src/functions/api.handler", true, "functions.api"},
			},
		},
	}

	for _, tt := range exampleTests {
		t.Run(tt.name, func(t *testing.T) {
			exampleDir := filepath.Join(repoRoot, tt.examplePath)

			// Check if example directory exists
			if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
				t.Skipf("Example directory does not exist: %s", exampleDir)
			}

			// Test each handler specification
			for _, handlerSpec := range tt.handlerSpecs {
				t.Run(handlerSpec.handler, func(t *testing.T) {
					// Test ProjectResolver can handle this structure
					resolver := NewProjectResolver(exampleDir)
					projectInfo, err := resolver.ResolveHandler(handlerSpec.handler)

					if handlerSpec.expectSuccess {
						if err != nil {
							t.Fatalf("Failed to resolve handler %s in %s: %v", handlerSpec.handler, tt.name, err)
						}

						// Verify the module path contains expected components
						if !strings.Contains(projectInfo.ModulePath, handlerSpec.expectedModule) {
							t.Errorf("Expected module path to contain %s, got %s", handlerSpec.expectedModule, projectInfo.ModulePath)
						}

						// Verify basic properties
						if projectInfo.ProjectRoot != exampleDir {
							t.Errorf("Expected project root %s, got %s", exampleDir, projectInfo.ProjectRoot)
						}

						t.Logf("✅ %s/%s: Successfully resolved - module=%s", tt.name, handlerSpec.handler, projectInfo.ModulePath)
					} else {
						if err == nil {
							t.Errorf("Expected handler resolution to fail for %s in %s but it succeeded", handlerSpec.handler, tt.name)
						}
						t.Logf("❌ %s/%s: Failed as expected - %v", tt.name, handlerSpec.handler, err)
					}
				})
			}
		})
	}
}

// TestRegressionScenarios tests specific scenarios that have been problematic in the past
func TestRegressionScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		description string
		setupFunc   func(t *testing.T, projectDir string) (handlerPath string)
		expectError bool
		errorCheck  func(t *testing.T, err error)
	}{
		{
			name:        "missing_init_files",
			description: "Package structure without __init__.py files",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create nested structure without __init__.py files
				handlerDir := filepath.Join(projectDir, "app", "handlers")
				if err := os.MkdirAll(handlerDir, 0755); err != nil {
					t.Fatalf("Failed to create handler directory: %v", err)
				}

				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'OK'}
`
				handlerPath := filepath.Join(handlerDir, "api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "app/handlers/api.lambda_handler"
			},
			expectError: false, // Should work even without __init__.py files
			errorCheck:  nil,
		},
		{
			name:        "handler_in_subdirectory_without_pyproject",
			description: "Handler in subdirectory without pyproject.toml",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create subdirectory structure
				subDir := filepath.Join(projectDir, "functions")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatalf("Failed to create subdirectory: %v", err)
				}

				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'OK'}
`
				handlerPath := filepath.Join(subDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "functions/handler.lambda_handler"
			},
			expectError: false, // Should work without pyproject.toml
			errorCheck:  nil,
		},
		{
			name:        "deeply_nested_handler",
			description: "Handler deeply nested in directory structure",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create deeply nested structure
				deepDir := filepath.Join(projectDir, "src", "app", "services", "api", "v1", "handlers")
				if err := os.MkdirAll(deepDir, 0755); err != nil {
					t.Fatalf("Failed to create deep directory: %v", err)
				}

				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Deep handler'}
`
				handlerPath := filepath.Join(deepDir, "users.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "src/app/services/api/v1/handlers/users.lambda_handler"
			},
			expectError: false, // Should handle deep nesting
			errorCheck:  nil,
		},
		{
			name:        "handler_with_dashes_in_path",
			description: "Handler in directory with dashes (common in real projects)",
			setupFunc: func(t *testing.T, projectDir string) string {
				// Create directory with dashes
				dashDir := filepath.Join(projectDir, "user-service", "api-handlers")
				if err := os.MkdirAll(dashDir, 0755); err != nil {
					t.Fatalf("Failed to create dash directory: %v", err)
				}

				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Dash handler'}
`
				handlerPath := filepath.Join(dashDir, "user_api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "user-service/api-handlers/user_api.lambda_handler"
			},
			expectError: false, // Should handle dashes in directory names
			errorCheck:  nil,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create temporary directory
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project directory: %v", err)
			}

			// Setup scenario
			handlerPath := scenario.setupFunc(t, projectDir)

			// Test ProjectResolver
			resolver := NewProjectResolver(projectDir)
			projectInfo, err := resolver.ResolveHandler(handlerPath)

			if scenario.expectError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", scenario.description)
				} else if scenario.errorCheck != nil {
					scenario.errorCheck(t, err)
				}
				t.Logf("❌ %s: Failed as expected - %v", scenario.description, err)
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for %s: %v", scenario.description, err)
				}

				// Verify basic properties
				if projectInfo.ProjectRoot != projectDir {
					t.Errorf("Expected project root %s, got %s", projectDir, projectInfo.ProjectRoot)
				}

				t.Logf("✅ %s: Successfully handled - module=%s", scenario.description, projectInfo.ModulePath)
			}
		})
	}
}

// createBuildInputForBackwardCompatibilityTest creates a build input for backward compatibility testing
func createBuildInputForBackwardCompatibilityTest(t *testing.T, tempDir, handler string) *runtime.BuildInput {
	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	properties := map[string]interface{}{
		"architecture": "x86_64",
		"container":    false,
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		t.Fatalf("Failed to marshal properties: %v", err)
	}

	return &runtime.BuildInput{
		FunctionID: "backward-compatibility-test-function",
		Handler:    handler,
		Runtime:    "python3.12",
		Properties: propertiesJSON,
	}
}

// findRepoRoot finds the repository root by looking for go.mod
func findRepoRoot() (string, error) {
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
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
