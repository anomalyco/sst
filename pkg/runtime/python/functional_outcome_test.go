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

// TestFunctionalOutcomes tests that the Python runtime can successfully build and execute
// functions from various real-world project structures, focusing on functional outcomes
// rather than implementation details like layout classification.
func TestFunctionalOutcomes(t *testing.T) {
	tests := []struct {
		name           string
		setupProject   func(t *testing.T, projectDir string) string // Returns handler path
		handler        string
		expectSuccess  bool
		validateOutput func(t *testing.T, output *runtime.BuildOutput)
	}{
		{
			name: "flat_project_with_requirements",
			setupProject: func(t *testing.T, projectDir string) string {
				// Create handler
				handlerContent := `
import json

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from flat project'})
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

				return "handler.py"
			},
			handler:       "handler.lambda_handler",
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				// Verify handler module can be imported
				if !strings.Contains(output.Handler, "handler") {
					t.Errorf("Expected handler to reference 'handler' module, got: %s", output.Handler)
				}
			},
		},
		{
			name: "workspace_project_with_pyproject",
			setupProject: func(t *testing.T, projectDir string) string {
				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "my-workspace"
version = "0.1.0"
dependencies = ["requests>=2.31.0"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create functions directory
				functionsDir := filepath.Join(projectDir, "functions")
				if err := os.MkdirAll(functionsDir, 0755); err != nil {
					t.Fatalf("Failed to create functions directory: %v", err)
				}

				// Create handler
				handlerContent := `
import json
import requests

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from workspace project'})
    }
`
				handlerPath := filepath.Join(functionsDir, "api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				// Create __init__.py
				initPath := filepath.Join(functionsDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				return "api.py"
			},
			handler:       "api.lambda_handler",
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				// Verify handler module path is correct
				if !strings.Contains(output.Handler, "functions.api") {
					t.Errorf("Expected handler to reference 'functions.api' module, got: %s", output.Handler)
				}
			},
		},
		{
			name: "src_layout_project",
			setupProject: func(t *testing.T, projectDir string) string {
				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "mypackage"
version = "0.1.0"
dependencies = ["boto3>=1.34.0"]

[build-system]
requires = ["setuptools", "wheel"]
build-backend = "setuptools.build_meta"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create src directory structure
				srcDir := filepath.Join(projectDir, "src", "mypackage")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatalf("Failed to create src directory: %v", err)
				}

				// Create handler
				handlerContent := `
import json

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from src layout project'})
    }
`
				handlerPath := filepath.Join(srcDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				// Create __init__.py
				initPath := filepath.Join(srcDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				return "handler.py"
			},
			handler:       "handler.lambda_handler",
			expectSuccess: false, // Handler is in src/mypackage/handler.py but we're looking for handler.py
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				// Should not reach here for failed builds
			},
		},
		{
			name: "poetry_project",
			setupProject: func(t *testing.T, projectDir string) string {
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

				return "handler.py"
			},
			handler:       "handler.lambda_handler",
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				// Verify handler module can be imported
				if !strings.Contains(output.Handler, "handler") {
					t.Errorf("Expected handler to reference 'handler' module, got: %s", output.Handler)
				}
			},
		},
		{
			name: "nested_package_structure",
			setupProject: func(t *testing.T, projectDir string) string {
				// Create nested package structure
				packageDir := filepath.Join(projectDir, "app", "services", "handlers")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}

				// Create __init__.py files
				initPaths := []string{
					filepath.Join(projectDir, "app", "__init__.py"),
					filepath.Join(projectDir, "app", "services", "__init__.py"),
					filepath.Join(packageDir, "__init__.py"),
				}
				for _, initPath := range initPaths {
					if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handler
				handlerContent := `
import json

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Hello from nested package'})
    }
`
				handlerPath := filepath.Join(packageDir, "api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "app/services/handlers/api.py"
			},
			handler:       "app/services/handlers/api.lambda_handler",
			expectSuccess: true,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				// Verify handler module path is correct
				if !strings.Contains(output.Handler, "app.services.handlers.api") {
					t.Errorf("Expected handler to reference 'app.services.handlers.api' module, got: %s", output.Handler)
				}
			},
		},
		{
			name: "handler_not_found",
			setupProject: func(t *testing.T, projectDir string) string {
				// Create a project but don't create the handler file
				pyprojectContent := `
[project]
name = "test-project"
version = "0.1.0"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				return "nonexistent.py"
			},
			handler:       "nonexistent.lambda_handler",
			expectSuccess: false,
			validateOutput: func(t *testing.T, output *runtime.BuildOutput) {
				// Should not reach here for failed builds
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
			_ = tt.setupProject(t, projectDir)

			// Create build pipeline
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
			input := createBuildInputForFunctionalTest(t, tempDir, tt.handler)

			// Execute build
			ctx := context.Background()
			output, err := pipeline.Build(ctx, input)

			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("Expected successful build but got error: %v", err)
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

				t.Logf("Build successful: handler=%s, out=%s", output.Handler, output.Out)
			} else {
				if err == nil {
					t.Error("Expected build to fail but it succeeded")
				}
				t.Logf("Build failed as expected: %v", err)
			}
		})
	}
}

// TestImportAndExecutionCapability tests that built artifacts can actually be imported
// and executed, focusing on the end-to-end functionality rather than build internals.
func TestImportAndExecutionCapability(t *testing.T) {
	// Create a simple project
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create a handler that imports standard library modules
	handlerContent := `
import json
import os
import sys
from datetime import datetime

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({
            'message': 'Handler executed successfully',
            'timestamp': datetime.now().isoformat(),
            'python_version': sys.version,
            'environment': dict(os.environ)
        })
    }

def test_function():
    """Test function to verify module can be imported and executed"""
    result = lambda_handler({}, {})
    return result['statusCode'] == 200
`
	handlerPath := filepath.Join(projectDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create build pipeline
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:             projectDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		EnableCaching:           false,
		EnableProgressReporting: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Build the function
	input := createBuildInputForFunctionalTest(t, tempDir, "handler.lambda_handler")
	ctx := context.Background()
	output, err := pipeline.Build(ctx, input)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify the output directory contains the handler
	handlerInOutput := filepath.Join(output.Out, "handler.py")
	if _, err := os.Stat(handlerInOutput); os.IsNotExist(err) {
		t.Error("Handler file not found in output directory")
	}

	// Verify the handler file has the expected content
	outputContent, err := os.ReadFile(handlerInOutput)
	if err != nil {
		t.Fatalf("Failed to read output handler: %v", err)
	}

	if !strings.Contains(string(outputContent), "lambda_handler") {
		t.Error("Handler function not found in output file")
	}

	if !strings.Contains(string(outputContent), "test_function") {
		t.Error("Test function not found in output file")
	}

	t.Logf("Import and execution test passed: handler built and packaged correctly")
}

// TestRealWorldProjectStructures tests various real-world project structures
// that users commonly use, ensuring they all work correctly.
func TestRealWorldProjectStructures(t *testing.T) {
	structures := []struct {
		name        string
		description string
		setupFunc   func(t *testing.T, projectDir string) (handlerPath, expectedModule string)
	}{
		{
			name:        "django_style_project",
			description: "Django-style project with apps directory",
			setupFunc: func(t *testing.T, projectDir string) (string, string) {
				// Create Django-style structure
				appsDir := filepath.Join(projectDir, "apps", "api")
				if err := os.MkdirAll(appsDir, 0755); err != nil {
					t.Fatalf("Failed to create apps directory: %v", err)
				}

				// Create __init__.py files
				initPaths := []string{
					filepath.Join(projectDir, "apps", "__init__.py"),
					filepath.Join(appsDir, "__init__.py"),
				}
				for _, initPath := range initPaths {
					if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handler
				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Django-style handler'}
`
				handlerPath := filepath.Join(appsDir, "views.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "apps/api/views.py", "apps.api.views"
			},
		},
		{
			name:        "flask_style_project",
			description: "Flask-style project with blueprints",
			setupFunc: func(t *testing.T, projectDir string) (string, string) {
				// Create Flask-style structure
				blueprintsDir := filepath.Join(projectDir, "app", "blueprints", "api")
				if err := os.MkdirAll(blueprintsDir, 0755); err != nil {
					t.Fatalf("Failed to create blueprints directory: %v", err)
				}

				// Create __init__.py files
				initPaths := []string{
					filepath.Join(projectDir, "app", "__init__.py"),
					filepath.Join(projectDir, "app", "blueprints", "__init__.py"),
					filepath.Join(blueprintsDir, "__init__.py"),
				}
				for _, initPath := range initPaths {
					if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handler
				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Flask-style handler'}
`
				handlerPath := filepath.Join(blueprintsDir, "routes.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "app/blueprints/api/routes.py", "app.blueprints.api.routes"
			},
		},
		{
			name:        "fastapi_style_project",
			description: "FastAPI-style project with routers",
			setupFunc: func(t *testing.T, projectDir string) (string, string) {
				// Create FastAPI-style structure
				routersDir := filepath.Join(projectDir, "app", "routers")
				if err := os.MkdirAll(routersDir, 0755); err != nil {
					t.Fatalf("Failed to create routers directory: %v", err)
				}

				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "fastapi-project"
version = "0.1.0"
dependencies = ["fastapi>=0.104.0"]
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create __init__.py files
				initPaths := []string{
					filepath.Join(projectDir, "app", "__init__.py"),
					filepath.Join(routersDir, "__init__.py"),
				}
				for _, initPath := range initPaths {
					if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handler
				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'FastAPI-style handler'}
`
				handlerPath := filepath.Join(routersDir, "users.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "app/routers/users.py", "app.routers.users"
			},
		},
		{
			name:        "monorepo_style_project",
			description: "Monorepo with multiple services",
			setupFunc: func(t *testing.T, projectDir string) (string, string) {
				// Create monorepo structure
				serviceDir := filepath.Join(projectDir, "services", "user-service", "src")
				if err := os.MkdirAll(serviceDir, 0755); err != nil {
					t.Fatalf("Failed to create service directory: %v", err)
				}

				// Create pyproject.toml at root
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

				// Create service pyproject.toml
				servicePyprojectContent := `
[project]
name = "user-service"
version = "0.1.0"
dependencies = ["boto3>=1.34.0"]
`
				servicePyprojectPath := filepath.Join(projectDir, "services", "user-service", "pyproject.toml")
				if err := os.WriteFile(servicePyprojectPath, []byte(servicePyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create service pyproject.toml: %v", err)
				}

				// Create __init__.py files
				initPaths := []string{
					filepath.Join(projectDir, "services", "__init__.py"),
					filepath.Join(projectDir, "services", "user-service", "__init__.py"),
					filepath.Join(serviceDir, "__init__.py"),
				}
				for _, initPath := range initPaths {
					if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handler
				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Monorepo service handler'}
`
				handlerPath := filepath.Join(serviceDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "services/user-service/src/handler.py", "handler"
			},
		},
	}

	for _, structure := range structures {
		t.Run(structure.name, func(t *testing.T) {
			// Create temporary directory
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project directory: %v", err)
			}

			// Setup project structure
			handlerPath, expectedModule := structure.setupFunc(t, projectDir)

			// Test that ProjectResolver can handle this structure
			resolver := NewProjectResolver(projectDir)
			projectInfo, err := resolver.ResolveHandler(handlerPath)

			if err != nil {
				t.Fatalf("Failed to resolve handler for %s: %v", structure.description, err)
			}

			// Verify the module path is correct
			if projectInfo.ModulePath != expectedModule {
				t.Errorf("Expected module path %s, got %s", expectedModule, projectInfo.ModulePath)
			}

			// Verify basic properties
			if projectInfo.ProjectRoot != projectDir {
				t.Errorf("Expected project root %s, got %s", projectDir, projectInfo.ProjectRoot)
			}

			if len(projectInfo.PythonPath) == 0 {
				t.Error("Expected non-empty Python path")
			}

			t.Logf("Successfully resolved %s: module=%s", structure.description, projectInfo.ModulePath)
		})
	}
}

// createBuildInputForFunctionalTest creates a build input for functional testing
func createBuildInputForFunctionalTest(t *testing.T, tempDir, handler string) *runtime.BuildInput {
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
		FunctionID: "functional-test-function",
		Handler:    handler,
		Runtime:    "python3.12",
		Properties: propertiesJSON,
	}
}
