package python

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ChangeDetector monitors file modifications and determines if rebuilds are needed
type ChangeDetector struct {
	// projectResolver is used to resolve project structure and dependencies
	projectResolver *ProjectResolver

	// buildCache stores build state and file hashes
	buildCache *BuildCache

	// mutex protects concurrent access
	mutex sync.RWMutex

	// watchPatterns defines file patterns to watch for changes
	watchPatterns []string

	// ignorePatterns defines file patterns to ignore
	ignorePatterns []string
}

// ChangeDetectorConfig configures the change detector
type ChangeDetectorConfig struct {
	// ProjectResolver for project analysis
	ProjectResolver *ProjectResolver

	// BuildCache for storing file hashes and build state
	BuildCache *BuildCache

	// WatchPatterns defines additional file patterns to watch
	WatchPatterns []string

	// IgnorePatterns defines file patterns to ignore
	IgnorePatterns []string
}

// ChangeResult represents the result of change detection
type ChangeResult struct {
	// HasChanges indicates if any relevant changes were detected
	HasChanges bool

	// ChangedFiles contains the list of files that changed
	ChangedFiles []string

	// ChangeTypes categorizes the types of changes detected
	ChangeTypes []ChangeType

	// Reason provides a human-readable explanation of why a rebuild is needed
	Reason string

	// AffectedDependencies lists dependencies that were affected by changes
	AffectedDependencies []string
}

// ChangeType represents different types of changes that can trigger rebuilds
type ChangeType string

const (
	ChangeTypeSourceCode     ChangeType = "source_code"     // Python source files changed
	ChangeTypeDependencies   ChangeType = "dependencies"    // pyproject.toml, uv.lock, requirements.txt changed
	ChangeTypeConfiguration  ChangeType = "configuration"   // Build configuration changed
	ChangeTypeBuildArtifacts ChangeType = "build_artifacts" // Build artifacts missing or corrupted
	ChangeTypeForced         ChangeType = "forced"          // Forced rebuild requested
)

// NewChangeDetector creates a new change detector with the given configuration
func NewChangeDetector(config ChangeDetectorConfig) (*ChangeDetector, error) {
	if config.ProjectResolver == nil {
		return nil, fmt.Errorf("project resolver is required")
	}

	if config.BuildCache == nil {
		return nil, fmt.Errorf("build cache is required")
	}

	// Default watch patterns for Python projects
	defaultWatchPatterns := []string{
		"*.py",
		"pyproject.toml",
		"uv.lock",
		"requirements.txt",
		"poetry.lock",
		"Pipfile.lock",
		"setup.py",
		"setup.cfg",
		"Dockerfile",
	}

	// Default ignore patterns
	defaultIgnorePatterns := []string{
		"__pycache__/*",
		"*.pyc",
		"*.pyo",
		"*.pyd",
		".pytest_cache/*",
		".mypy_cache/*",
		".coverage",
		"*.egg-info/*",
		"build/*",
		"dist/*",
		".git/*",
		".svn/*",
		".hg/*",
		"node_modules/*",
		".venv/*",
		"venv/*",
		"env/*",
	}

	watchPatterns := append(defaultWatchPatterns, config.WatchPatterns...)
	ignorePatterns := append(defaultIgnorePatterns, config.IgnorePatterns...)

	return &ChangeDetector{
		projectResolver: config.ProjectResolver,
		buildCache:      config.BuildCache,
		watchPatterns:   watchPatterns,
		ignorePatterns:  ignorePatterns,
	}, nil
}

// DetectChanges analyzes if a function needs to be rebuilt based on file changes
func (cd *ChangeDetector) DetectChanges(functionID, handler string) (*ChangeResult, error) {
	cd.mutex.RLock()
	defer cd.mutex.RUnlock()

	result := &ChangeResult{
		HasChanges:           false,
		ChangedFiles:         []string{},
		ChangeTypes:          []ChangeType{},
		AffectedDependencies: []string{},
	}

	// Get cached build information
	cacheEntry, hasCachedBuild := cd.buildCache.Get(functionID)
	if !hasCachedBuild {
		result.HasChanges = true
		result.ChangeTypes = append(result.ChangeTypes, ChangeTypeBuildArtifacts)
		result.Reason = "No cached build found - initial build required"
		return result, nil
	}

	// Use cached project info if available, otherwise resolve project structure
	var projectInfo *ProjectInfo
	if cacheEntry.ProjectInfo != nil {
		projectInfo = cacheEntry.ProjectInfo
	} else {
		// Fallback to resolving project structure (for older cache entries)
		var err error
		projectInfo, err = cd.projectResolver.ResolveHandler(handler)
		if err != nil {
			result.HasChanges = true
			result.ChangeTypes = append(result.ChangeTypes, ChangeTypeConfiguration)
			result.Reason = fmt.Sprintf("Failed to resolve project structure: %v", err)
			return result, nil
		}
	}

	// Check if project structure has changed (simplified approach)
	if cd.hasProjectStructureChanged(cacheEntry, projectInfo) {
		result.HasChanges = true
		result.ChangeTypes = append(result.ChangeTypes, ChangeTypeConfiguration)
		result.Reason = "Project configuration has changed"
		return result, nil
	}

	// Check if cache entry is still valid
	isValid, err := cd.buildCache.IsValid(cacheEntry)
	if err != nil {
		result.HasChanges = true
		result.ChangeTypes = append(result.ChangeTypes, ChangeTypeBuildArtifacts)
		result.Reason = fmt.Sprintf("Failed to validate cache: %v", err)
		return result, nil
	}

	if !isValid {
		// Determine what changed
		changedFiles, changeTypes := cd.analyzeFileChanges(cacheEntry, projectInfo)
		result.HasChanges = true
		result.ChangedFiles = changedFiles
		result.ChangeTypes = changeTypes
		result.Reason = cd.buildChangeReason(changedFiles, changeTypes)
		result.AffectedDependencies = cd.findAffectedDependencies(changedFiles, projectInfo)
		return result, nil
	}

	// Check for missing build artifacts
	if cd.hasMissingBuildArtifacts(cacheEntry) {
		result.HasChanges = true
		result.ChangeTypes = append(result.ChangeTypes, ChangeTypeBuildArtifacts)
		result.Reason = "Build artifacts are missing or corrupted"
		return result, nil
	}

	result.Reason = "No changes detected - using cached build"
	return result, nil
}

// hasProjectStructureChanged checks if the project structure has changed since the last build
// This is a simplified version that focuses on key structural changes
func (cd *ChangeDetector) hasProjectStructureChanged(cacheEntry *CacheEntry, projectInfo *ProjectInfo) bool {
	// Check if we have cached project info to compare against
	if cacheEntry.ProjectInfo == nil {
		// No cached project info, assume structure changed
		return true
	}

	cached := cacheEntry.ProjectInfo

	// Compare key structural elements
	if cached.SourceRoot != projectInfo.SourceRoot {
		return true
	}

	if cached.ProjectRoot != projectInfo.ProjectRoot {
		return true
	}

	if cached.ModulePath != projectInfo.ModulePath {
		return true
	}

	// Compare Python paths (slice comparison)
	if len(cached.PythonPath) != len(projectInfo.PythonPath) {
		return true
	}
	for i, path := range cached.PythonPath {
		if path != projectInfo.PythonPath[i] {
			return true
		}
	}

	return false
}

// analyzeFileChanges determines which files have changed and categorizes the changes
func (cd *ChangeDetector) analyzeFileChanges(cacheEntry *CacheEntry, projectInfo *ProjectInfo) ([]string, []ChangeType) {
	var changedFiles []string
	var changeTypes []ChangeType
	changeTypeSet := make(map[ChangeType]bool)

	// Check each file in the cache
	for filePath, expectedHash := range cacheEntry.FileHashes {
		currentHash, err := cd.buildCache.calculateFileHash(filePath)
		if err != nil {
			// File might be deleted or inaccessible
			changedFiles = append(changedFiles, filePath)
			changeType := cd.categorizeFileChange(filePath)
			if !changeTypeSet[changeType] {
				changeTypes = append(changeTypes, changeType)
				changeTypeSet[changeType] = true
			}
			continue
		}

		if currentHash != expectedHash {
			changedFiles = append(changedFiles, filePath)
			changeType := cd.categorizeFileChange(filePath)
			if !changeTypeSet[changeType] {
				changeTypes = append(changeTypes, changeType)
				changeTypeSet[changeType] = true
			}
		}
	}

	// Check for new files that should be tracked
	newFiles := cd.findNewRelevantFiles(cacheEntry, projectInfo)
	for _, filePath := range newFiles {
		changedFiles = append(changedFiles, filePath)
		changeType := cd.categorizeFileChange(filePath)
		if !changeTypeSet[changeType] {
			changeTypes = append(changeTypes, changeType)
			changeTypeSet[changeType] = true
		}
	}

	return changedFiles, changeTypes
}

// categorizeFileChange determines the type of change based on the file path
func (cd *ChangeDetector) categorizeFileChange(filePath string) ChangeType {
	filename := filepath.Base(filePath)

	// Dependency files
	dependencyFiles := []string{
		"pyproject.toml",
		"uv.lock",
		"requirements.txt",
		"poetry.lock",
		"Pipfile.lock",
		"setup.py",
		"setup.cfg",
	}

	for _, depFile := range dependencyFiles {
		if filename == depFile {
			return ChangeTypeDependencies
		}
	}

	// Configuration files
	configFiles := []string{
		"Dockerfile",
		".dockerignore",
		"tox.ini",
		"pytest.ini",
		"mypy.ini",
	}

	for _, configFile := range configFiles {
		if filename == configFile {
			return ChangeTypeConfiguration
		}
	}

	// Python source files
	if strings.HasSuffix(filePath, ".py") {
		return ChangeTypeSourceCode
	}

	return ChangeTypeConfiguration
}

// findNewRelevantFiles finds new files that should be tracked but aren't in the cache
func (cd *ChangeDetector) findNewRelevantFiles(cacheEntry *CacheEntry, projectInfo *ProjectInfo) []string {
	var newFiles []string

	// Get all relevant files in the project
	relevantFiles, err := cd.findRelevantFiles(projectInfo)
	if err != nil {
		return newFiles // Return empty list on error
	}

	// Find files not in cache
	for _, filePath := range relevantFiles {
		if _, exists := cacheEntry.FileHashes[filePath]; !exists {
			newFiles = append(newFiles, filePath)
		}
	}

	return newFiles
}

// findRelevantFiles finds all files that should be tracked for changes
func (cd *ChangeDetector) findRelevantFiles(projectInfo *ProjectInfo) ([]string, error) {
	var relevantFiles []string

	// Start from the source root directory
	sourceRoot := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		sourceRoot = filepath.Dir(projectInfo.PyprojectPath)
	}

	err := filepath.Walk(sourceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			// Check if we should skip this directory
			if cd.shouldIgnoreFile(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file should be ignored
		if cd.shouldIgnoreFile(path) {
			return nil
		}

		// Check if file matches watch patterns
		if cd.shouldWatchFile(path) {
			relevantFiles = append(relevantFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	return relevantFiles, nil
}

// shouldWatchFile determines if a file should be watched for changes
func (cd *ChangeDetector) shouldWatchFile(filePath string) bool {
	filename := filepath.Base(filePath)

	for _, pattern := range cd.watchPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}

		// Also check full path for patterns with directories
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}

	return false
}

// shouldIgnoreFile determines if a file should be ignored
func (cd *ChangeDetector) shouldIgnoreFile(filePath string) bool {
	filename := filepath.Base(filePath)

	for _, pattern := range cd.ignorePatterns {
		// Remove trailing /* from pattern for directory matching
		cleanPattern := strings.TrimSuffix(pattern, "/*")

		// Check exact filename match
		if matched, _ := filepath.Match(cleanPattern, filename); matched {
			return true
		}

		// Check if file is in a directory that should be ignored
		pathParts := strings.Split(filePath, string(filepath.Separator))
		for _, part := range pathParts {
			if matched, _ := filepath.Match(cleanPattern, part); matched {
				return true
			}
		}

		// Check if any parent directory matches the pattern
		dir := filepath.Dir(filePath)
		for dir != "." && dir != filepath.Dir(dir) {
			dirName := filepath.Base(dir)
			if matched, _ := filepath.Match(cleanPattern, dirName); matched {
				return true
			}
			dir = filepath.Dir(dir)
		}
	}

	return false
}

// hasMissingBuildArtifacts checks if build artifacts are missing
func (cd *ChangeDetector) hasMissingBuildArtifacts(cacheEntry *CacheEntry) bool {
	if cacheEntry.BuildOutput == nil {
		return true
	}

	// Check if output directory exists
	if cacheEntry.BuildOutput.OutputDir != "" {
		if _, err := os.Stat(cacheEntry.BuildOutput.OutputDir); err != nil {
			return true
		}
	}

	// Check if artifact files exist
	for _, artifactPath := range cacheEntry.BuildOutput.ArtifactPaths {
		if _, err := os.Stat(artifactPath); err != nil {
			return true
		}
	}

	return false
}

// buildChangeReason creates a human-readable reason for why a rebuild is needed
func (cd *ChangeDetector) buildChangeReason(changedFiles []string, changeTypes []ChangeType) string {
	if len(changedFiles) == 0 {
		return "Unknown changes detected"
	}

	// Categorize changes for better messaging
	var reasons []string

	for _, changeType := range changeTypes {
		switch changeType {
		case ChangeTypeSourceCode:
			reasons = append(reasons, "Python source files changed")
		case ChangeTypeDependencies:
			reasons = append(reasons, "Dependencies changed")
		case ChangeTypeConfiguration:
			reasons = append(reasons, "Configuration files changed")
		case ChangeTypeBuildArtifacts:
			reasons = append(reasons, "Build artifacts missing")
		}
	}

	if len(reasons) == 1 {
		return reasons[0]
	}

	return fmt.Sprintf("Multiple changes: %s", strings.Join(reasons, ", "))
}

// findAffectedDependencies determines which dependencies are affected by the changes
func (cd *ChangeDetector) findAffectedDependencies(changedFiles []string, projectInfo *ProjectInfo) []string {
	var affected []string
	affectedSet := make(map[string]bool)

	for _, filePath := range changedFiles {
		filename := filepath.Base(filePath)

		// Check if it's a dependency file
		dependencyFiles := []string{
			"pyproject.toml",
			"uv.lock",
			"requirements.txt",
			"poetry.lock",
			"Pipfile.lock",
		}

		for _, depFile := range dependencyFiles {
			if filename == depFile && !affectedSet[filePath] {
				affected = append(affected, filePath)
				affectedSet[filePath] = true
			}
		}
	}

	return affected
}

// ForceRebuild creates a change result that forces a rebuild
func (cd *ChangeDetector) ForceRebuild(reason string) *ChangeResult {
	return &ChangeResult{
		HasChanges:  true,
		ChangeTypes: []ChangeType{ChangeTypeForced},
		Reason:      fmt.Sprintf("Forced rebuild: %s", reason),
	}
}

// GetWatchPatterns returns the current watch patterns
func (cd *ChangeDetector) GetWatchPatterns() []string {
	cd.mutex.RLock()
	defer cd.mutex.RUnlock()

	patterns := make([]string, len(cd.watchPatterns))
	copy(patterns, cd.watchPatterns)
	return patterns
}

// GetIgnorePatterns returns the current ignore patterns
func (cd *ChangeDetector) GetIgnorePatterns() []string {
	cd.mutex.RLock()
	defer cd.mutex.RUnlock()

	patterns := make([]string, len(cd.ignorePatterns))
	copy(patterns, cd.ignorePatterns)
	return patterns
}

// AddWatchPattern adds a new file pattern to watch
func (cd *ChangeDetector) AddWatchPattern(pattern string) {
	cd.mutex.Lock()
	defer cd.mutex.Unlock()

	cd.watchPatterns = append(cd.watchPatterns, pattern)
}

// AddIgnorePattern adds a new file pattern to ignore
func (cd *ChangeDetector) AddIgnorePattern(pattern string) {
	cd.mutex.Lock()
	defer cd.mutex.Unlock()

	cd.ignorePatterns = append(cd.ignorePatterns, pattern)
}

// UpdateCacheAfterBuild updates the cache with new build information
func (cd *ChangeDetector) UpdateCacheAfterBuild(functionID, handler string, projectInfo *ProjectInfo, buildOutput *CachedBuildOutput) error {
	startTime := time.Now()

	// Find all relevant files
	slog.Info("⏱️ Finding relevant files for cache...", "functionID", functionID)
	relevantFiles, err := cd.findRelevantFiles(projectInfo)
	if err != nil {
		return fmt.Errorf("failed to find relevant files: %w", err)
	}
	slog.Info("⏱️ Found relevant files", "functionID", functionID, "count", len(relevantFiles), "elapsed", time.Since(startTime))

	// Create new cache entry
	entry := &CacheEntry{
		FunctionID:   functionID,
		Handler:      handler,
		Dependencies: projectInfo.Dependencies,
		BuildOutput:  buildOutput,
		ProjectInfo:  projectInfo,
		Properties:   make(map[string]any),
	}

	// Update file hashes
	hashStart := time.Now()
	slog.Info("⏱️ Updating file hashes...", "functionID", functionID, "fileCount", len(relevantFiles))
	if err := cd.buildCache.UpdateFileHashes(entry, relevantFiles); err != nil {
		return fmt.Errorf("failed to update file hashes: %w", err)
	}
	slog.Info("⏱️ File hashes updated", "functionID", functionID, "elapsed", time.Since(hashStart))

	// Store in cache
	cacheStart := time.Now()
	slog.Info("⏱️ Storing cache entry...", "functionID", functionID)
	err = cd.buildCache.Set(functionID, entry)
	slog.Info("⏱️ Cache entry stored", "functionID", functionID, "elapsed", time.Since(cacheStart), "totalElapsed", time.Since(startTime))
	return err
}
