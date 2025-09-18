package python

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// BuildPipeline provides a unified build process for all Python project types
// It replaces the complex layout-specific build logic with a simple resolve → analyze → package approach
type BuildPipeline struct {
	// projectResolver handles file resolution without layout classification
	projectResolver *ProjectResolver

	// dependencyAnalyzer analyzes project dependencies
	dependencyAnalyzer *DependencyAnalyzer

	// artifactPackager creates deployment artifacts
	artifactPackager *ArtifactPackager

	// buildCache manages build caching based on content
	buildCache *BuildCache

	// mutex protects concurrent access
	mutex sync.RWMutex

	// config stores pipeline configuration
	config BuildPipelineConfig
}

// BuildPipelineConfig configures the build pipeline
type BuildPipelineConfig struct {
	// ProjectRoot is the root directory of the project
	ProjectRoot string

	// CacheDir is the directory for storing cache files
	CacheDir string

	// ArtifactDir is the directory for storing build artifacts
	ArtifactDir string

	// EnableCaching enables content-based caching
	EnableCaching bool

	// EnableProgressReporting enables progress reporting
	EnableProgressReporting bool

	// ProgressCallback receives progress updates
	ProgressCallback ProgressCallback

	// EnableErrorRecovery enables automatic error recovery
	EnableErrorRecovery bool
}

// BuildContext contains context information for a build
type BuildContext struct {
	// FunctionID identifies the function being built
	FunctionID string

	// Handler is the handler path
	Handler string

	// ProjectInfo contains resolved project information
	ProjectInfo *ProjectInfo

	// BuildInput contains the original build input
	BuildInput *runtime.BuildInput

	// StartTime tracks when the build started
	StartTime time.Time

	// Architecture is the target architecture
	Architecture string

	// IsContainer indicates if this is a container build
	IsContainer bool
}

// BuildStage represents different stages of the build process
type BuildStage string

const (
	StageResolve  BuildStage = "resolve"
	StageAnalyze  BuildStage = "analyze"
	StagePackage  BuildStage = "package"
	StageComplete BuildStage = "complete"
)

// PipelineBuildResult contains the result of a build operation
type PipelineBuildResult struct {
	// Success indicates if the build was successful
	Success bool

	// Output contains the build output
	Output *runtime.BuildOutput

	// Duration is the total build time
	Duration time.Duration

	// Stage is the stage where the build completed or failed
	Stage BuildStage

	// Error contains any build error
	Error error

	// CacheHit indicates if the result came from cache
	CacheHit bool

	// Metadata contains additional build metadata
	Metadata map[string]interface{}
}

// ArtifactPackager handles packaging of build artifacts
type ArtifactPackager struct {
	config ArtifactPackagerConfig
}

// ArtifactPackagerConfig configures the artifact packager
type ArtifactPackagerConfig struct {
	OutputDir           string
	EnableOptimization  bool
	EnableCompression   bool
	ExcludePatterns     []string
	IncludePatterns     []string
	PreservePermissions bool
}

// NewBuildPipeline creates a new unified build pipeline
func NewBuildPipeline(config BuildPipelineConfig) (*BuildPipeline, error) {
	// Validate required configuration
	if config.ProjectRoot == "" {
		return nil, fmt.Errorf("project root is required")
	}
	if config.ArtifactDir == "" {
		return nil, fmt.Errorf("artifact directory is required")
	}

	// Create project resolver
	projectResolver := NewProjectResolver(config.ProjectRoot)

	// Create build cache if caching is enabled
	var buildCache *BuildCache
	if config.EnableCaching && config.CacheDir != "" {
		cache, err := NewDefaultBuildCache(config.CacheDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create build cache: %w", err)
		}
		buildCache = cache
	}

	// Create dependency analyzer
	dependencyAnalyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
		BuildCache: buildCache,
	})

	// Create artifact packager
	artifactPackager := &ArtifactPackager{
		config: ArtifactPackagerConfig{
			OutputDir:           config.ArtifactDir,
			EnableOptimization:  true,
			EnableCompression:   false,
			ExcludePatterns:     []string{"__pycache__", "*.pyc", "*.pyo", ".pytest_cache"},
			PreservePermissions: true,
		},
	}

	return &BuildPipeline{
		projectResolver:    projectResolver,
		dependencyAnalyzer: dependencyAnalyzer,
		artifactPackager:   artifactPackager,
		buildCache:         buildCache,
		config:             config,
	}, nil
}

// Build executes the unified build process: resolve → analyze → package
func (bp *BuildPipeline) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	// Create build context
	buildCtx := &BuildContext{
		FunctionID:   input.FunctionID,
		Handler:      input.Handler,
		BuildInput:   input,
		StartTime:    time.Now(),
		Architecture: bp.extractArchitecture(input),
		IsContainer:  bp.isContainerBuild(input),
	}

	// Report progress
	bp.reportProgress(StageResolve, "Starting build process")

	// First resolve the project to get project info
	bp.reportProgress(StageResolve, "Resolving project structure")

	projectInfo, err := bp.resolveProject(buildCtx)
	if err != nil {
		return nil, bp.wrapError(err, "project resolution failed")
	}
	buildCtx.ProjectInfo = projectInfo

	// Check cache after project resolution if enabled
	if bp.config.EnableCaching && bp.buildCache != nil {
		if cachedResult, err := bp.tryUseCachedResult(buildCtx); err == nil {
			bp.reportProgress(StageComplete, "Using cached build result")
			return cachedResult, nil
		}
	}

	// Execute remaining build stages
	result := bp.executeRemainingBuildStages(ctx, buildCtx)

	// Update cache if successful and caching is enabled
	if result.Success && bp.config.EnableCaching && bp.buildCache != nil {
		bp.updateCache(buildCtx, result)
	}

	if !result.Success {
		return nil, result.Error
	}

	return result.Output, nil
}

// executeRemainingBuildStages executes the analyze and package stages
func (bp *BuildPipeline) executeRemainingBuildStages(ctx context.Context, buildCtx *BuildContext) *PipelineBuildResult {
	// Stage 2: Analyze dependencies
	bp.reportProgress(StageAnalyze, "Analyzing dependencies")

	dependencies, err := bp.analyzeDependencies(ctx, buildCtx)
	if err != nil {
		return &PipelineBuildResult{
			Success:  false,
			Stage:    StageAnalyze,
			Error:    bp.wrapError(err, "dependency analysis failed"),
			Duration: time.Since(buildCtx.StartTime),
		}
	}

	// Stage 3: Package artifacts
	bp.reportProgress(StagePackage, "Packaging artifacts")

	output, err := bp.packageArtifacts(ctx, buildCtx, dependencies)
	if err != nil {
		return &PipelineBuildResult{
			Success:  false,
			Stage:    StagePackage,
			Error:    bp.wrapError(err, "artifact packaging failed"),
			Duration: time.Since(buildCtx.StartTime),
		}
	}

	bp.reportProgress(StageComplete, "Build completed successfully")

	return &PipelineBuildResult{
		Success:  true,
		Output:   output,
		Stage:    StageComplete,
		Duration: time.Since(buildCtx.StartTime),
		Metadata: map[string]interface{}{
			"functionID":   buildCtx.FunctionID,
			"architecture": buildCtx.Architecture,
			"isContainer":  buildCtx.IsContainer,
		},
	}
}

// resolveProject resolves the project structure without layout classification
func (bp *BuildPipeline) resolveProject(buildCtx *BuildContext) (*ProjectInfo, error) {
	projectInfo, err := bp.projectResolver.ResolveHandler(buildCtx.Handler)
	if err != nil {
		// Provide clear, actionable error messages
		if isHandlerNotFoundError(err) {
			searchPaths := bp.generateSearchPaths(buildCtx.Handler)
			suggestions := GenerateHandlerSuggestions(buildCtx.Handler, "", searchPaths)
			return nil, NewHandlerNotFoundError(
				buildCtx.Handler,
				searchPaths,
				suggestions,
			).WithSuggestion("Check that the handler file exists and the path is correct")
		}
		return nil, fmt.Errorf("failed to resolve handler %s: %w", buildCtx.Handler, err)
	}

	slog.Info("resolved project structure",
		"functionID", buildCtx.FunctionID,
		"handlerFile", projectInfo.HandlerFile,
		"sourceRoot", projectInfo.SourceRoot,
		"modulePath", projectInfo.ModulePath,
		"dependencies", len(projectInfo.Dependencies))

	return projectInfo, nil
}

// analyzeDependencies analyzes project dependencies using the resolved project info
func (bp *BuildPipeline) analyzeDependencies(ctx context.Context, buildCtx *BuildContext) (*DependencyAnalysis, error) {
	dependencies, err := bp.dependencyAnalyzer.AnalyzeDependencies(ctx, buildCtx.ProjectInfo)
	if err != nil {
		return nil, fmt.Errorf("dependency analysis failed: %w", err)
	}

	slog.Info("analyzed dependencies",
		"functionID", buildCtx.FunctionID,
		"localPackages", len(dependencies.LocalPackages),
		"externalDependencies", len(dependencies.ExternalDependencies))

	return dependencies, nil
}

// packageArtifacts packages the build artifacts
func (bp *BuildPipeline) packageArtifacts(ctx context.Context, buildCtx *BuildContext, dependencies *DependencyAnalysis) (*runtime.BuildOutput, error) {
	outputDir := buildCtx.BuildInput.Out()

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Package source files
	if err := bp.packageSourceFiles(buildCtx, dependencies, outputDir); err != nil {
		return nil, fmt.Errorf("failed to package source files: %w", err)
	}

	// Install dependencies if not a container build
	if !buildCtx.IsContainer {
		if err := bp.installDependencies(ctx, buildCtx, dependencies, outputDir); err != nil {
			return nil, fmt.Errorf("failed to install dependencies: %w", err)
		}
	}

	// Generate handler path for the packaged artifact
	handlerPath, err := bp.generateHandlerPath(buildCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate handler path: %w", err)
	}

	return &runtime.BuildOutput{
		Out:     outputDir,
		Handler: handlerPath,
	}, nil
}

// packageSourceFiles packages the source files into the output directory
func (bp *BuildPipeline) packageSourceFiles(buildCtx *BuildContext, dependencies *DependencyAnalysis, outputDir string) error {
	// Copy source files based on the project structure
	sourceRoot := buildCtx.ProjectInfo.SourceRoot

	// Copy all relevant source files
	for _, pkg := range dependencies.LocalPackages {
		if err := bp.copyPackageFiles(pkg.Path, outputDir, pkg.Name); err != nil {
			return fmt.Errorf("failed to copy package %s: %w", pkg.Name, err)
		}
	}

	// Copy the handler file if it's not already included
	handlerDir := filepath.Dir(buildCtx.ProjectInfo.HandlerFile)
	relHandlerDir, err := filepath.Rel(sourceRoot, handlerDir)
	if err == nil && !strings.HasPrefix(relHandlerDir, "..") {
		// Handler is within source root, copy it
		destDir := filepath.Join(outputDir, relHandlerDir)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create handler directory: %w", err)
		}

		handlerFileName := filepath.Base(buildCtx.ProjectInfo.HandlerFile)
		destFile := filepath.Join(destDir, handlerFileName)
		if err := bp.copyFile(buildCtx.ProjectInfo.HandlerFile, destFile); err != nil {
			return fmt.Errorf("failed to copy handler file: %w", err)
		}
	}

	return nil
}

// installDependencies installs external dependencies
func (bp *BuildPipeline) installDependencies(ctx context.Context, buildCtx *BuildContext, dependencies *DependencyAnalysis, outputDir string) error {
	if len(dependencies.ExternalDependencies) == 0 {
		slog.Info("no external dependencies to install", "functionID", buildCtx.FunctionID)
		return nil
	}

	// Create requirements.txt
	requirementsPath := filepath.Join(outputDir, "requirements.txt")
	depStrings := make([]string, len(dependencies.ExternalDependencies))
	for i, dep := range dependencies.ExternalDependencies {
		depStrings[i] = dep.Name
	}
	if err := bp.createRequirementsFile(depStrings, requirementsPath); err != nil {
		return fmt.Errorf("failed to create requirements.txt: %w", err)
	}

	// Install dependencies using uv
	if err := bp.runUvInstall(ctx, outputDir, buildCtx.Architecture); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	return nil
}

// Helper methods

// extractArchitecture extracts the target architecture from build input
func (bp *BuildPipeline) extractArchitecture(input *runtime.BuildInput) string {
	// Parse properties to get architecture
	type Properties struct {
		Architecture string `json:"architecture"`
	}

	var props Properties
	if err := json.Unmarshal(input.Properties, &props); err == nil && props.Architecture != "" {
		return props.Architecture
	}

	return "x86_64" // Default architecture
}

// isContainerBuild checks if this is a container build
func (bp *BuildPipeline) isContainerBuild(input *runtime.BuildInput) bool {
	type Properties struct {
		Container bool `json:"container"`
	}

	var props Properties
	if err := json.Unmarshal(input.Properties, &props); err == nil {
		return props.Container
	}

	return false
}

// extractPackageName extracts a package name from project info
func (bp *BuildPipeline) extractPackageName(projectInfo *ProjectInfo) string {
	if projectInfo.PyprojectPath != "" {
		if config, err := bp.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			if config.Project.Name != "" {
				return config.Project.Name
			}
		}
	}

	// Fallback to directory name
	return filepath.Base(projectInfo.SourceRoot)
}

// generateHandlerPath generates the handler path for the packaged artifact
func (bp *BuildPipeline) generateHandlerPath(buildCtx *BuildContext) (string, error) {
	// Use the module path from project info
	modulePath := buildCtx.ProjectInfo.ModulePath

	// Extract function name from original handler
	parts := strings.Split(buildCtx.Handler, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid handler format: %s", buildCtx.Handler)
	}

	functionName := parts[len(parts)-1]
	return fmt.Sprintf("%s.%s", modulePath, functionName), nil
}

// generateSearchPaths generates search paths for error messages
func (bp *BuildPipeline) generateSearchPaths(handler string) []string {
	paths := []string{}

	// Add direct path
	paths = append(paths, filepath.Join(bp.config.ProjectRoot, handler))

	// Add common directories
	commonDirs := []string{"src", "app", "functions", "lambda", "handlers"}
	for _, dir := range commonDirs {
		paths = append(paths, filepath.Join(bp.config.ProjectRoot, dir, handler))
	}

	return paths
}

// wrapError wraps an error with additional context
func (bp *BuildPipeline) wrapError(err error, message string) error {
	if pythonErr, ok := err.(*PythonRuntimeError); ok {
		return pythonErr.WithContext("stage", message)
	}
	return fmt.Errorf("%s: %w", message, err)
}

// reportProgress reports build progress if enabled
func (bp *BuildPipeline) reportProgress(stage BuildStage, message string) {
	if bp.config.EnableProgressReporting && bp.config.ProgressCallback != nil {
		bp.config.ProgressCallback(ProgressEvent{
			Stage:   string(stage),
			Message: message,
		})
	}

	slog.Info("build progress", "stage", stage, "message", message)
}

// tryUseCachedResult attempts to use a cached build result
func (bp *BuildPipeline) tryUseCachedResult(buildCtx *BuildContext) (*runtime.BuildOutput, error) {
	// Check if we have a cached entry
	entry, exists := bp.buildCache.Get(buildCtx.FunctionID)
	if !exists {
		return nil, fmt.Errorf("no cache entry found")
	}

	// Check if cache is still valid by comparing file hashes
	if bp.isCacheValid(buildCtx, entry) {
		return &runtime.BuildOutput{
			Out:     buildCtx.BuildInput.Out(),
			Handler: entry.BuildOutput.Handler,
		}, nil
	}

	return nil, fmt.Errorf("cache entry is outdated")
}

// isCacheValid checks if a cache entry is still valid
func (bp *BuildPipeline) isCacheValid(buildCtx *BuildContext, entry *CacheEntry) bool {
	// If project info is not resolved yet, we can't validate cache
	if buildCtx.ProjectInfo == nil {
		return false
	}

	// Check if any dependency files have changed
	for _, depFile := range buildCtx.ProjectInfo.Dependencies {
		if hash, err := bp.calculateFileHash(depFile); err != nil || entry.FileHashes[depFile] != hash {
			return false
		}
	}

	// Check if handler file has changed
	if hash, err := bp.calculateFileHash(buildCtx.ProjectInfo.HandlerFile); err != nil || entry.FileHashes[buildCtx.ProjectInfo.HandlerFile] != hash {
		return false
	}

	return true
}

// updateCache updates the build cache with new results
func (bp *BuildPipeline) updateCache(buildCtx *BuildContext, result *PipelineBuildResult) {
	// Calculate file hashes
	fileHashes := make(map[string]string)

	// Hash dependency files
	for _, depFile := range buildCtx.ProjectInfo.Dependencies {
		if hash, err := bp.calculateFileHash(depFile); err == nil {
			fileHashes[depFile] = hash
		}
	}

	// Hash handler file
	if hash, err := bp.calculateFileHash(buildCtx.ProjectInfo.HandlerFile); err == nil {
		fileHashes[buildCtx.ProjectInfo.HandlerFile] = hash
	}

	// Create cache entry
	entry := &CacheEntry{
		FunctionID: buildCtx.FunctionID,
		BuildTime:  time.Now(),
		FileHashes: fileHashes,
		BuildOutput: &CachedBuildOutput{
			Handler:       result.Output.Handler,
			OutputDir:     result.Output.Out,
			BuildDuration: result.Duration,
		},
	}

	// Store in cache
	bp.buildCache.Set(buildCtx.FunctionID, entry)
}

// Utility methods for file operations

// copyPackageFiles copies package files to the output directory
func (bp *BuildPipeline) copyPackageFiles(srcDir, destDir, packageName string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded patterns
		if bp.shouldExcludeFile(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate destination path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, packageName, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return bp.copyFile(path, destPath)
	})
}

// copyFile copies a single file
func (bp *BuildPipeline) copyFile(src, dest string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	// Copy file content
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(srcFile)
	return err
}

// shouldExcludeFile checks if a file should be excluded from packaging
func (bp *BuildPipeline) shouldExcludeFile(filename string) bool {
	for _, pattern := range bp.artifactPackager.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}
	return false
}

// createRequirementsFile creates a requirements.txt file
func (bp *BuildPipeline) createRequirementsFile(dependencies []string, path string) error {
	content := strings.Join(dependencies, "\n")
	return os.WriteFile(path, []byte(content), 0644)
}

// runUvInstall runs uv to install dependencies
func (bp *BuildPipeline) runUvInstall(ctx context.Context, outputDir, architecture string) error {
	// This is a placeholder - the actual UV installation logic would be implemented here
	// For now, we'll just log that we would install dependencies
	slog.Info("installing dependencies with uv",
		"outputDir", outputDir,
		"architecture", architecture)

	// TODO: Implement actual UV installation
	return nil
}

// calculateFileHash calculates a hash for a file
func (bp *BuildPipeline) calculateFileHash(filePath string) (string, error) {
	// This is a placeholder - actual hash calculation would be implemented here
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	// For now, use modification time as a simple hash
	return fmt.Sprintf("%d", info.ModTime().Unix()), nil
}

// Helper functions

// isHandlerNotFoundError checks if an error is a handler not found error
func isHandlerNotFoundError(err error) bool {
	if pythonErr, ok := err.(*PythonRuntimeError); ok {
		return pythonErr.Type == ErrorTypeHandlerNotFound
	}
	return false
}

// ShouldRebuild determines if a function should be rebuilt
func (bp *BuildPipeline) ShouldRebuild(functionID, handler string) bool {
	if !bp.config.EnableCaching || bp.buildCache == nil {
		return true
	}

	// Check if we have a cached entry
	_, exists := bp.buildCache.Get(functionID)
	if !exists {
		return true
	}

	// For now, always rebuild - more sophisticated change detection can be added later
	return true
}

// ClearCache clears the build cache
func (bp *BuildPipeline) ClearCache() error {
	if bp.buildCache != nil {
		return bp.buildCache.Clear()
	}
	return nil
}

// GetCacheStats returns cache statistics
func (bp *BuildPipeline) GetCacheStats() *CacheStats {
	if bp.buildCache != nil {
		stats := bp.buildCache.GetStats()
		return &stats
	}
	return nil
}
