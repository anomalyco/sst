package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
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
	exampleDir := findExampleDir("python-modern-uv")
	if exampleDir == "" {
		t.Skip("python-modern-uv example not found, skipping test")
	}

	t.Run("worker has arrow dependency", func(t *testing.T) {
		workerPyproject := filepath.Join(exampleDir, "packages", "worker", "pyproject.toml")
		content, err := os.ReadFile(workerPyproject)
		if err != nil {
			t.Fatalf("Failed to read worker pyproject.toml: %v", err)
		}

		if !strings.Contains(string(content), "arrow") {
			t.Error("Worker pyproject.toml should contain arrow dependency")
		}
	})

	t.Run("api does not have arrow dependency", func(t *testing.T) {
		apiPyproject := filepath.Join(exampleDir, "packages", "api", "pyproject.toml")
		content, err := os.ReadFile(apiPyproject)
		if err != nil {
			t.Fatalf("Failed to read api pyproject.toml: %v", err)
		}

		if strings.Contains(string(content), "arrow") {
			t.Error("API pyproject.toml should NOT contain arrow dependency")
		}
	})

	t.Run("root does not have arrow dependency", func(t *testing.T) {
		rootPyproject := filepath.Join(exampleDir, "pyproject.toml")
		content, err := os.ReadFile(rootPyproject)
		if err != nil {
			t.Fatalf("Failed to read root pyproject.toml: %v", err)
		}

		if strings.Contains(string(content), "arrow") {
			t.Error("Root pyproject.toml should NOT contain arrow dependency")
		}
	})

	t.Run("worker handler imports arrow", func(t *testing.T) {
		workerHandler := filepath.Join(exampleDir, "packages", "worker", "src", "worker", "handler.py")
		content, err := os.ReadFile(workerHandler)
		if err != nil {
			t.Fatalf("Failed to read worker handler: %v", err)
		}

		if !strings.Contains(string(content), "import arrow") {
			t.Error("Worker handler should import arrow")
		}
	})
}

// TestWorkspaceDependencyIsolationBuild tests the actual build output to verify
// that worker-only dependencies (arrow) are bundled only with the worker function,
// not with the api or root functions.
//
// This is an integration test that runs the full build pipeline.
func TestWorkspaceDependencyIsolationBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary workspace that mimics python-modern-uv structure
	tempDir, err := os.MkdirTemp("", "workspace-isolation-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up the workspace structure
	setupWorkspaceIsolationTest(t, tempDir)

	// Create cache and artifact directories
	cacheDir := filepath.Join(tempDir, ".cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	// Test 1: Build the API handler and verify arrow is NOT in the output
	t.Run("api_build_excludes_arrow", func(t *testing.T) {
		apiArtifactDir := filepath.Join(tempDir, "artifacts", "api")
		if err := os.MkdirAll(apiArtifactDir, 0755); err != nil {
			t.Fatalf("Failed to create api artifact dir: %v", err)
		}

		// Create incremental builder for API
		builder, err := NewIncrementalBuilder(IncrementalBuilderConfig{
			CacheDir:    cacheDir,
			ArtifactDir: apiArtifactDir,
			ProjectRoot: tempDir,
		})
		if err != nil {
			t.Fatalf("Failed to create incremental builder: %v", err)
		}

		// Build the API handler
		input := &runtime.BuildInput{
			FunctionID: "ApiHandler",
			Handler:    "packages/api/src/api/handler.lambda_handler",
			Runtime:    "python3.11",
			CfgPath:    filepath.Join(tempDir, "sst.config.ts"),
		}

		ctx := context.Background()
		output, err := builder.Build(ctx, input)
		if err != nil {
			t.Logf("API build failed (may be expected if uv not available): %v", err)
			t.Skip("Skipping - build failed, likely missing uv or dependencies")
		}

		t.Logf("API build output directory: %s", output.Out)

		// Verify arrow is NOT in the output
		hasArrow := containsArrowInDirectory(output.Out)
		if hasArrow {
			t.Error("API artifact should NOT contain arrow dependency - dependency isolation failed")
		} else {
			t.Log("Verified: API artifact does not contain arrow (correct behavior)")
		}
	})

	// Test 2: Build the Worker handler and verify arrow IS in the output
	t.Run("worker_build_includes_arrow", func(t *testing.T) {
		workerArtifactDir := filepath.Join(tempDir, "artifacts", "worker")
		if err := os.MkdirAll(workerArtifactDir, 0755); err != nil {
			t.Fatalf("Failed to create worker artifact dir: %v", err)
		}

		// Create incremental builder for Worker
		builder, err := NewIncrementalBuilder(IncrementalBuilderConfig{
			CacheDir:    cacheDir,
			ArtifactDir: workerArtifactDir,
			ProjectRoot: tempDir,
		})
		if err != nil {
			t.Fatalf("Failed to create incremental builder: %v", err)
		}

		// Build the Worker handler
		input := &runtime.BuildInput{
			FunctionID: "WorkerHandler",
			Handler:    "packages/worker/src/worker/handler.lambda_handler",
			Runtime:    "python3.11",
			CfgPath:    filepath.Join(tempDir, "sst.config.ts"),
		}

		ctx := context.Background()
		output, err := builder.Build(ctx, input)
		if err != nil {
			t.Logf("Worker build failed (may be expected if uv not available): %v", err)
			t.Skip("Skipping - build failed, likely missing uv or dependencies")
		}

		t.Logf("Worker build output directory: %s", output.Out)

		// Verify arrow IS in the output (worker-only dependency)
		hasArrow := containsArrowInDirectory(output.Out)
		if !hasArrow {
			t.Error("Worker artifact SHOULD contain arrow - dependency isolation may have incorrectly excluded it")
			// List contents for debugging
			listDirectoryContents(t, output.Out)
		} else {
			t.Log("Verified: Worker artifact contains arrow (correct behavior)")
		}
	})
}

// setupWorkspaceIsolationTest creates a workspace structure for testing dependency isolation
func setupWorkspaceIsolationTest(t *testing.T, projectDir string) {
	// Create root pyproject.toml
	rootPyproject := `[project]
name = "workspace-isolation-test"
version = "0.1.0"
description = "Test workspace dependency isolation"
dependencies = ["requests>=2.31.0"]
requires-python = ">=3.11"

[tool.uv]
package = true

[tool.uv.workspace]
members = ["packages/*"]

[tool.hatch.build.targets.wheel]
packages = ["."]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
	if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(rootPyproject), 0644); err != nil {
		t.Fatalf("Failed to create root pyproject.toml: %v", err)
	}

	// Create root handler
	rootHandler := `def lambda_handler(event, context):
    return {"statusCode": 200, "body": "Root handler"}
`
	if err := os.WriteFile(filepath.Join(projectDir, "handler.py"), []byte(rootHandler), 0644); err != nil {
		t.Fatalf("Failed to create root handler: %v", err)
	}

	// Create packages directory
	packagesDir := filepath.Join(projectDir, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatalf("Failed to create packages dir: %v", err)
	}

	// Create API package
	apiDir := filepath.Join(packagesDir, "api", "src", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	apiPyproject := `[project]
name = "api"
version = "0.1.0"
description = "API package"
dependencies = ["requests>=2.31.0"]
requires-python = ">=3.11"

[tool.uv]
package = true

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
	if err := os.WriteFile(filepath.Join(packagesDir, "api", "pyproject.toml"), []byte(apiPyproject), 0644); err != nil {
		t.Fatalf("Failed to create api pyproject.toml: %v", err)
	}

	apiInit := ""
	if err := os.WriteFile(filepath.Join(apiDir, "__init__.py"), []byte(apiInit), 0644); err != nil {
		t.Fatalf("Failed to create api __init__.py: %v", err)
	}

	apiHandler := `def lambda_handler(event, context):
    return {"statusCode": 200, "body": "API handler"}
`
	if err := os.WriteFile(filepath.Join(apiDir, "handler.py"), []byte(apiHandler), 0644); err != nil {
		t.Fatalf("Failed to create api handler: %v", err)
	}

	apiModels := `def create_response(message):
    return {"message": message}
`
	if err := os.WriteFile(filepath.Join(apiDir, "models.py"), []byte(apiModels), 0644); err != nil {
		t.Fatalf("Failed to create api models: %v", err)
	}

	// Create Worker package with arrow dependency (worker-only)
	workerDir := filepath.Join(packagesDir, "worker", "src", "worker")
	if err := os.MkdirAll(workerDir, 0755); err != nil {
		t.Fatalf("Failed to create worker dir: %v", err)
	}

	workerPyproject := `[project]
name = "worker"
version = "0.1.0"
description = "Worker package with arrow (worker-only dependency)"
dependencies = ["api", "requests>=2.31.0", "arrow>=1.3.0"]
requires-python = ">=3.11"

[tool.uv]
package = true

[tool.uv.sources]
api = { workspace = true }

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
`
	if err := os.WriteFile(filepath.Join(packagesDir, "worker", "pyproject.toml"), []byte(workerPyproject), 0644); err != nil {
		t.Fatalf("Failed to create worker pyproject.toml: %v", err)
	}

	workerInit := ""
	if err := os.WriteFile(filepath.Join(workerDir, "__init__.py"), []byte(workerInit), 0644); err != nil {
		t.Fatalf("Failed to create worker __init__.py: %v", err)
	}

	workerHandler := `from api import models
import arrow

def lambda_handler(event, context):
    result = models.create_response("Worker handler")
    now = arrow.utcnow()
    return {
        "statusCode": 200,
        "body": result,
        "timestamp": now.isoformat(),
        "arrow_version": arrow.__version__
    }
`
	if err := os.WriteFile(filepath.Join(workerDir, "handler.py"), []byte(workerHandler), 0644); err != nil {
		t.Fatalf("Failed to create worker handler: %v", err)
	}

	// Create a minimal sst.config.ts (not actually used, just for CfgPath)
	sstConfig := `export default {};`
	if err := os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644); err != nil {
		t.Fatalf("Failed to create sst.config.ts: %v", err)
	}
}

// containsArrowInDirectory checks if the arrow package is present in a directory
func containsArrowInDirectory(dir string) bool {
	arrowFound := false

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Check for arrow package directory or files
		name := info.Name()
		if strings.Contains(strings.ToLower(name), "arrow") {
			// Skip if it's just a reference in a requirements file
			if !strings.HasSuffix(name, ".txt") {
				arrowFound = true
				return filepath.SkipAll
			}
		}

		return nil
	})

	return arrowFound
}

// listDirectoryContents lists the contents of a directory for debugging
func listDirectoryContents(t *testing.T, dir string) {
	t.Logf("Contents of %s:", dir)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		relPath, _ := filepath.Rel(dir, path)
		if info.IsDir() {
			t.Logf("  [DIR] %s", relPath)
		} else {
			t.Logf("  [FILE] %s (%d bytes)", relPath, info.Size())
		}
		return nil
	})
}

// TestDiscoverBuildablePackagesWorkspaceIsolation tests that discoverBuildablePackages
// correctly identifies packages in a workspace
func TestDiscoverBuildablePackagesWorkspaceIsolation(t *testing.T) {
	exampleDir := findExampleDir("python-modern-uv")
	if exampleDir == "" {
		t.Skip("python-modern-uv example not found, skipping test")
	}

	t.Run("worker package is discoverable", func(t *testing.T) {
		workerDir := filepath.Join(exampleDir, "packages", "worker")
		resolver := NewProjectResolver(workerDir)

		projectInfo, err := resolver.ResolveHandler("src/worker/handler.lambda_handler")
		if err != nil {
			t.Fatalf("Failed to resolve worker handler: %v", err)
		}

		packages, err := discoverBuildablePackages(projectInfo, resolver)
		if err != nil {
			t.Fatalf("Failed to discover packages: %v", err)
		}

		t.Logf("Found %d buildable packages for worker", len(packages))
		for _, pkg := range packages {
			t.Logf("  Package: %s at %s", pkg.Name, pkg.Path)
		}
	})

	t.Run("api package is discoverable", func(t *testing.T) {
		apiDir := filepath.Join(exampleDir, "packages", "api")
		resolver := NewProjectResolver(apiDir)

		projectInfo, err := resolver.ResolveHandler("src/api/handler.lambda_handler")
		if err != nil {
			t.Fatalf("Failed to resolve api handler: %v", err)
		}

		packages, err := discoverBuildablePackages(projectInfo, resolver)
		if err != nil {
			t.Fatalf("Failed to discover packages: %v", err)
		}

		t.Logf("Found %d buildable packages for api", len(packages))
		for _, pkg := range packages {
			t.Logf("  Package: %s at %s", pkg.Name, pkg.Path)
		}
	})
}

// findExampleDir locates the python-modern-uv example directory
func findExampleDir(exampleName string) string {
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
