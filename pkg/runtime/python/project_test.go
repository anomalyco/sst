package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupSourceRoot_MonorepoStructure(t *testing.T) {
	// Create a temp directory structure that mimics the GTF monorepo:
	// /tmp/gtfd/                              <- root with workspace pyproject.toml
	// /tmp/gtfd/apps/main/                    <- SST project root
	// /tmp/gtfd/apps/main/packages/api/       <- workspace member with its own pyproject.toml
	// /tmp/gtfd/apps/main/packages/api/auth/  <- handler files
	//
	// In the real GTF layout, each workspace member package has its own pyproject.toml.
	// The resolver walks up from the handler file and finds the package-level one first,
	// which is within the project root boundary. It never needs to go above apps/main/.

	tmpDir, err := os.MkdirTemp("", "sst-test-monorepo")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure
	appsMainDir := filepath.Join(tmpDir, "apps", "main")
	packagesApiDir := filepath.Join(appsMainDir, "packages", "api")
	authDir := filepath.Join(packagesApiDir, "auth")

	if err := os.MkdirAll(authDir, 0755); err != nil {
		t.Fatalf("Failed to create auth dir: %v", err)
	}

	// Create root pyproject.toml (workspace root, above SST project)
	rootPyproject := filepath.Join(tmpDir, "pyproject.toml")
	if err := os.WriteFile(rootPyproject, []byte(`[project]
name = "gtf"
version = "1.0.0"

[tool.uv.workspace]
members = ["apps/main/packages/api"]
`), 0644); err != nil {
		t.Fatalf("Failed to write root pyproject.toml: %v", err)
	}

	// Create package-level pyproject.toml (this is what the resolver actually finds)
	packagePyproject := filepath.Join(packagesApiDir, "pyproject.toml")
	if err := os.WriteFile(packagePyproject, []byte(`[project]
name = "gtf-api"
version = "0.1.0"
requires-python = ">=3.13"
`), 0644); err != nil {
		t.Fatalf("Failed to write package pyproject.toml: %v", err)
	}

	// Create handler file
	handlerPath := filepath.Join(authDir, "login.py")
	if err := os.WriteFile(handlerPath, []byte(`def handler(event, context):
    return {"statusCode": 200}
`), 0644); err != nil {
		t.Fatalf("Failed to write handler: %v", err)
	}

	// Create resolver with apps/main as project root (simulating SST app location)
	resolver := NewProjectResolver(appsMainDir)

	// Resolve the handler
	info, err := resolver.ResolveHandler("packages/api/auth/login.handler")
	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	// Key assertions:
	// 1. PyprojectPath should be the package-level one (within project root)
	if info.PyprojectPath != packagePyproject {
		t.Errorf("Expected PyprojectPath=%s, got %s", packagePyproject, info.PyprojectPath)
	}

	// 2. SourceRoot should be the package directory (where pyproject.toml was found)
	if info.SourceRoot != packagesApiDir {
		t.Errorf("Expected SourceRoot=%s, got %s", packagesApiDir, info.SourceRoot)
	}

	// 3. ProjectRoot should be apps/main
	if info.ProjectRoot != appsMainDir {
		t.Errorf("Expected ProjectRoot=%s, got %s", appsMainDir, info.ProjectRoot)
	}

	t.Logf("PyprojectPath: %s", info.PyprojectPath)
	t.Logf("SourceRoot: %s", info.SourceRoot)
	t.Logf("ProjectRoot: %s", info.ProjectRoot)
	t.Logf("HandlerFile: %s", info.HandlerFile)
	t.Logf("ModulePath: %s", info.ModulePath)
}

func TestSetupSourceRoot_PyprojectInProjectRoot(t *testing.T) {
	// Standard case: pyproject.toml is in the SST project root
	tmpDir, err := os.MkdirTemp("", "sst-test-standard")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	packagesDir := filepath.Join(tmpDir, "packages", "api")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatalf("Failed to create packages dir: %v", err)
	}

	// Create pyproject.toml in project root
	pyprojectPath := filepath.Join(tmpDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(`[project]
name = "test"
version = "1.0.0"
`), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	// Create handler
	handlerPath := filepath.Join(packagesDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte(`def handler(event, context): pass`), 0644); err != nil {
		t.Fatalf("Failed to write handler: %v", err)
	}

	resolver := NewProjectResolver(tmpDir)
	info, err := resolver.ResolveHandler("packages/api/handler.handler")
	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	// In standard case, SourceRoot should equal ProjectRoot
	if info.SourceRoot != tmpDir {
		t.Errorf("Expected SourceRoot=%s, got %s", tmpDir, info.SourceRoot)
	}

	t.Logf("SourceRoot: %s", info.SourceRoot)
	t.Logf("ProjectRoot: %s", info.ProjectRoot)
}
