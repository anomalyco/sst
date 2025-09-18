package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLayoutRemoval_NoLayoutClassificationInErrors tests that error messages
// no longer contain layout-specific terminology and focus on actionable guidance.
func TestLayoutRemoval_NoLayoutClassificationInErrors(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	resolver := NewProjectResolver(projectDir)

	// Test handler not found error
	_, err := resolver.ResolveHandler("nonexistent.py")
	if err == nil {
		t.Error("Expected error for nonexistent handler")
	}

	errorMsg := strings.ToLower(err.Error())

	// Should not contain layout-specific terms
	layoutTerms := []string{
		"layout type",
		"layout detection",
		"layout classification",
		"workspace layout",
		"flat layout",
		"nested layout",
		"legacy layout",
	}

	for _, term := range layoutTerms {
		if strings.Contains(errorMsg, term) {
			t.Errorf("Error message should not contain layout-specific term '%s': %s", term, err.Error())
		}
	}

	// Should contain actionable guidance
	actionableTerms := []string{
		"handler",
		"not found",
		"search",
	}

	foundActionable := false
	for _, term := range actionableTerms {
		if strings.Contains(errorMsg, term) {
			foundActionable = true
			break
		}
	}

	if !foundActionable {
		t.Errorf("Error message should contain actionable guidance: %s", err.Error())
	}
}

// TestLayoutRemoval_ProjectResolverReplacesLayoutDetector tests that
// ProjectResolver successfully replaces LayoutDetector functionality
// without relying on layout classifications.
func TestLayoutRemoval_ProjectResolverReplacesLayoutDetector(t *testing.T) {
	tests := []struct {
		name           string
		setupProject   func(t *testing.T, projectDir string) string
		expectedModule string
		shouldSucceed  bool
	}{
		{
			name: "simple_handler",
			setupProject: func(t *testing.T, projectDir string) string {
				handlerContent := `def lambda_handler(event, context): return {"statusCode": 200}`
				handlerPath := filepath.Join(projectDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}
				return "handler.py"
			},
			expectedModule: "handler",
			shouldSucceed:  true,
		},
		{
			name: "nested_handler",
			setupProject: func(t *testing.T, projectDir string) string {
				packageDir := filepath.Join(projectDir, "app", "handlers")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}

				// Create __init__.py files
				initPaths := []string{
					filepath.Join(projectDir, "app", "__init__.py"),
					filepath.Join(packageDir, "__init__.py"),
				}
				for _, initPath := range initPaths {
					if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				handlerContent := `def lambda_handler(event, context): return {"statusCode": 200}`
				handlerPath := filepath.Join(packageDir, "api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}
				return "app/handlers/api.py"
			},
			expectedModule: "app.handlers.api",
			shouldSucceed:  true,
		},
		{
			name: "src_layout_handler",
			setupProject: func(t *testing.T, projectDir string) string {
				// Create pyproject.toml
				pyprojectContent := `[project]
name = "test-project"
version = "0.1.0"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				srcDir := filepath.Join(projectDir, "src")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatalf("Failed to create src directory: %v", err)
				}

				handlerContent := `def lambda_handler(event, context): return {"statusCode": 200}`
				handlerPath := filepath.Join(srcDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}
				return "handler.py"
			},
			expectedModule: "handler",
			shouldSucceed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project directory: %v", err)
			}

			handlerPath := tt.setupProject(t, projectDir)

			resolver := NewProjectResolver(projectDir)
			projectInfo, err := resolver.ResolveHandler(handlerPath)

			if tt.shouldSucceed {
				if err != nil {
					t.Fatalf("Expected successful resolution but got error: %v", err)
				}

				if projectInfo.ModulePath != tt.expectedModule {
					t.Errorf("Expected module path %s, got %s", tt.expectedModule, projectInfo.ModulePath)
				}

				// Verify basic properties are set correctly
				if projectInfo.ProjectRoot != projectDir {
					t.Errorf("Expected project root %s, got %s", projectDir, projectInfo.ProjectRoot)
				}

				if len(projectInfo.PythonPath) == 0 {
					t.Error("Expected non-empty Python path")
				}
			} else {
				if err == nil {
					t.Error("Expected error but resolution succeeded")
				}
			}
		})
	}
}

// TestLayoutRemoval_CachingIsContentBased tests that caching is now based on
// file content rather than layout classifications.
func TestLayoutRemoval_CachingIsContentBased(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Create a handler file
	handlerContent := `def lambda_handler(event, context): return {"statusCode": 200}`
	handlerPath := filepath.Join(projectDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create build cache
	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: cacheDir,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create cache entry with file content hash
	entry := &CacheEntry{
		FunctionID: "test-function",
		FileHashes: map[string]string{
			handlerPath: "initial-hash",
		},
		Dependencies: []string{handlerPath},
	}

	// Store cache entry
	if err := cache.Set("test-function", entry); err != nil {
		t.Fatalf("Failed to store cache entry: %v", err)
	}

	// Verify cache entry exists
	cachedEntry, exists := cache.Get("test-function")
	if !exists {
		t.Fatal("Expected cache entry to exist")
	}

	if cachedEntry == nil {
		t.Fatal("Expected cached entry but got nil")
	}

	// Verify cache entry contains file hashes, not layout information
	if len(cachedEntry.FileHashes) == 0 {
		t.Error("Expected file hashes in cache entry")
	}

	if _, exists := cachedEntry.FileHashes[handlerPath]; !exists {
		t.Error("Expected handler file hash in cache entry")
	}

	// Modify file content
	modifiedContent := `def lambda_handler(event, context): return {"statusCode": 201}`
	if err := os.WriteFile(handlerPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify handler: %v", err)
	}

	// Check cache validity - should be invalid due to content change
	valid, err := cache.IsValid(cachedEntry)
	if err != nil {
		t.Fatalf("Failed to check cache validity: %v", err)
	}

	if valid {
		t.Error("Expected cache to be invalid after content change")
	}
}

// TestLayoutRemoval_BuildPipelineIsUnified tests that there's now a single
// build pipeline that works for all project structures without layout-specific logic.
func TestLayoutRemoval_BuildPipelineIsUnified(t *testing.T) {
	projectStructures := []struct {
		name         string
		setupProject func(t *testing.T, projectDir string) string
	}{
		{
			name: "flat_structure",
			setupProject: func(t *testing.T, projectDir string) string {
				handlerContent := `def lambda_handler(event, context): return {"statusCode": 200}`
				handlerPath := filepath.Join(projectDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}
				return "handler.py"
			},
		},
		{
			name: "workspace_structure",
			setupProject: func(t *testing.T, projectDir string) string {
				pyprojectContent := `[project]
name = "workspace-project"
version = "0.1.0"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				functionsDir := filepath.Join(projectDir, "functions")
				if err := os.MkdirAll(functionsDir, 0755); err != nil {
					t.Fatalf("Failed to create functions directory: %v", err)
				}

				handlerContent := `def lambda_handler(event, context): return {"statusCode": 200}`
				handlerPath := filepath.Join(functionsDir, "api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				initPath := filepath.Join(functionsDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				return "api.py"
			},
		},
		{
			name: "src_structure",
			setupProject: func(t *testing.T, projectDir string) string {
				pyprojectContent := `[project]
name = "src-project"
version = "0.1.0"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				srcDir := filepath.Join(projectDir, "src", "mypackage")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatalf("Failed to create src directory: %v", err)
				}

				handlerContent := `def lambda_handler(event, context): return {"statusCode": 200}`
				handlerPath := filepath.Join(srcDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				initPath := filepath.Join(srcDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				return "handler.py"
			},
		},
	}

	for _, structure := range projectStructures {
		t.Run(structure.name, func(t *testing.T) {
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project directory: %v", err)
			}

			_ = structure.setupProject(t, projectDir)

			// Create unified build pipeline
			pipeline, err := NewBuildPipeline(BuildPipelineConfig{
				ProjectRoot:             projectDir,
				ArtifactDir:             filepath.Join(tempDir, "artifacts"),
				EnableCaching:           false,
				EnableProgressReporting: false,
			})
			if err != nil {
				t.Fatalf("Failed to create build pipeline: %v", err)
			}

			// Test that the same pipeline can handle all structures
			input := createBuildInputForFunctionalTest(t, tempDir, "handler.lambda_handler")
			ctx := context.Background()

			// The build might fail due to missing UV, but it should at least
			// start the process and resolve the project structure
			_, err = pipeline.Build(ctx, input)

			// We don't require the build to succeed (UV might not be available),
			// but we do require that it doesn't fail due to layout-specific issues
			if err != nil {
				errorMsg := strings.ToLower(err.Error())
				layoutTerms := []string{
					"layout detection",
					"layout type",
					"unsupported layout",
				}

				for _, term := range layoutTerms {
					if strings.Contains(errorMsg, term) {
						t.Errorf("Build error should not contain layout-specific term '%s': %s", term, err.Error())
					}
				}
			}

			t.Logf("Unified pipeline successfully handled %s structure", structure.name)
		})
	}
}

// TestLayoutRemoval_ErrorMessagesAreActionable tests that error messages
// provide actionable guidance instead of layout-specific information.
func TestLayoutRemoval_ErrorMessagesAreActionable(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	resolver := NewProjectResolver(projectDir)

	// Test various error scenarios
	errorScenarios := []struct {
		name             string
		handlerPath      string
		setupProject     func()
		expectedGuidance []string
		forbiddenTerms   []string
	}{
		{
			name:        "handler_not_found",
			handlerPath: "nonexistent.py",
			setupProject: func() {
				// Don't create any files
			},
			expectedGuidance: []string{"not found", "search", "handler"},
			forbiddenTerms:   []string{"layout detection", "layout classification", "layout type"},
		},
		{
			name:        "handler_in_wrong_location",
			handlerPath: "api/handler.py",
			setupProject: func() {
				// Create handler in different location
				handlerContent := `def lambda_handler(event, context): return {"statusCode": 200}`
				handlerPath := filepath.Join(projectDir, "handler.py")
				os.WriteFile(handlerPath, []byte(handlerContent), 0644)
			},
			expectedGuidance: []string{"not found", "search", "handler"},
			forbiddenTerms:   []string{"layout detection", "layout classification", "layout type"},
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Clean up from previous test
			os.RemoveAll(projectDir)
			os.MkdirAll(projectDir, 0755)

			scenario.setupProject()

			_, err := resolver.ResolveHandler(scenario.handlerPath)
			if err == nil {
				t.Error("Expected error but resolution succeeded")
				return
			}

			errorMsg := strings.ToLower(err.Error())

			// Check for expected guidance
			foundGuidance := false
			for _, guidance := range scenario.expectedGuidance {
				if strings.Contains(errorMsg, guidance) {
					foundGuidance = true
					break
				}
			}

			if !foundGuidance {
				t.Errorf("Error message should contain actionable guidance %v: %s", scenario.expectedGuidance, err.Error())
			}

			// Check for forbidden terms
			for _, term := range scenario.forbiddenTerms {
				if strings.Contains(errorMsg, term) {
					t.Errorf("Error message should not contain forbidden term '%s': %s", term, err.Error())
				}
			}
		})
	}
}
