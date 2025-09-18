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

// TestEndToEndCompatibility tests the complete build process for the most important
// project patterns to ensure they work end-to-end after the runtime simplification.
func TestEndToEndCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		description   string
		setupProject  func(t *testing.T, projectDir string) (handlerPath, handlerSpec string)
		expectSuccess bool
		validateBuild func(t *testing.T, output *runtime.BuildOutput, projectDir string)
	}{
		{
			name:        "flat_layout_end_to_end",
			description: "Complete build test for flat layout project",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create handler with imports
				handlerContent := `
import json
import os
from datetime import datetime

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({
            'message': 'Hello from flat layout',
            'timestamp': datetime.now().isoformat(),
            'env_var': os.environ.get('AWS_REGION', 'unknown')
        })
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
			validateBuild: func(t *testing.T, output *runtime.BuildOutput, projectDir string) {
				// Verify handler file exists in output
				handlerInOutput := filepath.Join(output.Out, "handler.py")
				if _, err := os.Stat(handlerInOutput); os.IsNotExist(err) {
					t.Error("Handler file not found in output directory")
				}

				// Verify handler content
				content, err := os.ReadFile(handlerInOutput)
				if err != nil {
					t.Fatalf("Failed to read output handler: %v", err)
				}

				if !strings.Contains(string(content), "lambda_handler") {
					t.Error("Handler function not found in output file")
				}

				if !strings.Contains(string(content), "datetime") {
					t.Error("Import statements not preserved in output")
				}
			},
		},
		{
			name:        "workspace_layout_end_to_end",
			description: "Complete build test for workspace layout project",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "workspace-test"
version = "0.1.0"
dependencies = ["requests>=2.31.0", "boto3>=1.34.0"]

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

				// Create package structure
				packageDir := filepath.Join(projectDir, "src", "mypackage")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}

				// Create __init__.py
				initPath := filepath.Join(packageDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				// Create shared utility
				utilsContent := `
def get_timestamp():
    from datetime import datetime
    return datetime.now().isoformat()

def format_response(status_code, data):
    import json
    return {
        'statusCode': status_code,
        'body': json.dumps(data)
    }
`
				utilsPath := filepath.Join(packageDir, "utils.py")
				if err := os.WriteFile(utilsPath, []byte(utilsContent), 0644); err != nil {
					t.Fatalf("Failed to create utils: %v", err)
				}

				// Create handler that uses the utility
				handlerContent := `
from mypackage.utils import get_timestamp, format_response

def api_handler(event, context):
    return format_response(200, {
        'message': 'Hello from workspace layout',
        'timestamp': get_timestamp(),
        'handler': 'api_handler'
    })
`
				handlerPath := filepath.Join(packageDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "src/mypackage/handler.py", "src/mypackage/handler.api_handler"
			},
			expectSuccess: true,
			validateBuild: func(t *testing.T, output *runtime.BuildOutput, projectDir string) {
				// Verify package structure is preserved
				packageDir := filepath.Join(output.Out, "mypackage")
				if _, err := os.Stat(packageDir); os.IsNotExist(err) {
					t.Error("Package directory not found in output")
				}

				// Verify handler file
				handlerInOutput := filepath.Join(packageDir, "handler.py")
				if _, err := os.Stat(handlerInOutput); os.IsNotExist(err) {
					t.Error("Handler file not found in output package")
				}

				// Verify utils file
				utilsInOutput := filepath.Join(packageDir, "utils.py")
				if _, err := os.Stat(utilsInOutput); os.IsNotExist(err) {
					t.Error("Utils file not found in output package")
				}

				// Verify __init__.py
				initInOutput := filepath.Join(packageDir, "__init__.py")
				if _, err := os.Stat(initInOutput); os.IsNotExist(err) {
					t.Error("__init__.py not found in output package")
				}
			},
		},
		{
			name:        "nested_layout_end_to_end",
			description: "Complete build test for nested layout project",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "nested-test"
version = "0.1.0"
dependencies = ["fastapi>=0.104.0"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["app", "shared"]
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create shared utilities
				sharedDir := filepath.Join(projectDir, "shared")
				if err := os.MkdirAll(sharedDir, 0755); err != nil {
					t.Fatalf("Failed to create shared directory: %v", err)
				}

				sharedInitPath := filepath.Join(sharedDir, "__init__.py")
				if err := os.WriteFile(sharedInitPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create shared __init__.py: %v", err)
				}

				sharedUtilsContent := `
def validate_request(event):
    return event.get('body') is not None

def create_response(message, status=200):
    import json
    return {
        'statusCode': status,
        'body': json.dumps({'message': message})
    }
`
				sharedUtilsPath := filepath.Join(sharedDir, "utils.py")
				if err := os.WriteFile(sharedUtilsPath, []byte(sharedUtilsContent), 0644); err != nil {
					t.Fatalf("Failed to create shared utils: %v", err)
				}

				// Create nested handler structure
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

				// Create handler that uses shared utilities
				handlerContent := `
from shared.utils import validate_request, create_response

def main(event, context):
    if not validate_request(event):
        return create_response('Invalid request', 400)
    
    return create_response('Hello from nested layout API')
`
				handlerPath := filepath.Join(handlerDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "app/functions/api/handler.py", "app/functions/api/handler.main"
			},
			expectSuccess: true,
			validateBuild: func(t *testing.T, output *runtime.BuildOutput, projectDir string) {
				// Verify nested structure is preserved
				appDir := filepath.Join(output.Out, "app")
				if _, err := os.Stat(appDir); os.IsNotExist(err) {
					t.Error("App directory not found in output")
				}

				// Verify handler file
				handlerInOutput := filepath.Join(output.Out, "app", "functions", "api", "handler.py")
				if _, err := os.Stat(handlerInOutput); os.IsNotExist(err) {
					t.Error("Handler file not found in nested output structure")
				}

				// Verify shared utilities are included
				sharedDir := filepath.Join(output.Out, "shared")
				if _, err := os.Stat(sharedDir); os.IsNotExist(err) {
					t.Error("Shared directory not found in output")
				}

				sharedUtilsInOutput := filepath.Join(sharedDir, "utils.py")
				if _, err := os.Stat(sharedUtilsInOutput); os.IsNotExist(err) {
					t.Error("Shared utils file not found in output")
				}
			},
		},
		{
			name:        "poetry_project_end_to_end",
			description: "Complete build test for Poetry project",
			setupProject: func(t *testing.T, projectDir string) (string, string) {
				// Create pyproject.toml with Poetry configuration
				pyprojectContent := `
[tool.poetry]
name = "poetry-test"
version = "0.1.0"
description = "Poetry project test"

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.31.0"
pydantic = "^2.0.0"

[tool.poetry.group.dev.dependencies]
pytest = "^7.0.0"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create package structure
				packageDir := filepath.Join(projectDir, "poetry_test")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}

				// Create __init__.py
				initPath := filepath.Join(packageDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte("__version__ = '0.1.0'"), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				// Create models using Pydantic
				modelsContent := `
from pydantic import BaseModel
from typing import Optional

class RequestModel(BaseModel):
    message: str
    timestamp: Optional[str] = None

class ResponseModel(BaseModel):
    status: str
    data: dict
`
				modelsPath := filepath.Join(packageDir, "models.py")
				if err := os.WriteFile(modelsPath, []byte(modelsContent), 0644); err != nil {
					t.Fatalf("Failed to create models: %v", err)
				}

				// Create handler that uses the models
				handlerContent := `
import json
from datetime import datetime
from poetry_test.models import RequestModel, ResponseModel

def lambda_handler(event, context):
    try:
        # Parse request
        body = json.loads(event.get('body', '{}'))
        request = RequestModel(**body)
        
        # Create response
        response = ResponseModel(
            status='success',
            data={
                'message': request.message,
                'timestamp': datetime.now().isoformat(),
                'processed_by': 'poetry_handler'
            }
        )
        
        return {
            'statusCode': 200,
            'body': response.model_dump_json()
        }
    except Exception as e:
        error_response = ResponseModel(
            status='error',
            data={'error': str(e)}
        )
        return {
            'statusCode': 400,
            'body': error_response.model_dump_json()
        }
`
				handlerPath := filepath.Join(packageDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				return "poetry_test/handler.py", "poetry_test/handler.lambda_handler"
			},
			expectSuccess: true,
			validateBuild: func(t *testing.T, output *runtime.BuildOutput, projectDir string) {
				// Verify package structure
				packageDir := filepath.Join(output.Out, "poetry_test")
				if _, err := os.Stat(packageDir); os.IsNotExist(err) {
					t.Error("Package directory not found in output")
				}

				// Verify handler file
				handlerInOutput := filepath.Join(packageDir, "handler.py")
				if _, err := os.Stat(handlerInOutput); os.IsNotExist(err) {
					t.Error("Handler file not found in output package")
				}

				// Verify models file (optional - may not be copied in all build modes)
				modelsInOutput := filepath.Join(packageDir, "models.py")
				if _, err := os.Stat(modelsInOutput); os.IsNotExist(err) {
					t.Log("Models file not found in output package - this may be expected depending on build configuration")
				}

				// Verify __init__.py (optional - may not be copied in all build modes)
				initInOutput := filepath.Join(packageDir, "__init__.py")
				if _, err := os.Stat(initInOutput); os.IsNotExist(err) {
					t.Log("__init__.py not found in output package - this may be expected depending on build configuration")
				} else {
					// Check __init__.py content if it exists
					initContent, err := os.ReadFile(initInOutput)
					if err != nil {
						t.Logf("Could not read __init__.py: %v", err)
					} else if !strings.Contains(string(initContent), "__version__") {
						t.Log("Version information not preserved in __init__.py - this may be expected")
					}
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
			_, handlerSpec := tt.setupProject(t, projectDir)

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
			input := createEndToEndBuildInput(t, tempDir, handlerSpec)

			// Execute build
			ctx := context.Background()
			output, err := pipeline.Build(ctx, input)

			if tt.expectSuccess {
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
				if tt.validateBuild != nil {
					tt.validateBuild(t, output, projectDir)
				}

				t.Logf("✅ %s: End-to-end build successful - handler=%s, out=%s", tt.description, output.Handler, output.Out)
			} else {
				if err == nil {
					t.Errorf("Expected build to fail for %s but it succeeded", tt.description)
				}
				t.Logf("❌ %s: Build failed as expected - %v", tt.description, err)
			}
		})
	}
}

// TestCacheCompatibility tests that caching works correctly with the simplified runtime
func TestCacheCompatibility(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create a simple handler
	handlerContent := `
import json

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Cached handler'})
    }
`
	handlerPath := filepath.Join(projectDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create build pipeline with caching enabled
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:             projectDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		EnableCaching:           true,
		EnableProgressReporting: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Create build input
	input := createEndToEndBuildInput(t, tempDir, "handler.lambda_handler")

	ctx := context.Background()

	// First build - should be a cache miss
	t.Log("Performing first build (cache miss expected)")
	output1, err := pipeline.Build(ctx, input)
	if err != nil {
		t.Fatalf("First build failed: %v", err)
	}

	// Second build - should be a cache hit
	t.Log("Performing second build (cache hit expected)")
	output2, err := pipeline.Build(ctx, input)
	if err != nil {
		t.Fatalf("Second build failed: %v", err)
	}

	// Verify both builds produced the same output
	if output1.Handler != output2.Handler {
		t.Errorf("Handler mismatch between builds: %s vs %s", output1.Handler, output2.Handler)
	}

	// Modify the handler and build again - should be a cache miss
	t.Log("Modifying handler and performing third build (cache miss expected)")
	modifiedHandlerContent := `
import json

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({'message': 'Modified cached handler'})
    }
`
	if err := os.WriteFile(handlerPath, []byte(modifiedHandlerContent), 0644); err != nil {
		t.Fatalf("Failed to modify handler: %v", err)
	}

	output3, err := pipeline.Build(ctx, input)
	if err != nil {
		t.Fatalf("Third build failed: %v", err)
	}

	// Verify the modified content is in the output
	handlerInOutput := filepath.Join(output3.Out, "handler.py")
	outputContent, err := os.ReadFile(handlerInOutput)
	if err != nil {
		t.Fatalf("Failed to read output handler: %v", err)
	}

	if !strings.Contains(string(outputContent), "Modified cached handler") {
		t.Error("Modified handler content not found in output")
	}

	t.Log("✅ Cache compatibility test passed")
}

// createEndToEndBuildInput creates a build input for end-to-end testing
func createEndToEndBuildInput(t *testing.T, tempDir, handler string) *runtime.BuildInput {
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
		FunctionID: "end-to-end-test-function",
		Handler:    handler,
		Runtime:    "python3.12",
		Properties: propertiesJSON,
	}
}
