package python

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLayoutDetection_EdgeCases tests various edge cases for layout detection
func TestLayoutDetection_EdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "layout_edge_cases_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot:  tempDir,
		CacheTimeout: time.Minute,
	})

	t.Run("handler with dots in name", func(t *testing.T) {
		// Create handler with dots in filename
		handlerPath := filepath.Join(tempDir, "my.complex.handler.py")
		if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		layout, err := detector.DetectLayout("my.complex.handler")
		if err != nil {
			t.Fatalf("Failed to detect layout: %v", err)
		}

		if layout.Type != LayoutTypeFlat {
			t.Errorf("Expected flat layout, got %s", layout.Type)
		}
	})

	t.Run("handler with underscores and hyphens", func(t *testing.T) {
		// Create handler with mixed naming conventions
		handlerPath := filepath.Join(tempDir, "my_handler-v2.py")
		if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		layout, err := detector.DetectLayout("my_handler-v2")
		if err != nil {
			t.Fatalf("Failed to detect layout: %v", err)
		}

		if layout.HandlerFile == "" {
			t.Error("Expected handler file to be found")
		}
	})

	t.Run("deeply nested handler", func(t *testing.T) {
		// Create very deeply nested structure
		deepPath := filepath.Join(tempDir, "level1", "level2", "level3", "level4", "level5", "handler.py")
		if err := os.MkdirAll(filepath.Dir(deepPath), 0755); err != nil {
			t.Fatalf("Failed to create deep directory: %v", err)
		}
		if err := os.WriteFile(deepPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create deep handler: %v", err)
		}

		layout, err := detector.DetectLayout("level1/level2/level3/level4/level5/handler")
		if err != nil {
			t.Fatalf("Failed to detect deep layout: %v", err)
		}

		if layout.Type != LayoutTypeNested {
			t.Errorf("Expected nested layout, got %s", layout.Type)
		}
	})

	t.Run("handler in __pycache__ directory", func(t *testing.T) {
		// Create handler in __pycache__ (should be ignored)
		pycachePath := filepath.Join(tempDir, "__pycache__", "handler.py")
		if err := os.MkdirAll(filepath.Dir(pycachePath), 0755); err != nil {
			t.Fatalf("Failed to create __pycache__ directory: %v", err)
		}
		if err := os.WriteFile(pycachePath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create __pycache__ handler: %v", err)
		}

		// Also create a valid handler
		validPath := filepath.Join(tempDir, "valid_handler.py")
		if err := os.WriteFile(validPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create valid handler: %v", err)
		}

		layout, err := detector.DetectLayout("handler")
		if err != nil {
			t.Fatalf("Failed to detect layout: %v", err)
		}

		// Should find the valid handler, not the one in __pycache__
		if strings.Contains(layout.HandlerFile, "__pycache__") {
			t.Error("Should not find handler in __pycache__ directory")
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		// Create handlers with different cases
		handlers := []string{"Handler.py", "HANDLER.py", "handler.py"}
		for _, handler := range handlers {
			handlerPath := filepath.Join(tempDir, "case_test", handler)
			if err := os.MkdirAll(filepath.Dir(handlerPath), 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
			if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
				t.Fatalf("Failed to create handler %s: %v", handler, err)
			}
		}

		// Should find handler regardless of case (on case-insensitive filesystems)
		layout, err := detector.DetectLayout("case_test/handler")
		if err != nil {
			t.Fatalf("Failed to detect layout: %v", err)
		}

		if layout.HandlerFile == "" {
			t.Error("Expected to find handler file")
		}
	})
}

// TestLayoutDetection_ComplexProjectStructures tests complex real-world project structures
func TestLayoutDetection_ComplexProjectStructures(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "complex_structures_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot:  tempDir,
		CacheTimeout: time.Minute,
	})

	t.Run("monorepo with multiple services", func(t *testing.T) {
		// Create monorepo structure
		services := []string{"auth-service", "user-service", "payment-service"}
		for _, service := range services {
			serviceDir := filepath.Join(tempDir, "services", service)
			if err := os.MkdirAll(serviceDir, 0755); err != nil {
				t.Fatalf("Failed to create service dir: %v", err)
			}

			// Create pyproject.toml for each service
			pyprojectPath := filepath.Join(serviceDir, "pyproject.toml")
			content := `[project]
name = "` + service + `"
version = "0.1.0"
dependencies = ["fastapi", "uvicorn"]`
			if err := os.WriteFile(pyprojectPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create pyproject.toml: %v", err)
			}

			// Create handler
			handlerPath := filepath.Join(serviceDir, "src", service, "handler.py")
			if err := os.MkdirAll(filepath.Dir(handlerPath), 0755); err != nil {
				t.Fatalf("Failed to create handler dir: %v", err)
			}
			if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
				t.Fatalf("Failed to create handler: %v", err)
			}
		}

		// Test detecting layout for each service
		for _, service := range services {
			handlerPath := "services/" + service + "/src/" + service + "/handler"
			layout, err := detector.DetectLayout(handlerPath)
			if err != nil {
				t.Fatalf("Failed to detect layout for %s: %v", service, err)
			}

			if layout.Type != LayoutTypeWorkspace {
				t.Errorf("Expected workspace layout for %s, got %s", service, layout.Type)
			}

			if layout.PackageName != service {
				t.Errorf("Expected package name %s, got %s", service, layout.PackageName)
			}
		}
	})

	t.Run("django project structure", func(t *testing.T) {
		// Create Django-like structure
		djangoDir := filepath.Join(tempDir, "django_project")
		if err := os.MkdirAll(djangoDir, 0755); err != nil {
			t.Fatalf("Failed to create Django dir: %v", err)
		}

		// Create Django project files
		files := map[string]string{
			"manage.py":               "#!/usr/bin/env python",
			"myproject/__init__.py":   "",
			"myproject/settings.py":   "DEBUG = True",
			"myproject/urls.py":       "urlpatterns = []",
			"myproject/wsgi.py":       "application = None",
			"myapp/__init__.py":       "",
			"myapp/models.py":         "from django.db import models",
			"myapp/views.py":          "from django.http import HttpResponse",
			"myapp/lambda_handler.py": "def handler(event, context): pass",
			"requirements.txt":        "django>=4.0",
		}

		for filePath, content := range files {
			fullPath := filepath.Join(djangoDir, filePath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Fatalf("Failed to create dir for %s: %v", filePath, err)
			}
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", filePath, err)
			}
		}

		layout, err := detector.DetectLayout("django_project/myapp/lambda_handler")
		if err != nil {
			t.Fatalf("Failed to detect Django layout: %v", err)
		}

		if layout.Type != LayoutTypeFlat {
			t.Errorf("Expected flat layout for Django project, got %s", layout.Type)
		}
	})

	t.Run("poetry project structure", func(t *testing.T) {
		// Create Poetry project structure
		poetryDir := filepath.Join(tempDir, "poetry_project")
		if err := os.MkdirAll(poetryDir, 0755); err != nil {
			t.Fatalf("Failed to create Poetry dir: %v", err)
		}

		// Create pyproject.toml with Poetry configuration
		pyprojectContent := `[tool.poetry]
name = "my-poetry-project"
version = "0.1.0"
description = "A Poetry project"
authors = ["Author <author@example.com>"]

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.28.0"

[tool.poetry.dev-dependencies]
pytest = "^7.0.0"

[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"`

		pyprojectPath := filepath.Join(poetryDir, "pyproject.toml")
		if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
			t.Fatalf("Failed to create pyproject.toml: %v", err)
		}

		// Create poetry.lock
		poetryLockPath := filepath.Join(poetryDir, "poetry.lock")
		if err := os.WriteFile(poetryLockPath, []byte("# Poetry lock file"), 0644); err != nil {
			t.Fatalf("Failed to create poetry.lock: %v", err)
		}

		// Create source structure
		handlerPath := filepath.Join(poetryDir, "my_poetry_project", "handler.py")
		if err := os.MkdirAll(filepath.Dir(handlerPath), 0755); err != nil {
			t.Fatalf("Failed to create handler dir: %v", err)
		}
		if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		layout, err := detector.DetectLayout("poetry_project/my_poetry_project/handler")
		if err != nil {
			t.Fatalf("Failed to detect Poetry layout: %v", err)
		}

		if layout.Type != LayoutTypeWorkspace {
			t.Errorf("Expected workspace layout for Poetry project, got %s", layout.Type)
		}
	})

	t.Run("flask application structure", func(t *testing.T) {
		// Create Flask application structure
		flaskDir := filepath.Join(tempDir, "flask_app")
		if err := os.MkdirAll(flaskDir, 0755); err != nil {
			t.Fatalf("Failed to create Flask dir: %v", err)
		}

		// Create Flask project files
		files := map[string]string{
			"app.py":                 "from flask import Flask",
			"wsgi.py":                "from app import app",
			"lambda_handler.py":      "def handler(event, context): pass",
			"config.py":              "DEBUG = True",
			"requirements.txt":       "flask>=2.0",
			"templates/index.html":   "<html></html>",
			"static/style.css":       "body { margin: 0; }",
			"blueprints/__init__.py": "",
			"blueprints/api.py":      "from flask import Blueprint",
			"models/__init__.py":     "",
			"models/user.py":         "class User: pass",
		}

		for filePath, content := range files {
			fullPath := filepath.Join(flaskDir, filePath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Fatalf("Failed to create dir for %s: %v", filePath, err)
			}
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", filePath, err)
			}
		}

		layout, err := detector.DetectLayout("flask_app/lambda_handler")
		if err != nil {
			t.Fatalf("Failed to detect Flask layout: %v", err)
		}

		if layout.Type != LayoutTypeFlat {
			t.Errorf("Expected flat layout for Flask app, got %s", layout.Type)
		}
	})
}

// TestLayoutDetection_ErrorConditions tests various error conditions
func TestLayoutDetection_ErrorConditions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "error_conditions_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot:  tempDir,
		CacheTimeout: time.Minute,
	})

	t.Run("nonexistent handler", func(t *testing.T) {
		_, err := detector.DetectLayout("nonexistent/handler")
		if err == nil {
			t.Error("Expected error for nonexistent handler")
		}

		if !strings.Contains(err.Error(), "handler not found") {
			t.Errorf("Expected 'handler not found' error, got: %v", err)
		}
	})

	t.Run("empty handler path", func(t *testing.T) {
		_, err := detector.DetectLayout("")
		if err == nil {
			t.Error("Expected error for empty handler path")
		}
	})

	t.Run("handler path with invalid characters", func(t *testing.T) {
		_, err := detector.DetectLayout("handler\x00invalid")
		if err == nil {
			t.Error("Expected error for handler path with null character")
		}
	})

	t.Run("circular symlinks", func(t *testing.T) {
		// Create circular symlinks
		link1 := filepath.Join(tempDir, "link1")
		link2 := filepath.Join(tempDir, "link2")

		if err := os.Symlink(link2, link1); err != nil {
			t.Skipf("Skipping circular symlink test: %v", err)
		}
		if err := os.Symlink(link1, link2); err != nil {
			t.Skipf("Skipping circular symlink test: %v", err)
		}

		// This should not cause infinite loop
		_, err := detector.HandleSymlinks(link1)
		if err == nil {
			t.Error("Expected error for circular symlinks")
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		// Create a directory with restricted permissions
		restrictedDir := filepath.Join(tempDir, "restricted")
		if err := os.MkdirAll(restrictedDir, 0000); err != nil {
			t.Fatalf("Failed to create restricted directory: %v", err)
		}
		defer os.Chmod(restrictedDir, 0755) // Restore permissions for cleanup

		// Try to detect layout in restricted directory
		_, err := detector.DetectLayout("restricted/handler")
		if err == nil {
			t.Error("Expected error for permission denied")
		}
	})

	t.Run("corrupted pyproject.toml", func(t *testing.T) {
		// Create directory with corrupted pyproject.toml
		corruptedDir := filepath.Join(tempDir, "corrupted")
		if err := os.MkdirAll(corruptedDir, 0755); err != nil {
			t.Fatalf("Failed to create corrupted dir: %v", err)
		}

		// Create invalid TOML file
		pyprojectPath := filepath.Join(corruptedDir, "pyproject.toml")
		if err := os.WriteFile(pyprojectPath, []byte("[invalid toml content"), 0644); err != nil {
			t.Fatalf("Failed to create corrupted pyproject.toml: %v", err)
		}

		// Create handler
		handlerPath := filepath.Join(corruptedDir, "handler.py")
		if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		// Should handle corrupted pyproject.toml gracefully
		layout, err := detector.DetectLayout("corrupted/handler")
		if err != nil {
			t.Fatalf("Should handle corrupted pyproject.toml gracefully: %v", err)
		}

		// Should fall back to a simpler layout type
		if layout.Type == LayoutTypeWorkspace {
			t.Error("Should not detect workspace layout with corrupted pyproject.toml")
		}
	})
}

// TestLayoutDetection_InvalidProjectStructures tests handling of invalid project structures
func TestLayoutDetection_InvalidProjectStructures(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "invalid_structures_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot:  tempDir,
		CacheTimeout: time.Minute,
	})

	t.Run("handler without .py extension", func(t *testing.T) {
		// Create handler without .py extension
		handlerPath := filepath.Join(tempDir, "handler_no_ext")
		if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		_, err := detector.DetectLayout("handler_no_ext")
		if err == nil {
			t.Error("Expected error for handler without .py extension")
		}
	})

	t.Run("binary file as handler", func(t *testing.T) {
		// Create binary file
		binaryPath := filepath.Join(tempDir, "binary_handler.py")
		binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
		if err := os.WriteFile(binaryPath, binaryContent, 0644); err != nil {
			t.Fatalf("Failed to create binary file: %v", err)
		}

		// Should handle binary files gracefully
		layout, err := detector.DetectLayout("binary_handler")
		if err != nil {
			t.Fatalf("Should handle binary files gracefully: %v", err)
		}

		// Should still detect layout but may have warnings
		if layout.HandlerFile == "" {
			t.Error("Should still find handler file even if it's binary")
		}
	})

	t.Run("extremely long file paths", func(t *testing.T) {
		// Create very long directory path
		longPath := tempDir
		for i := 0; i < 50; i++ {
			longPath = filepath.Join(longPath, "very_long_directory_name_that_exceeds_normal_limits")
		}

		// Try to create the path (may fail on some systems)
		if err := os.MkdirAll(longPath, 0755); err != nil {
			t.Skipf("Skipping long path test (path too long): %v", err)
		}

		handlerPath := filepath.Join(longPath, "handler.py")
		if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Skipf("Skipping long path test (cannot create file): %v", err)
		}

		// Should handle long paths gracefully
		relPath, _ := filepath.Rel(tempDir, handlerPath)
		handlerName := strings.TrimSuffix(relPath, ".py")
		_, err := detector.DetectLayout(handlerName)
		if err != nil {
			t.Logf("Long path handling failed (expected on some systems): %v", err)
		}
	})

	t.Run("special characters in paths", func(t *testing.T) {
		// Create paths with special characters
		specialChars := []string{
			"handler with spaces.py",
			"handler-with-dashes.py",
			"handler_with_underscores.py",
			"handler.with.dots.py",
		}

		for _, filename := range specialChars {
			handlerPath := filepath.Join(tempDir, "special", filename)
			if err := os.MkdirAll(filepath.Dir(handlerPath), 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
			if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
				t.Fatalf("Failed to create handler %s: %v", filename, err)
			}

			// Test detection
			handlerName := strings.TrimSuffix(filename, ".py")
			layout, err := detector.DetectLayout("special/" + handlerName)
			if err != nil {
				t.Errorf("Failed to detect layout for %s: %v", filename, err)
				continue
			}

			if layout.HandlerFile == "" {
				t.Errorf("Handler file not found for %s", filename)
			}
		}
	})

	t.Run("mixed line endings", func(t *testing.T) {
		// Create handler with mixed line endings
		content := "def handler():\r\n    pass\n\r\n    return 'hello'\r"
		handlerPath := filepath.Join(tempDir, "mixed_endings.py")
		if err := os.WriteFile(handlerPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		layout, err := detector.DetectLayout("mixed_endings")
		if err != nil {
			t.Fatalf("Failed to detect layout with mixed line endings: %v", err)
		}

		if layout.HandlerFile == "" {
			t.Error("Should handle mixed line endings")
		}
	})
}

// TestLayoutDetection_ConcurrentAccess tests concurrent access to layout detection
func TestLayoutDetection_ConcurrentAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot:  tempDir,
		CacheTimeout: time.Minute,
	})

	// Create multiple handlers
	handlers := []string{"handler1", "handler2", "handler3", "handler4", "handler5"}
	for _, handler := range handlers {
		handlerPath := filepath.Join(tempDir, handler+".py")
		if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
			t.Fatalf("Failed to create handler %s: %v", handler, err)
		}
	}

	// Test concurrent detection
	results := make(chan error, len(handlers))
	for _, handler := range handlers {
		go func(h string) {
			for i := 0; i < 10; i++ { // Multiple iterations per handler
				_, err := detector.DetectLayout(h)
				if err != nil {
					results <- err
					return
				}
			}
			results <- nil
		}(handler)
	}

	// Wait for all goroutines to complete
	for i := 0; i < len(handlers); i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent detection failed: %v", err)
		}
	}
}

// TestLayoutDetection_CacheInvalidation tests cache invalidation scenarios
func TestLayoutDetection_CacheInvalidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_invalidation_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detector := NewLayoutDetector(LayoutDetectorConfig{
		ProjectRoot:  tempDir,
		CacheTimeout: 100 * time.Millisecond, // Short timeout for testing
	})

	// Create initial handler
	handlerPath := filepath.Join(tempDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte("def handler(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// First detection - should cache result
	layout1, err := detector.DetectLayout("handler")
	if err != nil {
		t.Fatalf("First detection failed: %v", err)
	}

	// Second detection - should use cache
	layout2, err := detector.DetectLayout("handler")
	if err != nil {
		t.Fatalf("Second detection failed: %v", err)
	}

	if layout1.DetectedAt != layout2.DetectedAt {
		t.Error("Expected cached result to have same detection time")
	}

	// Wait for cache to expire
	time.Sleep(200 * time.Millisecond)

	// Third detection - should re-detect
	layout3, err := detector.DetectLayout("handler")
	if err != nil {
		t.Fatalf("Third detection failed: %v", err)
	}

	if layout1.DetectedAt == layout3.DetectedAt {
		t.Error("Expected cache to expire and re-detect")
	}

	// Test manual cache invalidation
	detector.InvalidateCache("handler")
	layout4, err := detector.DetectLayout("handler")
	if err != nil {
		t.Fatalf("Fourth detection failed: %v", err)
	}

	if layout3.DetectedAt == layout4.DetectedAt {
		t.Error("Expected manual cache invalidation to force re-detection")
	}
}
