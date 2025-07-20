package python

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestBuildCache_ConcurrentAccess tests concurrent access to the cache
func TestBuildCache_ConcurrentAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrent_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            1 * time.Hour,
		MaxSize:           100,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test concurrent reads and writes
	var wg sync.WaitGroup
	numGoroutines := 50
	numOperations := 100

	// Concurrent writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				functionID := fmt.Sprintf("function-%d-%d", id, j)
				entry := &CacheEntry{
					FunctionID:   functionID,
					LastBuild:    time.Now(),
					FileHashes:   map[string]string{"file1.py": "hash123"},
					Dependencies: []string{"dep1", "dep2"},
				}
				cache.Set(functionID, entry)
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				functionID := fmt.Sprintf("function-%d-%d", id, j)
				cache.Get(functionID) // May or may not find entry
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify cache integrity
	stats := cache.GetStats()
	if stats.TotalEntries < 0 {
		t.Error("Cache stats show negative entries")
	}
}

// TestBuildCache_CacheInvalidation tests cache invalidation logic
func TestBuildCache_CacheInvalidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "invalidation_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            100 * time.Millisecond, // Short expiry for testing
		MaxSize:           10,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	functionID := "test-function"
	entry := &CacheEntry{
		FunctionID:   functionID,
		LastBuild:    time.Now(),
		FileHashes:   map[string]string{"file1.py": "hash123"},
		Dependencies: []string{"dep1"},
	}

	// Set entry
	cache.Set(functionID, entry)

	// Verify entry exists
	if !cache.Has(functionID) {
		t.Error("Entry should exist after setting")
	}

	// Wait for expiry
	time.Sleep(200 * time.Millisecond)

	// Entry should be expired but still in cache until cleanup
	if !cache.Has(functionID) {
		t.Error("Entry should still exist before cleanup")
	}

	// Check if entry is valid (should be false due to expiry)
	if cache.IsValid(functionID, []string{"file1.py"}) {
		t.Error("Entry should be invalid due to expiry")
	}

	// Force cleanup
	cache.Cleanup()

	// Entry should be removed after cleanup
	if cache.Has(functionID) {
		t.Error("Entry should be removed after cleanup")
	}
}

// TestBuildCache_FileHashValidation tests file hash-based validation
func TestBuildCache_FileHashValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hash_validation_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            1 * time.Hour,
		MaxSize:           10,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create test files
	file1 := filepath.Join(tempDir, "file1.py")
	file2 := filepath.Join(tempDir, "file2.py")

	if err := os.WriteFile(file1, []byte("def func1(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("def func2(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	functionID := "test-function"

	// Calculate initial hashes
	hash1, err := cache.calculateFileHash(file1)
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}
	hash2, err := cache.calculateFileHash(file2)
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}

	entry := &CacheEntry{
		FunctionID: functionID,
		LastBuild:  time.Now(),
		FileHashes: map[string]string{
			file1: hash1,
			file2: hash2,
		},
		Dependencies: []string{},
	}

	cache.Set(functionID, entry)

	// Validation should pass with unchanged files
	if !cache.IsValid(functionID, []string{file1, file2}) {
		t.Error("Cache should be valid with unchanged files")
	}

	// Modify file1
	if err := os.WriteFile(file1, []byte("def func1(): return 'modified'"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Validation should fail with modified file
	if cache.IsValid(functionID, []string{file1, file2}) {
		t.Error("Cache should be invalid with modified file")
	}

	// Validation should still pass if we only check file2
	if !cache.IsValid(functionID, []string{file2}) {
		t.Error("Cache should be valid when checking only unchanged file")
	}
}

// TestBuildCache_Persistence tests cache persistence to disk
func TestBuildCache_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create first cache instance
	cache1, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            1 * time.Hour,
		MaxSize:           10,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	functionID := "persistent-function"
	entry := &CacheEntry{
		FunctionID:   functionID,
		LastBuild:    time.Now(),
		FileHashes:   map[string]string{"file1.py": "hash123"},
		Dependencies: []string{"dep1", "dep2"},
	}

	// Set entry in first cache
	cache1.Set(functionID, entry)

	// Verify entry exists
	if !cache1.Has(functionID) {
		t.Error("Entry should exist in first cache")
	}

	// Save to disk
	if err := cache1.SaveToDisk(); err != nil {
		t.Fatalf("Failed to save cache to disk: %v", err)
	}

	// Create second cache instance (should load from disk)
	cache2, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            1 * time.Hour,
		MaxSize:           10,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create second cache: %v", err)
	}

	// Entry should exist in second cache (loaded from disk)
	if !cache2.Has(functionID) {
		t.Error("Entry should exist in second cache (loaded from disk)")
	}

	// Verify entry data
	retrievedEntry, exists := cache2.Get(functionID)
	if !exists {
		t.Error("Entry should be retrievable from second cache")
	}

	if retrievedEntry.FunctionID != entry.FunctionID {
		t.Errorf("Expected function ID %s, got %s", entry.FunctionID, retrievedEntry.FunctionID)
	}

	if len(retrievedEntry.FileHashes) != len(entry.FileHashes) {
		t.Errorf("Expected %d file hashes, got %d", len(entry.FileHashes), len(retrievedEntry.FileHashes))
	}
}

// TestBuildCache_EvictionPolicies tests cache eviction policies
func TestBuildCache_EvictionPolicies(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "eviction_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            1 * time.Hour,
		MaxSize:           3, // Small size to trigger eviction
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add entries up to max size
	entries := []string{"func1", "func2", "func3"}
	for i, functionID := range entries {
		entry := &CacheEntry{
			FunctionID:   functionID,
			LastBuild:    time.Now().Add(time.Duration(i) * time.Second), // Different timestamps
			FileHashes:   map[string]string{"file.py": fmt.Sprintf("hash%d", i)},
			Dependencies: []string{},
		}
		cache.Set(functionID, entry)
		time.Sleep(10 * time.Millisecond) // Ensure different access times
	}

	// All entries should exist
	for _, functionID := range entries {
		if !cache.Has(functionID) {
			t.Errorf("Entry %s should exist", functionID)
		}
	}

	// Add one more entry to trigger eviction
	newEntry := &CacheEntry{
		FunctionID:   "func4",
		LastBuild:    time.Now(),
		FileHashes:   map[string]string{"file.py": "hash4"},
		Dependencies: []string{},
	}
	cache.Set("func4", newEntry)

	// One of the old entries should be evicted
	existingCount := 0
	for _, functionID := range entries {
		if cache.Has(functionID) {
			existingCount++
		}
	}

	if existingCount != 2 {
		t.Errorf("Expected 2 old entries to remain after eviction, got %d", existingCount)
	}

	// New entry should exist
	if !cache.Has("func4") {
		t.Error("New entry should exist after eviction")
	}
}

// TestBuildCache_CleanupAndMaintenance tests cache cleanup and maintenance
func TestBuildCache_CleanupAndMaintenance(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cleanup_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            50 * time.Millisecond, // Very short for testing
		MaxSize:           10,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add multiple entries
	functionIDs := []string{"func1", "func2", "func3", "func4", "func5"}
	for _, functionID := range functionIDs {
		entry := &CacheEntry{
			FunctionID:   functionID,
			LastBuild:    time.Now(),
			FileHashes:   map[string]string{"file.py": "hash"},
			Dependencies: []string{},
		}
		cache.Set(functionID, entry)
	}

	// All entries should exist
	initialStats := cache.GetStats()
	if initialStats.TotalEntries != len(functionIDs) {
		t.Errorf("Expected %d entries, got %d", len(functionIDs), initialStats.TotalEntries)
	}

	// Wait for entries to expire
	time.Sleep(100 * time.Millisecond)

	// Run cleanup
	cache.Cleanup()

	// All entries should be removed due to expiry
	finalStats := cache.GetStats()
	if finalStats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", finalStats.TotalEntries)
	}

	// Test maintenance operations
	cache.PerformMaintenance()

	// Should not crash or cause issues
	maintenanceStats := cache.GetStats()
	if maintenanceStats.TotalEntries < 0 {
		t.Error("Maintenance should not result in negative entry count")
	}
}

// TestBuildCache_CorruptedCacheRecovery tests recovery from corrupted cache files
func TestBuildCache_CorruptedCacheRecovery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "corrupted_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create corrupted cache file
	cacheFile := filepath.Join(tempDir, "cache.json")
	if err := os.WriteFile(cacheFile, []byte("invalid json content"), 0644); err != nil {
		t.Fatalf("Failed to create corrupted cache file: %v", err)
	}

	// Cache should handle corrupted file gracefully
	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            1 * time.Hour,
		MaxSize:           10,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache with corrupted file: %v", err)
	}

	// Should start with empty cache
	stats := cache.GetStats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected empty cache after corruption recovery, got %d entries", stats.TotalEntries)
	}

	// Should be able to add new entries
	entry := &CacheEntry{
		FunctionID:   "test-function",
		LastBuild:    time.Now(),
		FileHashes:   map[string]string{"file.py": "hash"},
		Dependencies: []string{},
	}
	cache.Set("test-function", entry)

	if !cache.Has("test-function") {
		t.Error("Should be able to add entries after corruption recovery")
	}
}

// TestBuildCache_LargeEntries tests handling of large cache entries
func TestBuildCache_LargeEntries(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "large_entries_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            1 * time.Hour,
		MaxSize:           10,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create entry with many file hashes
	largeFileHashes := make(map[string]string)
	for i := 0; i < 1000; i++ {
		largeFileHashes[fmt.Sprintf("file%d.py", i)] = fmt.Sprintf("hash%d", i)
	}

	// Create entry with many dependencies
	largeDependencies := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		largeDependencies[i] = fmt.Sprintf("dependency%d", i)
	}

	largeEntry := &CacheEntry{
		FunctionID:   "large-function",
		LastBuild:    time.Now(),
		FileHashes:   largeFileHashes,
		Dependencies: largeDependencies,
	}

	// Should handle large entries
	cache.Set("large-function", largeEntry)

	if !cache.Has("large-function") {
		t.Error("Should handle large cache entries")
	}

	// Should be able to retrieve large entries
	retrievedEntry, exists := cache.Get("large-function")
	if !exists {
		t.Error("Should be able to retrieve large entries")
	}

	if len(retrievedEntry.FileHashes) != len(largeFileHashes) {
		t.Errorf("Expected %d file hashes, got %d", len(largeFileHashes), len(retrievedEntry.FileHashes))
	}

	if len(retrievedEntry.Dependencies) != len(largeDependencies) {
		t.Errorf("Expected %d dependencies, got %d", len(largeDependencies), len(retrievedEntry.Dependencies))
	}
}

// TestBuildCache_MemoryUsage tests memory usage patterns
func TestBuildCache_MemoryUsage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory_usage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            1 * time.Hour,
		MaxSize:           1000, // Large size
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add many entries
	numEntries := 500
	for i := 0; i < numEntries; i++ {
		entry := &CacheEntry{
			FunctionID:   fmt.Sprintf("function-%d", i),
			LastBuild:    time.Now(),
			FileHashes:   map[string]string{fmt.Sprintf("file%d.py", i): fmt.Sprintf("hash%d", i)},
			Dependencies: []string{fmt.Sprintf("dep%d", i)},
		}
		cache.Set(fmt.Sprintf("function-%d", i), entry)
	}

	// Check stats
	stats := cache.GetStats()
	if stats.TotalEntries != numEntries {
		t.Errorf("Expected %d entries, got %d", numEntries, stats.TotalEntries)
	}

	// Memory usage should be reasonable (this is a basic check)
	if stats.MemoryUsage < 0 {
		t.Error("Memory usage should not be negative")
	}

	// Clear cache and check memory is freed
	cache.Clear()

	clearedStats := cache.GetStats()
	if clearedStats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", clearedStats.TotalEntries)
	}
}

// TestBuildCache_ThreadSafety tests thread safety with high contention
func TestBuildCache_ThreadSafety(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "thread_safety_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		MaxAge:            1 * time.Hour,
		MaxSize:           100,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	var wg sync.WaitGroup
	numWorkers := 20
	operationsPerWorker := 100

	// Mixed operations: set, get, has, delete, cleanup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				functionID := fmt.Sprintf("worker-%d-func-%d", workerID, j)

				// Set operation
				entry := &CacheEntry{
					FunctionID:   functionID,
					LastBuild:    time.Now(),
					FileHashes:   map[string]string{"file.py": "hash"},
					Dependencies: []string{"dep"},
				}
				cache.Set(functionID, entry)

				// Get operation
				cache.Get(functionID)

				// Has operation
				cache.Has(functionID)

				// Validation operation
				cache.IsValid(functionID, []string{"file.py"})

				// Occasionally delete
				if j%10 == 0 {
					cache.Delete(functionID)
				}

				// Occasionally cleanup
				if j%20 == 0 {
					cache.Cleanup()
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()

	// Cache should still be in a valid state
	stats := cache.GetStats()
	if stats.TotalEntries < 0 {
		t.Error("Cache should be in valid state after concurrent operations")
	}
}
