package python

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/runtime"
)

// Build stage constants
const (
	StageInit           = "init"
	StageProjectResolve = "project_resolve"
	StageDependencies   = "dependencies"
	StageBuildPackages  = "build_packages"
	StageBuildPlan      = "build_plan"
	StagePostProcess    = "post_process"
)

// ProgressEvent represents a build progress event
type ProgressEvent struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

// ProgressCallback is a function type for progress callbacks
type ProgressCallback func(ProgressEvent)

// ProgressReporter tracks and reports build progress
type ProgressReporter struct {
	callbacks []ProgressCallback
	progress  int
	status    string
}

// NewProgressReporter creates a new progress reporter
func NewProgressReporter() *ProgressReporter {
	return &ProgressReporter{
		callbacks: make([]ProgressCallback, 0),
		progress:  0,
		status:    "not_started",
	}
}

// RegisterCallback adds a progress callback
func (pr *ProgressReporter) RegisterCallback(callback ProgressCallback) {
	if pr == nil || callback == nil {
		return
	}
	pr.callbacks = append(pr.callbacks, callback)
}

// Report sends a progress event to all callbacks
func (pr *ProgressReporter) Report(stage, message string) {
	if pr == nil || pr.callbacks == nil {
		return
	}
	event := ProgressEvent{Stage: stage, Message: message}
	for _, callback := range pr.callbacks {
		if callback != nil {
			callback(event)
		}
	}
}

// StartStage starts a new build stage
func (pr *ProgressReporter) StartStage(stage, message string) {
	pr.Report(stage, message)
}

// UpdateProgress updates progress for current stage
func (pr *ProgressReporter) UpdateProgress(stage, message string) {
	pr.Report(stage, message)
}

// MarkCached marks a stage as using cached results
func (pr *ProgressReporter) MarkCached(stage, message string) {
	pr.Report(stage, message)
}

// GetProgressSummary returns a summary of progress
func (pr *ProgressReporter) GetProgressSummary() map[string]interface{} {
	if pr == nil {
		return map[string]interface{}{
			"status":   "not_started",
			"progress": 0,
		}
	}
	return map[string]interface{}{
		"status":   pr.status,
		"progress": pr.progress,
	}
}

// CompleteStage completes a build stage
func (pr *ProgressReporter) CompleteStage(stage, message string) {
	pr.Report(stage, message)
}

// FailStage marks a stage as failed
func (pr *ProgressReporter) FailStage(stage, message string, err error) {
	pr.Report(stage, message)
}

// Complete marks the entire build as complete
func (pr *ProgressReporter) Complete(message string, metadata map[string]interface{}) {
	if pr != nil {
		pr.progress = 100
		pr.status = "complete"
	}
	pr.Report("complete", message)
}

// Fail marks the entire build as failed
func (pr *ProgressReporter) Fail(message string, err error) {
	if pr != nil {
		pr.status = "failed"
	}
	pr.Report("failed", message)
}

// IncrementalBuilder coordinates selective builds and optimizes the build process
type IncrementalBuilder struct {
	// buildCache manages build state and change detection
	buildCache *BuildCache

	// projectResolver provides simplified project resolution without layout classification
	projectResolver *ProjectResolver

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

	// contentFilter handles file exclusion using hybrid approach
	contentFilter *ContentFilter

	// Removed fallback manager - using direct error handling instead

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

	// MaxCacheAge is the maximum age for dependency cache entries (not build cache)
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

	// ProjectRoot is the root directory of the project for ContentFilter
	ProjectRoot string

	// FunctionID is the ID of the function being built (for progress events)
	FunctionID string

	// Fallback mechanisms removed - using direct error handling instead

	// EnableDeprecationWarnings enables deprecation warnings
	EnableDeprecationWarnings bool
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
	// Add safety checks for required configuration
	if config.CacheDir == "" {
		return nil, fmt.Errorf("cache directory is required")
	}
	if config.ArtifactDir == "" {
		return nil, fmt.Errorf("artifact directory is required")
	}

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

	// Initialize project resolver
	projectResolver := NewProjectResolver(config.CacheDir) // Will be updated per build

	// Initialize build cache with sensible defaults
	buildCache, err := NewDefaultBuildCache(config.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create build cache: %w", err)
	}

	// Initialize change detector
	changeDetector, err := NewChangeDetector(ChangeDetectorConfig{
		ProjectResolver: projectResolver,
		BuildCache:      buildCache,
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
		ProjectResolver: projectResolver,
		BuildCache:      buildCache,
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

	// Create progress reporter
	progressReporter := NewProgressReporter()

	// Register progress callback if provided
	if config.ProgressCallback != nil {
		progressReporter.RegisterCallback(config.ProgressCallback)
	}

	// Fallback manager removed - using direct error handling instead

	// Create content filter for the project
	contentFilter := NewContentFilterForProject(config.ProjectRoot)

	return &IncrementalBuilder{
		buildCache:         buildCache,
		projectResolver:    projectResolver,
		changeDetector:     changeDetector,
		buildResultCache:   buildResultCache,
		dependencyAnalyzer: dependencyAnalyzer,
		buildPlanner:       buildPlanner,
		uvRunner:           uvRunner,
		dependencyCache:    dependencyCache,
		progressReporter:   progressReporter,
		contentFilter:      contentFilter,
		// fallbackManager removed
		config: config,
	}, nil
}

// Build performs an incremental build using selective package building and caching
func (ib *IncrementalBuilder) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Try the main build process with improved error handling
	result, err := ib.buildWithErrorHandling(ctx, input)
	return result, err
}

// buildWithErrorHandling performs the build with improved error handling
func (ib *IncrementalBuilder) buildWithErrorHandling(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Try the main build process
	result, err := ib.buildInternal(ctx, input)

	// Return the result directly - no fallback strategies needed
	// With simplified architecture, errors should be clear and actionable
	if err != nil {
		// Convert to structured error if needed
		if pythonErr, ok := err.(*PythonRuntimeError); ok {
			return result, pythonErr
		}

		// Wrap generic errors with better context
		wrappedErr := WrapError(err, fmt.Sprintf("building function %s", input.FunctionID))
		return result, wrappedErr
	}

	return result, nil
}

// buildInternal performs the actual incremental build logic
func (ib *IncrementalBuilder) buildInternal(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Note: Removed mutex lock here to avoid deadlock with parallel goroutines
	// The goroutines use their own mutex for thread-safe result updates

	startTime := time.Now()

	// Start progress reporting
	ib.progressReporter.StartStage(StageInit, "Starting incremental build")

	slog.Info("starting incremental build",
		"functionID", input.FunctionID,
		"handler", input.Handler)

	// Update project resolver with current project root
	workingDir := filepath.Dir(input.CfgPath)
	if workingDir == "" {
		workingDir = "."
	}
	ib.projectResolver.projectRoot = workingDir

	// Check if we can use cached results
	if ib.config.EnableBuildOptimization {
		ib.progressReporter.UpdateProgress(StageInit, "Checking for cached build results")

		if cachedResult, err := ib.tryUseCachedResult(ctx, input); err == nil && cachedResult != nil {
			slog.Info("using cached build result",
				"functionID", input.FunctionID)

			// Mark all stages as cached and complete the build
			ib.progressReporter.MarkCached(StageProjectResolve, "Using cached project resolution")
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

	// Resolve project structure with error recovery
	ib.progressReporter.StartStage(StageProjectResolve, "Resolving project structure")

	projectInfo, err := ib.projectResolver.ResolveHandler(input.Handler)
	if err != nil {
		// Try to recover from project structure errors
		if pythonErr, ok := err.(*PythonRuntimeError); ok {
			if pythonErr.Type == ErrorTypeProjectStructure {
				// Attempt recovery by using fallback project resolution
				slog.Warn("project structure resolution failed, attempting fallback", "error", err)
				// For now, return the error - fallback could be implemented later
			}
		}

		// Report failure in progress
		ib.progressReporter.FailStage(StageProjectResolve, "Failed to resolve project structure", err)
		ib.progressReporter.Fail("Build failed during project resolution", err)

		return nil, WrapError(err, "project resolution").
			WithContext("handler", input.Handler).
			WithContext("workingDir", workingDir)
	}

	ib.progressReporter.CompleteStage(StageProjectResolve, "Project structure resolved successfully")

	slog.Debug("resolved project structure",
		"handlerFile", projectInfo.HandlerFile,
		"sourceRoot", projectInfo.SourceRoot,
		"modulePath", projectInfo.ModulePath)

	// Analyze dependencies with error recovery
	ib.progressReporter.StartStage(StageDependencies, "Analyzing project dependencies")

	dependencies, err := ib.dependencyAnalyzer.AnalyzeDependencies(ctx, projectInfo)
	if err != nil {
		// Report failure in progress
		ib.progressReporter.FailStage(StageDependencies, "Failed to analyze dependencies", err)
		ib.progressReporter.Fail("Build failed during dependency analysis", err)

		return nil, WrapError(err, "dependency analysis").
			WithContext("sourceRoot", projectInfo.SourceRoot).
			WithContext("pyprojectPath", projectInfo.PyprojectPath).
			WithSuggestion("Check if pyproject.toml and uv.lock files are valid")
	}

	ib.progressReporter.CompleteStage(StageDependencies, "Dependencies analyzed successfully")

	// Create build plan with error recovery
	ib.progressReporter.StartStage(StageBuildPlan, "Creating build plan")

	buildPlan, err := ib.buildPlanner.CreateBuildPlan(ctx, input, projectInfo, dependencies)
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

	buildResult, err := ib.executeBuildPlan(ctx, input, projectInfo, buildPlan)
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

	if err := ib.updateCacheAfterBuild(input, projectInfo, buildResult); err != nil {
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

// SetProjectRoot updates the project root directory for project resolution
func (ib *IncrementalBuilder) SetProjectRoot(projectRoot string) {
	ib.mutex.Lock()
	defer ib.mutex.Unlock()

	if ib.projectResolver != nil {
		ib.projectResolver.projectRoot = projectRoot
	}
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

// Fallback manager methods removed - using direct error handling instead

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
			Out:        cachedResult.OutputDir,
			Handler:    cachedResult.Handler,
			Errors:     cachedResult.Errors,
			Sourcemaps: cachedResult.Sourcemaps,
		}, nil
	}

	return nil, fmt.Errorf("cached result is outdated")
}

// executeBuildPlan executes the build plan and builds the necessary packages
func (ib *IncrementalBuilder) executeBuildPlan(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, plan *BuildPlan) (*BuildResult, error) {
	// Add nil checks to prevent segfaults
	if ib == nil {
		return nil, fmt.Errorf("incremental builder is nil")
	}
	if projectInfo == nil {
		return nil, fmt.Errorf("project info is nil")
	}
	if plan == nil {
		return nil, fmt.Errorf("build plan is nil")
	}
	if input == nil {
		return nil, fmt.Errorf("build input is nil")
	}

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

	// Use cached results where possible (with nil check)
	if plan.CacheHits != nil {
		for _, cacheHit := range plan.CacheHits {
			result.PackagesCached = append(result.PackagesCached, cacheHit)
			slog.Debug("using cached result for package", "package", cacheHit)
		}
	}

	// Build packages that need rebuilding (with nil check)
	if plan.Packages != nil && len(plan.Packages) > 0 {
		slog.Info("about to build packages",
			"packageCount", len(plan.Packages),
			"enableParallelBuilds", ib.config.EnableParallelBuilds,
			"parallelGroups", len(plan.ParallelGroups))

		if ib.config.EnableParallelBuilds && len(plan.ParallelGroups) > 1 {
			slog.Info("executing parallel builds")
			// Execute parallel builds
			if err := ib.executeParallelBuilds(ctx, input, projectInfo, plan, result); err != nil {
				result.Success = false
				result.Errors = append(result.Errors, err.Error())
				return result, err
			}
		} else {
			slog.Info("executing sequential builds")
			// Execute sequential builds
			if err := ib.executeSequentialBuilds(ctx, input, projectInfo, plan, result); err != nil {
				result.Success = false
				result.Errors = append(result.Errors, err.Error())
				return result, err
			}
		}
		slog.Info("finished building packages")
	} else {
		slog.Info("no packages to build")
	}

	// Install dependencies with caching if needed
	if !input.IsContainer || input.Dev {
		if err := ib.installDependenciesForBuild(ctx, input, projectInfo, plan, result); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
			return result, err
		}
	}

	// Create final build output
	buildOutput, err := ib.createFinalBuildOutput(ctx, input, projectInfo, result)
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
func (ib *IncrementalBuilder) executeSequentialBuilds(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, plan *BuildPlan, result *BuildResult) error {
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
		ib.progressReporter.UpdateProgress(StageBuildPackages, fmt.Sprintf("Building package %s (%d/%d)", packageName, builtPackages+1, totalPackages))

		slog.Info("building package",
			"package", packageName,
			"reason", packageInfo.RebuildReason)

		slog.Info("about to call buildPackage", "package", packageName)
		if err := ib.buildPackage(ctx, input, projectInfo, packageInfo); err != nil {
			return fmt.Errorf("failed to build package %s: %w", packageName, err)
		}
		slog.Info("buildPackage completed successfully", "package", packageName)

		result.PackagesBuilt = append(result.PackagesBuilt, packageName)
		builtPackages++
	}

	return nil
}

// executeParallelBuilds builds packages in parallel where possible
func (ib *IncrementalBuilder) executeParallelBuilds(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, plan *BuildPlan, result *BuildResult) error {
	slog.Info("executeParallelBuilds starting", "totalGroups", len(plan.ParallelGroups))

	for i, group := range plan.ParallelGroups {
		slog.Info("processing parallel group", "groupIndex", i, "packages", group, "groupSize", len(group))

		// Build packages in this group in parallel
		slog.Info("about to call buildPackageGroup", "groupIndex", i)
		if err := ib.buildPackageGroup(ctx, input, projectInfo, plan, group, result); err != nil {
			slog.Error("buildPackageGroup failed", "groupIndex", i, "error", err)
			return fmt.Errorf("failed to build package group: %w", err)
		}

		slog.Info("completed parallel group", "groupIndex", i)
	}

	slog.Info("executeParallelBuilds completed")
	return nil
}

// buildPackageGroup builds a group of packages in parallel
func (ib *IncrementalBuilder) buildPackageGroup(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, plan *BuildPlan, group []string, result *BuildResult) error {
	slog.Info("building package group", "packages", group, "groupSize", len(group))

	// Create a channel for errors
	errChan := make(chan error, len(group))
	var wg sync.WaitGroup

	// Build each package in the group concurrently
	for _, packageName := range group {
		slog.Info("processing package in group", "package", packageName)

		// Find package info
		var packageInfo *PackageBuildInfo
		for _, pkg := range plan.Packages {
			if pkg.PackageName == packageName {
				packageInfo = pkg
				break
			}
		}

		if packageInfo == nil {
			slog.Warn("package info not found", "package", packageName)
			continue
		}

		if !packageInfo.RequiresRebuild {
			slog.Info("package does not require rebuild, skipping", "package", packageName)
			continue
		}

		slog.Info("package will be built", "package", packageName, "requiresRebuild", packageInfo.RequiresRebuild)

		wg.Add(1)
		go func(pkgName string, pkgInfo *PackageBuildInfo) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("goroutine panicked", "package", pkgName, "panic", r)
					errChan <- fmt.Errorf("goroutine panicked for package %s: %v", pkgName, r)
				}
				slog.Info("goroutine completing for package", "package", pkgName)
				wg.Done()
			}()

			slog.Info("building package (parallel)",
				"package", pkgName,
				"reason", pkgInfo.RebuildReason)

			if err := ib.buildPackage(ctx, input, projectInfo, pkgInfo); err != nil {
				slog.Error("buildPackage failed in goroutine", "package", pkgName, "error", err)
				errChan <- fmt.Errorf("failed to build package %s: %w", pkgName, err)
				return
			}

			slog.Info("buildPackage succeeded, updating result", "package", pkgName)
			// Thread-safe append to result
			ib.mutex.Lock()
			result.PackagesBuilt = append(result.PackagesBuilt, pkgName)
			ib.mutex.Unlock()
			slog.Info("result updated successfully", "package", pkgName)
		}(packageName, packageInfo)
	}

	// Wait for all builds to complete
	slog.Info("waiting for all builds to complete in group", "goroutinesStarted", len(group))
	wg.Wait()
	slog.Info("all builds completed, closing error channel")
	close(errChan)

	// Check for errors
	slog.Info("checking for errors in group")
	for err := range errChan {
		if err != nil {
			slog.Error("found error in group", "error", err)
			return err
		}
	}

	slog.Info("buildPackageGroup completed successfully")
	return nil
}

// buildPackage builds a single package with selective building logic
func (ib *IncrementalBuilder) buildPackage(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, packageInfo *PackageBuildInfo) error {
	slog.Info("ENTERED buildPackage method", "package", packageInfo.PackageName)

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

	// Optimize build type based on development vs production
	buildType := "sdist" // Default to source distribution
	if input.Dev {
		buildType = "wheel" // Use wheel for faster dev builds
	}

	// Build the package using UV
	buildCmd := &UvBuildCommand{
		PackageName:  packageInfo.PackageName,
		PackageDir:   packageInfo.PackageDir,
		OutputDir:    input.Out(),
		SourceFiles:  packageInfo.SourceFiles,
		Dependencies: packageInfo.Dependencies,
		BuildType:    buildType,
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
	slog.Info("about to start post-processing", "package", packageInfo.PackageName)
	if err := ib.postProcessPackageBuild(ctx, input, projectInfo, packageInfo); err != nil {
		slog.Error("post-processing failed", "package", packageInfo.PackageName, "error", err)
		return WrapError(err, "package post-processing").
			WithContext("package", packageInfo.PackageName).
			WithContext("outputDir", input.Out()).
			WithSuggestion("Check if the build output directory is writable")
	}
	slog.Info("post-processing completed", "package", packageInfo.PackageName)

	// Cache the build result for future use
	if err := ib.cachePackageBuildResult(ctx, input, packageInfo); err != nil {
		slog.Warn("failed to cache build result",
			"package", packageInfo.PackageName,
			"error", err)
	}

	slog.Info("successfully built package",
		"package", packageInfo.PackageName,
		"outputDir", input.Out())

	slog.Info("buildPackage method completing", "package", packageInfo.PackageName)
	return nil
}

// createFinalBuildOutput creates the final build output from the build results
func (ib *IncrementalBuilder) createFinalBuildOutput(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, result *BuildResult) (*runtime.BuildOutput, error) {
	// Copy source directories that weren't built as packages
	if err := ib.copySourceDirectories(ctx, input, projectInfo); err != nil {
		return nil, fmt.Errorf("failed to copy source directories: %w", err)
	}

	// Clean up any absolute path directories that may have been created by the cache system
	if err := ib.cleanupAbsolutePaths(input.Out()); err != nil {
		slog.Warn("failed to clean up absolute paths", "error", err)
	}

	// Adjust handler path based on project structure
	adjustedHandler, err := ib.adjustHandlerPath(input, projectInfo)
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
		Out:        input.Out(),
		Handler:    adjustedHandler,
		Errors:     result.Errors,
		Sourcemaps: []string{}, // TODO: collect sourcemaps
	}, nil
}

// copySourceDirectories copies source directories that weren't built as packages
func (ib *IncrementalBuilder) copySourceDirectories(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo) error {
	// Get dependency analysis to find local packages
	analyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
		ProjectResolver: ib.projectResolver,
	})

	analysis, err := analyzer.AnalyzeDependencies(ctx, projectInfo)
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies for source copying: %w", err)
	}

	// Copy source directories that are not buildable packages
	sourceCount := 0
	for _, pkg := range analysis.LocalPackages {
		if !pkg.BuildRequired {
			sourceCount++
		}
	}

	if sourceCount > 0 {
		ib.progressReporter.UpdateProgress(StagePostProcess, fmt.Sprintf("Including %d Python source directories (no build required)", sourceCount))
	}

	copiedCount := 0
	for _, pkg := range analysis.LocalPackages {
		if !pkg.BuildRequired {
			copiedCount++
			// This is a source directory that should be copied directly
			ib.progressReporter.UpdateProgress(StagePostProcess, fmt.Sprintf("Including Python source: %s (%d/%d)", pkg.Name, copiedCount, sourceCount))

			slog.Info("including python source directory",
				"package", pkg.Name,
				"path", pkg.Path,
				"sourceFiles", len(pkg.SourceFiles),
				"reason", "no build configuration required")

			if err := ib.copySourceDirectory(pkg.Path, input.Out(), projectInfo.SourceRoot, projectInfo); err != nil {
				return fmt.Errorf("failed to copy source directory %s: %w", pkg.Path, err)
			}
		}
	}

	return nil
}

// copySourceDirectory copies a source directory to the artifacts directory
func (ib *IncrementalBuilder) copySourceDirectory(srcPath, destDir, workspaceDir string, projectInfo *ProjectInfo) error {
	// Calculate the relative path from workspace to source directory
	relPath, err := filepath.Rel(workspaceDir, srcPath)
	if err != nil {
		return fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Prevent recursive copying by checking if destination is inside source
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute destination path: %w", err)
	}

	absSrcPath, err := filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute source path: %w", err)
	}

	// Check if destination is inside source (would cause recursive copy)
	if strings.HasPrefix(absDestDir, absSrcPath+string(filepath.Separator)) {
		slog.Warn("skipping recursive copy",
			"source", absSrcPath,
			"destination", absDestDir)
		return nil
	}

	destPath := filepath.Join(destDir, relPath)

	// Check if this is a simple structure (source directory is the project root)
	if projectInfo != nil && srcPath == projectInfo.ProjectRoot {
		// For simple structures, copy the contents directly to the artifact directory
		// instead of creating a subdirectory
		slog.Info("detected flat layout, copying contents directly to artifact root",
			"srcPath", srcPath,
			"destDir", destDir)
		return ib.copyFlatLayoutContents(srcPath, destDir)
	}

	// Check if this source directory needs src layout flattening
	// For simplified approach, we check if the source path contains src/ pattern
	if flattenedPath, shouldFlatten := ib.checkForSrcLayoutFlattening(srcPath, destPath, workspaceDir, projectInfo); shouldFlatten {
		slog.Info("flattening src layout for workspace package",
			"srcPath", srcPath,
			"originalDestPath", destPath,
			"flattenedDestPath", flattenedPath)
		destPath = flattenedPath
	}

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy the directory recursively
	return ib.copyDirectoryRecursive(srcPath, destPath)
}

// copyFlatLayoutContents copies the contents of a flat layout directory directly to the destination
func (ib *IncrementalBuilder) copyFlatLayoutContents(srcPath, destDir string) error {
	// Read the source directory
	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Copy each Python file and relevant files directly to the destination
	for _, entry := range entries {
		srcFile := filepath.Join(srcPath, entry.Name())
		destFile := filepath.Join(destDir, entry.Name())

		// Use ContentFilter to determine if file should be excluded
		if ib.contentFilter.ShouldExclude(entry.Name()) {
			slog.Debug("skipping file/directory by content filter", "name", entry.Name())
			continue
		}

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := ib.copyDirectoryRecursive(srcFile, destFile); err != nil {
				return fmt.Errorf("failed to copy directory %s: %w", entry.Name(), err)
			}
		} else {
			// Copy individual files
			if err := ib.copyFile(srcFile, destFile); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// checkForSrcLayoutFlattening determines if a source path needs src layout flattening
func (ib *IncrementalBuilder) checkForSrcLayoutFlattening(srcPath, destPath, workspaceDir string, projectInfo *ProjectInfo) (string, bool) {
	// Parse the source path to see if it matches src/{package_name} pattern
	relSrcPath, err := filepath.Rel(workspaceDir, srcPath)
	if err != nil {
		return destPath, false
	}

	// Split the path to analyze the structure
	pathParts := strings.Split(relSrcPath, string(filepath.Separator))
	if len(pathParts) < 3 {
		return destPath, false // Not enough parts for package/src/package_name
	}

	// Check if this matches the pattern: {package}/src/{package_name}
	packageDir := pathParts[0]
	srcDir := pathParts[1]
	packageName := pathParts[2]

	if srcDir != "src" {
		return destPath, false // Not a src directory
	}

	// Check if this follows the src/{package_name} pattern that needs flattening
	// This is a simplified check based on the path structure
	if strings.Contains(relSrcPath, "/src/") && strings.Contains(relSrcPath, packageName) {
		// This is a src layout that should be flattened
		// Change destination from package/src/package_name to package
		relDestPath, err := filepath.Rel(workspaceDir, destPath)
		if err != nil {
			return destPath, false
		}

		// Replace package/src/package_name with just package
		destPathParts := strings.Split(relDestPath, string(filepath.Separator))
		if len(destPathParts) >= 3 && destPathParts[1] == "src" && destPathParts[2] == packageName {
			// Flatten: package/src/package_name -> package
			flattenedRelPath := packageDir

			// Convert back to absolute path relative to destDir
			finalDestPath := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(destPath))), flattenedRelPath)

			return finalDestPath, true
		}
	}

	return destPath, false
}

// copyDirectoryRecursive copies a directory and all its contents recursively
func (ib *IncrementalBuilder) copyDirectoryRecursive(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories that shouldn't be copied
		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Use ContentFilter to determine if directory should be excluded
			if ib.contentFilter != nil && ib.contentFilter.ShouldExclude(relPath) {
				slog.Debug("skipping directory by content filter", "path", relPath)
				return filepath.SkipDir
			}
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(destPath, info.Mode())
		} else {
			// Use ContentFilter to determine if file should be excluded
			if ib.contentFilter != nil && ib.contentFilter.ShouldExclude(relPath) {
				slog.Debug("skipping file by content filter", "path", relPath)
				return nil // Skip this file
			}

			// Copy file
			return ib.copyFile(path, destPath)
		}
	})
}

// copyFile copies a single file
func (ib *IncrementalBuilder) copyFile(src, dest string) error {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Write to destination
	return os.WriteFile(dest, data, 0644)
}

// adjustHandlerPath adjusts the handler path based on the project structure
func (ib *IncrementalBuilder) adjustHandlerPath(input *runtime.BuildInput, projectInfo *ProjectInfo) (string, error) {
	originalHandler := input.Handler

	// Check if the workspace (pyproject.toml location) is a subdirectory
	// If so, we need to strip that prefix from the handler path
	if projectInfo != nil && projectInfo.PyprojectPath != "" {
		workspaceDir := filepath.Dir(projectInfo.PyprojectPath)
		projectRoot := projectInfo.ProjectRoot

		// Calculate relative path from project root to workspace
		if workspaceDir != projectRoot {
			relWorkspace, err := filepath.Rel(projectRoot, workspaceDir)
			if err == nil && relWorkspace != "." && !strings.HasPrefix(relWorkspace, "..") {
				// Convert to forward slashes for handler path comparison
				relWorkspaceSlash := strings.ReplaceAll(relWorkspace, string(filepath.Separator), "/")

				// Check if handler starts with the workspace prefix
				if strings.HasPrefix(originalHandler, relWorkspaceSlash+"/") {
					// Strip the workspace prefix from the handler
					adjustedHandler := strings.TrimPrefix(originalHandler, relWorkspaceSlash+"/")
					slog.Info("incremental builder handler path adjustment",
						"original", originalHandler,
						"adjusted", adjustedHandler,
						"reason", "stripped workspace prefix",
						"workspaceDir", relWorkspaceSlash)
					return adjustedHandler, nil
				}
			}
		}
	}

	// Check if handler path contains the pattern {package_name}/src/{package_name}
	// If so, we need to adjust it because the build process flattens this structure
	if strings.Contains(originalHandler, "/src/") {
		// Pattern: functions/src/functions/api.handler -> functions/api.handler
		parts := strings.Split(originalHandler, "/")
		var adjustedParts []string

		skipNext := false
		for i, part := range parts {
			if skipNext {
				skipNext = false
				continue
			}

			if part == "src" && i > 0 && i < len(parts)-1 {
				// Check if the next part matches the previous part (package name)
				if i+1 < len(parts) && parts[i+1] == parts[i-1] {
					// Skip both "src" and the duplicate package name
					skipNext = true
					continue
				}
			}
			adjustedParts = append(adjustedParts, part)
		}

		adjustedHandler := strings.Join(adjustedParts, "/")
		slog.Info("incremental builder handler path adjustment",
			"original", originalHandler,
			"adjusted", adjustedHandler,
			"reason", "flattened src structure")
		return adjustedHandler, nil
	}

	slog.Info("incremental builder handler path adjustment", "original", originalHandler, "adjusted", "no change needed")
	return originalHandler, nil
}

// updateCacheAfterBuild updates the cache with build results
func (ib *IncrementalBuilder) updateCacheAfterBuild(input *runtime.BuildInput, projectInfo *ProjectInfo, result *BuildResult) error {
	if result.CachedBuildOutput == nil {
		return fmt.Errorf("no cached build output to store")
	}

	// Update change detector cache
	return ib.changeDetector.UpdateCacheAfterBuild(
		input.FunctionID,
		input.Handler,
		projectInfo,
		result.CachedBuildOutput,
	)
}

// reuseCachedPackageBuild attempts to reuse a cached package build
func (ib *IncrementalBuilder) reuseCachedPackageBuild(ctx context.Context, input *runtime.BuildInput, packageInfo *PackageBuildInfo) error {
	// Check if we have cached build artifacts for this package
	// Use filesystem-safe cache key without special characters
	packageFunctionID := fmt.Sprintf("pkg_%s", packageInfo.PackageName)

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
func (ib *IncrementalBuilder) postProcessPackageBuild(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, packageInfo *PackageBuildInfo) error {
	// Extract and process the built package archive
	if err := ib.extractAndProcessPackageArchive(input.Out(), projectInfo, packageInfo); err != nil {
		return fmt.Errorf("failed to extract and process package archive: %w", err)
	}

	// Handle src/{package_name} structure adjustment if needed
	if err := ib.adjustPackageStructure(input.Out(), packageInfo); err != nil {
		return fmt.Errorf("failed to adjust package structure: %w", err)
	}

	return nil
}

// extractAndProcessPackageArchive extracts the built package archive and processes it
func (ib *IncrementalBuilder) extractAndProcessPackageArchive(outputDir string, projectInfo *ProjectInfo, packageInfo *PackageBuildInfo) error {
	// Look for the built package file (either .whl or .tar.gz)
	// Python normalizes package names by converting dashes to underscores
	normalizedName := strings.ReplaceAll(packageInfo.PackageName, "-", "_")

	// Try wheel files first (more common for dev builds)
	patterns := []string{
		filepath.Join(outputDir, normalizedName+"-*.whl"),
		filepath.Join(outputDir, normalizedName+"-*.tar.gz"),
		filepath.Join(outputDir, packageInfo.PackageName+"-*.whl"),
		filepath.Join(outputDir, packageInfo.PackageName+"-*.tar.gz"),
	}

	var files []string
	var err error

	for _, pattern := range patterns {
		files, err = filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("failed to find package archive: %w", err)
		}
		if len(files) > 0 {
			break
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no package archive found for %s (tried patterns: %s-*.whl, %s-*.tar.gz, %s-*.whl, %s-*.tar.gz)",
			packageInfo.PackageName, normalizedName, normalizedName, packageInfo.PackageName, packageInfo.PackageName)
	}

	// Process each archive file (should typically be just one)
	for _, archiveFile := range files {
		if err := ib.processPackageArchive(archiveFile, outputDir, projectInfo, packageInfo); err != nil {
			return fmt.Errorf("failed to process archive %s: %w", archiveFile, err)
		}
	}

	return nil
}

// processPackageArchive processes a single package archive file
func (ib *IncrementalBuilder) processPackageArchive(archiveFile, outputDir string, projectInfo *ProjectInfo, packageInfo *PackageBuildInfo) error {
	// Handle different archive types
	if strings.HasSuffix(archiveFile, ".whl") {
		// For wheel files, we can skip extraction since they're already built
		// The wheel file itself is the final artifact
		slog.Info("wheel file built successfully, skipping extraction", "wheel", archiveFile)
		return nil
	}

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
	if err := ib.moveExtractedPackage(extractedDir, targetDir, baseName, projectInfo); err != nil {
		return fmt.Errorf("failed to move extracted package: %w", err)
	}

	// Remove the original archive file
	if err := os.Remove(archiveFile); err != nil {
		slog.Warn("failed to remove archive file", "file", archiveFile, "error", err)
	}

	return nil
}

// moveExtractedPackage moves the extracted package to the correct location
func (ib *IncrementalBuilder) moveExtractedPackage(extractedDir, targetDir, baseName string, projectInfo *ProjectInfo) error {
	// Use a unified approach for all project structures
	// First, check if there's a src/{package_name} structure
	srcPath := filepath.Join(extractedDir, "src", baseName)
	if _, err := os.Stat(srcPath); err == nil {
		// For src layout, we need to flatten the structure by moving src/{package_name} contents
		// directly to the target directory to match the adjusted handler paths

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
		// Check if this is a package with Python files that need to be moved to root level
		if ib.shouldFlattenPackage(extractedDir) {
			return ib.flattenPackageToRoot(extractedDir, targetDir)
		}

		// Standard case: just rename the directory
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove old directory: %w", err)
		}

		if err := os.Rename(extractedDir, targetDir); err != nil {
			return fmt.Errorf("failed to rename directory: %w", err)
		}
	}

	return nil
}

// shouldFlattenPackage determines if a package should have its files moved to root level
func (ib *IncrementalBuilder) shouldFlattenPackage(extractedDir string) bool {
	// Check if the directory contains Python files that should be at root level
	// This is determined by the presence of Python files directly in the extracted directory
	entries, err := os.ReadDir(extractedDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
			return true
		}
	}
	return false
}

// flattenPackageToRoot moves Python files from a package directory to the root level
func (ib *IncrementalBuilder) flattenPackageToRoot(extractedDir, outputDir string) error {
	// Move Python files from the extracted package directory to the root level
	slog.Debug("flattening package to root level", "from", extractedDir, "to", outputDir)

	// Find all Python files in the extracted directory
	var pythonFiles []string
	err := filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Include Python files and other relevant files
		ext := filepath.Ext(path)
		if ext == ".py" || ext == ".pyi" || info.Name() == "py.typed" {
			pythonFiles = append(pythonFiles, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk extracted directory: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Move each Python file to the root of the output directory
	for _, srcFile := range pythonFiles {
		// For packages that need flattening, move files to root level
		fileName := filepath.Base(srcFile)
		destFile := filepath.Join(outputDir, fileName)

		// Copy the file
		if err := ib.copyFile(srcFile, destFile); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", srcFile, destFile, err)
		}

		slog.Debug("flattened package file", "from", srcFile, "to", destFile)
	}

	// Clean up the extracted directory
	if err := os.RemoveAll(extractedDir); err != nil {
		slog.Warn("failed to clean up extracted directory", "dir", extractedDir, "error", err)
	}

	slog.Info("successfully flattened package files to root level", "fileCount", len(pythonFiles))
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
	// Use filesystem-safe cache key without special characters
	packageFunctionID := fmt.Sprintf("pkg_%s", packageInfo.PackageName)

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

	ib.projectResolver.ClearCache()

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
		case ErrorTypeProjectStructure:
			return ib.recoverFromProjectStructureFailure(pythonErr)
		case ErrorTypeHandlerNotFound:
			return ib.recoverFromHandlerNotFoundFailure(pythonErr)
		case ErrorTypeConfigurationError:
			return ib.recoverFromConfigurationFailure(pythonErr)
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

// recoverFromProjectStructureFailure handles project structure failure recovery
func (ib *IncrementalBuilder) recoverFromProjectStructureFailure(err *PythonRuntimeError) error {
	slog.Info("attempting to recover from project structure failure")

	// Clear project resolver cache and try again
	ib.projectResolver.ClearCache()

	// For now, just return the error - more sophisticated fallback could be implemented
	return err
}

// recoverFromHandlerNotFoundFailure handles handler not found failure recovery
func (ib *IncrementalBuilder) recoverFromHandlerNotFoundFailure(err *PythonRuntimeError) error {
	slog.Info("attempting to recover from handler not found failure")

	// Extract handler path from error context
	handler, ok := err.Context["handler"].(string)
	if !ok {
		return err
	}

	// Try to suggest alternative handler locations
	if searchPaths, exists := err.Context["searchPaths"].([]string); exists {
		slog.Info("handler not found in search paths", "handler", handler, "searchPaths", searchPaths)
	}

	// For now, just return the error - could implement file discovery here
	return err
}

// recoverFromConfigurationFailure handles configuration failure recovery
func (ib *IncrementalBuilder) recoverFromConfigurationFailure(err *PythonRuntimeError) error {
	slog.Info("attempting to recover from configuration failure")

	// Extract configuration file from error context
	configFile, ok := err.Context["configFile"].(string)
	if !ok {
		return err
	}

	slog.Info("configuration issue detected", "configFile", configFile)

	// For now, just return the error - could implement config validation/repair here
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
func (ib *IncrementalBuilder) installDependenciesForBuild(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, plan *BuildPlan, result *BuildResult) error {
	// Generate requirements file first to get the dependency list
	requirementsFile := filepath.Join(input.Out(), "requirements.txt")
	if err := ib.generateRequirementsFile(ctx, input, projectInfo, requirementsFile); err != nil {
		return fmt.Errorf("failed to generate requirements file: %w", err)
	}

	// Determine architecture for Lambda
	architecture := "x86_64"
	if props, err := ib.parseInputProperties(input); err == nil && props.Architecture != "" {
		architecture = props.Architecture
	}

	// Install dependencies for the correct target platform (Linux)
	ib.progressReporter.UpdateProgress(StageBuildPackages, "Installing dependencies for Lambda")

	if err := ib.installDependenciesForLambda(ctx, input, projectInfo, requirementsFile, architecture); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	return nil
}

// generateRequirementsFile generates a requirements.txt file for the build
func (ib *IncrementalBuilder) generateRequirementsFile(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, outputFile string) error {
	// Use UV export to generate requirements
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	slog.Info("generateRequirementsFile starting",
		"pyprojectPath", projectInfo.PyprojectPath,
		"sourceRoot", projectInfo.SourceRoot,
		"workspaceDir", workspaceDir,
		"outputFile", outputFile)

	// Check if this is a source code project (not a buildable package)
	// If so, include dev dependencies since they might contain runtime deps
	noDev := true
	if projectInfo.PyprojectPath != "" {
		if content, err := os.ReadFile(projectInfo.PyprojectPath); err == nil {
			contentStr := string(content)
			if strings.Contains(contentStr, "NOT a buildable package") ||
				strings.Contains(contentStr, "Development environment - not a buildable package") ||
				strings.Contains(contentStr, "SST will treat this as source code") {
				noDev = false // Include dev dependencies for source code projects
				slog.Info("including dev dependencies for source code project", "path", projectInfo.PyprojectPath)
			}
		}
	}

	// Check if this is a workspace member (has its own pyproject.toml in a subdirectory)
	// If so, export only that package's dependencies for isolation
	packageName := ""
	useAllPackages := true

	if projectInfo.PyprojectPath != "" {
		slog.Info("checking for workspace member status", "pyprojectPath", projectInfo.PyprojectPath)
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			slog.Info("parsed pyproject.toml", "projectName", config.Project.Name)
			if config.Project.Name != "" {
				// Check if this pyproject.toml is in a subdirectory (workspace member)
				// by walking up to find a parent pyproject.toml (workspace root)
				pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)
				currentDir := filepath.Dir(pyprojectDir)

				slog.Info("walking up to find parent pyproject.toml",
					"pyprojectDir", pyprojectDir,
					"startingDir", currentDir,
					"sourceRoot", projectInfo.SourceRoot)

				// Walk up to find any parent pyproject.toml
				iterCount := 0
				for currentDir != "/" && currentDir != "." && currentDir != projectInfo.SourceRoot {
					iterCount++
					parentPyproject := filepath.Join(currentDir, "pyproject.toml")
					slog.Info("checking for parent pyproject.toml",
						"iteration", iterCount,
						"currentDir", currentDir,
						"checking", parentPyproject)
					if _, err := os.Stat(parentPyproject); err == nil {
						// Found a parent pyproject.toml - this is a workspace member
						packageName = config.Project.Name
						useAllPackages = false
						slog.Info("found parent pyproject.toml - using per-package dependency export",
							"package", packageName,
							"pyprojectPath", projectInfo.PyprojectPath,
							"parentPyproject", parentPyproject)
						break
					}
					currentDir = filepath.Dir(currentDir)
				}
				if useAllPackages {
					slog.Info("no parent pyproject.toml found - using all-packages export",
						"finalDir", currentDir,
						"iterations", iterCount,
						"stoppedBecause", fmt.Sprintf("currentDir=%s, sourceRoot=%s, isRoot=%v, isDot=%v",
							currentDir, projectInfo.SourceRoot,
							currentDir == "/", currentDir == "."))
				}
			}
		} else {
			slog.Warn("failed to parse pyproject.toml", "path", projectInfo.PyprojectPath, "error", err)
		}
	}

	slog.Info("generateRequirementsFile decision",
		"packageName", packageName,
		"useAllPackages", useAllPackages,
		"workspaceDir", workspaceDir)

	exportCmd := &UvExportCommand{
		WorkspaceDir:    workspaceDir,
		PackageName:     packageName,
		OutputFile:      outputFile,
		NoEmitWorkspace: false,
		NoDev:           noDev,
		AllPackages:     useAllPackages,
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

// cleanupAbsolutePaths removes any directories with absolute paths from the artifact directory
func (ib *IncrementalBuilder) cleanupAbsolutePaths(artifactDir string) error {
	slog.Info("cleaning up absolute paths from artifact directory", "artifactDir", artifactDir)

	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		return fmt.Errorf("failed to read artifact directory: %w", err)
	}

	var removedItems []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if directory name contains absolute path indicators
			name := entry.Name()
			if strings.Contains(name, ":") && (strings.Contains(name, "/Users/") || strings.Contains(name, "/home/") || strings.Contains(name, "C:\\")) {
				itemPath := filepath.Join(artifactDir, name)
				slog.Info("removing absolute path directory", "path", itemPath)

				if err := os.RemoveAll(itemPath); err != nil {
					slog.Warn("failed to remove absolute path directory", "path", itemPath, "error", err)
				} else {
					removedItems = append(removedItems, name)
				}
			}
		}
	}

	if len(removedItems) > 0 {
		slog.Info("cleaned up absolute path directories",
			"artifactDir", artifactDir,
			"removedItems", removedItems,
			"count", len(removedItems))
	}

	return nil
}

// installDependenciesForLambda installs dependencies for Lambda with correct platform targeting
func (ib *IncrementalBuilder) installDependenciesForLambda(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, requirementsFile string, architecture string) error {
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	// Check if this is a source code project (not a buildable package)
	// If so, include dev dependencies since they might contain runtime deps
	noDev := true
	if projectInfo.PyprojectPath != "" {
		if content, err := os.ReadFile(projectInfo.PyprojectPath); err == nil {
			contentStr := string(content)
			if strings.Contains(contentStr, "NOT a buildable package") ||
				strings.Contains(contentStr, "Development environment - not a buildable package") ||
				strings.Contains(contentStr, "SST will treat this as source code") {
				noDev = false // Include dev dependencies for source code projects
				slog.Info("including dev dependencies for source code project in sync", "path", projectInfo.PyprojectPath)
			}
		}
	}

	// Check if this is a workspace member (has its own pyproject.toml in a subdirectory)
	// If so, sync only that package's dependencies for isolation
	packageName := ""
	useAllPackages := true

	if projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			if config.Project.Name != "" {
				// Check if this pyproject.toml is in a subdirectory (workspace member)
				// by walking up to find a parent pyproject.toml (workspace root)
				pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)
				currentDir := filepath.Dir(pyprojectDir)

				// Walk up to find any parent pyproject.toml
				for currentDir != "/" && currentDir != "." && currentDir != projectInfo.SourceRoot {
					parentPyproject := filepath.Join(currentDir, "pyproject.toml")
					if _, err := os.Stat(parentPyproject); err == nil {
						// Found a parent pyproject.toml - this is a workspace member
						packageName = config.Project.Name
						useAllPackages = false
						slog.Info("using per-package sync for workspace member",
							"package", packageName,
							"pyprojectPath", projectInfo.PyprojectPath,
							"parentPyproject", parentPyproject)
						break
					}
					currentDir = filepath.Dir(currentDir)
				}
			}
		}
	}

	// Build sync command with appropriate package targeting
	packages := []string{}
	if packageName != "" {
		packages = []string{packageName}
	}

	syncCmd := &UvSyncCommand{
		WorkspaceDir: workspaceDir,
		AllPackages:  useAllPackages,
		NoDev:        noDev,
		Packages:     packages,
	}

	ib.progressReporter.UpdateProgress(StageBuildPackages, "Syncing workspace dependencies")

	if err := ib.uvRunner.ExecuteSyncCommand(ctx, syncCmd); err != nil {
		return fmt.Errorf("failed to sync workspace dependencies: %w", err)
	}

	// Simplified approach: copy source files based on handler path
	ib.progressReporter.UpdateProgress(StageBuildPackages, "Copying source files")
	slog.Info("copying source files based on handler path", "handler", input.Handler)

	if err := ib.copySourceFilesSimple(ctx, input, projectInfo); err != nil {
		return fmt.Errorf("failed to copy source files: %w", err)
	}

	// Copy the synced packages from .venv to output directory
	// This includes git dependencies like sst that are properly resolved after sync
	ib.progressReporter.UpdateProgress(StageBuildPackages, "Copying synced dependencies")

	if err := ib.copySyncedDependencies(ctx, input, projectInfo, architecture); err != nil {
		return fmt.Errorf("failed to copy synced dependencies: %w", err)
	}

	return nil
}

// copySourceFilesSimple copies source files using a simple, handler-path-based approach
func (ib *IncrementalBuilder) copySourceFilesSimple(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo) error {
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	slog.Info("copying source files with simple approach", "handler", input.Handler, "workspaceDir", workspaceDir)

	// Parse the handler path to understand what directories we need
	handlerPath := input.Handler

	// Extract the directory part of the handler path and resolve it relative to workspaceDir
	// Examples:
	// - "handler.main" -> copy all .py files from root
	// - "app/functions/api/handler.main" -> copy "app" directory
	// - "src/mypackage/handler.api_handler" -> copy "src" directory
	// - "functions/src/functions/user/handler.main" -> copy entire structure

	var dirsToInclude []string
	var rootFilesToInclude []string

	if strings.Contains(handlerPath, "/") {
		// Handler is in a subdirectory, determine what to copy
		parts := strings.Split(handlerPath, "/")

		// Find the actual handler file to verify the path structure
		handlerFile := strings.Join(parts[:len(parts)-1], "/") + ".py"
		fullHandlerPath := filepath.Join(workspaceDir, handlerFile)

		slog.Info("checking handler file path", "handlerFile", handlerFile, "fullPath", fullHandlerPath)

		// Check if the handler file exists at the expected location
		if _, err := os.Stat(fullHandlerPath); err == nil {
			// Handler file exists, determine what directories to copy
			// We need to copy the top-level directory that contains the handler
			topLevelDir := parts[0]

			// But first check if this directory actually exists in workspaceDir
			topLevelPath := filepath.Join(workspaceDir, topLevelDir)
			if _, err := os.Stat(topLevelPath); err == nil {
				dirsToInclude = append(dirsToInclude, topLevelDir)
				slog.Info("handler in subdirectory, including directory", "dir", topLevelDir)
			} else {
				// The top-level directory doesn't exist in workspaceDir
				// This might mean workspaceDir is already pointing to a subdirectory
				// In this case, copy everything from workspaceDir
				slog.Info("top-level directory not found in workspace, copying entire workspace", "topLevelDir", topLevelDir, "workspaceDir", workspaceDir)

				entries, err := os.ReadDir(workspaceDir)
				if err != nil {
					return fmt.Errorf("failed to read workspace directory: %w", err)
				}

				for _, entry := range entries {
					if entry.IsDir() {
						dirsToInclude = append(dirsToInclude, entry.Name())
					} else if strings.HasSuffix(entry.Name(), ".py") {
						rootFilesToInclude = append(rootFilesToInclude, entry.Name())
					}
				}
			}
		} else {
			// Handler file not found at expected location
			// Try to find it by removing parts of the path (in case SourceRoot is deeper)
			var actualHandlerPath string
			var relativeHandlerPath string

			for i := 1; i < len(parts)-1; i++ {
				// Try removing the first i parts of the path
				remainingParts := parts[i:]
				testHandlerFile := strings.Join(remainingParts[:len(remainingParts)-1], "/") + ".py"
				testFullPath := filepath.Join(workspaceDir, testHandlerFile)

				slog.Info("trying alternative handler path", "testHandlerFile", testHandlerFile, "testFullPath", testFullPath)

				if _, err := os.Stat(testFullPath); err == nil {
					actualHandlerPath = testFullPath
					relativeHandlerPath = testHandlerFile
					break
				}
			}

			if actualHandlerPath != "" {
				// Handler file found at alternative location
				relativeDir := filepath.Dir(relativeHandlerPath)
				if relativeDir == "." {
					// Handler is at root level of workspaceDir
					slog.Info("handler found at workspace root, copying all content")
					entries, err := os.ReadDir(workspaceDir)
					if err != nil {
						return fmt.Errorf("failed to read workspace directory: %w", err)
					}

					for _, entry := range entries {
						if entry.IsDir() {
							dirsToInclude = append(dirsToInclude, entry.Name())
						} else if strings.HasSuffix(entry.Name(), ".py") {
							rootFilesToInclude = append(rootFilesToInclude, entry.Name())
						}
					}
				} else {
					// Handler is in a subdirectory, copy the top-level directory
					topLevelDir := strings.Split(relativeDir, "/")[0]
					dirsToInclude = append(dirsToInclude, topLevelDir)
					slog.Info("handler found in subdirectory, including directory", "dir", topLevelDir, "relativeDir", relativeDir)
				}
			} else {
				// Handler file not found anywhere, fall back to copying based on original path
				topLevelDir := parts[0]
				topLevelPath := filepath.Join(workspaceDir, topLevelDir)
				if _, err := os.Stat(topLevelPath); err == nil {
					dirsToInclude = append(dirsToInclude, topLevelDir)
					slog.Info("handler file not found, but directory exists, including directory", "dir", topLevelDir)
				} else {
					// Last resort: copy everything from workspaceDir
					slog.Info("handler file not found and directory doesn't exist, copying entire workspace", "handlerPath", handlerPath, "workspaceDir", workspaceDir)

					entries, err := os.ReadDir(workspaceDir)
					if err != nil {
						return fmt.Errorf("failed to read workspace directory: %w", err)
					}

					for _, entry := range entries {
						if entry.IsDir() {
							// Skip virtual environments and build artifacts
							if ib.contentFilter != nil && ib.contentFilter.ShouldExclude(entry.Name()) {
								slog.Debug("skipping excluded directory", "name", entry.Name())
								continue
							}
							dirsToInclude = append(dirsToInclude, entry.Name())
						} else if strings.HasSuffix(entry.Name(), ".py") {
							rootFilesToInclude = append(rootFilesToInclude, entry.Name())
						}
					}
				}
			}
		}
	} else {
		// Handler is at root level, copy all Python files from root
		slog.Info("handler at root level, copying root Python files")

		entries, err := os.ReadDir(workspaceDir)
		if err != nil {
			return fmt.Errorf("failed to read workspace directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
				rootFilesToInclude = append(rootFilesToInclude, entry.Name())
			}
		}
	}

	// Also include any additional directories that might be needed (like shared utilities)
	// Look for common directory names that often contain shared code
	commonDirs := []string{"shared", "common", "utils", "lib", "core"}
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return fmt.Errorf("failed to read workspace directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirName := entry.Name()
			// Include common directories if they contain Python files
			for _, commonDir := range commonDirs {
				if dirName == commonDir {
					dirPath := filepath.Join(workspaceDir, dirName)
					if ib.containsPythonContent(dirPath) {
						// Only add if not already included
						found := false
						for _, existing := range dirsToInclude {
							if existing == dirName {
								found = true
								break
							}
						}
						if !found {
							dirsToInclude = append(dirsToInclude, dirName)
							slog.Info("including common directory", "dir", dirName)
						}
					}
					break
				}
			}
		}
	}

	// Copy directories
	for _, dirName := range dirsToInclude {
		srcPath := filepath.Join(workspaceDir, dirName)
		destPath := filepath.Join(input.Out(), dirName)

		slog.Info("copying directory", "src", srcPath, "dest", destPath)

		if err := ib.copyDirectoryRecursive(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy directory %s: %w", dirName, err)
		}
	}

	// Copy root Python files
	for _, fileName := range rootFilesToInclude {
		srcPath := filepath.Join(workspaceDir, fileName)
		destPath := filepath.Join(input.Out(), fileName)

		slog.Info("copying root file", "src", srcPath, "dest", destPath)

		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", fileName, err)
		}
	}

	slog.Info("successfully copied source files",
		"directories", dirsToInclude,
		"rootFiles", rootFilesToInclude)

	return nil
}

// containsPythonContent checks if a directory contains Python files
func (ib *IncrementalBuilder) containsPythonContent(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
			return true
		}
		if entry.IsDir() {
			subPath := filepath.Join(dirPath, entry.Name())
			if ib.containsPythonContent(subPath) {
				return true
			}
		}
	}
	return false
}

// copySyncedDependencies installs external dependencies with correct platform targeting
func (ib *IncrementalBuilder) copySyncedDependencies(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, architecture string) error {
	// Use the requirements.txt that was already exported by the export step
	requirementsPath := filepath.Join(input.Out(), "requirements.txt")

	// Check if requirements.txt exists
	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		slog.Warn("requirements.txt not found, skipping dependency installation", "path", requirementsPath)
		return nil
	}

	// Get workspace root directory - find the topmost pyproject.toml (UV workspace root)
	workspaceRoot := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		// Walk up from pyproject.toml to find the workspace root (where the root pyproject.toml with [tool.uv.workspace] is)
		// We want to find the HIGHEST directory that contains a pyproject.toml
		pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)
		workspaceRoot = pyprojectDir // Start with the current pyproject.toml location

		// Walk up the directory tree looking for pyproject.toml files
		currentDir := pyprojectDir
		for {
			parentDir := filepath.Dir(currentDir)
			// Stop if we've reached the root or haven't moved
			if parentDir == currentDir || parentDir == "/" || parentDir == "." {
				break
			}
			parentPyproject := filepath.Join(parentDir, "pyproject.toml")
			if _, err := os.Stat(parentPyproject); err == nil {
				// Found a pyproject.toml in parent, this becomes our new potential workspace root
				workspaceRoot = parentDir
				slog.Debug("found pyproject.toml in parent", "parentDir", parentDir)
			}
			currentDir = parentDir
		}
	}
	slog.Info("determined workspace root for dependency installation", "workspaceRoot", workspaceRoot, "pyprojectPath", projectInfo.PyprojectPath)

	// Filter out workspace packages from requirements.txt and copy their source code
	filteredRequirementsPath := filepath.Join(input.Out(), "requirements-filtered.txt")
	workspacePackagePaths, err := ib.filterWorkspacePackagesFromRequirementsAndGetPaths(requirementsPath, filteredRequirementsPath, projectInfo, workspaceRoot)
	if err != nil {
		return fmt.Errorf("failed to filter workspace packages from requirements: %w", err)
	}

	// Copy workspace package source code to artifact
	if err := ib.copyWorkspacePackageSources(ctx, workspacePackagePaths, workspaceRoot, input.Out()); err != nil {
		return fmt.Errorf("failed to copy workspace package sources: %w", err)
	}

	// Use the filtered requirements file
	requirementsPath = filteredRequirementsPath

	slog.Info("installing dependencies using uv pip install",
		"requirementsFile", requirementsPath,
		"targetDir", input.Out(),
		"architecture", architecture)

	// Build the uv pip install command with platform targeting (same as legacy builder)
	args := []string{"pip", "install", "-r", requirementsPath, "--target", input.Out()}

	// Add platform targeting for non-dev builds (same logic as legacy builder)
	if !input.Dev {
		pythonPlatform := "x86_64-unknown-linux-gnu"
		if architecture == "arm64" {
			pythonPlatform = "aarch64-unknown-linux-gnu"
		}
		args = append(args, "--python-platform", pythonPlatform)
		slog.Info("using platform targeting", "platform", pythonPlatform)
	}

	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	// Execute the uv pip install command
	// Run from the workspace directory where pyproject.toml exists, not the artifact directory
	installCmd := process.CommandContext(ctx, "uv", args...)
	installCmd.Dir = workspaceDir

	installOutput, err := installCmd.CombinedOutput()
	if err != nil {
		slog.Error("uv pip install failed",
			"command", strings.Join(installCmd.Args, " "),
			"error", err,
			"output", string(installOutput))
		return fmt.Errorf("failed to run uv pip install: %v\n%s", err, string(installOutput))
	}

	slog.Info("uv pip install completed successfully", "output", string(installOutput))

	// Clean up __pycache__ directories and .pyc files from installed dependencies
	if err := ib.cleanupInstalledDependencies(input.Out()); err != nil {
		slog.Warn("failed to clean up installed dependencies", "error", err)
		// Don't fail the build for cleanup issues, just warn
	}

	// Clean up the filtered requirements file
	os.Remove(filteredRequirementsPath)

	return nil
}

// filterWorkspacePackagesFromRequirementsAndGetPaths filters out workspace packages from requirements.txt
// and returns the paths of the filtered workspace packages for source copying
func (ib *IncrementalBuilder) filterWorkspacePackagesFromRequirementsAndGetPaths(inputPath, outputPath string, projectInfo *ProjectInfo, workspaceRoot string) ([]string, error) {
	// Read the original requirements.txt
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read requirements file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var filteredLines []string
	var workspacePackagePaths []string

	// Get workspace package names
	workspacePackages := ib.getWorkspacePackageNames(projectInfo)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			filteredLines = append(filteredLines, line)
			continue
		}

		// Skip all editable installs that reference local paths
		if strings.HasPrefix(line, "-e ") {
			editablePath := strings.TrimSpace(strings.TrimPrefix(line, "-e "))

			// Filter out any editable install that references a local path
			if strings.HasPrefix(editablePath, "./") || strings.HasPrefix(editablePath, "../") ||
				strings.HasPrefix(editablePath, "/") && !strings.Contains(editablePath, "://") {
				slog.Debug("filtering out editable local path from requirements", "line", line)
				// Record the path for source copying
				fullPath := filepath.Join(workspaceRoot, editablePath)
				if strings.HasPrefix(editablePath, "./") {
					fullPath = filepath.Join(workspaceRoot, strings.TrimPrefix(editablePath, "./"))
				}
				workspacePackagePaths = append(workspacePackagePaths, fullPath)
				continue
			}

			// Also check for workspace packages by name
			shouldSkip := false
			for _, pkg := range workspacePackages {
				if strings.Contains(editablePath, "./"+pkg) || strings.Contains(editablePath, pkg) {
					slog.Debug("filtering out workspace package from requirements", "line", line, "package", pkg)
					fullPath := filepath.Join(workspaceRoot, editablePath)
					if strings.HasPrefix(editablePath, "./") {
						fullPath = filepath.Join(workspaceRoot, strings.TrimPrefix(editablePath, "./"))
					}
					workspacePackagePaths = append(workspacePackagePaths, fullPath)
					shouldSkip = true
					break
				}
			}
			if shouldSkip {
				continue
			}
		}

		// Skip file:// URLs and local path references that might not exist in the build environment
		if strings.HasPrefix(line, "file://") || strings.Contains(line, "file://") {
			slog.Debug("filtering out file:// URL from requirements", "line", line)
			continue
		}

		// Skip local path references (relative paths starting with ./ or ../)
		if strings.HasPrefix(line, "./") || strings.HasPrefix(line, "../") {
			slog.Debug("filtering out local path reference from requirements", "line", line)
			continue
		}

		// Skip absolute local paths that might be workspace-specific
		if strings.HasPrefix(line, "/") && !strings.Contains(line, "://") {
			slog.Debug("filtering out absolute local path from requirements", "line", line)
			continue
		}

		// Include the line
		filteredLines = append(filteredLines, line)
	}

	// Write the filtered requirements.txt
	filteredContent := strings.Join(filteredLines, "\n")
	if err := os.WriteFile(outputPath, []byte(filteredContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write filtered requirements file: %w", err)
	}

	slog.Info("filtered workspace packages from requirements.txt",
		"originalLines", len(lines),
		"filteredLines", len(filteredLines),
		"workspacePackages", workspacePackages,
		"workspacePackagePaths", workspacePackagePaths)

	return workspacePackagePaths, nil
}

// copyWorkspacePackageSources copies workspace package source code to the artifact directory
func (ib *IncrementalBuilder) copyWorkspacePackageSources(ctx context.Context, packagePaths []string, workspaceRoot string, artifactDir string) error {
	if len(packagePaths) == 0 {
		return nil
	}

	slog.Info("copying workspace package sources to artifact",
		"packagePaths", packagePaths,
		"workspaceRoot", workspaceRoot,
		"artifactDir", artifactDir)

	for _, pkgPath := range packagePaths {
		// Parse the package's pyproject.toml to determine the package name and source layout
		pyprojectPath := filepath.Join(pkgPath, "pyproject.toml")
		if _, err := os.Stat(pyprojectPath); os.IsNotExist(err) {
			slog.Warn("workspace package has no pyproject.toml, skipping", "path", pkgPath)
			continue
		}

		config, err := ib.projectResolver.ParsePyprojectToml(pyprojectPath)
		if err != nil {
			slog.Warn("failed to parse workspace package pyproject.toml", "path", pyprojectPath, "error", err)
			continue
		}

		packageName := config.Project.Name
		if packageName == "" {
			packageName = filepath.Base(pkgPath)
		}

		// Determine source directory based on hatch configuration or convention
		// Check for src/{package_name}/ layout first (common convention)
		srcLayoutPath := filepath.Join(pkgPath, "src", packageName)
		if _, err := os.Stat(srcLayoutPath); err == nil {
			// Copy src/{package_name}/ as {package_name}/ in artifact
			destPath := filepath.Join(artifactDir, packageName)
			slog.Info("copying workspace package with src layout",
				"package", packageName,
				"source", srcLayoutPath,
				"dest", destPath)

			if err := ib.copyDirectory(srcLayoutPath, destPath); err != nil {
				return fmt.Errorf("failed to copy workspace package %s: %w", packageName, err)
			}
			continue
		}

		// Check for flat src/ layout (src/ contains the package code directly)
		srcPath := filepath.Join(pkgPath, "src")
		if _, err := os.Stat(srcPath); err == nil {
			// Check if src/ contains an __init__.py (flat package) or subdirectories
			srcInitPath := filepath.Join(srcPath, "__init__.py")
			if _, err := os.Stat(srcInitPath); err == nil {
				// Flat src layout - copy src/ as {package_name}/
				destPath := filepath.Join(artifactDir, packageName)
				slog.Info("copying workspace package with flat src layout",
					"package", packageName,
					"source", srcPath,
					"dest", destPath)

				if err := ib.copyDirectory(srcPath, destPath); err != nil {
					return fmt.Errorf("failed to copy workspace package %s: %w", packageName, err)
				}
				continue
			}
		}

		// Check for {package_name}/ directory at package root
		pkgDirPath := filepath.Join(pkgPath, packageName)
		if _, err := os.Stat(pkgDirPath); err == nil {
			destPath := filepath.Join(artifactDir, packageName)
			slog.Info("copying workspace package from root directory",
				"package", packageName,
				"source", pkgDirPath,
				"dest", destPath)

			if err := ib.copyDirectory(pkgDirPath, destPath); err != nil {
				return fmt.Errorf("failed to copy workspace package %s: %w", packageName, err)
			}
			continue
		}

		slog.Warn("could not determine source layout for workspace package",
			"package", packageName,
			"path", pkgPath)
	}

	return nil
}

// copyDirectory recursively copies a directory
func (ib *IncrementalBuilder) copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip __pycache__ directories
		if info.IsDir() && info.Name() == "__pycache__" {
			return filepath.SkipDir
		}

		// Skip .pyc files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".pyc") {
			return nil
		}

		// Calculate relative path and destination
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}

		return os.Chmod(dstPath, info.Mode())
	})
}

// findSitePackagesPath finds the site-packages directory in the venv
func (ib *IncrementalBuilder) findSitePackagesPath(venvPath string) (string, error) {
	// Try common Python version paths
	libPath := filepath.Join(venvPath, "lib")
	entries, err := os.ReadDir(libPath)
	if err != nil {
		return "", fmt.Errorf("failed to read lib directory: %w", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "python") && entry.IsDir() {
			sitePackagesPath := filepath.Join(libPath, entry.Name(), "site-packages")
			if _, err := os.Stat(sitePackagesPath); err == nil {
				return sitePackagesPath, nil
			}
		}
	}

	return "", fmt.Errorf("site-packages directory not found in %s", venvPath)
}

// cleanupInstalledDependencies removes __pycache__ directories, .pyc files, and system files from installed dependencies
func (ib *IncrementalBuilder) cleanupInstalledDependencies(targetDir string) error {
	slog.Info("cleaning up __pycache__ directories, .pyc files, and system files from installed dependencies", "targetDir", targetDir)

	var removedItems []string
	var totalSize int64

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}

		// Skip the target directory itself
		if path == targetDir {
			return nil
		}

		// Remove __pycache__ directories
		if info.IsDir() && info.Name() == "__pycache__" {
			if err := os.RemoveAll(path); err != nil {
				slog.Warn("failed to remove __pycache__ directory", "path", path, "error", err)
			} else {
				removedItems = append(removedItems, path)
				slog.Debug("removed __pycache__ directory", "path", path)
			}
			return filepath.SkipDir // Skip walking into the removed directory
		}

		// Remove .pyc, .pyo, .pyd files and .DS_Store files
		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			fileName := info.Name()
			if ext == ".pyc" || ext == ".pyo" || ext == ".pyd" || fileName == ".DS_Store" {
				totalSize += info.Size()
				if err := os.Remove(path); err != nil {
					if fileName == ".DS_Store" {
						slog.Warn("failed to remove .DS_Store file", "path", path, "error", err)
					} else {
						slog.Warn("failed to remove compiled Python file", "path", path, "error", err)
					}
				} else {
					removedItems = append(removedItems, path)
					if fileName == ".DS_Store" {
						slog.Debug("removed .DS_Store file", "path", path, "size", info.Size())
					} else {
						slog.Debug("removed compiled Python file", "path", path, "size", info.Size())
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking target directory during cleanup: %w", err)
	}

	slog.Info("dependency cleanup completed",
		"targetDir", targetDir,
		"removedItems", len(removedItems),
		"totalSizeRemoved", totalSize)

	return nil
}

// getWorkspacePackageNames returns a list of workspace package names
func (ib *IncrementalBuilder) getWorkspacePackageNames(projectInfo *ProjectInfo) []string {
	var packages []string

	// Add the main package name
	packageName := "unknown"
	if projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			if config.Project.Name != "" {
				packageName = config.Project.Name
			} else if config.Tool.Poetry.Name != "" {
				packageName = config.Tool.Poetry.Name
			}
		}
	}

	if packageName != "" && packageName != "unknown" {
		packages = append(packages, packageName)
	}

	// Add workspace package names if available
	// This would need to be enhanced to read from pyproject.toml workspace members
	// For now, we'll use common patterns
	packages = append(packages, "core", "functions")

	return packages
}

// isWorkspacePackage checks if a package name is a workspace package
func (ib *IncrementalBuilder) isWorkspacePackage(packageName string, workspacePackages []string) bool {
	for _, wp := range workspacePackages {
		if packageName == wp || strings.HasPrefix(packageName, wp+"-") {
			return true
		}
	}
	return false
}

// shouldSkipPackage determines if a package should be skipped
func (ib *IncrementalBuilder) shouldSkipPackage(packageName string) bool {
	skipPackages := []string{
		"pip", "setuptools", "wheel", "_distutils_hack",
		"pkg_resources", "distutils-precedence.pth",
		"__pycache__",
	}

	for _, skip := range skipPackages {
		if strings.HasPrefix(packageName, skip) {
			return true
		}
	}

	return false
}
