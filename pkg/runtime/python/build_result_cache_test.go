package python

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewBuildResultCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_result_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}
	
	artifactDir := filepath.Join(tempDir, "artifacts")
	
	config := BuildResultCacheConfig{
		BuildCache:  buildCache,
		ArtifactDir: artifactDir,
	}
	
	resultCache, err := NewBuildResultCache(config)
	if err != nil {
		t.Fatalf("Failed to create build result cache: %v", err)
	}
	
	if resultCache.buildCache != buildCache {
		t.Error("Build cache not set correctly")
	}
	
	if resultCache.artifactDir != artifactDir {
		t.Errorf("Expected artifact dir %s, got %s", artifactDir, resultCache.artifactDir)
	}
	
	// Check that artifact directory was created
	if _, err := os.Stat(artifactDir); err != nil {
		t.Errorf("Artifact directory was not created: %v", err)
	}
}

func TestNewBuildResultCache_InvalidConfig(t *testing.T) {
	testCases := []struct {
		name   string
		config BuildResultCacheConfig
	}{
		{
			name: "missing build cache",
			config: BuildResultCacheConfig{
				ArtifactDir: "/tmp/artifacts",
			},
		},
		{
			name: "missing artifact dir",
			config: BuildResultCacheConfig{
				BuildCache: &BuildCache{},
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewBuildResultCache(tc.config)
			if err == nil {
				t.Errorf("Expected error for %s", tc.name)
			}
		})
	}
}

func TestBuildResultCache_CacheBuildResult(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_result_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	resultCache := createTestBuildResultCache(t, tempDir)
	
	// Create test artifacts
	buildDir := filepath.Join(tempDir, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("Failed to create build dir: %v", err)
	}
	
	artifact1 := filepath.Join(buildDir, "handler.py")
	artifact2 := filepath.Join(buildDir, "requirements.txt")
	
	if err := os.WriteFile(artifact1, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create artifact 1: %v", err)
	}
	
	if err := os.WriteFile(artifact2, []byte("requests==2.28.0"), 0644); err != nil {
		t.Fatalf("Failed to create artifact 2: %v", err)
	}
	
	buildOutput := &CachedBuildOutput{
		Handler:   "handler.handler",
		OutputDir: buildDir,
		Errors:    []string{},
	}
	
	artifactPaths := []string{artifact1, artifact2}
	functionID := "test-function"
	
	// Cache build result
	err = resultCache.CacheBuildResult(functionID, buildOutput, artifactPaths)
	if err != nil {
		t.Fatalf("Failed to cache build result: %v", err)
	}
	
	// Verify artifacts were cached
	functionArtifactDir := filepath.Join(resultCache.artifactDir, functionID)
	if _, err := os.Stat(functionArtifactDir); err != nil {
		t.Errorf("Function artifact directory was not created: %v", err)
	}
	
	cachedArtifact1 := filepath.Join(functionArtifactDir, "handler.py")
	cachedArtifact2 := filepath.Join(functionArtifactDir, "requirements.txt")
	
	if _, err := os.Stat(cachedArtifact1); err != nil {
		t.Errorf("Cached artifact 1 not found: %v", err)
	}
	
	if _, err := os.Stat(cachedArtifact2); err != nil {
		t.Errorf("Cached artifact 2 not found: %v", err)
	}
	
	// Verify content
	content1, err := os.ReadFile(cachedArtifact1)
	if err != nil {
		t.Fatalf("Failed to read cached artifact 1: %v", err)
	}
	
	if string(content1) != "def handler(): pass" {
		t.Errorf("Cached artifact 1 content mismatch")
	}
}

func TestBuildResultCache_RestoreBuildResult(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_result_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	resultCache := createTestBuildResultCache(t, tempDir)
	
	// Create and cache a build result first
	buildDir := filepath.Join(tempDir, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("Failed to create build dir: %v", err)
	}
	
	artifact := filepath.Join(buildDir, "handler.py")
	if err := os.WriteFile(artifact, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create artifact: %v", err)
	}
	
	buildOutput := &CachedBuildOutput{
		Handler:   "handler.handler",
		OutputDir: buildDir,
		Errors:    []string{},
	}
	
	functionID := "test-function"
	
	// Create cache entry
	entry := &CacheEntry{
		FunctionID:  functionID,
		BuildOutput: buildOutput,
	}
	
	err = resultCache.buildCache.Set(functionID, entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}
	
	// Cache the build result
	err = resultCache.CacheBuildResult(functionID, buildOutput, []string{artifact})
	if err != nil {
		t.Fatalf("Failed to cache build result: %v", err)
	}
	
	// Restore to a different directory
	restoreDir := filepath.Join(tempDir, "restore")
	restoredOutput, err := resultCache.RestoreBuildResult(functionID, restoreDir)
	if err != nil {
		t.Fatalf("Failed to restore build result: %v", err)
	}
	
	if restoredOutput.Handler != buildOutput.Handler {
		t.Errorf("Expected handler %s, got %s", buildOutput.Handler, restoredOutput.Handler)
	}
	
	if restoredOutput.OutputDir != restoreDir {
		t.Errorf("Expected output dir %s, got %s", restoreDir, restoredOutput.OutputDir)
	}
	
	// Verify restored artifact
	restoredArtifact := filepath.Join(restoreDir, "handler.py")
	if _, err := os.Stat(restoredArtifact); err != nil {
		t.Errorf("Restored artifact not found: %v", err)
	}
	
	content, err := os.ReadFile(restoredArtifact)
	if err != nil {
		t.Fatalf("Failed to read restored artifact: %v", err)
	}
	
	if string(content) != "def handler(): pass" {
		t.Errorf("Restored artifact content mismatch")
	}
}

func TestBuildResultCache_HasCachedResult(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_result_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	resultCache := createTestBuildResultCache(t, tempDir)
	
	functionID := "test-function"
	
	// Initially should not have cached result
	if resultCache.HasCachedResult(functionID) {
		t.Error("Should not have cached result initially")
	}
	
	// Create cache entry and artifacts
	buildOutput := &CachedBuildOutput{
		Handler:   "handler.handler",
		OutputDir: tempDir,
	}
	
	entry := &CacheEntry{
		FunctionID:  functionID,
		BuildOutput: buildOutput,
	}
	
	err = resultCache.buildCache.Set(functionID, entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}
	
	// Create artifact directory
	functionArtifactDir := filepath.Join(resultCache.artifactDir, functionID)
	if err := os.MkdirAll(functionArtifactDir, 0755); err != nil {
		t.Fatalf("Failed to create artifact dir: %v", err)
	}
	
	// Now should have cached result
	if !resultCache.HasCachedResult(functionID) {
		t.Error("Should have cached result after setup")
	}
}

func TestBuildResultCache_InvalidateCachedResult(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_result_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	resultCache := createTestBuildResultCache(t, tempDir)
	
	functionID := "test-function"
	
	// Create cache entry and artifacts
	buildOutput := &CachedBuildOutput{
		Handler:   "handler.handler",
		OutputDir: tempDir,
	}
	
	entry := &CacheEntry{
		FunctionID:  functionID,
		BuildOutput: buildOutput,
	}
	
	err = resultCache.buildCache.Set(functionID, entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}
	
	// Create artifact directory and file
	functionArtifactDir := filepath.Join(resultCache.artifactDir, functionID)
	if err := os.MkdirAll(functionArtifactDir, 0755); err != nil {
		t.Fatalf("Failed to create artifact dir: %v", err)
	}
	
	artifactFile := filepath.Join(functionArtifactDir, "handler.py")
	if err := os.WriteFile(artifactFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create artifact file: %v", err)
	}
	
	// Verify cached result exists
	if !resultCache.HasCachedResult(functionID) {
		t.Fatal("Should have cached result before invalidation")
	}
	
	// Invalidate
	err = resultCache.InvalidateCachedResult(functionID)
	if err != nil {
		t.Fatalf("Failed to invalidate cached result: %v", err)
	}
	
	// Verify cached result is gone
	if resultCache.HasCachedResult(functionID) {
		t.Error("Should not have cached result after invalidation")
	}
	
	// Verify artifact directory is removed
	if _, err := os.Stat(functionArtifactDir); !os.IsNotExist(err) {
		t.Error("Artifact directory should be removed after invalidation")
	}
}

func TestBuildResultCache_GetCacheSize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_result_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	resultCache := createTestBuildResultCache(t, tempDir)
	
	// Initially should be zero
	size, err := resultCache.GetCacheSize()
	if err != nil {
		t.Fatalf("Failed to get cache size: %v", err)
	}
	
	if size != 0 {
		t.Errorf("Expected initial size 0, got %d", size)
	}
	
	// Create some cached artifacts
	functionID := "test-function"
	functionArtifactDir := filepath.Join(resultCache.artifactDir, functionID)
	if err := os.MkdirAll(functionArtifactDir, 0755); err != nil {
		t.Fatalf("Failed to create artifact dir: %v", err)
	}
	
	testContent := "def handler(): pass"
	artifactFile := filepath.Join(functionArtifactDir, "handler.py")
	if err := os.WriteFile(artifactFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create artifact file: %v", err)
	}
	
	// Check size again
	size, err = resultCache.GetCacheSize()
	if err != nil {
		t.Fatalf("Failed to get cache size: %v", err)
	}
	
	expectedSize := int64(len(testContent))
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestBuildResultCache_CleanupExpiredResults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_result_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	resultCache := createTestBuildResultCache(t, tempDir)
	
	// Create orphaned artifact directory (no cache entry)
	orphanedFunctionID := "orphaned-function"
	orphanedDir := filepath.Join(resultCache.artifactDir, orphanedFunctionID)
	if err := os.MkdirAll(orphanedDir, 0755); err != nil {
		t.Fatalf("Failed to create orphaned dir: %v", err)
	}
	
	orphanedFile := filepath.Join(orphanedDir, "handler.py")
	if err := os.WriteFile(orphanedFile, []byte("orphaned"), 0644); err != nil {
		t.Fatalf("Failed to create orphaned file: %v", err)
	}
	
	// Create valid cached result
	validFunctionID := "valid-function"
	validDir := filepath.Join(resultCache.artifactDir, validFunctionID)
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatalf("Failed to create valid dir: %v", err)
	}
	
	// Add cache entry for valid function
	entry := &CacheEntry{
		FunctionID: validFunctionID,
		BuildOutput: &CachedBuildOutput{
			Handler: "handler.handler",
		},
	}
	
	err = resultCache.buildCache.Set(validFunctionID, entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}
	
	// Verify both directories exist
	if _, err := os.Stat(orphanedDir); err != nil {
		t.Fatal("Orphaned directory should exist before cleanup")
	}
	
	if _, err := os.Stat(validDir); err != nil {
		t.Fatal("Valid directory should exist before cleanup")
	}
	
	// Run cleanup
	err = resultCache.CleanupExpiredResults()
	if err != nil {
		t.Fatalf("Failed to cleanup expired results: %v", err)
	}
	
	// Verify orphaned directory is removed
	if _, err := os.Stat(orphanedDir); !os.IsNotExist(err) {
		t.Error("Orphaned directory should be removed after cleanup")
	}
	
	// Verify valid directory still exists
	if _, err := os.Stat(validDir); err != nil {
		t.Error("Valid directory should still exist after cleanup")
	}
}

func TestBuildResultCache_GetStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_result_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	resultCache := createTestBuildResultCache(t, tempDir)
	
	// Initially should have zero stats
	stats, err := resultCache.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	
	if stats.TotalFunctions != 0 {
		t.Errorf("Expected 0 functions, got %d", stats.TotalFunctions)
	}
	
	if stats.TotalSize != 0 {
		t.Errorf("Expected 0 size, got %d", stats.TotalSize)
	}
	
	if stats.ArtifactDir != resultCache.artifactDir {
		t.Errorf("Expected artifact dir %s, got %s", resultCache.artifactDir, stats.ArtifactDir)
	}
	
	// Create some cached functions
	for i := 0; i < 3; i++ {
		functionID := fmt.Sprintf("function-%d", i)
		functionDir := filepath.Join(resultCache.artifactDir, functionID)
		if err := os.MkdirAll(functionDir, 0755); err != nil {
			t.Fatalf("Failed to create function dir: %v", err)
		}
		
		artifactFile := filepath.Join(functionDir, "handler.py")
		content := fmt.Sprintf("def handler_%d(): pass", i)
		if err := os.WriteFile(artifactFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create artifact file: %v", err)
		}
	}
	
	// Check stats again
	stats, err = resultCache.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	
	if stats.TotalFunctions != 3 {
		t.Errorf("Expected 3 functions, got %d", stats.TotalFunctions)
	}
	
	if stats.TotalSize <= 0 {
		t.Errorf("Expected positive size, got %d", stats.TotalSize)
	}
}

func TestBuildResultCache_EvictLeastRecentlyUsed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_result_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	resultCache := createTestBuildResultCache(t, tempDir)
	
	// Create multiple cached functions with different access times
	functions := []string{"function-1", "function-2", "function-3"}
	
	for i, functionID := range functions {
		// Create cache entry with different access times
		entry := &CacheEntry{
			FunctionID:   functionID,
			LastAccessed: time.Now().Add(time.Duration(-i) * time.Hour), // Older entries first
			BuildOutput: &CachedBuildOutput{
				Handler: "handler.handler",
			},
		}
		
		err = resultCache.buildCache.Set(functionID, entry)
		if err != nil {
			t.Fatalf("Failed to set cache entry: %v", err)
		}
		
		// Create artifacts
		functionDir := filepath.Join(resultCache.artifactDir, functionID)
		if err := os.MkdirAll(functionDir, 0755); err != nil {
			t.Fatalf("Failed to create function dir: %v", err)
		}
		
		artifactFile := filepath.Join(functionDir, "handler.py")
		content := fmt.Sprintf("def handler_%d(): pass # This is a longer content to make the file bigger", i)
		if err := os.WriteFile(artifactFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create artifact file: %v", err)
		}
	}
	
	// Get initial size
	initialSize, err := resultCache.GetCacheSize()
	if err != nil {
		t.Fatalf("Failed to get initial cache size: %v", err)
	}
	
	// Set a small max size to trigger eviction
	maxSize := initialSize / 2
	
	err = resultCache.EvictLeastRecentlyUsed(maxSize)
	if err != nil {
		t.Fatalf("Failed to evict LRU entries: %v", err)
	}
	
	// Check that some entries were evicted
	finalSize, err := resultCache.GetCacheSize()
	if err != nil {
		t.Fatalf("Failed to get final cache size: %v", err)
	}
	
	if finalSize > maxSize {
		t.Errorf("Cache size %d should be less than max size %d after eviction", finalSize, maxSize)
	}
	
	// The oldest entry (function-1) should be evicted first
	function1Dir := filepath.Join(resultCache.artifactDir, "function-1")
	if _, err := os.Stat(function1Dir); !os.IsNotExist(err) {
		t.Error("Oldest function should be evicted first")
	}
}

// Helper function to create a test build result cache
func createTestBuildResultCache(t *testing.T, tempDir string) *BuildResultCache {
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: tempDir,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}
	
	artifactDir := filepath.Join(tempDir, "artifacts")
	
	config := BuildResultCacheConfig{
		BuildCache:  buildCache,
		ArtifactDir: artifactDir,
	}
	
	resultCache, err := NewBuildResultCache(config)
	if err != nil {
		t.Fatalf("Failed to create build result cache: %v", err)
	}
	
	return resultCache
}