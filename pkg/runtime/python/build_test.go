package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

func TestRewriteContainerRequirements(t *testing.T) {
	t.Run("materializes parent workspace members inside the artifact", func(t *testing.T) {
		root := t.TempDir()
		workspaceRoot := filepath.Join(root, "projects", "app")
		libDir := filepath.Join(root, "lib")
		artifactRoot := filepath.Join(root, "artifact")
		for _, dir := range []string{workspaceRoot, libDir, artifactRoot} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}

		calls := 0
		rewritten, err := rewriteContainerRequirements(
			context.Background(),
			"../../lib[extra] ; python_version >= '3.12'\nrequests==2.32.0\n../../lib",
			workspaceRoot,
			artifactRoot,
			func(_ context.Context, artifactRoot string, packageDir string) (string, error) {
				calls++
				if packageDir != libDir {
					t.Fatalf("package dir = %s, want %s", packageDir, libDir)
				}
				archive := filepath.Join(artifactRoot, ".sst", "packages", "lib", "lib-0.1.0.tar.gz")
				if err := os.MkdirAll(filepath.Dir(archive), 0755); err != nil {
					return "", err
				}
				return archive, os.WriteFile(archive, nil, 0644)
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		if calls != 1 {
			t.Fatalf("sdist builder called %d times, want 1", calls)
		}
		if strings.Contains(rewritten, "..") {
			t.Fatalf("rewritten requirements contain a parent path:\n%s", rewritten)
		}
		if !strings.Contains(rewritten, "./.sst/packages/lib/lib-0.1.0.tar.gz[extra] ; python_version >= '3.12'") {
			t.Fatalf("local requirement was not rewritten with its suffix:\n%s", rewritten)
		}
		if !strings.Contains(rewritten, "requests==2.32.0") {
			t.Fatalf("registry requirement was changed:\n%s", rewritten)
		}
	})

	t.Run("supports descendant path dependencies", func(t *testing.T) {
		workspaceRoot := t.TempDir()
		packageDir := filepath.Join(workspaceRoot, "packages", "common")
		artifactRoot := filepath.Join(workspaceRoot, "artifact")
		for _, dir := range []string{packageDir, artifactRoot} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}

		rewritten, err := rewriteContainerRequirements(
			context.Background(),
			"./packages/common",
			workspaceRoot,
			artifactRoot,
			func(_ context.Context, artifactRoot string, packageDir string) (string, error) {
				archive := filepath.Join(artifactRoot, ".sst", "packages", "common", "common-0.1.0.tar.gz")
				return archive, nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		if rewritten != "./.sst/packages/common/common-0.1.0.tar.gz" {
			t.Fatalf("rewritten requirement = %q", rewritten)
		}
	})

	t.Run("rejects missing local packages", func(t *testing.T) {
		_, err := rewriteContainerRequirements(
			context.Background(),
			"../missing",
			t.TempDir(),
			t.TempDir(),
			func(context.Context, string, string) (string, error) {
				t.Fatal("sdist builder should not be called")
				return "", nil
			},
		)
		if err == nil {
			t.Fatal("expected missing package error")
		}
	})

	t.Run("rejects archives outside the artifact", func(t *testing.T) {
		workspaceRoot := t.TempDir()
		packageDir := filepath.Join(workspaceRoot, "package")
		if err := os.MkdirAll(packageDir, 0755); err != nil {
			t.Fatal(err)
		}

		_, err := rewriteContainerRequirements(
			context.Background(),
			"./package",
			workspaceRoot,
			t.TempDir(),
			func(context.Context, string, string) (string, error) {
				return filepath.Join(workspaceRoot, "outside.tar.gz"), nil
			},
		)
		if err == nil {
			t.Fatal("expected artifact containment error")
		}
	})
}

func TestSplitLocalRequirement(t *testing.T) {
	tests := []struct {
		line       string
		wantPath   string
		wantSuffix string
		ok         bool
	}{
		{"./package", "./package", "", true},
		{"../../lib[extra] ; python_version >= '3.12'", "../../lib", "[extra] ; python_version >= '3.12'", true},
		{"requests==2.32.0", "", "", false},
		{"-e ./package", "", "", false},
	}

	for _, tt := range tests {
		path, suffix, ok := splitLocalRequirement(tt.line)
		if path != tt.wantPath || suffix != tt.wantSuffix || ok != tt.ok {
			t.Errorf("splitLocalRequirement(%q) = (%q, %q, %v), want (%q, %q, %v)", tt.line, path, suffix, ok, tt.wantPath, tt.wantSuffix, tt.ok)
		}
	}
}

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

	if err := cleanupInstalledDependencies(tempDir); err != nil {
		t.Fatalf("cleanupInstalledDependencies failed: %v", err)
	}

	// All packages should be kept (no special stripping)
	for _, pkg := range []string{"boto3", "botocore", "requests"} {
		if _, err := os.Stat(filepath.Join(tempDir, pkg, "__init__.py")); err != nil {
			t.Errorf("package %s should have been kept", pkg)
		}
	}

	// __pycache__ and .pyc files should be preserved (needed for cold start performance)
	if _, err := os.Stat(filepath.Join(tempDir, "requests", "__pycache__")); err != nil {
		t.Error("__pycache__ should have been preserved for bytecode caching")
	}

	if _, err := os.Stat(filepath.Join(tempDir, "some_module.pyc")); err != nil {
		t.Error(".pyc files should have been preserved for bytecode caching")
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

		err = copySourceFilesSimple(input, projectInfo)
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

		err = filterEditableInstalls(inputPath, outputPath)
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

func TestIsIgnored(t *testing.T) {
	tests := []struct {
		name      string
		testPaths map[string]bool // path -> should be excluded
	}{
		{
			name: "default exclude patterns",
			testPaths: map[string]bool{
				"functions/handler.py":           false,
				"core/models.py":                 false,
				".sst/cache/build.json":          true,
				".git/config":                    true,
				"functions/__pycache__/test.pyc": false, // preserved for bytecode caching
				".pytest_cache/v/cache":          true,
				"node_modules/package/index.js":  true,
				".DS_Store":                      true,
				"test.pyc":                       false, // preserved for bytecode caching
				"module.pyo":                     true,
				".coverage":                      true,
				"htmlcov/index.html":             true,
				".venv/bin/python":               true,
				"venv/lib/python3.9":             true,
				".env":                           true,
				"requirements.txt":               false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for testPath, shouldExclude := range tt.testPaths {
				result := isIgnored(testPath)
				if result != shouldExclude {
					t.Errorf("Path %s: expected exclude=%v, got exclude=%v", testPath, shouldExclude, result)
				}
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
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
			for path, shouldMatch := range tt.paths {
				result := matchesPattern(path, tt.pattern)
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
			name:     "empty directory is not buildable",
			files:    map[string]string{},
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
