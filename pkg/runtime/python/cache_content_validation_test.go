package python

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBuildCacheContentValidation tests the core content-based cache validation
func TestBuildCacheContentValidation(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	// Create build cache
	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          cacheDir,
		MaxSize:           100,
		EnablePersistence: true,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test.py")
	initialContent := "print('hello world')"
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create cache entry
	entry := &CacheEntry{
		FunctionID:   "test-function",
		Handler:      "test.handler",
		BuildTime:    time.Now(),
		FileHashes:   make(map[string]string),
		Dependencies: []string{testFile},
	}

	// Calculate initial hash
	if err := cache.UpdateFileHashes(entry, []string{testFile}); err != nil {
		t.Fatalf("Failed to update file hashes: %v", err)
	}

	// Store in cache
	if err := cache.Set("test-function", entry); err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Verify cache is valid
	isValid, err := cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to check cache validity: %v", err)
	}
	if !isValid {
		t.Error("Cache should be valid initially")
	}

	// Modify file content
	modifiedContent := "print('hello modified world')"
	if err := os.WriteFile(testFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Cache should now be invalid
	isValid, err = cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to check cache validity after modification: %v", err)
	}
	if isValid {
		t.Error("Cache should be invalid after file content change")
	}
}

// TestFileHashCalculation tests that file hash calculation works correctly
func TestFileHashCalculation(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	// Create build cache
	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          cacheDir,
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test.py")
	content := "print('test content')"
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
		t.Fatalf("Failed to calculate file hash second time: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Hash should be consistent for same content")
	}

	// Modify file and calculate hash again
	modifiedContent := "print('modified test content')"
	if err := os.WriteFile(testFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	hash3, err := cache.calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate file hash after modification: %v", err)
	}

	if hash1 == hash3 {
		t.Error("Hash should be different after content change")
	}
}
