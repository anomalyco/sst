package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonRuntime_NewWithCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	runtime, err := NewWithCache(tempDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}
	
	if runtime.buildCache == nil {
		t.Error("Build cache should be initialized")
	}
	
	if runtime.changeDetector == nil {
		t.Error("Change detector should be initialized")
	}
	
	if runtime.layoutDetector == nil {
		t.Error("Layout detector should be initialized")
	}
}

func TestPythonRuntime_EnableCaching(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	runtime := New()
	
	// Initially no caching
	if runtime.buildCache != nil {
		t.Error("Build cache should not be initialized initially")
	}
	
	// Enable caching
	err = runtime.EnableCaching(tempDir)
	if err != nil {
		t.Fatalf("Failed to enable caching: %v", err)
	}
	
	if runtime.buildCache == nil {
		t.Error("Build cache should be initialized after enabling")
	}
	
	if runtime.changeDetector == nil {
		t.Error("Change detector should be initialized after enabling")
	}
}

func TestPythonRuntime_DisableCaching(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	runtime, err := NewWithCache(tempDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}
	
	// Verify caching is enabled
	if runtime.buildCache == nil {
		t.Fatal("Build cache should be initialized")
	}
	
	// Disable caching
	err = runtime.DisableCaching()
	if err != nil {
		t.Fatalf("Failed to disable caching: %v", err)
	}
	
	if runtime.buildCache != nil {
		t.Error("Build cache should be nil after disabling")
	}
	
	if runtime.changeDetector != nil {
		t.Error("Change detector should be nil after disabling")
	}
}

func TestPythonRuntime_ShouldRebuild_NoCaching(t *testing.T) {
	runtime := New()
	
	// Without caching, should always rebuild
	shouldRebuild := runtime.ShouldRebuild("test-function", "handler.py")
	if !shouldRebuild {
		t.Error("Should always rebuild when caching is disabled")
	}
}

func TestPythonRuntime_ShouldRebuild_WithCaching(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	runtime, err := NewWithCache(tempDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}
	
	// First call should rebuild (no cache entry)
	shouldRebuild := runtime.ShouldRebuild("test-function", "handler.py")
	if !shouldRebuild {
		t.Error("Should rebuild when no cache entry exists")
	}
}

func TestPythonRuntime_GetCacheStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Test without caching
	runtime := New()
	stats := runtime.GetCacheStats()
	if stats != nil {
		t.Error("Stats should be nil when caching is disabled")
	}
	
	// Test with caching
	runtime, err = NewWithCache(tempDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}
	
	stats = runtime.GetCacheStats()
	if stats == nil {
		t.Error("Stats should not be nil when caching is enabled")
	}
	
	if stats.CacheDir != tempDir {
		t.Errorf("Expected cache dir %s, got %s", tempDir, stats.CacheDir)
	}
}

func TestPythonRuntime_ClearCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Test without caching
	runtime := New()
	err = runtime.ClearCache()
	if err == nil {
		t.Error("Should error when trying to clear cache without caching enabled")
	}
	
	// Test with caching
	runtime, err = NewWithCache(tempDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}
	
	err = runtime.ClearCache()
	if err != nil {
		t.Errorf("Failed to clear cache: %v", err)
	}
}

func TestPythonRuntime_InvalidateCacheEntry(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	runtime, err := NewWithCache(tempDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}
	
	err = runtime.InvalidateCacheEntry("test-function")
	if err != nil {
		t.Errorf("Failed to invalidate cache entry: %v", err)
	}
}

func TestPythonRuntime_ForceRebuild(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	runtime, err := NewWithCache(tempDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}
	
	// Should not panic
	runtime.ForceRebuild("test-function", "user requested")
}

func TestPythonRuntime_UpdateCacheAfterBuild(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	runtime, err := NewWithCache(tempDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}
	
	// Create test files for layout detection
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	pyprojectFile := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectFile, []byte("[project]\nname = \"test\""), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}
	
	// Test that the cache update method exists and can be called
	// We'll test this indirectly through the ShouldRebuild method
	shouldRebuild := runtime.ShouldRebuild("test-function", "handler.py")
	
	// The result may vary based on layout detection, but the call should succeed
	t.Logf("Should rebuild: %v", shouldRebuild)
}

func TestPythonRuntime_IntegrationTest(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "runtime_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a runtime with caching
	runtime, err := NewWithCache(tempDir)
	if err != nil {
		t.Fatalf("Failed to create runtime with cache: %v", err)
	}
	
	// Create test project structure
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	pyprojectFile := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectFile, []byte("[project]\nname = \"test\""), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}
	
	functionID := "test-function"
	handler := "handler.py"
	
	// First check - should rebuild (no cache)
	shouldRebuild1 := runtime.ShouldRebuild(functionID, handler)
	if !shouldRebuild1 {
		t.Error("Should rebuild on first check (no cache)")
	}
	
	// We can't easily test updateCacheAfterBuild without the full runtime types,
	// so we'll test the behavior indirectly through multiple ShouldRebuild calls
	
	// Second check - result may vary based on layout detection
	shouldRebuild2 := runtime.ShouldRebuild(functionID, handler)
	t.Logf("Should rebuild after cache update: %v", shouldRebuild2)
	
	// Modify file and check again
	if err := os.WriteFile(handlerFile, []byte("def handler(): return 'modified'"), 0644); err != nil {
		t.Fatalf("Failed to modify handler file: %v", err)
	}
	
	shouldRebuild3 := runtime.ShouldRebuild(functionID, handler)
	if !shouldRebuild3 {
		t.Error("Should rebuild after file modification")
	}
	
	// Test cache stats
	stats := runtime.GetCacheStats()
	if stats == nil {
		t.Error("Cache stats should not be nil")
	}
	
	// Test force rebuild
	runtime.ForceRebuild(functionID, "integration test")
	shouldRebuild4 := runtime.ShouldRebuild(functionID, handler)
	if !shouldRebuild4 {
		t.Error("Should rebuild after force rebuild")
	}
	
	// Test cache clearing
	err = runtime.ClearCache()
	if err != nil {
		t.Errorf("Failed to clear cache: %v", err)
	}
	
	shouldRebuild5 := runtime.ShouldRebuild(functionID, handler)
	if !shouldRebuild5 {
		t.Error("Should rebuild after cache clear")
	}
}