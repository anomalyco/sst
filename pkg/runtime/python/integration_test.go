package python

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// TestEndToEndBuildPipeline tests the complete build pipeline end-to-end
func TestEndToEndBuildPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	testCases := []struct {
		name           string
		projectLayout  string
		expectedLayout LayoutType
		shouldSucceed  bool
	}{
		{
			name:           "workspace_layout",
			projectLayout:  "workspace",
			expectedLayout: LayoutTypeWorkspace,
			shouldSucceed:  true,
		},
		{
			name:           "flat_layout",
			projectLayout:  "flat",
			expectedLayout: LayoutTypeFlat,
			shouldSucceed:  true,
		},
		{
			name:           "nested_layout",
			projectLayout:  "nested",
			expectedLayout: LayoutTypeNested,
			shouldSucceed:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := filepath.Join(tempDir, tc.name)
			setupTestProject(t, projectDir, tc.projectLayout)

			config := IncrementalBuilderConfig{
				CacheDir:                  filepath.Join(tempDir, "cache", tc.name),
				ArtifactDir:               filepath.Join(tempDir, "artifacts", tc.name),
				MaxCacheSize:              10 * 1024 * 1024, // 10MB
				MaxCacheAge:               time.Hour,
				EnableParallelBuilds:      false, // Disable for predictable testing
				MaxParallelBuilds:         1,
				EnableProgressReporting:   true,
				EnableBuildOptimization:   true,
				EnableFallbacks:           true,
				EnableDeprecationWarnings: true,
			}

			builder, err := NewIncrementalBuilder(config)
			if err != nil {
				t.Fatalf("Failed to create incremental builder: %v", err)
			}

			// Test layout detection
			handlerPath := getHandlerPath(tc.projectLayout)
			layout, err := builder.layoutDetector.DetectLayout(handlerPath)
			if tc.shouldSucceed {
				if err != nil {
					t.Fatalf("Layout detection failed: %v", err)
				}
				if layout.Type != tc.expectedLayout {
					t.Errorf("Expected layout %s, got %s", tc.expectedLayout, layout.Type)
				}
			} else {
				if err == nil {
					t.Error("Expected layout detection to fail")
				}
			}

			if !tc.shouldSucceed {
				return
			}

			// Test dependency analysis
			dependencies, err := builder.dependencyAnalyzer.AnalyzeDependencies(context.Background(), layout)
			if err != nil {
				t.Fatalf("Dependency analysis failed: %v", err)
			}

			if dependencies == nil {
				t.Error("Dependencies should not be nil")
			}

			// Test build planning
			input := &runtime.BuildInput{
				FunctionID: fmt.Sprintf("test-%s", tc.name),
				Handler:    handlerPath,
				CfgPath:    layout.PyprojectPath,
			}

			buildPlan, err := builder.buildPlanner.CreateBuildPlan(context.Background(), input, layout, dependencies)
			if err != nil {
				t.Fatalf("Build planning failed: %v", err)
			}

			if buildPlan == nil {
				t.Error("Build plan should not be nil")
			}

			// Test rebuild decision
			shouldRebuild := builder.ShouldRebuild(input.FunctionID, input.Handler)
			if !shouldRebuild {
				t.Error("First build should require rebuild")
			}

			// Note: We don't test the actual build execution here as it requires UV
			// and external dependencies. The build execution is tested separately
			// in scenarios where UV is available.
		})
	}
}

// TestLiveSessionScenarios tests Live session scenarios with file changes
func TestLiveSessionScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "live_project")
	setupTestProject(t, projectDir, "workspace")

	config := IncrementalBuilderConfig{
		CacheDir:                filepath.Join(tempDir, "cache"),
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            10 * 1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableBuildOptimization: true,
		EnableFallbacks:         true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	input := &runtime.BuildInput{
		FunctionID: "live-function",
		Handler:    "src/mypackage/handler",
		CfgPath:    filepath.Join(projectDir, "pyproject.toml"),
	}

	// Initial build should require rebuild
	shouldRebuild1 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if !shouldRebuild1 {
		t.Error("Initial build should require rebuild")
	}

	// Simulate successful build by adding cache entry
	handlerPath := filepath.Join(projectDir, "src", "mypackage", "handler.py")
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")

	entry := &CacheEntry{
		FunctionID: input.FunctionID,
		LastBuild:  time.Now(),
		FileHashes: map[string]string{
			handlerPath:   "original-hash",
			pyprojectPath: "pyproject-hash",
		},
		Dependencies: []string{"requests"},
	}
	builder.buildCache.Set(input.FunctionID, entry)

	// Second check should not require rebuild
	shouldRebuild2 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if shouldRebuild2 {
		t.Error("Second check should not require rebuild (no changes)")
	}

	// Modify handler file
	modifiedContent := `def handler(event, context):
    return {"statusCode": 200, "body": "Modified handler"}`
	if err := os.WriteFile(handlerPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify handler: %v", err)
	}

	// Should require rebuild after file change
	shouldRebuild3 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if !shouldRebuild3 {
		t.Error("Should require rebuild after handler modification")
	}

	// Modify pyproject.toml (add dependency)
	pyprojectContent := `[project]
name = "mypackage"
version = "0.1.0"
dependencies = ["requests", "click"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"`

	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to modify pyproject.toml: %v", err)
	}

	// Should require rebuild after dependency change
	shouldRebuild4 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if !shouldRebuild4 {
		t.Error("Should require rebuild after dependency change")
	}

	// Test multiple functions in same project
	input2 := &runtime.BuildInput{
		FunctionID: "live-function-2",
		Handler:    "src/mypackage/handler2",
		CfgPath:    pyprojectPath,
	}

	// Create second handler
	handler2Path := filepath.Join(projectDir, "src", "mypackage", "handler2.py")
	if err := os.WriteFile(handler2Path, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create second handler: %v", err)
	}

	// Second function should require rebuild
	shouldRebuild5 := builder.ShouldRebuild(input2.FunctionID, input2.Handler)
	if !shouldRebuild5 {
		t.Error("Second function should require rebuild")
	}

	// Add cache entry for second function
	entry2 := &CacheEntry{
		FunctionID: input2.FunctionID,
		LastBuild:  time.Now(),
		FileHashes: map[string]string{
			handler2Path:  "handler2-hash",
			pyprojectPath: "pyproject-hash-2",
		},
		Dependencies: []string{"requests", "click"},
	}
	builder.buildCache.Set(input2.FunctionID, entry2)

	// Both functions should not require rebuild now
	shouldRebuild6 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	shouldRebuild7 := builder.ShouldRebuild(input2.FunctionID, input2.Handler)

	if shouldRebuild6 || shouldRebuild7 {
		t.Error("Neither function should require rebuild after caching")
	}

	// Modify shared dependency file - both should require rebuild
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent+"\n# comment"), 0644); err != nil {
		t.Fatalf("Failed to modify shared dependency: %v", err)
	}

	shouldRebuild8 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	shouldRebuild9 := builder.ShouldRebuild(input2.FunctionID, input2.Handler)

	if !shouldRebuild8 || !shouldRebuild9 {
		t.Error("Both functions should require rebuild after shared dependency change")
	}
}

// TestDeploymentOptimization tests deployment optimization scenarios
func TestDeploymentOptimization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	// Create multiple functions with shared dependencies
	functions := []struct {
		name    string
		handler string
		deps    []string
	}{
		{"api-function", "api/handler", []string{"fastapi", "uvicorn"}},
		{"worker-function", "worker/handler", []string{"celery", "redis"}},
		{"shared-function", "shared/handler", []string{"fastapi", "celery"}}, // Shares deps with both
	}

	for _, fn := range functions {
		projectDir := filepath.Join(tempDir, fn.name)
		setupTestProjectWithDeps(t, projectDir, fn.deps)
	}

	config := IncrementalBuilderConfig{
		CacheDir:                filepath.Join(tempDir, "cache"),
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            50 * 1024 * 1024, // 50MB
		MaxCacheAge:             time.Hour,
		EnableBuildOptimization: true,
		EnableParallelBuilds:    true,
		MaxParallelBuilds:       3,
		EnableFallbacks:         true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Test each function
	for _, fn := range functions {
		t.Run(fn.name, func(t *testing.T) {
			input := &runtime.BuildInput{
				FunctionID: fn.name,
				Handler:    fn.handler,
				CfgPath:    filepath.Join(tempDir, fn.name, "pyproject.toml"),
			}

			// Should require initial build
			shouldRebuild := builder.ShouldRebuild(input.FunctionID, input.Handler)
			if !shouldRebuild {
				t.Errorf("Function %s should require initial build", fn.name)
			}

			// Simulate build completion
			entry := &CacheEntry{
				FunctionID:   input.FunctionID,
				LastBuild:    time.Now(),
				FileHashes:   map[string]string{"handler.py": "hash"},
				Dependencies: fn.deps,
			}
			builder.buildCache.Set(input.FunctionID, entry)

			// Should not require rebuild after caching
			shouldRebuild2 := builder.ShouldRebuild(input.FunctionID, input.Handler)
			if shouldRebuild2 {
				t.Errorf("Function %s should not require rebuild after caching", fn.name)
			}
		})
	}

	// Test dependency sharing optimization
	t.Run("dependency_sharing", func(t *testing.T) {
		// Functions with shared dependencies should be able to reuse cached dependencies
		sharedDeps := []string{"fastapi"}

		for _, fn := range functions {
			for _, dep := range fn.deps {
				for _, sharedDep := range sharedDeps {
					if dep == sharedDep {
						// This function uses a shared dependency
						// In a real implementation, we would verify that the dependency
						// cache is shared between functions
						t.Logf("Function %s uses shared dependency %s", fn.name, dep)
					}
				}
			}
		}
	})
}

// TestPerformanceImprovements tests performance improvements and build time reductions
func TestPerformanceImprovements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "perf_project")
	setupTestProject(t, projectDir, "workspace")

	// Test with optimizations disabled
	configNoOpt := IncrementalBuilderConfig{
		CacheDir:                filepath.Join(tempDir, "cache_no_opt"),
		ArtifactDir:             filepath.Join(tempDir, "artifacts_no_opt"),
		MaxCacheSize:            10 * 1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableBuildOptimization: false,
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
	}

	builderNoOpt, err := NewIncrementalBuilder(configNoOpt)
	if err != nil {
		t.Fatalf("Failed to create builder without optimizations: %v", err)
	}

	// Test with optimizations enabled
	configOpt := IncrementalBuilderConfig{
		CacheDir:                filepath.Join(tempDir, "cache_opt"),
		ArtifactDir:             filepath.Join(tempDir, "artifacts_opt"),
		MaxCacheSize:            10 * 1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableBuildOptimization: true,
		EnableParallelBuilds:    true,
		MaxParallelBuilds:       4,
	}

	builderOpt, err := NewIncrementalBuilder(configOpt)
	if err != nil {
		t.Fatalf("Failed to create builder with optimizations: %v", err)
	}

	input := &runtime.BuildInput{
		FunctionID: "perf-function",
		Handler:    "src/mypackage/handler",
		CfgPath:    filepath.Join(projectDir, "pyproject.toml"),
	}

	// Measure rebuild decision time (should be fast with optimizations)
	start := time.Now()
	for i := 0; i < 100; i++ {
		builderOpt.ShouldRebuild(input.FunctionID, input.Handler)
	}
	optimizedTime := time.Since(start)

	start = time.Now()
	for i := 0; i < 100; i++ {
		builderNoOpt.ShouldRebuild(input.FunctionID, input.Handler)
	}
	unoptimizedTime := time.Since(start)

	t.Logf("Optimized rebuild decisions: %v", optimizedTime)
	t.Logf("Unoptimized rebuild decisions: %v", unoptimizedTime)

	// Test cache performance
	entry := &CacheEntry{
		FunctionID:   input.FunctionID,
		LastBuild:    time.Now(),
		FileHashes:   map[string]string{"handler.py": "hash"},
		Dependencies: []string{"requests"},
	}

	// Measure cache operations
	start = time.Now()
	for i := 0; i < 1000; i++ {
		builderOpt.buildCache.Set(fmt.Sprintf("function-%d", i), entry)
	}
	cacheWriteTime := time.Since(start)

	start = time.Now()
	for i := 0; i < 1000; i++ {
		builderOpt.buildCache.Get(fmt.Sprintf("function-%d", i))
	}
	cacheReadTime := time.Since(start)

	t.Logf("Cache write time (1000 entries): %v", cacheWriteTime)
	t.Logf("Cache read time (1000 entries): %v", cacheReadTime)

	// Cache operations should be reasonably fast
	if cacheWriteTime > time.Second {
		t.Error("Cache write operations are too slow")
	}
	if cacheReadTime > time.Second {
		t.Error("Cache read operations are too slow")
	}
}

// TestComplexProjectLayouts tests various complex project layouts
func TestComplexProjectLayouts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	testCases := []struct {
		name          string
		setupFunc     func(string)
		expectedType  LayoutType
		shouldSucceed bool
	}{
		{
			name:          "monorepo_structure",
			setupFunc:     setupMonorepoProject,
			expectedType:  LayoutTypeWorkspace,
			shouldSucceed: true,
		},
		{
			name:          "django_project",
			setupFunc:     setupDjangoProject,
			expectedType:  LayoutTypeFlat,
			shouldSucceed: true,
		},
		{
			name:          "poetry_project",
			setupFunc:     setupPoetryProject,
			expectedType:  LayoutTypeWorkspace,
			shouldSucceed: true,
		},
		{
			name:          "legacy_project",
			setupFunc:     setupLegacyProject,
			expectedType:  LayoutTypeLegacy,
			shouldSucceed: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := filepath.Join(tempDir, tc.name)
			tc.setupFunc(projectDir)

			config := IncrementalBuilderConfig{
				CacheDir:                  filepath.Join(tempDir, "cache", tc.name),
				ArtifactDir:               filepath.Join(tempDir, "artifacts", tc.name),
				MaxCacheSize:              10 * 1024 * 1024,
				MaxCacheAge:               time.Hour,
				EnableBuildOptimization:   true,
				EnableFallbacks:           true,
				EnableDeprecationWarnings: true,
			}

			builder, err := NewIncrementalBuilder(config)
			if err != nil {
				t.Fatalf("Failed to create incremental builder: %v", err)
			}

			// Test layout detection for complex project
			handlerPath := getComplexProjectHandlerPath(tc.name)
			layout, err := builder.layoutDetector.DetectLayout(handlerPath)

			if tc.shouldSucceed {
				if err != nil {
					t.Fatalf("Layout detection failed for %s: %v", tc.name, err)
				}
				if layout.Type != tc.expectedType {
					t.Errorf("Expected layout type %s for %s, got %s", tc.expectedType, tc.name, layout.Type)
				}
			} else {
				if err == nil {
					t.Errorf("Expected layout detection to fail for %s", tc.name)
				}
			}
		})
	}
}

// TestConcurrentBuildScenarios tests concurrent build scenarios
func TestConcurrentBuildScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            50 * 1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableBuildOptimization: true,
		EnableParallelBuilds:    true,
		MaxParallelBuilds:       8,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Test concurrent rebuild decisions
	var wg sync.WaitGroup
	numFunctions := 50
	numConcurrent := 10

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numFunctions; j++ {
				functionID := fmt.Sprintf("worker-%d-function-%d", workerID, j)
				handler := fmt.Sprintf("handler%d", j)

				// Multiple operations per function
				shouldRebuild := builder.ShouldRebuild(functionID, handler)
				if !shouldRebuild {
					t.Errorf("Function %s should require initial rebuild", functionID)
				}

				// Add to cache
				entry := &CacheEntry{
					FunctionID:   functionID,
					LastBuild:    time.Now(),
					FileHashes:   map[string]string{"handler.py": "hash"},
					Dependencies: []string{"requests"},
				}
				builder.buildCache.Set(functionID, entry)

				// Should not rebuild after caching
				shouldRebuild2 := builder.ShouldRebuild(functionID, handler)
				if shouldRebuild2 {
					t.Errorf("Function %s should not require rebuild after caching", functionID)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify cache integrity after concurrent operations
	stats := builder.buildCache.GetStats()
	expectedEntries := numConcurrent * numFunctions
	if stats.TotalEntries != expectedEntries {
		t.Errorf("Expected %d cache entries, got %d", expectedEntries, stats.TotalEntries)
	}
}

// Helper functions for setting up test projects

func setupTestProject(t *testing.T, projectDir, layout string) {
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	switch layout {
	case "workspace":
		setupWorkspaceProject(projectDir)
	case "flat":
		setupFlatProject(projectDir)
	case "nested":
		setupNestedProject(projectDir)
	default:
		t.Fatalf("Unknown project layout: %s", layout)
	}
}

func setupWorkspaceProject(projectDir string) {
	// Create pyproject.toml
	pyprojectContent := `[project]
name = "mypackage"
version = "0.1.0"
dependencies = ["requests"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"`

	os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644)

	// Create source structure
	srcDir := filepath.Join(projectDir, "src", "mypackage")
	os.MkdirAll(srcDir, 0755)

	handlerContent := `def handler(event, context):
    return {"statusCode": 200, "body": "Hello World"}`
	os.WriteFile(filepath.Join(srcDir, "handler.py"), []byte(handlerContent), 0644)
	os.WriteFile(filepath.Join(srcDir, "__init__.py"), []byte(""), 0644)
}

func setupFlatProject(projectDir string) {
	os.MkdirAll(projectDir, 0755)

	// Create requirements.txt
	os.WriteFile(filepath.Join(projectDir, "requirements.txt"), []byte("requests\nclick"), 0644)

	// Create handler
	handlerContent := `def handler(event, context):
    return {"statusCode": 200, "body": "Flat project"}`
	os.WriteFile(filepath.Join(projectDir, "handler.py"), []byte(handlerContent), 0644)
}

func setupNestedProject(projectDir string) {
	// Create nested structure
	nestedDir := filepath.Join(projectDir, "app", "functions", "api")
	os.MkdirAll(nestedDir, 0755)

	// Create pyproject.toml at root
	pyprojectContent := `[project]
name = "nested-project"
version = "0.1.0"
dependencies = ["fastapi"]`
	os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644)

	// Create handler in nested location
	handlerContent := `def handler(event, context):
    return {"statusCode": 200, "body": "Nested project"}`
	os.WriteFile(filepath.Join(nestedDir, "handler.py"), []byte(handlerContent), 0644)
}

func setupTestProjectWithDeps(t *testing.T, projectDir string, deps []string) {
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	depList := `"` + strings.Join(deps, `", "`) + `"`
	pyprojectContent := fmt.Sprintf(`[project]
name = "test-package"
version = "0.1.0"
dependencies = [%s]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"`, depList)

	os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644)
	os.WriteFile(filepath.Join(projectDir, "handler.py"), []byte("def handler(): pass"), 0644)
}

func setupMonorepoProject(projectDir string) {
	os.MkdirAll(projectDir, 0755)

	// Root pyproject.toml
	rootPyproject := `[project]
name = "monorepo"
version = "0.1.0"`
	os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(rootPyproject), 0644)

	// Service directories
	services := []string{"auth", "api", "worker"}
	for _, service := range services {
		serviceDir := filepath.Join(projectDir, "services", service)
		os.MkdirAll(serviceDir, 0755)

		servicePyproject := fmt.Sprintf(`[project]
name = "%s-service"
version = "0.1.0"
dependencies = ["fastapi"]`, service)
		os.WriteFile(filepath.Join(serviceDir, "pyproject.toml"), []byte(servicePyproject), 0644)
		os.WriteFile(filepath.Join(serviceDir, "handler.py"), []byte("def handler(): pass"), 0644)
	}
}

func setupDjangoProject(projectDir string) {
	os.MkdirAll(projectDir, 0755)

	// Django project structure
	files := map[string]string{
		"manage.py":               "#!/usr/bin/env python",
		"myproject/settings.py":   "DEBUG = True",
		"myproject/urls.py":       "urlpatterns = []",
		"myapp/models.py":         "from django.db import models",
		"myapp/views.py":          "from django.http import HttpResponse",
		"myapp/lambda_handler.py": "def handler(event, context): pass",
		"requirements.txt":        "django>=4.0",
	}

	for filePath, content := range files {
		fullPath := filepath.Join(projectDir, filePath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}
}

func setupPoetryProject(projectDir string) {
	os.MkdirAll(projectDir, 0755)

	pyprojectContent := `[tool.poetry]
name = "poetry-project"
version = "0.1.0"
description = "A Poetry project"

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.28.0"

[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"`

	os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644)
	os.WriteFile(filepath.Join(projectDir, "poetry.lock"), []byte("# Poetry lock file"), 0644)

	// Create source structure
	srcDir := filepath.Join(projectDir, "poetry_project")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "handler.py"), []byte("def handler(): pass"), 0644)
}

func setupLegacyProject(projectDir string) {
	os.MkdirAll(projectDir, 0755)

	// Legacy setup.py
	setupContent := `from setuptools import setup
setup(name="legacy-project", version="0.1.0")`
	os.WriteFile(filepath.Join(projectDir, "setup.py"), []byte(setupContent), 0644)
	os.WriteFile(filepath.Join(projectDir, "requirements.txt"), []byte("requests"), 0644)
	os.WriteFile(filepath.Join(projectDir, "handler.py"), []byte("def handler(): pass"), 0644)
}

func getHandlerPath(layout string) string {
	switch layout {
	case "workspace":
		return "src/mypackage/handler"
	case "flat":
		return "handler"
	case "nested":
		return "app/functions/api/handler"
	default:
		return "handler"
	}
}

func getComplexProjectHandlerPath(projectType string) string {
	switch projectType {
	case "monorepo_structure":
		return "services/api/handler"
	case "django_project":
		return "myapp/lambda_handler"
	case "poetry_project":
		return "poetry_project/handler"
	case "legacy_project":
		return "handler"
	default:
		return "handler"
	}
}
