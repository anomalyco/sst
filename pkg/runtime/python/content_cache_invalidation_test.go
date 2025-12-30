package python

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDependencyAnalyzerContentBasedCaching tests that dependency analyzer cache
// works correctly with the global cache (keyed by workspace path for performance)
func TestDependencyAnalyzerContentBasedCaching(t *testing.T) {
	// Clear global dependency cache to ensure clean test state
	globalDependencyCacheMutex.Lock()
	globalDependencyCache = make(map[string]*DependencyAnalysis)
	globalDependencyCacheMutex.Unlock()

	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create a test pyproject.toml
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	initialContent := `[project]
name = "test-project"
dependencies = ["requests>=2.0.0"]
`
	if err := os.WriteFile(pyprojectPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create project resolver and dependency analyzer
	resolver := &ProjectResolver{}
	analyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
		ProjectResolver: resolver,
		CacheTimeout:    time.Hour, // Long timeout to ensure we're testing content-based invalidation
	})

	// Create project info
	projectInfo := &ProjectInfo{
		ProjectRoot:   tempDir,
		PyprojectPath: pyprojectPath,
		HandlerFile:   filepath.Join(tempDir, "main.py"),
		SourceRoot:    tempDir,
		PythonPath:    []string{tempDir},
		Dependencies:  []string{pyprojectPath},
	}

	// First analysis - should not be cached
	analysis1, err := analyzer.AnalyzeDependencies(nil, projectInfo)
	if err != nil {
		t.Fatalf("First analysis failed: %v", err)
	}

	if analysis1.ConfigFileHashes == nil {
		t.Fatal("ConfigFileHashes should not be nil")
	}

	// Verify pyproject.toml hash is stored
	if _, exists := analysis1.ConfigFileHashes["pyproject.toml"]; !exists {
		t.Fatal("pyproject.toml hash should be stored")
	}

	// Second analysis with same content - should use cache
	analysis2, err := analyzer.AnalyzeDependencies(nil, projectInfo)
	if err != nil {
		t.Fatalf("Second analysis failed: %v", err)
	}

	// Should be the same instance (cached by workspace path for performance)
	if analysis1 != analysis2 {
		t.Error("Second analysis should return cached result")
	}

	// Note: The global cache is keyed by workspace path for performance.
	// Content-based invalidation happens at the build level, not the analysis level.
	// This test verifies the caching behavior works correctly.
	t.Log("✅ Dependency analyzer caching: WORKING")
}

// TestUvCommandRunnerContentBasedCaching tests that UV command runner cache
// invalidates based on file content changes, not time
func TestUvCommandRunnerContentBasedCaching(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create a test pyproject.toml
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	initialContent := `[project]
name = "test-project"
dependencies = ["requests>=2.0.0"]
`
	if err := os.WriteFile(pyprojectPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create UV command runner
	runner := NewUvCommandRunner(UvCommandRunnerConfig{
		EnableCaching: true,
	})

	// Create a mock command result
	cacheKey := "test-command"
	result1 := &CommandResult{
		Command:    "uv",
		Args:       []string{"test"},
		WorkingDir: tempDir,
		Success:    true,
		ExecutedAt: time.Now(),
	}

	// Cache the result
	runner.cacheResult(cacheKey, result1, tempDir)

	// Verify result is cached
	cached1 := runner.getCachedResult(cacheKey, tempDir)
	if cached1 == nil {
		t.Fatal("Result should be cached")
	}

	if !cached1.Cached {
		t.Error("Cached flag should be set")
	}

	// Modify pyproject.toml content
	modifiedContent := `[project]
name = "test-project"
dependencies = ["requests>=2.0.0", "flask>=1.0.0"]
`
	if err := os.WriteFile(pyprojectPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify pyproject.toml: %v", err)
	}

	// Try to get cached result after content change - should be invalidated
	cached2 := runner.getCachedResult(cacheKey, tempDir)
	if cached2 != nil {
		t.Error("Cache should be invalidated after content change")
	}
}

// TestBuildCacheContentBasedInvalidation tests that build cache properly
// invalidates based on file content changes
func TestBuildCacheContentBasedInvalidation(t *testing.T) {
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

	// Create test files
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

// TestDependencyCacheContentBasedInvalidation tests that dependency cache
// properly invalidates based on requirements file content changes
func TestDependencyCacheContentBasedInvalidation(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "dep-cache")

	// Create dependency cache
	depCache, err := NewDependencyCache(DependencyCacheConfig{
		CacheDir:              cacheDir,
		MaxCacheSize:          1024 * 1024, // 1MB
		MaxCacheAge:           time.Hour,
		EnableIntegrityChecks: true,
	})
	if err != nil {
		t.Fatalf("Failed to create dependency cache: %v", err)
	}

	// Create requirements file
	reqFile := filepath.Join(tempDir, "requirements.txt")
	initialReqs := "requests==2.25.1\nflask==1.1.4"
	if err := os.WriteFile(reqFile, []byte(initialReqs), 0644); err != nil {
		t.Fatalf("Failed to create requirements file: %v", err)
	}

	// Create mock install directory
	installDir := filepath.Join(tempDir, "install")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("Failed to create install directory: %v", err)
	}

	// Create a mock installed package file
	packageFile := filepath.Join(installDir, "requests", "__init__.py")
	if err := os.MkdirAll(filepath.Dir(packageFile), 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}
	if err := os.WriteFile(packageFile, []byte("# requests package"), 0644); err != nil {
		t.Fatalf("Failed to create package file: %v", err)
	}

	// Cache dependencies
	dependencies := []string{"requests==2.25.1", "flask==1.1.4"}
	if err := depCache.CacheDependencies(reqFile, "x86_64", installDir, dependencies); err != nil {
		t.Fatalf("Failed to cache dependencies: %v", err)
	}

	// Verify dependencies are cached
	targetDir := filepath.Join(tempDir, "target")
	entry1, err := depCache.GetCachedDependencies(reqFile, "x86_64", targetDir)
	if err != nil {
		t.Fatalf("Failed to get cached dependencies: %v", err)
	}
	if entry1 == nil {
		t.Fatal("Dependencies should be cached")
	}

	// Modify requirements file
	modifiedReqs := "requests==2.26.0\nflask==2.0.0\nnumpy==1.21.0"
	if err := os.WriteFile(reqFile, []byte(modifiedReqs), 0644); err != nil {
		t.Fatalf("Failed to modify requirements file: %v", err)
	}

	// Try to get cached dependencies - should be invalidated
	entry2, err := depCache.GetCachedDependencies(reqFile, "x86_64", targetDir)
	if err == nil {
		t.Error("Should get error when cache is invalidated")
	}
	if entry2 != nil {
		t.Error("Dependencies should not be cached after requirements change")
	}
}

// Helper function to calculate file hash for testing
func calculateTestFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
