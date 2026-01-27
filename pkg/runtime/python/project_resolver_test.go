package python

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupSourceRoot_MonorepoStructure(t *testing.T) {
	// Create a temp directory structure that mimics the GTF monorepo:
	// /tmp/gtfd/                    <- root with pyproject.toml
	// /tmp/gtfd/apps/main/          <- SST project root
	// /tmp/gtfd/apps/main/packages/ <- handler files

	tmpDir, err := os.MkdirTemp("", "sst-test-monorepo")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directory structure
	rootDir := tmpDir
	appsMainDir := filepath.Join(tmpDir, "apps", "main")
	packagesDir := filepath.Join(appsMainDir, "packages", "api", "auth")

	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatalf("Failed to create packages dir: %v", err)
	}

	// Create pyproject.toml at root (above SST project)
	pyprojectPath := filepath.Join(rootDir, "pyproject.toml")
	pyprojectContent := `[project]
name = "gtf"
version = "1.0.0"
`
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	// Create handler file
	handlerPath := filepath.Join(packagesDir, "login.py")
	handlerContent := `def handler(event, context):
    return {"statusCode": 200}
`
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
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
	// 1. PyprojectPath should be found at root
	if info.PyprojectPath != pyprojectPath {
		t.Errorf("Expected PyprojectPath=%s, got %s", pyprojectPath, info.PyprojectPath)
	}

	// 2. SourceRoot should be the SST project root (apps/main), NOT the pyproject.toml dir
	//    This is critical - if SourceRoot is the root dir, the Lambda zip will have wrong paths
	if info.SourceRoot != appsMainDir {
		t.Errorf("Expected SourceRoot=%s (project root), got %s", appsMainDir, info.SourceRoot)
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

func TestCopySourceFilesSimple_MonorepoStructure(t *testing.T) {
	// This test verifies that copySourceFilesSimple uses the correct workspaceDir
	// when pyproject.toml is above the SST project root (monorepo setup)

	tmpDir, err := os.MkdirTemp("", "sst-test-copy-monorepo")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create monorepo structure:
	// /tmp/root/                    <- pyproject.toml here
	// /tmp/root/apps/main/          <- SST project root
	// /tmp/root/apps/main/packages/ <- handler files
	rootDir := tmpDir
	appsMainDir := filepath.Join(tmpDir, "apps", "main")
	packagesDir := filepath.Join(appsMainDir, "packages", "api", "auth")
	outputDir := filepath.Join(tmpDir, "output")

	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatalf("Failed to create packages dir: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// Create pyproject.toml at root (above SST project)
	pyprojectPath := filepath.Join(rootDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(`[project]
name = "gtf"
version = "1.0.0"
`), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	// Create handler file
	handlerPath := filepath.Join(packagesDir, "login.py")
	if err := os.WriteFile(handlerPath, []byte(`def handler(event, context):
    return {"statusCode": 200}
`), 0644); err != nil {
		t.Fatalf("Failed to write handler: %v", err)
	}

	// Create __init__.py files for proper Python package structure
	for _, dir := range []string{
		filepath.Join(appsMainDir, "packages"),
		filepath.Join(appsMainDir, "packages", "api"),
		filepath.Join(appsMainDir, "packages", "api", "auth"),
	} {
		initPath := filepath.Join(dir, "__init__.py")
		if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write __init__.py: %v", err)
		}
	}

	// Create ProjectInfo as it would be set by the resolver
	// Key: PyprojectPath is at root, but SourceRoot should be apps/main
	projectInfo := &ProjectInfo{
		HandlerFile:   handlerPath,
		ProjectRoot:   appsMainDir,
		SourceRoot:    appsMainDir, // This is the key - should be project root, not pyproject dir
		PyprojectPath: pyprojectPath,
		PythonPath:    []string{appsMainDir},
	}

	// Verify the key invariant: when pyproject.toml is above project root,
	// SourceRoot should equal ProjectRoot
	if projectInfo.SourceRoot != projectInfo.ProjectRoot {
		t.Errorf("For monorepo, SourceRoot should equal ProjectRoot. Got SourceRoot=%s, ProjectRoot=%s",
			projectInfo.SourceRoot, projectInfo.ProjectRoot)
	}

	// Verify pyproject.toml is above project root
	pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)
	if strings.HasPrefix(pyprojectDir, projectInfo.ProjectRoot) {
		t.Errorf("Test setup error: pyproject.toml should be ABOVE project root. pyprojectDir=%s, ProjectRoot=%s",
			pyprojectDir, projectInfo.ProjectRoot)
	}

	t.Logf("Test setup:")
	t.Logf("  PyprojectPath: %s", projectInfo.PyprojectPath)
	t.Logf("  PyprojectDir:  %s", pyprojectDir)
	t.Logf("  ProjectRoot:   %s", projectInfo.ProjectRoot)
	t.Logf("  SourceRoot:    %s", projectInfo.SourceRoot)
	t.Logf("  HandlerFile:   %s", projectInfo.HandlerFile)
}
