package python

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestBuildCache_ContentBasedCaching tests the new content-based caching system
func TestBuildCache_ContentBasedCaching(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	// Create test files
	testFile1 := filepath.Join(tempDir, "test1.py")
	testFile2 := filepath.Join(tempDir, "test2.py")

	if err := os.WriteFile(testFile1, []byte("print('hello')"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("print('world')"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create cache
	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          cacheDir,
		MaxSize:           100,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test initial cache miss
	_, exists := cache.Get("test-function")
	if exists {
		t.Error("Expected cache miss for new function")
	}

	// Verify metrics recorded the miss
	if cache.metrics.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", cache.metrics.Misses)
	}

	// Create cache entry with file hashes
	entry := &CacheEntry{
		FunctionID:   "test-function",
		Handler:      "test.handler",
		FileHashes:   make(map[string]string),
		Dependencies: []string{testFile1, testFile2},
		BuildOutput: &CachedBuildOutput{
			Handler:   "test.handler",
			OutputDir: tempDir,
		},
	}

	// Calculate and store file hashes
	if err := cache.UpdateFileHashes(entry, []string{testFile1, testFile2}); err != nil {
		t.Fatalf("Failed to update file hashes: %v", err)
	}

	// Store in cache
	if err := cache.Set("test-function", entry); err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Test cache hit
	cachedEntry, exists := cache.Get("test-function")
	if !exists {
		t.Error("Expected cache hit")
	}
	if cachedEntry.FunctionID != "test-function" {
		t.Errorf("Expected function ID 'test-function', got %s", cachedEntry.FunctionID)
	}

	// Verify metrics recorded the hit
	if cache.metrics.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", cache.metrics.Hits)
	}

	// Test cache validation - should be valid initially
	valid, err := cache.IsValid(cachedEntry)
	if err != nil {
		t.Fatalf("Failed to validate cache: %v", err)
	}
	if !valid {
		t.Error("Expected cache to be valid")
	}

	// Modify a file to invalidate cache
	if err := os.WriteFile(testFile1, []byte("print('modified')"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Test cache invalidation
	valid, err = cache.IsValid(cachedEntry)
	if err != nil {
		t.Fatalf("Failed to validate cache: %v", err)
	}
	if valid {
		t.Error("Expected cache to be invalid after file modification")
	}

	// Verify invalidation was recorded
	if cache.metrics.InvalidationCount == 0 {
		t.Error("Expected invalidation to be recorded")
	}
}

// TestBuildCache_PerformanceMetrics tests cache performance monitoring
func TestBuildCache_PerformanceMetrics(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          cacheDir,
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test initial metrics
	stats := cache.GetStats()
	if stats.Metrics.Hits != 0 || stats.Metrics.Misses != 0 {
		t.Error("Expected zero initial metrics")
	}

	// Perform some cache operations
	for i := 0; i < 5; i++ {
		functionID := fmt.Sprintf("function-%d", i)

		// This should be a miss
		_, exists := cache.Get(functionID)
		if exists {
			t.Errorf("Unexpected cache hit for %s", functionID)
		}

		// Add entry
		entry := &CacheEntry{
			FunctionID: functionID,
			Handler:    fmt.Sprintf("handler-%d", i),
			FileHashes: map[string]string{
				fmt.Sprintf("file-%d.py", i): fmt.Sprintf("hash-%d", i),
			},
		}

		if err := cache.Set(functionID, entry); err != nil {
			t.Fatalf("Failed to set cache entry: %v", err)
		}

		// This should be a hit
		_, exists = cache.Get(functionID)
		if !exists {
			t.Errorf("Expected cache hit for %s", functionID)
		}
	}

	// Check final metrics
	finalStats := cache.GetStats()
	if finalStats.Metrics.Hits != 5 {
		t.Errorf("Expected 5 hits, got %d", finalStats.Metrics.Hits)
	}
	if finalStats.Metrics.Misses != 5 {
		t.Errorf("Expected 5 misses, got %d", finalStats.Metrics.Misses)
	}

	// Test hit rate calculation
	hitRate := cache.GetHitRate()
	expectedHitRate := 50.0 // 5 hits out of 10 total operations
	if hitRate != expectedHitRate {
		t.Errorf("Expected hit rate %.1f%%, got %.1f%%", expectedHitRate, hitRate)
	}

	// Test performance report
	report := cache.GetPerformanceReport()
	if report["hits"] != int64(5) {
		t.Errorf("Expected 5 hits in report, got %v", report["hits"])
	}
	if report["hitRate"] != 50.0 {
		t.Errorf("Expected 50%% hit rate in report, got %v", report["hitRate"])
	}

	// Test metrics reset
	cache.ResetMetrics()
	resetStats := cache.GetStats()
	if resetStats.Metrics.Hits != 0 || resetStats.Metrics.Misses != 0 {
		t.Error("Expected metrics to be reset to zero")
	}
}

// TestBuildCache_FileHashingPerformance tests file hashing performance tracking
func TestBuildCache_FileHashingPerformance(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	// Create test files of different sizes
	smallFile := filepath.Join(tempDir, "small.py")
	largeFile := filepath.Join(tempDir, "large.py")

	if err := os.WriteFile(smallFile, []byte("print('small')"), 0644); err != nil {
		t.Fatalf("Failed to create small file: %v", err)
	}

	// Create a larger file
	largeContent := make([]byte, 10000)
	for i := range largeContent {
		largeContent[i] = byte('a' + (i % 26))
	}
	if err := os.WriteFile(largeFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          cacheDir,
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Hash files multiple times to test performance tracking
	files := []string{smallFile, largeFile}
	for i := 0; i < 3; i++ {
		for _, file := range files {
			_, err := cache.calculateFileHash(file)
			if err != nil {
				t.Fatalf("Failed to hash file %s: %v", file, err)
			}
		}
	}

	// Check that hash operations were tracked
	stats := cache.GetStats()
	expectedOps := int64(6) // 2 files × 3 iterations
	if stats.Metrics.FileHashOperations != expectedOps {
		t.Errorf("Expected %d hash operations, got %d", expectedOps, stats.Metrics.FileHashOperations)
	}

	// Check that average hash time is reasonable
	if stats.Metrics.AverageHashTime <= 0 {
		t.Error("Expected positive average hash time")
	}

	// Check that total hash time is greater than average (since we did multiple operations)
	if stats.Metrics.TotalHashTime < stats.Metrics.AverageHashTime {
		t.Error("Expected total hash time to be >= average hash time")
	}
}

// TestBuildCache_ErrorTracking tests error tracking in cache operations
func TestBuildCache_ErrorTracking(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          cacheDir,
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test hash error tracking by trying to hash non-existent file
	nonExistentFile := filepath.Join(tempDir, "does-not-exist.py")
	_, err = cache.calculateFileHash(nonExistentFile)
	if err == nil {
		t.Error("Expected error when hashing non-existent file")
	}

	// Check that hash error was recorded
	stats := cache.GetStats()
	if stats.Metrics.HashErrors != 1 {
		t.Errorf("Expected 1 hash error, got %d", stats.Metrics.HashErrors)
	}

	// Test multiple hash errors
	for i := 0; i < 3; i++ {
		cache.calculateFileHash(filepath.Join(tempDir, fmt.Sprintf("missing-%d.py", i)))
	}

	finalStats := cache.GetStats()
	if finalStats.Metrics.HashErrors != 4 { // 1 + 3
		t.Errorf("Expected 4 hash errors, got %d", finalStats.Metrics.HashErrors)
	}
}

// TestBuildCache_CacheInvalidationScenarios tests various cache invalidation scenarios
func TestBuildCache_CacheInvalidationScenarios(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	// Create test files
	handlerFile := filepath.Join(tempDir, "handler.py")
	depFile1 := filepath.Join(tempDir, "dep1.py")
	depFile2 := filepath.Join(tempDir, "dep2.py")

	files := []string{handlerFile, depFile1, depFile2}
	for i, file := range files {
		content := fmt.Sprintf("# File %d\nprint('content %d')", i, i)
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          cacheDir,
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create and cache entry
	entry := &CacheEntry{
		FunctionID:   "test-function",
		Handler:      "handler.main",
		Dependencies: files,
		FileHashes:   make(map[string]string),
	}

	// Calculate initial hashes
	if err := cache.UpdateFileHashes(entry, files); err != nil {
		t.Fatalf("Failed to update file hashes: %v", err)
	}

	// Store in cache
	if err := cache.Set("test-function", entry); err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Verify cache is initially valid
	valid, err := cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to validate cache: %v", err)
	}
	if !valid {
		t.Error("Expected cache to be initially valid")
	}

	// Test scenario 1: Modify one dependency file
	modifiedContent := "# Modified file\nprint('modified content')"
	if err := os.WriteFile(depFile1, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify dependency file: %v", err)
	}

	valid, err = cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to validate cache: %v", err)
	}
	if valid {
		t.Error("Expected cache to be invalid after dependency modification")
	}

	// Test scenario 2: Delete a dependency file
	if err := os.Remove(depFile2); err != nil {
		t.Fatalf("Failed to remove dependency file: %v", err)
	}

	valid, err = cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to validate cache: %v", err)
	}
	if valid {
		t.Error("Expected cache to be invalid after dependency deletion")
	}

	// Check that invalidations were recorded
	stats := cache.GetStats()
	if stats.Metrics.InvalidationCount < 2 {
		t.Errorf("Expected at least 2 invalidations, got %d", stats.Metrics.InvalidationCount)
	}
}

// TestBuildCache_ContentBasedVsLayoutBased demonstrates the difference between content-based and layout-based caching
func TestBuildCache_ContentBasedVsLayoutBased(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	// Create a project structure
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}

	handlerFile := filepath.Join(srcDir, "handler.py")
	utilFile := filepath.Join(srcDir, "utils.py")

	if err := os.WriteFile(handlerFile, []byte("from utils import helper\ndef main(): return helper()"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	if err := os.WriteFile(utilFile, []byte("def helper(): return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create utils file: %v", err)
	}

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          cacheDir,
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create cache entry based on actual file content, not layout type
	entry := &CacheEntry{
		FunctionID:   "content-based-function",
		Handler:      "src.handler.main",
		Dependencies: []string{handlerFile, utilFile},
		FileHashes:   make(map[string]string),
		ProjectInfo: &ProjectInfo{
			HandlerFile:  handlerFile,
			SourceRoot:   srcDir,
			ModulePath:   "src.handler",
			Dependencies: []string{handlerFile, utilFile},
		},
	}

	// Calculate file hashes
	if err := cache.UpdateFileHashes(entry, []string{handlerFile, utilFile}); err != nil {
		t.Fatalf("Failed to update file hashes: %v", err)
	}

	// Store in cache
	if err := cache.Set("content-based-function", entry); err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Verify cache hit
	cachedEntry, exists := cache.Get("content-based-function")
	if !exists {
		t.Error("Expected cache hit")
	}

	// Verify that the cache entry contains ProjectInfo instead of LayoutInfo
	if cachedEntry.ProjectInfo == nil {
		t.Error("Expected ProjectInfo to be present in cache entry")
	}
	if cachedEntry.ProjectInfo.ModulePath != "src.handler" {
		t.Errorf("Expected module path 'src.handler', got %s", cachedEntry.ProjectInfo.ModulePath)
	}

	// Test that cache remains valid when file content doesn't change
	// (even if we were to change layout detection logic)
	valid, err := cache.IsValid(cachedEntry)
	if err != nil {
		t.Fatalf("Failed to validate cache: %v", err)
	}
	if !valid {
		t.Error("Expected cache to remain valid when content hasn't changed")
	}

	// Test that cache becomes invalid when actual content changes
	if err := os.WriteFile(utilFile, []byte("def helper(): return 'modified'"), 0644); err != nil {
		t.Fatalf("Failed to modify utils file: %v", err)
	}

	valid, err = cache.IsValid(cachedEntry)
	if err != nil {
		t.Fatalf("Failed to validate cache: %v", err)
	}
	if valid {
		t.Error("Expected cache to be invalid after content modification")
	}
}
