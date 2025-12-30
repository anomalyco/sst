package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileHelpers provides common file operations
type FileHelpers struct{}

// NewFileHelpers creates a new file helpers instance
func NewFileHelpers() *FileHelpers {
	return &FileHelpers{}
}

// CreateFileWithContent creates a file with the given content and default permissions
func (fh *FileHelpers) CreateFileWithContent(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), DefaultDirMode); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}
	return os.WriteFile(path, []byte(content), DefaultFileMode)
}

// FileExists checks if a file exists
func (fh *FileHelpers) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsPythonFile checks if a file is a Python source file
func (fh *FileHelpers) IsPythonFile(filename string) bool {
	return strings.HasSuffix(filename, PythonFileExt)
}

// IsDependencyFile checks if a file is a dependency configuration file
func (fh *FileHelpers) IsDependencyFile(filename string) bool {
	for _, depFile := range DependencyFiles {
		if filename == depFile {
			return true
		}
	}
	return false
}

// ShouldExcludeFile checks if a file should be excluded from builds
func (fh *FileHelpers) ShouldExcludeFile(filename string) bool {
	for _, pattern := range ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
		// Also check if it's a directory that should be excluded
		if strings.Contains(filename, "/"+strings.TrimSuffix(pattern, "/*")) {
			return true
		}
	}
	return false
}

// ShouldSkipInitPyDir checks if a directory should be skipped for __init__.py creation
func (fh *FileHelpers) ShouldSkipInitPyDir(dirName string) bool {
	for _, skipDir := range SkipInitPyDirs {
		if dirName == skipDir {
			return true
		}
	}
	// Skip .dist-info and .egg-info directories
	return strings.HasSuffix(dirName, ".dist-info") || strings.HasSuffix(dirName, ".egg-info")
}

// FormatSize formats a byte size as MB
func (fh *FileHelpers) FormatSize(bytes int64) string {
	return fmt.Sprintf("%.1fMB", float64(bytes)/BytesPerMB)
}

// PathHelpers provides common path operations
type PathHelpers struct{}

// NewPathHelpers creates a new path helpers instance
func NewPathHelpers() *PathHelpers {
	return &PathHelpers{}
}

// GetCacheDir returns the standard cache directory path (legacy, use GetCacheDirForMode)
func (ph *PathHelpers) GetCacheDir(workingDir string) string {
	return filepath.Join(workingDir, SstCacheDir)
}

// GetCacheDirForMode returns the cache directory for dev or deploy mode
func (ph *PathHelpers) GetCacheDirForMode(workingDir string, isDev bool) string {
	if isDev {
		return filepath.Join(workingDir, SstCacheDevDir)
	}
	return filepath.Join(workingDir, SstCacheDeployDir)
}

// GetSstDir returns the SST directory path
func (ph *PathHelpers) GetSstDir(workingDir string) string {
	return filepath.Join(workingDir, SstDir)
}

// IsWithinProject checks if a path is within the project root
func (ph *PathHelpers) IsWithinProject(filePath, projectRoot string) bool {
	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absFile, absRoot)
}

// ErrorHelpers provides common error message patterns
type ErrorHelpers struct{}

// NewErrorHelpers creates a new error helpers instance
func NewErrorHelpers() *ErrorHelpers {
	return &ErrorHelpers{}
}

// WrapFileError wraps a file operation error with context
func (eh *ErrorHelpers) WrapFileError(operation, filePath string, err error) error {
	return fmt.Errorf("failed to %s file %s: %w", operation, filePath, err)
}

// WrapDirError wraps a directory operation error with context
func (eh *ErrorHelpers) WrapDirError(operation, dirPath string, err error) error {
	return fmt.Errorf("failed to %s directory %s: %w", operation, dirPath, err)
}

// WrapBuildError wraps a build operation error with context
func (eh *ErrorHelpers) WrapBuildError(operation, target string, err error) error {
	return fmt.Errorf("failed to %s %s: %w", operation, target, err)
}
