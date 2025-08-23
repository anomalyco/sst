package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// TestIncrementalBuilder_SelectivePackageBuilding tests selective package building
func TestIncrementalBuilder_SelectivePackageBuilding(t *testing.T) {
	tempDir := t.TempDir()

	// Create a multi-package project structure
	projectDir := filepath.Join(tempDir, "project")
	packages := []string{"package1", "package2", "package3"}

	for _, pkg := range packages {
		pkgDir := filepath.Join(projectDir, pkg)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatalf("Failed to create package directory %s: %v", pkg, err)
		}

		// Create pyproject.toml for each package
		pyprojectContent := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
dependencies = []

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"`, pkg)

		pyprojectPath := filepath.Join(pkgDir, "pyproject.toml")
		if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
			t.Fatalf("Failed to create pyproject.toml for %s: %v", pkg, err)
		}

		// Create source files
		srcDir := filepath.Join(pkgDir, "src", pkg)
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatalf("Failed to create src directory for %s: %v", pkg, err)
		}

		handlerContent := fmt.Sprintf(`def handler():
    return "%s handler"`, pkg)
		handlerPath := filepath.Join(srcDir, "handler.py")
		if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
			t.Fatalf("Failed to create handler for %s: %v", pkg, err)
		}
	}

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableParallelBuilds:    false, // Disable for predictable testing
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Test building each package
	for _, pkg := range packages {
		t.Run(fmt.Sprintf("build_%s", pkg), func(t *testing.T) {
			input := &runtime.BuildInput{
				FunctionID: fmt.Sprintf("test-%s", pkg),
				Handler:    fmt.Sprintf("%s/src/%s/handler", pkg, pkg),
				CfgPath:    filepath.Join(projectDir, pkg, "pyproject.toml"),
			}

			// First build should not use cache
			shouldRebuild := builder.ShouldRebuild(input.FunctionID, input.Handler)
			if !shouldRebuild {
				t.Error("First build should require rebuild")
			}

			// Note: We can't actually run the full build without UV installed
			// So we test the components that don't require external tools
		})
	}
}

// TestIncrementalBuilder_DependencyScenarios tests various dependency scenarios
func TestIncrementalBuilder_DependencyScenarios(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name          string
		dependencies  []string
		expectRebuild bool
	}{
		{
			name:          "no dependencies",
			dependencies:  []string{},
			expectRebuild: true, // First build always rebuilds
		},
		{
			name:          "common dependencies",
			dependencies:  []string{"requests", "click"},
			expectRebuild: true,
		},
		{
			name:          "many dependencies",
			dependencies:  []string{"requests", "click", "flask", "sqlalchemy", "pytest", "black", "mypy"},
			expectRebuild: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := filepath.Join(tempDir, tc.name)
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project directory: %v", err)
			}

			// Create pyproject.toml with dependencies
			depList := ""
			if len(tc.dependencies) > 0 {
				depList = `"` + strings.Join(tc.dependencies, `", "`) + `"`
			}

			pyprojectContent := fmt.Sprintf(`[project]
name = "test-package"
version = "0.1.0"
dependencies = [%s]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"`, depList)

			pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
			if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
				t.Fatalf("Failed to create pyproject.toml: %v", err)
			}

			// Create handler
			handlerPath := filepath.Join(projectDir, "handler.py")
			if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
				t.Fatalf("Failed to create handler: %v", err)
			}

			config := IncrementalBuilderConfig{
				CacheDir:                filepath.Join(tempDir, "cache", tc.name),
				ArtifactDir:             filepath.Join(tempDir, "artifacts", tc.name),
				MaxCacheSize:            1024 * 1024,
				MaxCacheAge:             time.Hour,
				EnableBuildOptimization: true,
			}

			builder, err := NewIncrementalBuilder(config)
			if err != nil {
				t.Fatalf("Failed to create incremental builder: %v", err)
			}

			input := &runtime.BuildInput{
				FunctionID: fmt.Sprintf("test-%s", tc.name),
				Handler:    "handler",
				CfgPath:    pyprojectPath,
			}

			shouldRebuild := builder.ShouldRebuild(input.FunctionID, input.Handler)
			if shouldRebuild != tc.expectRebuild {
				t.Errorf("Expected rebuild %v, got %v", tc.expectRebuild, shouldRebuild)
			}
		})
	}
}

// TestIncrementalBuilder_BuildOptimizationDecisions tests build optimization decisions
func TestIncrementalBuilder_BuildOptimizationDecisions(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name                    string
		enableOptimization      bool
		enableParallelBuilds    bool
		maxParallelBuilds       int
		expectedOptimization    bool
		expectedParallelization bool
	}{
		{
			name:                    "optimizations enabled",
			enableOptimization:      true,
			enableParallelBuilds:    true,
			maxParallelBuilds:       4,
			expectedOptimization:    true,
			expectedParallelization: true,
		},
		{
			name:                    "optimizations disabled",
			enableOptimization:      false,
			enableParallelBuilds:    false,
			maxParallelBuilds:       1,
			expectedOptimization:    false,
			expectedParallelization: false,
		},
		{
			name:                    "parallel disabled",
			enableOptimization:      true,
			enableParallelBuilds:    false,
			maxParallelBuilds:       1,
			expectedOptimization:    true,
			expectedParallelization: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := IncrementalBuilderConfig{
				CacheDir:                filepath.Join(tempDir, "cache", tc.name),
				ArtifactDir:             filepath.Join(tempDir, "artifacts", tc.name),
				MaxCacheSize:            1024 * 1024,
				MaxCacheAge:             time.Hour,
				EnableBuildOptimization: tc.enableOptimization,
				EnableParallelBuilds:    tc.enableParallelBuilds,
				MaxParallelBuilds:       tc.maxParallelBuilds,
			}

			builder, err := NewIncrementalBuilder(config)
			if err != nil {
				t.Fatalf("Failed to create incremental builder: %v", err)
			}

			// Check configuration was applied correctly
			if builder.config.EnableBuildOptimization != tc.expectedOptimization {
				t.Errorf("Expected optimization %v, got %v", tc.expectedOptimization, builder.config.EnableBuildOptimization)
			}

			if builder.config.EnableParallelBuilds != tc.expectedParallelization {
				t.Errorf("Expected parallelization %v, got %v", tc.expectedParallelization, builder.config.EnableParallelBuilds)
			}

			if builder.config.MaxParallelBuilds != tc.maxParallelBuilds {
				t.Errorf("Expected max parallel builds %d, got %d", tc.maxParallelBuilds, builder.config.MaxParallelBuilds)
			}
		})
	}
}

// TestIncrementalBuilder_CacheReuse tests cache reuse scenarios
func TestIncrementalBuilder_CacheReuse(t *testing.T) {
	tempDir := t.TempDir()

	// Create project structure
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create pyproject.toml
	pyprojectContent := `[project]
name = "test-package"
version = "0.1.0"
dependencies = ["requests"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"`

	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create handler
	handlerPath := filepath.Join(projectDir, "handler.py")
	handlerContent := `def handler():
    return "hello world"`
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	config := IncrementalBuilderConfig{
		CacheDir:                filepath.Join(tempDir, "cache"),
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableBuildOptimization: true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Handler:    "handler",
		CfgPath:    pyprojectPath,
	}

	// First check - should rebuild (no cache)
	shouldRebuild1 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if !shouldRebuild1 {
		t.Error("First check should require rebuild")
	}

	// Simulate adding to cache
	entry := &CacheEntry{
		FunctionID:   input.FunctionID,
		BuildTime:    time.Now(),
		FileHashes:   map[string]string{handlerPath: "hash123"},
		Dependencies: []string{"requests"},
	}
	builder.buildCache.Set(input.FunctionID, entry)

	// Second check - should not rebuild (cache hit)
	shouldRebuild2 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if shouldRebuild2 {
		t.Error("Second check should not require rebuild (cache hit)")
	}

	// Modify handler file
	modifiedContent := `def handler():
    return "hello modified world"`
	if err := os.WriteFile(handlerPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify handler: %v", err)
	}

	// Third check - should rebuild (file changed)
	shouldRebuild3 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if !shouldRebuild3 {
		t.Error("Third check should require rebuild (file changed)")
	}
}

// TestIncrementalBuilder_ErrorHandlingAndRecovery tests error handling and recovery
func TestIncrementalBuilder_ErrorHandlingAndRecovery(t *testing.T) {
	tempDir := t.TempDir()

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableBuildOptimization: true,
		EnableFallbacks:         true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	t.Run("nonexistent handler", func(t *testing.T) {
		input := &runtime.BuildInput{
			FunctionID: "nonexistent-function",
			Handler:    "nonexistent/handler",
			CfgPath:    "/nonexistent/pyproject.toml",
		}

		// Should handle nonexistent handler gracefully
		shouldRebuild := builder.ShouldRebuild(input.FunctionID, input.Handler)
		if !shouldRebuild {
			t.Error("Should require rebuild for nonexistent handler")
		}
	})

	t.Run("corrupted cache", func(t *testing.T) {
		// Add corrupted entry to cache
		corruptedEntry := &CacheEntry{
			FunctionID:   "corrupted-function",
			BuildTime:    time.Time{}, // Invalid time
			FileHashes:   nil,         // Nil map
			Dependencies: nil,         // Nil slice
		}
		builder.buildCache.Set("corrupted-function", corruptedEntry)

		input := &runtime.BuildInput{
			FunctionID: "corrupted-function",
			Handler:    "handler",
			CfgPath:    "/some/pyproject.toml",
		}

		// Should handle corrupted cache gracefully
		shouldRebuild := builder.ShouldRebuild(input.FunctionID, input.Handler)
		if !shouldRebuild {
			t.Error("Should require rebuild for corrupted cache entry")
		}
	})

	t.Run("permission errors", func(t *testing.T) {
		// Create directory with restricted permissions
		restrictedDir := filepath.Join(tempDir, "restricted")
		if err := os.MkdirAll(restrictedDir, 0000); err != nil {
			t.Fatalf("Failed to create restricted directory: %v", err)
		}
		defer os.Chmod(restrictedDir, 0755) // Restore for cleanup

		input := &runtime.BuildInput{
			FunctionID: "restricted-function",
			Handler:    "restricted/handler",
			CfgPath:    filepath.Join(restrictedDir, "pyproject.toml"),
		}

		// Should handle permission errors gracefully
		shouldRebuild := builder.ShouldRebuild(input.FunctionID, input.Handler)
		if !shouldRebuild {
			t.Error("Should require rebuild when encountering permission errors")
		}
	})
}

// TestIncrementalBuilder_ParallelBuilding tests parallel building functionality
func TestIncrementalBuilder_ParallelBuilding(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple packages for parallel building
	packages := []string{"pkg1", "pkg2", "pkg3", "pkg4"}
	projectDir := filepath.Join(tempDir, "project")

	for _, pkg := range packages {
		pkgDir := filepath.Join(projectDir, pkg)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatalf("Failed to create package directory %s: %v", pkg, err)
		}

		// Create independent packages (no dependencies between them)
		pyprojectContent := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
dependencies = []

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"`, pkg)

		pyprojectPath := filepath.Join(pkgDir, "pyproject.toml")
		if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
			t.Fatalf("Failed to create pyproject.toml for %s: %v", pkg, err)
		}

		handlerPath := filepath.Join(pkgDir, "handler.py")
		if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create handler for %s: %v", pkg, err)
		}
	}

	testCases := []struct {
		name              string
		enableParallel    bool
		maxParallelBuilds int
	}{
		{
			name:              "sequential building",
			enableParallel:    false,
			maxParallelBuilds: 1,
		},
		{
			name:              "parallel building",
			enableParallel:    true,
			maxParallelBuilds: 2,
		},
		{
			name:              "high parallelism",
			enableParallel:    true,
			maxParallelBuilds: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := IncrementalBuilderConfig{
				CacheDir:                filepath.Join(tempDir, "cache", tc.name),
				ArtifactDir:             filepath.Join(tempDir, "artifacts", tc.name),
				MaxCacheSize:            1024 * 1024,
				MaxCacheAge:             time.Hour,
				EnableParallelBuilds:    tc.enableParallel,
				MaxParallelBuilds:       tc.maxParallelBuilds,
				EnableBuildOptimization: true,
			}

			builder, err := NewIncrementalBuilder(config)
			if err != nil {
				t.Fatalf("Failed to create incremental builder: %v", err)
			}

			// Test that configuration is applied correctly
			if builder.config.EnableParallelBuilds != tc.enableParallel {
				t.Errorf("Expected parallel builds %v, got %v", tc.enableParallel, builder.config.EnableParallelBuilds)
			}

			if builder.config.MaxParallelBuilds != tc.maxParallelBuilds {
				t.Errorf("Expected max parallel builds %d, got %d", tc.maxParallelBuilds, builder.config.MaxParallelBuilds)
			}

			// Test building multiple packages
			for _, pkg := range packages {
				input := &runtime.BuildInput{
					FunctionID: fmt.Sprintf("%s-%s", tc.name, pkg),
					Handler:    fmt.Sprintf("%s/handler", pkg),
					CfgPath:    filepath.Join(projectDir, pkg, "pyproject.toml"),
				}

				shouldRebuild := builder.ShouldRebuild(input.FunctionID, input.Handler)
				if !shouldRebuild {
					t.Errorf("Package %s should require rebuild", pkg)
				}
			}
		})
	}
}

// TestIncrementalBuilder_DependencyResolution tests dependency resolution
func TestIncrementalBuilder_DependencyResolution(t *testing.T) {
	tempDir := t.TempDir()

	// Create project with complex dependencies
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create pyproject.toml with various dependency types
	pyprojectContent := `[project]
name = "complex-package"
version = "0.1.0"
dependencies = [
    "requests>=2.25.0",
    "click>=8.0.0",
    "pydantic[email]>=1.8.0",
]

[project.optional-dependencies]
dev = ["pytest>=6.0.0", "black>=21.0.0"]
docs = ["sphinx>=4.0.0", "sphinx-rtd-theme"]

[tool.uv.sources]
local-dep = { path = "../local-package" }
git-dep = { git = "https://github.com/example/repo.git", branch = "main" }

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"`

	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create uv.lock file
	uvLockContent := `# This file is autogenerated by uv
version = 1

[[package]]
name = "requests"
version = "2.28.1"
source = { registry = "https://pypi.org/simple" }

[[package]]
name = "click"
version = "8.1.3"
source = { registry = "https://pypi.org/simple" }`

	uvLockPath := filepath.Join(projectDir, "uv.lock")
	if err := os.WriteFile(uvLockPath, []byte(uvLockContent), 0644); err != nil {
		t.Fatalf("Failed to create uv.lock: %v", err)
	}

	// Create handler
	handlerPath := filepath.Join(projectDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	config := IncrementalBuilderConfig{
		CacheDir:                filepath.Join(tempDir, "cache"),
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableBuildOptimization: true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	input := &runtime.BuildInput{
		FunctionID: "complex-function",
		Handler:    "handler",
		CfgPath:    pyprojectPath,
	}

	// Should require rebuild for complex dependencies
	shouldRebuild := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if !shouldRebuild {
		t.Error("Should require rebuild for complex dependencies")
	}

	// Test dependency change detection
	entry := &CacheEntry{
		FunctionID: input.FunctionID,
		BuildTime:  time.Now(),
		FileHashes: map[string]string{
			handlerPath:   "handler-hash",
			pyprojectPath: "pyproject-hash",
			uvLockPath:    "uvlock-hash",
		},
		Dependencies: []string{"requests", "click", "pydantic"},
	}
	builder.buildCache.Set(input.FunctionID, entry)

	// Should not rebuild if nothing changed
	shouldRebuild2 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if shouldRebuild2 {
		t.Error("Should not rebuild if dependencies haven't changed")
	}

	// Modify uv.lock to simulate dependency change
	modifiedUvLock := uvLockContent + `
[[package]]
name = "new-dependency"
version = "1.0.0"
source = { registry = "https://pypi.org/simple" }`

	if err := os.WriteFile(uvLockPath, []byte(modifiedUvLock), 0644); err != nil {
		t.Fatalf("Failed to modify uv.lock: %v", err)
	}

	// Should rebuild after dependency change
	shouldRebuild3 := builder.ShouldRebuild(input.FunctionID, input.Handler)
	if !shouldRebuild3 {
		t.Error("Should rebuild after dependency change")
	}
}

// TestIncrementalBuilder_ConcurrentBuilds tests concurrent build scenarios
func TestIncrementalBuilder_ConcurrentBuilds(t *testing.T) {
	tempDir := t.TempDir()

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableBuildOptimization: true,
		EnableParallelBuilds:    true,
		MaxParallelBuilds:       4,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Test concurrent ShouldRebuild calls
	var wg sync.WaitGroup
	numConcurrent := 20
	results := make(chan bool, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			functionID := fmt.Sprintf("concurrent-function-%d", id)
			handler := fmt.Sprintf("handler%d", id)

			// Multiple calls per goroutine
			for j := 0; j < 10; j++ {
				shouldRebuild := builder.ShouldRebuild(functionID, handler)
				results <- shouldRebuild
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// All calls should complete without deadlock
	resultCount := 0
	for range results {
		resultCount++
	}

	expectedResults := numConcurrent * 10
	if resultCount != expectedResults {
		t.Errorf("Expected %d results, got %d", expectedResults, resultCount)
	}
}

// TestIncrementalBuilder_ProgressReportingComprehensive tests progress reporting integration (comprehensive)
func TestIncrementalBuilder_ProgressReportingComprehensive(t *testing.T) {
	tempDir := t.TempDir()

	// Track progress events
	var progressEvents []ProgressEvent
	var mu sync.Mutex

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:            1024 * 1024,
		MaxCacheAge:             time.Hour,
		EnableProgressReporting: true,
		ProgressCallback: func(event ProgressEvent) {
			mu.Lock()
			progressEvents = append(progressEvents, event)
			mu.Unlock()
		},
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	if builder.progressReporter == nil {
		t.Error("Progress reporter should be initialized")
	}

	// Test progress reporting
	builder.progressReporter.StartStage(StageInit, "Testing progress")
	builder.progressReporter.UpdateProgress(StageInit, "Halfway done")
	builder.progressReporter.CompleteStage(StageInit, "Done")

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	eventCount := len(progressEvents)
	mu.Unlock()

	if eventCount < 3 {
		t.Errorf("Expected at least 3 progress events, got %d", eventCount)
	}

	// Test progress summary
	summary := builder.GetBuildProgress()
	if summary == nil {
		t.Error("Build progress summary should not be nil")
	}

	if progress, exists := summary["progress"]; !exists || progress.(int) < 0 {
		t.Error("Progress summary should contain valid progress")
	}
}

// TestIncrementalBuilder_FallbackIntegrationComprehensive tests fallback integration (comprehensive)
func TestIncrementalBuilder_FallbackIntegrationComprehensive(t *testing.T) {
	tempDir := t.TempDir()

	// Track fallback events
	var fallbackEvents []FallbackEvent
	var mu sync.Mutex

	config := IncrementalBuilderConfig{
		CacheDir:        tempDir,
		ArtifactDir:     filepath.Join(tempDir, "artifacts"),
		MaxCacheSize:    1024 * 1024,
		MaxCacheAge:     time.Hour,
		EnableFallbacks: true,
		FallbackCallback: func(event FallbackEvent) {
			mu.Lock()
			fallbackEvents = append(fallbackEvents, event)
			mu.Unlock()
		},
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	if builder.fallbackManager == nil {
		t.Error("Fallback manager should be initialized")
	}

	// Test fallback history
	history := builder.GetFallbackHistory()
	if history == nil {
		t.Error("Fallback history should not be nil")
	}

	if len(history) != 0 {
		t.Error("Initial fallback history should be empty")
	}

	// Test clearing fallback history
	builder.ClearFallbackHistory()

	clearedHistory := builder.GetFallbackHistory()
	if len(clearedHistory) != 0 {
		t.Error("Fallback history should be empty after clearing")
	}
}
