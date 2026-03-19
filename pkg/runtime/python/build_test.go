package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

func TestDeployBuilder_CleanupInstalledDependencies(t *testing.T) {
	tempDir := t.TempDir()

	testFiles := map[string]string{
		"requests/__init__.py":                          "# requests",
		"requests/api.py":                               "# api",
		"boto3/__init__.py":                             "# boto3",
		"botocore/__init__.py":                          "# botocore",
		"requests/__pycache__/__init__.cpython-312.pyc": "compiled",
		"boto3/__pycache__/__init__.cpython-312.pyc":    "compiled",
		"some_module.pyc":                               "compiled",
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	builder := &deployBuilder{}
	if err := builder.cleanupInstalledDependencies(tempDir, nil); err != nil {
		t.Fatalf("cleanupInstalledDependencies failed: %v", err)
	}

	// Lambda runtime packages should be removed
	for _, pkg := range []string{"boto3", "botocore"} {
		if _, err := os.Stat(filepath.Join(tempDir, pkg)); err == nil {
			t.Errorf("Lambda runtime package %s should have been removed", pkg)
		}
	}

	// Normal packages should be kept
	if _, err := os.Stat(filepath.Join(tempDir, "requests", "__init__.py")); err != nil {
		t.Error("requests package should have been kept")
	}

	// __pycache__ should be removed
	if _, err := os.Stat(filepath.Join(tempDir, "requests", "__pycache__")); err == nil {
		t.Error("__pycache__ should have been removed")
	}

	// .pyc files should be removed
	if _, err := os.Stat(filepath.Join(tempDir, "some_module.pyc")); err == nil {
		t.Error(".pyc files should have been removed")
	}
}

func TestLegacyStructureRegressionFixes(t *testing.T) {
	// Test 1: Path duplication fix for legacy functions/src/functions structure
	t.Run("Legacy path duplication regression", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sst-legacy-path-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create legacy structure: functions/src/functions/user/get_user_session.py
		functionsDir := filepath.Join(tempDir, "functions")
		srcDir := filepath.Join(functionsDir, "src")
		innerFunctionsDir := filepath.Join(srcDir, "functions")
		userDir := filepath.Join(innerFunctionsDir, "user")

		if err := os.MkdirAll(userDir, 0755); err != nil {
			t.Fatalf("Failed to create directory structure: %v", err)
		}

		handlerFile := filepath.Join(userDir, "get_user_session.py")
		if err := os.WriteFile(handlerFile, []byte("def handler(event, context): pass"), 0644); err != nil {
			t.Fatalf("Failed to create handler file: %v", err)
		}

		projectInfo := &projectInfo{
			SourceRoot: functionsDir, // This used to cause path duplication
		}

		input := &runtime.BuildInput{
			CfgPath:    tempDir,
			FunctionID: "legacy-test",
			Handler:    "functions/src/functions/user/get_user_session.handler",
		}

		actualOutputDir := input.Out()
		if err := os.MkdirAll(actualOutputDir, 0755); err != nil {
			t.Fatalf("Failed to create output dir: %v", err)
		}

		ib := &deployBuilder{}
		err = ib.copySourceFilesSimple(context.Background(), input, projectInfo)
		if err != nil {
			t.Fatalf("copySourceFilesSimple failed: %v", err)
		}

		// Verify the file was copied correctly
		copiedFile := filepath.Join(actualOutputDir, "src", "functions", "user", "get_user_session.py")
		if _, err := os.Stat(copiedFile); err != nil {
			t.Errorf("Expected file not found: %s", copiedFile)
		}
	})

	// Test 2: filterEditableInstalls keeps non-editable requirements unchanged
	t.Run("filterEditableInstalls preserves standard requirements", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sst-requirements-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		localPkgDir := filepath.Join(tempDir, "local-package")
		if err := os.MkdirAll(localPkgDir, 0755); err != nil {
			t.Fatalf("Failed to create local package dir: %v", err)
		}

		requirementsContent := `requests==2.31.0
boto3>=1.34.0`

		inputPath := filepath.Join(tempDir, "requirements.txt")
		outputPath := filepath.Join(tempDir, "requirements-filtered.txt")

		if err := os.WriteFile(inputPath, []byte(requirementsContent), 0644); err != nil {
			t.Fatalf("Failed to write requirements.txt: %v", err)
		}

		ib := &deployBuilder{}
		err = ib.filterEditableInstalls(inputPath, outputPath)
		if err != nil {
			t.Fatalf("filterEditableInstalls failed: %v", err)
		}

		filteredContent, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read filtered requirements: %v", err)
		}

		filteredStr := string(filteredContent)

		if !strings.Contains(filteredStr, "requests==2.31.0") {
			t.Errorf("Valid package requests was filtered out")
		}

		if !strings.Contains(filteredStr, "boto3") {
			t.Errorf("boto3 should be kept in requirements (cleanup handles removal)")
		}
	})
}

// --- Content filter tests (merged from content_filter_test.go) ---

// newContentFilterWithPatterns creates a contentFilter with the given exclude patterns for testing
func newContentFilterWithPatterns(patterns []string) *contentFilter {
	filter := newContentFilter()
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

func TestHasBuildConfig(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string // filename -> content
		expected bool
	}{
		{
			name: "setup.py makes it buildable",
			files: map[string]string{
				"setup.py": "from setuptools import setup\nsetup()",
			},
			expected: true,
		},
		{
			name: "pyproject.toml with build-system is buildable",
			files: map[string]string{
				"pyproject.toml": "[project]\nname = \"my-pkg\"\n\n[build-system]\nrequires = [\"hatchling\"]\n",
			},
			expected: true,
		},
		{
			name: "pyproject.toml without build-system is not buildable",
			files: map[string]string{
				"pyproject.toml": "[project]\nname = \"my-app\"\ndependencies = [\"requests\"]\n",
			},
			expected: false,
		},
		{
			name: "tool.sst.buildable = false overrides build-system",
			files: map[string]string{
				"pyproject.toml": "[project]\nname = \"my-app\"\n\n[build-system]\nrequires = [\"hatchling\"]\n\n[tool.sst]\nbuildable = false\n",
			},
			expected: false,
		},
		{
			name: "tool.sst.buildable = true marks as buildable without build-system",
			files: map[string]string{
				"pyproject.toml": "[project]\nname = \"my-app\"\n\n[tool.sst]\nbuildable = true\n",
			},
			expected: true,
		},
		{
			name:     "empty directory is not buildable",
			files:    map[string]string{},
			expected: false,
		},
		{
			name: "legacy magic string still works",
			files: map[string]string{
				"pyproject.toml": "[project]\nname = \"my-app\"\n# NOT a buildable package\n\n[build-system]\nrequires = [\"hatchling\"]\n",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for filename, content := range tt.files {
				path := filepath.Join(dir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write %s: %v", filename, err)
				}
			}

			got := hasBuildConfig(dir)
			if got != tt.expected {
				t.Errorf("hasBuildConfig() = %v, want %v", got, tt.expected)
			}
		})
	}
}
