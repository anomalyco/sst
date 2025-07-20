package python

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// IncrementalBuilder coordinates selective builds and optimizes the build process
type IncrementalBuilder struct {
	// buildCache manages build state and change detection
	buildCache *BuildCache

	// layoutDetector provides flexible project layout detection
	layoutDetector *LayoutDetector

	// changeDetector monitors file modifications
	changeDetector *ChangeDetector

	// buildResultCache manages caching of build artifacts
	buildResultCache *BuildResultCache

	// dependencyAnalyzer analyzes package dependencies
	dependencyAnalyzer *DependencyAnalyzer

	// buildPlanner optimizes build order and parallelization
	buildPlanner *BuildPlanner

	// uvRunner executes UV commands efficiently
	uvRunner *UvCommandRunner

	// dependencyCache manages caching of installed dependencies
	dependencyCache *DependencyCache

	// progressReporter tracks and reports build progress
	progressReporter *ProgressReporter

	// fallbackManager handles fallback strategies
	fallbackManager *FallbackManager

	// deprecationChecker checks for deprecated patterns
	deprecationChecker *DeprecationChecker

	// mutex protects concurrent access
	mutex sync.RWMutex

	// config stores builder configuration
	config IncrementalBuilderConfig
}

// IncrementalBuilderConfig configures the incremental builder
type IncrementalBuilderConfig struct {
	// CacheDir is the directory for storing cache files
	CacheDir string

	// ArtifactDir is the directory for storing build artifacts
	ArtifactDir string

	// MaxCacheSize is the maximum size of the cache in bytes
	MaxCacheSize int64

	// MaxCacheAge is the maximum age for cache entries
	MaxCacheAge time.Duration

	// EnableParallelBuilds enables parallel building of independent packages
	EnableParallelBuilds bool

	// MaxParallelBuilds is the maximum number of parallel builds
	MaxParallelBuilds int

	// EnableProgressReporting enables progress reporting for builds
	EnableProgressReporting bool

	// EnableBuildOptimization enables various build optimizations
	EnableBuildOptimization bool

	// ProgressCallback is a function to receive progress updates
	ProgressCallback ProgressCallback

	// EnableFallbacks enables fallback mechanisms
	EnableFallbacks bool

	// EnableLegacyFallback enables falling back to legacy builder
	EnableLegacyFallback bool

	// LegacyBuilder is the legacy Python builder for fallback
	LegacyBuilder LegacyPythonBuilder

	// FallbackCallback is called when fallbacks occur
	FallbackCallback FallbackCallback

	// EnableDeprecationWarnings enables deprecation warnings
	EnableDeprecationWarnings bool

	// DeprecationCallback is called when deprecation warnings are issued
	DeprecationCallback DeprecationCallback
}

// BuildPlan represents a plan for building packages
type BuildPlan struct {
	// Packages contains the packages that need to be built
	Packages []*PackageBuildInfo

	// BuildOrder defines the order in which packages should be built
	BuildOrder []string

	// ParallelGroups defines groups of packages that can be built in parallel
	ParallelGroups [][]string

	// EstimatedDuration is the estimated time for the entire build
	EstimatedDuration time.Duration

	// RequiredDependencies lists all dependencies that need to be installed
	RequiredDependencies []string

	// CacheHits lists packages that can use cached results
	CacheHits []string

	// ForcedRebuilds lists packages that must be rebuilt
	ForcedRebuilds []string
}

// PackageBuildInfo contains information about a package build
type PackageBuildInfo struct {
	// PackageName is the name of the package
	PackageName string

	// PackageDir is the directory containing the package
	PackageDir string

	// Dependencies lists the package dependencies
	Dependencies []string

	// SourceFiles lists all source files in the package
	SourceFiles []string

	// RequiresRebuild indicates if the package needs to be rebuilt
	RequiresRebuild bool

	// RebuildReason explains why a rebuild is needed
	RebuildReason string

	// EstimatedBuildTime is the estimated time to build this package
	EstimatedBuildTime time.Duration

	// CanUseCache indicates if cached results can be used
	CanUseCache bool

	// CacheKey is the key for cached results
	CacheKey string
}

// BuildResult represents the result of an incremental build
type BuildResult struct {
	// Success indicates if the build was successful
	Success bool

	// BuildOutput contains the standard build output
	BuildOutput *runtime.BuildOutput

	// CachedBuildOutput contains cached build information
	CachedBuildOutput *CachedBuildOutput

	// BuildDuration is the total time taken for the build
	BuildDuration time.Duration

	// PackagesBuilt lists the packages that were actually built
	PackagesBuilt []string

	// PackagesCached lists the packages that used cached results
	PackagesCached []string

	// Errors contains any build errors
	Errors []string

	// Warnings contains any build warnings
	Warnings []string

	// BuildPlan contains the build plan that was executed
	BuildPlan *BuildPlan
}

// NewIncrementalBuilder creates a new incremental builder with the given configuration
func NewIncrementalBuilder(config IncrementalBuilderConfig) (*IncrementalBuilder, error) {
	// Set default values
	if config.MaxCacheAge == 0 {
		config.MaxCacheAge = 24 * time.Hour
	}
	if config.MaxCacheSize == 0 {
		config.MaxCacheSize = 1024 * 1024 * 1024 // 1GB default
	}
	if config.MaxParallelBuilds == 0 {
		config.MaxParallelBuilds = 4
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create artifact directory if it doesn't exist
	if err := os.MkdirAll(config.ArtifactDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Initialize layout detector
	layoutDetector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot:  config.CacheDir, // Will be updated per build
		CacheTimeout: 5 * time.Minute,
	})

	// Initialize build cache
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          config.CacheDir,
		MaxAge:            config.MaxCacheAge,
		MaxSize:           1000,
		EnablePersistence: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create build cache: %w", err)
	}

	// Initialize change detector
	changeDetector, err := NewChangeDetector(ChangeDetectorConfig{
		LayoutDetector: layoutDetector,
		BuildCache:     buildCache,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create change detector: %w", err)
	}

	// Initialize build result cache
	buildResultCache, err := NewBuildResultCache(BuildResultCacheConfig{
		BuildCache:  buildCache,
		ArtifactDir: config.ArtifactDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create build result cache: %w", err)
	}

	// Initialize dependency analyzer
	dependencyAnalyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
		LayoutDetector: layoutDetector,
		BuildCache:     buildCache,
	})

	// Initialize build planner
	buildPlanner := NewBuildPlanner(BuildPlannerConfig{
		DependencyAnalyzer:   dependencyAnalyzer,
		EnableParallelBuilds: config.EnableParallelBuilds,
		MaxParallelBuilds:    config.MaxParallelBuilds,
	})

	// Initialize UV command runner
	uvRunner := NewUvCommandRunner(UvCommandRunnerConfig{
		EnableCaching:        config.EnableBuildOptimization,
		EnableProgressReport: config.EnableProgressReporting,
		BuildCache:           buildCache,
	})

	// Initialize dependency cache
	dependencyCache, err := NewDependencyCache(DependencyCacheConfig{
		CacheDir:              filepath.Join(config.CacheDir, "dependencies"),
		BuildCache:            buildCache,
		MaxCacheSize:          config.MaxCacheSize / 2, // Use half of total cache for dependencies
		MaxCacheAge:           config.MaxCacheAge,
		EnableSharedCache:     true,
		EnableIntegrityChecks: config.EnableBuildOptimization,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dependency cache: %w", err)
	}

	// Create progress reporter with estimated build duration of 30 seconds
	// This will be adjusted based on actual build times
	estimatedDuration := 30 * time.Second
	progressReporter := NewProgressReporter(estimatedDuration, config.EnableProgressReporting)

	// Register progress callback if provided
	if config.ProgressCallback != nil {
		progressReporter.RegisterCallback(config.ProgressCallback)
	}

	// Create fallback manager
	var fallbackManager *FallbackManager
	if config.EnableFallbacks {
		fallbackConfig := FallbackManagerConfig{
			EnableLegacyFallback: config.EnableLegacyFallback,
			EnableCacheCleanup:   true,
			MaxFallbackAttempts:  3,
			LegacyBuilder:        config.LegacyBuilder,
		}
		fallbackManager = NewFallbackManager(fallbackConfig)

		// Register fallback callback if provided
		if config.FallbackCallback != nil {
			fallbackManager.RegisterCallback(config.FallbackCallback)
		}
	}

	// Create deprecation checker
	deprecationChecker := NewDeprecationChecker(config.EnableDeprecationWarnings)
	if config.DeprecationCallback != nil {
		deprecationChecker.RegisterCallback(config.DeprecationCallback)
	}

	return &IncrementalBuilder{
		buildCache:         buildCache,
		layoutDetector:     layoutDetector,
		changeDetector:     changeDetector,
		buildResultCache:   buildResultCache,
		dependencyAnalyzer: dependencyAnalyzer,
		buildPlanner:       buildPlanner,
		uvRunner:           uvRunner,
		dependencyCache:    dependencyCache,
		progressReporter:   progressReporter,
		fallbackManager:    fallbackManager,
		deprecationChecker: deprecationChecker,
		config:             config,
	}, nil
}

// Build performs an incremental build using selective package building and caching
func (ib *IncrementalBuilder) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Try the main build process first
	result, err := ib.buildWithFallback(ctx, input)
	return result, err
}

// buildWithFallback performs the build with fallback handling
func (ib *IncrementalBuilder) buildWithFallback(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Try the main build process
	result, err := ib.buildInternal(ctx, input)

	// If build succeeded or fallbacks are disabled, return the result
	if err == nil || ib.fallbackManager == nil {
		return result, err
	}

	// Check if we should fallback
	shouldFallback, reason, strategy := ib.fallbackManager.ShouldFallback(err)
	if !shouldFallback {
		return result, err
	}

	slog.Warn("build failed, attempting fallback",
		"reason", string(reason),
		"strategy", string(strategy),
		"originalError", err.Error())

	// Execute fallback strategy
	fallbackResult, fallbackErr := ib.fallbackManager.ExecuteFallback(ctx, input, reason, strategy, err)
	if fallbackErr != nil {
		// Fallback also failed, return original error with context
		return nil, fmt.Errorf("build failed and fallback also failed: original error: %w, fallback error: %v", err, fallbackErr)
	}

	slog.Info("fallback succeeded",
		"reason", string(reason),
		"strategy", string(strategy))

	return fallbackResult, nil
}

// buildInternal performs the actual incremental build logic
func (ib *IncrementalBuilder) buildInternal(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	ib.mutex.Lock()
	defer ib.mutex.Unlock()

	startTime := time.Now()

	// Start progress reporting
	ib.progressReporter.StartStage(StageInit, "Starting incremental build")

	slog.Info("starting incremental build",
		"functionID", input.FunctionID,
		"handler", input.Handler)

	// Update layout detector with current project root
	workingDir := filepath.Dir(input.CfgPath)
	if workingDir == "" {
		workingDir = "."
	}
	ib.layoutDetector.projectRoot = workingDir

	// Check if we can use cached results
	if ib.config.EnableBuildOptimization {
		ib.progressReporter.UpdateProgress(10, "Checking for cached build results", map[string]interface{}{
			"functionID": input.FunctionID,
		})

		if cachedResult, err := ib.tryUseCachedResult(ctx, input); err == nil && cachedResult != nil {
			slog.Info("using cached build result",
				"functionID", input.FunctionID)

			// Mark all stages as cached and complete the build
			ib.progressReporter.MarkCached(StageLayoutDetect, "Using cached layout detection")
			ib.progressReporter.MarkCached(StageDependencies, "Using cached dependencies")
			ib.progressReporter.MarkCached(StageBuildPlan, "Using cached build plan")
			ib.progressReporter.MarkCached(StageBuildPackages, "Using cached build artifacts")
			ib.progressReporter.MarkCached(StagePostProcess, "Using cached post-processing")
			ib.progressReporter.Complete("Build completed using cached results", map[string]interface{}{
				"functionID": input.FunctionID,
				"duration":   time.Since(startTime).String(),
				"cached":     true,
			})

			return cachedResult, nil
		}
	}

	// Detect project layout with error recovery
	ib.progressReporter.StartStage(StageLayoutDetect, "Detecting project layout")

	layout, err := ib.layoutDetector.DetectLayout(input.Handler)
	if err != nil {
		// Try to recover from layout detection errors
		if pythonErr, ok := err.(*PythonRuntimeError); ok {
			if pythonErr.Type == ErrorTypeLayoutDetection {
				// Attempt recovery by using fallback layout detection
				slog.Warn("layout detection failed, attempting fallback", "error", err)
				// For now, return the error - fallback could be implemented later
			}
		}

		// Report failure in progress
		ib.progressReporter.FailStage(StageLayoutDetect, "Failed to detect project layout", err)
		ib.progressReporter.Fail("Build failed during layout detection", err)

		return nil, WrapError(err, "layout detection").
			WithContext("handler", input.Handler).
			WithContext("workingDir", workingDir)
	}

	ib.progressReporter.CompleteStage(StageLayoutDetect, "Project layout detected successfully")

	// Check for deprecated layout patterns
	if ib.deprecationChecker != nil {
		ib.deprecationChecker.CheckLayout(layout)
	}

	slog.Debug("detected project layout",
		"type", layout.Type,
		"workspaceDir", layout.WorkspaceDir,
		"packageName", layout.PackageName)

	// Analyze dependencies with error recovery
	ib.progressReporter.StartStage(StageDependencies, "Analyzing project dependencies")

	dependencies, err := ib.dependencyAnalyzer.AnalyzeDependencies(ctx, layout)
	if err != nil {
		// Report failure in progress
		ib.progressReporter.FailStage(StageDependencies, "Failed to analyze dependencies", err)
		ib.progressReporter.Fail("Build failed during dependency analysis", err)

		return nil, WrapError(err, "dependency analysis").
			WithContext("layout", layout.Type).
			WithContext("workspaceDir", layout.WorkspaceDir).
			WithSuggestion("Check if pyproject.toml and uv.lock files are valid")
	}

	ib.progressReporter.CompleteStage(StageDependencies, "Dependencies analyzed successfully")

	// Check for deprecated dependency patterns
	if ib.deprecationChecker != nil {
		ib.deprecationChecker.CheckDependencies(dependencies)
	}

	// Create build plan with error recovery
	ib.progressReporter.StartStage(StageBuildPlan, "Creating build plan")

	buildPlan, err := ib.buildPlanner.CreateBuildPlan(ctx, input, layout, dependencies)
	if err != nil {
		// Report failure in progress
		ib.progressReporter.FailStage(StageBuildPlan, "Failed to create build plan", err)
		ib.progressReporter.Fail("Build failed during build planning", err)

		return nil, WrapError(err, "build planning").
			WithContext("packageCount", len(dependencies.LocalPackages)).
			WithSuggestion("Check for circular dependencies in your project")
	}

	ib.progressReporter.CompleteStage(StageBuildPlan, "Build plan created successfully")

	slog.Info("created build plan",
		"packagesToBuild", len(buildPlan.Packages),
		"cacheHits", len(buildPlan.CacheHits),
		"estimatedDuration", buildPlan.EstimatedDuration)

	// Execute build plan with error recovery
	ib.progressReporter.StartStage(StageBuildPackages, "Building packages")

	buildResult, err := ib.executeBuildPlan(ctx, input, layout, buildPlan)
	if err != nil {
		// Try to recover from build failures
		if pythonErr, ok := err.(*PythonRuntimeError); ok {
			if pythonErr.IsRetryable() {
				slog.Info("build failed with retryable error, attempting recovery",
					"error", pythonErr.Message,
					"retryAfter", pythonErr.GetRetryAfter())

				// Implement retry logic here if needed
				// For now, we'll just return the enhanced error
			}
		}

		// Report failure in progress
		ib.progressReporter.FailStage(StageBuildPackages, "Failed to build packages", err)
		ib.progressReporter.Fail("Build failed during package building", err)

		return nil, WrapError(err, "build execution").
			WithContext("packagesPlanned", len(buildPlan.Packages)).
			WithContext("estimatedDuration", buildPlan.EstimatedDuration.String()).
			WithSuggestion("Check build logs for specific package failures").
			WithRecoveryAction(RecoveryAction{
				Name:        "clear_build_cache",
				Description: "Clear build cache and retry",
				Automatic:   false,
			})
	}

	ib.progressReporter.CompleteStage(StageBuildPackages, "Packages built successfully")

	// Update cache with build results
	ib.progressReporter.StartStage(StagePostProcess, "Post-processing build results")

	if err := ib.updateCacheAfterBuild(input, layout, buildResult); err != nil {
		slog.Warn("failed to update cache after build", "error", err)
	}

	ib.progressReporter.CompleteStage(StagePostProcess, "Post-processing completed")

	buildDuration := time.Since(startTime)

	// Complete the build with progress reporting
	ib.progressReporter.Complete("Build completed successfully", map[string]interface{}{
		"functionID":     input.FunctionID,
		"duration":       buildDuration.String(),
		"packagesBuilt":  len(buildResult.PackagesBuilt),
		"packagesCached": len(buildResult.PackagesCached),
		"cached":         false,
	})

	slog.Info("completed incremental build",
		"functionID", input.FunctionID,
		"buildDuration", buildDuration,
		"packagesBuilt", len(buildResult.PackagesBuilt),
		"packagesCached", len(buildResult.PackagesCached))

	return buildResult.BuildOutput, nil
}

// ShouldRebuild determines if a function needs to be rebuilt
func (ib *IncrementalBuilder) ShouldRebuild(functionID string, handler string) bool {
	ib.mutex.RLock()
	defer ib.mutex.RUnlock()

	// Use change detection to determine if rebuild is needed
	result, err := ib.changeDetector.DetectChanges(functionID, handler)
	if err != nil {
		slog.Warn("failed to detect changes, rebuilding",
			"functionID", functionID,
			"error", err)
		return true
	}

	if result.HasChanges {
		slog.Info("changes detected, rebuilding",
			"functionID", functionID,
			"reason", result.Reason,
			"changeTypes", result.ChangeTypes,
			"changedFiles", len(result.ChangedFiles))
	} else {
		slog.Debug("no changes detected, using cached build",
			"functionID", functionID)
	}

	return result.HasChanges
}

// GetProgressReporter returns the progress reporter for external access
func (ib *IncrementalBuilder) GetProgressReporter() *ProgressReporter {
	return ib.progressReporter
}

// GetBuildProgress returns the current build progress
func (ib *IncrementalBuilder) GetBuildProgress() map[string]interface{} {
	if ib.progressReporter == nil {
		return map[string]interface{}{
			"progress": 0,
			"stage":    "not_started",
		}
	}
	return ib.progressReporter.GetProgressSummary()
}

// GetFallbackManager returns the fallback manager for external access
func (ib *IncrementalBuilder) GetFallbackManager() *FallbackManager {
	return ib.fallbackManager
}

// GetFallbackHistory returns the recent fallback events
func (ib *IncrementalBuilder) GetFallbackHistory() []FallbackEvent {
	if ib.fallbackManager == nil {
		return []FallbackEvent{}
	}
	return ib.fallbackManager.GetFallbackHistory()
}

// ClearFallbackHistory clears the fallback history
func (ib *IncrementalBuilder) ClearFallbackHistory() {
	if ib.fallbackManager != nil {
		ib.fallbackManager.ClearFallbackHistory()
	}
}

// tryUseCachedResult attempts to use a cached build result
func (ib *IncrementalBuilder) tryUseCachedResult(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Check if we have a cached result
	if !ib.buildResultCache.HasCachedResult(input.FunctionID) {
		return nil, fmt.Errorf("no cached result available")
	}

	// Check if the cached result is still valid
	if !ib.ShouldRebuild(input.FunctionID, input.Handler) {
		// Restore cached build result
		cachedResult, err := ib.buildResultCache.RestoreBuildResult(input.FunctionID, input.Out())
		if err != nil {
			return nil, fmt.Errorf("failed to restore cached result: %w", err)
		}

		return &runtime.BuildOutput{
			Handler:    cachedResult.Handler,
			Errors:     cachedResult.Errors,
			Sourcemaps: cachedResult.Sourcemaps,
		}, nil
	}

	return nil, fmt.Errorf("cached result is outdated")
}

// executeBuildPlan executes the build plan and builds the necessary packages
func (ib *IncrementalBuilder) executeBuildPlan(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, plan *BuildPlan) (*BuildResult, error) {
	result := &BuildResult{
		Success:        true,
		BuildPlan:      plan,
		PackagesBuilt:  []string{},
		PackagesCached: []string{},
		Errors:         []string{},
		Warnings:       []string{},
		BuildDuration:  0,
	}

	startTime := time.Now()

	// Use cached results where possible
	for _, cacheHit := range plan.CacheHits {
		result.PackagesCached = append(result.PackagesCached, cacheHit)
		slog.Debug("using cached result for package", "package", cacheHit)
	}

	// Build packages that need rebuilding
	if len(plan.Packages) > 0 {
		if ib.config.EnableParallelBuilds && len(plan.ParallelGroups) > 1 {
			// Execute parallel builds
			if err := ib.executeParallelBuilds(ctx, input, layout, plan, result); err != nil {
				result.Success = false
				result.Errors = append(result.Errors, err.Error())
				return result, err
			}
		} else {
			// Execute sequential builds
			if err := ib.executeSequentialBuilds(ctx, input, layout, plan, result); err != nil {
				result.Success = false
				result.Errors = append(result.Errors, err.Error())
				return result, err
			}
		}
	}

	// Install dependencies with caching if needed
	if !input.IsContainer || input.Dev {
		if err := ib.installDependenciesForBuild(ctx, input, layout, plan, result); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
			return result, err
		}
	}

	// Create final build output
	buildOutput, err := ib.createFinalBuildOutput(ctx, input, layout, result)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	result.BuildOutput = buildOutput
	result.BuildDuration = time.Since(startTime)

	return result, nil
}

// executeSequentialBuilds builds packages one by one in the specified order
func (ib *IncrementalBuilder) executeSequentialBuilds(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, plan *BuildPlan, result *BuildResult) error {
	totalPackages := 0
	for _, pkg := range plan.Packages {
		if pkg.RequiresRebuild {
			totalPackages++
		}
	}

	builtPackages := 0
	for _, packageName := range plan.BuildOrder {
		// Find package info
		var packageInfo *PackageBuildInfo
		for _, pkg := range plan.Packages {
			if pkg.PackageName == packageName {
				packageInfo = pkg
				break
			}
		}

		if packageInfo == nil {
			continue // Skip packages not in our build list
		}

		if !packageInfo.RequiresRebuild {
			continue // Skip packages that don't need rebuilding
		}

		// Update progress for this package
		progress := (builtPackages * 100) / totalPackages
		ib.progressReporter.UpdateProgress(progress, fmt.Sprintf("Building package %s (%d/%d)", packageName, builtPackages+1, totalPackages), map[string]interface{}{
			"package":  packageName,
			"reason":   packageInfo.RebuildReason,
			"progress": fmt.Sprintf("%d/%d", builtPackages+1, totalPackages),
		})

		slog.Info("building package",
			"package", packageName,
			"reason", packageInfo.RebuildReason)

		if err := ib.buildPackage(ctx, input, layout, packageInfo); err != nil {
			return fmt.Errorf("failed to build package %s: %w", packageName, err)
		}

		result.PackagesBuilt = append(result.PackagesBuilt, packageName)
		builtPackages++
	}

	return nil
}

// executeParallelBuilds builds packages in parallel where possible
func (ib *IncrementalBuilder) executeParallelBuilds(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, plan *BuildPlan, result *BuildResult) error {
	for _, group := range plan.ParallelGroups {
		// Build packages in this group in parallel
		if err := ib.buildPackageGroup(ctx, input, layout, plan, group, result); err != nil {
			return fmt.Errorf("failed to build package group: %w", err)
		}
	}

	return nil
}

// buildPackageGroup builds a group of packages in parallel
func (ib *IncrementalBuilder) buildPackageGroup(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, plan *BuildPlan, group []string, result *BuildResult) error {
	// Create a channel for errors
	errChan := make(chan error, len(group))
	var wg sync.WaitGroup

	// Build each package in the group concurrently
	for _, packageName := range group {
		// Find package info
		var packageInfo *PackageBuildInfo
		for _, pkg := range plan.Packages {
			if pkg.PackageName == packageName {
				packageInfo = pkg
				break
			}
		}

		if packageInfo == nil || !packageInfo.RequiresRebuild {
			continue
		}

		wg.Add(1)
		go func(pkgName string, pkgInfo *PackageBuildInfo) {
			defer wg.Done()

			slog.Info("building package (parallel)",
				"package", pkgName,
				"reason", pkgInfo.RebuildReason)

			if err := ib.buildPackage(ctx, input, layout, pkgInfo); err != nil {
				errChan <- fmt.Errorf("failed to build package %s: %w", pkgName, err)
				return
			}

			// Thread-safe append to result
			ib.mutex.Lock()
			result.PackagesBuilt = append(result.PackagesBuilt, pkgName)
			ib.mutex.Unlock()
		}(packageName, packageInfo)
	}

	// Wait for all builds to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// buildPackage builds a single package with selective building logic
func (ib *IncrementalBuilder) buildPackage(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, packageInfo *PackageBuildInfo) error {
	slog.Info("building package selectively",
		"package", packageInfo.PackageName,
		"packageDir", packageInfo.PackageDir,
		"reason", packageInfo.RebuildReason)

	// Check if we can reuse cached build artifacts
	if !packageInfo.RequiresRebuild && packageInfo.CanUseCache {
		if cacheErr := ib.reuseCachedPackageBuild(ctx, input, packageInfo); cacheErr == nil {
			slog.Info("reused cached build for package", "package", packageInfo.PackageName)
			return nil
		} else {
			slog.Warn("failed to reuse cached build, building from scratch",
				"package", packageInfo.PackageName,
				"error", cacheErr)
		}
	}

	// Build the package using UV
	buildCmd := &UvBuildCommand{
		PackageName:  packageInfo.PackageName,
		PackageDir:   packageInfo.PackageDir,
		OutputDir:    input.Out(),
		SourceFiles:  packageInfo.SourceFiles,
		Dependencies: packageInfo.Dependencies,
		BuildType:    "sdist",  // Default to source distribution
		Architecture: "x86_64", // Default architecture
	}

	// Execute the build command with error recovery
	if err := ib.uvRunner.ExecuteBuildCommand(ctx, buildCmd); err != nil {
		buildErr := NewBuildFailedError(packageInfo.PackageName, err).
			WithContext("packageDir", packageInfo.PackageDir).
			WithContext("buildType", buildCmd.BuildType).
			WithContext("sourceFiles", len(packageInfo.SourceFiles)).
			WithSuggestion("Check if all source files are accessible").
			WithSuggestion("Verify pyproject.toml configuration for this package")

		// Add specific suggestions based on error type
		if strings.Contains(err.Error(), "permission") {
			buildErr.WithSuggestion("Check file permissions in the package directory")
		}
		if strings.Contains(err.Error(), "space") {
			buildErr.WithSuggestion("Check available disk space")
		}

		return buildErr
	}

	// Post-process the built package with error recovery
	if err := ib.postProcessPackageBuild(ctx, input, packageInfo); err != nil {
		return WrapError(err, "package post-processing").
			WithContext("package", packageInfo.PackageName).
			WithContext("outputDir", input.Out()).
			WithSuggestion("Check if the build output directory is writable")
	}

	// Cache the build result for future use
	if err := ib.cachePackageBuildResult(ctx, input, packageInfo); err != nil {
		slog.Warn("failed to cache build result",
			"package", packageInfo.PackageName,
			"error", err)
	}

	slog.Info("successfully built package",
		"package", packageInfo.PackageName,
		"outputDir", input.Out())

	return nil
}

// createFinalBuildOutput creates the final build output from the build results
func (ib *IncrementalBuilder) createFinalBuildOutput(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, result *BuildResult) (*runtime.BuildOutput, error) {
	// Adjust handler path based on layout
	adjustedHandler, err := ib.adjustHandlerPath(input, layout)
	if err != nil {
		return nil, fmt.Errorf("failed to adjust handler path: %w", err)
	}

	// Create cached build output for future use
	cachedBuildOutput := &CachedBuildOutput{
		Handler:       adjustedHandler,
		OutputDir:     input.Out(),
		Errors:        result.Errors,
		Sourcemaps:    []string{}, // TODO: collect sourcemaps
		ArtifactPaths: []string{}, // TODO: collect artifact paths
		BuildDuration: result.BuildDuration,
	}

	result.CachedBuildOutput = cachedBuildOutput

	return &runtime.BuildOutput{
		Handler:    adjustedHandler,
		Errors:     result.Errors,
		Sourcemaps: []string{}, // TODO: collect sourcemaps
	}, nil
}

// adjustHandlerPath adjusts the handler path based on the project layout
func (ib *IncrementalBuilder) adjustHandlerPath(input *runtime.BuildInput, layout *LayoutInfo) (string, error) {
	handlerParts := strings.Split(input.Handler, "/")
	adjustedHandler := input.Handler

	if len(handlerParts) >= 3 {
		// Look for the pattern {package_name}/src/{package_name}
		for i := len(handlerParts) - 3; i >= 0; i-- {
			if i+2 >= len(handlerParts) {
				continue
			}

			pkgName := handlerParts[i]
			if handlerParts[i+1] == "src" && handlerParts[i+2] == pkgName {
				// Found the pattern, remove the middle two parts (src/{package_name})
				newParts := append(
					handlerParts[:i+1],
					handlerParts[i+3:]...,
				)
				adjustedHandler = strings.Join(newParts, "/")
				slog.Info("adjusted handler path",
					"original", input.Handler,
					"adjusted", adjustedHandler)
				break
			}
		}
	}

	return adjustedHandler, nil
}

// updateCacheAfterBuild updates the cache with build results
func (ib *IncrementalBuilder) updateCacheAfterBuild(input *runtime.BuildInput, layout *LayoutInfo, result *BuildResult) error {
	if result.CachedBuildOutput == nil {
		return fmt.Errorf("no cached build output to store")
	}

	// Update change detector cache
	return ib.changeDetector.UpdateCacheAfterBuild(
		input.FunctionID,
		input.Handler,
		layout,
		result.CachedBuildOutput,
	)
}

// reuseCachedPackageBuild attempts to reuse a cached package build
func (ib *IncrementalBuilder) reuseCachedPackageBuild(ctx context.Context, input *runtime.BuildInput, packageInfo *PackageBuildInfo) error {
	// Check if we have cached build artifacts for this package
	packageFunctionID := fmt.Sprintf("package:%s:%s", packageInfo.PackageName, packageInfo.PackageDir)

	if !ib.buildResultCache.HasCachedResult(packageFunctionID) {
		return fmt.Errorf("no cached build result available for package %s", packageInfo.PackageName)
	}

	// Restore cached build artifacts to the output directory
	cachedResult, err := ib.buildResultCache.RestoreBuildResult(packageFunctionID, input.Out())
	if err != nil {
		return fmt.Errorf("failed to restore cached build result: %w", err)
	}

	slog.Debug("restored cached build artifacts",
		"package", packageInfo.PackageName,
		"artifactPaths", len(cachedResult.ArtifactPaths))

	return nil
}

// postProcessPackageBuild performs post-processing on a built package
func (ib *IncrementalBuilder) postProcessPackageBuild(ctx context.Context, input *runtime.BuildInput, packageInfo *PackageBuildInfo) error {
	// Extract and process the built package archive
	if err := ib.extractAndProcessPackageArchive(input.Out(), packageInfo); err != nil {
		return fmt.Errorf("failed to extract and process package archive: %w", err)
	}

	// Handle src/{package_name} structure adjustment if needed
	if err := ib.adjustPackageStructure(input.Out(), packageInfo); err != nil {
		return fmt.Errorf("failed to adjust package structure: %w", err)
	}

	return nil
}

// extractAndProcessPackageArchive extracts the built package archive and processes it
func (ib *IncrementalBuilder) extractAndProcessPackageArchive(outputDir string, packageInfo *PackageBuildInfo) error {
	// Look for the built tar.gz file for this package
	pattern := filepath.Join(outputDir, packageInfo.PackageName+"-*.tar.gz")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find package archive: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no package archive found for %s", packageInfo.PackageName)
	}

	// Process each archive file (should typically be just one)
	for _, archiveFile := range files {
		if err := ib.processPackageArchive(archiveFile, outputDir, packageInfo); err != nil {
			return fmt.Errorf("failed to process archive %s: %w", archiveFile, err)
		}
	}

	return nil
}

// processPackageArchive processes a single package archive file
func (ib *IncrementalBuilder) processPackageArchive(archiveFile, outputDir string, packageInfo *PackageBuildInfo) error {
	// Extract the tar.gz file
	extractCmd := []string{"tar", "-xzf", archiveFile, "-C", outputDir}
	if err := ib.executeCommand(extractCmd, outputDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Get the directory name without version number
	archiveBaseName := filepath.Base(archiveFile)
	dirName := strings.TrimSuffix(archiveBaseName, ".tar.gz")
	lastHyphen := strings.LastIndex(dirName, "-")
	if lastHyphen == -1 {
		return fmt.Errorf("invalid archive name format: %s", archiveBaseName)
	}

	baseName := dirName[:lastHyphen]
	extractedDir := filepath.Join(outputDir, dirName)
	targetDir := filepath.Join(outputDir, baseName)

	// Move extracted directory to target location
	if err := ib.moveExtractedPackage(extractedDir, targetDir, baseName); err != nil {
		return fmt.Errorf("failed to move extracted package: %w", err)
	}

	// Remove the original archive file
	if err := os.Remove(archiveFile); err != nil {
		slog.Warn("failed to remove archive file", "file", archiveFile, "error", err)
	}

	return nil
}

// moveExtractedPackage moves the extracted package to the correct location
func (ib *IncrementalBuilder) moveExtractedPackage(extractedDir, targetDir, baseName string) error {
	// Check if the package has a src/{package_name} structure
	srcPath := filepath.Join(extractedDir, "src", baseName)
	if _, err := os.Stat(srcPath); err == nil {
		// Remove old directory if it exists
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove old directory: %w", err)
		}

		// Move the contents from src/{package_name} directly to the target
		if err := os.Rename(srcPath, targetDir); err != nil {
			return fmt.Errorf("failed to move src directory contents: %w", err)
		}

		// Clean up the original extracted directory
		if err := os.RemoveAll(extractedDir); err != nil {
			return fmt.Errorf("failed to clean up extracted directory: %w", err)
		}
	} else {
		// Handle the regular case (no src directory)
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove old directory: %w", err)
		}

		if err := os.Rename(extractedDir, targetDir); err != nil {
			return fmt.Errorf("failed to rename directory: %w", err)
		}
	}

	return nil
}

// adjustPackageStructure adjusts the package structure for Lambda compatibility
func (ib *IncrementalBuilder) adjustPackageStructure(outputDir string, packageInfo *PackageBuildInfo) error {
	// This method handles any additional structure adjustments needed for Lambda
	// Currently, the main adjustment is handled in moveExtractedPackage

	packageDir := filepath.Join(outputDir, packageInfo.PackageName)
	if _, err := os.Stat(packageDir); err != nil {
		// Package directory doesn't exist, which might be expected for some packages
		slog.Debug("package directory not found after extraction",
			"package", packageInfo.PackageName,
			"expectedDir", packageDir)
		return nil
	}

	slog.Debug("package structure adjusted",
		"package", packageInfo.PackageName,
		"packageDir", packageDir)

	return nil
}

// cachePackageBuildResult caches the build result for a package
func (ib *IncrementalBuilder) cachePackageBuildResult(ctx context.Context, input *runtime.BuildInput, packageInfo *PackageBuildInfo) error {
	packageFunctionID := fmt.Sprintf("package:%s:%s", packageInfo.PackageName, packageInfo.PackageDir)

	// Collect artifact paths for this package
	artifactPaths, err := ib.collectPackageArtifacts(input.Out(), packageInfo)
	if err != nil {
		return fmt.Errorf("failed to collect package artifacts: %w", err)
	}

	// Create cached build output
	cachedBuildOutput := &CachedBuildOutput{
		Handler:       packageInfo.PackageName, // Use package name as handler for package builds
		OutputDir:     input.Out(),
		Errors:        []string{},
		Sourcemaps:    []string{},
		ArtifactPaths: artifactPaths,
		BuildDuration: packageInfo.EstimatedBuildTime,
	}

	// Store the cached result
	return ib.buildResultCache.CacheBuildResult(packageFunctionID, cachedBuildOutput, artifactPaths)
}

// collectPackageArtifacts collects all artifact paths for a package
func (ib *IncrementalBuilder) collectPackageArtifacts(outputDir string, packageInfo *PackageBuildInfo) ([]string, error) {
	var artifactPaths []string

	// Add the main package directory
	packageDir := filepath.Join(outputDir, packageInfo.PackageName)
	if _, err := os.Stat(packageDir); err == nil {
		artifactPaths = append(artifactPaths, packageDir)
	}

	// Add any additional artifacts that might have been created
	// This could include compiled extensions, data files, etc.

	return artifactPaths, nil
}

// executeCommand executes a shell command
func (ib *IncrementalBuilder) executeCommand(args []string, workingDir string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmd := exec.Command(args[0], args[1:]...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s\nOutput: %s", err, string(output))
	}

	return nil
}

// GetBuildStats returns statistics about the incremental builder
func (ib *IncrementalBuilder) GetBuildStats() *IncrementalBuilderStats {
	ib.mutex.RLock()
	defer ib.mutex.RUnlock()

	cacheStats := ib.buildCache.GetStats()

	buildResultStats, err := ib.buildResultCache.GetStats()
	if err != nil {
		buildResultStats = &BuildResultCacheStats{}
	}

	dependencyCacheStats, err := ib.getDependencyCacheStats()
	if err != nil {
		dependencyCacheStats = nil
	}

	return &IncrementalBuilderStats{
		CacheStats:           cacheStats,
		BuildResultStats:     *buildResultStats,
		DependencyCacheStats: dependencyCacheStats,
		Config:               ib.config,
	}
}

// IncrementalBuilderStats contains statistics about the incremental builder
type IncrementalBuilderStats struct {
	CacheStats           CacheStats               `json:"cacheStats"`
	BuildResultStats     BuildResultCacheStats    `json:"buildResultStats"`
	DependencyCacheStats *DependencyCacheStats    `json:"dependencyCacheStats,omitempty"`
	Config               IncrementalBuilderConfig `json:"config"`
}

// ClearCache clears all caches
func (ib *IncrementalBuilder) ClearCache() error {
	ib.mutex.Lock()
	defer ib.mutex.Unlock()

	if err := ib.buildCache.Clear(); err != nil {
		return fmt.Errorf("failed to clear build cache: %w", err)
	}

	if err := ib.buildResultCache.CleanupExpiredResults(); err != nil {
		return fmt.Errorf("failed to cleanup build result cache: %w", err)
	}

	if err := ib.dependencyCache.ClearCache(); err != nil {
		return fmt.Errorf("failed to clear dependency cache: %w", err)
	}

	ib.layoutDetector.ClearCache()

	return nil
}

// ForceRebuild forces a rebuild for a specific function
func (ib *IncrementalBuilder) ForceRebuild(functionID string, reason string) error {
	ib.mutex.Lock()
	defer ib.mutex.Unlock()

	// Remove from caches
	if err := ib.buildCache.Delete(functionID); err != nil {
		return fmt.Errorf("failed to remove from build cache: %w", err)
	}

	if err := ib.buildResultCache.InvalidateCachedResult(functionID); err != nil {
		return fmt.Errorf("failed to invalidate cached result: %w", err)
	}

	slog.Info("forced rebuild requested",
		"functionID", functionID,
		"reason", reason)

	return nil
}

// RecoverFromError attempts to recover from various error conditions
func (ib *IncrementalBuilder) RecoverFromError(err error) error {
	if pythonErr, ok := err.(*PythonRuntimeError); ok {
		switch pythonErr.Type {
		case ErrorTypeCacheCorrupted:
			return ib.recoverFromCacheCorruption(pythonErr)
		case ErrorTypeBuildFailed:
			return ib.recoverFromBuildFailure(pythonErr)
		case ErrorTypeLayoutDetection:
			return ib.recoverFromLayoutDetectionFailure(pythonErr)
		}
	}
	return err
}

// recoverFromCacheCorruption handles cache corruption recovery
func (ib *IncrementalBuilder) recoverFromCacheCorruption(err *PythonRuntimeError) error {
	slog.Info("attempting to recover from cache corruption")

	// Clear all caches
	if clearErr := ib.ClearCache(); clearErr != nil {
		return WrapError(clearErr, "cache recovery").
			WithContext("originalError", err.Message).
			WithSuggestion("Manually delete the cache directory and retry")
	}

	slog.Info("cache corruption recovery completed")
	return nil
}

// recoverFromBuildFailure handles build failure recovery
func (ib *IncrementalBuilder) recoverFromBuildFailure(err *PythonRuntimeError) error {
	slog.Info("attempting to recover from build failure")

	// Check if it's a transient error that might be retryable
	if err.IsRetryable() {
		slog.Info("build failure is retryable", "retryAfter", err.GetRetryAfter())
		return err // Let the caller handle the retry
	}

	// For non-retryable build failures, suggest clearing cache
	packageName, ok := err.Context["package"].(string)
	if ok {
		slog.Info("clearing cache for failed package", "package", packageName)
		// Clear package-specific cache if possible
		// For now, we'll just log and return the error
	}

	return err
}

// recoverFromLayoutDetectionFailure handles layout detection failure recovery
func (ib *IncrementalBuilder) recoverFromLayoutDetectionFailure(err *PythonRuntimeError) error {
	slog.Info("attempting to recover from layout detection failure")

	// Clear layout cache and try again
	ib.layoutDetector.ClearCache()

	// For now, just return the error - more sophisticated fallback could be implemented
	return err
}

// BuildWithRecovery performs a build with automatic error recovery
func (ib *IncrementalBuilder) BuildWithRecovery(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Create error recovery manager
	recoveryManager := NewErrorRecoveryManager()

	var result *runtime.BuildOutput

	// Attempt build with retry and recovery
	err := recoveryManager.RetryWithBackoff(func() error {
		buildResult, buildErr := ib.Build(ctx, input)
		if buildErr != nil {
			// Attempt recovery
			if recoveryErr := ib.RecoverFromError(buildErr); recoveryErr == nil {
				// Recovery successful, retry the build
				slog.Info("error recovery successful, retrying build")
				buildResult, buildErr = ib.Build(ctx, input)
			}
		}
		result = buildResult
		return buildErr
	})

	return result, err
}

// installDependenciesWithCaching installs dependencies using caching when possible
func (ib *IncrementalBuilder) installDependenciesWithCaching(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, requirementsFile string, architecture string) error {
	slog.Info("installing dependencies with caching",
		"requirementsFile", requirementsFile,
		"architecture", architecture,
		"outputDir", input.Out())

	// Try to use cached dependencies first
	cachedEntry, err := ib.dependencyCache.GetCachedDependencies(requirementsFile, architecture, input.Out())
	if err == nil {
		slog.Info("using cached dependencies",
			"dependencyCount", len(cachedEntry.DependencyList),
			"cacheAge", time.Since(cachedEntry.CreatedAt))
		return nil
	}

	slog.Info("cached dependencies not available, installing fresh",
		"reason", err.Error())

	// Install dependencies fresh
	if err := ib.installDependenciesFresh(ctx, input, requirementsFile, architecture); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// Cache the installed dependencies for future use
	if err := ib.cacheFreshDependencies(requirementsFile, architecture, input.Out()); err != nil {
		slog.Warn("failed to cache dependencies", "error", err)
	}

	return nil
}

// installDependenciesFresh installs dependencies without using cache
func (ib *IncrementalBuilder) installDependenciesFresh(ctx context.Context, input *runtime.BuildInput, requirementsFile string, architecture string) error {
	// Determine platform for installation
	pythonPlatform := "x86_64-unknown-linux-gnu"
	if architecture == "arm64" {
		pythonPlatform = "aarch64-unknown-linux-gnu"
	}

	// Create UV install command
	installCmd := &UvInstallCommand{
		WorkingDir:       input.Out(),
		RequirementsFile: requirementsFile,
		TargetDir:        input.Out(),
		PythonPlatform:   pythonPlatform,
		Architecture:     architecture,
	}

	// Execute install command
	return ib.uvRunner.ExecuteInstallCommand(ctx, installCmd)
}

// cacheFreshDependencies caches freshly installed dependencies
func (ib *IncrementalBuilder) cacheFreshDependencies(requirementsFile string, architecture string, installPath string) error {
	// Parse requirements file to get dependency list
	dependencyList, err := ib.parseDependencyList(requirementsFile)
	if err != nil {
		return fmt.Errorf("failed to parse dependency list: %w", err)
	}

	// Cache the dependencies
	return ib.dependencyCache.CacheDependencies(requirementsFile, architecture, installPath, dependencyList)
}

// parseDependencyList parses a requirements file to extract dependency names
func (ib *IncrementalBuilder) parseDependencyList(requirementsFile string) ([]string, error) {
	content, err := os.ReadFile(requirementsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read requirements file: %w", err)
	}

	var dependencies []string
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Extract package name (before version specifiers)
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '=' || r == '<' || r == '>' || r == '!' || r == '~' || r == ' '
		})

		if len(parts) > 0 {
			packageName := strings.TrimSpace(parts[0])
			if packageName != "" {
				dependencies = append(dependencies, packageName)
			}
		}
	}

	return dependencies, nil
}

// shouldUseDependencyCache determines if dependency caching should be used
func (ib *IncrementalBuilder) shouldUseDependencyCache(input *runtime.BuildInput) bool {
	// Use dependency cache for non-dev builds and when optimization is enabled
	return !input.Dev && ib.config.EnableBuildOptimization
}

// getDependencyCacheStats returns statistics about the dependency cache
func (ib *IncrementalBuilder) getDependencyCacheStats() (*DependencyCacheStats, error) {
	return ib.dependencyCache.GetStats()
}

// installDependenciesForBuild installs dependencies for the build
func (ib *IncrementalBuilder) installDependenciesForBuild(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, plan *BuildPlan, result *BuildResult) error {
	// Generate requirements file
	requirementsFile := filepath.Join(input.Out(), "requirements.txt")
	if err := ib.generateRequirementsFile(ctx, input, layout, requirementsFile); err != nil {
		return fmt.Errorf("failed to generate requirements file: %w", err)
	}

	// Determine architecture
	architecture := "x86_64"
	if props, err := ib.parseInputProperties(input); err == nil && props.Architecture != "" {
		architecture = props.Architecture
	}

	// Install dependencies with caching
	if ib.shouldUseDependencyCache(input) {
		return ib.installDependenciesWithCaching(ctx, input, layout, requirementsFile, architecture)
	} else {
		return ib.installDependenciesFresh(ctx, input, requirementsFile, architecture)
	}
}

// generateRequirementsFile generates a requirements.txt file for the build
func (ib *IncrementalBuilder) generateRequirementsFile(ctx context.Context, input *runtime.BuildInput, layout *LayoutInfo, outputFile string) error {
	// Use UV export to generate requirements file
	exportCmd := &UvExportCommand{
		WorkspaceDir:    layout.WorkspaceDir,
		PackageName:     layout.PackageName,
		OutputFile:      outputFile,
		NoEmitWorkspace: true,
		NoDev:           true,
	}

	return ib.uvRunner.ExecuteExportCommand(ctx, exportCmd)
}

// InputProperties represents the input properties structure
type InputProperties struct {
	Architecture string `json:"architecture"`
	Container    bool   `json:"container"`
}

// parseInputProperties parses the input properties JSON
func (ib *IncrementalBuilder) parseInputProperties(input *runtime.BuildInput) (*InputProperties, error) {
	var props InputProperties
	if err := json.Unmarshal(input.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	return &props, nil
}
