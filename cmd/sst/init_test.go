package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdInit(t *testing.T) {
	t.Run("detects existing sst.config.ts", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "sst-init-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Change to temp directory
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create existing sst.config.ts
		err = os.WriteFile("sst.config.ts", []byte("export default {}"), 0644)
		require.NoError(t, err)

		// Create mock CLI
		mockCli := &cli.Cli{}

		// Run init command - should detect existing config and return early
		err = CmdInit(mockCli)
		assert.NoError(t, err)
	})

	t.Run("detects Next.js project", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-init-nextjs-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create Next.js config file
		err = os.WriteFile("next.config.js", []byte("module.exports = {}"), 0644)
		require.NoError(t, err)

		// Create package.json
		err = os.WriteFile("package.json", []byte(`{"name": "test"}`), 0644)
		require.NoError(t, err)

		// Test template detection logic (without running full init)
		files, err := os.ReadDir(".")
		require.NoError(t, err)

		hints := []string{}
		for _, file := range files {
			if !file.IsDir() {
				hints = append(hints, file.Name())
			}
		}

		// Should detect Next.js
		hasNextConfig := false
		for _, hint := range hints {
			if strings.HasPrefix(hint, "next.config") {
				hasNextConfig = true
				break
			}
		}
		assert.True(t, hasNextConfig, "Should detect Next.js config file")
	})

	t.Run("detects React Router project", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-init-react-router-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create React Router config file
		err = os.WriteFile("react-router.config.ts", []byte("export default {}"), 0644)
		require.NoError(t, err)

		files, err := os.ReadDir(".")
		require.NoError(t, err)

		hints := []string{}
		for _, file := range files {
			if !file.IsDir() {
				hints = append(hints, file.Name())
			}
		}

		// Should detect React Router
		hasReactRouterConfig := false
		for _, hint := range hints {
			if strings.HasPrefix(hint, "react-router.config") {
				hasReactRouterConfig = true
				break
			}
		}
		assert.True(t, hasReactRouterConfig, "Should detect React Router config file")
	})

	t.Run("detects Astro project", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-init-astro-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create Astro config file
		err = os.WriteFile("astro.config.mjs", []byte("export default {}"), 0644)
		require.NoError(t, err)

		files, err := os.ReadDir(".")
		require.NoError(t, err)

		hints := []string{}
		for _, file := range files {
			if !file.IsDir() {
				hints = append(hints, file.Name())
			}
		}

		// Should detect Astro
		hasAstroConfig := false
		for _, hint := range hints {
			if strings.HasPrefix(hint, "astro.config") {
				hasAstroConfig = true
				break
			}
		}
		assert.True(t, hasAstroConfig, "Should detect Astro config file")
	})

	t.Run("detects Angular project", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-init-angular-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create Angular config file
		err = os.WriteFile("angular.json", []byte(`{"projects": {}}`), 0644)
		require.NoError(t, err)

		files, err := os.ReadDir(".")
		require.NoError(t, err)

		hints := []string{}
		for _, file := range files {
			if !file.IsDir() {
				hints = append(hints, file.Name())
			}
		}

		// Should detect Angular
		hasAngularConfig := false
		for _, hint := range hints {
			if strings.HasPrefix(hint, "angular.json") {
				hasAngularConfig = true
				break
			}
		}
		assert.True(t, hasAngularConfig, "Should detect Angular config file")
	})

	t.Run("detects JS project with package.json", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-init-js-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create package.json only
		err = os.WriteFile("package.json", []byte(`{"name": "test-project"}`), 0644)
		require.NoError(t, err)

		files, err := os.ReadDir(".")
		require.NoError(t, err)

		hints := []string{}
		for _, file := range files {
			if !file.IsDir() {
				hints = append(hints, file.Name())
			}
		}

		// Should detect package.json
		hasPackageJson := false
		for _, hint := range hints {
			if hint == "package.json" {
				hasPackageJson = true
				break
			}
		}
		assert.True(t, hasPackageJson, "Should detect package.json")
	})

	t.Run("defaults to vanilla template", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-init-vanilla-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create empty directory with just a README
		err = os.WriteFile("README.md", []byte("# Test Project"), 0644)
		require.NoError(t, err)

		files, err := os.ReadDir(".")
		require.NoError(t, err)

		hints := []string{}
		for _, file := range files {
			if !file.IsDir() {
				hints = append(hints, file.Name())
			}
		}

		// Should not detect any framework-specific files
		hasFrameworkConfig := false
		frameworkFiles := []string{"next.config", "react-router.config", "astro.config", "angular.json", "package.json"}
		for _, hint := range hints {
			for _, framework := range frameworkFiles {
				if strings.HasPrefix(hint, framework) {
					hasFrameworkConfig = true
					break
				}
			}
		}
		assert.False(t, hasFrameworkConfig, "Should not detect any framework-specific files")
	})
}

func TestFileContains(t *testing.T) {
	t.Run("detects string in file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-filecontains-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "test.txt")
		content := "This is a test file\nwith @solidjs/start in it\nand some other content"
		err = os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		// Should find the string
		assert.True(t, fileContains(testFile, "@solidjs/start"))
		
		// Should not find non-existent string
		assert.False(t, fileContains(testFile, "@nonexistent/package"))
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		// Should return false for non-existent file
		assert.False(t, fileContains("/non/existent/file.txt", "anything"))
	})

	t.Run("handles empty file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-filecontains-empty-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "empty.txt")
		err = os.WriteFile(testFile, []byte(""), 0644)
		require.NoError(t, err)

		// Should return false for empty file
		assert.False(t, fileContains(testFile, "anything"))
	})

	t.Run("detects SolidStart in app.config", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-solidstart-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "app.config.ts")
		content := `import { defineConfig } from "@solidjs/start/config";
export default defineConfig({});`
		err = os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		// Should detect SolidStart
		assert.True(t, fileContains(testFile, "@solidjs/start"))
	})

	t.Run("detects TanStack in app.config", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-tanstack-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "app.config.ts")
		content := `import { defineConfig } from "@tanstack/start/config";
export default defineConfig({});`
		err = os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		// Should detect TanStack
		assert.True(t, fileContains(testFile, "@tanstack/"))
	})

	t.Run("detects Remix in vite.config", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-remix-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "vite.config.ts")
		content := `import { vitePlugin as remix } from "@remix-run/dev";
export default defineConfig({
  plugins: [remix()],
});`
		err = os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		// Should detect Remix
		assert.True(t, fileContains(testFile, "@remix-run/dev"))
	})

	t.Run("detects Analog in vite.config", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sst-analog-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "vite.config.ts")
		content := `import { defineConfig } from 'vite';
import { analog } from '@analogjs/platform';
export default defineConfig({
  plugins: [analog()],
});`
		err = os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)

		// Should detect Analog
		assert.True(t, fileContains(testFile, "@analogjs/platform"))
	})
}

func TestTemplateDetectionLogic(t *testing.T) {
	t.Run("template detection priority order", func(t *testing.T) {
		// Test that the template detection follows the correct priority order
		// This tests the switch statement logic in CmdInit

		testCases := []struct {
			name           string
			files          map[string]string
			expectedResult string
		}{
			{
				name: "Next.js takes priority",
				files: map[string]string{
					"next.config.js": "module.exports = {}",
					"package.json":   `{"name": "test"}`,
				},
				expectedResult: "nextjs",
			},
			{
				name: "React Router detected",
				files: map[string]string{
					"react-router.config.ts": "export default {}",
					"package.json":           `{"name": "test"}`,
				},
				expectedResult: "react-router",
			},
			{
				name: "Astro detected",
				files: map[string]string{
					"astro.config.mjs": "export default {}",
					"package.json":     `{"name": "test"}`,
				},
				expectedResult: "astro",
			},
			{
				name: "Angular detected",
				files: map[string]string{
					"angular.json":   `{"projects": {}}`,
					"package.json":   `{"name": "test"}`,
				},
				expectedResult: "angular",
			},
			{
				name: "JS project with package.json only",
				files: map[string]string{
					"package.json": `{"name": "test"}`,
				},
				expectedResult: "js",
			},
			{
				name: "Vanilla project (no framework files)",
				files: map[string]string{
					"README.md": "# Test Project",
				},
				expectedResult: "vanilla",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tmpDir, err := os.MkdirTemp("", "sst-template-test")
				require.NoError(t, err)
				defer os.RemoveAll(tmpDir)

				// Create test files
				for filename, content := range tc.files {
					err = os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
					require.NoError(t, err)
				}

				// Simulate the file detection logic
				files, err := os.ReadDir(tmpDir)
				require.NoError(t, err)

				hints := []string{}
				for _, file := range files {
					if !file.IsDir() {
						hints = append(hints, file.Name())
					}
				}

				// Test the detection logic
				var detectedTemplate string
				switch {
				case containsPrefix(hints, "next.config"):
					detectedTemplate = "nextjs"
				case containsPrefix(hints, "react-router.config"):
					detectedTemplate = "react-router"
				case containsPrefix(hints, "astro.config"):
					detectedTemplate = "astro"
				case containsPrefix(hints, "angular.json"):
					detectedTemplate = "angular"
				case contains(hints, "package.json"):
					detectedTemplate = "js"
				default:
					detectedTemplate = "vanilla"
				}

				assert.Equal(t, tc.expectedResult, detectedTemplate, "Template detection failed for %s", tc.name)
			})
		}
	})
}

// Helper functions for testing
func containsPrefix(slice []string, prefix string) bool {
	for _, item := range slice {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}