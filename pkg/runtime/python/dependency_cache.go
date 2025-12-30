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

// DependencyCache manages caching of installed Python dependencies
type DependencyCache struct {
	// cacheDir is the directory where dependency caches are stored
	cacheDir string

	// buildCache provides access to build cache for metadata
	buildCache *BuildCache

	// mutex protects concurrent access
	mutex sync.RWMutex

	// config stores cache configuration
	config DependencyCacheConfig
}

// DependencyCacheConfig configures the dependency cache
type DependencyCacheConfig struct {
	// CacheDir is the directory for storing cached dependencies
	CacheDir string

	// BuildCache for accessing build metadata
	BuildCache *BuildCache

	// MaxCacheSize is the maximum size of the cache in bytes
	MaxCacheSize int64

	// MaxCacheAge is the maximum age for cache entries
	MaxCacheAge time.Duration

	// EnableSharedCache enables sharing dependencies across functions
	EnableSharedCache bool

	// EnableIntegrityChecks enables validation of cached dependencies
	EnableIntegrityChecks bool
}

// DependencyCacheEntry represents a cached dependency installation
type DependencyCacheEntry struct {
	// RequirementsHash is the hash of the requirements file
	RequirementsHash string `json:"requirementsHash"`

	// DependencyList contains the list of installed dependencies
	DependencyList []string `json:"dependencyList"`

	// InstallPath is the path where dependencies are installed
	InstallPath string `json:"installPath"`

	// Architecture is the target architecture
	Architecture string `json:"architecture"`

	// PythonVersion is the Python version used
	PythonVersion string `json:"pythonVersion"`

	// CreatedAt is when the cache entry was created
	CreatedAt time.Time `json:"createdAt"`

	// LastUsed is when the cache entry was last used
	LastUsed time.Time `json:"lastUsed"`

	// Size is the total size of cached dependencies in bytes
	Size int64 `json:"size"`

	// IntegrityHash is a hash of all installed files for validation
	IntegrityHash string `json:"integrityHash"`
}

// DependencyCacheStats contains statistics about the dependency cache
type DependencyCacheStats struct {
	// TotalEntries is the number of cache entries
	TotalEntries int `json:"totalEntries"`

	// TotalSize is the total size of cached dependencies
	TotalSize int64 `json:"totalSize"`

	// HitRate is the cache hit rate
	HitRate float64 `json:"hitRate"`

	// OldestEntry is the age of the oldest cache entry
	OldestEntry time.Duration `json:"oldestEntry"`

	// NewestEntry is the age of the newest cache entry
	NewestEntry time.Duration `json:"newestEntry"`
}

// NewDependencyCache creates a new dependency cache
func NewDependencyCache(config DependencyCacheConfig) (*DependencyCache, error) {
	if config.CacheDir == "" {
		return nil, fmt.Errorf("cache directory is required")
	}

	if config.MaxCacheAge == 0 {
		config.MaxCacheAge = 7 * 24 * time.Hour // 7 days default
	}

	if config.MaxCacheSize == 0 {
		config.MaxCacheSize = 5 * 1024 * 1024 * 1024 // 5GB default
	}

	// Create cache directory
	if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &DependencyCache{
		cacheDir:   config.CacheDir,
		buildCache: config.BuildCache,
		config:     config,
	}, nil
}

// GetCachedDependencies retrieves cached dependencies if available and valid
func (dc *DependencyCache) GetCachedDependencies(requirementsFile string, architecture string, targetDir string) (*DependencyCacheEntry, error) {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()

	// Calculate requirements hash with error recovery
	requirementsHash, err := dc.calculateRequirementsHash(requirementsFile)
	if err != nil {
		return nil, WrapError(err, "requirements hash calculation").
			WithContext("requirementsFile", requirementsFile).
			WithSuggestion("Check if the requirements file exists and is readable")
	}

	// Generate cache key
	cacheKey := dc.generateCacheKey(requirementsHash, architecture)
	cacheEntryPath := filepath.Join(dc.cacheDir, cacheKey+".json")

	// Check if cache entry exists
	if _, err := os.Stat(cacheEntryPath); os.IsNotExist(err) {
		return nil, NewPythonRuntimeError(ErrorTypeCacheInvalid, ErrorSeverityInfo, "cache entry not found").
			WithContext("cacheKey", cacheKey).
			WithContext("requirementsFile", requirementsFile)
	}

	// Load cache entry with error recovery
	entry, err := dc.loadCacheEntry(cacheEntryPath)
	if err != nil {
		// If cache entry is corrupted, remove it and return error
		corruptedErr := NewCacheCorruptedError(dc.cacheDir, err).
			WithContext("cacheEntryPath", cacheEntryPath)

		// Attempt automatic recovery
		if err := dc.removeCacheEntry(cacheKey); err != nil {
			slog.Warn("failed to remove corrupted cache entry", "error", err)
		}

		return nil, corruptedErr
	}

	// Validate cache entry with error recovery
	if err := dc.validateCacheEntry(entry); err != nil {
		slog.Warn("cache entry validation failed, removing",
			"cacheKey", cacheKey,
			"error", err)

		// Automatic recovery: remove invalid cache entry
		if removeErr := dc.removeCacheEntry(cacheKey); removeErr != nil {
			slog.Warn("failed to remove invalid cache entry", "error", removeErr)
		}

		return nil, NewCacheCorruptedError(dc.cacheDir, err).
			WithContext("cacheKey", cacheKey).
			WithContext("validationError", err.Error())
	}

	// Copy cached dependencies to target directory
	if err := dc.copyCachedDependencies(entry, targetDir); err != nil {
		return nil, fmt.Errorf("failed to copy cached dependencies: %w", err)
	}

	// Update last used time
	entry.LastUsed = time.Now()
	if err := dc.saveCacheEntry(cacheEntryPath, entry); err != nil {
		slog.Warn("failed to update cache entry last used time", "error", err)
	}

	slog.Info("using cached dependencies",
		"requirementsHash", requirementsHash,
		"architecture", architecture,
		"dependencyCount", len(entry.DependencyList),
		"cacheAge", time.Since(entry.CreatedAt))

	return entry, nil
}

// CacheDependencies stores installed dependencies in the cache
func (dc *DependencyCache) CacheDependencies(requirementsFile string, architecture string, installPath string, dependencyList []string) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	// Calculate requirements hash
	requirementsHash, err := dc.calculateRequirementsHash(requirementsFile)
	if err != nil {
		return fmt.Errorf("failed to calculate requirements hash: %w", err)
	}

	// Generate cache key
	cacheKey := dc.generateCacheKey(requirementsHash, architecture)
	cacheEntryPath := filepath.Join(dc.cacheDir, cacheKey+".json")
	cachedInstallPath := filepath.Join(dc.cacheDir, cacheKey)

	// Copy installed dependencies to cache
	if err := dc.copyDependenciesToCache(installPath, cachedInstallPath); err != nil {
		return fmt.Errorf("failed to copy dependencies to cache: %w", err)
	}

	// Calculate cache size and integrity hash
	size, err := dc.calculateDirectorySize(cachedInstallPath)
	if err != nil {
		slog.Warn("failed to calculate cache size", "error", err)
		size = 0
	}

	integrityHash := ""
	if dc.config.EnableIntegrityChecks {
		integrityHash, err = dc.calculateIntegrityHash(cachedInstallPath)
		if err != nil {
			slog.Warn("failed to calculate integrity hash", "error", err)
		}
	}

	// Create cache entry
	entry := &DependencyCacheEntry{
		RequirementsHash: requirementsHash,
		DependencyList:   dependencyList,
		InstallPath:      cachedInstallPath,
		Architecture:     architecture,
		PythonVersion:    dc.getPythonVersion(),
		CreatedAt:        time.Now(),
		LastUsed:         time.Now(),
		Size:             size,
		IntegrityHash:    integrityHash,
	}

	// Save cache entry
	if err := dc.saveCacheEntry(cacheEntryPath, entry); err != nil {
		// Clean up cached files if we can't save the entry
		os.RemoveAll(cachedInstallPath)
		return fmt.Errorf("failed to save cache entry: %w", err)
	}

	slog.Info("cached dependencies",
		"requirementsHash", requirementsHash,
		"architecture", architecture,
		"dependencyCount", len(dependencyList),
		"cacheSize", size)

	// Clean up old entries if cache is getting too large
	if err := dc.cleanupOldEntries(); err != nil {
		slog.Warn("failed to cleanup old cache entries", "error", err)
	}

	return nil
}

// calculateRequirementsHash calculates a hash of the requirements file
func (dc *DependencyCache) calculateRequirementsHash(requirementsFile string) (string, error) {
	file, err := os.Open(requirementsFile)
	if err != nil {
		return "", fmt.Errorf("failed to open requirements file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash requirements file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// generateCacheKey generates a cache key for the given parameters
func (dc *DependencyCache) generateCacheKey(requirementsHash string, architecture string) string {
	pythonVersion := dc.getPythonVersion()
	key := fmt.Sprintf("%s-%s-%s", requirementsHash, architecture, pythonVersion)

	// Hash the key to ensure it's a valid filename
	hasher := sha256.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars for shorter filenames
}

// loadCacheEntry loads a cache entry from disk
func (dc *DependencyCache) loadCacheEntry(entryPath string) (*DependencyCacheEntry, error) {
	data, err := os.ReadFile(entryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache entry: %w", err)
	}

	var entry DependencyCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	return &entry, nil
}

// saveCacheEntry saves a cache entry to disk
func (dc *DependencyCache) saveCacheEntry(entryPath string, entry *DependencyCacheEntry) error {
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	if err := os.WriteFile(entryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache entry: %w", err)
	}

	return nil
}

// validateCacheEntry validates that a cache entry is still valid
func (dc *DependencyCache) validateCacheEntry(entry *DependencyCacheEntry) error {
	// Check if cache entry is too old
	if time.Since(entry.CreatedAt) > dc.config.MaxCacheAge {
		return fmt.Errorf("cache entry is too old: %v", time.Since(entry.CreatedAt))
	}

	// Check if install path exists
	if _, err := os.Stat(entry.InstallPath); os.IsNotExist(err) {
		return fmt.Errorf("cached install path does not exist: %s", entry.InstallPath)
	}

	// Validate integrity if enabled
	if dc.config.EnableIntegrityChecks && entry.IntegrityHash != "" {
		currentHash, err := dc.calculateIntegrityHash(entry.InstallPath)
		if err != nil {
			return fmt.Errorf("failed to calculate current integrity hash: %w", err)
		}

		if currentHash != entry.IntegrityHash {
			return fmt.Errorf("integrity check failed: expected %s, got %s", entry.IntegrityHash, currentHash)
		}
	}

	return nil
}

// copyCachedDependencies copies cached dependencies to the target directory
func (dc *DependencyCache) copyCachedDependencies(entry *DependencyCacheEntry, targetDir string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Copy cached dependencies
	return dc.copyDirectory(entry.InstallPath, targetDir)
}

// copyDependenciesToCache copies installed dependencies to the cache
func (dc *DependencyCache) copyDependenciesToCache(installPath string, cachedPath string) error {
	// Remove existing cached path if it exists
	if err := os.RemoveAll(cachedPath); err != nil {
		return fmt.Errorf("failed to remove existing cached path: %w", err)
	}

	// Copy dependencies to cache
	return dc.copyDirectory(installPath, cachedPath)
}

// copyDirectory recursively copies a directory using hardlinks for speed
func (dc *DependencyCache) copyDirectory(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Skip unwanted directories (shouldn't be in cache, but be defensive)
		if info.IsDir() {
			dirName := filepath.Base(path)
			skipDirs := []string{
				".venv",
				"venv",
				"env",
				".env",
				"__pycache__",
				".pytest_cache",
				".mypy_cache",
				"node_modules",
			}
			for _, skipDir := range skipDirs {
				if dirName == skipDir {
					return filepath.SkipDir
				}
			}
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Check if source is a symlink - if so, we must copy instead of hardlink
		// UV installs use symlinks to its cache, and hardlinking to those would
		// create a hardlink to the UV cache file itself. If anything later modifies
		// the hardlinked file, it would corrupt the UV cache!
		fileInfo, err := os.Lstat(path)
		if err != nil {
			return err
		}

		if fileInfo.Mode()&os.ModeSymlink != 0 {
			// Source is a symlink - copy the file contents instead of hardlinking
			return dc.copyFile(path, dstPath)
		}

		// Try hardlink first (instant), fall back to copy if it fails
		if err := os.Link(path, dstPath); err == nil {
			return nil
		}

		// Hardlink failed (maybe cross-device), do regular copy
		return dc.copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func (dc *DependencyCache) copyFile(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// calculateDirectorySize calculates the total size of a directory
func (dc *DependencyCache) calculateDirectorySize(dirPath string) (int64, error) {
	var size int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// calculateIntegrityHash calculates a hash of all files in a directory
func (dc *DependencyCache) calculateIntegrityHash(dirPath string) (string, error) {
	hasher := sha256.New()

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Add file path to hash
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		hasher.Write([]byte(relPath))

		// Add file content to hash
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(hasher, file); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// getPythonVersion returns the Python version (simplified for now)
func (dc *DependencyCache) getPythonVersion() string {
	// TODO: Implement actual Python version detection
	return "3.11" // Default for now
}

// removeCacheEntry removes a cache entry and its associated files
func (dc *DependencyCache) removeCacheEntry(cacheKey string) error {
	cacheEntryPath := filepath.Join(dc.cacheDir, cacheKey+".json")
	cachedInstallPath := filepath.Join(dc.cacheDir, cacheKey)

	// Remove cache entry file
	if err := os.Remove(cacheEntryPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("failed to remove cache entry file", "path", cacheEntryPath, "error", err)
	}

	// Remove cached dependencies
	if err := os.RemoveAll(cachedInstallPath); err != nil {
		slog.Warn("failed to remove cached dependencies", "path", cachedInstallPath, "error", err)
	}

	return nil
}

// cleanupOldEntries removes old cache entries to keep cache size manageable
func (dc *DependencyCache) cleanupOldEntries() error {
	// Get all cache entries
	entries, err := dc.getAllCacheEntries()
	if err != nil {
		return fmt.Errorf("failed to get cache entries: %w", err)
	}

	// Calculate total cache size
	var totalSize int64
	for _, entry := range entries {
		totalSize += entry.Size
	}

	// If cache is within limits, no cleanup needed
	if totalSize <= dc.config.MaxCacheSize {
		return nil
	}

	slog.Info("cleaning up dependency cache",
		"totalSize", totalSize,
		"maxSize", dc.config.MaxCacheSize,
		"entries", len(entries))

	// Sort entries by last used time (oldest first)
	// Simple bubble sort for now
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].LastUsed.After(entries[j].LastUsed) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove oldest entries until we're under the size limit
	removedCount := 0
	for _, entry := range entries {
		if totalSize <= dc.config.MaxCacheSize {
			break
		}

		cacheKey := dc.generateCacheKey(entry.RequirementsHash, entry.Architecture)
		if err := dc.removeCacheEntry(cacheKey); err != nil {
			slog.Warn("failed to remove cache entry during cleanup", "cacheKey", cacheKey, "error", err)
			continue
		}

		totalSize -= entry.Size
		removedCount++
	}

	slog.Info("dependency cache cleanup completed",
		"removedEntries", removedCount,
		"newTotalSize", totalSize)

	return nil
}

// getAllCacheEntries returns all cache entries
func (dc *DependencyCache) getAllCacheEntries() ([]*DependencyCacheEntry, error) {
	var entries []*DependencyCacheEntry

	files, err := os.ReadDir(dc.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		entryPath := filepath.Join(dc.cacheDir, file.Name())
		entry, err := dc.loadCacheEntry(entryPath)
		if err != nil {
			slog.Warn("failed to load cache entry", "path", entryPath, "error", err)
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetStats returns statistics about the dependency cache
func (dc *DependencyCache) GetStats() (*DependencyCacheStats, error) {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()

	entries, err := dc.getAllCacheEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache entries: %w", err)
	}

	stats := &DependencyCacheStats{
		TotalEntries: len(entries),
		HitRate:      0.0, // TODO: Track hit rate
	}

	if len(entries) == 0 {
		return stats, nil
	}

	var totalSize int64
	oldest := entries[0].CreatedAt
	newest := entries[0].CreatedAt

	for _, entry := range entries {
		totalSize += entry.Size

		if entry.CreatedAt.Before(oldest) {
			oldest = entry.CreatedAt
		}
		if entry.CreatedAt.After(newest) {
			newest = entry.CreatedAt
		}
	}

	stats.TotalSize = totalSize
	stats.OldestEntry = time.Since(oldest)
	stats.NewestEntry = time.Since(newest)

	return stats, nil
}

// ClearCache removes all cached dependencies
func (dc *DependencyCache) ClearCache() error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	// Remove all files in cache directory
	files, err := os.ReadDir(dc.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, file := range files {
		filePath := filepath.Join(dc.cacheDir, file.Name())
		if err := os.RemoveAll(filePath); err != nil {
			slog.Warn("failed to remove cache file", "path", filePath, "error", err)
		}
	}

	slog.Info("dependency cache cleared")
	return nil
}

// InvalidateCache invalidates cached dependencies for specific requirements
func (dc *DependencyCache) InvalidateCache(requirementsFile string, architecture string) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	requirementsHash, err := dc.calculateRequirementsHash(requirementsFile)
	if err != nil {
		return fmt.Errorf("failed to calculate requirements hash: %w", err)
	}

	cacheKey := dc.generateCacheKey(requirementsHash, architecture)
	return dc.removeCacheEntry(cacheKey)
}
