package python

import (
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

func TestCopyWorkspacePackagesForContainer(t *testing.T) {
	// Builds a workspace where the member lives above the workspace root:
	//
	//   repo/
	//     lib/                  <- workspace member ("../../lib" from app)
	//     projects/app/         <- workspace root (has [tool.uv.workspace])
	//
	// and a build output dir seeded with the given requirements.txt.
	setup := func(t *testing.T, requirements string) (*runtime.BuildInput, *projectInfo, string) {
		tmp := t.TempDir()
		appDir := filepath.Join(tmp, "repo", "projects", "app")
		libDir := filepath.Join(tmp, "repo", "lib")

		if err := os.MkdirAll(filepath.Join(libDir, "src", "shared"), 0755); err != nil {
			t.Fatalf("failed to create lib dir: %v", err)
		}
		if err := os.MkdirAll(appDir, 0755); err != nil {
			t.Fatalf("failed to create app dir: %v", err)
		}
		files := map[string]string{
			filepath.Join(appDir, "pyproject.toml"):               "[project]\nname = \"app\"\n\n[tool.uv.workspace]\nmembers = [\"../../lib\"]\n",
			filepath.Join(libDir, "pyproject.toml"):               "[project]\nname = \"shared\"\n",
			filepath.Join(libDir, "src", "shared", "__init__.py"): "",
		}
		for path, content := range files {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write %s: %v", path, err)
			}
		}

		input := &runtime.BuildInput{
			CfgPath:     filepath.Join(tmp, "sst.config.ts"),
			FunctionID:  "testfn",
			IsContainer: true,
		}
		outDir := input.Out()
		if err := os.MkdirAll(outDir, 0755); err != nil {
			t.Fatalf("failed to create output dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "requirements.txt"), []byte(requirements), 0644); err != nil {
			t.Fatalf("failed to write requirements.txt: %v", err)
		}

		info := &projectInfo{
			ProjectRoot:   tmp,
			SourceRoot:    appDir,
			PyprojectPath: filepath.Join(appDir, "pyproject.toml"),
		}
		return input, info, outDir
	}

	readRequirements := func(t *testing.T, outDir string) string {
		content, err := os.ReadFile(filepath.Join(outDir, "requirements.txt"))
		if err != nil {
			t.Fatalf("failed to read requirements.txt: %v", err)
		}
		return string(content)
	}

	t.Run("member above workspace root is re-homed and requirements rewritten", func(t *testing.T) {
		input, info, outDir := setup(t, "# generated\n.\n../../lib\nboto3==1.34.0\n")

		if err := copyWorkspacePackagesForContainer(input, info); err != nil {
			t.Fatalf("copyWorkspacePackagesForContainer failed: %v", err)
		}

		got := readRequirements(t, outDir)
		want := "# generated\n.\n./lib\nboto3==1.34.0\n"
		if got != want {
			t.Errorf("requirements.txt = %q, want %q", got, want)
		}

		// Package copied inside the build context, not above it
		if _, err := os.Stat(filepath.Join(outDir, "lib", "pyproject.toml")); err != nil {
			t.Error("lib/pyproject.toml should have been copied into the build context")
		}
		if _, err := os.Stat(filepath.Join(outDir, "lib", "src", "shared", "__init__.py")); err != nil {
			t.Error("lib package source should have been copied into the build context")
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(filepath.Dir(outDir)), "lib")); err == nil {
			t.Error("lib should not have been copied outside the build context")
		}

		// The artifact's pyproject.toml must point at the re-homed member, or
		// uv's workspace validation fails when building the project in-image.
		pyproject, err := os.ReadFile(filepath.Join(outDir, "pyproject.toml"))
		if err != nil {
			t.Fatalf("artifact pyproject.toml should exist: %v", err)
		}
		if !strings.Contains(string(pyproject), "\"./lib\"") {
			t.Errorf("artifact pyproject.toml should reference \"./lib\", got: %s", pyproject)
		}
		if strings.Contains(string(pyproject), "../../lib") {
			t.Errorf("artifact pyproject.toml still references \"../../lib\": %s", pyproject)
		}

		// The source tree's pyproject.toml must be untouched
		srcPyproject, err := os.ReadFile(info.PyprojectPath)
		if err != nil {
			t.Fatalf("failed to read source pyproject.toml: %v", err)
		}
		if !strings.Contains(string(srcPyproject), "../../lib") {
			t.Error("source pyproject.toml should not have been modified")
		}
	})

	t.Run("extras and markers are preserved on rewritten lines", func(t *testing.T) {
		input, info, outDir := setup(t, "../../lib[crypto] ; python_version >= \"3.11\"\n")

		if err := copyWorkspacePackagesForContainer(input, info); err != nil {
			t.Fatalf("copyWorkspacePackagesForContainer failed: %v", err)
		}

		got := readRequirements(t, outDir)
		want := "./lib[crypto] ; python_version >= \"3.11\"\n"
		if got != want {
			t.Errorf("requirements.txt = %q, want %q", got, want)
		}
	})

	t.Run("member inside workspace root is unchanged", func(t *testing.T) {
		input, info, outDir := setup(t, "./core\nboto3==1.34.0\n")

		coreDir := filepath.Join(filepath.Dir(info.PyprojectPath), "core")
		if err := os.MkdirAll(coreDir, 0755); err != nil {
			t.Fatalf("failed to create core dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(coreDir, "pyproject.toml"), []byte("[project]\nname = \"core\"\n"), 0644); err != nil {
			t.Fatalf("failed to write core pyproject: %v", err)
		}

		if err := copyWorkspacePackagesForContainer(input, info); err != nil {
			t.Fatalf("copyWorkspacePackagesForContainer failed: %v", err)
		}

		got := readRequirements(t, outDir)
		want := "./core\nboto3==1.34.0\n"
		if got != want {
			t.Errorf("requirements.txt = %q, want %q", got, want)
		}
		if _, err := os.Stat(filepath.Join(outDir, "core", "pyproject.toml")); err != nil {
			t.Error("core package should have been copied into the build context")
		}
	})

	t.Run("missing member directory leaves requirements unchanged", func(t *testing.T) {
		input, info, outDir := setup(t, "../../missing\nboto3==1.34.0\n")

		if err := copyWorkspacePackagesForContainer(input, info); err != nil {
			t.Fatalf("copyWorkspacePackagesForContainer failed: %v", err)
		}

		got := readRequirements(t, outDir)
		want := "../../missing\nboto3==1.34.0\n"
		if got != want {
			t.Errorf("requirements.txt = %q, want %q", got, want)
		}
	})
}
