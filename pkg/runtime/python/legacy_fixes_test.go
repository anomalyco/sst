package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

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

		ib := &IncrementalBuilder{}
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

	// Test 2: Requirements filtering fix for local file references
	t.Run("Local dependency filtering regression", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sst-requirements-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create problematic requirements.txt
		requirementsContent := `requests==2.31.0
file:///Users/dramus/tce_new/functions/lib
-e ./local-package
../parent-package
/absolute/path/to/package
boto3>=1.34.0`

		inputPath := filepath.Join(tempDir, "requirements.txt")
		outputPath := filepath.Join(tempDir, "requirements-filtered.txt")

		if err := os.WriteFile(inputPath, []byte(requirementsContent), 0644); err != nil {
			t.Fatalf("Failed to write requirements.txt: %v", err)
		}

		projectInfo := &ProjectInfo{
			SourceRoot: tempDir,
		}

		ib := &IncrementalBuilder{}
		err = ib.filterWorkspacePackagesFromRequirements(inputPath, outputPath, projectInfo)
		if err != nil {
			t.Fatalf("filterWorkspacePackagesFromRequirements failed: %v", err)
		}

		// Read filtered content
		filteredContent, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read filtered requirements: %v", err)
		}

		filteredStr := string(filteredContent)

		// Verify problematic entries were filtered out
		if strings.Contains(filteredStr, "file://") {
			t.Errorf("file:// URL was not filtered out")
		}
		if strings.Contains(filteredStr, "-e ./") {
			t.Errorf("Editable local package was not filtered out")
		}
		if strings.Contains(filteredStr, "../parent-package") {
			t.Errorf("Relative path was not filtered out")
		}
		if strings.Contains(filteredStr, "/absolute/path") {
			t.Errorf("Absolute path was not filtered out")
		}

		// Verify valid packages were kept
		if !strings.Contains(filteredStr, "requests==2.31.0") {
			t.Errorf("Valid package was filtered out")
		}
		if !strings.Contains(filteredStr, "boto3>=1.34.0") {
			t.Errorf("Valid package was filtered out")
		}
	})
}
