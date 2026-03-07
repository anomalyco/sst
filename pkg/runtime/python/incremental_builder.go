package python

import (
	"context"
	"encoding/json"
	"fmt"
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

// IncrementalBuilder coordinates selective builds and optimizes the build process
type IncrementalBuilder struct {
	// projectResolver provides simplified project resolution without layout classification
	projectResolver *ProjectResolver

	// uvRunner executes UV commands efficiently
	uvRunner *UvCommandRunner

	// contentFilter handles file exclusion using hybrid approach
	contentFilter *ContentFilter

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

	// ProjectRoot is the root directory of the project for ContentFilter
	ProjectRoot string

	// FunctionID is the ID of the function being built (for progress events)
	FunctionID string
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

	// Initialize UV command runner
	uvRunner := NewUvCommandRunner(UvCommandRunnerConfig{})

	// Create content filter for the project
	contentFilter := NewContentFilterForProject(config.ProjectRoot)

	return &IncrementalBuilder{
		projectResolver: projectResolver,
		uvRunner:        uvRunner,
		contentFilter:   contentFilter,
		config:          config,
	}, nil
}

// Build performs an incremental build using selective package building and caching
func (ib *IncrementalBuilder) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	startTime := time.Now()

	// Update project resolver with current project root
	workingDir := filepath.Dir(input.CfgPath)
	if workingDir == "" {
		workingDir = "."
	}
	ib.projectResolver.projectRoot = workingDir

	// Resolve project structure
	projectInfo, err := ib.projectResolver.ResolveHandler(input.Handler)
	if err != nil {
		return nil, WrapError(err, "project resolution")
	}

	// Discover local packages that need building
	localPackages, err := discoverBuildablePackages(projectInfo, ib.projectResolver)
	if err != nil {
		return nil, WrapError(err, "package discovery")
	}

	// Build local packages that need building
	var packagesBuilt []string
	for _, pkg := range localPackages {
		if err := ib.buildPackage(ctx, input, projectInfo, pkg); err != nil {
			return nil, WrapError(err, "build execution")
		}
		packagesBuilt = append(packagesBuilt, pkg.Name)
	}

	// Install dependencies
	if err := ib.installDependenciesForBuild(ctx, input, projectInfo); err != nil {
		return nil, WrapError(err, "dependency installation")
	}

	// Create final build output
	output, err := ib.createFinalBuildOutput(ctx, input, projectInfo)
	if err != nil {
		return nil, WrapError(err, "build output")
	}

	slog.Info("built",
		"functionID", input.FunctionID,
		"duration", time.Since(startTime),
		"packagesBuilt", len(packagesBuilt))

	return output, nil
}

// buildPackage builds a single local package
func (ib *IncrementalBuilder) buildPackage(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, pkg *LocalPackageInfo) error {
	// Optimize build type based on development vs production
	buildType := "sdist"
	if input.Dev {
		buildType = "wheel"
	}

	// Build the package using UV
	buildCmd := &UvBuildCommand{
		PackageName: pkg.Name,
		PackageDir:  pkg.Path,
		OutputDir:   input.Out(),
		BuildType:   buildType,
	}

	if err := ib.uvRunner.ExecuteBuildCommand(ctx, buildCmd); err != nil {
		return NewBuildFailedError(pkg.Name, err)
	}

	// Post-process the built package
	if err := ib.postProcessPackageBuild(ctx, input, projectInfo, pkg); err != nil {
		return WrapError(err, "package post-processing")
	}

	return nil
}

// createFinalBuildOutput creates the final build output from the build results
func (ib *IncrementalBuilder) createFinalBuildOutput(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo) (*runtime.BuildOutput, error) {
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

	// Create build output
	return &runtime.BuildOutput{
		Out:        input.Out(),
		Handler:    adjustedHandler,
		Errors:     []string{},
		Sourcemaps: []string{},
	}, nil
}

// ensureDockerfile ensures a Dockerfile exists in the output directory for container builds.
// It first checks if a custom Dockerfile exists in the workspace directory, and if not,
// copies the default Python Dockerfile from the platform directory.
// It also exports requirements.txt for the Docker build.
func (ib *IncrementalBuilder) ensureDockerfile(input *runtime.BuildInput, projectInfo *ProjectInfo) error {
	outputDockerfile := filepath.Join(input.Out(), "Dockerfile")

	// requirements.txt is already generated by installDependenciesForBuild → generateOrCopyRequirementsFile.
	// Do NOT regenerate it here — that would overwrite the version that includes workspace packages.

	// Ensure pyproject.toml is in the build context — the Dockerfile may need it for `pip install .`
	outputPyproject := filepath.Join(input.Out(), "pyproject.toml")
	if _, err := os.Stat(outputPyproject); err != nil && projectInfo.PyprojectPath != "" {
		if _, err := os.Stat(projectInfo.PyprojectPath); err == nil {
			_ = copyFile(projectInfo.PyprojectPath, outputPyproject)
		}
	}

	// Check if Dockerfile already exists in output
	if _, err := os.Stat(outputDockerfile); err == nil {
		return nil
	}

	// Check if there's a custom Dockerfile in the project root directory
	projectRoot := projectInfo.ProjectRoot
	if projectRoot == "" {
		projectRoot = path.ResolveRootDir(input.CfgPath)
	}

	customDockerfile := filepath.Join(projectRoot, "Dockerfile")
	if _, err := os.Stat(customDockerfile); err == nil {
		return copyFile(customDockerfile, outputDockerfile)
	}

	// Check if there's a custom Dockerfile in the handler's package directory
	if projectInfo.PyprojectPath != "" {
		handlerPkgDir := filepath.Dir(projectInfo.PyprojectPath)
		if handlerPkgDir != projectRoot {
			handlerDockerfile := filepath.Join(handlerPkgDir, "Dockerfile")
			if _, err := os.Stat(handlerDockerfile); err == nil {
				return copyFile(handlerDockerfile, outputDockerfile)
			}
		}
	}

	// Use the default Python Dockerfile from the platform directory
	defaultDockerfile := filepath.Join(path.ResolvePlatformDir(input.CfgPath), "dist", "dockerfiles", "python.Dockerfile")
	if _, err := os.Stat(defaultDockerfile); err != nil {
		return fmt.Errorf("default Python Dockerfile not found at %s: %w", defaultDockerfile, err)
	}

	return copyFile(defaultDockerfile, outputDockerfile)
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

// adjustHandlerPath adjusts the handler path based on the project structure
// For modern UV workspace projects, the handler path should match the artifact structure exactly.
// We only adjust for legacy patterns like src/ prefix flattening.
func (ib *IncrementalBuilder) adjustHandlerPath(input *runtime.BuildInput, projectInfo *ProjectInfo) (string, error) {
	originalHandler := input.Handler

	// Strip leading ./ if present
	handler := strings.TrimPrefix(originalHandler, "./")

	// For container builds, strip the workspace prefix from the handler path.
	// Source files are copied to the root of the build context (not nested),
	// so the handler path inside the container is relative to the workspace dir.
	// e.g., "functions/ingestion/workspace/handler/main.handler" becomes "handler/main.handler"
	// when workspaceDir is "functions/ingestion/workspace".
	if input.IsContainer && projectInfo.ProjectRoot != "" && projectInfo.PyprojectPath != "" {
		workspaceDir := filepath.Dir(projectInfo.PyprojectPath)
		if workspaceDir != projectInfo.ProjectRoot {
			relWorkspacePath, err := filepath.Rel(projectInfo.ProjectRoot, workspaceDir)
			if err == nil && relWorkspacePath != "." {
				prefix := filepath.ToSlash(relWorkspacePath) + "/"
				handler = strings.TrimPrefix(handler, prefix)
			}
		}
	}

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
				adjustedHandler := adjustedPath + "." + funcName
				return adjustedHandler, nil
			}

			// For zip builds, verify the flattened file exists in the artifact
			adjustedFile := filepath.Join(input.Out(), adjustedPath+".py")
			if _, err := os.Stat(adjustedFile); err == nil {
				adjustedHandler := adjustedPath + "." + funcName
				return adjustedHandler, nil
			}
		}
	}

	// Check if the handler file exists at the expected path in the artifact
	expectedFile := filepath.Join(input.Out(), filePath+".py")
	if _, err := os.Stat(expectedFile); err == nil {
		return handler, nil
	}

	// No file found at either path, return original
	return handler, nil
}

// postProcessPackageBuild performs post-processing on a built package
func (ib *IncrementalBuilder) postProcessPackageBuild(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, pkg *LocalPackageInfo) error {
	// Extract and process the built package archive
	if err := ib.extractAndProcessPackageArchive(input.Out(), projectInfo, pkg); err != nil {
		return fmt.Errorf("failed to extract and process package archive: %w", err)
	}

	return nil
}

// extractAndProcessPackageArchive extracts the built package archive and processes it
func (ib *IncrementalBuilder) extractAndProcessPackageArchive(outputDir string, projectInfo *ProjectInfo, pkg *LocalPackageInfo) error {
	// Look for the built package file (either .whl or .tar.gz)
	// Python normalizes package names by converting dashes to underscores
	normalizedName := strings.ReplaceAll(pkg.Name, "-", "_")

	// Try wheel files first (more common for dev builds)
	patterns := []string{
		filepath.Join(outputDir, normalizedName+"-*.whl"),
		filepath.Join(outputDir, normalizedName+"-*.tar.gz"),
		filepath.Join(outputDir, pkg.Name+"-*.whl"),
		filepath.Join(outputDir, pkg.Name+"-*.tar.gz"),
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
			pkg.Name, normalizedName, normalizedName, pkg.Name, pkg.Name)
	}

	// Process each archive file (should typically be just one)
	for _, archiveFile := range files {
		if err := ib.processPackageArchive(archiveFile, outputDir, projectInfo); err != nil {
			return fmt.Errorf("failed to process archive %s: %w", archiveFile, err)
		}
	}

	return nil
}

// processPackageArchive processes a single package archive file
func (ib *IncrementalBuilder) processPackageArchive(archiveFile, outputDir string, projectInfo *ProjectInfo) error {
	// Handle different archive types
	if strings.HasSuffix(archiveFile, ".whl") {
		// Wheel files are zip archives — extract to get the correctly structured package
		extractCmd := []string{"unzip", "-o", "-q", archiveFile, "-d", outputDir}
		if err := ib.executeCommand(extractCmd, outputDir); err != nil {
			return fmt.Errorf("failed to extract wheel: %w", err)
		}

		os.Remove(archiveFile)

		// Remove .dist-info directories (not needed for Lambda)
		distInfoPattern := filepath.Join(outputDir, "*.dist-info")
		distInfoDirs, _ := filepath.Glob(distInfoPattern)
		for _, distInfoDir := range distInfoDirs {
			os.RemoveAll(distInfoDir)
		}

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
	os.Remove(archiveFile)

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
		if err := copyFile(srcFile, destFile); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", srcFile, destFile, err)
		}
	}

	// Clean up the extracted directory
	os.RemoveAll(extractedDir)

	return nil
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

// installDependenciesForBuild installs dependencies for the build
func (ib *IncrementalBuilder) installDependenciesForBuild(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo) error {
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

	// Check if this is a source code project (not a buildable package)
	// If so, include dev dependencies since they might contain runtime deps
	noDev := true
	if projectInfo.PyprojectPath != "" {
		if content, err := os.ReadFile(projectInfo.PyprojectPath); err == nil {
			contentStr := string(content)
			if strings.Contains(contentStr, "NOT a buildable package") ||
				strings.Contains(contentStr, "Development environment - not a buildable package") ||
				strings.Contains(contentStr, "SST will treat this as source code") {
				noDev = false
			}
		}
	}

	// Check if this is a workspace member (has its own pyproject.toml in a subdirectory)
	// If so, export only that package's dependencies for isolation
	packageName := ""
	useAllPackages := true
	workspaceRoot := workspaceDir // Default to current workspaceDir

	if projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			if config.Project.Name != "" {
				// Check if this pyproject.toml is in a subdirectory (workspace member)
				pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)
				currentDir := filepath.Dir(pyprojectDir)

				// Walk up to find any parent pyproject.toml
				for currentDir != filepath.Dir(currentDir) && currentDir != "." && currentDir != projectInfo.SourceRoot {
					parentPyproject := filepath.Join(currentDir, "pyproject.toml")
					if _, err := os.Stat(parentPyproject); err == nil {
						// Found a parent pyproject.toml — this is a workspace member
						packageName = config.Project.Name
						useAllPackages = false
						workspaceRoot = currentDir
						break
					}
					currentDir = filepath.Dir(currentDir)
				}
			}
		} else {
			slog.Warn("failed to parse pyproject.toml", "path", projectInfo.PyprojectPath, "error", err)
		}
	}

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
		return nil
	}

	// First function for this workspace/package — generate requirements.txt
	if err := ib.uvRunner.ExecuteExportCommand(ctx, exportCmd); err != nil {
		return err
	}

	// Store as shared requirements.txt for this workspace/package
	globalRequirementsFilesMutex.Lock()
	globalRequirementsFiles[cacheKey] = outputFile
	globalRequirementsFilesMutex.Unlock()

	return nil
}

// InputProperties represents the input properties structure
type InputProperties struct {
	Architecture string `json:"architecture"`
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
	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		return fmt.Errorf("failed to read artifact directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if strings.Contains(name, ":") && (strings.Contains(name, "/Users/") || strings.Contains(name, "/home/") || strings.Contains(name, "C:\\")) {
				os.RemoveAll(filepath.Join(artifactDir, name))
			}
		}
	}

	return nil
}

// installDependenciesForLambda installs dependencies for Lambda with correct platform targeting
func (ib *IncrementalBuilder) installDependenciesForLambda(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, requirementsFile string, architecture string) error {
	// Skip uv sync for deployment — we use uv export + uv pip install instead
	if err := ib.copySourceFilesSimple(ctx, input, projectInfo); err != nil {
		return fmt.Errorf("failed to copy source files: %w", err)
	}

	// For container builds, skip dependency installation — the Dockerfile handles it
	if input.IsContainer {
		if err := ib.copyWorkspacePackagesForContainer(input, projectInfo); err != nil {
			return fmt.Errorf("failed to copy workspace packages for container: %w", err)
		}
	} else {
		if err := ib.copySyncedDependencies(ctx, input, projectInfo, architecture); err != nil {
			return fmt.Errorf("failed to copy synced dependencies: %w", err)
		}
	}

	return nil
}

// copyWorkspacePackagesForContainer copies workspace package directories into the artifact
// so the Dockerfile's `uv pip install -r requirements.txt` can resolve relative paths like ./core.
func (ib *IncrementalBuilder) copyWorkspacePackagesForContainer(input *runtime.BuildInput, projectInfo *ProjectInfo) error {
	// Find the uv workspace root — this is where uv export runs from, so local paths
	// in requirements.txt (like ./sst_sdk) are relative to this directory.
	// Walk up from the handler's pyproject.toml to find the root workspace pyproject.toml
	// (the one with [tool.uv.workspace]).
	workspaceRoot := projectInfo.ProjectRoot
	if workspaceRoot == "" {
		workspaceRoot = path.ResolveRootDir(input.CfgPath)
	}

	if projectInfo.PyprojectPath != "" {
		currentDir := filepath.Dir(filepath.Dir(projectInfo.PyprojectPath))
		for currentDir != filepath.Dir(currentDir) && currentDir != "." {
			parentPyproject := filepath.Join(currentDir, "pyproject.toml")
			if content, err := os.ReadFile(parentPyproject); err == nil {
				// Keep walking up until we find the root workspace (with [tool.uv.workspace])
				// or stop at the last pyproject.toml we find
				workspaceRoot = currentDir
				if strings.Contains(string(content), "[tool.uv.workspace]") {
					break
				}
			}
			currentDir = filepath.Dir(currentDir)
		}
	}

	// Read the requirements.txt from the artifact to find workspace package paths
	requirementsPath := filepath.Join(input.Out(), "requirements.txt")
	content, err := os.ReadFile(requirementsPath)
	if err != nil {
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
		if err := copyDirUnfiltered(fullPath, destPath); err != nil {
			return fmt.Errorf("failed to copy workspace package %s: %w", pkgPath, err)
		}

		copied = append(copied, pkgPath)
	}

	return nil
}

// copySourceFilesSimple copies handler source files to the build output directory.
// For UV workspace projects, workspace packages are installed via uv pip install,
// so this only copies the handler's own source files (functions/, etc.).
func (ib *IncrementalBuilder) copySourceFilesSimple(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo) error {
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	handlerPath := input.Handler

	// Strip workspace prefix from handler path if workspaceDir is a subdirectory of ProjectRoot
	// e.g., handler "packages/api/auth/handler.main" with workspaceDir "/project/packages/api"
	// becomes "auth/handler.main"
	var outputPrefix string
	if projectInfo.ProjectRoot != "" && workspaceDir != projectInfo.ProjectRoot {
		relWorkspacePath, err := filepath.Rel(projectInfo.ProjectRoot, workspaceDir)
		if err == nil && relWorkspacePath != "." {
			relWorkspacePath = filepath.ToSlash(relWorkspacePath)
			prefix := relWorkspacePath + "/"
			if strings.HasPrefix(handlerPath, prefix) {
				handlerPath = strings.TrimPrefix(handlerPath, prefix)
				outputPrefix = relWorkspacePath
			}
		}
	}

	outputBase := input.Out()
	// For container builds, source files must be at the root of the build context
	// because the Dockerfile references paths relative to itself (e.g., COPY handler/ ./handler/).
	// The outputPrefix preserves the nested directory structure which is needed for zip builds
	// but breaks container builds where the Dockerfile is copied to the root.
	if outputPrefix != "" && !input.IsContainer {
		outputBase = filepath.Join(input.Out(), outputPrefix)
	}

	if strings.Contains(handlerPath, "/") {
		// Handler is in a subdirectory — find the top-level directory that actually exists.
		// The handler path may contain stale prefixes already resolved by workspaceDir,
		// e.g. handler "functions/src/functions/user/handler.main" when workspaceDir
		// is already the "functions" dir — so "functions/" doesn't exist inside it.
		parts := strings.Split(handlerPath, "/")
		copied := false
		for i := 0; i < len(parts)-1; i++ {
			candidate := parts[i]
			candidatePath := filepath.Join(workspaceDir, candidate)
			if info, err := os.Stat(candidatePath); err == nil && info.IsDir() {
				if err := copyDir(candidatePath, filepath.Join(outputBase, candidate), ib.contentFilter, ""); err != nil {
					return fmt.Errorf("failed to copy directory %s: %w", candidate, err)
				}
				copied = true
				break
			}
		}
		if !copied {
			// None of the path segments matched a directory — copy all .py files from root
			// (handler path is probably fully resolved by workspaceDir already)
		}
	}

	// Always copy root-level .py files (handler might be at root, or there may be __init__.py, etc.)
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return fmt.Errorf("failed to read workspace directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
			if err := copyFile(filepath.Join(workspaceDir, entry.Name()), filepath.Join(outputBase, entry.Name())); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// copySyncedDependencies installs all dependencies (external + workspace packages) with correct platform targeting
func (ib *IncrementalBuilder) copySyncedDependencies(ctx context.Context, input *runtime.BuildInput, projectInfo *ProjectInfo, architecture string) error {
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
						break
					}
				}
			}
			currentDir = parentDir
		}
	}

	// Filter only boto3/botocore (Lambda runtime packages), keep everything else including workspace packages
	filteredRequirementsPath := filepath.Join(input.Out(), "requirements-filtered.txt")
	_, err := ib.filterWorkspacePackagesFromRequirementsAndGetPaths(requirementsPath, filteredRequirementsPath, projectInfo, workspaceRoot)
	if err != nil {
		return fmt.Errorf("failed to filter requirements: %w", err)
	}
	requirementsPath = filteredRequirementsPath

	// Calculate cache key from requirements hash + architecture
	requirementsHash, err := hashFileContents(requirementsPath)
	var cacheKey string
	var depsCacheDir string

	if err == nil {
		cacheKey = fmt.Sprintf("%s-%s", requirementsHash, architecture)
		depsCacheDir = filepath.Join(filepath.Dir(input.Out()), ".deps", cacheKey)

		// Get or create a lock for this cache key to prevent concurrent installs
		globalDependencyInstallLocksMutex.Lock()
		cacheLock, exists := globalDependencyInstallLocks[cacheKey]
		if !exists {
			cacheLock = &sync.Mutex{}
			globalDependencyInstallLocks[cacheKey] = cacheLock
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

		// Check if disk cache exists (from this deploy or previous deploy)
		if entries, err := os.ReadDir(depsCacheDir); err == nil && len(entries) > 0 {
			if err := ib.copyDependencyPackages(depsCacheDir, input.Out()); err != nil {
				slog.Warn("failed to copy from disk cache, will reinstall", "error", err)
				// Remove bad cache and continue to reinstall
				os.RemoveAll(depsCacheDir)
			} else {
				return nil
			}
		}

		// Cache miss - create the cache directory
		if err := os.MkdirAll(depsCacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create deps cache directory: %w", err)
		}
	} else {
		// Fallback to installing directly to output (shouldn't happen normally)
		depsCacheDir = input.Out()
	}

	// Workspace packages are installed by uv pip install (with --no-editable export).
	// We use --reinstall-package for each workspace package to force uv to rebuild them
	// from source every time, bypassing uv's cache. Without this, uv caches local packages
	// by name+version from pyproject.toml and serves stale builds when source files change.
	workspacePackages := ib.getWorkspacePackageNames(projectInfo)

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
		}
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
				slog.Warn("uv pip install still running", "elapsed", elapsed)
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

	for _, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			filteredLines = append(filteredLines, originalLine)
			continue
		}

		// Skip all editable installs that reference local paths
		// Editable installs create symlinks which won't work in Lambda
		// We need to copy these manually
		if strings.HasPrefix(line, "-e ") {
			editablePath := strings.TrimSpace(strings.TrimPrefix(line, "-e "))

			// Filter out any editable install that references a local path
			if strings.HasPrefix(editablePath, "./") || strings.HasPrefix(editablePath, "../") ||
				strings.HasPrefix(editablePath, "/") && !strings.Contains(editablePath, "://") {
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
		// file:// URLs and absolute local paths are also handled by uv pip install

		// boto3/botocore filtering is handled by cleanupInstalledDependencies
		// which catches both direct and transitive dependencies

		// Include the line
		filteredLines = append(filteredLines, originalLine)
	}

	// Write the filtered requirements.txt
	filteredContent := strings.Join(filteredLines, "\n")
	if err := os.WriteFile(outputPath, []byte(filteredContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write filtered requirements file: %w", err)
	}

	return workspacePackagePaths, nil
}

// cleanupInstalledDependencies removes __pycache__ directories, .pyc files, and system files from installed dependencies
// It also removes boto3/botocore packages since Lambda provides them (saves ~22MB per function)
func (ib *IncrementalBuilder) cleanupInstalledDependencies(targetDir string, projectInfo *ProjectInfo) error {

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
				}
			}
			// Also remove the dist-info directory
			entries, _ := os.ReadDir(targetDir)
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), pkg+"-") && strings.HasSuffix(entry.Name(), ".dist-info") {
					distInfoDir := filepath.Join(targetDir, entry.Name())
					if err := os.RemoveAll(distInfoDir); err != nil {
						slog.Warn("failed to remove dist-info directory", "dir", entry.Name(), "error", err)
					}
				}
			}
		}
	}

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}

		if path == targetDir {
			return nil
		}

		// Remove __pycache__ directories
		if info.IsDir() && info.Name() == "__pycache__" {
			os.RemoveAll(path)
			return filepath.SkipDir
		}

		// Remove .pyc, .pyo, .pyd files and .DS_Store files
		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			fileName := info.Name()
			if ext == ".pyc" || ext == ".pyo" || ext == ".pyd" || fileName == ".DS_Store" {
				os.Remove(path)
			}
			// Remove .pyi type stub files and py.typed markers (only useful for IDE, not runtime)
			if ext == ".pyi" || fileName == "py.typed" {
				os.Remove(path)
			}
		}

		// Remove .dist-info directories (pip metadata, not needed at runtime)
		if info.IsDir() && strings.HasSuffix(info.Name(), ".dist-info") {
			os.RemoveAll(path)
			return filepath.SkipDir
		}

		// Remove test directories (e.g., Crypto/SelfTest, tests/, test/)
		if info.IsDir() {
			dirName := info.Name()
			if dirName == "SelfTest" || dirName == "tests" || dirName == "test" {
				os.RemoveAll(path)
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking target directory during cleanup: %w", err)
	}

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

// copyDependencyPackages copies only the installed dependency packages (not requirements.txt, etc.)
func (ib *IncrementalBuilder) copyDependencyPackages(srcDir, destDir string) error {
	slog.Debug("copying dependency packages", "src", srcDir)

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
			if err := copyDir(srcPath, destPath, ib.contentFilter, ""); err != nil {
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
			slog.Debug("copying package from .pth reference",
				"packageName", packageName,
				"source", packageDir)

			if err := copyDir(packageDir, packageDestPath, ib.contentFilter, ""); err != nil {
				slog.Warn("failed to copy package from .pth", "package", packageName, "error", err)
				continue
			}
			copiedPthPackages++
		} else if strings.HasSuffix(name, ".so") || strings.HasSuffix(name, ".py") {
			// Copy root-level .so files and .py files
			if err := copyFile(srcPath, destPath); err != nil {
				slog.Warn("failed to copy root file", "file", name, "error", err)
				continue
			}
			copiedFiles++
		}
	}

	slog.Debug("copied dependency packages", "directories", copiedCount, "rootFiles", copiedFiles, "pthPackages", copiedPthPackages)
	return nil
}

// LocalPackageInfo contains information about a local package
type LocalPackageInfo struct {
	Name string
	Path string
}

// discoverBuildablePackages finds local packages that need building.
// It first checks pyproject.toml for workspace members, then falls back to directory scanning.
// Only packages with build configuration ([build-system], [tool.setuptools], etc.) are returned.
func discoverBuildablePackages(projectInfo *ProjectInfo, resolver *ProjectResolver) ([]*LocalPackageInfo, error) {
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	// Try workspace members from pyproject.toml first
	pyprojectPath := filepath.Join(workspaceDir, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		if config, err := resolver.ParsePyprojectToml(pyprojectPath); err == nil {
			paths := workspacePackagePaths(config, workspaceDir)
			if len(paths) > 0 {
				return buildableFromPaths(paths, resolver)
			}
		}
	}

	// Fallback: scan common directories for buildable packages
	return buildableFromScan(workspaceDir, resolver)
}

// workspacePackagePaths extracts package paths from a parsed pyproject.toml
func workspacePackagePaths(config *PyprojectConfig, workspaceDir string) []string {
	var paths []string

	// UV workspace members
	for _, member := range config.Tool.UV.Workspace.Members {
		paths = append(paths, filepath.Join(workspaceDir, member))
	}

	// UV sources with local paths
	for _, source := range config.Tool.UV.Sources {
		if source.Path != "" {
			paths = append(paths, filepath.Join(workspaceDir, source.Path))
		}
	}

	// Setuptools packages.find.where
	for _, where := range config.Tool.Setuptools.Packages.Find.Where {
		paths = append(paths, filepath.Join(workspaceDir, where))
	}

	// Hatch build targets
	for _, pkg := range config.Tool.Hatch.Build.Targets.Wheel.Packages {
		paths = append(paths, filepath.Join(workspaceDir, pkg))
	}

	return paths
}

// buildableFromPaths filters a list of paths to only those with build configuration
func buildableFromPaths(paths []string, resolver *ProjectResolver) ([]*LocalPackageInfo, error) {
	var packages []*LocalPackageInfo
	for _, p := range paths {
		if !hasBuildConfig(p) {
			continue
		}
		name := filepath.Base(p)
		pyprojectPath := filepath.Join(p, "pyproject.toml")
		if config, err := resolver.ParsePyprojectToml(pyprojectPath); err == nil && config.Project.Name != "" {
			name = config.Project.Name
		}
		packages = append(packages, &LocalPackageInfo{Name: name, Path: p})
	}
	return packages, nil
}

// buildableFromScan scans common directories for buildable packages
func buildableFromScan(workspaceDir string, resolver *ProjectResolver) ([]*LocalPackageInfo, error) {
	var packages []*LocalPackageInfo

	// Check workspace root
	if hasBuildConfig(workspaceDir) {
		name := filepath.Base(workspaceDir)
		if config, err := resolver.ParsePyprojectToml(filepath.Join(workspaceDir, "pyproject.toml")); err == nil && config.Project.Name != "" {
			name = config.Project.Name
		}
		packages = append(packages, &LocalPackageInfo{Name: name, Path: workspaceDir})
	}

	// Check common package locations one level deep
	for _, dir := range []string{"src", "packages", "libs"} {
		searchDir := filepath.Join(workspaceDir, dir)
		entries, err := os.ReadDir(searchDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			candidatePath := filepath.Join(searchDir, entry.Name())
			if !hasBuildConfig(candidatePath) {
				continue
			}
			name := entry.Name()
			if config, err := resolver.ParsePyprojectToml(filepath.Join(candidatePath, "pyproject.toml")); err == nil && config.Project.Name != "" {
				name = config.Project.Name
			}
			packages = append(packages, &LocalPackageInfo{Name: name, Path: candidatePath})
		}
	}

	// Check immediate subdirectories of workspace root
	entries, err := os.ReadDir(workspaceDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			candidatePath := filepath.Join(workspaceDir, entry.Name())
			if !hasBuildConfig(candidatePath) {
				continue
			}
			// Skip if already found
			found := false
			for _, p := range packages {
				if p.Path == candidatePath {
					found = true
					break
				}
			}
			if found {
				continue
			}
			name := entry.Name()
			if config, err := resolver.ParsePyprojectToml(filepath.Join(candidatePath, "pyproject.toml")); err == nil && config.Project.Name != "" {
				name = config.Project.Name
			}
			packages = append(packages, &LocalPackageInfo{Name: name, Path: candidatePath})
		}
	}

	return packages, nil
}

// hasBuildConfig checks if a directory has build configuration (pyproject.toml with [build-system], setup.py, etc.)
func hasBuildConfig(dir string) bool {
	// setup.py or setup.cfg → buildable
	for _, f := range []string{"setup.py", "setup.cfg"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}

	// pyproject.toml with build configuration
	pyprojectPath := filepath.Join(dir, "pyproject.toml")
	content, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return false
	}

	contentStr := string(content)

	// Explicit markers that this is NOT buildable
	if strings.Contains(contentStr, "NOT a buildable package") ||
		strings.Contains(contentStr, "Development environment - not a buildable package") ||
		strings.Contains(contentStr, "SST will treat this as source code") {
		return false
	}

	// Check for build system or tool-specific build config
	buildIndicators := []string{
		"[build-system]",
		"[tool.setuptools]",
		"[tool.poetry]",
		"[tool.hatch]",
		"[tool.flit]",
		"[tool.pdm]",
	}
	for _, indicator := range buildIndicators {
		if strings.Contains(contentStr, indicator) {
			return true
		}
	}

	return false
}
