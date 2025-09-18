package python

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// TestCacheIsWorking validates that the caching system is functioning correctly
func TestCacheIsWorking(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	cacheDir := filepath.Join(tempDir, "cache")

	// Create a simple test project
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	pyprojectContent := `
[project]
name = "cache-test"
version = "0.1.0"
dependencies = ["requests>=2.31.0"]
`
	if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	handlerContent := `
import json

def lambda_handler(event, context):
    return {'statusCode': 200, 'body': json.dumps({'message': 'cache test'})}
`
	handlerPath := filepath.Join(projectDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create Python runtime with caching enabled
	pythonRuntime, err := NewWithCache(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}

	// Create build input
	properties := map[string]interface{}{
		"architecture": "x86_64",
		"container":    false,
	}
	propertiesJSON, _ := json.Marshal(properties)

	buildInput := &runtime.BuildInput{
		CfgPath:    filepath.Join(projectDir, "sst.config.ts"), // ResolveRootDir takes the dir of this path
		FunctionID: "cache-test-function",
		Handler:    "handler.lambda_handler",
		Runtime:    "python3.12",
		Properties: propertiesJSON,
	}

	ctx := context.Background()

	t.Log("=== Testing Cache Functionality ===")

	// First build - should be a cache miss
	t.Log("1. First build (cache miss expected)")
	start := time.Now()
	shouldRebuild := pythonRuntime.ShouldRebuild("cache-test-function", "handler.lambda_handler")
	firstBuildTime := time.Since(start)

	if !shouldRebuild {
		t.Error("Expected first build to require rebuild (cache miss)")
	}
	t.Logf("   First build check took: %v", firstBuildTime)

	// Perform the actual build
	buildStart := time.Now()
	_, err = pythonRuntime.Build(ctx, buildInput)
	if err != nil {
		t.Fatalf("First build failed: %v", err)
	}
	actualBuildTime := time.Since(buildStart)
	t.Logf("   First build completed in: %v", actualBuildTime)

	// Second build - should be a cache hit
	t.Log("2. Second build (cache hit expected)")
	start = time.Now()
	shouldRebuild = pythonRuntime.ShouldRebuild("cache-test-function", "handler.lambda_handler")
	secondBuildTime := time.Since(start)

	if shouldRebuild {
		t.Error("Expected second build to use cache (cache hit)")
	}
	t.Logf("   Second build check took: %v", secondBuildTime)

	// Verify cache stats
	stats := pythonRuntime.GetCacheStats()
	if stats == nil {
		t.Error("Expected cache stats to be available")
	} else {
		t.Logf("   Cache stats: %d entries, hit rate: %.2f%%",
			stats.TotalEntries, stats.Metrics.getHitRate())
	}

	// Test cache invalidation by modifying the handler file
	t.Log("3. Testing cache invalidation")
	modifiedContent := `
import json

def lambda_handler(event, context):
    # Modified handler
    return {'statusCode': 200, 'body': json.dumps({'message': 'cache test modified'})}
`
	if err := os.WriteFile(handlerPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify handler: %v", err)
	}

	// Should now require rebuild due to file change
	start = time.Now()
	shouldRebuild = pythonRuntime.ShouldRebuild("cache-test-function", "handler.lambda_handler")
	invalidationTime := time.Since(start)

	if !shouldRebuild {
		t.Error("Expected rebuild after file modification (cache invalidation)")
	}
	t.Logf("   Cache invalidation check took: %v", invalidationTime)

	// Test cache clearing
	t.Log("4. Testing cache clearing")
	err = pythonRuntime.ClearCache()
	if err != nil {
		t.Errorf("Failed to clear cache: %v", err)
	}

	// After clearing, should require rebuild
	shouldRebuild = pythonRuntime.ShouldRebuild("cache-test-function", "handler.lambda_handler")
	if !shouldRebuild {
		t.Error("Expected rebuild after cache clear")
	}

	t.Log("=== Cache Validation Complete ===")
	t.Log("✅ Cache miss detection: WORKING")
	t.Log("✅ Cache hit detection: WORKING")
	t.Log("✅ Cache invalidation: WORKING")
	t.Log("✅ Cache clearing: WORKING")
}

// TestCachePerformance measures basic cache performance
func TestCachePerformance(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: cacheDir,
		MaxSize:  100,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	t.Log("=== Cache Performance Test ===")

	// Test cache set performance
	entry := &CacheEntry{
		FunctionID:   "perf-test",
		BuildTime:    time.Now(),
		FileHashes:   map[string]string{"handler.py": "testhash"},
		Dependencies: []string{"handler.py"},
	}

	start := time.Now()
	err = cache.Set("perf-test", entry)
	setTime := time.Since(start)

	if err != nil {
		t.Fatalf("Cache set failed: %v", err)
	}
	t.Logf("Cache SET took: %v", setTime)

	// Test cache get performance
	start = time.Now()
	_, exists := cache.Get("perf-test")
	getTime := time.Since(start)

	if !exists {
		t.Fatal("Cache entry not found")
	}
	t.Logf("Cache GET took: %v", getTime)

	// Performance thresholds (reasonable expectations)
	if setTime > 100*time.Millisecond {
		t.Errorf("Cache SET too slow: %v (expected < 100ms)", setTime)
	}

	if getTime > 10*time.Millisecond {
		t.Errorf("Cache GET too slow: %v (expected < 10ms)", getTime)
	}

	t.Log("✅ Cache performance: ACCEPTABLE")
}
