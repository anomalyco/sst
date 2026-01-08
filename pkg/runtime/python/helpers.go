package python

import (
	"path/filepath"
	"strings"
)

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
