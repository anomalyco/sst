package python

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DependencyAnalyzer analyzes package dependencies and determines build requirements
type DependencyAnalyzer struct {
	// projectResolver provides project resolution information
	projectResolver *ProjectResolver

	// buildCache provides access to cached dependency information
	buildCache *BuildCache

	// mutex protects concurrent access
	mutex sync.RWMutex

	// dependencyCache caches dependency analysis results
	dependencyCache map[string]*DependencyAnalysis

	// cacheTimeout determines how long cache entries are valid
	cacheTimeout time.Duration
}

// DependencyAnalyzerConfig configures the dependency analyzer
type DependencyAnalyzerConfig struct {
	// ProjectResolver for project analysis
	ProjectResolver *ProjectResolver

	// BuildCache for accessing cached information
	BuildCache *BuildCache

	// CacheTimeout for dependency analysis cache
	CacheTimeout time.Duration
}

// DependencyAnalysis contains the results of dependency analysis
type DependencyAnalysis struct {
	// WorkspaceDir is the workspace directory
	WorkspaceDir string

	// PackageName is the main package name
	PackageName string

	// LocalPackages contains information about local packages
	LocalPackages []*LocalPackageInfo

	// ExternalDependencies contains external package dependencies
	ExternalDependencies []*ExternalDependencyInfo

	// DependencyFiles contains paths to dependency configuration files
	DependencyFiles []string

	// BuildOrder defines the order in which packages should be built
	BuildOrder []string

	// DependencyGraph represents the dependency relationships
	DependencyGraph map[string][]string

	// ConfigFileHashes contains hashes of configuration files for cache validation
	ConfigFileHashes map[string]string

	// AnalyzedAt is when this analysis was performed
	AnalyzedAt time.Time
}

// LocalPackageInfo contains information about a local package
type LocalPackageInfo struct {
	// Name is the package name
	Name string

	// Path is the package directory path
	Path string

	// SourceFiles contains all Python source files in the package
	SourceFiles []string

	// Dependencies contains the names of packages this package depends on
	Dependencies []string

	// HasChanges indicates if the package has changes since last build
	HasChanges bool

	// ChangeReason explains why the package has changes
	ChangeReason string

	// BuildRequired indicates if this package needs to be built
	BuildRequired bool

	// EstimatedBuildTime is the estimated time to build this package
	EstimatedBuildTime time.Duration
}

// ExternalDependencyInfo contains information about external dependencies
type ExternalDependencyInfo struct {
	// Name is the dependency name
	Name string

	// Version is the required version
	Version string

	// Source indicates where the dependency comes from (pypi, git, etc.)
	Source string

	// IsOptional indicates if the dependency is optional
	IsOptional bool

	// Groups contains the dependency groups this belongs to
	Groups []string
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(config DependencyAnalyzerConfig) *DependencyAnalyzer {
	if config.CacheTimeout == 0 {
		config.CacheTimeout = 10 * time.Minute
	}

	return &DependencyAnalyzer{
		projectResolver: config.ProjectResolver,
		buildCache:      config.BuildCache,
		dependencyCache: make(map[string]*DependencyAnalysis),
		cacheTimeout:    config.CacheTimeout,
	}
}

// AnalyzeDependencies performs comprehensive dependency analysis for a project
func (da *DependencyAnalyzer) AnalyzeDependencies(ctx context.Context, projectInfo *ProjectInfo) (*DependencyAnalysis, error) {
	// Check GLOBAL cache first (shared across all function builds)
	cacheKey := da.generateCacheKey(projectInfo)

	globalDependencyCacheMutex.RLock()
	if cached, exists := globalDependencyCache[cacheKey]; exists {
		globalDependencyCacheMutex.RUnlock()
		slog.Info("⚡ Using GLOBAL dependency cache (shared across all functions)", "cacheKey", cacheKey)
		return cached, nil
	}
	globalDependencyCacheMutex.RUnlock()

	da.mutex.Lock()
	defer da.mutex.Unlock()

	// Check instance cache using content-based invalidation
	if cached, exists := da.dependencyCache[cacheKey]; exists {
		// Validate cache using content hashes instead of time
		if da.isCacheValid(cached, projectInfo) {
			return cached, nil
		}
		// Cache invalid, remove it
		delete(da.dependencyCache, cacheKey)
	}

	// Perform dependency analysis
	slog.Info("⏱️ Analyzing dependencies (not in global cache)", "cacheKey", cacheKey)
	analysis, err := da.analyzeDependenciesInternal(ctx, projectInfo)
	if err != nil {
		return nil, err
	}

	// Cache the result in both instance and GLOBAL cache
	analysis.AnalyzedAt = time.Now()
	da.dependencyCache[cacheKey] = analysis

	globalDependencyCacheMutex.Lock()
	globalDependencyCache[cacheKey] = analysis
	globalDependencyCacheMutex.Unlock()
	slog.Info("⚡ Cached dependency analysis globally for reuse", "cacheKey", cacheKey)

	return analysis, nil
}

// analyzeDependenciesInternal performs the actual dependency analysis
func (da *DependencyAnalyzer) analyzeDependenciesInternal(ctx context.Context, projectInfo *ProjectInfo) (*DependencyAnalysis, error) {
	// Extract workspace directory and package name from project info
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	packageName := "unknown"
	if projectInfo.PyprojectPath != "" {
		if config, err := da.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			if config.Project.Name != "" {
				packageName = config.Project.Name
			} else if config.Tool.Poetry.Name != "" {
				packageName = config.Tool.Poetry.Name
			}
		}
	}

	analysis := &DependencyAnalysis{
		WorkspaceDir:         workspaceDir,
		PackageName:          packageName,
		LocalPackages:        []*LocalPackageInfo{},
		ExternalDependencies: []*ExternalDependencyInfo{},
		DependencyFiles:      []string{},
		DependencyGraph:      make(map[string][]string),
		ConfigFileHashes:     make(map[string]string),
	}

	// Find all dependency files
	if err := da.findDependencyFiles(analysis); err != nil {
		return nil, fmt.Errorf("failed to find dependency files: %w", err)
	}

	// Analyze local packages
	if err := da.analyzeLocalPackages(ctx, projectInfo, analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze local packages: %w", err)
	}

	// Analyze external dependencies
	if err := da.analyzeExternalDependencies(analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze external dependencies: %w", err)
	}

	// Build dependency graph
	if err := da.buildDependencyGraph(analysis); err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Determine build order
	if err := da.determineBuildOrder(analysis); err != nil {
		return nil, fmt.Errorf("failed to determine build order: %w", err)
	}

	// Calculate hashes of configuration files for cache validation
	if err := da.calculateConfigFileHashes(analysis, projectInfo); err != nil {
		return nil, fmt.Errorf("failed to calculate config file hashes: %w", err)
	}

	// Set analysis timestamp
	analysis.AnalyzedAt = time.Now()

	// Cache the result
	cacheKey := da.generateCacheKey(projectInfo)
	da.dependencyCache[cacheKey] = analysis

	return analysis, nil
}

// findDependencyFiles locates all dependency configuration files
func (da *DependencyAnalyzer) findDependencyFiles(analysis *DependencyAnalysis) error {
	dependencyFileNames := []string{
		"pyproject.toml",
		"uv.lock",
		"requirements.txt",
		"poetry.lock",
		"Pipfile.lock",
		"setup.py",
		"setup.cfg",
	}

	for _, fileName := range dependencyFileNames {
		filePath := filepath.Join(analysis.WorkspaceDir, fileName)
		if _, err := os.Stat(filePath); err == nil {
			analysis.DependencyFiles = append(analysis.DependencyFiles, filePath)
		}
	}

	return nil
}

// analyzeLocalPackages discovers and analyzes local packages in the workspace
func (da *DependencyAnalyzer) analyzeLocalPackages(ctx context.Context, projectInfo *ProjectInfo, analysis *DependencyAnalysis) error {
	// Find all local packages
	packages, err := da.discoverLocalPackages(analysis.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("failed to discover local packages: %w", err)
	}

	for _, pkg := range packages {
		// Analyze package source files
		sourceFiles, err := da.findPackageSourceFiles(pkg.Path)
		if err != nil {
			return fmt.Errorf("failed to find source files for package %s: %w", pkg.Name, err)
		}
		pkg.SourceFiles = sourceFiles

		// Analyze package dependencies
		dependencies, err := da.analyzePackageDependencies(pkg.Path)
		if err != nil {
			return fmt.Errorf("failed to analyze dependencies for package %s: %w", pkg.Name, err)
		}
		pkg.Dependencies = dependencies

		// Check if package has changes
		hasChanges, reason, err := da.checkPackageChanges(pkg)
		if err != nil {
			return fmt.Errorf("failed to check changes for package %s: %w", pkg.Name, err)
		}
		pkg.HasChanges = hasChanges
		pkg.ChangeReason = reason
		// Only require build if package has changes AND is actually buildable
		isBuildable := pkg.BuildRequired // Save the buildable status from createLocalPackageInfo
		pkg.BuildRequired = hasChanges && isBuildable

		// Estimate build time
		pkg.EstimatedBuildTime = da.estimatePackageBuildTime(pkg)

		analysis.LocalPackages = append(analysis.LocalPackages, pkg)
	}

	return nil
}

// discoverLocalPackages finds all local packages in the workspace
func (da *DependencyAnalyzer) discoverLocalPackages(workspaceDir string) ([]*LocalPackageInfo, error) {
	var packages []*LocalPackageInfo

	// First try to find packages from pyproject.toml
	pyprojectPath := filepath.Join(workspaceDir, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		packagePaths, err := da.parseWorkspacePackages(pyprojectPath)
		if err == nil && len(packagePaths) > 0 {
			// Found packages in pyproject.toml, use those
			slog.Info("using explicit package configuration",
				"workspace", workspaceDir,
				"packages", len(packagePaths),
				"method", "explicit")
			for _, pkgPath := range packagePaths {
				absPath := filepath.Join(workspaceDir, pkgPath)
				if pkg, err := da.createLocalPackageInfo(absPath); err == nil {
					packages = append(packages, pkg)
				}
			}
			return packages, nil
		}
	}

	// If we couldn't find packages in pyproject.toml, fall back to selective directory scanning
	// but be very selective about what we consider a package
	slog.Info("falling back to selective package discovery",
		"workspace", workspaceDir,
		"method", "fallback")
	candidatePackages := da.findCandidatePackages(workspaceDir)

	for _, candidatePath := range candidatePackages {
		if da.isPackageDirectory(candidatePath) {
			if pkg, err := da.createLocalPackageInfo(candidatePath); err == nil {
				packages = append(packages, pkg)
			}
		}
	}

	slog.Info("package discovery completed",
		"workspace", workspaceDir,
		"packagesFound", len(packages),
		"method", "fallback")
	return packages, nil
}

// findCandidatePackages finds potential package directories without recursive walking
func (da *DependencyAnalyzer) findCandidatePackages(workspaceDir string) []string {
	var candidates []string

	// Check the workspace root itself
	if da.hasPackageIndicators(workspaceDir) {
		candidates = append(candidates, workspaceDir)
	}

	// Check common package locations (only one level deep)
	commonPackageDirs := []string{"src", "packages", "libs"}

	for _, dir := range commonPackageDirs {
		searchDir := filepath.Join(workspaceDir, dir)
		if _, err := os.Stat(searchDir); err != nil {
			continue
		}

		// Only scan one level deep in these directories
		entries, err := os.ReadDir(searchDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// Skip obviously non-package directories
			name := entry.Name()
			if da.shouldSkipDirectory(name) {
				continue
			}

			candidatePath := filepath.Join(searchDir, name)
			if da.hasPackageIndicators(candidatePath) {
				candidates = append(candidates, candidatePath)
			}
		}
	}

	// Also check immediate subdirectories of workspace root for packages
	entries, err := os.ReadDir(workspaceDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			name := entry.Name()
			if da.shouldSkipDirectory(name) {
				continue
			}

			candidatePath := filepath.Join(workspaceDir, name)
			if da.hasPackageIndicators(candidatePath) {
				candidates = append(candidates, candidatePath)
			}
		}
	}

	return candidates
}

// shouldSkipDirectory determines if a directory should be skipped during package discovery
func (da *DependencyAnalyzer) shouldSkipDirectory(name string) bool {
	skipDirs := []string{
		".git", ".venv", "venv", "__pycache__", ".pytest_cache", ".mypy_cache",
		".sst", "node_modules", "dist", "build", ".egg-info", "site-packages",
		".tox", ".coverage", ".idea", ".vscode", "bin", "lib", "include",
		"Scripts", "pyvenv.cfg", // Additional virtual environment directories
	}

	for _, skipDir := range skipDirs {
		if name == skipDir {
			return true
		}
	}

	// Skip any directory starting with a dot (except current directory)
	if strings.HasPrefix(name, ".") && name != "." {
		return true
	}

	return false
}

// hasPackageIndicators checks if a directory has indicators that it's a Python package
func (da *DependencyAnalyzer) hasPackageIndicators(path string) bool {
	// Check if this directory should be skipped entirely
	dirName := filepath.Base(path)
	if da.shouldSkipDirectory(dirName) {
		return false
	}

	// For SST Python functions, we should only treat directories as "packages"
	// if they are meant to be built, not just included as source code.
	//
	// Strong indicators of a BUILDABLE Python package:
	buildableIndicators := []string{
		"pyproject.toml", // Has its own build configuration
		"setup.py",       // Has setuptools configuration
		"setup.cfg",      // Has setuptools configuration
	}

	for _, indicator := range buildableIndicators {
		indicatorPath := filepath.Join(path, indicator)
		if _, err := os.Stat(indicatorPath); err == nil {
			// Additional check: make sure it's actually a buildable package
			// and not just a workspace configuration
			if indicator == "pyproject.toml" {
				if da.hasBuildConfiguration(indicatorPath) {
					return true
				}
			} else {
				return true
			}
		}
	}

	// For SST, directories with just __init__.py or Python files
	// should be treated as source directories, not buildable packages
	return false
}

// hasBuildConfiguration checks if a pyproject.toml file has build configuration
func (da *DependencyAnalyzer) hasBuildConfiguration(pyprojectPath string) bool {
	content, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return false
	}

	contentStr := string(content)

	// Check for explicit markers that indicate this should NOT be built as a package
	if strings.Contains(contentStr, "NOT a buildable package") ||
		strings.Contains(contentStr, "Development environment - not a buildable package") ||
		strings.Contains(contentStr, "SST will treat this as source code") {
		slog.Debug("package explicitly marked as non-buildable", "path", pyprojectPath)
		return false
	}

	// Look for build system configuration
	if strings.Contains(contentStr, "[build-system]") {
		return true
	}

	// Look for tool-specific build configuration
	buildTools := []string{
		"[tool.setuptools]",
		"[tool.poetry]",
		"[tool.hatch]",
		"[tool.flit]",
		"[tool.pdm]",
	}

	for _, tool := range buildTools {
		if strings.Contains(contentStr, tool) {
			return true
		}
	}

	return false
}

// parseWorkspacePackages extracts workspace package paths from pyproject.toml
func (da *DependencyAnalyzer) parseWorkspacePackages(pyprojectPath string) ([]string, error) {
	// Parse pyproject.toml to find workspace configuration
	pyproject, err := da.projectResolver.ParsePyprojectToml(pyprojectPath)
	if err != nil {
		return nil, err
	}

	var packages []string

	// Check for UV workspace members (primary method for UV workspaces)
	if len(pyproject.Tool.UV.Workspace.Members) > 0 {
		packages = append(packages, pyproject.Tool.UV.Workspace.Members...)
		slog.Debug("found UV workspace members", "members", pyproject.Tool.UV.Workspace.Members)
	}

	// Check for UV workspace sources (alternative method)
	for _, source := range pyproject.Tool.UV.Sources {
		if source.Path != "" {
			packages = append(packages, source.Path)
		}
	}

	// Check for Poetry packages (tool.poetry.packages)
	if poetryPackages, ok := da.getPoetryPackages(pyproject); ok {
		packages = append(packages, poetryPackages...)
	}

	// Check for PEP 621 packages (project.packages)
	if pep621Packages, ok := da.getPEP621Packages(pyproject); ok {
		packages = append(packages, pep621Packages...)
	}

	// Check for setuptools packages (tool.setuptools.packages.find)
	if setuptoolsPackages, ok := da.getSetuptoolsPackages(pyproject); ok {
		packages = append(packages, setuptoolsPackages...)
	}

	// Check for Hatch packages (tool.hatch.build.targets.wheel.packages)
	if hatchPackages, ok := da.getHatchPackages(pyproject); ok {
		packages = append(packages, hatchPackages...)
	}

	// If we found any packages, return them
	if len(packages) > 0 {
		return packages, nil
	}

	// If no packages were found, use intelligent fallback logic
	workspaceDir := filepath.Dir(pyprojectPath)
	var fallbackPackages []string

	// Check if root directory has Python files
	if da.hasPackageIndicators(workspaceDir) {
		fallbackPackages = append(fallbackPackages, ".")
	}

	// Check if src directory exists and has Python files
	srcDir := filepath.Join(workspaceDir, "src")
	if _, err := os.Stat(srcDir); err == nil && da.hasPackageIndicators(srcDir) {
		fallbackPackages = append(fallbackPackages, "src")
	}

	// If we found any packages, return them
	if len(fallbackPackages) > 0 {
		return fallbackPackages, nil
	}

	// Final fallback to root directory
	return []string{"."}, nil
}

// getPoetryPackages extracts packages from Poetry configuration
func (da *DependencyAnalyzer) getPoetryPackages(pyproject *PyprojectConfig) ([]string, bool) {
	var packages []string

	// Note: PyprojectConfig doesn't have the same Poetry.Packages structure
	// This is a simplified approach - in practice, Poetry packages are usually auto-discovered
	// For now, we'll return empty since this is mainly used for workspace detection
	// which is better handled by UV workspace configuration

	return packages, len(packages) > 0
}

// getPEP621Packages extracts packages from PEP 621 configuration
func (da *DependencyAnalyzer) getPEP621Packages(pyproject *PyprojectConfig) ([]string, bool) {
	// PEP 621 doesn't typically define workspace packages, it defines dependencies
	// For workspace packages, we'd need to look at other tools like setuptools or hatch
	return nil, false
}

// getSetuptoolsPackages extracts packages from setuptools configuration
func (da *DependencyAnalyzer) getSetuptoolsPackages(pyproject *PyprojectConfig) ([]string, bool) {
	var packages []string

	// Check setuptools packages.find.where configuration
	for _, where := range pyproject.Tool.Setuptools.Packages.Find.Where {
		packages = append(packages, where)
	}

	return packages, len(packages) > 0
}

// getHatchPackages extracts packages from Hatch configuration
func (da *DependencyAnalyzer) getHatchPackages(pyproject *PyprojectConfig) ([]string, bool) {
	packages := pyproject.Tool.Hatch.Build.Targets.Wheel.Packages
	return packages, len(packages) > 0
}

// isPackageDirectory checks if a directory contains a Python package
func (da *DependencyAnalyzer) isPackageDirectory(path string) bool {
	// Check for __init__.py
	initFile := filepath.Join(path, "__init__.py")
	if _, err := os.Stat(initFile); err == nil {
		return true
	}

	// Check for setup.py or pyproject.toml
	setupFiles := []string{"setup.py", "pyproject.toml"}
	for _, setupFile := range setupFiles {
		setupPath := filepath.Join(path, setupFile)
		if _, err := os.Stat(setupPath); err == nil {
			return true
		}
	}

	// Check for Python files
	entries, err := os.ReadDir(path)
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

// createLocalPackageInfo creates package info for a local package
func (da *DependencyAnalyzer) createLocalPackageInfo(packagePath string) (*LocalPackageInfo, error) {
	// Determine package name
	packageName := filepath.Base(packagePath)

	// Check for pyproject.toml to get the actual package name
	pyprojectPath := filepath.Join(packagePath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		if pyproject, err := da.projectResolver.ParsePyprojectToml(pyprojectPath); err == nil {
			if pyproject.Project.Name != "" {
				packageName = pyproject.Project.Name
			}
		}
	}

	// Check if this package should be built (vs just included as source)
	buildRequired := da.isPackageBuildable(packagePath)

	return &LocalPackageInfo{
		Name:          packageName,
		Path:          packagePath,
		Dependencies:  []string{},
		SourceFiles:   []string{},
		BuildRequired: buildRequired,
	}, nil
}

// isPackageBuildable checks if a package should be built (vs just included)
func (da *DependencyAnalyzer) isPackageBuildable(packagePath string) bool {
	// For SST Python functions, we should only build packages that have
	// explicit build configuration, not just source directories.

	// Strong indicators of a BUILDABLE Python package:
	buildableIndicators := []string{
		"pyproject.toml", // Has its own build configuration
		"setup.py",       // Has setuptools configuration
		"setup.cfg",      // Has setuptools configuration
	}

	for _, indicator := range buildableIndicators {
		indicatorPath := filepath.Join(packagePath, indicator)

		if _, err := os.Stat(indicatorPath); err == nil {
			// Additional check: make sure it's actually a buildable package
			// and not just a workspace configuration
			if indicator == "pyproject.toml" {
				if da.hasBuildConfiguration(indicatorPath) {
					slog.Debug("package marked as buildable", "path", packagePath, "reason", "has build configuration")
					return true
				}
				// Has pyproject.toml but no build config - this is normal for simple Python projects
			} else {
				slog.Debug("package marked as buildable", "path", packagePath, "reason", fmt.Sprintf("has %s", indicator))
				return true
			}
		}
	}

	// For SST, directories with just __init__.py or Python files
	// should be treated as source directories, not buildable packages
	// This is NORMAL and CORRECT behavior for simple Python functions
	slog.Debug("package will be copied as source", "path", packagePath, "reason", "no build configuration found")
	return false
}

// findPackageSourceFiles finds all Python source files in a package
func (da *DependencyAnalyzer) findPackageSourceFiles(packagePath string) ([]string, error) {
	var sourceFiles []string

	err := filepath.Walk(packagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Skip directories and non-Python files
		if info.IsDir() || !strings.HasSuffix(path, ".py") {
			return nil
		}

		// Skip test files and cache directories
		if da.shouldSkipSourceFile(path) {
			return nil
		}

		sourceFiles = append(sourceFiles, path)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk package directory: %w", err)
	}

	return sourceFiles, nil
}

// walkPackageDirectory walks a package directory more selectively than filepath.Walk
// shouldSkipSourceFile determines if a source file should be skipped
func (da *DependencyAnalyzer) shouldSkipSourceFile(filePath string) bool {
	// Skip files in specific directories (check path components)
	pathComponents := strings.Split(filePath, string(filepath.Separator))
	skipDirs := []string{
		"__pycache__",
		".pytest_cache",
		".mypy_cache",
		".sst",
		".venv",
		"venv",
		"node_modules",
		"site-packages",
		"tests",
		"build",
		"dist",
		".git",
		".tox",
		".coverage",
	}

	for _, component := range pathComponents {
		for _, skipDir := range skipDirs {
			if component == skipDir {
				return true
			}
		}
	}

	// Skip test files by filename patterns
	filename := filepath.Base(filePath)
	if strings.HasPrefix(filename, "test_") || strings.HasSuffix(filename, "_test.py") {
		return true
	}

	// Skip files in .egg-info directories
	if strings.Contains(filePath, ".egg-info") {
		return true
	}

	return false
}

// analyzePackageDependencies analyzes the dependencies of a package
func (da *DependencyAnalyzer) analyzePackageDependencies(packagePath string) ([]string, error) {
	var dependencies []string

	// Check pyproject.toml for dependencies
	pyprojectPath := filepath.Join(packagePath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		deps, err := da.extractDependenciesFromPyproject(pyprojectPath)
		if err == nil {
			dependencies = append(dependencies, deps...)
		}
	}

	// Check requirements.txt
	requirementsPath := filepath.Join(packagePath, "requirements.txt")
	if _, err := os.Stat(requirementsPath); err == nil {
		deps, err := da.extractDependenciesFromRequirements(requirementsPath)
		if err == nil {
			dependencies = append(dependencies, deps...)
		}
	}

	// Remove duplicates
	dependencies = da.removeDuplicateStrings(dependencies)

	return dependencies, nil
}

// extractDependenciesFromPyproject extracts dependencies from pyproject.toml
func (da *DependencyAnalyzer) extractDependenciesFromPyproject(pyprojectPath string) ([]string, error) {
	pyproject, err := da.projectResolver.ParsePyprojectToml(pyprojectPath)
	if err != nil {
		return nil, err
	}

	return pyproject.Project.Dependencies, nil
}

// extractDependenciesFromRequirements extracts dependencies from requirements.txt
func (da *DependencyAnalyzer) extractDependenciesFromRequirements(requirementsPath string) ([]string, error) {
	content, err := os.ReadFile(requirementsPath)
	if err != nil {
		return nil, err
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
			return r == '=' || r == '<' || r == '>' || r == '!' || r == '~'
		})

		if len(parts) > 0 {
			dependencies = append(dependencies, strings.TrimSpace(parts[0]))
		}
	}

	return dependencies, nil
}

// checkPackageChanges checks if a package has changes since the last build
func (da *DependencyAnalyzer) checkPackageChanges(pkg *LocalPackageInfo) (bool, string, error) {
	// Check if we have cached build information for this package
	if da.buildCache != nil {
		// Generate a package-specific function ID using only the package name
		// to avoid absolute paths in cache keys
		packageFunctionID := fmt.Sprintf("package:%s", pkg.Name)

		if cacheEntry, exists := da.buildCache.Get(packageFunctionID); exists {
			// Check if any source files have changed
			for _, sourceFile := range pkg.SourceFiles {
				if expectedHash, exists := cacheEntry.FileHashes[sourceFile]; exists {
					currentHash, err := da.buildCache.calculateFileHash(sourceFile)
					if err != nil {
						return true, fmt.Sprintf("Failed to calculate hash for %s", sourceFile), nil
					}

					if currentHash != expectedHash {
						return true, fmt.Sprintf("Source file changed: %s", sourceFile), nil
					}
				} else {
					return true, fmt.Sprintf("New source file detected: %s", sourceFile), nil
				}
			}

			// Check if package configuration has changed
			packageConfigFiles := []string{
				filepath.Join(pkg.Path, "pyproject.toml"),
				filepath.Join(pkg.Path, "setup.py"),
				filepath.Join(pkg.Path, "setup.cfg"),
			}

			for _, configFile := range packageConfigFiles {
				if _, err := os.Stat(configFile); err == nil {
					if expectedHash, exists := cacheEntry.FileHashes[configFile]; exists {
						currentHash, err := da.buildCache.calculateFileHash(configFile)
						if err != nil {
							return true, fmt.Sprintf("Failed to calculate hash for %s", configFile), nil
						}

						if currentHash != expectedHash {
							return true, fmt.Sprintf("Configuration file changed: %s", configFile), nil
						}
					} else {
						return true, fmt.Sprintf("New configuration file detected: %s", configFile), nil
					}
				}
			}

			// No changes detected
			return false, "No changes detected", nil
		}
	}

	// No cache entry found, assume changes
	return true, "No cached build information found", nil
}

// estimatePackageBuildTime estimates how long it will take to build a package
func (da *DependencyAnalyzer) estimatePackageBuildTime(pkg *LocalPackageInfo) time.Duration {
	// Simple estimation based on number of source files
	baseTime := 5 * time.Second
	fileTime := time.Duration(len(pkg.SourceFiles)) * 100 * time.Millisecond

	return baseTime + fileTime
}

// analyzeExternalDependencies analyzes external package dependencies
func (da *DependencyAnalyzer) analyzeExternalDependencies(analysis *DependencyAnalysis) error {
	// Collect all external dependencies from local packages
	externalDeps := make(map[string]*ExternalDependencyInfo)

	for _, pkg := range analysis.LocalPackages {
		for _, dep := range pkg.Dependencies {
			if !da.isLocalPackage(dep, analysis.LocalPackages) {
				if _, exists := externalDeps[dep]; !exists {
					externalDeps[dep] = &ExternalDependencyInfo{
						Name:       dep,
						Version:    "*", // Default to any version
						Source:     "pypi",
						IsOptional: false,
						Groups:     []string{"main"},
					}
				}
			}
		}
	}

	// Convert map to slice
	for _, dep := range externalDeps {
		analysis.ExternalDependencies = append(analysis.ExternalDependencies, dep)
	}

	return nil
}

// isLocalPackage checks if a dependency is a local package
func (da *DependencyAnalyzer) isLocalPackage(depName string, localPackages []*LocalPackageInfo) bool {
	for _, pkg := range localPackages {
		if pkg.Name == depName {
			return true
		}
	}
	return false
}

// buildDependencyGraph creates a dependency graph for the packages
func (da *DependencyAnalyzer) buildDependencyGraph(analysis *DependencyAnalysis) error {
	// Build graph of local package dependencies
	for _, pkg := range analysis.LocalPackages {
		analysis.DependencyGraph[pkg.Name] = []string{}

		for _, dep := range pkg.Dependencies {
			if da.isLocalPackage(dep, analysis.LocalPackages) {
				analysis.DependencyGraph[pkg.Name] = append(analysis.DependencyGraph[pkg.Name], dep)
			}
		}
	}

	return nil
}

// determineBuildOrder determines the order in which packages should be built
func (da *DependencyAnalyzer) determineBuildOrder(analysis *DependencyAnalysis) error {
	// Perform topological sort on the dependency graph
	buildOrder, err := da.topologicalSort(analysis.DependencyGraph)
	if err != nil {
		return fmt.Errorf("failed to determine build order: %w", err)
	}

	analysis.BuildOrder = buildOrder
	return nil
}

// topologicalSort performs topological sorting on the dependency graph
func (da *DependencyAnalyzer) topologicalSort(graph map[string][]string) ([]string, error) {
	// Kahn's algorithm for topological sorting
	inDegree := make(map[string]int)
	nodes := make(map[string]bool)

	// Initialize in-degree count and collect all nodes
	for node := range graph {
		nodes[node] = true
		if _, exists := inDegree[node]; !exists {
			inDegree[node] = 0
		}
	}

	// Calculate in-degrees
	for _, neighbors := range graph {
		for _, neighbor := range neighbors {
			nodes[neighbor] = true
			inDegree[neighbor]++
		}
	}

	// Find nodes with no incoming edges
	var queue []string
	for node := range nodes {
		if inDegree[node] == 0 {
			queue = append(queue, node)
		}
	}

	var result []string

	for len(queue) > 0 {
		// Remove node from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each neighbor of current node
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(result) != len(nodes) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

// generateCacheKey generates a cache key for dependency analysis
// The key is based on the workspace directory only (not the module path)
// because dependency analysis is the same for all functions in the same workspace
func (da *DependencyAnalyzer) generateCacheKey(projectInfo *ProjectInfo) string {
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}
	// NOTE: We intentionally do NOT include ModulePath in the cache key
	// because dependency analysis (pyproject.toml, uv.lock) is shared across
	// all functions in the same workspace. Including ModulePath would cause
	// each function to re-analyze dependencies, which is slow (~1 minute).
	return workspaceDir
}

// removeDuplicateStrings removes duplicate strings from a slice
func (da *DependencyAnalyzer) removeDuplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// isCacheValid validates cache entry using content hashes instead of time
func (da *DependencyAnalyzer) isCacheValid(cached *DependencyAnalysis, projectInfo *ProjectInfo) bool {
	// Check if pyproject.toml has changed
	if cached.ConfigFileHashes == nil {
		return false
	}

	// Validate pyproject.toml hash
	if projectInfo.PyprojectPath != "" {
		currentHash, err := da.calculateFileHash(projectInfo.PyprojectPath)
		if err != nil {
			return false
		}
		if cachedHash, exists := cached.ConfigFileHashes["pyproject.toml"]; !exists || cachedHash != currentHash {
			return false
		}
	}

	// Validate requirements files if they exist
	requirementsFiles := []string{"requirements.txt", "requirements-dev.txt", "dev-requirements.txt"}
	for _, reqFile := range requirementsFiles {
		reqPath := filepath.Join(projectInfo.ProjectRoot, reqFile)
		if _, err := os.Stat(reqPath); err == nil {
			currentHash, err := da.calculateFileHash(reqPath)
			if err != nil {
				return false
			}
			if cachedHash, exists := cached.ConfigFileHashes[reqFile]; !exists || cachedHash != currentHash {
				return false
			}
		}
	}

	return true
}

// calculateFileHash calculates SHA256 hash of a file
func (da *DependencyAnalyzer) calculateFileHash(filePath string) (string, error) {
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

// calculateConfigFileHashes calculates hashes for all configuration files
func (da *DependencyAnalyzer) calculateConfigFileHashes(analysis *DependencyAnalysis, projectInfo *ProjectInfo) error {
	// Hash pyproject.toml if it exists
	if projectInfo.PyprojectPath != "" {
		hash, err := da.calculateFileHash(projectInfo.PyprojectPath)
		if err != nil {
			return fmt.Errorf("failed to hash pyproject.toml: %w", err)
		}
		analysis.ConfigFileHashes["pyproject.toml"] = hash
	}

	// Hash other dependency files
	dependencyFiles := []string{"requirements.txt", "requirements-dev.txt", "dev-requirements.txt", "uv.lock", "poetry.lock"}
	for _, fileName := range dependencyFiles {
		filePath := filepath.Join(projectInfo.ProjectRoot, fileName)
		if _, err := os.Stat(filePath); err == nil {
			hash, err := da.calculateFileHash(filePath)
			if err != nil {
				slog.Warn("failed to hash dependency file", "file", fileName, "error", err)
				continue
			}
			analysis.ConfigFileHashes[fileName] = hash
		}
	}

	return nil
}
