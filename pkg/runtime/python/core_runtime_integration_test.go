package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCoreRuntimeIntegration tests the integration of core runtime components
// using ProjectResolver instead of LayoutDetector
func TestCoreRuntimeIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "python-runtime-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple Python project structure
	projectDir := filepath.Join(tempDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create a simple Python handler
	handlerContent := `def lambda_handler(event, context):
    return {"statusCode": 200, "body": "Hello World"}
`
	handlerPath := filepath.Join(projectDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	// Create a pyproject.toml
	pyprojectContent := `[project]
name = "test-project"
version = "0.1.0"
dependencies = []

[build-system]
requires = ["setuptools", "wheel"]
build-backend = "setuptools.build_meta"
`
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	t.Run("ProjectResolver resolves handler correctly", func(t *testing.T) {
		resolver := NewProjectResolver(projectDir)

		projectInfo, err := resolver.ResolveHandler("handler.lambda_handler")
		if err != nil {
			t.Fatalf("Failed to resolve handler: %v", err)
		}

		if projectInfo.HandlerFile != handlerPath {
			t.Errorf("Expected handler file %s, got %s", handlerPath, projectInfo.HandlerFile)
		}

		if projectInfo.PyprojectPath != pyprojectPath {
			t.Errorf("Expected pyproject path %s, got %s", pyprojectPath, projectInfo.PyprojectPath)
		}

		if projectInfo.ModulePath != "handler" {
			t.Errorf("Expected module path 'handler', got %s", projectInfo.ModulePath)
		}
	})

	t.Run("DependencyAnalyzer works with ProjectInfo", func(t *testing.T) {
		resolver := NewProjectResolver(projectDir)
		projectInfo, err := resolver.ResolveHandler("handler.lambda_handler")
		if err != nil {
			t.Fatalf("Failed to resolve handler: %v", err)
		}

		analyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
			ProjectResolver: resolver,
		})

		ctx := context.Background()
		analysis, err := analyzer.AnalyzeDependencies(ctx, projectInfo)
		if err != nil {
			t.Fatalf("Failed to analyze dependencies: %v", err)
		}

		if analysis.PackageName != "test-project" {
			t.Errorf("Expected package name 'test-project', got %s", analysis.PackageName)
		}

		if len(analysis.DependencyFiles) == 0 {
			t.Error("Expected at least one dependency file (pyproject.toml)")
		}
	})

	t.Run("IncrementalBuilder uses ProjectResolver", func(t *testing.T) {
		// Create a cache directory
		cacheDir := filepath.Join(tempDir, "cache")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			t.Fatalf("Failed to create cache dir: %v", err)
		}

		// Create an output directory
		outputDir := filepath.Join(tempDir, "output")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Fatalf("Failed to create output dir: %v", err)
		}

		// Create incremental builder
		builder, err := NewIncrementalBuilder(IncrementalBuilderConfig{
			CacheDir:    cacheDir,
			ArtifactDir: outputDir,
		})
		if err != nil {
			t.Fatalf("Failed to create incremental builder: %v", err)
		}

		// Set project root
		builder.SetProjectRoot(projectDir)

		// Test that the builder has a project resolver
		if builder.projectResolver == nil {
			t.Error("Expected builder to have a project resolver")
		}

		if builder.projectResolver.projectRoot != projectDir {
			t.Errorf("Expected project root %s, got %s", projectDir, builder.projectResolver.projectRoot)
		}
	})

	t.Run("ChangeDetector uses ProjectResolver", func(t *testing.T) {
		resolver := NewProjectResolver(projectDir)

		// Create a simple build cache
		buildCache, err := NewDefaultBuildCache(filepath.Join(tempDir, "cache"))
		if err != nil {
			t.Fatalf("Failed to create build cache: %v", err)
		}

		changeDetector, err := NewChangeDetector(ChangeDetectorConfig{
			ProjectResolver: resolver,
			BuildCache:      buildCache,
		})
		if err != nil {
			t.Fatalf("Failed to create change detector: %v", err)
		}

		// Test change detection
		result, err := changeDetector.DetectChanges("test-function", "handler.lambda_handler")
		if err != nil {
			t.Fatalf("Failed to detect changes: %v", err)
		}

		// Should indicate changes needed since no cached build exists
		if !result.HasChanges {
			t.Error("Expected changes to be detected for initial build")
		}

		if result.Reason == "" {
			t.Error("Expected a reason for the changes")
		}
	})
}

// TestProjectResolverEdgeCases tests edge cases in project resolution
func TestProjectResolverEdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "python-resolver-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("Handler not found", func(t *testing.T) {
		resolver := NewProjectResolver(tempDir)

		_, err := resolver.ResolveHandler("nonexistent.handler")
		if err == nil {
			t.Error("Expected error for nonexistent handler")
		}

		// Should be a HandlerNotFoundError (check error message contains expected text)
		if !strings.Contains(err.Error(), "handler not found") && !strings.Contains(err.Error(), "could not find") && !strings.Contains(err.Error(), "handler_not_found") {
			t.Errorf("Expected handler not found error, got: %v", err)
		}
	})

	t.Run("Project without pyproject.toml", func(t *testing.T) {
		projectDir := filepath.Join(tempDir, "no-pyproject")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project dir: %v", err)
		}

		// Create a simple handler
		handlerContent := `def handler(event, context):
    return "ok"
`
		handlerPath := filepath.Join(projectDir, "simple.py")
		if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
			t.Fatalf("Failed to create handler file: %v", err)
		}

		resolver := NewProjectResolver(projectDir)
		projectInfo, err := resolver.ResolveHandler("simple.handler")
		if err != nil {
			t.Fatalf("Failed to resolve handler: %v", err)
		}

		// Should work without pyproject.toml
		if projectInfo.PyprojectPath != "" {
			t.Error("Expected empty pyproject path for project without pyproject.toml")
		}

		if projectInfo.SourceRoot != projectDir {
			t.Errorf("Expected source root to be project dir %s, got %s", projectDir, projectInfo.SourceRoot)
		}
	})

	t.Run("Nested project structure", func(t *testing.T) {
		projectDir := filepath.Join(tempDir, "nested")
		srcDir := filepath.Join(projectDir, "src")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatalf("Failed to create src dir: %v", err)
		}

		// Create pyproject.toml in root
		pyprojectContent := `[project]
name = "nested-project"
`
		pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
		if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
			t.Fatalf("Failed to create pyproject.toml: %v", err)
		}

		// Create handler in src directory
		handlerContent := `def handler(event, context):
    return "nested"
`
		handlerPath := filepath.Join(srcDir, "nested_handler.py")
		if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
			t.Fatalf("Failed to create handler file: %v", err)
		}

		resolver := NewProjectResolver(projectDir)
		projectInfo, err := resolver.ResolveHandler("src/nested_handler.handler")
		if err != nil {
			t.Fatalf("Failed to resolve nested handler: %v", err)
		}

		if projectInfo.PyprojectPath != pyprojectPath {
			t.Errorf("Expected pyproject path %s, got %s", pyprojectPath, projectInfo.PyprojectPath)
		}

		if projectInfo.SourceRoot != srcDir {
			t.Errorf("Expected source root to be src dir %s, got %s", srcDir, projectInfo.SourceRoot)
		}
	})
}
