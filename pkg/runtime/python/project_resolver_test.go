package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewProjectResolver(t *testing.T) {
	projectRoot := "/test/project"
	resolver := NewProjectResolver(projectRoot)

	if resolver.projectRoot != projectRoot {
		t.Errorf("Expected project root %s, got %s", projectRoot, resolver.projectRoot)
	}

	if resolver.cache == nil {
		t.Error("Cache should be initialized")
	}
}

func TestProjectResolver_ResolveHandler_DirectPath(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple Python file
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(event, context):\n    return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	projectInfo, err := resolver.ResolveHandler("handler.py")

	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	if projectInfo.HandlerFile != handlerFile {
		t.Errorf("Expected handler file %s, got %s", handlerFile, projectInfo.HandlerFile)
	}

	if projectInfo.ProjectRoot != tempDir {
		t.Errorf("Expected project root %s, got %s", tempDir, projectInfo.ProjectRoot)
	}

	if projectInfo.SourceRoot != filepath.Dir(handlerFile) {
		t.Errorf("Expected source root %s, got %s", filepath.Dir(handlerFile), projectInfo.SourceRoot)
	}

	if projectInfo.ModulePath != "handler" {
		t.Errorf("Expected module path 'handler', got %s", projectInfo.ModulePath)
	}
}

func TestProjectResolver_ResolveHandler_WithoutExtension(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple Python file
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(event, context):\n    return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	projectInfo, err := resolver.ResolveHandler("handler")

	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	if projectInfo.HandlerFile != handlerFile {
		t.Errorf("Expected handler file %s, got %s", handlerFile, projectInfo.HandlerFile)
	}

	if projectInfo.ModulePath != "handler" {
		t.Errorf("Expected module path 'handler', got %s", projectInfo.ModulePath)
	}
}

func TestProjectResolver_ResolveHandler_SrcDirectory(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create src directory structure
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	handlerFile := filepath.Join(srcDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(event, context):\n    return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	projectInfo, err := resolver.ResolveHandler("handler.py")

	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	if projectInfo.HandlerFile != handlerFile {
		t.Errorf("Expected handler file %s, got %s", handlerFile, projectInfo.HandlerFile)
	}

	if projectInfo.SourceRoot != tempDir {
		t.Errorf("Expected source root %s, got %s", tempDir, projectInfo.SourceRoot)
	}

	if projectInfo.ModulePath != "src.handler" {
		t.Errorf("Expected module path 'src.handler', got %s", projectInfo.ModulePath)
	}
}

func TestProjectResolver_ResolveHandler_NestedPath(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create nested directory structure
	nestedDir := filepath.Join(tempDir, "app", "handlers")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}

	handlerFile := filepath.Join(nestedDir, "api.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(event, context):\n    return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	projectInfo, err := resolver.ResolveHandler("handlers/api.py")

	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	if projectInfo.HandlerFile != handlerFile {
		t.Errorf("Expected handler file %s, got %s", handlerFile, projectInfo.HandlerFile)
	}

	if projectInfo.ModulePath != "app.handlers.api" {
		t.Errorf("Expected module path 'app.handlers.api', got %s", projectInfo.ModulePath)
	}
}

func TestProjectResolver_ResolveHandler_WithPyprojectToml(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create pyproject.toml
	pyprojectContent := `[project]
name = "test-project"
version = "0.1.0"
dependencies = ["requests>=2.31.0"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create handler file
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(event, context):\n    return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	projectInfo, err := resolver.ResolveHandler("handler.py")

	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	if projectInfo.PyprojectPath != pyprojectPath {
		t.Errorf("Expected pyproject path %s, got %s", pyprojectPath, projectInfo.PyprojectPath)
	}

	if len(projectInfo.Dependencies) == 0 {
		t.Error("Expected dependencies to include pyproject.toml")
	}

	found := false
	for _, dep := range projectInfo.Dependencies {
		if dep == pyprojectPath {
			found = true
			break
		}
	}
	if !found {
		t.Error("pyproject.toml should be in dependencies")
	}
}

func TestProjectResolver_ResolveHandler_WithSrcAndPyprojectToml(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create pyproject.toml
	pyprojectContent := `[project]
name = "test-project"
version = "0.1.0"
dependencies = ["requests>=2.31.0"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create src directory and handler
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	handlerFile := filepath.Join(srcDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(event, context):\n    return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	projectInfo, err := resolver.ResolveHandler("handler.py")

	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	if projectInfo.SourceRoot != srcDir {
		t.Errorf("Expected source root %s, got %s", srcDir, projectInfo.SourceRoot)
	}

	if projectInfo.PyprojectPath != pyprojectPath {
		t.Errorf("Expected pyproject path %s, got %s", pyprojectPath, projectInfo.PyprojectPath)
	}

	// Check that both src and project root are in Python path
	foundSrc := false
	foundRoot := false
	for _, path := range projectInfo.PythonPath {
		if path == srcDir {
			foundSrc = true
		}
		if path == tempDir {
			foundRoot = true
		}
	}
	if !foundSrc {
		t.Error("src directory should be in Python path")
	}
	if !foundRoot {
		t.Error("project root should be in Python path")
	}
}

func TestProjectResolver_ResolveHandler_HandlerNotFound(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	resolver := NewProjectResolver(tempDir)
	_, err = resolver.ResolveHandler("nonexistent.py")

	if err == nil {
		t.Error("Expected error for nonexistent handler")
	}

	// Check that the error contains the expected handler name
	if !strings.Contains(err.Error(), "nonexistent.py") {
		t.Errorf("Expected 'nonexistent.py' in error message, got %s", err.Error())
	}
}

func TestProjectResolver_ResolveHandler_Caching(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create handler file
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(event, context):\n    return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)

	// First resolution
	projectInfo1, err := resolver.ResolveHandler("handler.py")
	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	// Second resolution should use cache
	projectInfo2, err := resolver.ResolveHandler("handler.py")
	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	// Should be the same object (from cache)
	if projectInfo1 != projectInfo2 {
		t.Error("Expected cached result to be the same object")
	}
}

func TestProjectResolver_ParsePyprojectToml(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create pyproject.toml with various configurations
	pyprojectContent := `[project]
name = "test-project"
version = "0.1.0"
description = "A test project"
dependencies = [
    "requests>=2.31.0",
    "boto3>=1.34.0"
]

[tool.uv.sources]
local-package = { path = "../local-package" }

[tool.poetry]
name = "poetry-project"
version = "0.1.0"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	config, err := resolver.ParsePyprojectToml(pyprojectPath)

	if err != nil {
		t.Fatalf("Failed to parse pyproject.toml: %v", err)
	}

	if config.Project.Name != "test-project" {
		t.Errorf("Expected project name 'test-project', got %s", config.Project.Name)
	}

	if config.Project.Version != "0.1.0" {
		t.Errorf("Expected version '0.1.0', got %s", config.Project.Version)
	}

	if len(config.Project.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(config.Project.Dependencies))
	}

	if config.Tool.Poetry.Name != "poetry-project" {
		t.Errorf("Expected poetry name 'poetry-project', got %s", config.Tool.Poetry.Name)
	}

	if config.BuildSystem.BuildBackend != "hatchling.build" {
		t.Errorf("Expected build backend 'hatchling.build', got %s", config.BuildSystem.BuildBackend)
	}

	// Check UV sources
	if localPkg, exists := config.Tool.UV.Sources["local-package"]; !exists {
		t.Error("Expected local-package in UV sources")
	} else if localPkg.Path != "../local-package" {
		t.Errorf("Expected path '../local-package', got %s", localPkg.Path)
	}
}

func TestProjectResolver_ResolvePythonPath(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create handler file
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(event, context):\n    return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	projectInfo, err := resolver.ResolveHandler("handler.py")
	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	pythonPath := resolver.ResolvePythonPath(projectInfo)

	if len(pythonPath) == 0 {
		t.Error("Expected non-empty Python path")
	}

	// Should contain at least the source root
	found := false
	for _, path := range pythonPath {
		if path == projectInfo.SourceRoot {
			found = true
			break
		}
	}
	if !found {
		t.Error("Python path should contain source root")
	}

	// Modifying returned slice should not affect original
	originalLen := len(projectInfo.PythonPath)
	pythonPath = append(pythonPath, "/extra/path")
	if len(projectInfo.PythonPath) != originalLen {
		t.Error("Modifying returned Python path should not affect original")
	}
}

func TestProjectResolver_ClearCache(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create handler file
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(event, context):\n    return 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)

	// Resolve handler to populate cache
	_, err = resolver.ResolveHandler("handler.py")
	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	// Verify cache has entry
	if len(resolver.cache) == 0 {
		t.Error("Expected cache to have entries")
	}

	// Clear cache
	resolver.ClearCache()

	// Verify cache is empty
	if len(resolver.cache) != 0 {
		t.Error("Expected cache to be empty after clearing")
	}
}

func TestProjectResolver_InvalidateCache(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create handler files
	handler1File := filepath.Join(tempDir, "handler1.py")
	if err := os.WriteFile(handler1File, []byte("def handler(event, context):\n    return 'hello1'"), 0644); err != nil {
		t.Fatalf("Failed to create handler1 file: %v", err)
	}

	handler2File := filepath.Join(tempDir, "handler2.py")
	if err := os.WriteFile(handler2File, []byte("def handler(event, context):\n    return 'hello2'"), 0644); err != nil {
		t.Fatalf("Failed to create handler2 file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)

	// Resolve both handlers to populate cache
	_, err = resolver.ResolveHandler("handler1.py")
	if err != nil {
		t.Fatalf("Failed to resolve handler1: %v", err)
	}

	_, err = resolver.ResolveHandler("handler2.py")
	if err != nil {
		t.Fatalf("Failed to resolve handler2: %v", err)
	}

	// Verify cache has both entries
	if len(resolver.cache) != 2 {
		t.Errorf("Expected cache to have 2 entries, got %d", len(resolver.cache))
	}

	// Invalidate one handler
	resolver.InvalidateCache("handler1.py")

	// Verify only one entry remains
	if len(resolver.cache) != 1 {
		t.Errorf("Expected cache to have 1 entry after invalidation, got %d", len(resolver.cache))
	}

	// Verify the correct entry was removed
	if _, exists := resolver.cache["handler1.py"]; exists {
		t.Error("handler1.py should have been removed from cache")
	}

	if _, exists := resolver.cache["handler2.py"]; !exists {
		t.Error("handler2.py should still be in cache")
	}
}

func TestProjectResolver_generateCandidatePaths(t *testing.T) {
	tempDir := "/test/project"
	resolver := NewProjectResolver(tempDir)

	tests := []struct {
		name         string
		handlerPath  string
		expectedSome []string // Some paths we expect to see
	}{
		{
			name:        "simple handler",
			handlerPath: "handler.py",
			expectedSome: []string{
				"/test/project/handler.py",
				"/test/project/src/handler.py",
				"/test/project/app/handler.py",
			},
		},
		{
			name:        "handler without extension",
			handlerPath: "handler",
			expectedSome: []string{
				"/test/project/handler",
				"/test/project/handler.py",
				"/test/project/src/handler.py",
			},
		},
		{
			name:        "nested handler",
			handlerPath: "api/users.py",
			expectedSome: []string{
				"/test/project/api/users.py",
				"/test/project/src/api/users.py",
				"/test/project/app/api/users.py",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := resolver.generateCandidatePaths(tt.handlerPath)

			if len(candidates) == 0 {
				t.Error("Expected some candidate paths")
			}

			// Check that expected paths are present
			for _, expected := range tt.expectedSome {
				found := false
				for _, candidate := range candidates {
					if candidate == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected candidate path %s not found in %v", expected, candidates)
				}
			}
		})
	}
}

func TestProjectResolver_isValidPythonFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	resolver := NewProjectResolver(tempDir)

	// Create valid Python file
	validFile := filepath.Join(tempDir, "valid.py")
	if err := os.WriteFile(validFile, []byte("def test(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create valid file: %v", err)
	}

	// Create non-Python file
	nonPythonFile := filepath.Join(tempDir, "invalid.txt")
	if err := os.WriteFile(nonPythonFile, []byte("not python"), 0644); err != nil {
		t.Fatalf("Failed to create non-Python file: %v", err)
	}

	// Create directory with .py extension (should be invalid)
	dirWithPyExt := filepath.Join(tempDir, "directory.py")
	if err := os.MkdirAll(dirWithPyExt, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"valid Python file", validFile, true},
		{"non-Python file", nonPythonFile, false},
		{"nonexistent file", filepath.Join(tempDir, "nonexistent.py"), false},
		{"directory with .py extension", dirWithPyExt, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.isValidPythonFile(tt.path)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.path, result)
			}
		})
	}
}

func TestProjectResolver_findPyprojectToml(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create nested directory structure
	nestedDir := filepath.Join(tempDir, "src", "package")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}

	// Create pyproject.toml in root
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte("[project]\nname = \"test\""), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create handler in nested directory
	handlerFile := filepath.Join(nestedDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)

	// Should find pyproject.toml by traversing up
	foundPath, err := resolver.findPyprojectToml(handlerFile)
	if err != nil {
		t.Fatalf("Failed to find pyproject.toml: %v", err)
	}

	if foundPath != pyprojectPath {
		t.Errorf("Expected %s, got %s", pyprojectPath, foundPath)
	}
}

func TestProjectResolver_findPyprojectToml_NotFound(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create handler without pyproject.toml
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}

	resolver := NewProjectResolver(tempDir)

	// Should return error when not found
	_, err = resolver.findPyprojectToml(handlerFile)
	if err == nil {
		t.Error("Expected error when pyproject.toml not found")
	}

	if !strings.Contains(err.Error(), "no pyproject.toml found") {
		t.Errorf("Expected 'no pyproject.toml found' in error, got: %v", err)
	}
}

func TestProjectResolver_setupSourceRoot(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "project_resolver_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	resolver := NewProjectResolver(tempDir)

	tests := []struct {
		name           string
		setupFunc      func() *ProjectInfo
		expectedSource string
	}{
		{
			name: "with pyproject.toml and src directory",
			setupFunc: func() *ProjectInfo {
				// Create src directory
				srcDir := filepath.Join(tempDir, "src")
				os.MkdirAll(srcDir, 0755)

				// Create pyproject.toml
				pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
				os.WriteFile(pyprojectPath, []byte("[project]\nname = \"test\""), 0644)

				return &ProjectInfo{
					HandlerFile:   filepath.Join(srcDir, "handler.py"),
					PyprojectPath: pyprojectPath,
				}
			},
			expectedSource: filepath.Join(tempDir, "src"),
		},
		{
			name: "with pyproject.toml but no src directory",
			setupFunc: func() *ProjectInfo {
				// Create pyproject.toml
				pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
				os.WriteFile(pyprojectPath, []byte("[project]\nname = \"test\""), 0644)

				return &ProjectInfo{
					HandlerFile:   filepath.Join(tempDir, "handler.py"),
					PyprojectPath: pyprojectPath,
				}
			},
			expectedSource: tempDir,
		},
		{
			name: "without pyproject.toml",
			setupFunc: func() *ProjectInfo {
				return &ProjectInfo{
					HandlerFile:   filepath.Join(tempDir, "handler.py"),
					PyprojectPath: "",
				}
			},
			expectedSource: tempDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up from previous test
			os.RemoveAll(filepath.Join(tempDir, "src"))
			os.Remove(filepath.Join(tempDir, "pyproject.toml"))

			projectInfo := tt.setupFunc()
			err := resolver.setupSourceRoot(projectInfo)

			if err != nil {
				t.Fatalf("Failed to setup source root: %v", err)
			}

			if projectInfo.SourceRoot != tt.expectedSource {
				t.Errorf("Expected source root %s, got %s", tt.expectedSource, projectInfo.SourceRoot)
			}
		})
	}
}

// Test various project structures as mentioned in the requirements

func TestProjectResolver_VariousProjectStructures(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(tempDir string) (string, error) // Returns handler path
		expectedModule string
		expectError    bool
	}{
		{
			name: "flat structure",
			setupFunc: func(tempDir string) (string, error) {
				handlerFile := filepath.Join(tempDir, "handler.py")
				err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644)
				return "handler.py", err
			},
			expectedModule: "handler",
			expectError:    false,
		},
		{
			name: "workspace structure with pyproject.toml",
			setupFunc: func(tempDir string) (string, error) {
				// Create pyproject.toml
				pyprojectContent := `[project]
name = "workspace-project"
version = "0.1.0"
`
				if err := os.WriteFile(filepath.Join(tempDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
					return "", err
				}

				// Create handler
				handlerFile := filepath.Join(tempDir, "handler.py")
				err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644)
				return "handler.py", err
			},
			expectedModule: "handler",
			expectError:    false,
		},
		{
			name: "nested structure with packages",
			setupFunc: func(tempDir string) (string, error) {
				// Create nested package structure
				packageDir := filepath.Join(tempDir, "mypackage", "subpackage")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					return "", err
				}

				// Create __init__.py files
				if err := os.WriteFile(filepath.Join(tempDir, "mypackage", "__init__.py"), []byte(""), 0644); err != nil {
					return "", err
				}
				if err := os.WriteFile(filepath.Join(packageDir, "__init__.py"), []byte(""), 0644); err != nil {
					return "", err
				}

				// Create handler
				handlerFile := filepath.Join(packageDir, "handler.py")
				err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644)
				return "mypackage/subpackage/handler.py", err
			},
			expectedModule: "mypackage.subpackage.handler",
			expectError:    false,
		},
		{
			name: "src layout with pyproject.toml",
			setupFunc: func(tempDir string) (string, error) {
				// Create pyproject.toml
				pyprojectContent := `[project]
name = "src-project"
version = "0.1.0"
`
				if err := os.WriteFile(filepath.Join(tempDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
					return "", err
				}

				// Create src directory
				srcDir := filepath.Join(tempDir, "src")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					return "", err
				}

				// Create handler in src
				handlerFile := filepath.Join(srcDir, "handler.py")
				err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644)
				return "handler.py", err
			},
			expectedModule: "handler",
			expectError:    false,
		},
		{
			name: "poetry project structure",
			setupFunc: func(tempDir string) (string, error) {
				// Create pyproject.toml with Poetry configuration
				pyprojectContent := `[tool.poetry]
name = "poetry-project"
version = "0.1.0"

[tool.poetry.dependencies]
python = "^3.9"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`
				if err := os.WriteFile(filepath.Join(tempDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
					return "", err
				}

				// Create handler
				handlerFile := filepath.Join(tempDir, "handler.py")
				err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644)
				return "handler.py", err
			},
			expectedModule: "handler",
			expectError:    false,
		},
		{
			name: "functions directory structure",
			setupFunc: func(tempDir string) (string, error) {
				// Create functions directory
				functionsDir := filepath.Join(tempDir, "functions")
				if err := os.MkdirAll(functionsDir, 0755); err != nil {
					return "", err
				}

				// Create handler in functions directory
				handlerFile := filepath.Join(functionsDir, "api.py")
				err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644)
				return "api.py", err
			},
			expectedModule: "functions.api",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "project_resolver_structure_test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup the project structure
			handlerPath, err := tt.setupFunc(tempDir)
			if err != nil {
				t.Fatalf("Failed to setup project structure: %v", err)
			}

			// Test resolution
			resolver := NewProjectResolver(tempDir)
			projectInfo, err := resolver.ResolveHandler(handlerPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if projectInfo.ModulePath != tt.expectedModule {
				t.Errorf("Expected module path %s, got %s", tt.expectedModule, projectInfo.ModulePath)
			}

			// Verify basic properties
			if projectInfo.ProjectRoot != tempDir {
				t.Errorf("Expected project root %s, got %s", tempDir, projectInfo.ProjectRoot)
			}

			if len(projectInfo.PythonPath) == 0 {
				t.Error("Expected non-empty Python path")
			}
		})
	}
}

func TestProjectResolver_DependencyDetection(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "project_resolver_deps_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create pyproject.toml
	pyprojectContent := `[project]
name = "test-project"
version = "0.1.0"
dependencies = ["requests>=2.31.0"]
`
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create uv.lock
	uvLockPath := filepath.Join(tempDir, "uv.lock")
	if err := os.WriteFile(uvLockPath, []byte("# UV lock file"), 0644); err != nil {
		t.Fatalf("Failed to create uv.lock: %v", err)
	}

	// Create requirements.txt
	requirementsPath := filepath.Join(tempDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("requests>=2.31.0"), 0644); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	// Create handler
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	projectInfo, err := resolver.ResolveHandler("handler.py")

	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	// Check that all dependency files are detected
	expectedDeps := []string{pyprojectPath, uvLockPath, requirementsPath}
	for _, expectedDep := range expectedDeps {
		found := false
		for _, dep := range projectInfo.Dependencies {
			if dep == expectedDep {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected dependency %s not found in %v", expectedDep, projectInfo.Dependencies)
		}
	}
}

func TestProjectResolver_PythonPathResolution(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "project_resolver_pythonpath_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create pyproject.toml
	pyprojectContent := `[project]
name = "test-project"
version = "0.1.0"
`
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create src directory
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}

	// Create handler in src
	handlerFile := filepath.Join(srcDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	projectInfo, err := resolver.ResolveHandler("handler.py")

	if err != nil {
		t.Fatalf("Failed to resolve handler: %v", err)
	}

	// Check that Python path contains both src and project root
	pythonPath := resolver.ResolvePythonPath(projectInfo)

	expectedPaths := []string{srcDir, tempDir}
	for _, expectedPath := range expectedPaths {
		found := false
		for _, path := range pythonPath {
			if path == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected Python path %s not found in %v", expectedPath, pythonPath)
		}
	}
}

func TestProjectResolver_ConcurrentAccess(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "project_resolver_concurrent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple handler files
	for i := 0; i < 5; i++ {
		handlerFile := filepath.Join(tempDir, fmt.Sprintf("handler%d.py", i))
		if err := os.WriteFile(handlerFile, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create handler%d: %v", i, err)
		}
	}

	resolver := NewProjectResolver(tempDir)

	// Test concurrent access
	var wg sync.WaitGroup
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			handlerPath := fmt.Sprintf("handler%d.py", index)
			_, err := resolver.ResolveHandler(handlerPath)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}

	// Verify cache has all entries
	if len(resolver.cache) != 5 {
		t.Errorf("Expected 5 cache entries, got %d", len(resolver.cache))
	}
}

// Test configuration parsing with various pyproject.toml formats
func TestProjectResolver_ParsePyprojectToml_VariousFormats(t *testing.T) {
	tests := []struct {
		name           string
		pyprojectToml  string
		expectError    bool
		expectedName   string
		expectedPoetry string
		errorContains  string
	}{
		{
			name: "standard PEP 621 format",
			pyprojectToml: `[project]
name = "standard-project"
version = "0.1.0"
description = "A standard Python project"
dependencies = [
    "requests>=2.31.0",
    "boto3>=1.34.0"
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`,
			expectError:  false,
			expectedName: "standard-project",
		},
		{
			name: "poetry format",
			pyprojectToml: `[tool.poetry]
name = "poetry-project"
version = "0.1.0"
description = "A Poetry project"

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.31.0"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`,
			expectError:    false,
			expectedPoetry: "poetry-project",
		},
		{
			name: "mixed format with both project and poetry",
			pyprojectToml: `[project]
name = "mixed-project"
version = "0.1.0"

[tool.poetry]
name = "poetry-name"
version = "0.1.0"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`,
			expectError:  false,
			expectedName: "mixed-project", // project takes precedence
		},
		{
			name: "UV workspace configuration",
			pyprojectToml: `[project]
name = "uv-workspace"
version = "0.1.0"

[tool.uv.workspace]
members = ["packages/*"]

[tool.uv.sources]
local-package = { path = "../local-package" }
remote-package = { git = "https://github.com/example/repo.git" }

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`,
			expectError:  false,
			expectedName: "uv-workspace",
		},
		{
			name: "minimal valid configuration",
			pyprojectToml: `[project]
name = "minimal"
`,
			expectError:  false,
			expectedName: "minimal",
		},
		{
			name: "hatch configuration",
			pyprojectToml: `[project]
name = "hatch-project"
version = "0.1.0"

[tool.hatch.build.targets.wheel]
packages = ["src/mypackage"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`,
			expectError:  false,
			expectedName: "hatch-project",
		},
		{
			name: "setuptools configuration",
			pyprojectToml: `[project]
name = "setuptools-project"
version = "0.1.0"

[tool.setuptools.packages.find]
where = ["src"]

[build-system]
requires = ["setuptools", "wheel"]
build-backend = "setuptools.build_meta"
`,
			expectError:  false,
			expectedName: "setuptools-project",
		},
		{
			name: "invalid TOML syntax",
			pyprojectToml: `[project]
name = "invalid-project
version = "0.1.0"
`,
			expectError:   true,
			errorContains: "TOML parsing error",
		},
		{
			name: "missing project name",
			pyprojectToml: `[project]
version = "0.1.0"
description = "No name specified"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`,
			expectError:   true,
			errorContains: "project must have a name",
		},
		{
			name: "invalid UV source - multiple types",
			pyprojectToml: `[project]
name = "invalid-uv-source"
version = "0.1.0"

[tool.uv.sources]
bad-source = { path = "../local", git = "https://github.com/example/repo.git" }
`,
			expectError:   true,
			errorContains: "multiple source types",
		},
		{
			name: "invalid UV source - no source specified",
			pyprojectToml: `[project]
name = "invalid-uv-source"
version = "0.1.0"

[tool.uv.sources]
empty-source = { }
`,
			expectError:   true,
			errorContains: "no source specified",
		},
		{
			name: "comprehensive configuration with all fields",
			pyprojectToml: `[project]
name = "comprehensive-project"
version = "0.1.0"
description = "A comprehensive Python project"
authors = [
    {name = "John Doe", email = "john@example.com"}
]
maintainers = [
    {name = "Jane Smith", email = "jane@example.com"}
]
license = {text = "MIT"}
readme = "README.md"
homepage = "https://example.com"
repository = "https://github.com/example/repo"
keywords = ["python", "example"]
classifiers = [
    "Development Status :: 4 - Beta",
    "Programming Language :: Python :: 3"
]
requires-python = ">=3.9"
dependencies = [
    "requests>=2.31.0",
    "boto3>=1.34.0"
]

[tool.uv.dev-dependencies]
pytest = ">=7.0.0"
black = ">=23.0.0"

[tool.poetry]
name = "poetry-name"
version = "0.1.0"
description = "Poetry description"

[tool.poetry.dependencies]
python = "^3.9"

[tool.poetry.dev-dependencies]
pytest = "^7.0.0"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`,
			expectError:  false,
			expectedName: "comprehensive-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "pyproject_parse_test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Write pyproject.toml
			pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
			if err := os.WriteFile(pyprojectPath, []byte(tt.pyprojectToml), 0644); err != nil {
				t.Fatalf("Failed to write pyproject.toml: %v", err)
			}

			// Parse the configuration
			resolver := NewProjectResolver(tempDir)
			config, err := resolver.ParsePyprojectToml(pyprojectPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify project name
			if tt.expectedName != "" && config.Project.Name != tt.expectedName {
				t.Errorf("Expected project name '%s', got '%s'", tt.expectedName, config.Project.Name)
			}

			// Verify poetry name
			if tt.expectedPoetry != "" && config.Tool.Poetry.Name != tt.expectedPoetry {
				t.Errorf("Expected poetry name '%s', got '%s'", tt.expectedPoetry, config.Tool.Poetry.Name)
			}
		})
	}
}

func TestProjectResolver_ParsePyprojectToml_PermissiveHandling(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "pyproject_permissive_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test that we can handle files with extra fields that we don't explicitly define
	pyprojectContent := `[project]
name = "permissive-project"
version = "0.1.0"
# Extra fields that aren't in our struct
custom-field = "custom-value"
extra-list = ["item1", "item2"]

[project.urls]
Homepage = "https://example.com"
Documentation = "https://docs.example.com"

[tool.custom-tool]
setting = "value"

[tool.uv.sources]
valid-source = { path = "../local-package" }

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`

	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	resolver := NewProjectResolver(tempDir)
	config, err := resolver.ParsePyprojectToml(pyprojectPath)

	if err != nil {
		t.Fatalf("Should handle extra fields gracefully, got error: %v", err)
	}

	if config.Project.Name != "permissive-project" {
		t.Errorf("Expected project name 'permissive-project', got '%s'", config.Project.Name)
	}

	// Verify UV source was parsed correctly
	if source, exists := config.Tool.UV.Sources["valid-source"]; !exists {
		t.Error("Expected valid-source in UV sources")
	} else if source.Path != "../local-package" {
		t.Errorf("Expected path '../local-package', got '%s'", source.Path)
	}
}

func TestProjectResolver_ParsePyprojectToml_ErrorMessages(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "pyproject_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	resolver := NewProjectResolver(tempDir)

	// Test file not found
	nonexistentPath := filepath.Join(tempDir, "nonexistent.toml")
	_, err = resolver.ParsePyprojectToml(nonexistentPath)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "failed to read pyproject.toml") {
		t.Errorf("Expected 'failed to read pyproject.toml' in error, got: %v", err)
	}

	// Test invalid TOML with helpful error message
	invalidTomlPath := filepath.Join(tempDir, "invalid.toml")
	invalidContent := `[project]
name = "unclosed-string
version = "0.1.0"
`
	if err := os.WriteFile(invalidTomlPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid TOML: %v", err)
	}

	_, err = resolver.ParsePyprojectToml(invalidTomlPath)
	if err == nil {
		t.Error("Expected error for invalid TOML")
	}
	if !strings.Contains(err.Error(), "TOML parsing error") {
		t.Errorf("Expected 'TOML parsing error' in error, got: %v", err)
	}
}

func TestProjectResolver_ValidateUVSource(t *testing.T) {
	resolver := NewProjectResolver("/test")

	tests := []struct {
		name        string
		sourceName  string
		source      UVSource
		expectError bool
		errorMsg    string
	}{
		{
			name:       "valid path source",
			sourceName: "local-pkg",
			source:     UVSource{Path: "../local-package"},
		},
		{
			name:       "valid git source",
			sourceName: "git-pkg",
			source:     UVSource{Git: "https://github.com/example/repo.git"},
		},
		{
			name:       "valid git source with branch",
			sourceName: "git-branch",
			source:     UVSource{Git: "https://github.com/example/repo.git", Branch: "main"},
		},
		{
			name:       "valid git source with tag",
			sourceName: "git-tag",
			source:     UVSource{Git: "https://github.com/example/repo.git", Tag: "v1.0.0"},
		},
		{
			name:       "valid git source with rev",
			sourceName: "git-rev",
			source:     UVSource{Git: "https://github.com/example/repo.git", Rev: "abc123"},
		},
		{
			name:       "valid url source",
			sourceName: "url-pkg",
			source:     UVSource{URL: "https://example.com/package.tar.gz"},
		},
		{
			name:        "no source specified",
			sourceName:  "empty",
			source:      UVSource{},
			expectError: true,
			errorMsg:    "no source specified",
		},
		{
			name:       "multiple sources - path and git",
			sourceName: "multiple",
			source: UVSource{
				Path: "../local",
				Git:  "https://github.com/example/repo.git",
			},
			expectError: true,
			errorMsg:    "multiple source types",
		},
		{
			name:       "multiple sources - all three",
			sourceName: "all-three",
			source: UVSource{
				Path: "../local",
				Git:  "https://github.com/example/repo.git",
				URL:  "https://example.com/package.tar.gz",
			},
			expectError: true,
			errorMsg:    "multiple source types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolver.validateUVSource(tt.sourceName, tt.source)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestProjectResolver_ConfigurationStructureSimplification(t *testing.T) {
	// Test that the simplified configuration structure works with real-world examples

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "config_simplification_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test various real-world pyproject.toml configurations
	realWorldConfigs := map[string]string{
		"fastapi_project": `[project]
name = "fastapi-app"
version = "0.1.0"
description = "A FastAPI application"
dependencies = [
    "fastapi>=0.104.0",
    "uvicorn[standard]>=0.24.0",
    "pydantic>=2.5.0"
]
requires-python = ">=3.9"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`,
		"django_project": `[project]
name = "django-app"
version = "0.1.0"
description = "A Django application"
dependencies = [
    "Django>=4.2.0",
    "psycopg2-binary>=2.9.0",
    "redis>=5.0.0"
]

[tool.setuptools.packages.find]
where = ["src"]

[build-system]
requires = ["setuptools", "wheel"]
build-backend = "setuptools.build_meta"
`,
		"data_science_project": `[project]
name = "data-analysis"
version = "0.1.0"
description = "Data analysis project"
dependencies = [
    "pandas>=2.1.0",
    "numpy>=1.24.0",
    "matplotlib>=3.7.0",
    "jupyter>=1.0.0"
]

[tool.uv.sources]
custom-ml-lib = { git = "https://github.com/company/ml-lib.git", tag = "v2.1.0" }

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`,
		"poetry_legacy": `[tool.poetry]
name = "legacy-poetry-project"
version = "0.1.0"
description = "Legacy Poetry project"

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.31.0"
click = "^8.1.0"

[tool.poetry.dev-dependencies]
pytest = "^7.4.0"
black = "^23.9.0"
flake8 = "^6.1.0"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`,
	}

	for projectType, config := range realWorldConfigs {
		t.Run(projectType, func(t *testing.T) {
			// Write configuration
			pyprojectPath := filepath.Join(tempDir, fmt.Sprintf("pyproject_%s.toml", projectType))
			if err := os.WriteFile(pyprojectPath, []byte(config), 0644); err != nil {
				t.Fatalf("Failed to write pyproject.toml: %v", err)
			}

			// Parse configuration
			resolver := NewProjectResolver(tempDir)
			parsedConfig, err := resolver.ParsePyprojectToml(pyprojectPath)

			if err != nil {
				t.Fatalf("Failed to parse %s configuration: %v", projectType, err)
			}

			// Verify basic parsing worked
			hasName := parsedConfig.Project.Name != "" || parsedConfig.Tool.Poetry.Name != ""
			if !hasName {
				t.Errorf("Configuration should have a project name for %s", projectType)
			}

			// Verify dependencies were parsed (if present)
			if len(parsedConfig.Project.Dependencies) > 0 || len(parsedConfig.Tool.Poetry.Dependencies) > 0 {
				t.Logf("Successfully parsed dependencies for %s", projectType)
			}

			// Clean up for next iteration
			os.Remove(pyprojectPath)
		})
	}
}
