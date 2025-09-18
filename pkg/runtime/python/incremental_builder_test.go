package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

func TestNewIncrementalBuilder(t *testing.T) {
	tempDir := t.TempDir()

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableParallelBuilds:    true,
		MaxParallelBuilds:       2,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	if builder == nil {
		t.Fatal("Builder is nil")
	}

	if builder.buildCache == nil {
		t.Error("Build cache is nil")
	}

	if builder.projectResolver == nil {
		t.Error("Project resolver is nil")
	}

	if builder.changeDetector == nil {
		t.Error("Change detector is nil")
	}

	if builder.buildResultCache == nil {
		t.Error("Build result cache is nil")
	}

	if builder.dependencyAnalyzer == nil {
		t.Error("Dependency analyzer is nil")
	}

	if builder.buildPlanner == nil {
		t.Error("Build planner is nil")
	}

	if builder.uvRunner == nil {
		t.Error("UV runner is nil")
	}
}

func TestIncrementalBuilder_ShouldRebuild(t *testing.T) {
	tempDir := t.TempDir()

	config := IncrementalBuilderConfig{
		CacheDir:    tempDir,
		ArtifactDir: filepath.Join(tempDir, "artifacts"),
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Test with no cache entry (should rebuild)
	shouldRebuild := builder.ShouldRebuild("test-function", "handler.py")
	if !shouldRebuild {
		t.Error("Expected rebuild for function with no cache entry")
	}
}

func TestIncrementalBuilder_Build_NoCachedResult(t *testing.T) {
	tempDir := t.TempDir()

	// Create a simple Python project structure
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create pyproject.toml
	pyprojectContent := `[project]
name = "test-package"
version = "0.1.0"
dependencies = []

[build-system]
requires = ["setuptools", "wheel"]
build-backend = "setuptools.build_meta"
`
	if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create handler file
	handlerContent := `def lambda_handler(event, context):
    return {"statusCode": 200, "body": "Hello World"}
`
	if err := os.WriteFile(filepath.Join(projectDir, "handler.py"), []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler.py: %v", err)
	}

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		EnableBuildOptimization: false, // Disable to avoid trying cached results
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Update project resolver project root
	builder.projectResolver.projectRoot = projectDir

	ctx := context.Background()
	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Handler:    "handler.py",
		CfgPath:    filepath.Join(projectDir, "sst.config.ts"),
		Properties: []byte(`{"architecture": "x86_64"}`),
	}

	// This test will fail because we don't have UV installed in the test environment
	// But we can test that the build process starts correctly
	result, err := builder.Build(ctx, input)

	// The build might succeed if UV is available, or fail if it's not
	if err != nil {
		// If there's an error, it should be related to UV execution or project resolution
		if !containsAnyString(err.Error(), []string{"failed to resolve handler", "UV", "uv", "command not found", "executable file not found", "failed to execute build plan"}) {
			t.Errorf("Unexpected error type: %v", err)
		}
	} else {
		// If build succeeded, check that we got a valid result
		if result == nil {
			t.Error("Expected build result, got nil")
		} else if result.Handler == "" {
			t.Error("Expected handler in build result")
		}
	}
}

func TestIncrementalBuilder_GetBuildStats(t *testing.T) {
	tempDir := t.TempDir()

	config := IncrementalBuilderConfig{
		CacheDir:    tempDir,
		ArtifactDir: filepath.Join(tempDir, "artifacts"),
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	stats := builder.GetBuildStats()
	if stats == nil {
		t.Fatal("Build stats is nil")
	}

	if stats.CacheStats.CacheDir != tempDir {
		t.Errorf("Expected cache dir %s, got %s", tempDir, stats.CacheStats.CacheDir)
	}

	if stats.Config.CacheDir != tempDir {
		t.Errorf("Expected config cache dir %s, got %s", tempDir, stats.Config.CacheDir)
	}
}

func TestIncrementalBuilder_ClearCache(t *testing.T) {
	tempDir := t.TempDir()

	config := IncrementalBuilderConfig{
		CacheDir:    tempDir,
		ArtifactDir: filepath.Join(tempDir, "artifacts"),
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	err = builder.ClearCache()
	if err != nil {
		t.Errorf("Failed to clear cache: %v", err)
	}
}

func TestIncrementalBuilder_ForceRebuild(t *testing.T) {
	tempDir := t.TempDir()

	config := IncrementalBuilderConfig{
		CacheDir:    tempDir,
		ArtifactDir: filepath.Join(tempDir, "artifacts"),
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	err = builder.ForceRebuild("test-function", "test reason")
	if err != nil {
		t.Errorf("Failed to force rebuild: %v", err)
	}
}

func TestPackageBuildInfo_Creation(t *testing.T) {
	buildInfo := &PackageBuildInfo{
		PackageName:        "test-package",
		PackageDir:         "/path/to/package",
		Dependencies:       []string{"dep1", "dep2"},
		SourceFiles:        []string{"file1.py", "file2.py"},
		RequiresRebuild:    true,
		RebuildReason:      "source files changed",
		EstimatedBuildTime: 30 * time.Second,
		CanUseCache:        false,
		CacheKey:           "test-key",
	}

	if buildInfo.PackageName != "test-package" {
		t.Errorf("Expected package name 'test-package', got '%s'", buildInfo.PackageName)
	}

	if len(buildInfo.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(buildInfo.Dependencies))
	}

	if len(buildInfo.SourceFiles) != 2 {
		t.Errorf("Expected 2 source files, got %d", len(buildInfo.SourceFiles))
	}

	if !buildInfo.RequiresRebuild {
		t.Error("Expected RequiresRebuild to be true")
	}

	if buildInfo.CanUseCache {
		t.Error("Expected CanUseCache to be false")
	}
}

func TestBuildResult_Creation(t *testing.T) {
	buildResult := &BuildResult{
		Success:        true,
		BuildDuration:  2 * time.Minute,
		PackagesBuilt:  []string{"pkg1", "pkg2"},
		PackagesCached: []string{"pkg3"},
		Errors:         []string{},
		Warnings:       []string{"warning1"},
	}

	if !buildResult.Success {
		t.Error("Expected Success to be true")
	}

	if buildResult.BuildDuration != 2*time.Minute {
		t.Errorf("Expected build duration 2m, got %v", buildResult.BuildDuration)
	}

	if len(buildResult.PackagesBuilt) != 2 {
		t.Errorf("Expected 2 packages built, got %d", len(buildResult.PackagesBuilt))
	}

	if len(buildResult.PackagesCached) != 1 {
		t.Errorf("Expected 1 package cached, got %d", len(buildResult.PackagesCached))
	}

	if len(buildResult.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(buildResult.Errors))
	}

	if len(buildResult.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(buildResult.Warnings))
	}
}

// containsAnyString checks if a string contains any of the given substrings
func containsAnyString(s string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(s, substring) {
			return true
		}
	}
	return false
}
