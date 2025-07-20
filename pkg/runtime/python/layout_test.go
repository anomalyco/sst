package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLayoutDetector(t *testing.T) {
	config := LayoutDetectorConfig{
		ProjectRoot:  "/test/project",
		CacheTimeout: 10 * time.Minute,
	}
	
	detector := NewLayoutDetector(config)
	
	if detector.projectRoot != config.ProjectRoot {
		t.Errorf("Expected projectRoot %s, got %s", config.ProjectRoot, detector.projectRoot)
	}
	
	if detector.cacheTimeout != config.CacheTimeout {
		t.Errorf("Expected cacheTimeout %v, got %v", config.CacheTimeout, detector.cacheTimeout)
	}
	
	if detector.cache == nil {
		t.Error("Expected cache to be initialized")
	}
}

func TestNewLayoutDetectorWithDefaults(t *testing.T) {
	config := LayoutDetectorConfig{
		ProjectRoot: "/test/project",
		// CacheTimeout not set, should use default
	}
	
	detector := NewLayoutDetector(config)
	
	expectedTimeout := 5 * time.Minute
	if detector.cacheTimeout != expectedTimeout {
		t.Errorf("Expected default cacheTimeout %v, got %v", expectedTimeout, detector.cacheTimeout)
	}
}

func TestLayoutInfo_Structure(t *testing.T) {
	layout := &LayoutInfo{
		Type:            LayoutTypeWorkspace,
		HandlerFile:     "/test/handler.py",
		WorkspaceDir:    "/test",
		PackageName:     "mypackage",
		PythonPath:      []string{"/test/src"},
		Dependencies:    []string{"/test/pyproject.toml"},
		ModulePath:      "mypackage.handler",
		SourceRoot:      "/test/src",
		HasSrcDirectory: true,
		DetectedAt:      time.Now(),
	}
	
	// Test that all fields are properly set
	if layout.Type != LayoutTypeWorkspace {
		t.Errorf("Expected Type %s, got %s", LayoutTypeWorkspace, layout.Type)
	}
	
	if layout.HandlerFile != "/test/handler.py" {
		t.Errorf("Expected HandlerFile /test/handler.py, got %s", layout.HandlerFile)
	}
	
	if layout.PackageName != "mypackage" {
		t.Errorf("Expected PackageName mypackage, got %s", layout.PackageName)
	}
	
	if !layout.HasSrcDirectory {
		t.Error("Expected HasSrcDirectory to be true")
	}
	
	if len(layout.PythonPath) != 1 || layout.PythonPath[0] != "/test/src" {
		t.Errorf("Expected PythonPath [/test/src], got %v", layout.PythonPath)
	}
	
	if len(layout.Dependencies) != 1 || layout.Dependencies[0] != "/test/pyproject.toml" {
		t.Errorf("Expected Dependencies [/test/pyproject.toml], got %v", layout.Dependencies)
	}
}

func TestLayoutTypes(t *testing.T) {
	expectedTypes := []LayoutType{
		LayoutTypeWorkspace,
		LayoutTypeFlat,
		LayoutTypeNested,
		LayoutTypeLegacy,
	}
	
	expectedValues := []string{
		"workspace",
		"flat",
		"nested",
		"legacy",
	}
	
	for i, layoutType := range expectedTypes {
		if string(layoutType) != expectedValues[i] {
			t.Errorf("Expected layout type %s, got %s", expectedValues[i], string(layoutType))
		}
	}
}

func TestLayoutDetector_CacheOperations(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot:  "/test",
		CacheTimeout: 1 * time.Second,
	})
	
	handler := "test/handler.py"
	layout := &LayoutInfo{
		Type:        LayoutTypeFlat,
		HandlerFile: "/test/handler.py",
		DetectedAt:  time.Now(),
	}
	
	// Test cache miss
	cached, exists := detector.GetCachedLayout(handler)
	if exists {
		t.Error("Expected cache miss, but got cache hit")
	}
	if cached != nil {
		t.Error("Expected nil cached layout")
	}
	
	// Add to cache manually
	detector.cache[handler] = layout
	
	// Test cache hit
	cached, exists = detector.GetCachedLayout(handler)
	if !exists {
		t.Error("Expected cache hit, but got cache miss")
	}
	if cached == nil {
		t.Error("Expected non-nil cached layout")
	}
	if cached.Type != LayoutTypeFlat {
		t.Errorf("Expected cached type %s, got %s", LayoutTypeFlat, cached.Type)
	}
	
	// Test cache expiration
	time.Sleep(2 * time.Second)
	cached, exists = detector.GetCachedLayout(handler)
	if exists {
		t.Error("Expected cache miss due to expiration, but got cache hit")
	}
}

func TestLayoutDetector_InvalidateCache(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	
	handler := "test/handler.py"
	layout := &LayoutInfo{
		Type:       LayoutTypeFlat,
		DetectedAt: time.Now(),
	}
	
	// Add to cache
	detector.cache[handler] = layout
	
	// Verify it's cached
	if _, exists := detector.GetCachedLayout(handler); !exists {
		t.Error("Expected layout to be cached")
	}
	
	// Invalidate cache
	detector.InvalidateCache(handler)
	
	// Verify it's no longer cached
	if _, exists := detector.GetCachedLayout(handler); exists {
		t.Error("Expected layout to be removed from cache")
	}
}

func TestLayoutDetector_ClearCache(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	
	// Add multiple entries to cache
	handlers := []string{"handler1.py", "handler2.py", "handler3.py"}
	for _, handler := range handlers {
		detector.cache[handler] = &LayoutInfo{
			Type:       LayoutTypeFlat,
			DetectedAt: time.Now(),
		}
	}
	
	// Verify all are cached
	for _, handler := range handlers {
		if _, exists := detector.GetCachedLayout(handler); !exists {
			t.Errorf("Expected handler %s to be cached", handler)
		}
	}
	
	// Clear cache
	detector.ClearCache()
	
	// Verify all are removed
	for _, handler := range handlers {
		if _, exists := detector.GetCachedLayout(handler); exists {
			t.Errorf("Expected handler %s to be removed from cache", handler)
		}
	}
}

// Test helper functions for file operations
func TestFindPythonFile_DirectPath(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "layout_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a Python file
	handlerPath := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte("# test handler"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Test finding the file with direct path
	found, err := detector.FindPythonFile("handler.py")
	if err != nil {
		t.Fatalf("Failed to find Python file: %v", err)
	}
	
	expectedPath, _ := filepath.Abs(handlerPath)
	if found != expectedPath {
		t.Errorf("Expected to find %s, got %s", expectedPath, found)
	}
}

func TestResolveWorkspace_WithPyprojectToml(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "workspace_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create pyproject.toml
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte("[project]\nname = \"test\""), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}
	
	// Create a handler file in a subdirectory
	subDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	
	handlerPath := filepath.Join(subDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte("# test handler"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Test resolving workspace
	workspace, err := detector.ResolveWorkspace(handlerPath)
	if err != nil {
		t.Fatalf("Failed to resolve workspace: %v", err)
	}
	
	if workspace != tempDir {
		t.Errorf("Expected workspace %s, got %s", tempDir, workspace)
	}
}

func TestResolveWorkspace_NoWorkspaceFound(t *testing.T) {
	// Create a temporary directory without pyproject.toml
	tempDir, err := os.MkdirTemp("", "no_workspace_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	handlerPath := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte("# test handler"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Test that workspace resolution fails
	_, err = detector.ResolveWorkspace(handlerPath)
	if err == nil {
		t.Error("Expected error when no pyproject.toml found, but got nil")
	}
}

// Tests for flexible Python file discovery

func TestFindPythonFile_FlexibleDiscovery(t *testing.T) {
	// Create a complex directory structure for testing
	tempDir, err := os.MkdirTemp("", "flexible_discovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create various directory structures
	structures := map[string]string{
		"handler.py":                    "# direct handler",
		"src/handler.py":               "# src handler",
		"app/handler.py":               "# app handler", 
		"functions/handler.py":         "# functions handler",
		"lambda/handler.py":            "# lambda handler",
		"nested/package/handler.py":    "# nested handler",
		"src/mypackage/handler.py":     "# package handler",
	}
	
	for path, content := range structures {
		fullPath := filepath.Join(tempDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	testCases := []struct {
		handler  string
		expected string
	}{
		{"handler", "handler.py"},                    // Should find direct match
		{"src/handler", "src/handler.py"},           // Should find in src
		{"app/handler", "app/handler.py"},           // Should find in app
		{"functions/handler", "functions/handler.py"}, // Should find in functions
		{"nested/package/handler", "nested/package/handler.py"}, // Should find nested
	}
	
	for _, tc := range testCases {
		t.Run(tc.handler, func(t *testing.T) {
			found, err := detector.FindPythonFile(tc.handler)
			if err != nil {
				t.Fatalf("Failed to find Python file for %s: %v", tc.handler, err)
			}
			
			expectedPath := filepath.Join(tempDir, tc.expected)
			expectedAbs, _ := filepath.Abs(expectedPath)
			
			if found != expectedAbs {
				t.Errorf("Expected to find %s, got %s", expectedAbs, found)
			}
		})
	}
}

func TestFindPythonFile_DynamicDiscovery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dynamic_discovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a handler in an unusual location
	unusualPath := filepath.Join(tempDir, "unusual", "location", "myhandler.py")
	if err := os.MkdirAll(filepath.Dir(unusualPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(unusualPath, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Try to find with a handler path that doesn't match the structure
	found, err := detector.FindPythonFile("some/other/path/myhandler")
	if err != nil {
		t.Fatalf("Dynamic discovery failed: %v", err)
	}
	
	expectedAbs, _ := filepath.Abs(unusualPath)
	if found != expectedAbs {
		t.Errorf("Expected dynamic discovery to find %s, got %s", expectedAbs, found)
	}
}

func TestIsValidPythonFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "valid_python_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	testCases := []struct {
		filename string
		content  string
		valid    bool
	}{
		{"handler.py", "def handler(): pass", true},
		{"module.py", "import os\nclass MyClass: pass", true},
		{"script.py", "#!/usr/bin/env python\nprint('hello')", true},
		{"empty.py", "", true}, // Empty Python files are valid
		{"notpython.txt", "some text", false},
		{"noext", "def handler(): pass", false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tc.filename)
			if err := os.WriteFile(filePath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			
			valid := detector.isValidPythonFile(filePath)
			if valid != tc.valid {
				t.Errorf("Expected isValidPythonFile(%s) = %v, got %v", tc.filename, tc.valid, valid)
			}
		})
	}
}

func TestShouldSkipDirectory(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	
	testCases := []struct {
		dirname string
		skip    bool
	}{
		{"src", false},
		{"app", false},
		{"mypackage", false},
		{".git", true},
		{"__pycache__", true},
		{"node_modules", true},
		{".venv", true},
		{"venv", true},
		{".pytest_cache", true},
		{".idea", true},
		{".vscode", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.dirname, func(t *testing.T) {
			skip := detector.shouldSkipDirectory(tc.dirname)
			if skip != tc.skip {
				t.Errorf("Expected shouldSkipDirectory(%s) = %v, got %v", tc.dirname, tc.skip, skip)
			}
		})
	}
}

func TestSelectBestMatch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "best_match_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Create multiple potential matches
	matches := []string{
		filepath.Join(tempDir, "test", "handler.py"),      // In test directory (should score lower)
		filepath.Join(tempDir, "src", "handler.py"),       // In src directory (should score higher)
		filepath.Join(tempDir, "deep", "nested", "path", "handler.py"), // Deep nesting (should score lower)
		filepath.Join(tempDir, "handler.py"),             // Root level (should score well)
	}
	
	// Create the files
	for _, match := range matches {
		if err := os.MkdirAll(filepath.Dir(match), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", match, err)
		}
		if err := os.WriteFile(match, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", match, err)
		}
	}
	
	// Test best match selection
	best, err := detector.selectBestMatch("handler", matches)
	if err != nil {
		t.Fatalf("Failed to select best match: %v", err)
	}
	
	// Should prefer src/handler.py or handler.py over test/handler.py or deeply nested paths
	expectedCandidates := []string{
		filepath.Join(tempDir, "src", "handler.py"),
		filepath.Join(tempDir, "handler.py"),
	}
	
	found := false
	for _, candidate := range expectedCandidates {
		if best == candidate {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Expected best match to be one of %v, got %s", expectedCandidates, best)
	}
}

func TestHandleSymlinks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "symlink_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Get absolute path for the temp directory to ensure consistency
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("Failed to get absolute temp dir: %v", err)
	}
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: absTempDir,
	})
	
	// Create a real file
	realFile := filepath.Join(absTempDir, "real_handler.py")
	if err := os.WriteFile(realFile, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create real file: %v", err)
	}
	
	// Create a symlink to the real file
	symlinkFile := filepath.Join(absTempDir, "symlink_handler.py")
	if err := os.Symlink(realFile, symlinkFile); err != nil {
		t.Skipf("Skipping symlink test (symlinks not supported): %v", err)
	}
	
	// Test handling regular file
	resolved, err := detector.HandleSymlinks(realFile)
	if err != nil {
		t.Fatalf("Failed to handle regular file: %v", err)
	}
	if resolved != realFile {
		t.Errorf("Expected regular file to return itself, got %s", resolved)
	}
	
	// Test handling symlink
	resolved, err = detector.HandleSymlinks(symlinkFile)
	if err != nil {
		t.Fatalf("Failed to handle symlink: %v", err)
	}
	
	// Both paths should resolve to the same canonical path
	expectedResolved, _ := filepath.EvalSymlinks(realFile)
	actualResolved, _ := filepath.EvalSymlinks(resolved)
	
	if actualResolved != expectedResolved {
		t.Errorf("Expected symlink to resolve to %s, got %s", expectedResolved, actualResolved)
	}
}

func TestGenerateCandidatePaths(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test/project",
	})
	
	candidates := detector.generateCandidatePaths("mypackage/handler.py", "mypackage", "handler")
	
	// Should include various common patterns
	expectedPatterns := []string{
		"/test/project/mypackage/handler.py",
		"/test/project/src/mypackage/handler.py",
		"/test/project/app/mypackage/handler.py",
		"/test/project/functions/mypackage/handler.py",
	}
	
	for _, expected := range expectedPatterns {
		found := false
		for _, candidate := range candidates {
			if candidate == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected candidate path %s not found in generated candidates", expected)
		}
	}
	
	if len(candidates) == 0 {
		t.Error("Expected at least some candidate paths to be generated")
	}
}

func TestContainsPythonContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "python_content_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	testCases := []struct {
		name     string
		content  string
		expected bool
	}{
		{"function definition", "def my_function():\n    pass", true},
		{"class definition", "class MyClass:\n    pass", true},
		{"import statement", "import os\nimport sys", true},
		{"from import", "from typing import Dict", true},
		{"shebang", "#!/usr/bin/env python\nprint('hello')", true},
		{"coding declaration", "# -*- coding: utf-8 -*-\nprint('hello')", true},
		{"empty file", "", true}, // Empty files are considered valid Python
		{"plain text", "This is just plain text with no Python", true}, // Assume valid if .py extension
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test.py")
			if err := os.WriteFile(testFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			
			result := detector.containsPythonContent(testFile)
			if result != tc.expected {
				t.Errorf("Expected containsPythonContent for %q = %v, got %v", tc.content, tc.expected, result)
			}
			
			// Clean up for next test
			os.Remove(testFile)
		})
	}
}

// Tests for workspace detection algorithm

func TestParsePyprojectToml(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pyproject_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Create a sample pyproject.toml
	pyprojectContent := `[project]
name = "mypackage"
version = "0.1.0"
description = "A test package"
dependencies = ["requests", "click"]

[tool.uv.sources]
local-dep = { path = "../local-dep" }
git-dep = { git = "https://github.com/example/repo.git", branch = "main" }

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"`
	
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}
	
	// Test parsing
	pyproject, err := detector.parsePyprojectToml(pyprojectPath)
	if err != nil {
		t.Fatalf("Failed to parse pyproject.toml: %v", err)
	}
	
	// Verify parsed content
	if pyproject.Project.Name != "mypackage" {
		t.Errorf("Expected project name 'mypackage', got '%s'", pyproject.Project.Name)
	}
	
	if pyproject.Project.Version != "0.1.0" {
		t.Errorf("Expected version '0.1.0', got '%s'", pyproject.Project.Version)
	}
	
	if len(pyproject.Project.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(pyproject.Project.Dependencies))
	}
	
	if len(pyproject.Tool.UV.Sources) != 2 {
		t.Errorf("Expected 2 UV sources, got %d", len(pyproject.Tool.UV.Sources))
	}
	
	if pyproject.BuildSystem.BuildBackend != "hatchling.build" {
		t.Errorf("Expected build backend 'hatchling.build', got '%s'", pyproject.BuildSystem.BuildBackend)
	}
}

func TestFindAllWorkspaces(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspaces_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Create multiple workspace directories
	workspaceDirs := []string{
		".",
		"packages/pkg1",
		"packages/pkg2",
		"services/api",
	}
	
	for _, dir := range workspaceDirs {
		fullDir := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(fullDir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", fullDir, err)
		}
		
		pyprojectPath := filepath.Join(fullDir, "pyproject.toml")
		content := fmt.Sprintf(`[project]
name = "test-%s"
version = "0.1.0"`, strings.ReplaceAll(dir, "/", "-"))
		
		if err := os.WriteFile(pyprojectPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create pyproject.toml in %s: %v", fullDir, err)
		}
	}
	
	// Find all workspaces
	workspaces, err := detector.FindAllWorkspaces()
	if err != nil {
		t.Fatalf("Failed to find workspaces: %v", err)
	}
	
	if len(workspaces) != len(workspaceDirs) {
		t.Errorf("Expected %d workspaces, found %d", len(workspaceDirs), len(workspaces))
	}
	
	// Verify all expected workspaces are found
	expectedWorkspaces := make(map[string]bool)
	for _, dir := range workspaceDirs {
		expectedWorkspaces[filepath.Join(tempDir, dir)] = true
	}
	
	for _, workspace := range workspaces {
		if !expectedWorkspaces[workspace] {
			t.Errorf("Unexpected workspace found: %s", workspace)
		}
		delete(expectedWorkspaces, workspace)
	}
	
	if len(expectedWorkspaces) > 0 {
		t.Errorf("Missing expected workspaces: %v", expectedWorkspaces)
	}
}

func TestValidateWorkspaceConfiguration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "validate_workspace_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	testCases := []struct {
		name        string
		content     string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid workspace",
			content: `[project]
name = "mypackage"
version = "0.1.0"`,
			shouldError: false,
		},
		{
			name: "missing project name",
			content: `[project]
version = "0.1.0"`,
			shouldError: true,
			errorMsg:    "project name is required",
		},
		{
			name: "invalid TOML",
			content: `[project
name = "mypackage"`,
			shouldError: true,
			errorMsg:    "invalid pyproject.toml",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workspaceDir := filepath.Join(tempDir, tc.name)
			if err := os.MkdirAll(workspaceDir, 0755); err != nil {
				t.Fatalf("Failed to create workspace dir: %v", err)
			}
			
			pyprojectPath := filepath.Join(workspaceDir, "pyproject.toml")
			if err := os.WriteFile(pyprojectPath, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to create pyproject.toml: %v", err)
			}
			
			err := detector.ValidateWorkspaceConfiguration(workspaceDir)
			
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got nil", tc.errorMsg)
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestResolveNestedWorkspaces(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nested_workspaces_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Create nested workspace structure
	workspaces := []string{
		".",                    // Root workspace
		"packages/shared",     // Nested workspace
		"packages/api",        // Another nested workspace
		"packages/api/core",   // Deeply nested workspace
	}
	
	for _, workspace := range workspaces {
		workspaceDir := filepath.Join(tempDir, workspace)
		if err := os.MkdirAll(workspaceDir, 0755); err != nil {
			t.Fatalf("Failed to create workspace dir: %v", err)
		}
		
		pyprojectPath := filepath.Join(workspaceDir, "pyproject.toml")
		content := fmt.Sprintf(`[project]
name = "test-%s"
version = "0.1.0"`, strings.ReplaceAll(workspace, "/", "-"))
		
		if err := os.WriteFile(pyprojectPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create pyproject.toml: %v", err)
		}
	}
	
	// Test resolving for a handler in the deeply nested workspace
	handlerPath := filepath.Join(tempDir, "packages/api/core/handler.py")
	if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	
	candidates, err := detector.ResolveNestedWorkspaces(handlerPath)
	if err != nil {
		t.Fatalf("Failed to resolve nested workspaces: %v", err)
	}
	
	// Should find workspaces in order of specificity (deepest first)
	expectedOrder := []string{
		filepath.Join(tempDir, "packages/api/core"),
		filepath.Join(tempDir, "packages/api"),
		filepath.Join(tempDir),
	}
	
	if len(candidates) != len(expectedOrder) {
		t.Errorf("Expected %d candidates, got %d", len(expectedOrder), len(candidates))
	}
	
	for i, expected := range expectedOrder {
		if i >= len(candidates) || candidates[i] != expected {
			t.Errorf("Expected candidate %d to be %s, got %s", i, expected, candidates[i])
		}
	}
}

func TestCreateFallbackWorkspace(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fallback_workspace_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Create a handler without any workspace
	handlerPath := filepath.Join(tempDir, "standalone", "handler.py")
	if err := os.MkdirAll(filepath.Dir(handlerPath), 0755); err != nil {
		t.Fatalf("Failed to create handler directory: %v", err)
	}
	if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	
	// Create fallback workspace
	layout, err := detector.CreateFallbackWorkspace(handlerPath)
	if err != nil {
		t.Fatalf("Failed to create fallback workspace: %v", err)
	}
	
	// Verify fallback layout
	if layout.Type != LayoutTypeLegacy {
		t.Errorf("Expected layout type %s, got %s", LayoutTypeLegacy, layout.Type)
	}
	
	if layout.HandlerFile != handlerPath {
		t.Errorf("Expected handler file %s, got %s", handlerPath, layout.HandlerFile)
	}
	
	expectedWorkspaceDir := filepath.Dir(handlerPath)
	if layout.WorkspaceDir != expectedWorkspaceDir {
		t.Errorf("Expected workspace dir %s, got %s", expectedWorkspaceDir, layout.WorkspaceDir)
	}
	
	if layout.PackageName != "fallback" {
		t.Errorf("Expected package name 'fallback', got '%s'", layout.PackageName)
	}
	
	if len(layout.PythonPath) == 0 || layout.PythonPath[0] != expectedWorkspaceDir {
		t.Errorf("Expected Python path to include %s, got %v", expectedWorkspaceDir, layout.PythonPath)
	}
}

func TestGetWorkspaceDependencies(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workspace_deps_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Create various dependency files
	depFiles := []string{
		"pyproject.toml",
		"uv.lock",
		"requirements.txt",
		"poetry.lock",
		"Pipfile.lock",
	}
	
	for _, file := range depFiles {
		filePath := filepath.Join(tempDir, file)
		if err := os.WriteFile(filePath, []byte("# test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}
	
	// Get workspace dependencies
	deps, err := detector.GetWorkspaceDependencies(tempDir)
	if err != nil {
		t.Fatalf("Failed to get workspace dependencies: %v", err)
	}
	
	if len(deps) != len(depFiles) {
		t.Errorf("Expected %d dependencies, got %d", len(depFiles), len(deps))
	}
	
	// Verify all expected files are included
	expectedDeps := make(map[string]bool)
	for _, file := range depFiles {
		expectedDeps[filepath.Join(tempDir, file)] = true
	}
	
	for _, dep := range deps {
		if !expectedDeps[dep] {
			t.Errorf("Unexpected dependency: %s", dep)
		}
		delete(expectedDeps, dep)
	}
	
	if len(expectedDeps) > 0 {
		t.Errorf("Missing expected dependencies: %v", expectedDeps)
	}
}

func TestValidateUVSource(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "uv_source_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	
	// Create a local source directory
	localSourceDir := filepath.Join(tempDir, "local-package")
	if err := os.MkdirAll(localSourceDir, 0755); err != nil {
		t.Fatalf("Failed to create local source dir: %v", err)
	}
	
	testCases := []struct {
		name        string
		source      UVSource
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid local source",
			source:      UVSource{Path: "local-package"},
			shouldError: false,
		},
		{
			name:        "valid git source",
			source:      UVSource{Git: "https://github.com/example/repo.git"},
			shouldError: false,
		},
		{
			name:        "valid URL source",
			source:      UVSource{URL: "https://example.com/package.tar.gz"},
			shouldError: false,
		},
		{
			name:        "missing local source",
			source:      UVSource{Path: "nonexistent-package"},
			shouldError: true,
			errorMsg:    "does not exist",
		},
		{
			name:        "no source specified",
			source:      UVSource{},
			shouldError: true,
			errorMsg:    "no source specified",
		},
		{
			name:        "multiple sources",
			source:      UVSource{Path: "local-package", Git: "https://github.com/example/repo.git"},
			shouldError: true,
			errorMsg:    "multiple source types",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := detector.validateUVSource(tempDir, tc.name, tc.source)
			
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got nil", tc.errorMsg)
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// Tests for handler path resolution logic

func TestHandlerPathResolver_ParseHandlerSpec(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	resolver := NewHandlerPathResolver(detector)
	
	testCases := []struct {
		handler  string
		expected HandlerSpec
	}{
		{
			handler: "handler.py",
			expected: HandlerSpec{
				Module:   "handler",
				Function: "handler",
				FilePath: "handler.py",
			},
		},
		{
			handler: "src/mypackage/handler.py",
			expected: HandlerSpec{
				Module:   "src.mypackage.handler",
				Function: "handler",
				FilePath: "src/mypackage/handler.py",
			},
		},
		{
			handler: "mypackage.handler.lambda_handler",
			expected: HandlerSpec{
				Module:   "mypackage.handler",
				Function: "lambda_handler",
				FilePath: "mypackage.handler.lambda_handler",
			},
		},
		{
			handler: "handler.my_function",
			expected: HandlerSpec{
				Module:   "handler",
				Function: "my_function",
				FilePath: "handler.my_function",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.handler, func(t *testing.T) {
			spec, err := resolver.parseHandlerSpec(tc.handler)
			if err != nil {
				t.Fatalf("Failed to parse handler spec: %v", err)
			}
			
			if spec.Module != tc.expected.Module {
				t.Errorf("Expected module %s, got %s", tc.expected.Module, spec.Module)
			}
			
			if spec.Function != tc.expected.Function {
				t.Errorf("Expected function %s, got %s", tc.expected.Function, spec.Function)
			}
			
			if spec.FilePath != tc.expected.FilePath {
				t.Errorf("Expected file path %s, got %s", tc.expected.FilePath, spec.FilePath)
			}
		})
	}
}

func TestHandlerPathResolver_ResolveWorkspaceHandler(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	resolver := NewHandlerPathResolver(detector)
	
	layout := &LayoutInfo{
		Type:        LayoutTypeWorkspace,
		PackageName: "mypackage",
		SourceRoot:  "/test/src",
	}
	
	testCases := []struct {
		name     string
		spec     HandlerSpec
		expected string
	}{
		{
			name: "simple handler",
			spec: HandlerSpec{
				Module:   "handler",
				Function: "lambda_handler",
			},
			expected: "handler.lambda_handler",
		},
		{
			name: "handler with package prefix",
			spec: HandlerSpec{
				Module:   "mypackage.handler",
				Function: "lambda_handler",
			},
			expected: "handler.lambda_handler",
		},
		{
			name: "handler with src/package structure",
			spec: HandlerSpec{
				Module:   "src.mypackage.handler",
				Function: "lambda_handler",
			},
			expected: "handler.lambda_handler",
		},
		{
			name: "nested handler",
			spec: HandlerSpec{
				Module:   "mypackage.api.routes",
				Function: "handle_request",
			},
			expected: "api.routes.handle_request",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := resolver.resolveWorkspaceHandler(&tc.spec, layout)
			if err != nil {
				t.Fatalf("Failed to resolve workspace handler: %v", err)
			}
			
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestHandlerPathResolver_ResolveFlatHandler(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	resolver := NewHandlerPathResolver(detector)
	
	layout := &LayoutInfo{
		Type:       LayoutTypeFlat,
		SourceRoot: "/test",
	}
	
	spec := HandlerSpec{
		Module:   "handler",
		Function: "lambda_handler",
	}
	
	result, err := resolver.resolveFlatHandler(&spec, layout)
	if err != nil {
		t.Fatalf("Failed to resolve flat handler: %v", err)
	}
	
	expected := "handler.lambda_handler"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestHandlerPathResolver_ResolveNestedHandler(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	resolver := NewHandlerPathResolver(detector)
	
	layout := &LayoutInfo{
		Type:         LayoutTypeNested,
		WorkspaceDir: "/test",
		SourceRoot:   "/test/src",
	}
	
	spec := HandlerSpec{
		Module:   "src.api.handler",
		Function: "lambda_handler",
	}
	
	result, err := resolver.resolveNestedHandler(&spec, layout)
	if err != nil {
		t.Fatalf("Failed to resolve nested handler: %v", err)
	}
	
	expected := "api.handler.lambda_handler"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestHandlerPathResolver_ValidateHandlerPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "validate_handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	resolver := NewHandlerPathResolver(detector)
	
	// Create test files
	handlerDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(handlerDir, 0755); err != nil {
		t.Fatalf("Failed to create handler dir: %v", err)
	}
	
	handlerFile := filepath.Join(handlerDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def lambda_handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	layout := &LayoutInfo{
		SourceRoot: handlerDir,
	}
	
	testCases := []struct {
		name        string
		handlerPath string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid handler path",
			handlerPath: "handler.lambda_handler",
			shouldError: false,
		},
		{
			name:        "missing function",
			handlerPath: "handler",
			shouldError: true,
			errorMsg:    "must contain at least module and function",
		},
		{
			name:        "invalid function name",
			handlerPath: "handler.123invalid",
			shouldError: true,
			errorMsg:    "invalid function name",
		},
		{
			name:        "nonexistent module",
			handlerPath: "nonexistent.lambda_handler",
			shouldError: true,
			errorMsg:    "module file not found",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := resolver.ValidateHandlerPath(tc.handlerPath, layout)
			
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got nil", tc.errorMsg)
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestHandlerPathResolver_IsValidPythonIdentifier(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	resolver := NewHandlerPathResolver(detector)
	
	testCases := []struct {
		identifier string
		valid      bool
	}{
		{"handler", true},
		{"lambda_handler", true},
		{"_private", true},
		{"Handler123", true},
		{"123invalid", false},
		{"", false},
		{"with-dash", false},
		{"def", false},      // Python keyword
		{"class", false},    // Python keyword
		{"import", false},   // Python keyword
		{"my_func", true},
		{"MyClass", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.identifier, func(t *testing.T) {
			result := resolver.isValidPythonIdentifier(tc.identifier)
			if result != tc.valid {
				t.Errorf("Expected isValidPythonIdentifier(%s) = %v, got %v", tc.identifier, tc.valid, result)
			}
		})
	}
}

func TestHandlerPathResolver_GetImportPath(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	resolver := NewHandlerPathResolver(detector)
	
	layout := &LayoutInfo{
		Type:       LayoutTypeFlat,
		SourceRoot: "/test",
	}
	
	testCases := []struct {
		handler  string
		expected string
	}{
		{"handler.lambda_handler", "handler"},
		{"api.routes.handle_request", "api.routes"},
		{"mypackage.handler", "mypackage"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.handler, func(t *testing.T) {
			result, err := resolver.GetImportPath(tc.handler, layout)
			if err != nil {
				t.Fatalf("Failed to get import path: %v", err)
			}
			
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestHandlerPathResolver_GetFunctionName(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	resolver := NewHandlerPathResolver(detector)
	
	testCases := []struct {
		handler  string
		expected string
	}{
		{"handler.py", "handler"},
		{"handler.lambda_handler", "lambda_handler"},
		{"api.routes.handle_request", "handle_request"},
		{"mypackage.handler", "handler"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.handler, func(t *testing.T) {
			result, err := resolver.GetFunctionName(tc.handler)
			if err != nil {
				t.Fatalf("Failed to get function name: %v", err)
			}
			
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestHandlerPathResolver_NormalizeHandlerPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "normalize_handler_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	resolver := NewHandlerPathResolver(detector)
	
	// Create test file
	handlerFile := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerFile, []byte("def lambda_handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	layout := &LayoutInfo{
		Type:       LayoutTypeFlat,
		SourceRoot: tempDir,
	}
	
	result, err := resolver.NormalizeHandlerPath("handler.lambda_handler", layout)
	if err != nil {
		t.Fatalf("Failed to normalize handler path: %v", err)
	}
	
	expected := "handler.lambda_handler"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestHandlerPathResolver_SupportedHandlerFormats(t *testing.T) {
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: "/test",
	})
	resolver := NewHandlerPathResolver(detector)
	
	formats := resolver.SupportedHandlerFormats()
	
	if len(formats) == 0 {
		t.Error("Expected at least some supported handler formats")
	}
	
	// Check that all formats are strings
	for i, format := range formats {
		if format == "" {
			t.Errorf("Format %d is empty", i)
		}
	}
}

func TestHandlerPathResolver_IntegrationTest(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot: tempDir,
	})
	resolver := NewHandlerPathResolver(detector)
	
	// Create a complex project structure
	structures := map[string]string{
		"pyproject.toml": `[project]
name = "myproject"
version = "0.1.0"`,
		"src/myproject/handler.py":     "def lambda_handler(): pass",
		"src/myproject/api/routes.py":  "def handle_request(): pass",
		"functions/simple.py":          "def handler(): pass",
		"flat_handler.py":              "def my_function(): pass",
	}
	
	for path, content := range structures {
		fullPath := filepath.Join(tempDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}
	
	// Test different handler specifications
	testCases := []struct {
		name            string
		handler         string
		expectedModule  string
		expectedFunc    string
		shouldNormalize bool
	}{
		{
			name:            "workspace handler",
			handler:         "src/myproject/handler.lambda_handler",
			expectedModule:  "handler",
			expectedFunc:    "lambda_handler",
			shouldNormalize: true,
		},
		{
			name:            "nested API handler",
			handler:         "src/myproject/api/routes.handle_request",
			expectedModule:  "myproject/api/routes", // This will be adjusted by the resolver
			expectedFunc:    "handle_request",
			shouldNormalize: true,
		},
		{
			name:            "simple function handler",
			handler:         "functions/simple.handler",
			expectedModule:  "functions.simple",
			expectedFunc:    "handler",
			shouldNormalize: false, // This would need layout detection
		},
		{
			name:            "flat handler",
			handler:         "flat_handler.my_function",
			expectedModule:  "flat_handler",
			expectedFunc:    "my_function",
			shouldNormalize: false, // This would need layout detection
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test parsing
			spec, err := resolver.parseHandlerSpec(tc.handler)
			if err != nil {
				t.Fatalf("Failed to parse handler spec: %v", err)
			}
			
			// Test function name extraction
			funcName, err := resolver.GetFunctionName(tc.handler)
			if err != nil {
				t.Fatalf("Failed to get function name: %v", err)
			}
			
			if funcName != tc.expectedFunc {
				t.Errorf("Expected function name %s, got %s", tc.expectedFunc, funcName)
			}
			
			// For workspace handlers, test full resolution
			if tc.shouldNormalize {
				// Detect layout for the handler
				layout, err := detector.DetectLayout(tc.handler)
				if err != nil {
					t.Fatalf("Failed to detect layout: %v", err)
				}
				
				// Test import path extraction
				importPath, err := resolver.GetImportPath(tc.handler, layout)
				if err != nil {
					t.Fatalf("Failed to get import path: %v", err)
				}
				
				// Just check that we got a valid import path
				if importPath == "" {
					t.Errorf("Expected non-empty import path, got empty string")
				}
			}
			
			t.Logf("Handler: %s -> Module: %s, Function: %s", tc.handler, spec.Module, spec.Function)
		})
	}
}