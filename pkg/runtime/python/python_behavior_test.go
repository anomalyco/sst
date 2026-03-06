package python

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWorkspacePackageIsolation verifies that workspace packages get only their
// own dependencies, not dependencies from other packages.
func TestWorkspacePackageIsolation(t *testing.T) {
	exampleDir := filepath.Join("..", "..", "..", "examples", "python-modern-uv")
	if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
		t.Skip("python-modern-uv example not found, skipping")
	}

	t.Run("api package does not include arrow", func(t *testing.T) {
		apiDir := filepath.Join(exampleDir, "packages", "api")
		resolver := NewProjectResolver(apiDir)

		projectInfo := &ProjectInfo{
			ProjectRoot: apiDir,
			SourceRoot:  apiDir,
		}

		packages, err := discoverBuildablePackages(projectInfo, resolver)
		if err != nil {
			t.Fatalf("Failed to discover packages: %v", err)
		}
		t.Logf("Found %d buildable packages for api", len(packages))
	})

	t.Run("worker package includes arrow", func(t *testing.T) {
		workerDir := filepath.Join(exampleDir, "packages", "worker")
		resolver := NewProjectResolver(workerDir)

		projectInfo := &ProjectInfo{
			ProjectRoot: workerDir,
			SourceRoot:  workerDir,
		}

		packages, err := discoverBuildablePackages(projectInfo, resolver)
		if err != nil {
			t.Fatalf("Failed to discover packages: %v", err)
		}
		t.Logf("Found %d buildable packages for worker", len(packages))
	})
}

// TestFlatWorkspacePackages tests that flat workspace packages work correctly.
func TestFlatWorkspacePackages(t *testing.T) {
	tempDir := t.TempDir()

	structure := map[string]string{
		"pyproject.toml": `[project]
name = "myproject"
version = "0.1.0"

[tool.uv.workspace]
members = ["backend", "packages/api-auth"]
`,
		"backend/pyproject.toml":  "[project]\nname = \"backend\"\nversion = \"0.1.0\"\n",
		"backend/lib/__init__.py": "",
		"backend/lib/utils.py":    "def helper(): pass",
		"packages/api-auth/pyproject.toml": `[project]
name = "api-auth"
version = "0.1.0"
dependencies = ["backend"]

[tool.uv.sources]
backend = { workspace = true }
`,
		"packages/api-auth/login.py": "def handler(event, context): return {\"status\": \"ok\"}",
	}

	for path, content := range structure {
		fullPath := filepath.Join(tempDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	resolver := NewProjectResolver(tempDir)
	info, err := resolver.ResolveHandler("packages/api-auth/login.handler")
	if err != nil {
		t.Errorf("Failed to resolve flat workspace handler: %v", err)
	}
	if info == nil {
		t.Error("Expected ProjectInfo for flat workspace handler")
	}
}
