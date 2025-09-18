package python

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDependencyCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dependency_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          filepath.Join(tempDir, "build"),
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	config := DependencyCacheConfig{
		CacheDir:              filepath.Join(tempDir, "deps"),
		BuildCache:            buildCache,
		MaxCacheSize:          1024 * 1024,
		MaxCacheAge:           time.Hour,
		EnableSharedCache:     true,
		EnableIntegrityChecks: true,
	}

	cache, err := NewDependencyCache(config)
	if err != nil {
		t.Fatalf("Failed to create dependency cache: %v", err)
	}

	if cache.cacheDir != config.CacheDir {
		t.Errorf("Expected cache dir %s, got %s", config.CacheDir, cache.cacheDir)
	}

	if cache.config.MaxCacheAge != config.MaxCacheAge {
		t.Errorf("Expected max cache age %v, got %v", config.MaxCacheAge, cache.config.MaxCacheAge)
	}
}

func TestNewDependencyCache_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config DependencyCacheConfig
	}{
		{
			name: "missing cache dir",
			config: DependencyCacheConfig{
				CacheDir: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDependencyCache(tt.config)
			if err == nil {
				t.Error("Expected error for invalid config, got nil")
			}
		})
	}
}

func TestDependencyCache_CacheDependencies(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dependency_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create build cache
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          filepath.Join(tempDir, "build"),
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create dependency cache
	cache, err := NewDependencyCache(DependencyCacheConfig{
		CacheDir:              filepath.Join(tempDir, "deps"),
		BuildCache:            buildCache,
		MaxCacheSize:          1024 * 1024,
		MaxCacheAge:           time.Hour,
		EnableSharedCache:     true,
		EnableIntegrityChecks: false, // Disable for simpler testing
	})
	if err != nil {
		t.Fatalf("Failed to create dependency cache: %v", err)
	}

	// Create test requirements file
	requirementsFile := filepath.Join(tempDir, "requirements.txt")
	requirementsContent := "requests==2.28.0\nnumpy>=1.20.0\n"
	if err := os.WriteFile(requirementsFile, []byte(requirementsContent), 0644); err != nil {
		t.Fatalf("Failed to create requirements file: %v", err)
	}

	// Create test install directory with some fake dependencies
	installDir := filepath.Join(tempDir, "install")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("Failed to create install dir: %v", err)
	}

	// Create fake dependency files
	requestsDir := filepath.Join(installDir, "requests")
	if err := os.MkdirAll(requestsDir, 0755); err != nil {
		t.Fatalf("Failed to create requests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(requestsDir, "__init__.py"), []byte("# requests"), 0644); err != nil {
		t.Fatalf("Failed to create requests __init__.py: %v", err)
	}

	numpyDir := filepath.Join(installDir, "numpy")
	if err := os.MkdirAll(numpyDir, 0755); err != nil {
		t.Fatalf("Failed to create numpy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(numpyDir, "__init__.py"), []byte("# numpy"), 0644); err != nil {
		t.Fatalf("Failed to create numpy __init__.py: %v", err)
	}

	// Cache the dependencies
	dependencyList := []string{"requests", "numpy"}
	architecture := "x86_64"

	err = cache.CacheDependencies(requirementsFile, architecture, installDir, dependencyList)
	if err != nil {
		t.Fatalf("Failed to cache dependencies: %v", err)
	}

	// Verify cache entry was created
	stats, err := cache.GetStats()
	if err != nil {
		t.Fatalf("Failed to get cache stats: %v", err)
	}

	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 cache entry, got %d", stats.TotalEntries)
	}

	if stats.TotalSize == 0 {
		t.Error("Expected non-zero cache size")
	}
}

func TestDependencyCache_GetCachedDependencies(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dependency_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create build cache
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          filepath.Join(tempDir, "build"),
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create dependency cache
	cache, err := NewDependencyCache(DependencyCacheConfig{
		CacheDir:              filepath.Join(tempDir, "deps"),
		BuildCache:            buildCache,
		MaxCacheSize:          1024 * 1024,
		MaxCacheAge:           time.Hour,
		EnableSharedCache:     true,
		EnableIntegrityChecks: false,
	})
	if err != nil {
		t.Fatalf("Failed to create dependency cache: %v", err)
	}

	// Create test requirements file
	requirementsFile := filepath.Join(tempDir, "requirements.txt")
	requirementsContent := "requests==2.28.0\n"
	if err := os.WriteFile(requirementsFile, []byte(requirementsContent), 0644); err != nil {
		t.Fatalf("Failed to create requirements file: %v", err)
	}

	// Create test install directory
	installDir := filepath.Join(tempDir, "install")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("Failed to create install dir: %v", err)
	}

	requestsDir := filepath.Join(installDir, "requests")
	if err := os.MkdirAll(requestsDir, 0755); err != nil {
		t.Fatalf("Failed to create requests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(requestsDir, "__init__.py"), []byte("# requests"), 0644); err != nil {
		t.Fatalf("Failed to create requests __init__.py: %v", err)
	}

	// Cache the dependencies first
	dependencyList := []string{"requests"}
	architecture := "x86_64"

	err = cache.CacheDependencies(requirementsFile, architecture, installDir, dependencyList)
	if err != nil {
		t.Fatalf("Failed to cache dependencies: %v", err)
	}

	// Now try to get cached dependencies
	targetDir := filepath.Join(tempDir, "target")
	cachedEntry, err := cache.GetCachedDependencies(requirementsFile, architecture, targetDir)
	if err != nil {
		t.Fatalf("Failed to get cached dependencies: %v", err)
	}

	if len(cachedEntry.DependencyList) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(cachedEntry.DependencyList))
	}

	if cachedEntry.DependencyList[0] != "requests" {
		t.Errorf("Expected 'requests', got %s", cachedEntry.DependencyList[0])
	}

	if cachedEntry.Architecture != architecture {
		t.Errorf("Expected architecture %s, got %s", architecture, cachedEntry.Architecture)
	}

	// Verify dependencies were copied to target directory
	targetRequestsDir := filepath.Join(targetDir, "requests")
	if _, err := os.Stat(targetRequestsDir); os.IsNotExist(err) {
		t.Error("Expected requests directory to be copied to target")
	}

	targetInitFile := filepath.Join(targetRequestsDir, "__init__.py")
	if _, err := os.Stat(targetInitFile); os.IsNotExist(err) {
		t.Error("Expected __init__.py to be copied to target")
	}
}

func TestDependencyCache_GetCachedDependencies_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dependency_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create build cache
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          filepath.Join(tempDir, "build"),
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create dependency cache
	cache, err := NewDependencyCache(DependencyCacheConfig{
		CacheDir:     filepath.Join(tempDir, "deps"),
		BuildCache:   buildCache,
		MaxCacheSize: 1024 * 1024,
		MaxCacheAge:  time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create dependency cache: %v", err)
	}

	// Create test requirements file
	requirementsFile := filepath.Join(tempDir, "requirements.txt")
	requirementsContent := "requests==2.28.0\n"
	if err := os.WriteFile(requirementsFile, []byte(requirementsContent), 0644); err != nil {
		t.Fatalf("Failed to create requirements file: %v", err)
	}

	// Try to get cached dependencies that don't exist
	targetDir := filepath.Join(tempDir, "target")
	_, err = cache.GetCachedDependencies(requirementsFile, "x86_64", targetDir)
	if err == nil {
		t.Error("Expected error for non-existent cache entry, got nil")
	}
}

func TestDependencyCache_ClearCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dependency_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create build cache
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          filepath.Join(tempDir, "build"),
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create dependency cache
	cache, err := NewDependencyCache(DependencyCacheConfig{
		CacheDir:     filepath.Join(tempDir, "deps"),
		BuildCache:   buildCache,
		MaxCacheSize: 1024 * 1024,
		MaxCacheAge:  time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create dependency cache: %v", err)
	}

	// Create and cache some dependencies
	requirementsFile := filepath.Join(tempDir, "requirements.txt")
	requirementsContent := "requests==2.28.0\n"
	if err := os.WriteFile(requirementsFile, []byte(requirementsContent), 0644); err != nil {
		t.Fatalf("Failed to create requirements file: %v", err)
	}

	installDir := filepath.Join(tempDir, "install")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("Failed to create install dir: %v", err)
	}

	err = cache.CacheDependencies(requirementsFile, "x86_64", installDir, []string{"requests"})
	if err != nil {
		t.Fatalf("Failed to cache dependencies: %v", err)
	}

	// Verify cache has entries
	stats, err := cache.GetStats()
	if err != nil {
		t.Fatalf("Failed to get cache stats: %v", err)
	}

	if stats.TotalEntries == 0 {
		t.Error("Expected cache to have entries before clearing")
	}

	// Clear cache
	err = cache.ClearCache()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Verify cache is empty
	stats, err = cache.GetStats()
	if err != nil {
		t.Fatalf("Failed to get cache stats after clearing: %v", err)
	}

	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 cache entries after clearing, got %d", stats.TotalEntries)
	}
}

func TestDependencyCache_GetStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dependency_cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create build cache
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          filepath.Join(tempDir, "build"),
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create dependency cache
	cache, err := NewDependencyCache(DependencyCacheConfig{
		CacheDir:     filepath.Join(tempDir, "deps"),
		BuildCache:   buildCache,
		MaxCacheSize: 1024 * 1024,
		MaxCacheAge:  time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create dependency cache: %v", err)
	}

	// Get stats for empty cache
	stats, err := cache.GetStats()
	if err != nil {
		t.Fatalf("Failed to get cache stats: %v", err)
	}

	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries for empty cache, got %d", stats.TotalEntries)
	}

	if stats.TotalSize != 0 {
		t.Errorf("Expected 0 size for empty cache, got %d", stats.TotalSize)
	}

	// Add some cache entries and verify stats
	requirementsFile := filepath.Join(tempDir, "requirements.txt")
	requirementsContent := "requests==2.28.0\n"
	if err := os.WriteFile(requirementsFile, []byte(requirementsContent), 0644); err != nil {
		t.Fatalf("Failed to create requirements file: %v", err)
	}

	installDir := filepath.Join(tempDir, "install")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("Failed to create install dir: %v", err)
	}

	// Create a test file to give the cache some size
	testFile := filepath.Join(installDir, "test.py")
	if err := os.WriteFile(testFile, []byte("# test file content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = cache.CacheDependencies(requirementsFile, "x86_64", installDir, []string{"requests"})
	if err != nil {
		t.Fatalf("Failed to cache dependencies: %v", err)
	}

	// Get stats after adding cache entry
	stats, err = cache.GetStats()
	if err != nil {
		t.Fatalf("Failed to get cache stats: %v", err)
	}

	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 entry after caching, got %d", stats.TotalEntries)
	}

	if stats.TotalSize == 0 {
		t.Error("Expected non-zero size after caching")
	}

	if stats.OldestEntry == 0 {
		t.Error("Expected non-zero oldest entry age")
	}

	if stats.NewestEntry == 0 {
		t.Error("Expected non-zero newest entry age")
	}
}
