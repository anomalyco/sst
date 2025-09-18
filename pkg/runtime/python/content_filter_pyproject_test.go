package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContentFilter_PyprojectConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "content_filter_pyproject_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a pyproject.toml with SST configuration
	pyprojectContent := `
[project]
name = "test-project"

[tool.sst]
include = ["important.txt", "config/*.json"]
exclude = ["temp/*", "*.log"]
`
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create content filter for the project
	filter := NewContentFilterForProject(tempDir)

	tests := []struct {
		path     string
		expected bool // true = should exclude
		reason   string
	}{
		// Explicit includes (should not be excluded)
		{"important.txt", false, "explicitly included"},
		{"config/app.json", false, "matches include pattern"},

		// Explicit excludes (should be excluded)
		{"temp/cache.dat", true, "matches exclude pattern"},
		{"debug.log", true, "matches exclude pattern"},

		// Standard exclusions (should be excluded)
		{"__pycache__/test.pyc", true, "standard Python exclusion"},
		{".git/config", true, "standard git exclusion"},

		// Normal files (should not be excluded)
		{"main.py", false, "normal Python file"},
		{"README.md", true, "standard exclusion for README"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := filter.ShouldExclude(tt.path)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, expected %v (%s)",
					tt.path, result, tt.expected, tt.reason)
			}
		})
	}
}

func TestContentFilter_NoPyprojectConfig(t *testing.T) {
	// Create a temporary directory without pyproject.toml
	tempDir, err := os.MkdirTemp("", "content_filter_no_pyproject_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create content filter for the project
	filter := NewContentFilterForProject(tempDir)

	tests := []struct {
		path     string
		expected bool // true = should exclude
		reason   string
	}{
		// Standard exclusions should still work
		{"__pycache__/test.pyc", true, "standard Python exclusion"},
		{".git/config", true, "standard git exclusion"},
		{"README.md", true, "standard exclusion for README"},

		// Normal files should not be excluded
		{"main.py", false, "normal Python file"},
		{"requirements.txt", false, "requirements file should be included"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := filter.ShouldExclude(tt.path)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, expected %v (%s)",
					tt.path, result, tt.expected, tt.reason)
			}
		})
	}
}
