package python

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ContentFilter filters out unnecessary files and directories from deployment artifacts
type ContentFilter struct {
	excludePatterns []string
	projectRoot     string
	pyprojectCache  *PyprojectConfig
}

// NewContentFilter creates a new content filter with default exclude patterns
func NewContentFilter() *ContentFilter {
	return &ContentFilter{
		excludePatterns: getDefaultExcludePatterns(),
	}
}

// NewContentFilterForProject creates a content filter for a specific project
func NewContentFilterForProject(projectRoot string) *ContentFilter {
	return &ContentFilter{
		excludePatterns: getDefaultExcludePatterns(),
		projectRoot:     projectRoot,
	}
}

// getDefaultExcludePatterns returns the default patterns to exclude from deployment artifacts.
// Directory names (e.g. ".git") automatically match all files underneath them.
func getDefaultExcludePatterns() []string {
	return []string{
		// SST / VCS
		".sst", ".git", ".gitignore", ".gitattributes",

		// Python cache and build artifacts
		"__pycache__", "*.pyc", "*.pyo", "*.pyd",
		".pytest_cache", "*.egg-info", ".coverage", "htmlcov",

		// Virtual environments
		".venv", "venv", ".env", "env",

		// IDE / editor
		".vscode", ".idea", "*.swp", "*.swo", "*~", ".DS_Store",

		// Node.js (mixed projects)
		"node_modules", "package-lock.json", "yarn.lock", "bun.lockb",

		// Docs and dev files
		"README.md", "README.rst", "README.txt",
		"CHANGELOG.md", "CHANGELOG.rst", "CHANGELOG.txt",
		"LICENSE", "LICENSE.txt", "MANIFEST.in",
		"setup.cfg", "tox.ini", "Makefile",
		"Dockerfile", "docker-compose.yml", "docker-compose.yaml",

		// Test directories
		"tests", "test",

		// Config files not needed at runtime
		"pyproject.toml", "setup.py",
		"requirements-dev.txt", "requirements.dev.txt", "dev-requirements.txt",
		".python-version", ".pre-commit-config.yaml",

		// Temporary / log files
		"*.log", "*.tmp", "tmp", "temp",
	}
}

// ShouldExclude checks if a file or directory should be excluded using a two-layer approach:
// 1. Check pyproject.toml [tool.sst] configuration first (explicit control)
// 2. Apply standard Python conventions (sensible defaults)
func (cf *ContentFilter) ShouldExclude(path string) bool {
	normalizedPath := filepath.ToSlash(path)

	// Check pyproject.toml [tool.sst] overrides first
	if shouldSkip, found := cf.checkPyprojectConfig(normalizedPath); found {
		return shouldSkip
	}

	// Apply standard exclusion patterns
	for _, pattern := range cf.excludePatterns {
		if cf.matchesPattern(normalizedPath, pattern) {
			return true
		}
	}
	return false
}

// matchesPattern checks if a path matches a pattern.
// Supports wildcards (*.pyc), directory names (.git matches .git/anything),
// and ** glob patterns (stripped to match the non-** portion against path components).
func (cf *ContentFilter) matchesPattern(path, pattern string) bool {
	// Handle directory patterns ending with /
	if dir, ok := strings.CutSuffix(pattern, "/"); ok {
		return strings.HasPrefix(path, dir+"/") || path == dir
	}

	// Strip ** from glob patterns — the component-level matching below handles recursion
	if strings.Contains(pattern, "**") {
		pattern = strings.ReplaceAll(pattern, "**/", "")
		pattern = strings.ReplaceAll(pattern, "/**", "")
		pattern = strings.ReplaceAll(pattern, "**", "")
		if pattern == "" {
			return true
		}
	}

	// Check each path component against the pattern
	for _, part := range strings.Split(path, "/") {
		if part == pattern {
			return true
		}
		if matched, err := filepath.Match(pattern, part); err == nil && matched {
			return true
		}
	}

	// Full path match
	if matched, err := filepath.Match(pattern, path); err == nil && matched {
		return true
	}

	return false
}

// checkPyprojectConfig checks [tool.sst] include/exclude configuration.
// Returns (shouldExclude, found) where found indicates explicit configuration was matched.
func (cf *ContentFilter) checkPyprojectConfig(path string) (bool, bool) {
	if cf.projectRoot == "" {
		return false, false
	}

	if cf.pyprojectCache == nil {
		cf.loadPyprojectConfig()
	}
	if cf.pyprojectCache == nil {
		return false, false
	}

	for _, pattern := range cf.pyprojectCache.Tool.SST.Include {
		if cf.matchesPattern(path, pattern) {
			return false, true
		}
	}
	for _, pattern := range cf.pyprojectCache.Tool.SST.Exclude {
		if cf.matchesPattern(path, pattern) {
			return true, true
		}
	}
	return false, false
}

// loadPyprojectConfig loads and parses the pyproject.toml file
func (cf *ContentFilter) loadPyprojectConfig() {
	pyprojectPath := filepath.Join(cf.projectRoot, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); os.IsNotExist(err) {
		return
	}

	var config PyprojectConfig
	if _, err := toml.DecodeFile(pyprojectPath, &config); err != nil {
		slog.Warn("failed to parse pyproject.toml", "path", pyprojectPath, "error", err)
		return
	}

	if len(config.Tool.SST.Include) > 0 || len(config.Tool.SST.Exclude) > 0 {
		cf.pyprojectCache = &config
	}
}
