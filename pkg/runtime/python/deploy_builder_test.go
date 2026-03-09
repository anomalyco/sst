package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

func TestNewDeployBuilder(t *testing.T) {
	tempDir := t.TempDir()

	config := DeployBuilderConfig{
		CacheDir:    tempDir,
		ArtifactDir: filepath.Join(tempDir, "artifacts"),
	}

	builder, err := NewDeployBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create deploy builder: %v", err)
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

	builder := &DeployBuilder{}
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

		projectInfo := &ProjectInfo{
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

		ib := &DeployBuilder{}
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

	// Test 2: Requirements filtering - local paths are now passed through to uv pip install
	t.Run("Local dependency passthrough to uv", func(t *testing.T) {
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

		ib := &DeployBuilder{}
		err = ib.filterEditableInstalls(inputPath, outputPath, tempDir)
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
