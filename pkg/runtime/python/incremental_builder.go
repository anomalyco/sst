package python

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/path"
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

	// progressReporter tracks and reports build progress
	progressReporter *ProgressReporter

	// contentFilter handles file exclusion using hybrid approach
	contentFilter *ContentFilter

	// Removed fallback manager - using direct error handling instead

	// syncCompleted tracks if uv sync has been run for this build session
	syncCompleted map[string]bool // workspaceDir -> whether sync completed

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

	// DependencyAnalysis caches the dependency analysis to avoid re-analyzing
	DependencyAnalysis *DependencyAnalysis
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
		progressReporter:   progressReporter,
		contentFilter:      contentFilter,
		syncCompleted:      make(map[string]bool),
		config:             config,
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

	// Pass dependencies to executeBuildPlan so it can cache them in the result
	buildResult, err := ib.executeBuildPlan(ctx, input, projectInfo, buildPlan, dependencies)
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

	// Skip cache updates during deployment - they're only useful for dev mode
	// This saves ~17 seconds per function during deployment
	if input.Dev {
		if err := ib.updateCacheAfterBuild(input, projectInfo, buildResult); err != nil {
			slog.Warn("failed to update cache after build", "error", err)
		}
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
func (ib *IncrementalBuilder) executeBuildPlan(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, plan *BuildPlan, dependencies *DependencyAnalysis) (*BuildResult, error) {
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
		Success:            true,
		BuildPlan:          plan,
		PackagesBuilt:      []string{},
		PackagesCached:     []string{},
		Errors:             []string{},
		Warnings:           []string{},
		BuildDuration:      0,
		DependencyAnalysis: dependencies, // Cache for later use
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
	// For container builds, we also need dependencies installed since our simplified
	// Dockerfile just copies the pre-built artifacts (no multi-stage uv install)
	ib.progressReporter.UpdateProgress(StageBuildPackages, "Installing dependencies")
	if err := ib.installDependenciesForBuild(ctx, input, projectInfo, plan, result); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	// Create final build output (pass result which contains cached dependency analysis)
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

	// Wait for all builds to complete with timeout to prevent indefinite hangs
	// This is critical for CI/CD environments where a hung goroutine can block the entire pipeline
	slog.Info("waiting for all builds to complete in group", "goroutinesStarted", len(group))

	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	// Use a 10-minute timeout for the entire group - individual packages should complete much faster
	groupTimeout := 10 * time.Minute
	select {
	case <-waitDone:
		slog.Info("all builds completed, closing error channel")
	case <-time.After(groupTimeout):
		slog.Error("TIMEOUT: parallel build group timed out after 10 minutes - possible deadlock or hung process",
			"group", group)
		// Don't wait for goroutines - they may be stuck. Return error immediately.
		// The goroutines will eventually complete or be cleaned up when the process exits.
		return fmt.Errorf("parallel build group timed out after %v - check for hung uv commands or deadlocks", groupTimeout)
	case <-ctx.Done():
		slog.Error("context cancelled while waiting for parallel builds", "error", ctx.Err())
		return fmt.Errorf("context cancelled during parallel builds: %w", ctx.Err())
	}

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
	// Clean up any absolute path directories that may have been created by the cache system
	if err := ib.cleanupAbsolutePaths(input.Out()); err != nil {
		slog.Warn("failed to clean up absolute paths", "error", err)
	}

	// Adjust handler path based on project structure
	adjustedHandler, err := ib.adjustHandlerPath(input, projectInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to adjust handler path: %w", err)
	}

	// Handle Dockerfile for container builds
	if input.IsContainer {
		if err := ib.ensureDockerfile(input, projectInfo); err != nil {
			return nil, fmt.Errorf("failed to ensure Dockerfile: %w", err)
		}
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

// ensureDockerfile ensures a Dockerfile exists in the output directory for container builds.
// It first checks if a custom Dockerfile exists in the workspace directory, and if not,
// copies the default Python Dockerfile from the platform directory.
// It also exports requirements.txt for the Docker build.
func (ib *IncrementalBuilder) ensureDockerfile(input *runtime.BuildInput, projectInfo *ProjectInfo) error {
	outputDockerfile := filepath.Join(input.Out(), "Dockerfile")
	outputRequirements := filepath.Join(input.Out(), "requirements.txt")

	// Export requirements.txt for the Docker build
	if err := ib.exportRequirementsForContainer(input, projectInfo, outputRequirements); err != nil {
		return fmt.Errorf("failed to export requirements.txt: %w", err)
	}

	// Check if Dockerfile already exists in output
	if _, err := os.Stat(outputDockerfile); err == nil {
		slog.Info("Dockerfile already exists in output directory", "path", outputDockerfile)
		return nil
	}

	// Check if there's a custom Dockerfile in the project root directory
	projectRoot := projectInfo.ProjectRoot
	if projectRoot == "" {
		projectRoot = path.ResolveRootDir(input.CfgPath)
	}

	customDockerfile := filepath.Join(projectRoot, "Dockerfile")
	if _, err := os.Stat(customDockerfile); err == nil {
		slog.Info("copying custom Dockerfile from project root", "src", customDockerfile, "dest", outputDockerfile)
		return ib.copyFileSimple(customDockerfile, outputDockerfile)
	}

	// Check if there's a custom Dockerfile in the handler's package directory
	// (e.g., functions/Dockerfile when handler is functions/src/functions/api.handler)
	if projectInfo.PyprojectPath != "" {
		handlerPkgDir := filepath.Dir(projectInfo.PyprojectPath)
		if handlerPkgDir != projectRoot {
			handlerDockerfile := filepath.Join(handlerPkgDir, "Dockerfile")
			if _, err := os.Stat(handlerDockerfile); err == nil {
				slog.Info("copying custom Dockerfile from handler package directory", "src", handlerDockerfile, "dest", outputDockerfile)
				return ib.copyFileSimple(handlerDockerfile, outputDockerfile)
			}
		}
	}

	// Use the default Python Dockerfile from the platform directory
	defaultDockerfile := filepath.Join(path.ResolvePlatformDir(input.CfgPath), "dist", "dockerfiles", "python.Dockerfile")
	if _, err := os.Stat(defaultDockerfile); err != nil {
		return fmt.Errorf("default Python Dockerfile not found at %s: %w", defaultDockerfile, err)
	}

	slog.Info("copying default Python Dockerfile", "src", defaultDockerfile, "dest", outputDockerfile)
	return ib.copyFileSimple(defaultDockerfile, outputDockerfile)
}

// exportRequirementsForContainer exports dependencies to requirements.txt for container builds
func (ib *IncrementalBuilder) exportRequirementsForContainer(input *runtime.BuildInput, projectInfo *ProjectInfo, outputPath string) error {
	// Find the workspace root (where uv.lock is)
	workspaceRoot := projectInfo.ProjectRoot
	if workspaceRoot == "" {
		workspaceRoot = path.ResolveRootDir(input.CfgPath)
	}

	// Use uv export to generate requirements.txt
	ctx := context.Background()
	exportCmd := &UvExportCommand{
		WorkspaceDir:    workspaceRoot,
		OutputFile:      outputPath,
		NoEmitWorkspace: false, // Include workspace packages
		NoDev:           true,  // Exclude dev dependencies
		AllPackages:     true,  // Export all packages
	}

	if ib.uvRunner != nil {
		if err := ib.uvRunner.ExecuteExportCommand(ctx, exportCmd); err != nil {
			// If uv export fails, try to create a minimal requirements.txt from pyproject.toml
			slog.Warn("uv export failed, creating minimal requirements.txt", "error", err)
			return ib.createMinimalRequirements(projectInfo, outputPath)
		}
	} else {
		// No UV runner, create minimal requirements
		return ib.createMinimalRequirements(projectInfo, outputPath)
	}

	slog.Info("exported requirements.txt for container build", "path", outputPath)
	return nil
}

// createMinimalRequirements creates a minimal requirements.txt from pyproject.toml dependencies
func (ib *IncrementalBuilder) createMinimalRequirements(projectInfo *ProjectInfo, outputPath string) error {
	// Create an empty requirements.txt if we can't export
	// The container build will still work, just without pre-installed deps
	if err := os.WriteFile(outputPath, []byte("# Auto-generated requirements.txt\n"), 0644); err != nil {
		return fmt.Errorf("failed to create requirements.txt: %w", err)
	}
	slog.Warn("created empty requirements.txt - dependencies may need to be added manually")
	return nil
}

// copyFileSimple copies a single file from src to dest
func (ib *IncrementalBuilder) copyFileSimple(src, dest string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
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

// copyDirectoryUnfiltered copies a directory recursively without any content filtering.
// Used for workspace packages in container builds where pyproject.toml and other metadata
// files must be preserved for uv pip install to work inside the container.
func copyDirectoryUnfiltered(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Skip common non-essential directories
		if info.IsDir() {
			name := info.Name()
			if name == "__pycache__" || name == ".venv" || name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0644)
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
// For modern UV workspace projects, the handler path should match the artifact structure exactly.
// We only adjust for legacy patterns like src/ prefix flattening.
func (ib *IncrementalBuilder) adjustHandlerPath(input *runtime.BuildInput, projectInfo *ProjectInfo) (string, error) {
	originalHandler := input.Handler

	// Strip leading ./ if present
	handler := strings.TrimPrefix(originalHandler, "./")

	// Split handler into file path and function name
	// e.g., "functions/src/functions/api.handler" -> filePath="functions/src/functions/api", funcName="handler"
	lastDot := strings.LastIndex(handler, ".")
	if lastDot == -1 {
		return handler, nil
	}
	filePath := handler[:lastDot]
	funcName := handler[lastDot+1:]

	// When workspace packages use src/ layout (PEP 517), uv pip install flattens
	// src/pkg/ to pkg/ in the installed package. The handler path in sst.config.ts
	// points to the source tree (e.g., functions/src/functions/api) but the installed
	// package will be at functions/api.
	//
	// For container builds: dependencies are installed inside the Docker container,
	// so we can't check the artifact filesystem. Instead, detect the src/ pattern
	// and adjust proactively.
	//
	// For zip builds: both the raw source and installed package may exist in the
	// artifact. Python's import system finds the installed package first, so we
	// prefer the flattened path.
	parts := strings.Split(filePath, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "src" && i+1 < len(parts) {
			// Try removing "src/{next}" from the path
			adjusted := append([]string{}, parts[:i]...)
			adjusted = append(adjusted, parts[i+2:]...)
			adjustedPath := strings.Join(adjusted, "/")

			if input.IsContainer {
				// For containers, we can't check the artifact (deps installed in Docker).
				// Proactively flatten the src/ layout since uv pip install will do so.
				adjustedHandler := adjustedPath + "." + funcName
				slog.Info("handler path adjustment: src/ layout detected for container build",
					"original", originalHandler,
					"adjusted", adjustedHandler,
					"reason", "uv pip install in Docker will flatten src/ layout")
				return adjustedHandler, nil
			}

			// For zip builds, verify the flattened file exists in the artifact
			adjustedFile := filepath.Join(input.Out(), adjustedPath+".py")
			if _, err := os.Stat(adjustedFile); err == nil {
				adjustedHandler := adjustedPath + "." + funcName
				slog.Info("handler path adjustment: src/ layout flattened by package install",
					"original", originalHandler,
					"adjusted", adjustedHandler,
					"reason", "installed package takes precedence over raw source tree")
				return adjustedHandler, nil
			}
		}
	}

	// Check if the handler file exists at the expected path in the artifact
	expectedFile := filepath.Join(input.Out(), filePath+".py")
	if _, err := os.Stat(expectedFile); err == nil {
		slog.Info("handler path adjustment: file found at expected path", "handler", handler)
		return handler, nil
	}

	// No file found at either path, return original (stripped of ./)
	slog.Info("handler path adjustment: no change needed", "handler", handler)
	return handler, nil
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
		// Wheel files are zip archives containing the correctly structured package
		// We need to extract them to get the package with source remapping applied
		slog.Info("extracting wheel file to get correctly structured package", "wheel", archiveFile)

		// Extract the wheel (it's a zip file)
		extractCmd := []string{"unzip", "-o", "-q", archiveFile, "-d", outputDir}
		if err := ib.executeCommand(extractCmd, outputDir); err != nil {
			return fmt.Errorf("failed to extract wheel: %w", err)
		}

		// Remove the wheel file after extraction
		if err := os.Remove(archiveFile); err != nil {
			slog.Warn("failed to remove wheel file", "file", archiveFile, "error", err)
		}

		// Remove the .dist-info directory (not needed for Lambda)
		distInfoPattern := filepath.Join(outputDir, "*.dist-info")
		distInfoDirs, _ := filepath.Glob(distInfoPattern)
		for _, distInfoDir := range distInfoDirs {
			if err := os.RemoveAll(distInfoDir); err != nil {
				slog.Warn("failed to remove dist-info directory", "dir", distInfoDir, "error", err)
			}
		}

		slog.Info("wheel extracted successfully", "wheel", archiveFile)
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

	return &IncrementalBuilderStats{
		CacheStats:       cacheStats,
		BuildResultStats: *buildResultStats,
		Config:           ib.config,
	}
}

// IncrementalBuilderStats contains statistics about the incremental builder
type IncrementalBuilderStats struct {
	CacheStats       CacheStats               `json:"cacheStats"`
	BuildResultStats BuildResultCacheStats    `json:"buildResultStats"`
	Config           IncrementalBuilderConfig `json:"config"`
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

// installDependenciesForBuild installs dependencies for the build
func (ib *IncrementalBuilder) installDependenciesForBuild(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, plan *BuildPlan, result *BuildResult) error {
	// Ensure output directory exists (it may not if no packages needed building)
	if err := os.MkdirAll(input.Out(), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate requirements file - uses shared global file for all functions in same workspace
	requirementsFile := filepath.Join(input.Out(), "requirements.txt")
	if err := ib.generateOrCopyRequirementsFile(ctx, input, projectInfo, requirementsFile); err != nil {
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

// generateOrCopyRequirementsFile generates requirements.txt once per workspace in a shared location,
// then copies it to each function's output directory
func (ib *IncrementalBuilder) generateOrCopyRequirementsFile(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, outputFile string) error {
	// Use UV export to generate requirements - supports both all-packages and per-package modes
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
	workspaceRoot := workspaceDir // Default to current workspaceDir

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
				for currentDir != filepath.Dir(currentDir) && currentDir != "." && currentDir != projectInfo.SourceRoot {
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
						workspaceRoot = currentDir // Use the workspace root for uv export
						slog.Info("found parent pyproject.toml - using per-package dependency export",
							"package", packageName,
							"pyprojectPath", projectInfo.PyprojectPath,
							"parentPyproject", parentPyproject,
							"workspaceRoot", workspaceRoot)
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
							currentDir == filepath.Dir(currentDir), currentDir == "."))
				}
			}
		} else {
			slog.Warn("failed to parse pyproject.toml", "path", projectInfo.PyprojectPath, "error", err)
		}
	}

	slog.Info("generateRequirementsFile decision",
		"packageName", packageName,
		"useAllPackages", useAllPackages,
		"workspaceDir", workspaceDir,
		"workspaceRoot", workspaceRoot)

	exportCmd := &UvExportCommand{
		WorkspaceDir:    workspaceRoot, // Use workspace root where uv.lock is
		PackageName:     packageName,
		OutputFile:      outputFile,
		NoEmitWorkspace: false,
		NoDev:           noDev,
		AllPackages:     useAllPackages,
		NoEmitProject:   !useAllPackages, // When exporting for a specific package, don't emit the project itself
		NoEditable:      true,            // Export workspace packages as non-editable for proper installation
	}

	// Create cache key that includes package name for per-package exports
	// This ensures different handlers get their own requirements.txt
	cacheKey := workspaceRoot
	if packageName != "" {
		cacheKey = workspaceRoot + ":" + packageName
	}

	// Check if we've already generated requirements.txt for this workspace/package
	globalRequirementsFilesMutex.Lock()
	sharedRequirementsFile, exists := globalRequirementsFiles[cacheKey]
	globalRequirementsFilesMutex.Unlock()

	if exists {
		// Copy the shared requirements.txt to this function's output directory
		data, err := os.ReadFile(sharedRequirementsFile)
		if err != nil {
			return fmt.Errorf("failed to read shared requirements.txt: %w", err)
		}
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write requirements.txt: %w", err)
		}
		slog.Info("copied shared requirements.txt",
			"from", sharedRequirementsFile,
			"to", outputFile,
			"size", len(data),
			"hash", fmt.Sprintf("%x", sha256.Sum256(data))[:16],
			"cacheKey", cacheKey)
		return nil
	}

	// First function for this workspace/package - generate requirements.txt
	if err := ib.uvRunner.ExecuteExportCommand(ctx, exportCmd); err != nil {
		return err
	}

	// Verify the generated file
	if data, err := os.ReadFile(outputFile); err == nil {
		slog.Info("generated shared requirements.txt",
			"file", outputFile,
			"size", len(data),
			"hash", fmt.Sprintf("%x", sha256.Sum256(data))[:16],
			"cacheKey", cacheKey)
	}

	// Store this as the shared requirements.txt for this workspace/package
	globalRequirementsFilesMutex.Lock()
	globalRequirementsFiles[cacheKey] = outputFile
	globalRequirementsFilesMutex.Unlock()

	return nil
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

	// Skip uv sync for deployment - we use uv export + uv pip install instead
	// uv sync tries to build workspace packages as wheels, which fails for handler packages
	// The export already resolves all dependencies including git deps
	slog.Info("skipping uv sync for deployment (using export + pip install instead)",
		"workspaceDir", workspaceDir)

	// Simplified approach: copy source files based on handler path
	ib.progressReporter.UpdateProgress(StageBuildPackages, "Copying source files")
	slog.Info("copying source files based on handler path", "handler", input.Handler)

	if err := ib.copySourceFilesSimple(ctx, input, projectInfo); err != nil {
		return fmt.Errorf("failed to copy source files: %w", err)
	}

	// Copy the synced packages from .venv to output directory
	// This includes git dependencies like sst that are properly resolved after sync
	// For container builds, skip dependency installation — the Dockerfile handles it
	// inside the container to ensure native binaries are compiled for Linux.
	// Instead, copy workspace package directories so the Dockerfile can install them.
	if input.IsContainer {
		slog.Info("skipping copySyncedDependencies for container build — Dockerfile will install dependencies",
			"functionID", input.FunctionID)

		// Copy workspace package directories into the artifact so the Dockerfile's
		// `uv pip install -r requirements.txt` can resolve relative paths like ./core
		if err := ib.copyWorkspacePackagesForContainer(input, projectInfo); err != nil {
			return fmt.Errorf("failed to copy workspace packages for container: %w", err)
		}
	} else {
		ib.progressReporter.UpdateProgress(StageBuildPackages, "Copying synced dependencies")

		if err := ib.copySyncedDependencies(ctx, input, projectInfo, architecture); err != nil {
			return fmt.Errorf("failed to copy synced dependencies: %w", err)
		}
	}

	return nil
}

// copyWorkspacePackagesForContainer copies workspace package directories into the artifact
// so the Dockerfile's `uv pip install -r requirements.txt` can resolve relative paths like ./core.
func (ib *IncrementalBuilder) copyWorkspacePackagesForContainer(input *runtime.BuildInput, projectInfo *ProjectInfo) error {
	// Find the workspace root
	workspaceRoot := projectInfo.ProjectRoot
	if workspaceRoot == "" {
		workspaceRoot = path.ResolveRootDir(input.CfgPath)
	}

	// Read the requirements.txt from the artifact to find workspace package paths
	requirementsPath := filepath.Join(input.Out(), "requirements.txt")
	content, err := os.ReadFile(requirementsPath)
	if err != nil {
		slog.Info("no requirements.txt found for container build, skipping workspace package copy")
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var copied []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comments, empty lines, and non-local-path lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		// Detect local path references: ./core, ./functions, etc.
		if !strings.HasPrefix(line, "./") && !strings.HasPrefix(line, "../") {
			continue
		}

		// Strip any extras or markers (e.g., "./core[extra] ; python_version >= '3.11'")
		pkgPath := line
		for _, sep := range []string{" ", "[", ";"} {
			if idx := strings.Index(pkgPath, sep); idx > 0 {
				pkgPath = pkgPath[:idx]
			}
		}

		// Resolve the full path relative to workspace root
		fullPath := filepath.Join(workspaceRoot, pkgPath)
		if _, err := os.Stat(fullPath); err != nil {
			slog.Warn("workspace package directory not found", "path", fullPath, "line", line)
			continue
		}

		// Copy to the artifact at the same relative path
		destPath := filepath.Join(input.Out(), pkgPath)
		if _, err := os.Stat(destPath); err == nil {
			// Directory already exists (e.g., copySourceFilesSimple already copied handler source files).
			// Ensure pyproject.toml is present — it's needed for uv pip install but may have been
			// filtered out by the content filter during source file copying.
			srcPyproject := filepath.Join(fullPath, "pyproject.toml")
			destPyproject := filepath.Join(destPath, "pyproject.toml")
			if _, err := os.Stat(srcPyproject); err == nil {
				if _, err := os.Stat(destPyproject); err != nil {
					slog.Info("copying missing pyproject.toml into existing workspace package", "path", pkgPath)
					data, readErr := os.ReadFile(srcPyproject)
					if readErr != nil {
						return fmt.Errorf("failed to read pyproject.toml for workspace package %s: %w", pkgPath, readErr)
					}
					if err := os.WriteFile(destPyproject, data, 0644); err != nil {
						return fmt.Errorf("failed to copy pyproject.toml for workspace package %s: %w", pkgPath, err)
					}
				}
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for workspace package %s: %w", pkgPath, err)
		}

		// Use unfiltered copy — workspace packages need pyproject.toml and all
		// metadata files for uv pip install to work inside the container.
		// copyDirectoryRecursive uses ContentFilter which strips pyproject.toml.
		if err := copyDirectoryUnfiltered(fullPath, destPath); err != nil {
			return fmt.Errorf("failed to copy workspace package %s: %w", pkgPath, err)
		}

		copied = append(copied, pkgPath)
		slog.Info("copied workspace package for container build", "path", pkgPath)
	}

	if len(copied) > 0 {
		slog.Info("copied workspace packages for container build", "packages", copied)
	}

	return nil
}

// copySourceFilesSimple copies source files using a simple, handler-path-based approach
// NOTE: This function is only needed for projects that don't use UV workspace packages.
// For UV workspace projects, copyWorkspacePackageSources handles copying the handler's package.
func (ib *IncrementalBuilder) copySourceFilesSimple(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo) error {
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	// NOTE: We always copy handler source files here. Workspace packages (like backend/)
	// are installed via uv pip install from the .deps cache, so there's no duplication.
	// The handler source (functions/, etc.) must be copied separately.

	slog.Info("copySourceFilesSimple ENTRY",
		"handler", input.Handler,
		"workspaceDir", workspaceDir,
		"sourceRoot", projectInfo.SourceRoot,
		"projectRoot", projectInfo.ProjectRoot,
		"pyprojectPath", projectInfo.PyprojectPath)

	// Parse the handler path to understand what directories we need
	handlerPath := input.Handler

	// Make handler path relative to workspaceDir if the handler path starts with a prefix
	// that matches the relative path from project root to workspaceDir
	// e.g., if handler is "packages/api/auth/handler.main" and workspaceDir is "/project/packages/api"
	// then we need to strip "packages/api/" from the handler path to get "auth/handler.main"
	if projectInfo.ProjectRoot != "" && workspaceDir != projectInfo.ProjectRoot {
		relWorkspacePath, err := filepath.Rel(projectInfo.ProjectRoot, workspaceDir)
		if err == nil && relWorkspacePath != "." {
			// Normalize to forward slashes for comparison
			relWorkspacePath = filepath.ToSlash(relWorkspacePath)
			prefix := relWorkspacePath + "/"
			if strings.HasPrefix(handlerPath, prefix) {
				handlerPath = strings.TrimPrefix(handlerPath, prefix)
				slog.Info("adjusted handler path relative to workspace", "originalHandler", input.Handler, "adjustedHandler", handlerPath, "strippedPrefix", prefix)
			}
		}
	}

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
		// The last part is "filename.function_name", so we need to extract just the filename
		lastPart := parts[len(parts)-1]
		fileName := strings.Split(lastPart, ".")[0] // Get filename without function name
		handlerFile := strings.Join(append(parts[:len(parts)-1], fileName), "/") + ".py"
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

	// If we stripped a prefix from the handler path, we need to preserve that structure in the output
	// so Lambda can find the handler at the expected path
	var outputPrefix string
	if projectInfo.ProjectRoot != "" && workspaceDir != projectInfo.ProjectRoot {
		relWorkspacePath, err := filepath.Rel(projectInfo.ProjectRoot, workspaceDir)
		if err == nil && relWorkspacePath != "." {
			// Check if the original handler starts with this prefix
			relWorkspacePath = filepath.ToSlash(relWorkspacePath)
			prefix := relWorkspacePath + "/"
			if strings.HasPrefix(input.Handler, prefix) {
				outputPrefix = relWorkspacePath
			}
		}
	}

	// Copy directories
	for _, dirName := range dirsToInclude {
		srcPath := filepath.Join(workspaceDir, dirName)
		var destPath string
		if outputPrefix != "" {
			// Preserve the directory structure that Lambda expects
			destPath = filepath.Join(input.Out(), outputPrefix, dirName)
		} else {
			destPath = filepath.Join(input.Out(), dirName)
		}

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		slog.Info("copying directory", "src", srcPath, "dest", destPath, "outputPrefix", outputPrefix)

		if err := ib.copyDirectoryRecursive(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy directory %s: %w", dirName, err)
		}
	}

	// Copy root Python files
	for _, fileName := range rootFilesToInclude {
		srcPath := filepath.Join(workspaceDir, fileName)
		var destPath string
		if outputPrefix != "" {
			// Preserve the directory structure that Lambda expects
			destPath = filepath.Join(input.Out(), outputPrefix, fileName)
		} else {
			destPath = filepath.Join(input.Out(), fileName)
		}

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		slog.Info("copying root file", "src", srcPath, "dest", destPath, "outputPrefix", outputPrefix)

		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", fileName, err)
		}
	}

	slog.Info("successfully copied source files",
		"directories", dirsToInclude,
		"rootFiles", rootFilesToInclude)

	return nil
}

// copySyncedDependencies installs all dependencies (external + workspace packages) with correct platform targeting
func (ib *IncrementalBuilder) copySyncedDependencies(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, architecture string) error {
	startTime := time.Now()
	slog.Info("⏱️ copySyncedDependencies START", "functionID", input.FunctionID)

	// Use the requirements.txt that was already exported by the export step
	// With --no-editable, workspace packages appear as ./path instead of -e ./path
	// and uv pip install handles them correctly
	requirementsPath := filepath.Join(input.Out(), "requirements.txt")

	// Check if requirements.txt exists
	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		slog.Warn("requirements.txt not found, skipping dependency installation", "path", requirementsPath)
		return nil
	}

	// Get workspace root directory - find the UV workspace root (pyproject.toml with [tool.uv.workspace])
	// This is needed because requirements.txt contains relative paths like ./backend_pkg
	// that are relative to the workspace root
	workspaceRoot := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)
		workspaceRoot = pyprojectDir

		// Walk up the directory tree looking for UV workspace root
		currentDir := pyprojectDir
		for {
			parentDir := filepath.Dir(currentDir)
			if parentDir == currentDir || parentDir == "." {
				break
			}
			parentPyproject := filepath.Join(parentDir, "pyproject.toml")
			if _, err := os.Stat(parentPyproject); err == nil {
				if config, parseErr := ib.projectResolver.ParsePyprojectToml(parentPyproject); parseErr == nil {
					if len(config.Tool.UV.Workspace.Members) > 0 {
						workspaceRoot = parentDir
						slog.Debug("found UV workspace root", "parentDir", parentDir)
						break
					}
				}
			}
			currentDir = parentDir
		}
	}
	slog.Debug("determined workspace root for dependency installation", "workspaceRoot", workspaceRoot)

	// Filter only boto3/botocore (Lambda runtime packages), keep everything else including workspace packages
	filteredRequirementsPath := filepath.Join(input.Out(), "requirements-filtered.txt")
	_, err := ib.filterWorkspacePackagesFromRequirementsAndGetPaths(requirementsPath, filteredRequirementsPath, projectInfo, workspaceRoot)
	if err != nil {
		return fmt.Errorf("failed to filter requirements: %w", err)
	}
	requirementsPath = filteredRequirementsPath

	// Calculate cache key from requirements hash + architecture
	requirementsHash, err := ib.hashFile(requirementsPath)
	var cacheKey string
	var depsCacheDir string

	if err == nil {
		cacheKey = fmt.Sprintf("%s-%s", requirementsHash, architecture)
		depsCacheDir = filepath.Join(filepath.Dir(input.Out()), ".deps", cacheKey)

		// Log cache key for debugging
		if data, err := os.ReadFile(requirementsPath); err == nil {
			lines := strings.Split(string(data), "\n")
			firstLine := "empty"
			if len(lines) > 0 && lines[0] != "" {
				firstLine = lines[0]
				if len(firstLine) > 50 {
					firstLine = firstLine[:50] + "..."
				}
			}
			ib.progressReporter.UpdateProgress(StageBuildPackages, fmt.Sprintf("🔑 Key: %s | First: %s", cacheKey[:16], firstLine))
		}

		// Get or create a lock for this cache key to prevent concurrent installs
		globalDependencyInstallLocksMutex.Lock()
		cacheLock, exists := globalDependencyInstallLocks[cacheKey]
		if !exists {
			cacheLock = &sync.Mutex{}
			globalDependencyInstallLocks[cacheKey] = cacheLock
			ib.progressReporter.UpdateProgress(StageBuildPackages, "🔒 Created install lock")
		} else {
			ib.progressReporter.UpdateProgress(StageBuildPackages, "🔒 Waiting for install lock...")
		}
		globalDependencyInstallLocksMutex.Unlock()

		// Acquire lock with timeout (5 minutes)
		lockDeadline := time.Now().Add(5 * time.Minute)
		gotLock := false
		for time.Now().Before(lockDeadline) {
			if cacheLock.TryLock() {
				gotLock = true
				break
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled while waiting for dependency install lock")
			case <-time.After(500 * time.Millisecond):
				// Try again
			}
		}
		if !gotLock {
			return fmt.Errorf("timed out waiting for dependency install lock after 5 minutes")
		}
		defer cacheLock.Unlock()
		ib.progressReporter.UpdateProgress(StageBuildPackages, "✅ Got install lock")

		// Check if disk cache exists (from this deploy or previous deploy)
		if entries, err := os.ReadDir(depsCacheDir); err == nil && len(entries) > 0 {
			slog.Info("disk cache hit", "depsCacheDir", depsCacheDir, "entries", len(entries))
			ib.progressReporter.UpdateProgress(StageBuildPackages, "⚡ CACHE HIT! Copying from disk cache...")

			if err := ib.copyDependencyPackages(depsCacheDir, input.Out()); err != nil {
				slog.Warn("failed to copy from disk cache, will reinstall", "error", err)
				// Remove bad cache and continue to reinstall
				os.RemoveAll(depsCacheDir)
			} else {
				totalDuration := time.Since(startTime)
				slog.Info("✅ copySyncedDependencies COMPLETE (disk cache hit)", "functionID", input.FunctionID, "elapsed", totalDuration)
				ib.progressReporter.UpdateProgress(StageBuildPackages, fmt.Sprintf("✅ Copied from cache (%v)", totalDuration))
				return nil
			}
		}

		// Cache miss - create the cache directory
		ib.progressReporter.UpdateProgress(StageBuildPackages, "📦 CACHE MISS - Installing dependencies...")
		if err := os.MkdirAll(depsCacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create deps cache directory: %w", err)
		}
		slog.Info("using dedicated deps cache directory", "depsCacheDir", depsCacheDir, "cacheKey", cacheKey)
	} else {
		// Fallback to installing directly to output (shouldn't happen normally)
		depsCacheDir = input.Out()
	}

	// Workspace packages are installed by uv pip install (with --no-editable export).
	// We use --reinstall-package for each workspace package to force uv to rebuild them
	// from source every time, bypassing uv's cache. Without this, uv caches local packages
	// by name+version from pyproject.toml and serves stale builds when source files change.
	workspacePackages := ib.getWorkspacePackageNames(projectInfo)

	slog.Info("installing dependencies using uv pip install",
		"requirementsFile", requirementsPath,
		"targetDir", depsCacheDir,
		"architecture", architecture,
		"workspacePackages", workspacePackages)

	// Build the uv pip install command with platform targeting (same as legacy builder)
	// Note: We don't use --reinstall because it forces re-fetching git dependencies every time,
	// which is very slow for packages like sst that come from large git repos.
	// Instead, we use --reinstall-package for workspace packages only.
	// Install to depsCacheDir (dedicated cache directory) instead of input.Out()
	args := []string{"pip", "install", "-r", requirementsPath, "--target", depsCacheDir}

	// Force reinstall of workspace packages to avoid stale uv cache
	for _, pkg := range workspacePackages {
		args = append(args, "--reinstall-package", pkg)
	}

	// Add platform targeting for Lambda deployments
	// Use platform targeting for actual deployments (not dev mode, not containers)
	// In dev mode, we need native binaries for the local machine
	//
	// OPTIMIZATION: Skip platform targeting if we're already on the target platform
	// This allows UV to use native wheels from cache instead of cross-compiling
	if !input.Dev && !input.IsContainer {
		// Determine if we need cross-platform targeting
		needsCrossPlatform := false
		currentArch := goruntime.GOARCH // "amd64" or "arm64"
		currentOS := goruntime.GOOS     // "linux", "darwin", etc.

		// We need cross-platform targeting if:
		// 1. We're not on Linux (Lambda runs on Linux)
		// 2. Our architecture doesn't match the target
		targetIsArm := architecture == "arm64"
		currentIsArm := currentArch == "arm64"

		if currentOS != "linux" || targetIsArm != currentIsArm {
			needsCrossPlatform = true
		}

		if needsCrossPlatform {
			// Use manylinux platform tags to ensure GLIBC compatibility with Lambda.
			// Lambda python3.11 and below run on Amazon Linux 2 (GLIBC 2.17 = manylinux2014).
			// Lambda python3.12 and above run on Amazon Linux 2023 (GLIBC 2.28 = manylinux_2_28).
			// Using the correct manylinux tag prevents uv from downloading wheels
			// built against a newer GLIBC than the Lambda runtime provides.
			pythonVersion := strings.TrimPrefix(input.Runtime, "python")
			if pythonVersion == "" || pythonVersion == input.Runtime {
				pythonVersion = "3.13" // fallback to 3.13 if parsing fails
			}

			// Determine manylinux tag based on Python minor version
			// Parse minor version: "3.11" -> 11, "3.12" -> 12
			manylinuxTag := "manylinux_2_28" // AL2023 default (3.12+)
			if parts := strings.SplitN(pythonVersion, ".", 2); len(parts) == 2 {
				if minor, err := strconv.Atoi(parts[1]); err == nil && minor <= 11 {
					manylinuxTag = "manylinux2014" // AL2 (3.9-3.11)
				}
			}

			archPrefix := "x86_64"
			if architecture == "arm64" {
				archPrefix = "aarch64"
			}
			pythonPlatform := archPrefix + "-" + manylinuxTag

			args = append(args, "--python-platform", pythonPlatform, "--python-version", pythonVersion)
			slog.Info("using platform targeting for Lambda deployment (cross-platform build)",
				"platform", pythonPlatform,
				"manylinuxTag", manylinuxTag,
				"pythonVersion", pythonVersion,
				"runtime", input.Runtime,
				"currentOS", currentOS,
				"currentArch", currentArch)
		} else {
			slog.Info("skipping platform targeting - native build on matching platform",
				"targetArch", architecture,
				"currentOS", currentOS,
				"currentArch", currentArch)
		}
	} else if input.Dev {
		slog.Info("skipping platform targeting for dev mode (using native binaries)")
	}

	// Run uv pip install from the WORKSPACE ROOT, not the handler's package directory
	// This is critical because requirements.txt contains relative paths like ./vendored_sst
	// that are relative to the workspace root (where uv export was run)
	installWorkspaceDir := workspaceRoot

	// Execute the uv pip install command with timeout
	// Run from the workspace directory where pyproject.toml exists, not the artifact directory
	// Use a 15-minute timeout to allow for large dependency installations
	installCtx, installCancel := context.WithTimeout(ctx, 15*time.Minute)
	defer installCancel()

	installCmd := process.CommandContext(installCtx, "uv", args...)
	installCmd.Dir = installWorkspaceDir

	slog.Info("starting uv pip install", "command", strings.Join(args, " "), "workingDir", installWorkspaceDir)
	ib.progressReporter.UpdateProgress(StageBuildPackages, "📦 Running uv pip install (this may take a while)...")

	// Use a channel to handle timeout properly since CombinedOutput blocks
	type cmdResult struct {
		output []byte
		err    error
	}
	resultChan := make(chan cmdResult, 1)

	// Start a ticker to log progress every 30 seconds - helps diagnose hangs in CI/CD
	installStartTime := time.Now()
	progressDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-progressDone:
				return
			case <-ticker.C:
				elapsed := time.Since(installStartTime)
				slog.Info("uv pip install still running",
					"elapsed", elapsed,
					"functionID", input.FunctionID,
					"workingDir", installWorkspaceDir)
				ib.progressReporter.UpdateProgress(StageBuildPackages,
					fmt.Sprintf("📦 uv pip install running... (%v elapsed)", elapsed.Round(time.Second)))
			}
		}
	}()

	go func() {
		output, err := installCmd.CombinedOutput()
		resultChan <- cmdResult{output, err}
	}()

	var installOutput []byte
	select {
	case result := <-resultChan:
		close(progressDone) // Stop the progress ticker
		installOutput = result.output
		err = result.err
	case <-installCtx.Done():
		close(progressDone) // Stop the progress ticker
		// Kill the process if it's still running
		if installCmd.Process != nil {
			installCmd.Process.Kill()
		}
		// Remove partial cache on timeout
		if cacheKey != "" {
			os.RemoveAll(depsCacheDir)
		}
		return fmt.Errorf("uv pip install timed out after 15 minutes - check network connectivity and try again")
	}

	if err != nil {
		slog.Error("uv pip install failed",
			"command", strings.Join(installCmd.Args, " "),
			"error", err,
			"output", string(installOutput),
			"functionID", input.FunctionID,
			"handler", input.Handler,
			"workingDir", installWorkspaceDir,
			"pyprojectPath", projectInfo.PyprojectPath)
		// Remove partial cache on failure
		if cacheKey != "" {
			os.RemoveAll(depsCacheDir)
		}
		return fmt.Errorf("failed to run uv pip install: %v\n%s\n\nFunction: %s\nHandler: %s\nWorking directory: %s\nPyproject path: %s",
			err, string(installOutput), input.FunctionID, input.Handler, installWorkspaceDir, projectInfo.PyprojectPath)
	}

	slog.Info("uv pip install completed successfully", "output", string(installOutput))

	// Clean up __pycache__ directories and .pyc files from installed dependencies
	// Also removes boto3/botocore (Lambda provides them) unless user opts in
	if err := ib.cleanupInstalledDependencies(depsCacheDir, projectInfo); err != nil {
		slog.Warn("failed to clean up installed dependencies", "error", err)
		// Don't fail the build for cleanup issues, just warn
	}

	// Copy dependencies from cache directory to function artifact
	if err := ib.copyDependencyPackages(depsCacheDir, input.Out()); err != nil {
		return fmt.Errorf("failed to copy dependencies to artifact: %w", err)
	}

	// Clean up the filtered requirements file
	os.Remove(filteredRequirementsPath)

	totalDuration := time.Since(startTime)
	slog.Info("✅ copySyncedDependencies COMPLETE (fresh install)", "functionID", input.FunctionID, "elapsed", totalDuration)
	ib.progressReporter.UpdateProgress(StageBuildPackages, fmt.Sprintf("✅ Installed dependencies (%v)", totalDuration))

	return nil
}

// filterWorkspacePackagesFromRequirementsAndGetPaths filters out workspace packages from requirements.txt
// and returns the paths of the filtered workspace packages for source copying.
// By default, it also filters out boto3/botocore since Lambda provides them.
// To include them, set [tool.sst] include-lambda-runtime = true in pyproject.toml
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

	// Check if user wants to include Lambda runtime packages (boto3/botocore)
	// Default is to exclude them since Lambda provides them
	includeLambdaRuntime := false
	if projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			includeLambdaRuntime = config.Tool.SST.IncludeLambdaRuntime
		}
	}

	// Lambda runtime packages to exclude by default (Lambda provides these)
	lambdaRuntimePackages := []string{"boto3", "botocore"}

	// Track if we're skipping a multi-line requirement (package with continuation lines)
	skippingMultiLine := false
	lambdaPackagesFiltered := 0

	for _, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)

		// Check if this is a continuation line (starts with whitespace and contains \)
		isContinuation := len(originalLine) > 0 && (originalLine[0] == ' ' || originalLine[0] == '\t')

		// If we're skipping a multi-line requirement, skip continuation lines too
		if skippingMultiLine {
			if isContinuation || strings.HasPrefix(line, "--hash=") {
				slog.Debug("skipping continuation line", "line", line)
				continue
			} else {
				// End of multi-line requirement
				skippingMultiLine = false
			}
		}

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			filteredLines = append(filteredLines, originalLine)
			continue
		}

		// Check if this line should be filtered
		shouldSkip := false

		// Skip all editable installs that reference local paths
		// Editable installs create symlinks which won't work in Lambda
		// We need to copy these manually
		if strings.HasPrefix(line, "-e ") {
			editablePath := strings.TrimSpace(strings.TrimPrefix(line, "-e "))

			// Filter out any editable install that references a local path
			if strings.HasPrefix(editablePath, "./") || strings.HasPrefix(editablePath, "../") ||
				strings.HasPrefix(editablePath, "/") && !strings.Contains(editablePath, "://") {
				slog.Debug("filtering out editable local path from requirements (creates symlinks)", "line", line)
				// Record the path for source copying
				fullPath := filepath.Join(workspaceRoot, editablePath)
				if strings.HasPrefix(editablePath, "./") {
					fullPath = filepath.Join(workspaceRoot, strings.TrimPrefix(editablePath, "./"))
				}
				workspacePackagePaths = append(workspacePackagePaths, fullPath)
				continue
			}
		}

		// NON-editable local path references (./path, ../path) are handled by uv pip install
		// Don't filter them out - uv knows how to install packages from local paths correctly
		// This is cleaner than trying to manually copy with all the layout detection complexity

		// file:// URLs are also handled by uv pip install - don't filter them

		// Absolute local paths are also handled by uv pip install - don't filter them

		// Filter out Lambda runtime packages (boto3/botocore) unless user opts in
		// These are provided by Lambda runtime, so including them wastes ~23MB per function
		if !shouldSkip && !includeLambdaRuntime {
			// Extract package name from requirement line (e.g., "boto3==1.34.0" -> "boto3")
			pkgName := line
			// Handle version specifiers
			for _, sep := range []string{"==", ">=", "<=", "!=", "~=", ">", "<", "[", " "} {
				if idx := strings.Index(pkgName, sep); idx > 0 {
					pkgName = pkgName[:idx]
				}
			}
			pkgName = strings.TrimSpace(pkgName)

			for _, lambdaPkg := range lambdaRuntimePackages {
				if strings.EqualFold(pkgName, lambdaPkg) {
					slog.Debug("filtering out Lambda runtime package from requirements",
						"line", line,
						"package", lambdaPkg,
						"reason", "provided by Lambda runtime")
					shouldSkip = true
					lambdaPackagesFiltered++
					break
				}
			}
		}

		if shouldSkip {
			// Check if this is a multi-line requirement (ends with \)
			if strings.HasSuffix(line, "\\") {
				skippingMultiLine = true
				slog.Debug("starting to skip multi-line requirement", "line", line)
			}
			continue
		}

		// Include the line
		filteredLines = append(filteredLines, originalLine)
	}

	// Write the filtered requirements.txt
	filteredContent := strings.Join(filteredLines, "\n")
	if err := os.WriteFile(outputPath, []byte(filteredContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write filtered requirements file: %w", err)
	}

	slog.Info("filtered packages from requirements.txt",
		"originalLines", len(lines),
		"filteredLines", len(filteredLines),
		"workspacePackages", workspacePackages,
		"workspacePackagePaths", workspacePackagePaths,
		"lambdaRuntimePackagesFiltered", lambdaPackagesFiltered,
		"includeLambdaRuntime", includeLambdaRuntime)

	return workspacePackagePaths, nil
}

// copyDirectory recursively copies a directory
// filterPrefix is prepended to relative paths when checking the content filter
// (e.g., when copying "backend/", pass "backend" so "tests/" matches "backend/tests/**" patterns)
func (ib *IncrementalBuilder) copyDirectory(src, dst string, filterPrefix ...string) error {
	prefix := ""
	if len(filterPrefix) > 0 && filterPrefix[0] != "" {
		prefix = filterPrefix[0]
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path for filtering
		relPath, _ := filepath.Rel(src, path)

		// Skip __pycache__ directories
		if info.IsDir() && info.Name() == "__pycache__" {
			return filepath.SkipDir
		}

		// Skip .pyc files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".pyc") {
			return nil
		}

		// Use ContentFilter to determine if file/directory should be excluded
		if ib.contentFilter != nil {
			// Build the path to check against filter patterns
			// If prefix is set, prepend it so "tests/" matches "backend/tests/**"
			filterPath := relPath
			if prefix != "" && relPath != "." {
				filterPath = filepath.Join(prefix, relPath)
			}

			if ib.contentFilter.ShouldExclude(filterPath) {
				if info.IsDir() {
					slog.Debug("skipping directory by content filter in copyDirectory", "path", relPath, "filterPath", filterPath)
					return filepath.SkipDir
				}
				slog.Debug("skipping file by content filter in copyDirectory", "path", relPath, "filterPath", filterPath)
				return nil
			}
		}

		// Calculate relative path and destination
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

// cleanupInstalledDependencies removes __pycache__ directories, .pyc files, and system files from installed dependencies
// It also removes boto3/botocore packages since Lambda provides them (saves ~22MB per function)
func (ib *IncrementalBuilder) cleanupInstalledDependencies(targetDir string, projectInfo *ProjectInfo) error {
	slog.Info("cleaning up __pycache__ directories, .pyc files, and system files from installed dependencies", "targetDir", targetDir)

	// Check if user wants to include Lambda runtime packages (boto3/botocore)
	// Default is to exclude them since Lambda provides them
	includeLambdaRuntime := false
	if projectInfo != nil && projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			includeLambdaRuntime = config.Tool.SST.IncludeLambdaRuntime
		}
	}

	// Remove boto3/botocore if present and user hasn't opted in (Lambda provides these, saves ~22MB)
	// These packages come in as transitive dependencies (e.g., aioboto3 -> aiobotocore -> botocore)
	if !includeLambdaRuntime {
		lambdaRuntimePackages := []string{"boto3", "botocore"}
		for _, pkg := range lambdaRuntimePackages {
			pkgDir := filepath.Join(targetDir, pkg)
			if info, err := os.Stat(pkgDir); err == nil && info.IsDir() {
				if err := os.RemoveAll(pkgDir); err != nil {
					slog.Warn("failed to remove Lambda runtime package", "package", pkg, "error", err)
				} else {
					slog.Info("removed Lambda runtime package (provided by Lambda)", "package", pkg)
				}
			}
			// Also remove the dist-info directory
			entries, _ := os.ReadDir(targetDir)
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), pkg+"-") && strings.HasSuffix(entry.Name(), ".dist-info") {
					distInfoDir := filepath.Join(targetDir, entry.Name())
					if err := os.RemoveAll(distInfoDir); err != nil {
						slog.Warn("failed to remove dist-info directory", "dir", entry.Name(), "error", err)
					} else {
						slog.Debug("removed dist-info directory", "dir", entry.Name())
					}
				}
			}
		}
	}

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
			// Remove .pyi type stub files and py.typed markers (only useful for IDE, not runtime)
			if ext == ".pyi" || fileName == "py.typed" {
				totalSize += info.Size()
				if err := os.Remove(path); err != nil {
					slog.Warn("failed to remove type stub file", "path", path, "error", err)
				} else {
					removedItems = append(removedItems, path)
					slog.Debug("removed type stub file", "path", path, "size", info.Size())
				}
			}
		}

		// Remove .dist-info directories (pip metadata, not needed at runtime)
		if info.IsDir() && strings.HasSuffix(info.Name(), ".dist-info") {
			dirSize := int64(0)
			filepath.Walk(path, func(_ string, fi os.FileInfo, _ error) error {
				if fi != nil && !fi.IsDir() {
					dirSize += fi.Size()
				}
				return nil
			})
			totalSize += dirSize
			if err := os.RemoveAll(path); err != nil {
				slog.Warn("failed to remove .dist-info directory", "path", path, "error", err)
			} else {
				removedItems = append(removedItems, path)
				slog.Debug("removed .dist-info directory", "path", path, "size", dirSize)
			}
			return filepath.SkipDir
		}

		// Remove test directories (e.g., Crypto/SelfTest, tests/, test/)
		if info.IsDir() {
			dirName := info.Name()
			if dirName == "SelfTest" || dirName == "tests" || dirName == "test" {
				dirSize := int64(0)
				filepath.Walk(path, func(_ string, fi os.FileInfo, _ error) error {
					if fi != nil && !fi.IsDir() {
						dirSize += fi.Size()
					}
					return nil
				})
				totalSize += dirSize
				if err := os.RemoveAll(path); err != nil {
					slog.Warn("failed to remove test directory", "path", path, "error", err)
				} else {
					removedItems = append(removedItems, path)
					slog.Debug("removed test directory", "path", path, "size", dirSize)
				}
				return filepath.SkipDir
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
// getWorkspacePackageNames returns a list of workspace package names by reading the workspace root's pyproject.toml
func (ib *IncrementalBuilder) getWorkspacePackageNames(projectInfo *ProjectInfo) []string {
	var packages []string

	// Add the main package name from the service's pyproject.toml
	if projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			if config.Project.Name != "" {
				packages = append(packages, config.Project.Name)
			} else if config.Tool.Poetry.Name != "" {
				packages = append(packages, config.Tool.Poetry.Name)
			}

			// Also add any workspace packages referenced via { workspace = true }
			for name, source := range config.Tool.UV.Sources {
				if source.Workspace {
					packages = append(packages, name)
				}
			}
		}
	}

	return packages
}

// hashFile computes a hash of a file's contents for cache key generation
func (ib *IncrementalBuilder) hashFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file for hashing: %w", err)
	}

	// Use SHA256 for proper hashing of the entire file
	// This ensures identical files always produce the same hash
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	return hash, nil
}

// copyDependencyPackages copies only the installed dependency packages (not requirements.txt, etc.)
func (ib *IncrementalBuilder) copyDependencyPackages(srcDir, destDir string) error {
	slog.Info("copying dependency packages", "src", srcDir, "dest", destDir)

	// Read source directory to find package directories and root-level files
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	copiedCount := 0
	copiedFiles := 0
	copiedPthPackages := 0
	for _, entry := range entries {
		name := entry.Name()

		// Skip special directories and files we don't want to copy
		if name == "__pycache__" || strings.HasPrefix(name, ".") {
			continue
		}

		// Skip non-package files (requirements.txt, resource.enc, etc.)
		if name == "requirements.txt" || name == "requirements-filtered.txt" || name == "resource.enc" {
			continue
		}

		srcPath := filepath.Join(srcDir, name)
		destPath := filepath.Join(destDir, name)

		if entry.IsDir() {
			// Copy package directory
			if err := ib.copyDirectory(srcPath, destPath); err != nil {
				slog.Warn("failed to copy package", "package", name, "error", err)
				continue
			}
			copiedCount++
		} else if strings.HasSuffix(name, ".pth") {
			// Handle .pth files - these are path configuration files that UV creates
			// for workspace packages. We need to read the path and copy the actual package.
			pthContent, err := os.ReadFile(srcPath)
			if err != nil {
				slog.Warn("failed to read .pth file", "file", name, "error", err)
				continue
			}

			// .pth file contains the path to the package source directory
			packageSourcePath := strings.TrimSpace(string(pthContent))
			if packageSourcePath == "" {
				slog.Warn(".pth file is empty", "file", name)
				continue
			}

			// The .pth file might be a symlink itself, resolve it first
			if info, err := os.Lstat(srcPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
				// It's a symlink, read the actual .pth file content from the target
				realPath, err := filepath.EvalSymlinks(srcPath)
				if err != nil {
					slog.Warn("failed to resolve .pth symlink", "file", name, "error", err)
					continue
				}
				pthContent, err = os.ReadFile(realPath)
				if err != nil {
					slog.Warn("failed to read resolved .pth file", "file", name, "realPath", realPath, "error", err)
					continue
				}
				packageSourcePath = strings.TrimSpace(string(pthContent))
			}

			// Extract package name from .pth filename (e.g., "_sst.pth" -> "sst")
			pthBaseName := strings.TrimSuffix(name, ".pth")
			packageName := strings.TrimPrefix(pthBaseName, "_")

			// Find the actual package directory within the source path
			// The .pth points to the workspace package root (e.g., /path/to/sst_sdk)
			// We need to find the actual Python package inside (e.g., sst_sdk/sst/)
			var packageDir string

			// First, check if there's a directory matching the package name
			candidatePath := filepath.Join(packageSourcePath, packageName)
			if info, err := os.Stat(candidatePath); err == nil && info.IsDir() {
				packageDir = candidatePath
			} else {
				// Try to find the package by looking for pyproject.toml and reading the package config
				pyprojectPath := filepath.Join(packageSourcePath, "pyproject.toml")
				if _, err := os.Stat(pyprojectPath); err == nil {
					if config, err := ib.projectResolver.ParsePyprojectToml(pyprojectPath); err == nil {
						// Check hatch build targets for package location
						if len(config.Tool.Hatch.Build.Targets.Wheel.Packages) > 0 {
							pkgName := config.Tool.Hatch.Build.Targets.Wheel.Packages[0]
							candidatePath = filepath.Join(packageSourcePath, pkgName)
							if info, err := os.Stat(candidatePath); err == nil && info.IsDir() {
								packageDir = candidatePath
								packageName = pkgName // Use the actual package name from config
							}
						}
					}
				}
			}

			if packageDir == "" {
				slog.Warn("could not find package directory for .pth file",
					"pthFile", name,
					"packageSourcePath", packageSourcePath,
					"packageName", packageName)
				continue
			}

			// Copy the package directory to the destination
			packageDestPath := filepath.Join(destDir, packageName)
			slog.Info("copying package from .pth reference",
				"pthFile", name,
				"packageName", packageName,
				"source", packageDir,
				"dest", packageDestPath)

			if err := ib.copyDirectory(packageDir, packageDestPath); err != nil {
				slog.Warn("failed to copy package from .pth", "package", packageName, "error", err)
				continue
			}
			copiedPthPackages++
		} else if strings.HasSuffix(name, ".so") || strings.HasSuffix(name, ".py") {
			// Copy root-level .so files (like _cffi_backend.cpython-313-aarch64-linux-gnu.so)
			// and root-level .py files (like typing_extensions.py, six.py)
			// These are packages that install as single files at the root level
			if err := ib.copyFile(srcPath, destPath); err != nil {
				slog.Warn("failed to copy root file", "file", name, "error", err)
				continue
			}
			copiedFiles++
			slog.Info("copied root-level file", "file", name)
		}
	}

	slog.Info("copied dependency packages", "directories", copiedCount, "rootFiles", copiedFiles, "pthPackages", copiedPthPackages)
	return nil
}
