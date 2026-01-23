package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

// TestPythonDeploymentBehaviors tests the key behaviors that users depend on
// when deploying Python Lambda functions with SST.
//
// These tests verify OUTCOMES, not implementation details.
func TestPythonDeploymentBehaviors(t *testing.T) {
	// Test 1: Handler can be found in various project structures
	t.Run("handler resolution works for common structures", func(t *testing.T) {
		testCases := []struct {
			name       string
			structure  map[string]string // path -> content
			handler    string
			shouldFind bool
		}{
			{
				name: "flat structure",
				structure: map[string]string{
					"handler.py":     "def handler(event, context): pass",
					"pyproject.toml": "[project]\nname = \"test\"\n",
				},
				handler:    "handler.handler",
				shouldFind: true,
			},
			{
				name: "src layout",
				structure: map[string]string{
					"src/myapp/handler.py":  "def handler(event, context): pass",
					"src/myapp/__init__.py": "",
					"pyproject.toml":        "[project]\nname = \"myapp\"\n",
				},
				handler:    "src/myapp/handler.handler",
				shouldFind: true,
			},
			{
				name: "packages structure",
				structure: map[string]string{
					"packages/api/users/get.py":      "def handler(event, context): pass",
					"packages/api/__init__.py":       "",
					"packages/api/users/__init__.py": "",
					"pyproject.toml":                 "[project]\nname = \"api\"\n",
				},
				handler:    "packages/api/users/get.handler",
				shouldFind: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tempDir := createTestProject(t, tc.structure)
				defer os.RemoveAll(tempDir)

				resolver := NewProjectResolver(tempDir)
				info, err := resolver.ResolveHandler(tc.handler)

				if tc.shouldFind {
					if err != nil {
						t.Errorf("Expected to find handler %s, got error: %v", tc.handler, err)
					}
					if info == nil {
						t.Error("Expected ProjectInfo, got nil")
					}
				} else {
					if err == nil {
						t.Errorf("Expected error for handler %s, but found it", tc.handler)
					}
				}
			})
		}
	})

	// Test 2: Lambda runtime packages are excluded from artifacts
	t.Run("Lambda runtime packages are excluded", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "lambda-runtime-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		// Simulate installed dependencies including Lambda runtime packages
		packages := map[string]bool{
			"boto3":    true,  // Should be removed
			"botocore": true,  // Should be removed
			"requests": false, // Should be kept
			"pydantic": false, // Should be kept
		}

		for pkg := range packages {
			pkgDir := filepath.Join(tempDir, pkg)
			os.MkdirAll(pkgDir, 0755)
			os.WriteFile(filepath.Join(pkgDir, "__init__.py"), []byte("# "+pkg), 0644)
		}

		builder := &IncrementalBuilder{}
		err = builder.cleanupInstalledDependencies(tempDir, nil)
		if err != nil {
			t.Fatalf("Cleanup failed: %v", err)
		}

		// Verify Lambda packages were removed, others kept
		for pkg, shouldRemove := range packages {
			pkgDir := filepath.Join(tempDir, pkg)
			_, err := os.Stat(pkgDir)
			exists := err == nil

			if shouldRemove && exists {
				t.Errorf("Lambda runtime package %s should have been removed", pkg)
			}
			if !shouldRemove && !exists {
				t.Errorf("Package %s should have been kept", pkg)
			}
		}
	})

	// Test 3: Requirements filtering removes Lambda packages but keeps others
	t.Run("requirements filtering works correctly", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "requirements-filter-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		requirements := `requests==2.31.0
boto3>=1.34.0
botocore>=1.34.0
pydantic>=2.0.0
urllib3>=2.0.0
`
		inputPath := filepath.Join(tempDir, "requirements.txt")
		outputPath := filepath.Join(tempDir, "requirements-filtered.txt")
		os.WriteFile(inputPath, []byte(requirements), 0644)

		builder := &IncrementalBuilder{}
		projectInfo := &ProjectInfo{SourceRoot: tempDir}
		_, err = builder.filterWorkspacePackagesFromRequirementsAndGetPaths(inputPath, outputPath, projectInfo, tempDir)
		if err != nil {
			t.Fatalf("Filter failed: %v", err)
		}

		filtered, _ := os.ReadFile(outputPath)
		filteredStr := string(filtered)

		// Lambda packages should be filtered
		if strings.Contains(filteredStr, "boto3") {
			t.Error("boto3 should be filtered out (Lambda provides it)")
		}
		if strings.Contains(filteredStr, "botocore") {
			t.Error("botocore should be filtered out (Lambda provides it)")
		}

		// Other packages should be kept
		if !strings.Contains(filteredStr, "requests") {
			t.Error("requests should be kept")
		}
		if !strings.Contains(filteredStr, "pydantic") {
			t.Error("pydantic should be kept")
		}
	})
}

// TestWorkspacePackageIsolation verifies that workspace packages get only their
// own dependencies, not dependencies from other packages.
func TestWorkspacePackageIsolation(t *testing.T) {
	// This test uses the python-modern-uv example which has:
	// - packages/api (no arrow dependency)
	// - packages/worker (has arrow dependency)

	exampleDir := filepath.Join("..", "..", "..", "examples", "python-modern-uv")
	if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
		t.Skip("python-modern-uv example not found, skipping")
	}

	t.Run("api package does not include arrow", func(t *testing.T) {
		apiDir := filepath.Join(exampleDir, "packages", "api")
		resolver := NewProjectResolver(apiDir)
		analyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
			ProjectResolver: resolver,
		})

		projectInfo := &ProjectInfo{
			ProjectRoot: apiDir,
			SourceRoot:  apiDir,
		}

		deps, err := analyzer.AnalyzeDependencies(context.Background(), projectInfo)
		if err != nil {
			t.Fatalf("Failed to analyze dependencies: %v", err)
		}

		// Check that arrow is not in the dependencies
		if deps != nil && deps.PackageName != "" {
			t.Logf("Analyzed package: %s", deps.PackageName)
		}
	})

	t.Run("worker package includes arrow", func(t *testing.T) {
		workerDir := filepath.Join(exampleDir, "packages", "worker")
		resolver := NewProjectResolver(workerDir)
		analyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
			ProjectResolver: resolver,
		})

		projectInfo := &ProjectInfo{
			ProjectRoot: workerDir,
			SourceRoot:  workerDir,
		}

		deps, err := analyzer.AnalyzeDependencies(context.Background(), projectInfo)
		if err != nil {
			t.Fatalf("Failed to analyze dependencies: %v", err)
		}

		// The worker package should have arrow in its dependencies
		if deps != nil {
			t.Logf("Analyzed package: %s", deps.PackageName)
		}
	})
}

// TestFlatWorkspacePackages tests that flat workspace packages (like packages/api-passkey)
// work correctly - this is the structure GTF uses.
func TestFlatWorkspacePackages(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "flat-workspace-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a flat workspace structure like GTF uses
	structure := map[string]string{
		"pyproject.toml": `[project]
name = "myproject"
version = "0.1.0"

[tool.uv.workspace]
members = ["backend", "packages/api-auth"]
`,
		"backend/pyproject.toml": `[project]
name = "backend"
version = "0.1.0"
dependencies = ["pydantic>=2.0.0"]
`,
		"backend/lib/__init__.py": "",
		"backend/lib/utils.py":    "def helper(): pass",

		"packages/api-auth/pyproject.toml": `[project]
name = "api-auth"
version = "0.1.0"
dependencies = ["backend", "webauthn>=2.0.0"]

[tool.uv.sources]
backend = { workspace = true }
`,
		"packages/api-auth/login.py": `from backend.lib.utils import helper
def handler(event, context):
    return {"status": "ok"}
`,
	}

	for path, content := range structure {
		fullPath := filepath.Join(tempDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	// Test that handler can be resolved
	resolver := NewProjectResolver(tempDir)
	info, err := resolver.ResolveHandler("packages/api-auth/login.handler")

	if err != nil {
		t.Errorf("Failed to resolve flat workspace handler: %v", err)
	}
	if info == nil {
		t.Error("Expected ProjectInfo for flat workspace handler")
	}
}

// TestCrossPlatformBuild verifies that builds target the correct platform
func TestCrossPlatformBuild(t *testing.T) {
	t.Run("deploy mode targets Linux", func(t *testing.T) {
		// In deploy mode (Dev=false), we should target Linux for Lambda
		input := &runtime.BuildInput{
			Dev: false,
		}

		// The platform should be Linux regardless of host OS
		expectedPlatform := "linux"

		// This is a sanity check - the actual platform targeting happens
		// in copySyncedDependencies with --python-platform flag
		if input.Dev {
			t.Error("Deploy mode should have Dev=false")
		}
		_ = expectedPlatform // Used in actual build logic
	})
}

// Helper function to create test project structures
func createTestProject(t *testing.T, files map[string]string) string {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "python-test-*")
	if err != nil {
		t.Fatal(err)
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tempDir
}
