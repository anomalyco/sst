package python

import (
	"os"
	"path/filepath"
	"testing"
)

// newContentFilterWithPatterns creates a ContentFilter with the given exclude patterns for testing
func newContentFilterWithPatterns(patterns []string) *ContentFilter {
	filter := NewContentFilter()
	filter.excludePatterns = patterns
	return filter
}

func TestContentFilter_ShouldExclude(t *testing.T) {
	tests := []struct {
		name            string
		excludePatterns []string
		testPaths       map[string]bool // path -> should be excluded
	}{
		{
			name: "default exclude patterns",
			excludePatterns: []string{
				".sst",
				".git",
				"__pycache__",
				".pytest_cache",
				"node_modules",
				".DS_Store",
				"*.pyc",
				"*.pyo",
				"*.pyd",
				".coverage",
				"htmlcov",
				".tox",
				".venv",
				"venv",
				".env",
			},
			testPaths: map[string]bool{
				"functions/handler.py":           false,
				"core/models.py":                 false,
				".sst/cache/build.json":          true,
				".git/config":                    true,
				"functions/__pycache__/test.pyc": true,
				".pytest_cache/v/cache":          true,
				"node_modules/package/index.js":  true,
				".DS_Store":                      true,
				"test.pyc":                       true,
				"module.pyo":                     true,
				"extension.pyd":                  true,
				".coverage":                      true,
				"htmlcov/index.html":             true,
				".tox/py39/lib":                  true,
				".venv/bin/python":               true,
				"venv/lib/python3.9":             true,
				".env":                           true,
				"requirements.txt":               false,
				"setup.py":                       false,
			},
		},
		{
			name: "custom exclude patterns",
			excludePatterns: []string{
				"*.log",
				"temp*",
				"build/",
				"dist/",
			},
			testPaths: map[string]bool{
				"functions/handler.py":   false,
				"debug.log":              true,
				"application.log":        true,
				"temp_file.txt":          true,
				"temporary_data.json":    true,
				"build/output.tar.gz":    true,
				"dist/package-1.0.0.whl": true,
				"src/main.py":            false,
				"logs/info.txt":          false, // doesn't match *.log pattern
			},
		},
		{
			name: "nested path patterns",
			excludePatterns: []string{
				"**/test_*",
				"**/.*",
				"docs/**",
			},
			testPaths: map[string]bool{
				"functions/handler.py":      false,
				"functions/test_handler.py": true,
				"core/models/test_user.py":  true,
				"src/.hidden_file":          true,
				"functions/.secret":         true,
				"docs/README.md":            true,
				"docs/api/endpoints.md":     true,
				"src/docs_helper.py":        false, // doesn't match docs/** pattern
			},
		},
		{
			name:            "empty exclude patterns",
			excludePatterns: []string{},
			testPaths: map[string]bool{
				"functions/handler.py":           false,
				".sst/cache/build.json":          false,
				".git/config":                    false,
				"functions/__pycache__/test.pyc": false,
				"node_modules/package/index.js":  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := newContentFilterWithPatterns(tt.excludePatterns)

			for testPath, shouldExclude := range tt.testPaths {
				result := filter.ShouldExclude(testPath)
				if result != shouldExclude {
					t.Errorf("Path %s: expected exclude=%v, got exclude=%v", testPath, shouldExclude, result)
				}
			}
		})
	}
}

func TestContentFilter_FilterDirectory(t *testing.T) {
	tests := []struct {
		name            string
		excludePatterns []string
		setupFunc       func(string) error
		expectedFiles   []string // Files that should exist in target after filtering
		excludedFiles   []string // Files that should be excluded
	}{
		{
			name: "filter common build artifacts",
			excludePatterns: []string{
				".sst",
				"__pycache__",
				"*.pyc",
				".git",
			},
			setupFunc: func(sourceDir string) error {
				files := map[string]string{
					"functions/handler.py":           "def handler(): pass",
					"functions/__init__.py":          "# Functions package",
					"functions/__pycache__/test.pyc": "compiled python",
					"core/models.py":                 "class Model: pass",
					"core/__init__.py":               "# Core package",
					".sst/cache/build.json":          "{}",
					".sst/state/terraform.tfstate":   "{}",
					".git/config":                    "[core]",
					"requirements.txt":               "requests==2.28.1",
					"setup.py":                       "from setuptools import setup",
					"compiled.pyc":                   "compiled",
				}

				for file, content := range files {
					fullPath := filepath.Join(sourceDir, file)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return err
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedFiles: []string{
				"functions/handler.py",
				"functions/__init__.py",
				"core/models.py",
				"core/__init__.py",
				"requirements.txt",
				"setup.py",
			},
			excludedFiles: []string{
				"functions/__pycache__/test.pyc",
				".sst/cache/build.json",
				".sst/state/terraform.tfstate",
				".git/config",
				"compiled.pyc",
			},
		},
		{
			name: "preserve nested package structure",
			excludePatterns: []string{
				"*.log",
				"temp*",
				"test_*",
			},
			setupFunc: func(sourceDir string) error {
				files := map[string]string{
					"functions/__init__.py":        "# Functions",
					"functions/handler.py":         "def handler(): pass",
					"functions/utils/__init__.py":  "# Utils",
					"functions/utils/helpers.py":   "def help(): pass",
					"functions/models/__init__.py": "# Models",
					"functions/models/user.py":     "class User: pass",
					"functions/test_handler.py":    "def test(): pass",
					"core/__init__.py":             "# Core",
					"core/database.py":             "def connect(): pass",
					"core/config.py":               "CONFIG = {}",
					"debug.log":                    "debug info",
					"temp_data.json":               "{}",
					"temporary_file.txt":           "temp content",
				}

				for file, content := range files {
					fullPath := filepath.Join(sourceDir, file)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return err
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedFiles: []string{
				"functions/__init__.py",
				"functions/handler.py",
				"functions/utils/__init__.py",
				"functions/utils/helpers.py",
				"functions/models/__init__.py",
				"functions/models/user.py",
				"core/__init__.py",
				"core/database.py",
				"core/config.py",
			},
			excludedFiles: []string{
				"functions/test_handler.py",
				"debug.log",
				"temp_data.json",
				"temporary_file.txt",
			},
		},
		{
			name: "handle empty directories",
			excludePatterns: []string{
				"*.tmp",
			},
			setupFunc: func(sourceDir string) error {
				// Create directories with some empty ones
				dirs := []string{
					"functions",
					"core",
					"empty_dir",
					"temp_dir",
				}

				for _, dir := range dirs {
					if err := os.MkdirAll(filepath.Join(sourceDir, dir), 0755); err != nil {
						return err
					}
				}

				// Add files to some directories
				files := map[string]string{
					"functions/handler.py": "def handler(): pass",
					"core/models.py":       "class Model: pass",
					"temp_dir/cache.tmp":   "temp cache",
				}

				for file, content := range files {
					fullPath := filepath.Join(sourceDir, file)
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedFiles: []string{
				"functions/handler.py",
				"core/models.py",
			},
			excludedFiles: []string{
				"temp_dir/cache.tmp",
			},
		},
		{
			name:            "no exclusions",
			excludePatterns: []string{},
			setupFunc: func(sourceDir string) error {
				files := map[string]string{
					"functions/handler.py":           "def handler(): pass",
					"functions/__pycache__/test.pyc": "compiled",
					".sst/cache/build.json":          "{}",
					".git/config":                    "[core]",
					"debug.log":                      "debug",
				}

				for file, content := range files {
					fullPath := filepath.Join(sourceDir, file)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return err
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedFiles: []string{
				"functions/handler.py",
				"functions/__pycache__/test.pyc",
				".sst/cache/build.json",
				".git/config",
				"debug.log",
			},
			excludedFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directories
			sourceDir, err := os.MkdirTemp("", "filter_source")
			if err != nil {
				t.Fatalf("Failed to create source directory: %v", err)
			}
			defer os.RemoveAll(sourceDir)

			targetDir, err := os.MkdirTemp("", "filter_target")
			if err != nil {
				t.Fatalf("Failed to create target directory: %v", err)
			}
			defer os.RemoveAll(targetDir)

			// Setup test scenario
			if err := tt.setupFunc(sourceDir); err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create filter and apply
			filter := newContentFilterWithPatterns(tt.excludePatterns)
			err = filter.FilterDirectory(sourceDir, targetDir)

			if err != nil {
				t.Fatalf("FilterDirectory failed: %v", err)
			}

			// Check that expected files exist in target
			for _, expectedFile := range tt.expectedFiles {
				targetPath := filepath.Join(targetDir, expectedFile)
				if _, err := os.Stat(targetPath); os.IsNotExist(err) {
					t.Errorf("Expected file not found in target: %s", expectedFile)
				}
			}

			// Check that excluded files don't exist in target
			for _, excludedFile := range tt.excludedFiles {
				targetPath := filepath.Join(targetDir, excludedFile)
				if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
					t.Errorf("Excluded file found in target: %s", excludedFile)
				}
			}
		})
	}
}

func TestContentFilter_GetExcludePatterns(t *testing.T) {
	tests := []struct {
		name            string
		excludePatterns []string
	}{
		{
			name:            "default patterns",
			excludePatterns: []string{".sst", ".git", "__pycache__"},
		},
		{
			name:            "custom patterns",
			excludePatterns: []string{"*.log", "temp*", "build/"},
		},
		{
			name:            "empty patterns",
			excludePatterns: []string{},
		},
		{
			name:            "single pattern",
			excludePatterns: []string{"*.pyc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := newContentFilterWithPatterns(tt.excludePatterns)
			patterns := filter.GetExcludePatterns()

			if len(patterns) != len(tt.excludePatterns) {
				t.Errorf("Expected %d patterns, got %d", len(tt.excludePatterns), len(patterns))
			}

			for i, pattern := range patterns {
				if pattern != tt.excludePatterns[i] {
					t.Errorf("Pattern %d: expected %s, got %s", i, tt.excludePatterns[i], pattern)
				}
			}
		})
	}
}

func TestContentFilter_PatternMatching(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		paths   map[string]bool // path -> should match
	}{
		{
			name:    "exact match",
			pattern: ".sst",
			paths: map[string]bool{
				".sst":                    true,
				".sst/cache/build.json":   true,
				"functions/.sst":          true,
				"sst":                     false,
				"functions/sst_config.py": false,
			},
		},
		{
			name:    "wildcard match",
			pattern: "*.pyc",
			paths: map[string]bool{
				"test.pyc":                       true,
				"functions/__pycache__/test.pyc": true,
				"module.py":                      false,
				"test.pyo":                       false,
			},
		},
		{
			name:    "directory pattern",
			pattern: "__pycache__",
			paths: map[string]bool{
				"__pycache__":                    true,
				"__pycache__/test.pyc":           true,
				"functions/__pycache__":          true,
				"functions/__pycache__/test.pyc": true,
				"pycache":                        false,
				"my__pycache__":                  false,
			},
		},
		{
			name:    "prefix pattern",
			pattern: "temp*",
			paths: map[string]bool{
				"temp":           true,
				"temp.txt":       true,
				"temporary":      true,
				"temp_file.json": true,
				"my_temp.txt":    false,
				"not_temp":       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := newContentFilterWithPatterns([]string{tt.pattern})

			for path, shouldMatch := range tt.paths {
				result := filter.ShouldExclude(path)
				if result != shouldMatch {
					t.Errorf("Pattern %s, Path %s: expected match=%v, got match=%v", tt.pattern, path, shouldMatch, result)
				}
			}
		})
	}
}

func TestContentFilter_Integration(t *testing.T) {
	// Integration test that simulates filtering a realistic Python project
	tempDir, err := os.MkdirTemp("", "content_filter_integration")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	// Create a realistic Python project structure
	projectFiles := map[string]string{
		// Python source files (should be included)
		"functions/__init__.py":       "# Functions package",
		"functions/handler.py":        "def lambda_handler(event, context): pass",
		"functions/utils/__init__.py": "# Utils package",
		"functions/utils/helpers.py":  "def helper(): pass",
		"core/__init__.py":            "# Core package",
		"core/models.py":              "class User: pass",
		"core/database.py":            "def connect(): pass",
		"requirements.txt":            "requests==2.28.1\npsycopg2==2.9.3",
		"setup.py":                    "from setuptools import setup",

		// Build artifacts (should be excluded)
		".sst/cache/build.json":             "{}",
		".sst/state/terraform.tfstate":      "{}",
		"functions/__pycache__/handler.pyc": "compiled python",
		"core/__pycache__/models.pyc":       "compiled python",
		".pytest_cache/v/cache/nodeids":     "cache data",
		"htmlcov/index.html":                "<html>coverage report</html>",
		".coverage":                         "coverage data",

		// Version control (should be excluded)
		".git/config": "[core]",
		".git/HEAD":   "ref: refs/heads/main",
		".gitignore":  "*.pyc\n__pycache__/",

		// Development files (should be excluded)
		".venv/bin/python":                 "#!/usr/bin/env python",
		"venv/lib/python3.9/site-packages": "packages",
		".env":                             "SECRET_KEY=secret",
		".DS_Store":                        "mac metadata",

		// Log files (should be excluded)
		"debug.log":       "debug information",
		"application.log": "app logs",

		// Temporary files (should be excluded)
		"temp_data.json":     "{}",
		"temporary_file.txt": "temp content",

		// Test files (should be excluded)
		"test_handler.py":         "def test_handler(): pass",
		"functions/test_utils.py": "def test_utils(): pass",
	}

	// Create source files
	for file, content := range projectFiles {
		fullPath := filepath.Join(sourceDir, file)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", file, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	// Create filter with comprehensive exclude patterns
	excludePatterns := []string{
		".sst",
		".git",
		"__pycache__",
		"*.pyc",
		"*.pyo",
		"*.pyd",
		".pytest_cache",
		"htmlcov",
		".coverage",
		".tox",
		".venv",
		"venv",
		".env",
		".DS_Store",
		"*.log",
		"temp*",
		"test_*",
	}

	filter := newContentFilterWithPatterns(excludePatterns)

	// Apply filter
	if err := filter.FilterDirectory(sourceDir, targetDir); err != nil {
		t.Fatalf("FilterDirectory failed: %v", err)
	}

	// Define expected results
	expectedIncluded := []string{
		"functions/__init__.py",
		"functions/handler.py",
		"functions/utils/__init__.py",
		"functions/utils/helpers.py",
		"core/__init__.py",
		"core/models.py",
		"core/database.py",
		"requirements.txt",
		"setup.py",
		".gitignore", // Should be included (not in exclude patterns)
	}

	expectedExcluded := []string{
		".sst/cache/build.json",
		".sst/state/terraform.tfstate",
		"functions/__pycache__/handler.pyc",
		"core/__pycache__/models.pyc",
		".pytest_cache/v/cache/nodeids",
		"htmlcov/index.html",
		".coverage",
		".git/config",
		".git/HEAD",
		".venv/bin/python",
		"venv/lib/python3.9/site-packages",
		".env",
		".DS_Store",
		"debug.log",
		"application.log",
		"temp_data.json",
		"temporary_file.txt",
		"test_handler.py",
		"functions/test_utils.py",
	}

	// Verify included files
	for _, file := range expectedIncluded {
		targetPath := filepath.Join(targetDir, file)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			t.Errorf("Expected file not found in target: %s", file)
		}
	}

	// Verify excluded files
	for _, file := range expectedExcluded {
		targetPath := filepath.Join(targetDir, file)
		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			t.Errorf("Excluded file found in target: %s", file)
		}
	}

	// Verify directory structure is preserved
	expectedDirs := []string{
		"functions",
		"functions/utils",
		"core",
	}

	for _, dir := range expectedDirs {
		targetPath := filepath.Join(targetDir, dir)
		if info, err := os.Stat(targetPath); err != nil || !info.IsDir() {
			t.Errorf("Expected directory not found or not a directory: %s", dir)
		}
	}
}
