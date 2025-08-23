package python

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// LayoutType represents different Python project layout types
type LayoutType string

const (
	LayoutTypeWorkspace LayoutType = "workspace" // UV workspace with pyproject.toml
	LayoutTypeFlat      LayoutType = "flat"      // Flat structure without src/ directory
	LayoutTypeNested    LayoutType = "nested"    // Nested packages with various structures
	LayoutTypeLegacy    LayoutType = "legacy"    // Legacy Python projects
)

// LayoutInfo stores detected project layout information
type LayoutInfo struct {
	// Type of layout detected
	Type LayoutType `json:"type"`

	// HandlerFile is the absolute path to the Python handler file
	HandlerFile string `json:"handlerFile"`

	// WorkspaceDir is the directory containing pyproject.toml
	WorkspaceDir string `json:"workspaceDir"`

	// PackageName is the name of the Python package
	PackageName string `json:"packageName"`

	// PythonPath contains additional Python paths for module resolution
	PythonPath []string `json:"pythonPath"`

	// Dependencies contains the list of dependency files that affect this layout
	Dependencies []string `json:"dependencies"`

	// ModulePath is the Python import path for the handler
	ModulePath string `json:"modulePath"`

	// SourceRoot is the root directory for source files
	SourceRoot string `json:"sourceRoot"`

	// HasSrcDirectory indicates if the layout uses a src/ directory
	HasSrcDirectory bool `json:"hasSrcDirectory"`

	// WorkspacePackagesWithSrc tracks which workspace packages use src/{package_name} layout
	WorkspacePackagesWithSrc map[string]bool `json:"workspacePackagesWithSrc,omitempty"`

	// DetectedAt is when this layout was detected (for cache invalidation)
	DetectedAt time.Time `json:"detectedAt"`

	// PyprojectPath is the path to pyproject.toml file
	PyprojectPath string `json:"pyprojectPath"`

	// RequirementsPath is the path to requirements.txt file
	RequirementsPath string `json:"requirementsPath"`

	// SetupPyPath is the path to setup.py file
	SetupPyPath string `json:"setupPyPath"`

	// HandlerPath is the path to the handler file
	HandlerPath string `json:"handlerPath"`
}

// LayoutDetector provides flexible Python project layout detection
type LayoutDetector struct {
	// projectRoot is the root directory of the project
	projectRoot string

	// cache stores detected layout information keyed by handler path
	cache map[string]*LayoutInfo

	// mutex protects concurrent access to the cache
	mutex sync.RWMutex

	// cacheTimeout determines how long cache entries are valid
	cacheTimeout time.Duration
}

// LayoutDetectorConfig configures the layout detector
type LayoutDetectorConfig struct {
	ProjectRoot  string
	CacheTimeout time.Duration
}

// NewLayoutDetector creates a new layout detector with the given configuration
func NewLayoutDetector(config LayoutDetectorConfig) *LayoutDetector {
	if config.CacheTimeout == 0 {
		config.CacheTimeout = 5 * time.Minute // Default cache timeout
	}

	return &LayoutDetector{
		projectRoot:  config.ProjectRoot,
		cache:        make(map[string]*LayoutInfo),
		cacheTimeout: config.CacheTimeout,
	}
}

// DetectLayout analyzes the project structure and returns layout information for the given handler
func (ld *LayoutDetector) DetectLayout(handler string) (*LayoutInfo, error) {
	ld.mutex.Lock()
	defer ld.mutex.Unlock()

	// Check cache first
	if cached, exists := ld.cache[handler]; exists {
		if time.Since(cached.DetectedAt) < ld.cacheTimeout {
			return cached, nil
		}
		// Cache expired, remove it
		delete(ld.cache, handler)
	}

	// Detect the layout
	layout, err := ld.detectLayoutInternal(handler)
	if err != nil {
		// Wrap the error with proper context
		if pythonErr, ok := err.(*PythonRuntimeError); ok {
			return nil, pythonErr
		}
		return nil, NewLayoutDetectionError(err.Error(), handler).WithCause(err)
	}

	// Cache the result
	layout.DetectedAt = time.Now()
	ld.cache[handler] = layout

	return layout, nil
}

// detectLayoutInternal performs the actual layout detection logic
func (ld *LayoutDetector) detectLayoutInternal(handler string) (*LayoutInfo, error) {
	// Find the Python file for this handler
	handlerFile, err := ld.FindPythonFile(handler)
	if err != nil {
		return nil, fmt.Errorf("failed to find Python file for handler %s: %w", handler, err)
	}

	// Find the workspace directory (containing pyproject.toml)
	workspaceDir, err := ld.ResolveWorkspace(handlerFile)
	if err != nil {
		// If no workspace found, create a fallback configuration
		fallbackLayout, fallbackErr := ld.CreateFallbackWorkspace(handlerFile)
		if fallbackErr != nil {
			return nil, fmt.Errorf("failed to resolve workspace and create fallback for handler %s: %w (original error: %v)", handler, fallbackErr, err)
		}
		return fallbackLayout, nil
	}

	// Determine the layout type and gather information
	layout := &LayoutInfo{
		HandlerFile:  handlerFile,
		WorkspaceDir: workspaceDir,
		Dependencies: []string{},
		PythonPath:   []string{},
	}

	// Analyze the project structure to determine layout type
	if err := ld.analyzeProjectStructure(layout); err != nil {
		return nil, fmt.Errorf("failed to analyze project structure: %w", err)
	}

	return layout, nil
}

// FindPythonFile locates the Python file for the given handler path using flexible pattern matching
func (ld *LayoutDetector) FindPythonFile(handler string) (string, error) {
	var searchPaths []string

	// First try direct resolution
	if found, err := ld.tryDirectResolution(handler); err == nil {
		return found, nil
	} else {
		// Collect search paths for error reporting
		dir := filepath.Dir(handler)
		base := strings.TrimSuffix(filepath.Base(handler), filepath.Ext(handler))
		searchPaths = ld.generateCandidatePaths(handler, dir, base)
	}

	// If direct resolution fails, use dynamic discovery
	found, err := ld.dynamicFileDiscovery(handler)
	if err != nil {
		// Create a detailed error with search paths
		return "", NewHandlerNotFoundError(handler, searchPaths).WithCause(err)
	}

	return found, nil
}

// tryDirectResolution attempts to find the Python file using common patterns
func (ld *LayoutDetector) tryDirectResolution(handler string) (string, error) {
	dir := filepath.Dir(handler)
	base := strings.TrimSuffix(filepath.Base(handler), filepath.Ext(handler))

	// Generate candidate paths in order of preference
	candidates := ld.generateCandidatePaths(handler, dir, base)

	for _, candidate := range candidates {
		if ld.isValidPythonFile(candidate) {
			absPath, err := filepath.Abs(candidate)
			if err != nil {
				return "", fmt.Errorf("failed to get absolute path for %s: %w", candidate, err)
			}
			return absPath, nil
		}
	}

	return "", fmt.Errorf("no direct resolution found for handler %s", handler)
}

// generateCandidatePaths creates a list of potential file locations
func (ld *LayoutDetector) generateCandidatePaths(handler, dir, base string) []string {
	candidates := []string{}

	// 1. Direct paths (as specified)
	candidates = append(candidates, filepath.Join(ld.projectRoot, handler))
	if !strings.HasSuffix(handler, ".py") {
		candidates = append(candidates, filepath.Join(ld.projectRoot, handler+".py"))
		candidates = append(candidates, filepath.Join(ld.projectRoot, dir, base+".py"))
	}

	// 2. Common Python project structures
	commonDirs := []string{"src", "app", "functions", "lambda", "handlers", "lib"}
	for _, commonDir := range commonDirs {
		candidates = append(candidates, filepath.Join(ld.projectRoot, commonDir, handler))
		if !strings.HasSuffix(handler, ".py") {
			candidates = append(candidates, filepath.Join(ld.projectRoot, commonDir, handler+".py"))
			candidates = append(candidates, filepath.Join(ld.projectRoot, commonDir, dir, base+".py"))
		}
	}

	// 3. Nested package structures
	if dir != "." && dir != "" {
		// Try treating directory components as package names
		pathParts := strings.Split(dir, string(filepath.Separator))
		for i := range pathParts {
			partialPath := filepath.Join(pathParts[:i+1]...)
			candidates = append(candidates, filepath.Join(ld.projectRoot, partialPath, base+".py"))
			candidates = append(candidates, filepath.Join(ld.projectRoot, "src", partialPath, base+".py"))
		}
	}

	return candidates
}

// dynamicFileDiscovery performs a more thorough search when direct resolution fails
func (ld *LayoutDetector) dynamicFileDiscovery(handler string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(handler), filepath.Ext(handler))

	// Search for files with the same base name
	matches, err := ld.findFilesByName(base + ".py")
	if err != nil {
		return "", fmt.Errorf("dynamic discovery failed: %w", err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no Python files found matching handler %s", handler)
	}

	// If multiple matches, try to find the best one
	bestMatch, err := ld.selectBestMatch(handler, matches)
	if err != nil {
		return "", fmt.Errorf("failed to select best match: %w", err)
	}

	return bestMatch, nil
}

// findFilesByName searches for Python files with the given name throughout the project
func (ld *LayoutDetector) findFilesByName(filename string) ([]string, error) {
	var matches []string

	err := filepath.Walk(ld.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		// Skip hidden directories and common ignore patterns
		if info.IsDir() && ld.shouldSkipDirectory(info.Name()) {
			return filepath.SkipDir
		}

		// Check for matching filename
		if !info.IsDir() && info.Name() == filename {
			// Verify it's a valid Python file
			if ld.isValidPythonFile(path) {
				matches = append(matches, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	return matches, nil
}

// shouldSkipDirectory determines if a directory should be skipped during traversal
func (ld *LayoutDetector) shouldSkipDirectory(dirname string) bool {
	skipDirs := []string{
		".", ".git", ".svn", ".hg", ".bzr",
		"__pycache__", ".pytest_cache", ".mypy_cache",
		"node_modules", ".venv", "venv", "env",
		".tox", "build", "dist", ".egg-info",
		".idea", ".vscode", ".DS_Store",
	}

	for _, skip := range skipDirs {
		if dirname == skip || strings.HasPrefix(dirname, skip) {
			return true
		}
	}

	return false
}

// isValidPythonFile checks if a file is a valid Python file
func (ld *LayoutDetector) isValidPythonFile(path string) bool {
	// Check file extension
	if !strings.HasSuffix(path, ".py") {
		return false
	}

	// Check if file exists and is readable
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Skip directories and non-regular files
	if !info.Mode().IsRegular() {
		return false
	}

	// Basic validation: check if file contains Python-like content
	return ld.containsPythonContent(path)
}

// containsPythonContent performs basic validation that the file contains Python code
func (ld *LayoutDetector) containsPythonContent(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first few lines to check for Python indicators
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" && n == 0 {
		return false
	}

	// Empty files are valid Python files
	if n == 0 {
		return true
	}

	content := string(buffer[:n])

	// Look for Python indicators
	pythonIndicators := []string{
		"def ", "class ", "import ", "from ",
		"#!/usr/bin/env python", "#!/usr/bin/python",
		"# -*- coding:", "# coding:",
	}

	for _, indicator := range pythonIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}

	// If no clear indicators but has .py extension, assume it's Python
	// (could be an empty file or just comments)
	return true
}

// selectBestMatch chooses the most appropriate file when multiple matches are found
func (ld *LayoutDetector) selectBestMatch(handler string, matches []string) (string, error) {
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Score each match based on how well it matches the expected structure
	type scoredMatch struct {
		path  string
		score int
	}

	var scored []scoredMatch
	handlerDir := filepath.Dir(handler)

	for _, match := range matches {
		score := ld.scoreMatch(handler, handlerDir, match)
		scored = append(scored, scoredMatch{path: match, score: score})
	}

	// Find the highest scoring match
	bestScore := -1
	bestMatch := ""

	for _, sm := range scored {
		if sm.score > bestScore {
			bestScore = sm.score
			bestMatch = sm.path
		}
	}

	if bestMatch == "" {
		return matches[0], nil // Fallback to first match
	}

	return bestMatch, nil
}

// scoreMatch assigns a score to a potential match based on how well it fits the expected structure
func (ld *LayoutDetector) scoreMatch(handler, handlerDir, matchPath string) int {
	score := 0

	// Get relative path from project root
	relPath, err := filepath.Rel(ld.projectRoot, matchPath)
	if err != nil {
		return score
	}

	// Prefer matches that preserve the directory structure
	if handlerDir != "." && handlerDir != "" {
		if strings.Contains(relPath, handlerDir) {
			score += 10
		}
	}

	// Prefer matches in common Python directories
	commonDirs := []string{"src", "app", "functions", "lambda", "handlers"}
	for _, dir := range commonDirs {
		if strings.HasPrefix(relPath, dir+string(filepath.Separator)) {
			score += 5
		}
	}

	// Prefer shorter paths (less nested)
	pathDepth := len(strings.Split(relPath, string(filepath.Separator)))
	score += (10 - pathDepth) // Subtract depth from base score

	// Prefer paths that don't contain test directories
	testDirs := []string{"test", "tests", "__pycache__"}
	for _, testDir := range testDirs {
		if strings.Contains(relPath, testDir) {
			score -= 5
		}
	}

	return score
}

// HandleSymlinks resolves symbolic links and validates the target
func (ld *LayoutDetector) HandleSymlinks(path string) (string, error) {
	// Check if the path is a symbolic link
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		// Not a symlink, return as-is
		return path, nil
	}

	// Resolve the symlink
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink %s: %w", path, err)
	}

	// Ensure the resolved path is still within the project root
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for resolved symlink: %w", err)
	}

	absProjectRoot, err := filepath.Abs(ld.projectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute project root: %w", err)
	}

	// Handle macOS symlink resolution (/var -> /private/var)
	resolvedClean, err := filepath.EvalSymlinks(absResolved)
	if err == nil {
		absResolved = resolvedClean
	}

	projectRootClean, err := filepath.EvalSymlinks(absProjectRoot)
	if err == nil {
		absProjectRoot = projectRootClean
	}

	if !strings.HasPrefix(absResolved, absProjectRoot) {
		return "", fmt.Errorf("symlink %s resolves to path outside project root: %s", path, absResolved)
	}

	return absResolved, nil
}

// ResolveWorkspace finds the workspace directory by looking for pyproject.toml
func (ld *LayoutDetector) ResolveWorkspace(handlerPath string) (string, error) {
	// Ensure we're within the project root
	if !strings.HasPrefix(handlerPath, ld.projectRoot) {
		return "", fmt.Errorf("handler file %s is not within project root %s", handlerPath, ld.projectRoot)
	}

	// Try to find nested workspaces first
	candidateWorkspaces, err := ld.ResolveNestedWorkspaces(handlerPath)
	if err == nil && len(candidateWorkspaces) > 0 {
		// Use the most specific (deepest) workspace
		bestWorkspace := candidateWorkspaces[0]

		// Validate the workspace configuration
		if err := ld.ValidateWorkspaceConfiguration(bestWorkspace); err == nil {
			return bestWorkspace, nil
		}
		// If validation fails, try the next candidate
		for _, workspace := range candidateWorkspaces[1:] {
			if err := ld.ValidateWorkspaceConfiguration(workspace); err == nil {
				return workspace, nil
			}
		}
	}

	// Fallback to simple traversal up the directory tree
	currentDir := filepath.Dir(handlerPath)
	for {
		pyprojectPath := filepath.Join(currentDir, "pyproject.toml")
		if _, err := os.Stat(pyprojectPath); err == nil {
			// Validate the workspace
			if err := ld.ValidateWorkspaceConfiguration(currentDir); err == nil {
				return currentDir, nil
			}
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)

		// Stop if we've reached the project root or can't go up further
		if parentDir == currentDir || currentDir == ld.projectRoot {
			break
		}

		currentDir = parentDir
	}

	// Final fallback: check if there's a pyproject.toml in the project root
	rootPyproject := filepath.Join(ld.projectRoot, "pyproject.toml")
	if _, err := os.Stat(rootPyproject); err == nil {
		if err := ld.ValidateWorkspaceConfiguration(ld.projectRoot); err == nil {
			return ld.projectRoot, nil
		}
	}

	return "", fmt.Errorf("no valid pyproject.toml found in directory tree from %s up to project root %s",
		filepath.Dir(handlerPath), ld.projectRoot)
}

// analyzeProjectStructure determines the layout type and populates layout information
func (ld *LayoutDetector) analyzeProjectStructure(layout *LayoutInfo) error {
	// Read pyproject.toml to get package information
	if err := ld.extractPackageInfo(layout); err != nil {
		return fmt.Errorf("failed to extract package info: %w", err)
	}

	// Determine layout type based on directory structure
	if err := ld.determineLayoutType(layout); err != nil {
		return fmt.Errorf("failed to determine layout type: %w", err)
	}

	// Set up Python paths and dependencies
	if err := ld.setupPythonPaths(layout); err != nil {
		return fmt.Errorf("failed to setup Python paths: %w", err)
	}

	// Generate module path for the handler
	if err := ld.generateModulePath(layout); err != nil {
		return fmt.Errorf("failed to generate module path: %w", err)
	}

	return nil
}

// extractPackageInfo reads pyproject.toml and extracts package information
func (ld *LayoutDetector) extractPackageInfo(layout *LayoutInfo) error {
	pyprojectPath := filepath.Join(layout.WorkspaceDir, "pyproject.toml")
	layout.Dependencies = append(layout.Dependencies, pyprojectPath)

	// Parse pyproject.toml to extract package information
	pyproject, err := ld.parsePyprojectToml(pyprojectPath)
	if err != nil {
		return fmt.Errorf("failed to parse pyproject.toml: %w", err)
	}

	layout.PackageName = pyproject.Project.Name
	if layout.PackageName == "" {
		layout.PackageName = "unknown"
	}

	// Add additional dependencies from pyproject.toml
	if pyproject.Tool.UV.Sources != nil {
		for _, source := range pyproject.Tool.UV.Sources {
			if source.Path != "" {
				depPath := filepath.Join(layout.WorkspaceDir, source.Path)
				layout.Dependencies = append(layout.Dependencies, depPath)
			}
		}
	}

	// Check for uv.lock file
	uvLockPath := filepath.Join(layout.WorkspaceDir, "uv.lock")
	if _, err := os.Stat(uvLockPath); err == nil {
		layout.Dependencies = append(layout.Dependencies, uvLockPath)
	}

	return nil
}

// determineLayoutType analyzes the directory structure to determine the layout type
func (ld *LayoutDetector) determineLayoutType(layout *LayoutInfo) error {
	// First check if this is a UV workspace by parsing pyproject.toml
	pyprojectPath := filepath.Join(layout.WorkspaceDir, "pyproject.toml")
	if pyproject, err := ld.parsePyprojectToml(pyprojectPath); err == nil {
		// Check for UV workspace members
		if len(pyproject.Tool.UV.Workspace.Members) > 0 {
			layout.Type = LayoutTypeWorkspace
			layout.HasSrcDirectory = false
			layout.SourceRoot = layout.WorkspaceDir

			// Check for src layouts within workspace packages
			if err := ld.detectWorkspacePackageSrcLayouts(layout, pyproject); err != nil {
				return fmt.Errorf("failed to detect workspace package src layouts: %w", err)
			}

			return nil
		}
	}

	// Check if there's a src/ directory structure
	srcDir := filepath.Join(layout.WorkspaceDir, "src")
	if _, err := os.Stat(srcDir); err == nil {
		layout.HasSrcDirectory = true
		layout.SourceRoot = srcDir

		// Check if it follows the src/{package_name} pattern
		packageSrcDir := filepath.Join(srcDir, layout.PackageName)
		if _, err := os.Stat(packageSrcDir); err == nil {
			layout.Type = LayoutTypeWorkspace
		} else {
			layout.Type = LayoutTypeNested
		}
	} else {
		layout.HasSrcDirectory = false
		layout.SourceRoot = layout.WorkspaceDir

		// Check if it's a flat structure or nested without src/
		// Look at the actual structure to determine layout type
		if ld.hasNestedPythonStructure(layout.WorkspaceDir) {
			layout.Type = LayoutTypeNested
		} else {
			layout.Type = LayoutTypeFlat
		}
	}

	return nil
}

// hasNestedPythonStructure checks if the directory has a nested Python structure
func (ld *LayoutDetector) hasNestedPythonStructure(workspaceDir string) bool {
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return false
	}

	// Count Python files at root level vs in subdirectories
	rootPythonFiles := 0
	nestedPythonFiles := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			// Check for Python files at root level
			if strings.HasSuffix(entry.Name(), ".py") &&
				entry.Name() != "__init__.py" && // __init__.py doesn't count as a main file
				!strings.HasPrefix(entry.Name(), "test_") { // ignore test files
				rootPythonFiles++
			}
		} else {
			// Check for Python files in subdirectories
			subDir := filepath.Join(workspaceDir, entry.Name())
			if ld.containsPythonFiles(subDir) {
				nestedPythonFiles++
			}
		}
	}

	// If we have more nested Python directories than root Python files,
	// and we have at least one nested directory, it's likely a nested layout
	return nestedPythonFiles > 0 && nestedPythonFiles >= rootPythonFiles
}

// containsPythonFiles checks if a directory contains Python files (recursively)
func (ld *LayoutDetector) containsPythonFiles(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			if strings.HasSuffix(entry.Name(), ".py") {
				return true
			}
		} else {
			// Recursively check subdirectories
			subPath := filepath.Join(dirPath, entry.Name())
			if ld.containsPythonFiles(subPath) {
				return true
			}
		}
	}
	return false
}

// detectWorkspacePackageSrcLayouts checks for src/{package_name} layouts within workspace packages
func (ld *LayoutDetector) detectWorkspacePackageSrcLayouts(layout *LayoutInfo, pyproject *PyprojectToml) error {
	layout.WorkspacePackagesWithSrc = make(map[string]bool)

	// Check each workspace member for src layout
	for _, member := range pyproject.Tool.UV.Workspace.Members {
		memberPath := filepath.Join(layout.WorkspaceDir, member)

		// Read the member's pyproject.toml to get package name
		memberPyprojectPath := filepath.Join(memberPath, "pyproject.toml")
		memberPyproject, err := ld.parsePyprojectToml(memberPyprojectPath)
		if err != nil {
			continue // Skip if we can't read the member's pyproject.toml
		}

		packageName := memberPyproject.Project.Name
		if packageName == "" {
			continue // Skip if no package name
		}

		// Check if this member has src/{package_name} structure
		srcPackageDir := filepath.Join(memberPath, "src", packageName)
		if _, err := os.Stat(srcPackageDir); err == nil {
			layout.WorkspacePackagesWithSrc[packageName] = true
			layout.HasSrcDirectory = true // Mark that workspace has src layouts

			slog.Info("detected src layout in workspace package",
				"package", packageName,
				"member", member,
				"srcDir", srcPackageDir)
		}
	}

	return nil
}

// setupPythonPaths configures the Python paths for module resolution
func (ld *LayoutDetector) setupPythonPaths(layout *LayoutInfo) error {
	// Add the source root to Python path
	layout.PythonPath = append(layout.PythonPath, layout.SourceRoot)

	// Add workspace directory if different from source root
	if layout.WorkspaceDir != layout.SourceRoot {
		layout.PythonPath = append(layout.PythonPath, layout.WorkspaceDir)
	}

	return nil
}

// generateModulePath creates the Python import path for the handler
func (ld *LayoutDetector) generateModulePath(layout *LayoutInfo) error {
	// Calculate relative path from source root to handler file
	relPath, err := filepath.Rel(layout.SourceRoot, layout.HandlerFile)
	if err != nil {
		return fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Convert file path to Python module path
	modulePath := strings.TrimSuffix(relPath, ".py")
	modulePath = strings.ReplaceAll(modulePath, string(filepath.Separator), ".")

	layout.ModulePath = modulePath

	return nil
}

// ClearCache removes all cached layout information
func (ld *LayoutDetector) ClearCache() {
	ld.mutex.Lock()
	defer ld.mutex.Unlock()

	ld.cache = make(map[string]*LayoutInfo)
}

// InvalidateCache removes cached layout information for a specific handler
func (ld *LayoutDetector) InvalidateCache(handler string) {
	ld.mutex.Lock()
	defer ld.mutex.Unlock()

	delete(ld.cache, handler)
}

// GetCachedLayout returns cached layout information without triggering detection
func (ld *LayoutDetector) GetCachedLayout(handler string) (*LayoutInfo, bool) {
	ld.mutex.RLock()
	defer ld.mutex.RUnlock()

	if cached, exists := ld.cache[handler]; exists {
		if time.Since(cached.DetectedAt) < ld.cacheTimeout {
			return cached, true
		}
	}

	return nil, false
}

// PyprojectToml represents the structure of a pyproject.toml file
type PyprojectToml struct {
	Project struct {
		Name         string   `toml:"name"`
		Version      string   `toml:"version"`
		Description  string   `toml:"description"`
		Dependencies []string `toml:"dependencies"`
	} `toml:"project"`

	Tool struct {
		UV struct {
			Sources   map[string]UVSource `toml:"sources"`
			Workspace struct {
				Members []string `toml:"members"`
			} `toml:"workspace"`
		} `toml:"uv"`

		Hatch struct {
			Build struct {
				Targets struct {
					Wheel struct {
						Packages []string `toml:"packages"`
					} `toml:"wheel"`
				} `toml:"targets"`
			} `toml:"build"`
		} `toml:"hatch"`

		Setuptools struct {
			Packages struct {
				Find struct {
					Where []string `toml:"where"`
				} `toml:"find"`
			} `toml:"packages"`
		} `toml:"setuptools"`

		Poetry struct {
			Packages []struct {
				Include string `toml:"include"`
				From    string `toml:"from"`
			} `toml:"packages"`
		} `toml:"poetry"`
	} `toml:"tool"`

	BuildSystem struct {
		Requires     []string `toml:"requires"`
		BuildBackend string   `toml:"build-backend"`
	} `toml:"build-system"`
}

// UVSource represents a UV source configuration
type UVSource struct {
	Path         string `toml:"path"`
	URL          string `toml:"url"`
	Git          string `toml:"git"`
	Branch       string `toml:"branch"`
	Tag          string `toml:"tag"`
	Rev          string `toml:"rev"`
	Subdirectory string `toml:"subdirectory"`
}

// parsePyprojectToml reads and parses a pyproject.toml file
func (ld *LayoutDetector) parsePyprojectToml(path string) (*PyprojectToml, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read pyproject.toml: %w", err)
	}

	var pyproject PyprojectToml
	if err := toml.Unmarshal(content, &pyproject); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return &pyproject, nil
}

// FindAllWorkspaces discovers all pyproject.toml files in the project hierarchy
func (ld *LayoutDetector) FindAllWorkspaces() ([]string, error) {
	var workspaces []string

	err := filepath.Walk(ld.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		// Skip hidden directories and common ignore patterns
		if info.IsDir() && ld.shouldSkipDirectory(info.Name()) {
			return filepath.SkipDir
		}

		// Check for pyproject.toml files
		if !info.IsDir() && info.Name() == "pyproject.toml" {
			workspaces = append(workspaces, filepath.Dir(path))
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	return workspaces, nil
}

// ValidateWorkspaceConfiguration checks if a workspace configuration is valid
func (ld *LayoutDetector) ValidateWorkspaceConfiguration(workspaceDir string) error {
	pyprojectPath := filepath.Join(workspaceDir, "pyproject.toml")

	// Check if pyproject.toml exists
	if _, err := os.Stat(pyprojectPath); err != nil {
		return fmt.Errorf("pyproject.toml not found in workspace %s: %w", workspaceDir, err)
	}

	// Parse and validate the configuration
	pyproject, err := ld.parsePyprojectToml(pyprojectPath)
	if err != nil {
		return fmt.Errorf("invalid pyproject.toml in workspace %s: %w", workspaceDir, err)
	}

	// Validate required fields
	if pyproject.Project.Name == "" {
		return fmt.Errorf("project name is required in pyproject.toml at %s", workspaceDir)
	}

	// Validate UV sources if present
	for sourceName, source := range pyproject.Tool.UV.Sources {
		if err := ld.validateUVSource(workspaceDir, sourceName, source); err != nil {
			return fmt.Errorf("invalid UV source %s in workspace %s: %w", sourceName, workspaceDir, err)
		}
	}

	return nil
}

// validateUVSource validates a UV source configuration
func (ld *LayoutDetector) validateUVSource(workspaceDir, sourceName string, source UVSource) error {
	// Check if local path sources exist
	if source.Path != "" {
		sourcePath := filepath.Join(workspaceDir, source.Path)
		if _, err := os.Stat(sourcePath); err != nil {
			return fmt.Errorf("local source path %s does not exist", sourcePath)
		}
	}

	// Validate that only one source type is specified
	sourceTypes := 0
	if source.Path != "" {
		sourceTypes++
	}
	if source.URL != "" {
		sourceTypes++
	}
	if source.Git != "" {
		sourceTypes++
	}

	if sourceTypes == 0 {
		return fmt.Errorf("no source specified for %s", sourceName)
	}
	if sourceTypes > 1 {
		return fmt.Errorf("multiple source types specified for %s", sourceName)
	}

	return nil
}

// ResolveNestedWorkspaces finds the most appropriate workspace for a given handler
func (ld *LayoutDetector) ResolveNestedWorkspaces(handlerPath string) ([]string, error) {
	allWorkspaces, err := ld.FindAllWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("failed to find workspaces: %w", err)
	}

	var candidateWorkspaces []string

	// Find workspaces that could contain this handler
	for _, workspace := range allWorkspaces {
		// Check if the handler is within this workspace
		if strings.HasPrefix(handlerPath, workspace) {
			candidateWorkspaces = append(candidateWorkspaces, workspace)
		}
	}

	if len(candidateWorkspaces) == 0 {
		return nil, fmt.Errorf("no workspace found for handler %s", handlerPath)
	}

	// Sort by depth (most specific first)
	sort.Slice(candidateWorkspaces, func(i, j int) bool {
		depthI := len(strings.Split(candidateWorkspaces[i], string(filepath.Separator)))
		depthJ := len(strings.Split(candidateWorkspaces[j], string(filepath.Separator)))
		return depthI > depthJ // More nested workspaces first
	})

	return candidateWorkspaces, nil
}

// CreateFallbackWorkspace creates a minimal workspace configuration when none exists
func (ld *LayoutDetector) CreateFallbackWorkspace(handlerPath string) (*LayoutInfo, error) {
	// Use the directory containing the handler as the workspace
	handlerDir := filepath.Dir(handlerPath)

	// Create a minimal layout info
	layout := &LayoutInfo{
		Type:            LayoutTypeLegacy,
		HandlerFile:     handlerPath,
		WorkspaceDir:    handlerDir,
		PackageName:     "fallback",
		PythonPath:      []string{handlerDir},
		Dependencies:    []string{},
		SourceRoot:      handlerDir,
		HasSrcDirectory: false,
		DetectedAt:      time.Now(),
	}

	// Generate module path
	if err := ld.generateModulePath(layout); err != nil {
		return nil, fmt.Errorf("failed to generate module path for fallback: %w", err)
	}

	return layout, nil
}

// GetWorkspaceDependencies returns all files that affect the workspace configuration
func (ld *LayoutDetector) GetWorkspaceDependencies(workspaceDir string) ([]string, error) {
	var dependencies []string

	// Add pyproject.toml
	pyprojectPath := filepath.Join(workspaceDir, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		dependencies = append(dependencies, pyprojectPath)
	}

	// Add uv.lock if it exists
	uvLockPath := filepath.Join(workspaceDir, "uv.lock")
	if _, err := os.Stat(uvLockPath); err == nil {
		dependencies = append(dependencies, uvLockPath)
	}

	// Add requirements.txt if it exists
	requirementsPath := filepath.Join(workspaceDir, "requirements.txt")
	if _, err := os.Stat(requirementsPath); err == nil {
		dependencies = append(dependencies, requirementsPath)
	}

	// Add poetry.lock if it exists (for compatibility)
	poetryLockPath := filepath.Join(workspaceDir, "poetry.lock")
	if _, err := os.Stat(poetryLockPath); err == nil {
		dependencies = append(dependencies, poetryLockPath)
	}

	// Add Pipfile.lock if it exists (for compatibility)
	pipfileLockPath := filepath.Join(workspaceDir, "Pipfile.lock")
	if _, err := os.Stat(pipfileLockPath); err == nil {
		dependencies = append(dependencies, pipfileLockPath)
	}

	return dependencies, nil
}

// HandlerPathResolver provides robust handler path resolution for various Python project layouts
type HandlerPathResolver struct {
	detector *LayoutDetector
}

// NewHandlerPathResolver creates a new handler path resolver
func NewHandlerPathResolver(detector *LayoutDetector) *HandlerPathResolver {
	return &HandlerPathResolver{
		detector: detector,
	}
}

// ResolveHandlerPath converts a handler specification to the appropriate Python import path
func (hpr *HandlerPathResolver) ResolveHandlerPath(handler string, layout *LayoutInfo) (string, error) {
	// Parse the handler specification
	handlerSpec, err := hpr.parseHandlerSpec(handler)
	if err != nil {
		return "", fmt.Errorf("failed to parse handler specification: %w", err)
	}

	// Resolve the path based on the layout type
	switch layout.Type {
	case LayoutTypeWorkspace:
		return hpr.resolveWorkspaceHandler(handlerSpec, layout)
	case LayoutTypeFlat:
		return hpr.resolveFlatHandler(handlerSpec, layout)
	case LayoutTypeNested:
		return hpr.resolveNestedHandler(handlerSpec, layout)
	case LayoutTypeLegacy:
		return hpr.resolveLegacyHandler(handlerSpec, layout)
	default:
		return "", fmt.Errorf("unsupported layout type: %s", layout.Type)
	}
}

// HandlerSpec represents a parsed handler specification
type HandlerSpec struct {
	Module   string // Python module path (e.g., "mypackage.handler")
	Function string // Function name (e.g., "lambda_handler")
	FilePath string // Original file path from the handler string
}

// parseHandlerSpec parses a handler string into its components
func (hpr *HandlerPathResolver) parseHandlerSpec(handler string) (*HandlerSpec, error) {
	// Handler can be in various formats:
	// 1. "path/to/file.py" -> module: "path.to.file", function: "handler" (default)
	// 2. "path/to/file.function_name" -> module: "path.to.file", function: "function_name"
	// 3. "module.submodule.function_name" -> module: "module.submodule", function: "function_name"

	spec := &HandlerSpec{
		FilePath: handler,
		Function: "handler", // Default function name
	}

	// First check if it's a file path ending with .py
	if strings.HasSuffix(handler, ".py") {
		// It's a file path, convert to module path
		spec.Module = strings.TrimSuffix(handler, ".py")
		spec.Module = strings.ReplaceAll(spec.Module, string(filepath.Separator), ".")
		return spec, nil
	}

	// Check if handler contains a function specification
	if strings.Contains(handler, ".") {
		parts := strings.Split(handler, ".")
		if len(parts) >= 2 {
			// Last part might be the function name
			lastPart := parts[len(parts)-1]

			// Check if the last part looks like a function name (no path separators)
			if !strings.Contains(lastPart, string(filepath.Separator)) {
				// Check if it's likely a function name (starts with letter/underscore)
				if len(lastPart) > 0 && (lastPart[0] >= 'a' && lastPart[0] <= 'z' ||
					lastPart[0] >= 'A' && lastPart[0] <= 'Z' || lastPart[0] == '_') {
					spec.Function = lastPart
					spec.Module = strings.Join(parts[:len(parts)-1], ".")
				} else {
					// Treat the whole thing as a module path
					spec.Module = handler
				}
			} else {
				// Contains path separators, treat as file path
				spec.Module = strings.ReplaceAll(handler, string(filepath.Separator), ".")
			}
		} else {
			// Single part with dot, treat as module
			spec.Module = handler
		}
	} else {
		// No dots, treat as simple module name or file path
		if strings.Contains(handler, string(filepath.Separator)) {
			// Contains path separators, convert to module path
			spec.Module = strings.ReplaceAll(handler, string(filepath.Separator), ".")
		} else {
			// Simple module name
			spec.Module = handler
		}
	}

	return spec, nil
}

// resolveWorkspaceHandler handles workspace layout (UV workspace with src/ structure)
func (hpr *HandlerPathResolver) resolveWorkspaceHandler(spec *HandlerSpec, layout *LayoutInfo) (string, error) {
	// For workspace layouts, we need to adjust for the src/ directory structure
	modulePath := spec.Module

	// Remove package name prefix if it exists in the module path
	if layout.PackageName != "" && layout.PackageName != "unknown" {
		// Check if module starts with package name
		packagePrefix := layout.PackageName + "."
		if strings.HasPrefix(modulePath, packagePrefix) {
			modulePath = strings.TrimPrefix(modulePath, packagePrefix)
		}

		// Also handle the case where the module path includes src/package structure
		srcPackagePrefix := layout.PackageName + ".src." + layout.PackageName + "."
		if strings.HasPrefix(modulePath, srcPackagePrefix) {
			modulePath = strings.TrimPrefix(modulePath, srcPackagePrefix)
		}

		// Handle src/package pattern
		srcPrefix := "src." + layout.PackageName + "."
		if strings.HasPrefix(modulePath, srcPrefix) {
			modulePath = strings.TrimPrefix(modulePath, srcPrefix)
		}
	}

	// Construct the final handler path
	return modulePath + "." + spec.Function, nil
}

// resolveFlatHandler handles flat layout (no src/ directory)
func (hpr *HandlerPathResolver) resolveFlatHandler(spec *HandlerSpec, layout *LayoutInfo) (string, error) {
	// For flat layouts, the module path is straightforward
	return spec.Module + "." + spec.Function, nil
}

// resolveNestedHandler handles nested package layouts
func (hpr *HandlerPathResolver) resolveNestedHandler(spec *HandlerSpec, layout *LayoutInfo) (string, error) {
	// For nested layouts, we need to calculate the relative path from the source root
	modulePath := spec.Module

	// If we have a source root different from workspace dir, adjust the path
	if layout.SourceRoot != layout.WorkspaceDir {
		sourceRootRel, err := filepath.Rel(layout.WorkspaceDir, layout.SourceRoot)
		if err == nil && sourceRootRel != "." {
			sourceRootModule := strings.ReplaceAll(sourceRootRel, string(filepath.Separator), ".")
			if strings.HasPrefix(modulePath, sourceRootModule+".") {
				modulePath = strings.TrimPrefix(modulePath, sourceRootModule+".")
			}
		}
	}

	return modulePath + "." + spec.Function, nil
}

// resolveLegacyHandler handles legacy Python projects without workspace configuration
func (hpr *HandlerPathResolver) resolveLegacyHandler(spec *HandlerSpec, layout *LayoutInfo) (string, error) {
	// For legacy layouts, use the module path as-is
	return spec.Module + "." + spec.Function, nil
}

// ValidateHandlerPath checks if a resolved handler path is valid
func (hpr *HandlerPathResolver) ValidateHandlerPath(handlerPath string, layout *LayoutInfo) error {
	// Split handler path into module and function
	parts := strings.Split(handlerPath, ".")
	if len(parts) < 2 {
		return fmt.Errorf("handler path must contain at least module and function: %s", handlerPath)
	}

	functionName := parts[len(parts)-1]
	modulePath := strings.Join(parts[:len(parts)-1], ".")

	// Validate function name
	if err := hpr.validateFunctionName(functionName); err != nil {
		return fmt.Errorf("invalid function name %s: %w", functionName, err)
	}

	// Validate module path
	if err := hpr.validateModulePath(modulePath, layout); err != nil {
		return fmt.Errorf("invalid module path %s: %w", modulePath, err)
	}

	return nil
}

// validateFunctionName checks if a function name is valid Python identifier
func (hpr *HandlerPathResolver) validateFunctionName(functionName string) error {
	if functionName == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	// Check if it's a valid Python identifier
	if !hpr.isValidPythonIdentifier(functionName) {
		return fmt.Errorf("function name must be a valid Python identifier")
	}

	return nil
}

// validateModulePath checks if a module path is valid and accessible
func (hpr *HandlerPathResolver) validateModulePath(modulePath string, layout *LayoutInfo) error {
	if modulePath == "" {
		return fmt.Errorf("module path cannot be empty")
	}

	// Check if module path contains valid Python identifiers
	parts := strings.Split(modulePath, ".")
	for _, part := range parts {
		if !hpr.isValidPythonIdentifier(part) {
			return fmt.Errorf("module path part %s must be a valid Python identifier", part)
		}
	}

	// Try to find the corresponding Python file
	expectedFilePath := filepath.Join(layout.SourceRoot, strings.ReplaceAll(modulePath, ".", string(filepath.Separator))+".py")
	if _, err := os.Stat(expectedFilePath); err != nil {
		// Also try __init__.py in a package directory
		packagePath := filepath.Join(layout.SourceRoot, strings.ReplaceAll(modulePath, ".", string(filepath.Separator)), "__init__.py")
		if _, err := os.Stat(packagePath); err != nil {
			return fmt.Errorf("module file not found: neither %s nor %s exists", expectedFilePath, packagePath)
		}
	}

	return nil
}

// isValidPythonIdentifier checks if a string is a valid Python identifier
func (hpr *HandlerPathResolver) isValidPythonIdentifier(identifier string) bool {
	if identifier == "" {
		return false
	}

	// Python identifier rules:
	// - Must start with letter or underscore
	// - Can contain letters, digits, and underscores
	// - Cannot be a Python keyword

	// Check first character
	first := identifier[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	// Check remaining characters
	for i := 1; i < len(identifier); i++ {
		char := identifier[i]
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}

	// Check if it's a Python keyword
	pythonKeywords := []string{
		"False", "None", "True", "and", "as", "assert", "async", "await", "break", "class",
		"continue", "def", "del", "elif", "else", "except", "finally", "for", "from", "global",
		"if", "import", "in", "is", "lambda", "nonlocal", "not", "or", "pass", "raise",
		"return", "try", "while", "with", "yield",
	}

	for _, keyword := range pythonKeywords {
		if identifier == keyword {
			return false
		}
	}

	return true
}

// GetImportPath returns the Python import path for a given handler
func (hpr *HandlerPathResolver) GetImportPath(handler string, layout *LayoutInfo) (string, error) {
	resolvedPath, err := hpr.ResolveHandlerPath(handler, layout)
	if err != nil {
		return "", err
	}

	// Extract just the module part (without function)
	parts := strings.Split(resolvedPath, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid resolved path: %s", resolvedPath)
	}

	return strings.Join(parts[:len(parts)-1], "."), nil
}

// GetFunctionName returns the function name from a handler specification
func (hpr *HandlerPathResolver) GetFunctionName(handler string) (string, error) {
	spec, err := hpr.parseHandlerSpec(handler)
	if err != nil {
		return "", err
	}

	return spec.Function, nil
}

// NormalizeHandlerPath converts various handler path formats to a consistent format
func (hpr *HandlerPathResolver) NormalizeHandlerPath(handler string, layout *LayoutInfo) (string, error) {
	// Resolve the handler path
	resolvedPath, err := hpr.ResolveHandlerPath(handler, layout)
	if err != nil {
		return "", fmt.Errorf("failed to resolve handler path: %w", err)
	}

	// Validate the resolved path
	if err := hpr.ValidateHandlerPath(resolvedPath, layout); err != nil {
		return "", fmt.Errorf("invalid resolved handler path: %w", err)
	}

	return resolvedPath, nil
}

// SupportedHandlerFormats returns examples of supported handler formats
func (hpr *HandlerPathResolver) SupportedHandlerFormats() []string {
	return []string{
		"handler.py",                    // File path with default function
		"src/mypackage/handler.py",      // Nested file path
		"mypackage.handler",             // Module path with default function
		"mypackage.handler.my_function", // Module path with specific function
		"handler.lambda_handler",        // File with specific function
		"app/api/routes.handle_request", // Nested module with function
	}
}
