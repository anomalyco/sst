package python

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

// TestDeploymentArtifactIntegration tests end-to-end deployment artifact creation
// This test addresses task 5.2 requirements for comprehensive integration testing
func TestDeploymentArtifactIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name                 string
		setupFunc            func(string) error
		handlerPath          string
		expectedLayout       LayoutType
		expectedModules      []string
		maxArtifactSizeMB    int64
		shouldIncludeModules bool
		shouldExcludeContent []string
	}{
		{
			name:                 "workspace_layout_modules_included",
			setupFunc:            setupWorkspaceLayoutProject,
			handlerPath:          "functions/handler.lambda_handler",
			expectedLayout:       LayoutTypeWorkspace,
			expectedModules:      []string{"functions", "core"},
			maxArtifactSizeMB:    50, // Should be much smaller than this
			shouldIncludeModules: true,
			shouldExcludeContent: []string{".sst/", "__pycache__/", ".venv/", ".git/"},
		},
		{
			name:                 "flat_layout_no_excessive_content",
			setupFunc:            setupFlatLayoutProject,
			handlerPath:          "handler.lambda_handler",
			expectedLayout:       LayoutTypeFlat,
			expectedModules:      []string{"flat_example"},
			maxArtifactSizeMB:    10, // Should be very small for flat layout
			shouldIncludeModules: true,
			shouldExcludeContent: []string{".sst/", "__pycache__/", ".venv/", ".git/"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testProjectDir := filepath.Join(tempDir, "project")

			// Setup test project
			if err := tc.setupFunc(testProjectDir); err != nil {
				t.Fatalf("Failed to setup test project: %v", err)
			}

			// Create Python runtime
			pythonRuntime := New()

			input := &runtime.BuildInput{
				FunctionID: fmt.Sprintf("test-%s", tc.name),
				Handler:    tc.handlerPath,
				CfgPath:    filepath.Join(testProjectDir, "pyproject.toml"),
				Properties: []byte(`{"architecture": "x86_64"}`),
			}

			// Test build process
			ctx := context.Background()
			result, err := pythonRuntime.Build(ctx, input)

			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			if result == nil {
				t.Fatal("Build result is nil")
			}

			// Validate artifact was created
			if result.Handler == "" {
				t.Error("Handler path is empty in build result")
			}

			// Test artifact contents - this is the key regression test
			t.Run("validate_artifact_contents", func(t *testing.T) {
				validateArtifactContents(t, result, tc.expectedModules, tc.shouldExcludeContent)
			})

			// Test artifact size - prevents oversized artifacts regression
			t.Run("validate_artifact_size", func(t *testing.T) {
				validateArtifactSize(t, result, tc.maxArtifactSizeMB)
			})

			// Test module importability - ensures modules are properly structured
			if tc.shouldIncludeModules {
				t.Run("validate_module_importability", func(t *testing.T) {
					validateModuleImportability(t, result, tc.expectedModules)
				})
			}
		})
	}
}

// TestRegressionPrevention tests specific regression scenarios mentioned in task 5.2
func TestRegressionPrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("missing_modules_regression", func(t *testing.T) {
		// Test that modules are not missing from workspace layout (aws-python example issue)
		testMissingModulesRegression(t)
	})

	t.Run("oversized_artifacts_regression", func(t *testing.T) {
		// Test that artifacts don't include excessive content (flat-layout example issue)
		testOversizedArtifactsRegression(t)
	})

	t.Run("silent_build_failures_regression", func(t *testing.T) {
		// Test that build failures are properly reported
		testSilentBuildFailuresRegression(t)
	})
}

// Helper functions for validation

func validateArtifactContents(t *testing.T, result *runtime.BuildOutput, expectedModules []string, shouldExcludeContent []string) {
	if result.Handler == "" {
		t.Error("Build result handler is empty")
		return
	}

	// Get artifact directory from build output
	artifactDir := result.Out

	// Check that expected modules are present
	for _, module := range expectedModules {
		found := false

		// Check for module as directory
		modulePath := filepath.Join(artifactDir, module)
		if stat, err := os.Stat(modulePath); err == nil && stat.IsDir() {
			found = true
		}

		// Check for module as .py file
		if !found {
			moduleFile := filepath.Join(artifactDir, module+".py")
			if _, err := os.Stat(moduleFile); err == nil {
				found = true
			}
		}

		if !found {
			t.Errorf("Expected module %s not found in artifact directory", module)
		}
	}

	// Check that excluded content is not present
	err := filepath.Walk(artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(artifactDir, path)
		for _, excludePattern := range shouldExcludeContent {
			if strings.Contains(relPath, excludePattern) {
				t.Errorf("Excluded content found in artifact: %s (matches pattern %s)", relPath, excludePattern)
			}
		}
		return nil
	})

	if err != nil {
		t.Errorf("Error walking artifact directory: %v", err)
	}
}

func validateArtifactSize(t *testing.T, result *runtime.BuildOutput, maxSizeMB int64) {
	if result.Handler == "" {
		t.Error("Build result handler is empty")
		return
	}

	artifactDir := result.Out
	var totalSize int64

	err := filepath.Walk(artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		t.Errorf("Error calculating artifact size: %v", err)
		return
	}

	maxSizeBytes := maxSizeMB * 1024 * 1024
	if totalSize > maxSizeBytes {
		t.Errorf("Artifact size %d bytes exceeds maximum %d bytes (%d MB)", totalSize, maxSizeBytes, maxSizeMB)
	}

	t.Logf("Artifact size: %d bytes (%.2f MB)", totalSize, float64(totalSize)/(1024*1024))
}

func validateModuleImportability(t *testing.T, result *runtime.BuildOutput, expectedModules []string) {
	if result.Handler == "" {
		t.Error("Build result handler is empty")
		return
	}

	artifactDir := result.Out

	for _, module := range expectedModules {
		// Check if module has proper Python package structure
		modulePath := filepath.Join(artifactDir, module)

		if stat, err := os.Stat(modulePath); err == nil && stat.IsDir() {
			// Directory module - should have __init__.py
			initPath := filepath.Join(modulePath, "__init__.py")
			if _, err := os.Stat(initPath); err != nil {
				t.Errorf("Module directory %s missing __init__.py file", module)
			}
		} else {
			// File module - should be .py file
			moduleFile := filepath.Join(artifactDir, module+".py")
			if _, err := os.Stat(moduleFile); err != nil {
				t.Errorf("Module file %s.py not found", module)
			}
		}
	}
}

// Regression test functions

func testMissingModulesRegression(t *testing.T) {
	// This test specifically checks for the regression where Python modules
	// were missing from deployment artifacts in workspace layouts

	tempDir := t.TempDir()
	testProjectDir := filepath.Join(tempDir, "project")

	// Setup workspace layout project
	if err := setupWorkspaceLayoutProject(testProjectDir); err != nil {
		t.Fatalf("Failed to setup workspace project: %v", err)
	}

	// Build with Python runtime
	pythonRuntime := New()
	input := &runtime.BuildInput{
		FunctionID: "regression-test-missing-modules",
		Handler:    "functions/handler.lambda_handler",
		CfgPath:    filepath.Join(testProjectDir, "pyproject.toml"),
		Properties: []byte(`{"architecture": "x86_64"}`),
	}

	ctx := context.Background()
	result, err := pythonRuntime.Build(ctx, input)
	if err != nil {
		t.Fatalf("Build failed (this indicates the regression is present): %v", err)
	}

	if result == nil {
		t.Fatal("Build result is nil")
	}

	// The key regression test: ensure modules are actually present
	expectedModules := []string{"functions", "core"}
	artifactDir := result.Out

	for _, module := range expectedModules {
		found := false

		// Check multiple possible locations
		possiblePaths := []string{
			filepath.Join(artifactDir, module),
			filepath.Join(artifactDir, module+".py"),
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("REGRESSION: Module %s is missing from deployment artifact", module)
		}
	}
}

func testOversizedArtifactsRegression(t *testing.T) {
	// This test specifically checks for the regression where deployment artifacts
	// included excessive content like .sst directories and build cache

	tempDir := t.TempDir()
	testProjectDir := filepath.Join(tempDir, "project")

	// Setup flat layout project
	if err := setupFlatLayoutProject(testProjectDir); err != nil {
		t.Fatalf("Failed to setup flat layout project: %v", err)
	}

	// Build with Python runtime
	pythonRuntime := New()
	input := &runtime.BuildInput{
		FunctionID: "regression-test-oversized-artifacts",
		Handler:    "handler.lambda_handler",
		CfgPath:    filepath.Join(testProjectDir, "pyproject.toml"),
		Properties: []byte(`{"architecture": "x86_64"}`),
	}

	ctx := context.Background()
	result, err := pythonRuntime.Build(ctx, input)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check artifact size - should be small for flat layout
	artifactDir := result.Out
	var totalSize int64

	err = filepath.Walk(artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}

		// Check for problematic content
		relPath, _ := filepath.Rel(artifactDir, path)
		problematicPatterns := []string{".sst/", "__pycache__/", ".venv/", ".git/"}

		for _, pattern := range problematicPatterns {
			if strings.Contains(relPath, pattern) {
				t.Errorf("REGRESSION: Problematic content found in artifact: %s", relPath)
			}
		}

		return nil
	})

	if err != nil {
		t.Errorf("Error walking artifact directory: %v", err)
	}

	// Flat layout should be very small (under 1MB for basic example)
	maxSizeBytes := int64(1024 * 1024) // 1MB
	if totalSize > maxSizeBytes {
		t.Errorf("REGRESSION: Artifact size %d bytes is too large (over %d bytes), likely includes excessive content", totalSize, maxSizeBytes)
	}
}

func testSilentBuildFailuresRegression(t *testing.T) {
	// This test checks that build failures are properly reported rather than
	// silently producing empty or incomplete artifacts

	tempDir := t.TempDir()

	// Create a project with intentionally broken configuration
	brokenProjectDir := filepath.Join(tempDir, "broken-project")
	if err := os.MkdirAll(brokenProjectDir, 0755); err != nil {
		t.Fatalf("Failed to create broken project directory: %v", err)
	}

	// Create invalid pyproject.toml
	invalidPyproject := `[project]
name = "broken-project"
version = "0.1.0"
dependencies = ["nonexistent-package-that-does-not-exist"]

[build-system]
requires = ["invalid-build-backend"]
build-backend = "invalid.backend"`

	if err := os.WriteFile(filepath.Join(brokenProjectDir, "pyproject.toml"), []byte(invalidPyproject), 0644); err != nil {
		t.Fatalf("Failed to create invalid pyproject.toml: %v", err)
	}

	// Create handler file
	if err := os.WriteFile(filepath.Join(brokenProjectDir, "handler.py"), []byte("def lambda_handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	// Attempt build - should fail with clear error, not silently succeed
	pythonRuntime := New()
	input := &runtime.BuildInput{
		FunctionID: "regression-test-silent-failures",
		Handler:    "handler.lambda_handler",
		CfgPath:    filepath.Join(brokenProjectDir, "pyproject.toml"),
		Properties: []byte(`{"architecture": "x86_64"}`),
	}

	ctx := context.Background()
	result, err := pythonRuntime.Build(ctx, input)

	// The key regression test: build should fail with clear error
	if err == nil {
		t.Error("REGRESSION: Build should have failed with broken configuration but succeeded silently")
	}

	if result != nil && result.Handler != "" {
		// If result is returned, verify it's not an empty/incomplete artifact
		artifactDir := result.Out
		if _, err := os.Stat(artifactDir); err == nil {
			// Artifact directory exists - check if it's empty or incomplete
			isEmpty := true
			filepath.Walk(artifactDir, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					isEmpty = false
				}
				return nil
			})

			if isEmpty {
				t.Error("REGRESSION: Build produced empty artifact directory instead of failing")
			}
		}
	}
}

// Project setup functions

func setupWorkspaceLayoutProject(projectDir string) error {
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}

	// Create pyproject.toml for workspace layout
	pyprojectContent := `[project]
name = "test-workspace"
version = "0.1.0"
dependencies = ["requests"]

[tool.uv.workspace]
members = ["functions", "core"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["functions", "core"]`

	if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
		return err
	}

	// Create functions package
	functionsDir := filepath.Join(projectDir, "functions")
	if err := os.MkdirAll(functionsDir, 0755); err != nil {
		return err
	}

	functionsPyproject := `[project]
name = "functions"
version = "0.1.0"
dependencies = ["requests"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
include = ["*.py"]`

	if err := os.WriteFile(filepath.Join(functionsDir, "pyproject.toml"), []byte(functionsPyproject), 0644); err != nil {
		return err
	}

	handlerContent := `def lambda_handler(event, context):
    return {"statusCode": 200, "body": "Hello from functions"}`
	if err := os.WriteFile(filepath.Join(functionsDir, "__init__.py"), []byte(""), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(functionsDir, "handler.py"), []byte(handlerContent), 0644); err != nil {
		return err
	}

	// Create core package
	coreDir := filepath.Join(projectDir, "core")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		return err
	}

	corePyproject := `[project]
name = "core"
version = "0.1.0"
dependencies = []

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
include = ["*.py"]`

	if err := os.WriteFile(filepath.Join(coreDir, "pyproject.toml"), []byte(corePyproject), 0644); err != nil {
		return err
	}

	coreContent := `class CoreModel:
    pass`
	if err := os.WriteFile(filepath.Join(coreDir, "__init__.py"), []byte(""), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(coreDir, "models.py"), []byte(coreContent), 0644); err != nil {
		return err
	}

	return nil
}

func setupFlatLayoutProject(projectDir string) error {
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}

	// Create pyproject.toml for flat layout - minimal dependencies for size testing
	pyprojectContent := `[project]
name = "flat-example"
version = "0.1.0"
description = "SST Python flat layout example"
dependencies = []

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
include = ["handler.py", "utils.py"]`

	if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
		return err
	}

	// Create handler file
	handlerContent := `def lambda_handler(event, context):
    return {"statusCode": 200, "body": "Hello from flat layout"}`
	if err := os.WriteFile(filepath.Join(projectDir, "handler.py"), []byte(handlerContent), 0644); err != nil {
		return err
	}

	// Create utils file
	utilsContent := `def helper_function():
    return "Helper from utils"`
	if err := os.WriteFile(filepath.Join(projectDir, "utils.py"), []byte(utilsContent), 0644); err != nil {
		return err
	}

	return nil
}

// Utility functions

func copyFileHelper(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
