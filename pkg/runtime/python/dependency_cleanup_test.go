package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIncrementalBuilder_CleanupInstalledDependencies(t *testing.T) {
	// Create a temporary directory simulating installed dependencies
	tempDir, err := os.MkdirTemp("", "dependency_cleanup_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test structure simulating installed Python packages with __pycache__
	testFiles := map[string]string{
		// Normal package files (should be kept)
		"requests/__init__.py": "# requests package",
		"requests/api.py":      "# requests api",
		"boto3/__init__.py":    "# boto3 package",
		"boto3/session.py":     "# boto3 session",

		// __pycache__ directories and files (should be removed)
		"requests/__pycache__/__init__.cpython-312.pyc": "compiled python",
		"requests/__pycache__/api.cpython-312.pyc":      "compiled python",
		"boto3/__pycache__/__init__.cpython-312.pyc":    "compiled python",
		"boto3/__pycache__/session.cpython-312.pyc":     "compiled python",

		// Compiled files outside __pycache__ (should be removed)
		"some_module.pyc": "compiled python",
		"another.pyo":     "compiled python",
		"native.pyd":      "compiled python",

		// Other files (should be kept)
		"package.json":     "{}",
		"README.md":        "# Package readme",
		"data/config.json": "{}",
	}

	// Create all test files
	for filePath, content := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", filePath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	// Create IncrementalBuilder and run cleanup
	builder := &IncrementalBuilder{}
	if err := builder.cleanupInstalledDependencies(tempDir); err != nil {
		t.Fatalf("cleanupInstalledDependencies failed: %v", err)
	}

	// Verify results
	var remainingFiles []string
	var removedFiles []string

	for filePath := range testFiles {
		fullPath := filepath.Join(tempDir, filePath)
		if _, err := os.Stat(fullPath); err == nil {
			remainingFiles = append(remainingFiles, filePath)
		} else if os.IsNotExist(err) {
			removedFiles = append(removedFiles, filePath)
		}
	}

	// Check that __pycache__ files were removed
	expectedRemoved := []string{
		"requests/__pycache__/__init__.cpython-312.pyc",
		"requests/__pycache__/api.cpython-312.pyc",
		"boto3/__pycache__/__init__.cpython-312.pyc",
		"boto3/__pycache__/session.cpython-312.pyc",
		"some_module.pyc",
		"another.pyo",
		"native.pyd",
	}

	for _, expectedFile := range expectedRemoved {
		found := false
		for _, removedFile := range removedFiles {
			if removedFile == expectedFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s to be removed, but it still exists", expectedFile)
		}
	}

	// Check that normal files were kept
	expectedKept := []string{
		"requests/__init__.py",
		"requests/api.py",
		"boto3/__init__.py",
		"boto3/session.py",
		"package.json",
		"README.md",
		"data/config.json",
	}

	for _, expectedFile := range expectedKept {
		found := false
		for _, remainingFile := range remainingFiles {
			if remainingFile == expectedFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s to be kept, but it was removed", expectedFile)
		}
	}

	// Verify __pycache__ directories were removed
	pycacheDirs := []string{
		filepath.Join(tempDir, "requests", "__pycache__"),
		filepath.Join(tempDir, "boto3", "__pycache__"),
	}

	for _, dir := range pycacheDirs {
		if _, err := os.Stat(dir); err == nil {
			t.Errorf("__pycache__ directory %s should have been removed", dir)
		}
	}

	t.Logf("Cleanup completed successfully:")
	t.Logf("  Removed files: %v", removedFiles)
	t.Logf("  Remaining files: %v", remainingFiles)
}
