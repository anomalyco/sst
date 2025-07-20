package python

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DependencyAnalyzer analyzes package dependencies and determines build requirements
type DependencyAnalyzer struct {
	// layoutDetector provides project layout information
	layoutDetector *LayoutDetector

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
	// LayoutDetector for project analysis
	LayoutDetector *LayoutDetector

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
		layoutDetector:  config.LayoutDetector,
		buildCache:      config.BuildCache,
		dependencyCache: make(map[string]*DependencyAnalysis),
		cacheTimeout:    config.CacheTimeout,
	}
}

// AnalyzeDependencies performs comprehensive dependency analysis for a project
func (da *DependencyAnalyzer) AnalyzeDependencies(ctx context.Context, layout *LayoutInfo) (*DependencyAnalysis, error) {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	// Check cache first
	cacheKey := da.generateCacheKey(layout)
	if cached, exists := da.dependencyCache[cacheKey]; exists {
		if time.Since(cached.AnalyzedAt) < da.cacheTimeout {
			return cached, nil
		}
		// Cache expired, remove it
		delete(da.dependencyCache, cacheKey)
	}

	// Perform dependency analysis
	analysis, err := da.analyzeDependenciesInternal(ctx, layout)
	if err != nil {
		return nil, err
	}

	// Cache the result
	analysis.AnalyzedAt = time.Now()
	da.dependencyCache[cacheKey] = analysis

	return analysis, nil
}

// analyzeDependenciesInternal performs the actual dependency analysis
func (da *DependencyAnalyzer) analyzeDependenciesInternal(ctx context.Context, layout *LayoutInfo) (*DependencyAnalysis, error) {
	analysis := &DependencyAnalysis{
		WorkspaceDir:         layout.WorkspaceDir,
		PackageName:          layout.PackageName,
		LocalPackages:        []*LocalPackageInfo{},
		ExternalDependencies: []*ExternalDependencyInfo{},
		DependencyFiles:      []string{},
		DependencyGraph:      make(map[string][]string),
	}

	// Find all dependency files
	if err := da.findDependencyFiles(analysis); err != nil {
		return nil, fmt.Errorf("failed to find dependency files: %w", err)
	}

	// Analyze local packages
	if err := da.analyzeLocalPackages(ctx, layout, analysis); err != nil {
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
func (da *DependencyAnalyzer) analyzeLocalPackages(ctx context.Context, layout *LayoutInfo, analysis *DependencyAnalysis) error {
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
		pkg.BuildRequired = hasChanges

		// Estimate build time
		pkg.EstimatedBuildTime = da.estimatePackageBuildTime(pkg)

		analysis.LocalPackages = append(analysis.LocalPackages, pkg)
	}

	return nil
}

// discoverLocalPackages finds all local packages in the workspace
func (da *DependencyAnalyzer) discoverLocalPackages(workspaceDir string) ([]*LocalPackageInfo, error) {
	var packages []*LocalPackageInfo

	// Check if there's a pyproject.toml with workspace configuration
	pyprojectPath := filepath.Join(workspaceDir, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		// Parse pyproject.toml to find workspace packages
		workspacePackages, err := da.parseWorkspacePackages(pyprojectPath)
		if err == nil && len(workspacePackages) > 0 {
			for _, pkgPath := range workspacePackages {
				absPath := filepath.Join(workspaceDir, pkgPath)
				if pkg, err := da.createLocalPackageInfo(absPath); err == nil {
					packages = append(packages, pkg)
				}
			}
			return packages, nil
		}
	}

	// Fallback: look for packages in common locations
	commonPackageDirs := []string{
		".",
		"src",
		"packages",
		"libs",
	}

	for _, dir := range commonPackageDirs {
		searchDir := filepath.Join(workspaceDir, dir)
		if _, err := os.Stat(searchDir); err != nil {
			continue
		}

		err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip inaccessible paths
			}

			// Look for directories with Python packages
			if info.IsDir() && da.isPackageDirectory(path) {
				if pkg, err := da.createLocalPackageInfo(path); err == nil {
					packages = append(packages, pkg)
				}
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", searchDir, err)
		}
	}

	return packages, nil
}

// parseWorkspacePackages extracts workspace package paths from pyproject.toml
func (da *DependencyAnalyzer) parseWorkspacePackages(pyprojectPath string) ([]string, error) {
	// Parse pyproject.toml to find workspace configuration
	pyproject, err := da.layoutDetector.parsePyprojectToml(pyprojectPath)
	if err != nil {
		return nil, err
	}

	var packages []string

	// Check for UV workspace sources
	for _, source := range pyproject.Tool.UV.Sources {
		if source.Path != "" {
			packages = append(packages, source.Path)
		}
	}

	return packages, nil
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
		if pyproject, err := da.layoutDetector.parsePyprojectToml(pyprojectPath); err == nil {
			if pyproject.Project.Name != "" {
				packageName = pyproject.Project.Name
			}
		}
	}

	return &LocalPackageInfo{
		Name:         packageName,
		Path:         packagePath,
		Dependencies: []string{},
		SourceFiles:  []string{},
	}, nil
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

// shouldSkipSourceFile determines if a source file should be skipped
func (da *DependencyAnalyzer) shouldSkipSourceFile(filePath string) bool {
	skipPatterns := []string{
		"__pycache__",
		".pytest_cache",
		".mypy_cache",
		"test_",
		"_test.py",
		"tests/",
		"build/",
		"dist/",
		".egg-info/",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(filePath, pattern) {
			return true
		}
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
	pyproject, err := da.layoutDetector.parsePyprojectToml(pyprojectPath)
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
		// Generate a package-specific function ID
		packageFunctionID := fmt.Sprintf("package:%s:%s", pkg.Name, pkg.Path)

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
func (da *DependencyAnalyzer) generateCacheKey(layout *LayoutInfo) string {
	return fmt.Sprintf("%s:%s:%s", layout.WorkspaceDir, layout.PackageName, layout.Type)
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

// ClearCache clears the dependency analysis cache
func (da *DependencyAnalyzer) ClearCache() {
	da.mutex.Lock()
	defer da.mutex.Unlock()

	da.dependencyCache = make(map[string]*DependencyAnalysis)
}

// GetCachedAnalysis returns cached dependency analysis if available
func (da *DependencyAnalyzer) GetCachedAnalysis(layout *LayoutInfo) (*DependencyAnalysis, bool) {
	da.mutex.RLock()
	defer da.mutex.RUnlock()

	cacheKey := da.generateCacheKey(layout)
	if cached, exists := da.dependencyCache[cacheKey]; exists {
		if time.Since(cached.AnalyzedAt) < da.cacheTimeout {
			return cached, true
		}
	}

	return nil, false
}
