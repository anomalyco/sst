package fs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/internal/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindDown_WithExcludePatterns(t *testing.T) {
	tests := []struct {
		name             string
		setupDirs        []string
		setupFiles       []string
		searchFile       string
		excludePatterns  []string
		expectedCount    int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:             "basic - no excludes",
			setupDirs:        []string{"src", "lib"},
			setupFiles:       []string{"src/package.json", "lib/package.json"},
			searchFile:       "package.json",
			excludePatterns:  []string{},
			expectedCount:    2,
			shouldContain:    []string{"src/package.json", "lib/package.json"},
			shouldNotContain: []string{},
		},
		{
			name:             "exclude node_modules",
			setupDirs:        []string{"src", "node_modules", "node_modules/lib"},
			setupFiles:       []string{"src/package.json", "node_modules/package.json", "node_modules/lib/package.json"},
			searchFile:       "package.json",
			excludePatterns:  []string{"node_modules"},
			expectedCount:    1,
			shouldContain:    []string{"src/package.json"},
			shouldNotContain: []string{"node_modules"},
		},
		{
			name:             "exclude external directory",
			setupDirs:        []string{"src", "external", "external/submodule"},
			setupFiles:       []string{"src/package.json", "external/package.json", "external/submodule/package.json"},
			searchFile:       "package.json",
			excludePatterns:  []string{"external"},
			expectedCount:    1,
			shouldContain:    []string{"src/package.json"},
			shouldNotContain: []string{"external"},
		},
		{
			name:             "exclude multiple patterns",
			setupDirs:        []string{"src", "external", "vendor", "lib"},
			setupFiles:       []string{"src/package.json", "external/package.json", "vendor/package.json", "lib/package.json"},
			searchFile:       "package.json",
			excludePatterns:  []string{"external", "vendor"},
			expectedCount:    2,
			shouldContain:    []string{"src/package.json", "lib/package.json"},
			shouldNotContain: []string{"external", "vendor"},
		},
		{
			name:             "exclude with glob pattern",
			setupDirs:        []string{"src", "test-fixtures", "test-data", "lib"},
			setupFiles:       []string{"src/package.json", "test-fixtures/package.json", "test-data/package.json", "lib/package.json"},
			searchFile:       "package.json",
			excludePatterns:  []string{"test-*"},
			expectedCount:    2,
			shouldContain:    []string{"src/package.json", "lib/package.json"},
			shouldNotContain: []string{"test-fixtures", "test-data"},
		},
		{
			name:             "exclude nested directories",
			setupDirs:        []string{"src", "src/external", "src/external/deep", "lib"},
			setupFiles:       []string{"src/package.json", "src/external/package.json", "src/external/deep/package.json", "lib/package.json"},
			searchFile:       "package.json",
			excludePatterns:  []string{"external"},
			expectedCount:    2,
			shouldContain:    []string{"src/package.json", "lib/package.json"},
			shouldNotContain: []string{"external"},
		},
		{
			name:             "existing node_modules still excluded by default",
			setupDirs:        []string{"src", "node_modules"},
			setupFiles:       []string{"src/package.json", "node_modules/package.json"},
			searchFile:       "package.json",
			excludePatterns:  []string{}, // No custom excludes
			expectedCount:    1,
			shouldContain:    []string{"src/package.json"},
			shouldNotContain: []string{"node_modules"},
		},
		{
			name:             "existing dotfiles still excluded by default",
			setupDirs:        []string{"src", ".git", ".github"},
			setupFiles:       []string{"src/package.json", ".git/package.json", ".github/package.json"},
			searchFile:       "package.json",
			excludePatterns:  []string{}, // No custom excludes
			expectedCount:    1,
			shouldContain:    []string{"src/package.json"},
			shouldNotContain: []string{".git", ".github"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Setup directory structure
			for _, dir := range tt.setupDirs {
				err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
				require.NoError(t, err, "Failed to create directory: %s", dir)
			}

			// Create files
			for _, file := range tt.setupFiles {
				fullPath := filepath.Join(tmpDir, file)
				f, err := os.Create(fullPath)
				require.NoError(t, err, "Failed to create file: %s", file)
				f.Close()
			}

			// Execute - this will fail until we implement the feature
			results := fs.FindDownWithExcludes(tmpDir, tt.searchFile, tt.excludePatterns)

			// Assert count
			assert.Len(t, results, tt.expectedCount, "Expected %d results, got %d", tt.expectedCount, len(results))

			// Assert contains expected paths
			for _, expectedPath := range tt.shouldContain {
				assert.True(t, containsPath(results, expectedPath),
					"Results should contain path with: %s\nGot: %v", expectedPath, results)
			}

			// Assert does not contain excluded paths
			for _, excludedPath := range tt.shouldNotContain {
				assert.False(t, containsPath(results, excludedPath),
					"Results should NOT contain path with: %s\nGot: %v", excludedPath, results)
			}
		})
	}
}

// Helper to check if results contain a path ending with the given suffix
func containsPath(results []string, pathSuffix string) bool {
	for _, r := range results {
		// Normalize both paths and check if result ends with the suffix
		cleanResult := filepath.Clean(r)
		cleanSuffix := filepath.Clean(pathSuffix)

		// Check if it's a suffix match (e.g., "src/package.json" matches ".../src/package.json")
		if strings.HasSuffix(cleanResult, string(filepath.Separator)+cleanSuffix) ||
			strings.HasSuffix(cleanResult, cleanSuffix) {
			return true
		}

		// Also check if pathSuffix is just a directory name component (for shouldNotContain checks)
		if !strings.Contains(pathSuffix, string(filepath.Separator)) {
			parts := strings.Split(cleanResult, string(filepath.Separator))
			for _, part := range parts {
				if part == pathSuffix {
					return true
				}
			}
		}
	}
	return false
}
