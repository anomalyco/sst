package python

import (
	"path/filepath"
)

// PathHelpers provides common path operations
type PathHelpers struct{}

// NewPathHelpers creates a new path helpers instance
func NewPathHelpers() *PathHelpers {
	return &PathHelpers{}
}

// GetCacheDirForMode returns the cache directory for dev or deploy mode
func (ph *PathHelpers) GetCacheDirForMode(workingDir string, isDev bool) string {
	if isDev {
		return filepath.Join(workingDir, SstCacheDevDir)
	}
	return filepath.Join(workingDir, SstCacheDeployDir)
}
