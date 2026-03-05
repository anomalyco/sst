package python

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ContentFilter filters out unnecessary files and directories from deployment artifacts
type ContentFilter struct {
	excludePatterns []string
	logger          *slog.Logger
	projectRoot     string
	pyprojectCache  *PyprojectConfig // Cache for parsed pyproject.toml
}

// NewContentFilter creates a new content filter with default exclude patterns
func NewContentFilter() *ContentFilter {
	return &ContentFilter{
		excludePatterns: getDefaultExcludePatterns(),
		logger:          slog.Default(),
	}
}

// NewContentFilterForProject creates a content filter for a specific project
func NewContentFilterForProject(projectRoot string) *ContentFilter {
	return &ContentFilter{
		excludePatterns: getDefaultExcludePatterns(),
		logger:          slog.Default(),
		projectRoot:     projectRoot,
	}
}

// getDefaultExcludePatterns returns the default patterns to exclude from deployment artifacts
func getDefaultExcludePatterns() []string {
	return []string{
		// SST build artifacts and cache
		".sst",
		".sst/**",

		// Version control
		".git",
		".git/**",
		".gitignore",
		".gitattributes",

		// Python cache and build artifacts
		"__pycache__",
		"__pycache__/**",
		"*.pyc",
		"*.pyo",
		"*.pyd",
		".pytest_cache",
		".pytest_cache/**",
		"*.egg-info",
		"*.egg-info/**",
		".coverage",
		"htmlcov",
		"htmlcov/**",

		// Virtual environments
		".venv",
		".venv/**",
		"venv",
		"venv/**",
		".env",
		"env",
		"env/**",

		// IDE and editor files
		".vscode",
		".vscode/**",
		".idea",
		".idea/**",
		"*.swp",
		"*.swo",
		"*~",
		".DS_Store",

		// Node.js (in case of mixed projects)
		"node_modules",
		"node_modules/**",
		"package-lock.json",
		"yarn.lock",
		"bun.lockb",

		// Documentation and development files
		"README.md",
		"README.rst",
		"README.txt",
		"CHANGELOG.md",
		"CHANGELOG.rst",
		"CHANGELOG.txt",
		"LICENSE",
		"LICENSE.txt",
		"MANIFEST.in",
		"setup.cfg",
		"tox.ini",
		"Makefile",
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.yaml",

		// Test directories
		"tests",
		"tests/**",
		"test",
		"test/**",

		// Configuration files that shouldn't be deployed
		"pyproject.toml",
		"setup.py",
		"requirements-dev.txt",
		"requirements.dev.txt",
		"dev-requirements.txt",
		".python-version",
		".pre-commit-config.yaml",

		// Temporary and log files
		"*.log",
		"*.tmp",
		"tmp",
		"tmp/**",
		"temp",
		"temp/**",
	}
}

// ShouldExclude checks if a file or directory should be excluded using a two-layer approach:
// 1. Check pyproject.toml [tool.sst] configuration first (explicit control)
// 2. Apply standard Python conventions (sensible defaults)
func (cf *ContentFilter) ShouldExclude(path string) bool {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)

	// 1. Check pyproject.toml [tool.sst] configuration first
	if shouldSkip, found := cf.checkPyprojectConfig(normalizedPath); found {
		if shouldSkip {
			cf.logger.Debug("excluding file by pyproject.toml configuration",
				"path", path,
				"source", "pyproject.toml")
		} else {
			cf.logger.Debug("including file by pyproject.toml configuration",
				"path", path,
				"source", "pyproject.toml")
		}
		return shouldSkip
	}

	// 2. Apply standard Python conventions (baseline)
	for _, pattern := range cf.excludePatterns {
		if cf.matchesPattern(normalizedPath, pattern) {
			cf.logger.Debug("excluding file by standard pattern",
				"path", path,
				"pattern", pattern,
				"source", "standard")
			return true
		}
	}

	return false
}

// matchesPattern checks if a path matches a glob pattern with support for ** and directory patterns
func (cf *ContentFilter) matchesPattern(path, pattern string) bool {
	// Handle directory patterns ending with /
	if strings.HasSuffix(pattern, "/") {
		dirPattern := strings.TrimSuffix(pattern, "/")
		// Check if path is under this directory
		return strings.HasPrefix(path, dirPattern+"/") || path == dirPattern
	}

	// Handle ** patterns for recursive matching
	if strings.Contains(pattern, "**") {
		return cf.matchesGlobPattern(path, pattern)
	}

	// Simple pattern matching
	baseName := filepath.Base(path)

	// Direct name match
	if baseName == pattern {
		return true
	}

	// Pattern match with wildcards on basename
	if matched, err := filepath.Match(pattern, baseName); err == nil && matched {
		return true
	}

	// Full path pattern match
	if matched, err := filepath.Match(pattern, path); err == nil && matched {
		return true
	}

	// Check if any directory in the path matches the pattern (for directory exclusions)
	pathParts := strings.Split(path, "/")
	for _, part := range pathParts {
		if part == pattern {
			return true
		}
		// Also check pattern matching on directory names
		if matched, err := filepath.Match(pattern, part); err == nil && matched {
			return true
		}
	}

	return false
}

// matchesGlobPattern handles ** patterns for recursive directory matching
func (cf *ContentFilter) matchesGlobPattern(path, pattern string) bool {
	// Split pattern by **
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		// Invalid pattern, fallback to simple match
		if matched, err := filepath.Match(pattern, path); err == nil {
			return matched
		}
		return false
	}

	prefix := parts[0]
	suffix := parts[1]

	// Remove leading/trailing slashes from suffix
	suffix = strings.Trim(suffix, "/")

	// Check if path starts with prefix (if any)
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/")
		if !strings.HasPrefix(path, prefix) {
			return false
		}
		// Remove the prefix from path for suffix matching
		path = strings.TrimPrefix(path, prefix)
		path = strings.TrimPrefix(path, "/")
	}

	// If no suffix, any path under prefix matches
	if suffix == "" {
		return true
	}

	// Check if any part of the remaining path matches the suffix
	pathParts := strings.Split(path, "/")
	for i := 0; i < len(pathParts); i++ {
		remainingPath := strings.Join(pathParts[i:], "/")
		if matched, err := filepath.Match(suffix, remainingPath); err == nil && matched {
			return true
		}
		// Also check individual path components
		if matched, err := filepath.Match(suffix, pathParts[i]); err == nil && matched {
			return true
		}
	}

	return false
}

// FilterDirectory filters files in a directory, copying only allowed files to the target
func (cf *ContentFilter) FilterDirectory(sourceDir, targetDir string) error {
	cf.logger.Info("filtering directory contents",
		"sourceDir", sourceDir,
		"targetDir", targetDir)

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	var totalFiles, excludedFiles, copiedFiles int

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			cf.logger.Warn("error walking directory",
				"path", path,
				"error", err)
			return nil // Continue walking despite errors
		}

		// Skip the source directory itself
		if path == sourceDir {
			return nil
		}

		totalFiles++

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			cf.logger.Error("failed to get relative path",
				"path", path,
				"sourceDir", sourceDir,
				"error", err)
			return nil
		}

		// Check if this file/directory should be excluded
		if cf.ShouldExclude(relPath) {
			excludedFiles++
			cf.logger.Debug("excluding path",
				"path", relPath,
				"fullPath", path)

			// If it's a directory, skip walking into it
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate target path
		targetPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			// Create directory in target
			if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
				cf.logger.Error("failed to create target directory",
					"targetPath", targetPath,
					"error", err)
				return nil
			}
			cf.logger.Debug("created directory",
				"targetPath", targetPath)
		} else {
			// Copy file to target
			if err := cf.copyFile(path, targetPath); err != nil {
				cf.logger.Error("failed to copy file",
					"sourcePath", path,
					"targetPath", targetPath,
					"error", err)
				return nil
			}
			copiedFiles++
			cf.logger.Debug("copied file",
				"sourcePath", path,
				"targetPath", targetPath,
				"size", info.Size())
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking source directory: %w", err)
	}

	cf.logger.Info("directory filtering completed",
		"sourceDir", sourceDir,
		"targetDir", targetDir,
		"totalFiles", totalFiles,
		"excludedFiles", excludedFiles,
		"copiedFiles", copiedFiles)

	return nil
}

// copyFile copies a single file from src to dst
func (cf *ContentFilter) copyFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy file contents
	if _, err := srcFile.WriteTo(dstFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

// GetExcludePatterns returns the current exclude patterns
func (cf *ContentFilter) GetExcludePatterns() []string {
	return cf.excludePatterns
}

// checkPyprojectConfig checks if a file should be included/excluded based on pyproject.toml [tool.sst] configuration
// Returns (shouldExclude, found) where found indicates if explicit configuration was found
func (cf *ContentFilter) checkPyprojectConfig(path string) (bool, bool) {
	if cf.projectRoot == "" {
		return false, false
	}

	// Load pyproject.toml if not cached
	if cf.pyprojectCache == nil {
		cf.loadPyprojectConfig()
	}

	// If no pyproject.toml or no [tool.sst] section, return not found
	if cf.pyprojectCache == nil {
		return false, false
	}

	// Check explicit include patterns first
	for _, pattern := range cf.pyprojectCache.Tool.SST.Include {
		if cf.matchesPattern(path, pattern) {
			return false, true // Explicitly include
		}
	}

	// Check explicit exclude patterns
	for _, pattern := range cf.pyprojectCache.Tool.SST.Exclude {
		if cf.matchesPattern(path, pattern) {
			return true, true // Explicitly exclude
		}
	}

	// No explicit configuration found
	return false, false
}

// loadPyprojectConfig loads and parses the pyproject.toml file
func (cf *ContentFilter) loadPyprojectConfig() {
	pyprojectPath := filepath.Join(cf.projectRoot, "pyproject.toml")

	// Check if file exists
	if _, err := os.Stat(pyprojectPath); os.IsNotExist(err) {
		cf.logger.Debug("no pyproject.toml found", "path", pyprojectPath)
		return
	}

	// Read and parse the file
	var config PyprojectConfig
	if _, err := toml.DecodeFile(pyprojectPath, &config); err != nil {
		cf.logger.Warn("failed to parse pyproject.toml",
			"path", pyprojectPath,
			"error", err)
		return
	}

	// Only cache if there's actually SST configuration
	if len(config.Tool.SST.Include) > 0 || len(config.Tool.SST.Exclude) > 0 {
		cf.pyprojectCache = &config
		cf.logger.Debug("loaded pyproject.toml SST configuration",
			"path", pyprojectPath,
			"includePatterns", len(config.Tool.SST.Include),
			"excludePatterns", len(config.Tool.SST.Exclude))
	} else {
		cf.logger.Debug("no [tool.sst] configuration found in pyproject.toml",
			"path", pyprojectPath)
	}
}
