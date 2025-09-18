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

// TestNewBuildPipeline tests the creation of a new build pipeline
func TestNewBuildPipeline(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		config      BuildPipelineConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_config",
			config: BuildPipelineConfig{
				ProjectRoot:   tempDir,
				ArtifactDir:   filepath.Join(tempDir, "artifacts"),
				EnableCaching: true,
				CacheDir:      filepath.Join(tempDir, "cache"),
			},
			expectError: false,
		},
		{
			name: "missing_project_root",
			config: BuildPipelineConfig{
				ArtifactDir: filepath.Join(tempDir, "artifacts"),
			},
			expectError: true,
			errorMsg:    "project root is required",
		},
		{
			name: "missing_artifact_dir",
			config: BuildPipelineConfig{
				ProjectRoot: tempDir,
			},
			expectError: true,
			errorMsg:    "artifact directory is required",
		},
		{
			name: "caching_without_cache_dir",
			config: BuildPipelineConfig{
				ProjectRoot:   tempDir,
				ArtifactDir:   filepath.Join(tempDir, "artifacts"),
				EnableCaching: true,
				// CacheDir is empty
			},
			expectError: false, // Should work, just disable caching
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline, err := NewBuildPipeline(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if pipeline == nil {
					t.Errorf("Expected pipeline to be created")
				}
			}
		})
	}
}

// TestBuildPipeline_Build_FlatProject tests building a flat project structure
func TestBuildPipeline_Build_FlatProject(t *testing.T) {
	tempDir := t.TempDir()

	// Create a flat project structure
	setupFlatProjectForPipeline(t, tempDir)

	// Create build pipeline
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:             tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		EnableCaching:           false,
		EnableProgressReporting: true,
		ProgressCallback: func(event ProgressEvent) {
			t.Logf("Progress: %s - %s", event.Stage, event.Message)
		},
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Create build input
	input := createBuildInput(t, tempDir, "handler.lambda_handler")

	// Execute build
	ctx := context.Background()
	output, err := pipeline.Build(ctx, input)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify output
	if output == nil {
		t.Fatal("Expected build output")
	}

	if output.Out == "" {
		t.Error("Expected output directory")
	}

	if output.Handler == "" {
		t.Error("Expected handler path")
	}

	// Verify artifacts were created
	if _, err := os.Stat(output.Out); os.IsNotExist(err) {
		t.Error("Output directory was not created")
	}

	t.Logf("Build successful: out=%s, handler=%s", output.Out, output.Handler)
}

// TestBuildPipeline_Build_WorkspaceProject tests building a workspace project structure
func TestBuildPipeline_Build_WorkspaceProject(t *testing.T) {
	tempDir := t.TempDir()

	// Create a workspace project structure
	setupWorkspaceProjectForPipeline(t, tempDir)

	// Create build pipeline
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:             tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		EnableCaching:           false,
		EnableProgressReporting: true,
		ProgressCallback: func(event ProgressEvent) {
			t.Logf("Progress: %s - %s", event.Stage, event.Message)
		},
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Create build input
	input := createBuildInput(t, tempDir, "functions/api.lambda_handler")

	// Execute build
	ctx := context.Background()
	output, err := pipeline.Build(ctx, input)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify output
	if output == nil {
		t.Fatal("Expected build output")
	}

	// Verify artifacts were created
	if _, err := os.Stat(output.Out); os.IsNotExist(err) {
		t.Error("Output directory was not created")
	}

	t.Logf("Build successful: out=%s, handler=%s", output.Out, output.Handler)
}

// TestBuildPipeline_Build_NestedProject tests building a nested project structure
func TestBuildPipeline_Build_NestedProject(t *testing.T) {
	tempDir := t.TempDir()

	// Create a nested project structure
	setupNestedProjectForPipeline(t, tempDir)

	// Create build pipeline
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:             tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		EnableCaching:           false,
		EnableProgressReporting: true,
		ProgressCallback: func(event ProgressEvent) {
			t.Logf("Progress: %s - %s", event.Stage, event.Message)
		},
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Create build input
	input := createBuildInput(t, tempDir, "src/mypackage/handlers/api.lambda_handler")

	// Execute build
	ctx := context.Background()
	output, err := pipeline.Build(ctx, input)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify output
	if output == nil {
		t.Fatal("Expected build output")
	}

	t.Logf("Build successful: out=%s, handler=%s", output.Out, output.Handler)
}

// TestBuildPipeline_Build_HandlerNotFound tests error handling for missing handlers
func TestBuildPipeline_Build_HandlerNotFound(t *testing.T) {
	tempDir := t.TempDir()

	// Create build pipeline
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:   tempDir,
		ArtifactDir:   filepath.Join(tempDir, "artifacts"),
		EnableCaching: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Create build input with non-existent handler
	input := createBuildInput(t, tempDir, "nonexistent.handler")

	// Execute build
	ctx := context.Background()
	output, err := pipeline.Build(ctx, input)

	// Should fail with handler not found error
	if err == nil {
		t.Fatal("Expected build to fail with handler not found error")
	}

	if output != nil {
		t.Error("Expected no output on error")
	}

	// Check error type
	if !strings.Contains(err.Error(), "project resolution failed") {
		t.Errorf("Expected project resolution error, got: %v", err)
	}

	t.Logf("Expected error: %v", err)
}

// TestBuildPipeline_Build_WithCaching tests build caching functionality
func TestBuildPipeline_Build_WithCaching(t *testing.T) {
	tempDir := t.TempDir()

	// Create a flat project structure
	setupFlatProjectForPipeline(t, tempDir)

	// Create build pipeline with caching enabled
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:             tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		EnableCaching:           true,
		CacheDir:                filepath.Join(tempDir, "cache"),
		EnableProgressReporting: true,
		ProgressCallback: func(event ProgressEvent) {
			t.Logf("Progress: %s - %s", event.Stage, event.Message)
		},
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Create build input
	input := createBuildInput(t, tempDir, "handler.lambda_handler")
	ctx := context.Background()

	// First build - should not use cache
	output1, err := pipeline.Build(ctx, input)
	if err != nil {
		t.Fatalf("First build failed: %v", err)
	}

	// Second build - should potentially use cache (though our implementation always rebuilds for now)
	output2, err := pipeline.Build(ctx, input)
	if err != nil {
		t.Fatalf("Second build failed: %v", err)
	}

	// Verify both builds succeeded
	if output1 == nil || output2 == nil {
		t.Fatal("Expected build outputs")
	}

	t.Logf("First build: %s", output1.Out)
	t.Logf("Second build: %s", output2.Out)
}

// TestBuildPipeline_Build_ContainerBuild tests container build handling
func TestBuildPipeline_Build_ContainerBuild(t *testing.T) {
	tempDir := t.TempDir()

	// Create a flat project structure
	setupFlatProjectForPipeline(t, tempDir)

	// Create build pipeline
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:   tempDir,
		ArtifactDir:   filepath.Join(tempDir, "artifacts"),
		EnableCaching: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Create build input for container build
	input := createContainerBuildInput(t, tempDir, "handler.lambda_handler")

	// Execute build
	ctx := context.Background()
	output, err := pipeline.Build(ctx, input)

	if err != nil {
		t.Fatalf("Container build failed: %v", err)
	}

	// Verify output
	if output == nil {
		t.Fatal("Expected build output")
	}

	t.Logf("Container build successful: out=%s, handler=%s", output.Out, output.Handler)
}

// TestBuildPipeline_ShouldRebuild tests rebuild decision logic
func TestBuildPipeline_ShouldRebuild(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		enableCaching bool
		expectRebuild bool
	}{
		{
			name:          "no_caching",
			enableCaching: false,
			expectRebuild: true,
		},
		{
			name:          "with_caching",
			enableCaching: true,
			expectRebuild: true, // Our implementation always rebuilds for now
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := BuildPipelineConfig{
				ProjectRoot:   tempDir,
				ArtifactDir:   filepath.Join(tempDir, "artifacts"),
				EnableCaching: tt.enableCaching,
			}

			if tt.enableCaching {
				config.CacheDir = filepath.Join(tempDir, "cache")
			}

			pipeline, err := NewBuildPipeline(config)
			if err != nil {
				t.Fatalf("Failed to create build pipeline: %v", err)
			}

			shouldRebuild := pipeline.ShouldRebuild("test-function", "handler.lambda_handler")

			if shouldRebuild != tt.expectRebuild {
				t.Errorf("Expected shouldRebuild=%v, got %v", tt.expectRebuild, shouldRebuild)
			}
		})
	}
}

// TestBuildPipeline_ClearCache tests cache clearing functionality
func TestBuildPipeline_ClearCache(t *testing.T) {
	tempDir := t.TempDir()

	// Create build pipeline with caching
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:   tempDir,
		ArtifactDir:   filepath.Join(tempDir, "artifacts"),
		EnableCaching: true,
		CacheDir:      filepath.Join(tempDir, "cache"),
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Clear cache should not error
	err = pipeline.ClearCache()
	if err != nil {
		t.Errorf("ClearCache failed: %v", err)
	}

	// Test with pipeline without caching
	pipelineNoCache, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:   tempDir,
		ArtifactDir:   filepath.Join(tempDir, "artifacts"),
		EnableCaching: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Clear cache should not error even without caching
	err = pipelineNoCache.ClearCache()
	if err != nil {
		t.Errorf("ClearCache failed on pipeline without caching: %v", err)
	}
}

// TestBuildPipeline_GetCacheStats tests cache statistics functionality
func TestBuildPipeline_GetCacheStats(t *testing.T) {
	tempDir := t.TempDir()

	// Test with caching enabled
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:   tempDir,
		ArtifactDir:   filepath.Join(tempDir, "artifacts"),
		EnableCaching: true,
		CacheDir:      filepath.Join(tempDir, "cache"),
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	stats := pipeline.GetCacheStats()
	if stats == nil {
		t.Error("Expected cache stats when caching is enabled")
	}

	// Test with caching disabled
	pipelineNoCache, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:   tempDir,
		ArtifactDir:   filepath.Join(tempDir, "artifacts"),
		EnableCaching: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	statsNoCache := pipelineNoCache.GetCacheStats()
	if statsNoCache != nil {
		t.Error("Expected no cache stats when caching is disabled")
	}
}

// Helper functions for setting up test projects

// setupFlatProjectForPipeline creates a flat project structure for testing
func setupFlatProjectForPipeline(t *testing.T, projectDir string) {
	// Create handler file
	handlerContent := `
def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': 'Hello from flat project!'
    }
`
	handlerPath := filepath.Join(projectDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	// Create requirements.txt
	requirementsContent := "requests==2.28.1\n"
	requirementsPath := filepath.Join(projectDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte(requirementsContent), 0644); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}
}

// setupWorkspaceProjectForPipeline creates a workspace project structure for testing
func setupWorkspaceProjectForPipeline(t *testing.T, projectDir string) {
	// Create pyproject.toml
	pyprojectContent := `
[project]
name = "my-workspace"
version = "0.1.0"
dependencies = ["requests>=2.28.0"]

[tool.uv.workspace]
members = ["functions"]
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

	// Create handler file
	handlerContent := `
def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': 'Hello from workspace project!'
    }
`
	handlerPath := filepath.Join(functionsDir, "api.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	// Create __init__.py
	initPath := filepath.Join(functionsDir, "__init__.py")
	if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create __init__.py: %v", err)
	}
}

// setupNestedProjectForPipeline creates a nested project structure for testing
func setupNestedProjectForPipeline(t *testing.T, projectDir string) {
	// Create pyproject.toml
	pyprojectContent := `
[project]
name = "mypackage"
version = "0.1.0"
dependencies = ["requests>=2.28.0"]

[tool.setuptools.packages.find]
where = ["src"]
`
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create src directory structure
	srcDir := filepath.Join(projectDir, "src", "mypackage", "handlers")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}

	// Create handler file
	handlerContent := `
def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': 'Hello from nested project!'
    }
`
	handlerPath := filepath.Join(srcDir, "api.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	// Create __init__.py files
	initPaths := []string{
		filepath.Join(projectDir, "src", "mypackage", "__init__.py"),
		filepath.Join(projectDir, "src", "mypackage", "handlers", "__init__.py"),
	}

	for _, initPath := range initPaths {
		if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create __init__.py: %v", err)
		}
	}
}

// createBuildInput creates a build input for testing
func createBuildInput(t *testing.T, projectDir, handler string) *runtime.BuildInput {
	outputDir := filepath.Join(projectDir, "output")
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

	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Handler:    handler,
		CfgPath:    filepath.Join(projectDir, "sst.config.ts"),
		Properties: json.RawMessage(propertiesJSON),
		Dev:        false,
	}

	// The Out() method will generate the output directory path
	return input
}

// createContainerBuildInput creates a container build input for testing
func createContainerBuildInput(t *testing.T, projectDir, handler string) *runtime.BuildInput {
	input := createBuildInput(t, projectDir, handler)

	properties := map[string]interface{}{
		"architecture": "x86_64",
		"container":    true,
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		t.Fatalf("Failed to marshal properties: %v", err)
	}

	input.Properties = json.RawMessage(propertiesJSON)
	return input
}

// TestBuildPipeline_ErrorHandling tests comprehensive error handling
func TestBuildPipeline_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Create build pipeline
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:         tempDir,
		ArtifactDir:         filepath.Join(tempDir, "artifacts"),
		EnableCaching:       false,
		EnableErrorRecovery: true,
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	tests := []struct {
		name          string
		setupFunc     func() *runtime.BuildInput
		expectError   bool
		errorContains string
	}{
		{
			name: "invalid_handler_path",
			setupFunc: func() *runtime.BuildInput {
				return createBuildInput(t, tempDir, "invalid/path/handler.lambda_handler")
			},
			expectError:   true,
			errorContains: "project resolution failed",
		},
		{
			name: "empty_handler",
			setupFunc: func() *runtime.BuildInput {
				return createBuildInput(t, tempDir, "")
			},
			expectError:   true,
			errorContains: "project resolution failed",
		},
		{
			name: "malformed_properties",
			setupFunc: func() *runtime.BuildInput {
				// Create a handler file for this test
				setupFlatProjectForPipeline(t, tempDir)
				input := createBuildInput(t, tempDir, "handler.lambda_handler")
				input.Properties = json.RawMessage(`{"invalid": json}`)
				return input
			},
			expectError: false, // Should handle malformed properties gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.setupFunc()
			ctx := context.Background()

			output, err := pipeline.Build(ctx, input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				if output != nil {
					t.Error("Expected no output on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestBuildPipeline_ProgressReporting tests progress reporting functionality
func TestBuildPipeline_ProgressReporting(t *testing.T) {
	tempDir := t.TempDir()
	setupFlatProjectForPipeline(t, tempDir)

	var progressEvents []ProgressEvent

	// Create build pipeline with progress reporting
	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:             tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		EnableCaching:           false,
		EnableProgressReporting: true,
		ProgressCallback: func(event ProgressEvent) {
			progressEvents = append(progressEvents, event)
		},
	})
	if err != nil {
		t.Fatalf("Failed to create build pipeline: %v", err)
	}

	// Execute build
	input := createBuildInput(t, tempDir, "handler.lambda_handler")
	ctx := context.Background()

	_, err = pipeline.Build(ctx, input)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify progress events were reported
	if len(progressEvents) == 0 {
		t.Error("Expected progress events to be reported")
	}

	// Check for expected stages
	expectedStages := []string{"resolve", "analyze", "package", "complete"}
	reportedStages := make(map[string]bool)

	for _, event := range progressEvents {
		reportedStages[event.Stage] = true
		t.Logf("Progress event: %s - %s", event.Stage, event.Message)
	}

	for _, expectedStage := range expectedStages {
		if !reportedStages[expectedStage] {
			t.Errorf("Expected stage '%s' to be reported", expectedStage)
		}
	}
}
