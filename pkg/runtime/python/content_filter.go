package python

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ContentFilter filters out unnecessary files and directories from deployment artifacts
type ContentFilter struct {
	excludePatterns []string
	logger          *slog.Logger
}

// NewContentFilter creates a new content filter with default exclude patterns
func NewContentFilter() *ContentFilter {
	return &ContentFilter{
		excludePatterns: getDefaultExcludePatterns(),
		logger:          slog.Default(),
	}
}

// NewContentFilterWithPatterns creates a content filter with custom exclude patterns
func NewContentFilterWithPatterns(patterns []string) *ContentFilter {
	return &ContentFilter{
		excludePatterns: patterns,
		logger:          slog.Default(),
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

		// Test directories and files
		"tests",
		"tests/**",
		"test",
		"test/**",
		"*_test.py",
		"test_*.py",

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

// ShouldExclude checks if a file or directory should be excluded based on patterns
func (cf *ContentFilter) ShouldExclude(path string) bool {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)
	baseName := filepath.Base(path)

	for _, pattern := range cf.excludePatterns {
		// Direct name match
		if baseName == pattern {
			cf.logger.Debug("excluding file by basename match",
				"path", path,
				"pattern", pattern)
			return true
		}

		// Pattern match with wildcards
		if matched, err := filepath.Match(pattern, baseName); err == nil && matched {
			cf.logger.Debug("excluding file by pattern match",
				"path", path,
				"pattern", pattern)
			return true
		}

		// Full path pattern match
		if matched, err := filepath.Match(pattern, normalizedPath); err == nil && matched {
			cf.logger.Debug("excluding file by full path pattern match",
				"path", path,
				"pattern", pattern)
			return true
		}

		// Directory prefix match (for patterns ending with /**)
		if strings.HasSuffix(pattern, "/**") {
			dirPattern := strings.TrimSuffix(pattern, "/**")
			if strings.HasPrefix(normalizedPath, dirPattern+"/") || normalizedPath == dirPattern {
				cf.logger.Debug("excluding file by directory pattern match",
					"path", path,
					"pattern", pattern)
				return true
			}
		}

		// Check if any parent directory matches the pattern
		dir := filepath.Dir(normalizedPath)
		for dir != "." && dir != "/" {
			if filepath.Base(dir) == pattern {
				cf.logger.Debug("excluding file by parent directory match",
					"path", path,
					"parentDir", dir,
					"pattern", pattern)
				return true
			}
			dir = filepath.Dir(dir)
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

// AddExcludePattern adds a new exclude pattern
func (cf *ContentFilter) AddExcludePattern(pattern string) {
	cf.excludePatterns = append(cf.excludePatterns, pattern)
}

// ValidateFilteredContent validates that the filtered content is reasonable for deployment
func (cf *ContentFilter) ValidateFilteredContent(targetDir string, maxSizeBytes int64) error {
	cf.logger.Info("validating filtered content", "targetDir", targetDir)

	var totalSize int64
	var fileCount int
	var pythonFiles []string

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++

			if strings.HasSuffix(info.Name(), ".py") {
				relPath, _ := filepath.Rel(targetDir, path)
				pythonFiles = append(pythonFiles, relPath)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk filtered directory: %w", err)
	}

	cf.logger.Info("filtered content validation results",
		"targetDir", targetDir,
		"totalSize", totalSize,
		"fileCount", fileCount,
		"pythonFiles", len(pythonFiles),
		"maxSizeBytes", maxSizeBytes)

	// Check size limits
	if maxSizeBytes > 0 && totalSize > maxSizeBytes {
		return &BuildValidationError{
			Stage:      "filter",
			Command:    "content filtering",
			Files:      []string{targetDir},
			Expected:   []string{fmt.Sprintf("size <= %d bytes", maxSizeBytes)},
			Actual:     []string{fmt.Sprintf("size = %d bytes", totalSize)},
			Suggestion: fmt.Sprintf("Filtered content is %d bytes, which exceeds the maximum size limit of %d bytes. Consider adding more exclude patterns or reducing the content size.", totalSize, maxSizeBytes),
		}
	}

	// Check that we have some Python files
	if len(pythonFiles) == 0 {
		cf.logger.Warn("no Python files found in filtered content",
			"targetDir", targetDir,
			"fileCount", fileCount)
	}

	return nil
}
