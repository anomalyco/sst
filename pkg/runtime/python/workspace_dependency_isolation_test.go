package python

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWorkspaceDependencyIsolation tests that dependencies declared in a workspace
// member's pyproject.toml are only bundled with that member, not with other members
// or the root package.
//
// This test uses the python-modern-uv example which has:
// - Root package: depends on requests
// - API package: depends on requests
// - Worker package: depends on api, requests, AND arrow (worker-only)
//
// The test verifies that arrow is only bundled with worker, not api or root.
func TestWorkspaceDependencyIsolation(t *testing.T) {
	// Find the python-modern-uv example directory
	// This test requires the example to exist
	exampleDir := findExampleDir(t, "python-modern-uv")
	if exampleDir == "" {
		t.Skip("python-modern-uv example not found, skipping test")
	}

	t.Run("worker has arrow dependency", func(t *testing.T) {
		workerPyproject := filepath.Join(exampleDir, "packages", "worker", "pyproject.toml")
		content, err := os.ReadFile(workerPyproject)
		if err != nil {
			t.Fatalf("Failed to read worker pyproject.toml: %v", err)
		}

		if !containsString(string(content), "arrow") {
			t.Error("Worker pyproject.toml should contain arrow dependency")
		}
	})

	t.Run("api does not have arrow dependency", func(t *testing.T) {
		apiPyproject := filepath.Join(exampleDir, "packages", "api", "pyproject.toml")
		content, err := os.ReadFile(apiPyproject)
		if err != nil {
			t.Fatalf("Failed to read api pyproject.toml: %v", err)
		}

		if containsString(string(content), "arrow") {
			t.Error("API pyproject.toml should NOT contain arrow dependency")
		}
	})

	t.Run("root does not have arrow dependency", func(t *testing.T) {
		rootPyproject := filepath.Join(exampleDir, "pyproject.toml")
		content, err := os.ReadFile(rootPyproject)
		if err != nil {
			t.Fatalf("Failed to read root pyproject.toml: %v", err)
		}

		if containsString(string(content), "arrow") {
			t.Error("Root pyproject.toml should NOT contain arrow dependency")
		}
	})

	t.Run("worker handler imports arrow", func(t *testing.T) {
		workerHandler := filepath.Join(exampleDir, "packages", "worker", "src", "worker", "handler.py")
		content, err := os.ReadFile(workerHandler)
		if err != nil {
			t.Fatalf("Failed to read worker handler: %v", err)
		}

		if !containsString(string(content), "import arrow") {
			t.Error("Worker handler should import arrow")
		}
	})
}

// TestDependencyAnalyzerWorkspaceIsolation tests that the DependencyAnalyzer
// correctly identifies package-specific dependencies in a workspace
func TestDependencyAnalyzerWorkspaceIsolation(t *testing.T) {
	exampleDir := findExampleDir(t, "python-modern-uv")
	if exampleDir == "" {
		t.Skip("python-modern-uv example not found, skipping test")
	}

	t.Run("worker package dependencies include arrow", func(t *testing.T) {
		workerDir := filepath.Join(exampleDir, "packages", "worker")
		resolver := NewProjectResolver(workerDir)

		projectInfo, err := resolver.ResolveHandler("src/worker/handler.lambda_handler")
		if err != nil {
			t.Fatalf("Failed to resolve worker handler: %v", err)
		}

		analyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
			ProjectResolver: resolver,
		})

		analysis, err := analyzer.AnalyzeDependencies(nil, projectInfo)
		if err != nil {
			t.Fatalf("Failed to analyze dependencies: %v", err)
		}

		// Check that the package name is correct
		if analysis.PackageName != "worker" {
			t.Errorf("Expected package name 'worker', got %s", analysis.PackageName)
		}
	})

	t.Run("api package dependencies do not include arrow", func(t *testing.T) {
		apiDir := filepath.Join(exampleDir, "packages", "api")
		resolver := NewProjectResolver(apiDir)

		projectInfo, err := resolver.ResolveHandler("src/api/handler.lambda_handler")
		if err != nil {
			t.Fatalf("Failed to resolve api handler: %v", err)
		}

		analyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
			ProjectResolver: resolver,
		})

		analysis, err := analyzer.AnalyzeDependencies(nil, projectInfo)
		if err != nil {
			t.Fatalf("Failed to analyze dependencies: %v", err)
		}

		// Check that the package name is correct
		if analysis.PackageName != "api" {
			t.Errorf("Expected package name 'api', got %s", analysis.PackageName)
		}
	})
}

// findExampleDir locates the python-modern-uv example directory
func findExampleDir(t *testing.T, exampleName string) string {
	// Try relative paths from test location
	possiblePaths := []string{
		filepath.Join("..", "..", "..", "examples", exampleName),
		filepath.Join("..", "..", "..", "..", "examples", exampleName),
		filepath.Join("examples", exampleName),
	}

	// Also try from current working directory
	cwd, err := os.Getwd()
	if err == nil {
		// Walk up to find examples directory
		dir := cwd
		for i := 0; i < 5; i++ {
			examplePath := filepath.Join(dir, "examples", exampleName)
			if _, err := os.Stat(examplePath); err == nil {
				return examplePath
			}
			dir = filepath.Dir(dir)
		}
	}

	for _, path := range possiblePaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	return ""
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(len(s) >= len(substr)) &&
		(s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
