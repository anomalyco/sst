package python

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewBuildCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := BuildCacheConfig{
		CacheDir:          tempDir,
		MaxSize:           100,
		EnablePersistence: true,
	}

	cache, err := NewBuildCache(config)
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	if cache.cacheDir != tempDir {
		t.Errorf("Expected cache dir %s, got %s", tempDir, cache.cacheDir)
	}

	// maxAge field removed in content-based caching

	if cache.maxSize != 100 {
		t.Errorf("Expected max size 100, got %d", cache.maxSize)
	}

	// Check that cache directory was created
	if _, err := os.Stat(tempDir); err != nil {
		t.Errorf("Cache directory was not created: %v", err)
	}
}

func TestNewBuildCache_Defaults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := BuildCacheConfig{
		CacheDir: tempDir,
		// MaxSize not set, should use defaults
	}

	cache, err := NewBuildCache(config)
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// maxAge field removed in content-based caching

	if cache.maxSize != 1000 {
		t.Errorf("Expected default max size 1000, got %d", cache.maxSize)
	}
}

func TestNewBuildCache_InvalidConfig(t *testing.T) {
	config := BuildCacheConfig{
		// CacheDir not set
		MaxSize: 100,
	}

	_, err := NewBuildCache(config)
	if err == nil {
		t.Error("Expected error for missing cache directory")
	}
}

func TestBuildCache_SetAndGet(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
		MaxSize:  10,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create a test entry
	entry := &CacheEntry{
		Handler:      "test.handler",
		Dependencies: []string{"test.py", "requirements.txt"},
		FileHashes: map[string]string{
			"test.py":          "hash1",
			"requirements.txt": "hash2",
		},
		Properties: map[string]interface{}{
			"architecture": "x86_64",
		},
	}

	functionID := "test-function"

	// Test Set
	err = cache.Set(functionID, entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Test Get
	retrieved, exists := cache.Get(functionID)
	if !exists {
		t.Fatal("Cache entry should exist")
	}

	if retrieved.Handler != entry.Handler {
		t.Errorf("Expected handler %s, got %s", entry.Handler, retrieved.Handler)
	}

	if retrieved.FunctionID != functionID {
		t.Errorf("Expected function ID %s, got %s", functionID, retrieved.FunctionID)
	}

	if len(retrieved.FileHashes) != 2 {
		t.Errorf("Expected 2 file hashes, got %d", len(retrieved.FileHashes))
	}

	// Check that timestamps were set
	if retrieved.BuildTime.IsZero() {
		t.Error("Build time should be set")
	}

	if retrieved.LastAccessed.IsZero() {
		t.Error("Last accessed time should be set")
	}
}

func TestBuildCache_GetNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	_, exists := cache.Get("nonexistent")
	if exists {
		t.Error("Non-existent entry should not exist")
	}
}

func TestBuildCache_ExpiredEntry(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	entry := &CacheEntry{
		Handler: "test.handler",
	}

	functionID := "test-function"

	// Set entry
	err = cache.Set(functionID, entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// With content-based caching, entries don't expire based on time
	// They remain valid until content changes
	_, exists := cache.Get(functionID)
	if !exists {
		t.Error("Entry should still exist with content-based caching")
	}
}

func TestBuildCache_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	entry := &CacheEntry{
		Handler: "test.handler",
	}

	functionID := "test-function"

	// Set entry
	err = cache.Set(functionID, entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Verify it exists
	_, exists := cache.Get(functionID)
	if !exists {
		t.Fatal("Entry should exist before deletion")
	}

	// Delete entry
	err = cache.Delete(functionID)
	if err != nil {
		t.Fatalf("Failed to delete cache entry: %v", err)
	}

	// Verify it's gone
	_, exists = cache.Get(functionID)
	if exists {
		t.Error("Entry should not exist after deletion")
	}

	// Verify cache file is gone
	cacheFile := filepath.Join(tempDir, functionID+".json")
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Error("Cache file should be deleted")
	}
}

func TestBuildCache_Clear(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add multiple entries
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Handler: fmt.Sprintf("test%d.handler", i),
		}
		functionID := fmt.Sprintf("test-function-%d", i)

		err = cache.Set(functionID, entry)
		if err != nil {
			t.Fatalf("Failed to set cache entry %d: %v", i, err)
		}
	}

	// Verify entries exist
	stats := cache.GetStats()
	if stats.TotalEntries != 5 {
		t.Errorf("Expected 5 entries, got %d", stats.TotalEntries)
	}

	// Clear cache
	err = cache.Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Verify cache is empty
	stats = cache.GetStats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.TotalEntries)
	}

	if stats.DiskFiles != 0 {
		t.Errorf("Expected 0 disk files after clear, got %d", stats.DiskFiles)
	}
}

func TestBuildCache_CalculateFileHash(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test.py")
	content := "def hello(): return 'world'"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate hash
	hash1, err := cache.calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate file hash: %v", err)
	}

	if hash1 == "" {
		t.Error("Hash should not be empty")
	}

	// Calculate hash again - should be the same
	hash2, err := cache.calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate file hash: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash should be consistent: %s != %s", hash1, hash2)
	}

	// Modify file and check hash changes
	newContent := "def hello(): return 'universe'"
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	hash3, err := cache.calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate file hash: %v", err)
	}

	if hash1 == hash3 {
		t.Error("Hash should change when file content changes")
	}
}

func TestBuildCache_UpdateFileHashes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create test files
	testFiles := []string{"test1.py", "test2.py"}
	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		content := fmt.Sprintf("# Content of %s", filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create cache entry
	entry := &CacheEntry{
		Handler: "test.handler",
	}

	// Update file hashes
	filePaths := make([]string, len(testFiles))
	for i, filename := range testFiles {
		filePaths[i] = filepath.Join(tempDir, filename)
	}

	err = cache.UpdateFileHashes(entry, filePaths)
	if err != nil {
		t.Fatalf("Failed to update file hashes: %v", err)
	}

	// Verify hashes were set
	if len(entry.FileHashes) != len(testFiles) {
		t.Errorf("Expected %d file hashes, got %d", len(testFiles), len(entry.FileHashes))
	}

	for _, filePath := range filePaths {
		if _, exists := entry.FileHashes[filePath]; !exists {
			t.Errorf("Hash for file %s should exist", filePath)
		}
	}
}

func TestBuildCache_IsValid(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test.py")
	content := "def hello(): return 'world'"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate initial hash
	hash, err := cache.calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate file hash: %v", err)
	}

	// Create cache entry with correct hash
	entry := &CacheEntry{
		Handler: "test.handler",
		FileHashes: map[string]string{
			testFile: hash,
		},
	}

	// Should be valid
	valid, err := cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to check validity: %v", err)
	}
	if !valid {
		t.Error("Entry should be valid with correct hash")
	}

	// Modify file
	newContent := "def hello(): return 'universe'"
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Should now be invalid
	valid, err = cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to check validity: %v", err)
	}
	if valid {
		t.Error("Entry should be invalid after file modification")
	}
}

func TestBuildCache_IsValid_MissingFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Create cache entry with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.py")
	entry := &CacheEntry{
		Handler: "test.handler",
		FileHashes: map[string]string{
			nonExistentFile: "somehash",
		},
	}

	// Should be invalid
	valid, err := cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to check validity: %v", err)
	}
	if valid {
		t.Error("Entry should be invalid for non-existent file")
	}
}

func TestBuildCache_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create cache with persistence enabled
	cache1, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add entry
	entry := &CacheEntry{
		Handler: "test.handler",
		FileHashes: map[string]string{
			"test.py": "hash123",
		},
	}

	functionID := "test-function"
	err = cache1.Set(functionID, entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Create new cache instance (simulating restart)
	cache2, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          tempDir,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create second cache: %v", err)
	}

	// Entry should be loaded from disk
	retrieved, exists := cache2.Get(functionID)
	if !exists {
		t.Fatal("Entry should be loaded from disk")
	}

	if retrieved.Handler != entry.Handler {
		t.Errorf("Expected handler %s, got %s", entry.Handler, retrieved.Handler)
	}

	if len(retrieved.FileHashes) != 1 {
		t.Errorf("Expected 1 file hash, got %d", len(retrieved.FileHashes))
	}
}

func TestBuildCache_MaxSizeCleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
		MaxSize:  3, // Small max size to trigger cleanup
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add more entries than max size
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Handler: fmt.Sprintf("test%d.handler", i),
		}
		functionID := fmt.Sprintf("test-function-%d", i)

		err = cache.Set(functionID, entry)
		if err != nil {
			t.Fatalf("Failed to set cache entry %d: %v", i, err)
		}

		// Add small delay to ensure different access times
		time.Sleep(1 * time.Millisecond)
	}

	// Should have cleaned up to max size
	stats := cache.GetStats()
	if stats.TotalEntries > 3 {
		t.Errorf("Expected at most 3 entries after cleanup, got %d", stats.TotalEntries)
	}
}

func TestBuildCache_GetStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
		MaxSize:  100,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add some entries
	for i := 0; i < 3; i++ {
		entry := &CacheEntry{
			Handler: fmt.Sprintf("test%d.handler", i),
		}
		functionID := fmt.Sprintf("test-function-%d", i)

		err = cache.Set(functionID, entry)
		if err != nil {
			t.Fatalf("Failed to set cache entry %d: %v", i, err)
		}
	}

	stats := cache.GetStats()

	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 total entries, got %d", stats.TotalEntries)
	}

	if stats.MaxSize != 100 {
		t.Errorf("Expected max size 100, got %d", stats.MaxSize)
	}

	// MaxAge field removed in content-based caching

	if stats.CacheDir != tempDir {
		t.Errorf("Expected cache dir %s, got %s", tempDir, stats.CacheDir)
	}

	if stats.DiskFiles != 3 {
		t.Errorf("Expected 3 disk files, got %d", stats.DiskFiles)
	}

	if stats.DiskSize <= 0 {
		t.Error("Expected positive disk size")
	}
}

func TestBuildCache_Cleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add entries
	for i := 0; i < 3; i++ {
		entry := &CacheEntry{
			Handler: fmt.Sprintf("test%d.handler", i),
		}
		functionID := fmt.Sprintf("test-function-%d", i)

		err = cache.Set(functionID, entry)
		if err != nil {
			t.Fatalf("Failed to set cache entry %d: %v", i, err)
		}
	}

	// Verify entries exist
	stats := cache.GetStats()
	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 entries before cleanup, got %d", stats.TotalEntries)
	}

	// Run cleanup - with content-based caching, cleanup only removes orphaned entries
	err = cache.Cleanup()
	if err != nil {
		t.Fatalf("Failed to cleanup cache: %v", err)
	}

	// Verify entries still exist since they're not orphaned (no files to track)
	stats = cache.GetStats()
	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 entries after cleanup (content-based caching), got %d", stats.TotalEntries)
	}
}

func TestCacheEntry_JSON(t *testing.T) {
	entry := &CacheEntry{
		FunctionID:   "test-function",
		Handler:      "test.handler",
		BuildTime:    time.Now(),
		LastAccessed: time.Now(),
		FileHashes: map[string]string{
			"test.py":          "hash1",
			"requirements.txt": "hash2",
		},
		Dependencies: []string{"test.py", "requirements.txt"},
		Properties: map[string]interface{}{
			"architecture": "x86_64",
			"container":    false,
		},
		BuildOutput: &CachedBuildOutput{
			Handler:       "test.lambda_handler",
			OutputDir:     "/tmp/build",
			Errors:        []string{},
			Sourcemaps:    []string{},
			ArtifactPaths: []string{"/tmp/build/test.py"},
			BuildDuration: 5 * time.Second,
		},
	}

	// Test marshaling
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal cache entry: %v", err)
	}

	// Test unmarshaling
	var unmarshaled CacheEntry
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal cache entry: %v", err)
	}

	// Verify fields
	if unmarshaled.FunctionID != entry.FunctionID {
		t.Errorf("Expected function ID %s, got %s", entry.FunctionID, unmarshaled.FunctionID)
	}

	if unmarshaled.Handler != entry.Handler {
		t.Errorf("Expected handler %s, got %s", entry.Handler, unmarshaled.Handler)
	}

	if len(unmarshaled.FileHashes) != len(entry.FileHashes) {
		t.Errorf("Expected %d file hashes, got %d", len(entry.FileHashes), len(unmarshaled.FileHashes))
	}

	if len(unmarshaled.Dependencies) != len(entry.Dependencies) {
		t.Errorf("Expected %d dependencies, got %d", len(entry.Dependencies), len(unmarshaled.Dependencies))
	}

	if unmarshaled.BuildOutput == nil {
		t.Error("Build output should not be nil")
	} else {
		if unmarshaled.BuildOutput.Handler != entry.BuildOutput.Handler {
			t.Errorf("Expected build output handler %s, got %s",
				entry.BuildOutput.Handler, unmarshaled.BuildOutput.Handler)
		}
	}
}
