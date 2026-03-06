package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewIncrementalBuilder(t *testing.T) {
	tempDir := t.TempDir()

	config := IncrementalBuilderConfig{
		CacheDir:    tempDir,
		ArtifactDir: filepath.Join(tempDir, "artifacts"),
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	if builder == nil {
		t.Fatal("Builder is nil")
	}
	if builder.projectResolver == nil {
		t.Error("Project resolver is nil")
	}
	if builder.uvRunner == nil {
		t.Error("UV runner is nil")
	}
}

func TestLocalPackageInfo_Creation(t *testing.T) {
	pkg := &LocalPackageInfo{
		Name: "test-package",
		Path: "/path/to/package",
	}

	if pkg.Name != "test-package" {
		t.Errorf("Expected package name 'test-package', got '%s'", pkg.Name)
	}
}

func TestIncrementalBuilder_CleanupInstalledDependencies(t *testing.T) {
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

	builder := &IncrementalBuilder{}
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
