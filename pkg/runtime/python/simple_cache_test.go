package python

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBasicCacheOperations tests the basic cache operations directly
func TestBasicCacheOperations(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	// Create a build cache
	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: cacheDir,
		MaxSize:  10,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	t.Log("=== Testing Basic Cache Operations ===")

	// Test 1: Cache should be empty initially
	_, exists := cache.Get("test-function")
	if exists {
		t.Error("Expected cache to be empty initially")
	}
	t.Log("✅ Empty cache check: PASSED")

	// Test 2: Set a cache entry
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
	t.Log("✅ Cache set: PASSED")

	// Test 3: Get the cache entry
	retrievedEntry, exists := cache.Get("test-function")
	if !exists {
		t.Error("Expected cache entry to exist after setting")
	}
	if retrievedEntry.FunctionID != "test-function" {
		t.Errorf("Expected FunctionID 'test-function', got '%s'", retrievedEntry.FunctionID)
	}
	t.Log("✅ Cache get: PASSED")

	// Test 4: Check cache stats
	stats := cache.GetStats()
	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 cache entry, got %d", stats.TotalEntries)
	}
	t.Log("✅ Cache stats: PASSED")

	// Test 5: Cache validity check
	// Create a temporary file for validation
	testFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Update entry with real file hash using the same key as in the entry
	entry.FileHashes = make(map[string]string)
	err = cache.UpdateFileHashes(entry, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to update file hashes: %v", err)
	}

	valid, err := cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to check cache validity: %v", err)
	}
	if !valid {
		t.Error("Expected cache entry to be valid")
	}
	t.Log("✅ Cache validity: PASSED")

	// Test 6: Cache invalidation by file change
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	valid, err = cache.IsValid(entry)
	if err != nil {
		t.Fatalf("Failed to check cache validity after file change: %v", err)
	}
	if valid {
		t.Error("Expected cache entry to be invalid after file change")
	}
	t.Log("✅ Cache invalidation: PASSED")

	t.Log("=== All Basic Cache Operations: WORKING ===")
}

// TestChangeDetector tests the change detection system
func TestChangeDetector(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	// Create change detector
	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: cacheDir,
		MaxSize:  10,
	})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	projectResolver := NewProjectResolver(tempDir)

	changeDetector, err := NewChangeDetector(ChangeDetectorConfig{
		ProjectResolver: projectResolver,
		BuildCache:      cache,
	})
	if err != nil {
		t.Fatalf("Failed to create change detector: %v", err)
	}

	t.Log("=== Testing Change Detection ===")

	// Create a test file
	testFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test 1: First detection should indicate changes (no cache)
	result, err := changeDetector.DetectChanges("test-function", "handler.lambda_handler")
	if err != nil {
		t.Fatalf("Failed to detect changes: %v", err)
	}
	if !result.HasChanges {
		t.Error("Expected changes on first detection (no cache)")
	}
	t.Logf("✅ First detection: %s", result.Reason)

	// Test 2: Create a cache entry with build output
	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create project info for the cache entry
	projectInfo := &ProjectInfo{
		ProjectRoot: tempDir,
		SourceRoot:  tempDir,
		ModulePath:  "handler",
		PythonPath:  []string{tempDir},
	}

	entry := &CacheEntry{
		FunctionID:   "test-function",
		BuildTime:    time.Now(),
		FileHashes:   map[string]string{},
		Dependencies: []string{testFile},
		BuildOutput: &CachedBuildOutput{
			Handler: "handler.lambda_handler",
		},
		ProjectInfo: projectInfo,
	}

	err = cache.UpdateFileHashes(entry, []string{testFile})
	if err != nil {
		t.Fatalf("Failed to update file hashes: %v", err)
	}

	err = cache.Set("test-function", entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Test 3: Second detection should not indicate changes (cache hit)
	result, err = changeDetector.DetectChanges("test-function", "handler.lambda_handler")
	if err != nil {
		t.Fatalf("Failed to detect changes: %v", err)
	}
	if result.HasChanges {
		t.Errorf("Expected no changes on second detection (cache hit), but got: %s", result.Reason)
	}
	t.Log("✅ Cache hit detection: PASSED")

	// Test 4: Modify file and detect changes
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	result, err = changeDetector.DetectChanges("test-function", "handler.lambda_handler")
	if err != nil {
		t.Fatalf("Failed to detect changes after modification: %v", err)
	}
	if !result.HasChanges {
		t.Error("Expected changes after file modification")
	}
	t.Logf("✅ File change detection: %s", result.Reason)

	t.Log("=== Change Detection: WORKING ===")
}
