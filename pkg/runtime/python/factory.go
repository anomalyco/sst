package python

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

const (
	sstCacheDevDir    = ".sst/cache/dev"
	sstCacheDeployDir = ".sst/cache/deploy"

	defaultDependencyCacheAge = 24 * time.Hour
	defaultMaxCacheSize       = 1024 * 1024 * 1024 // 1GB
)

// getCacheDirForMode returns the cache directory for dev or deploy mode
func getCacheDirForMode(workingDir string, isDev bool) string {
	if isDev {
		return filepath.Join(workingDir, sstCacheDevDir)
	}
	return filepath.Join(workingDir, sstCacheDeployDir)
}

// RuntimeFactory creates Python runtime components with sensible defaults
// This decouples high-level code from implementation details
type RuntimeFactory struct{}

// NewRuntimeFactory creates a new runtime factory
func NewRuntimeFactory() *RuntimeFactory {
	return &RuntimeFactory{}
}

// CreateCacheSystem creates a complete cache system with sensible defaults
func (rf *RuntimeFactory) CreateCacheSystem(cacheDir string) (*BuildCache, *ChangeDetector, *ProjectResolver, error) {
	// Create project resolver with defaults
	projectResolver := NewProjectResolver(cacheDir)

	// Create build cache with defaults
	buildCache, err := NewDefaultBuildCache(cacheDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create build cache: %w", err)
	}

	// Create change detector
	changeDetector, err := NewChangeDetector(ChangeDetectorConfig{
		ProjectResolver: projectResolver,
		BuildCache:      buildCache,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create change detector: %w", err)
	}

	return buildCache, changeDetector, projectResolver, nil
}

// CreateIncrementalBuilder creates an incremental builder with sensible defaults
func (rf *RuntimeFactory) CreateIncrementalBuilder(workingDir string, input *runtime.BuildInput, progressCallback ProgressCallback) (*IncrementalBuilder, error) {
	return NewIncrementalBuilder(IncrementalBuilderConfig{
		CacheDir:                getCacheDirForMode(workingDir, input.Dev),
		ArtifactDir:             input.Out(),
		MaxCacheAge:             defaultDependencyCacheAge, // For dependency cache, NOT build cache
		MaxCacheSize:            defaultMaxCacheSize,
		EnableParallelBuilds:    false, // Disabled to prevent thread explosion
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
		FunctionID:              input.FunctionID,
		ProgressCallback:        progressCallback,
		ProjectRoot:             workingDir,
	})
}

// CreateIncrementalBuilderWithCacheDir creates an incremental builder with a specific cache directory
func (rf *RuntimeFactory) CreateIncrementalBuilderWithCacheDir(workingDir string, input *runtime.BuildInput, progressCallback ProgressCallback, cacheDir string) (*IncrementalBuilder, error) {
	return NewIncrementalBuilder(IncrementalBuilderConfig{
		CacheDir:                cacheDir,
		ArtifactDir:             input.Out(),
		MaxCacheAge:             defaultDependencyCacheAge, // For dependency cache, NOT build cache
		MaxCacheSize:            defaultMaxCacheSize,
		EnableParallelBuilds:    false, // Disabled to prevent thread explosion
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
		FunctionID:              input.FunctionID,
		ProgressCallback:        progressCallback,
		ProjectRoot:             workingDir,
	})
}
