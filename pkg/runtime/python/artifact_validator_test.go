package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactValidator_ValidateArtifact(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "artifact_validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock artifact structure
	// Create a Python module directory
	moduleDir := filepath.Join(tempDir, "functions")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("Failed to create module directory: %v", err)
	}

	// Create __init__.py file
	initPyPath := filepath.Join(moduleDir, "__init__.py")
	if err := os.WriteFile(initPyPath, []byte("# Functions module\n"), 0644); err != nil {
		t.Fatalf("Failed to create __init__.py: %v", err)
	}

	// Create a handler file
	handlerPath := filepath.Join(moduleDir, "handler.py")
	handlerContent := `
def lambda_handler(event, context):
    return {"statusCode": 200, "body": "Hello World"}
`
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	// Create requirements.txt
	requirementsPath := filepath.Join(tempDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("requests==2.28.1\n"), 0644); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	// Test the validator
	expectedModules := []string{"functions"}
	maxSizeBytes := int64(10 * 1024 * 1024) // 10MB limit

	validator := NewArtifactValidator(tempDir, expectedModules, maxSizeBytes)
	result, err := validator.ValidateArtifact("functions/handler.lambda_handler")

	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected validation to succeed, but it failed with errors: %v", result.ErrorMessages)
	}

	if !result.HandlerCompatible {
		t.Errorf("Expected handler to be compatible, but it was not")
	}

	if !result.SizeWithinLimits {
		t.Errorf("Expected size to be within limits, but it was not")
	}

	if len(result.PythonModules) != 1 || result.PythonModules[0] != "functions" {
		t.Errorf("Expected to find 'functions' module, but found: %v", result.PythonModules)
	}

	if len(result.MissingModules) != 0 {
		t.Errorf("Expected no missing modules, but found: %v", result.MissingModules)
	}
}

func TestArtifactValidator_ValidateArtifact_MissingModule(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "artifact_validator_test_missing")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create only requirements.txt, no modules
	requirementsPath := filepath.Join(tempDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("requests==2.28.1\n"), 0644); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	// Test the validator with expected modules that don't exist
	expectedModules := []string{"functions", "core"}
	maxSizeBytes := int64(10 * 1024 * 1024) // 10MB limit

	validator := NewArtifactValidator(tempDir, expectedModules, maxSizeBytes)
	result, err := validator.ValidateArtifact("functions/handler.lambda_handler")

	// Should not return an error, but result should indicate failure
	if err != nil {
		t.Fatalf("Validation should not return error, but got: %v", err)
	}

	if result.Success {
		t.Errorf("Expected validation to fail, but it succeeded")
	}

	if len(result.MissingModules) != 2 {
		t.Errorf("Expected 2 missing modules, but found: %v", result.MissingModules)
	}

	if result.HandlerCompatible {
		t.Errorf("Expected handler to be incompatible due to missing module, but it was compatible")
	}
}

func TestArtifactValidator_ValidateArtifact_SizeLimit(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "artifact_validator_test_size")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a Python module directory
	moduleDir := filepath.Join(tempDir, "functions")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("Failed to create module directory: %v", err)
	}

	// Create __init__.py file
	initPyPath := filepath.Join(moduleDir, "__init__.py")
	if err := os.WriteFile(initPyPath, []byte("# Functions module\n"), 0644); err != nil {
		t.Fatalf("Failed to create __init__.py: %v", err)
	}

	// Create a large file to exceed size limit
	largePath := filepath.Join(moduleDir, "large_file.py")
	largeContent := make([]byte, 2048) // 2KB file
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	if err := os.WriteFile(largePath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Test the validator with a very small size limit
	expectedModules := []string{"functions"}
	maxSizeBytes := int64(1024) // 1KB limit (smaller than our file)

	validator := NewArtifactValidator(tempDir, expectedModules, maxSizeBytes)
	result, err := validator.ValidateArtifact("functions/handler.lambda_handler")

	// Should not return an error, but result should indicate failure
	if err != nil {
		t.Fatalf("Validation should not return error, but got: %v", err)
	}

	if result.Success {
		t.Errorf("Expected validation to fail due to size, but it succeeded")
	}

	if result.SizeWithinLimits {
		t.Errorf("Expected size to exceed limits, but it was within limits")
	}

	// Should still find the module even though size is exceeded
	if len(result.PythonModules) != 1 || result.PythonModules[0] != "functions" {
		t.Errorf("Expected to find 'functions' module despite size issue, but found: %v", result.PythonModules)
	}
}

func TestArtifactValidator_ListArtifactContents(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "artifact_validator_test_list")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some test files and directories
	moduleDir := filepath.Join(tempDir, "functions")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("Failed to create module directory: %v", err)
	}

	files := []string{
		filepath.Join(moduleDir, "__init__.py"),
		filepath.Join(moduleDir, "handler.py"),
		filepath.Join(tempDir, "requirements.txt"),
	}

	for _, file := range files {
		if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Test listing contents
	validator := NewArtifactValidator(tempDir, []string{}, 0)
	contents, err := validator.ListArtifactContents()

	if err != nil {
		t.Fatalf("Failed to list artifact contents: %v", err)
	}

	if len(contents) == 0 {
		t.Errorf("Expected to find contents, but list was empty")
	}

	// Check that we found the expected items
	foundDir := false
	foundFiles := 0

	for _, item := range contents {
		if item == "DIR:  functions/" {
			foundDir = true
		}
		if item == "FILE: functions/__init__.py (12 bytes)" ||
			item == "FILE: functions/handler.py (12 bytes)" ||
			item == "FILE: requirements.txt (12 bytes)" {
			foundFiles++
		}
	}

	if !foundDir {
		t.Errorf("Expected to find functions directory in contents: %v", contents)
	}

	if foundFiles != 3 {
		t.Errorf("Expected to find 3 files, but found %d in contents: %v", foundFiles, contents)
	}
}
