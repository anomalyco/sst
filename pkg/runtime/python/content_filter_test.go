package python

import (
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
				"logs/info.txt":          false,
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
				"src/docs_helper.py":        false,
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
