package python

import (
	"testing"
	"time"
)

// TestCacheIsWorking validates that the caching system is functioning correctly
// This tests the BuildCache directly, not through ShouldRebuild (which is for dev mode file watching)
func TestCacheIsWorking(t *testing.T) {
	tempDir := t.TempDir()

	// Create cache
	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
		MaxSize:  100,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	t.Log("=== Testing Cache Functionality ===")

	// 1. First check - should be a cache miss
	t.Log("1. First check (cache miss expected)")
	_, exists := cache.Get("test-function")
	if exists {
		t.Error("Expected cache miss for new function")
	}
	t.Log("   ✅ Cache miss detection: WORKING")

	// 2. Set cache entry
	t.Log("2. Setting cache entry")
	entry := &CacheEntry{
		FunctionID:   "test-function",
		BuildTime:    time.Now(),
		FileHashes:   map[string]string{"handler.py": "abc123"},
		Dependencies: []string{"handler.py"},
	}
	err = cache.Set("test-function", entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}
	t.Log("   ✅ Cache set: WORKING")

	// 3. Second check - should be a cache hit
	t.Log("3. Second check (cache hit expected)")
	retrieved, exists := cache.Get("test-function")
	if !exists {
		t.Error("Expected cache hit after setting entry")
	}
	if retrieved.FunctionID != "test-function" {
		t.Errorf("Retrieved wrong function ID: %s", retrieved.FunctionID)
	}
	t.Log("   ✅ Cache hit detection: WORKING")

	// 4. Test cache deletion (invalidation)
	t.Log("4. Testing cache deletion")
	err = cache.Delete("test-function")
	if err != nil {
		t.Errorf("Failed to delete cache entry: %v", err)
	}
	_, exists = cache.Get("test-function")
	if exists {
		t.Error("Expected cache miss after deletion")
	}
	t.Log("   ✅ Cache deletion: WORKING")

	// 5. Test cache clearing
	t.Log("5. Testing cache clearing")
	// Set multiple entries
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			FunctionID: "func-" + string(rune('a'+i)),
			BuildTime:  time.Now(),
			FileHashes: map[string]string{"handler.py": "hash"},
		}
		cache.Set(entry.FunctionID, entry)
	}

	// Clear all
	err = cache.Clear()
	if err != nil {
		t.Errorf("Failed to clear cache: %v", err)
	}

	// Verify all cleared
	for i := 0; i < 5; i++ {
		_, exists := cache.Get("func-" + string(rune('a'+i)))
		if exists {
			t.Error("Expected cache miss after clear")
		}
	}
	t.Log("   ✅ Cache clearing: WORKING")

	t.Log("=== Cache Validation Complete ===")
}

// TestCachePerformance measures basic cache performance
func TestCachePerformance(t *testing.T) {
	tempDir := t.TempDir()

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
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
