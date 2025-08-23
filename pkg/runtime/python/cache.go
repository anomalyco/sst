package python

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BuildCache manages build state and change detection for Python functions
type BuildCache struct {
	// cacheDir is the directory where cache files are stored
	cacheDir string

	// entries stores cache entries in memory for fast access
	entries map[string]*CacheEntry

	// mutex protects concurrent access to the cache
	mutex sync.RWMutex

	// maxAge is the maximum age for cache entries before they're considered stale
	maxAge time.Duration

	// maxSize is the maximum number of cache entries to keep in memory
	maxSize int
}

// CacheEntry stores build metadata and file hashes for a specific function
type CacheEntry struct {
	// FunctionID uniquely identifies the function
	FunctionID string `json:"functionId"`

	// Handler is the handler path for this function
	Handler string `json:"handler"`

	// BuildTime is when this entry was last built
	BuildTime time.Time `json:"buildTime"`

	// LastAccessed is when this entry was last accessed
	LastAccessed time.Time `json:"lastAccessed"`

	// FileHashes contains SHA256 hashes of all relevant files
	FileHashes map[string]string `json:"fileHashes"`

	// Dependencies contains paths to all dependency files
	Dependencies []string `json:"dependencies"`

	// BuildOutput contains information about the build result
	BuildOutput *CachedBuildOutput `json:"buildOutput,omitempty"`

	// Properties contains the build properties used for this build
	Properties map[string]interface{} `json:"properties"`

	// LayoutInfo contains the detected layout information
	LayoutInfo *LayoutInfo `json:"layoutInfo,omitempty"`
}

// CachedBuildOutput stores information about a successful build
type CachedBuildOutput struct {
	// Handler is the resolved handler path
	Handler string `json:"handler"`

	// OutputDir is the directory containing build artifacts
	OutputDir string `json:"outputDir"`

	// Errors contains any build errors (usually empty for cached builds)
	Errors []string `json:"errors"`

	// Sourcemaps contains paths to source map files
	Sourcemaps []string `json:"sourcemaps"`

	// ArtifactPaths contains paths to all build artifacts
	ArtifactPaths []string `json:"artifactPaths"`

	// BuildDuration is how long the build took
	BuildDuration time.Duration `json:"buildDuration"`
}

// BuildCacheConfig configures the build cache
type BuildCacheConfig struct {
	// CacheDir is the directory to store cache files
	CacheDir string

	// MaxAge is the maximum age for cache entries
	MaxAge time.Duration

	// MaxSize is the maximum number of entries to keep in memory
	MaxSize int

	// EnablePersistence determines if cache should be persisted to disk
	EnablePersistence bool
}

// NewBuildCache creates a new build cache with the given configuration
func NewBuildCache(config BuildCacheConfig) (*BuildCache, error) {
	if config.CacheDir == "" {
		return nil, fmt.Errorf("cache directory is required")
	}

	if config.MaxAge == 0 {
		config.MaxAge = 24 * time.Hour // Default to 24 hours
	}

	if config.MaxSize == 0 {
		config.MaxSize = 1000 // Default to 1000 entries
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &BuildCache{
		cacheDir: config.CacheDir,
		entries:  make(map[string]*CacheEntry),
		maxAge:   config.MaxAge,
		maxSize:  config.MaxSize,
	}

	// Load existing cache entries if persistence is enabled
	if config.EnablePersistence {
		if err := cache.loadFromDisk(); err != nil {
			// Log error but don't fail - we can continue with empty cache
			fmt.Printf("Warning: failed to load cache from disk: %v\n", err)
		}
	}

	return cache, nil
}

// Get retrieves a cache entry for the given function ID
func (bc *BuildCache) Get(functionID string) (*CacheEntry, bool) {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	entry, exists := bc.entries[functionID]
	if !exists {
		return nil, false
	}

	// Check if entry is expired
	if time.Since(entry.BuildTime) > bc.maxAge {
		return nil, false
	}

	// Update last accessed time
	entry.LastAccessed = time.Now()

	return entry, true
}

// Set stores a cache entry for the given function ID
func (bc *BuildCache) Set(functionID string, entry *CacheEntry) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	// Update timestamps
	entry.FunctionID = functionID
	entry.BuildTime = time.Now()
	entry.LastAccessed = time.Now()

	// Store in memory
	bc.entries[functionID] = entry

	// Cleanup old entries if we exceed max size
	if len(bc.entries) > bc.maxSize {
		bc.cleanupOldEntries()
	}

	// Persist to disk with error handling
	if err := bc.persistEntry(functionID, entry); err != nil {
		// If persistence fails, still keep in memory but return wrapped error
		return WrapError(err, "cache persistence").
			WithContext("functionID", functionID).
			WithSuggestion("Check disk space and permissions for cache directory")
	}

	return nil
}

// Delete removes a cache entry for the given function ID
func (bc *BuildCache) Delete(functionID string) error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	delete(bc.entries, functionID)

	// Remove from disk
	cacheFile := filepath.Join(bc.cacheDir, functionID+".json")
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	return nil
}

// Clear removes all cache entries
func (bc *BuildCache) Clear() error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	bc.entries = make(map[string]*CacheEntry)

	// Remove all cache files from disk
	files, err := filepath.Glob(filepath.Join(bc.cacheDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list cache files: %w", err)
	}

	for _, file := range files {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("failed to remove cache file %s: %w", file, err)
		}
	}

	return nil
}

// IsValid checks if a cache entry is still valid by comparing file hashes
func (bc *BuildCache) IsValid(entry *CacheEntry) (bool, error) {
	// Check if all dependency files still exist and have the same hash
	for filePath, expectedHash := range entry.FileHashes {
		currentHash, err := bc.calculateFileHash(filePath)
		if err != nil {
			// File might have been deleted or is inaccessible
			return false, nil
		}

		if currentHash != expectedHash {
			return false, nil
		}
	}

	// Check if build output still exists
	if entry.BuildOutput != nil {
		for _, artifactPath := range entry.BuildOutput.ArtifactPaths {
			if _, err := os.Stat(artifactPath); err != nil {
				return false, nil
			}
		}
	}

	return true, nil
}

// UpdateFileHashes updates the file hashes for a cache entry
func (bc *BuildCache) UpdateFileHashes(entry *CacheEntry, filePaths []string) error {
	if entry.FileHashes == nil {
		entry.FileHashes = make(map[string]string)
	}

	for _, filePath := range filePaths {
		hash, err := bc.calculateFileHash(filePath)
		if err != nil {
			return fmt.Errorf("failed to calculate hash for %s: %w", filePath, err)
		}
		entry.FileHashes[filePath] = hash
	}

	return nil
}

// calculateFileHash calculates the SHA256 hash of a file
func (bc *BuildCache) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// persistEntry saves a cache entry to disk
func (bc *BuildCache) persistEntry(functionID string, entry *CacheEntry) error {
	cacheFile := filepath.Join(bc.cacheDir, functionID+".json")

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// loadFromDisk loads all cache entries from disk
func (bc *BuildCache) loadFromDisk() error {
	files, err := filepath.Glob(filepath.Join(bc.cacheDir, "*.json"))
	if err != nil {
		return WrapError(err, "cache loading").
			WithContext("cacheDir", bc.cacheDir).
			WithSuggestion("Check if cache directory exists and is accessible")
	}

	var corruptedFiles []string
	loadedCount := 0

	for _, file := range files {
		functionID := strings.TrimSuffix(filepath.Base(file), ".json")

		data, err := os.ReadFile(file)
		if err != nil {
			corruptedFiles = append(corruptedFiles, file)
			continue
		}

		var entry CacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			corruptedFiles = append(corruptedFiles, file)
			continue
		}

		// Only load entries that aren't expired
		if time.Since(entry.BuildTime) <= bc.maxAge {
			bc.entries[functionID] = &entry
			loadedCount++
		}
	}

	// Clean up corrupted files automatically
	if len(corruptedFiles) > 0 {
		slog.Warn("found corrupted cache files, cleaning up",
			"corruptedCount", len(corruptedFiles),
			"loadedCount", loadedCount)

		for _, file := range corruptedFiles {
			if err := os.Remove(file); err != nil {
				slog.Warn("failed to remove corrupted cache file", "file", file, "error", err)
			}
		}
	}

	slog.Debug("loaded cache entries from disk",
		"loadedCount", loadedCount,
		"corruptedCount", len(corruptedFiles))

	return nil
}

// cleanupOldEntries removes the oldest entries to stay within maxSize limit
func (bc *BuildCache) cleanupOldEntries() {
	if len(bc.entries) <= bc.maxSize {
		return
	}

	// Find the oldest entries by last accessed time
	type entryAge struct {
		functionID string
		lastAccess time.Time
	}

	var ages []entryAge
	for functionID, entry := range bc.entries {
		ages = append(ages, entryAge{
			functionID: functionID,
			lastAccess: entry.LastAccessed,
		})
	}

	// Sort by last access time (oldest first)
	for i := 0; i < len(ages)-1; i++ {
		for j := i + 1; j < len(ages); j++ {
			if ages[i].lastAccess.After(ages[j].lastAccess) {
				ages[i], ages[j] = ages[j], ages[i]
			}
		}
	}

	// Remove oldest entries until we're within the limit
	entriesToRemove := len(bc.entries) - bc.maxSize
	for i := 0; i < entriesToRemove && i < len(ages); i++ {
		functionID := ages[i].functionID
		delete(bc.entries, functionID)

		// Also remove from disk
		cacheFile := filepath.Join(bc.cacheDir, functionID+".json")
		os.Remove(cacheFile) // Ignore errors
	}
}

// GetStats returns statistics about the cache
func (bc *BuildCache) GetStats() CacheStats {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	stats := CacheStats{
		TotalEntries: len(bc.entries),
		MaxSize:      bc.maxSize,
		MaxAge:       bc.maxAge,
		CacheDir:     bc.cacheDir,
	}

	// Calculate cache size on disk
	files, err := filepath.Glob(filepath.Join(bc.cacheDir, "*.json"))
	if err == nil {
		stats.DiskFiles = len(files)

		var totalSize int64
		for _, file := range files {
			if info, err := os.Stat(file); err == nil {
				totalSize += info.Size()
			}
		}
		stats.DiskSize = totalSize
	}

	return stats
}

// CacheStats contains statistics about the build cache
type CacheStats struct {
	TotalEntries int           `json:"totalEntries"`
	MaxSize      int           `json:"maxSize"`
	MaxAge       time.Duration `json:"maxAge"`
	CacheDir     string        `json:"cacheDir"`
	DiskFiles    int           `json:"diskFiles"`
	DiskSize     int64         `json:"diskSize"`
}

// Cleanup removes expired entries and performs maintenance
func (bc *BuildCache) Cleanup() error {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	now := time.Now()
	var toDelete []string

	// Find expired entries
	for functionID, entry := range bc.entries {
		if now.Sub(entry.BuildTime) > bc.maxAge {
			toDelete = append(toDelete, functionID)
		}
	}

	// Remove expired entries
	for _, functionID := range toDelete {
		delete(bc.entries, functionID)

		// Remove from disk
		cacheFile := filepath.Join(bc.cacheDir, functionID+".json")
		os.Remove(cacheFile) // Ignore errors
	}

	return nil
}

// BuildResultCache manages caching of build results and artifacts
type BuildResultCache struct {
	// buildCache is the underlying build cache
	buildCache *BuildCache

	// artifactDir is the directory where cached artifacts are stored
	artifactDir string

	// mutex protects concurrent access
	mutex sync.RWMutex
}

// BuildResultCacheConfig configures the build result cache
type BuildResultCacheConfig struct {
	// BuildCache is the underlying build cache
	BuildCache *BuildCache

	// ArtifactDir is the directory to store cached artifacts
	ArtifactDir string
}

// NewBuildResultCache creates a new build result cache
func NewBuildResultCache(config BuildResultCacheConfig) (*BuildResultCache, error) {
	if config.BuildCache == nil {
		return nil, fmt.Errorf("build cache is required")
	}

	if config.ArtifactDir == "" {
		return nil, fmt.Errorf("artifact directory is required")
	}

	// Ensure artifact directory exists
	if err := os.MkdirAll(config.ArtifactDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	return &BuildResultCache{
		buildCache:  config.BuildCache,
		artifactDir: config.ArtifactDir,
	}, nil
}

// CacheBuildResult stores build artifacts and metadata for reuse
func (brc *BuildResultCache) CacheBuildResult(functionID string, buildOutput *CachedBuildOutput, artifactPaths []string) error {
	brc.mutex.Lock()
	defer brc.mutex.Unlock()

	// Create function-specific artifact directory
	functionArtifactDir := filepath.Join(brc.artifactDir, functionID)
	if err := os.MkdirAll(functionArtifactDir, 0755); err != nil {
		return fmt.Errorf("failed to create function artifact directory: %w", err)
	}

	// Copy artifacts to cache directory
	var cachedArtifactPaths []string
	for _, artifactPath := range artifactPaths {
		if _, err := os.Stat(artifactPath); err != nil {
			continue // Skip missing artifacts
		}

		// Determine destination path
		relativePath, err := filepath.Rel(buildOutput.OutputDir, artifactPath)
		if err != nil {
			// If we can't get relative path, use the base name
			relativePath = filepath.Base(artifactPath)
		}

		destPath := filepath.Join(functionArtifactDir, relativePath)

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Copy artifact
		if err := brc.copyFile(artifactPath, destPath); err != nil {
			return fmt.Errorf("failed to copy artifact %s: %w", artifactPath, err)
		}

		cachedArtifactPaths = append(cachedArtifactPaths, destPath)
	}

	// Update build output with cached paths
	buildOutput.ArtifactPaths = cachedArtifactPaths
	buildOutput.OutputDir = functionArtifactDir

	return nil
}

// RestoreBuildResult restores cached build artifacts to the target directory
func (brc *BuildResultCache) RestoreBuildResult(functionID string, targetDir string) (*CachedBuildOutput, error) {
	brc.mutex.RLock()
	defer brc.mutex.RUnlock()

	// Get cached build entry
	entry, exists := brc.buildCache.Get(functionID)
	if !exists || entry.BuildOutput == nil {
		return nil, fmt.Errorf("no cached build result found for function %s", functionID)
	}

	buildOutput := entry.BuildOutput
	functionArtifactDir := filepath.Join(brc.artifactDir, functionID)

	// Check if cached artifacts exist
	if _, err := os.Stat(functionArtifactDir); err != nil {
		return nil, fmt.Errorf("cached artifacts not found for function %s", functionID)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	// Check if this is a package cache (starts with pkg_)
	isPackageCache := strings.HasPrefix(functionID, "pkg_")

	// Copy cached artifacts to target directory
	var restoredPaths []string
	err := filepath.Walk(functionArtifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Calculate relative path from function artifact directory
		relativePath, err := filepath.Rel(functionArtifactDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// For package caches, we need to be careful about path adjustments
		// The issue is that pkg_ directories shouldn't appear in the final output
		var destPath string
		if isPackageCache {
			// Check if the relative path contains the cache key (pkg_packagename)
			// If so, we need to remove the pkg_ prefix from directory names
			pathParts := strings.Split(relativePath, string(filepath.Separator))
			for i, part := range pathParts {
				if strings.HasPrefix(part, "pkg_") {
					// Remove the pkg_ prefix from this path component
					pathParts[i] = strings.TrimPrefix(part, "pkg_")
				}
			}
			adjustedPath := strings.Join(pathParts, string(filepath.Separator))
			destPath = filepath.Join(targetDir, adjustedPath)
		} else {
			destPath = filepath.Join(targetDir, relativePath)
		}

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Copy file
		if err := brc.copyFile(path, destPath); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", path, err)
		}

		restoredPaths = append(restoredPaths, destPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to restore build artifacts: %w", err)
	}

	// Create new build output with restored paths
	restoredBuildOutput := &CachedBuildOutput{
		Handler:       buildOutput.Handler,
		OutputDir:     targetDir,
		Errors:        buildOutput.Errors,
		Sourcemaps:    buildOutput.Sourcemaps,
		ArtifactPaths: restoredPaths,
		BuildDuration: buildOutput.BuildDuration,
	}

	return restoredBuildOutput, nil
}

// HasCachedResult checks if a cached build result exists for the function
func (brc *BuildResultCache) HasCachedResult(functionID string) bool {
	brc.mutex.RLock()
	defer brc.mutex.RUnlock()

	return brc.hasCachedResultInternal(functionID)
}

// hasCachedResultInternal checks if a cached build result exists without acquiring lock
func (brc *BuildResultCache) hasCachedResultInternal(functionID string) bool {
	entry, exists := brc.buildCache.Get(functionID)
	if !exists || entry.BuildOutput == nil {
		return false
	}

	// Check if artifact directory exists
	functionArtifactDir := filepath.Join(brc.artifactDir, functionID)
	if _, err := os.Stat(functionArtifactDir); err != nil {
		return false
	}

	return true
}

// InvalidateCachedResult removes cached build result for a function
func (brc *BuildResultCache) InvalidateCachedResult(functionID string) error {
	brc.mutex.Lock()
	defer brc.mutex.Unlock()

	return brc.invalidateCachedResultInternal(functionID)
}

// invalidateCachedResultInternal removes cached build result without acquiring lock
func (brc *BuildResultCache) invalidateCachedResultInternal(functionID string) error {
	// Remove from build cache
	if err := brc.buildCache.Delete(functionID); err != nil {
		return fmt.Errorf("failed to remove from build cache: %w", err)
	}

	// Remove cached artifacts
	functionArtifactDir := filepath.Join(brc.artifactDir, functionID)
	if err := os.RemoveAll(functionArtifactDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cached artifacts: %w", err)
	}

	return nil
}

// CleanupExpiredResults removes expired cached results
func (brc *BuildResultCache) CleanupExpiredResults() error {
	brc.mutex.Lock()
	defer brc.mutex.Unlock()

	// Get all function directories
	entries, err := os.ReadDir(brc.artifactDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, nothing to clean
		}
		return fmt.Errorf("failed to read artifact directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		functionID := entry.Name()

		// Check if function still has valid cache entry
		if !brc.hasCachedResultInternal(functionID) {
			// Remove orphaned artifact directory
			functionArtifactDir := filepath.Join(brc.artifactDir, functionID)
			if err := os.RemoveAll(functionArtifactDir); err != nil {
				// Log error but continue cleanup
				fmt.Printf("Warning: failed to remove orphaned artifacts for %s: %v\n", functionID, err)
			}
		}
	}

	return nil
}

// GetCacheSize returns the total size of cached artifacts
func (brc *BuildResultCache) GetCacheSize() (int64, error) {
	brc.mutex.RLock()
	defer brc.mutex.RUnlock()

	var totalSize int64

	err := filepath.Walk(brc.artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate cache size: %w", err)
	}

	return totalSize, nil
}

// EvictLeastRecentlyUsed removes the least recently used cached results to free space
func (brc *BuildResultCache) EvictLeastRecentlyUsed(maxSize int64) error {
	brc.mutex.Lock()
	defer brc.mutex.Unlock()

	// Calculate current size without calling GetCacheSize (which would deadlock)
	var currentSize int64
	err := filepath.Walk(brc.artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if !info.IsDir() {
			currentSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to calculate current cache size: %w", err)
	}

	if currentSize <= maxSize {
		return nil // No eviction needed
	}

	// Get all cache entries sorted by last accessed time
	type entryInfo struct {
		functionID   string
		lastAccessed time.Time
		size         int64
	}

	var entries []entryInfo

	// Read artifact directories
	dirEntries, err := os.ReadDir(brc.artifactDir)
	if err != nil {
		return fmt.Errorf("failed to read artifact directory: %w", err)
	}

	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}

		functionID := dirEntry.Name()
		cacheEntry, exists := brc.buildCache.Get(functionID)
		if !exists {
			continue
		}

		// Calculate size of this function's artifacts
		functionArtifactDir := filepath.Join(brc.artifactDir, functionID)
		var functionSize int64

		filepath.Walk(functionArtifactDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				functionSize += info.Size()
			}
			return nil
		})

		entries = append(entries, entryInfo{
			functionID:   functionID,
			lastAccessed: cacheEntry.LastAccessed,
			size:         functionSize,
		})
	}

	// Sort by last accessed time (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].lastAccessed.After(entries[j].lastAccessed) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Evict entries until we're under the size limit
	sizeToFree := currentSize - maxSize
	var freedSize int64

	for _, entry := range entries {
		if freedSize >= sizeToFree {
			break
		}

		if err := brc.invalidateCachedResultInternal(entry.functionID); err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to evict cached result for %s: %v\n", entry.functionID, err)
			continue
		}

		freedSize += entry.size
	}

	return nil
}

// copyFile copies a file from src to dst
func (brc *BuildResultCache) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Copy file permissions
	srcInfo, err := srcFile.Stat()
	if err == nil {
		dstFile.Chmod(srcInfo.Mode())
	}

	return nil
}

// BuildResultCacheStats contains statistics about the build result cache
type BuildResultCacheStats struct {
	TotalFunctions int    `json:"totalFunctions"`
	TotalSize      int64  `json:"totalSize"`
	ArtifactDir    string `json:"artifactDir"`
}

// GetStats returns statistics about the build result cache
func (brc *BuildResultCache) GetStats() (*BuildResultCacheStats, error) {
	brc.mutex.RLock()
	defer brc.mutex.RUnlock()

	totalSize, err := brc.GetCacheSize()
	if err != nil {
		return nil, err
	}

	// Count function directories
	entries, err := os.ReadDir(brc.artifactDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &BuildResultCacheStats{
				TotalFunctions: 0,
				TotalSize:      0,
				ArtifactDir:    brc.artifactDir,
			}, nil
		}
		return nil, fmt.Errorf("failed to read artifact directory: %w", err)
	}

	functionCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			functionCount++
		}
	}

	return &BuildResultCacheStats{
		TotalFunctions: functionCount,
		TotalSize:      totalSize,
		ArtifactDir:    brc.artifactDir,
	}, nil
}
