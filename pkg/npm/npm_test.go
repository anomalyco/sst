package npm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackage_Structure(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected Package
	}{
		"basic package": {
			input: `{
				"name": "test-package",
				"version": "1.0.0"
			}`,
			expected: Package{
				Name:    "test-package",
				Version: "1.0.0",
				Pulumi:  nil,
			},
		},
		"package with pulumi": {
			input: `{
				"name": "sst-package",
				"version": "2.1.0",
				"pulumi": {
					"name": "pulumi-package",
					"version": "3.0.0"
				}
			}`,
			expected: Package{
				Name:    "sst-package",
				Version: "2.1.0",
				Pulumi: &struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "pulumi-package",
					Version: "3.0.0",
				},
			},
		},
		"empty pulumi object": {
			input: `{
				"name": "empty-pulumi",
				"version": "1.0.0",
				"pulumi": {}
			}`,
			expected: Package{
				Name:    "empty-pulumi",
				Version: "1.0.0",
				Pulumi: &struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "",
					Version: "",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var pkg Package
			err := json.Unmarshal([]byte(test.input), &pkg)
			require.NoError(t, err)
			assert.Equal(t, test.expected, pkg)
		})
	}
}

func TestGet_Success(t *testing.T) {
	tests := map[string]struct {
		name           string
		version        string
		responseBody   string
		expectedResult *Package
	}{
		"basic package": {
			name:    "test-package",
			version: "1.0.0",
			responseBody: `{
				"name": "test-package",
				"version": "1.0.0"
			}`,
			expectedResult: &Package{
				Name:    "test-package",
				Version: "1.0.0",
				Pulumi:  nil,
			},
		},
		"package with pulumi": {
			name:    "sst-package",
			version: "2.1.0",
			responseBody: `{
				"name": "sst-package",
				"version": "2.1.0",
				"pulumi": {
					"name": "pulumi-package",
					"version": "3.0.0"
				}
			}`,
			expectedResult: &Package{
				Name:    "sst-package",
				Version: "2.1.0",
				Pulumi: &struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}{
					Name:    "pulumi-package",
					Version: "3.0.0",
				},
			},
		},
		"scoped package": {
			name:    "@sst/core",
			version: "1.2.3",
			responseBody: `{
				"name": "@sst/core",
				"version": "1.2.3"
			}`,
			expectedResult: &Package{
				Name:    "@sst/core",
				Version: "1.2.3",
				Pulumi:  nil,
			},
		},
		"package with special characters": {
			name:    "test-package-with-dashes",
			version: "1.0.0-beta.1",
			responseBody: `{
				"name": "test-package-with-dashes",
				"version": "1.0.0-beta.1"
			}`,
			expectedResult: &Package{
				Name:    "test-package-with-dashes",
				Version: "1.0.0-beta.1",
				Pulumi:  nil,
			},
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/%s/%s", test.name, test.version)
				assert.Equal(t, expectedPath, r.URL.Path)
				assert.Equal(t, "GET", r.Method)
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(test.responseBody))
			}))
			defer server.Close()

			// Set custom registry URL
			originalRegistry := os.Getenv("NPM_REGISTRY")
			os.Setenv("NPM_REGISTRY", server.URL)
			defer func() {
				if originalRegistry == "" {
					os.Unsetenv("NPM_REGISTRY")
				} else {
					os.Setenv("NPM_REGISTRY", originalRegistry)
				}
			}()

			// Test Get function
			result, err := Get(test.name, test.version)
			require.NoError(t, err)
			assert.Equal(t, test.expectedResult, result)
		})
	}
}

func TestGet_DefaultRegistry(t *testing.T) {
	// Ensure NPM_REGISTRY is not set
	originalRegistry := os.Getenv("NPM_REGISTRY")
	os.Unsetenv("NPM_REGISTRY")
	defer func() {
		if originalRegistry != "" {
			os.Setenv("NPM_REGISTRY", originalRegistry)
		}
	}()

	// Create test server to simulate npmjs.org
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test-package/1.0.0", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test-package", "version": "1.0.0"}`))
	}))
	defer server.Close()

	// This test would normally hit the real npmjs.org, so we'll skip it
	// unless we're in a controlled environment
	t.Skip("Skipping test that would hit real npmjs.org registry")
}

func TestGet_HTTPErrors(t *testing.T) {
	tests := map[string]struct {
		statusCode   int
		responseBody string
		expectError  bool
		errorMessage string
	}{
		"404 not found": {
			statusCode:   http.StatusNotFound,
			responseBody: "Not Found",
			expectError:  true,
			errorMessage: "failed to fetch package: 404 Not Found",
		},
		"500 server error": {
			statusCode:   http.StatusInternalServerError,
			responseBody: "Internal Server Error",
			expectError:  true,
			errorMessage: "failed to fetch package: 500 Internal Server Error",
		},
		"403 forbidden": {
			statusCode:   http.StatusForbidden,
			responseBody: "Forbidden",
			expectError:  true,
			errorMessage: "failed to fetch package: 403 Forbidden",
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.statusCode)
				w.Write([]byte(test.responseBody))
			}))
			defer server.Close()

			// Set custom registry URL
			originalRegistry := os.Getenv("NPM_REGISTRY")
			os.Setenv("NPM_REGISTRY", server.URL)
			defer func() {
				if originalRegistry == "" {
					os.Unsetenv("NPM_REGISTRY")
				} else {
					os.Setenv("NPM_REGISTRY", originalRegistry)
				}
			}()

			result, err := Get("test-package", "1.0.0")
			if test.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.errorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test-package", "version": "1.0.0", invalid json`))
	}))
	defer server.Close()

	// Set custom registry URL
	originalRegistry := os.Getenv("NPM_REGISTRY")
	os.Setenv("NPM_REGISTRY", server.URL)
	defer func() {
		if originalRegistry == "" {
			os.Unsetenv("NPM_REGISTRY")
		} else {
			os.Setenv("NPM_REGISTRY", originalRegistry)
		}
	}()

	result, err := Get("test-package", "1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
	assert.Nil(t, result)
}

func TestGet_NetworkError(t *testing.T) {
	// Set invalid registry URL to simulate network error
	originalRegistry := os.Getenv("NPM_REGISTRY")
	os.Setenv("NPM_REGISTRY", "http://invalid-url-that-does-not-exist.local")
	defer func() {
		if originalRegistry == "" {
			os.Unsetenv("NPM_REGISTRY")
		} else {
			os.Setenv("NPM_REGISTRY", originalRegistry)
		}
	}()

	result, err := Get("test-package", "1.0.0")
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestDetectPackageManager_Success(t *testing.T) {
	tests := map[string]struct {
		lockFiles        []string
		expectedManager  string
		expectedLockPath string
	}{
		"npm package-lock.json": {
			lockFiles:        []string{"package-lock.json"},
			expectedManager:  "npm",
			expectedLockPath: "package-lock.json",
		},
		"yarn yarn.lock": {
			lockFiles:        []string{"yarn.lock"},
			expectedManager:  "yarn",
			expectedLockPath: "yarn.lock",
		},
		"pnpm pnpm-lock.yaml": {
			lockFiles:        []string{"pnpm-lock.yaml"},
			expectedManager:  "pnpm",
			expectedLockPath: "pnpm-lock.yaml",
		},
		"bun bun.lockb": {
			lockFiles:        []string{"bun.lockb"},
			expectedManager:  "bun",
			expectedLockPath: "bun.lockb",
		},
		"bun bun.lock": {
			lockFiles:        []string{"bun.lock"},
			expectedManager:  "bun",
			expectedLockPath: "bun.lock",
		},
		"priority npm over yarn": {
			lockFiles:        []string{"package-lock.json", "yarn.lock"},
			expectedManager:  "npm",
			expectedLockPath: "package-lock.json",
		},
		"priority yarn over pnpm": {
			lockFiles:        []string{"yarn.lock", "pnpm-lock.yaml"},
			expectedManager:  "yarn",
			expectedLockPath: "yarn.lock",
		},
		"priority pnpm over bun": {
			lockFiles:        []string{"pnpm-lock.yaml", "bun.lockb"},
			expectedManager:  "pnpm",
			expectedLockPath: "pnpm-lock.yaml",
		},
		"bun.lockb over bun.lock": {
			lockFiles:        []string{"bun.lockb", "bun.lock"},
			expectedManager:  "bun",
			expectedLockPath: "bun.lockb",
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "npm-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create lock files
			for _, lockFile := range test.lockFiles {
				lockPath := filepath.Join(tempDir, lockFile)
				err := os.WriteFile(lockPath, []byte("{}"), 0644)
				require.NoError(t, err)
			}

			// Test detection
			manager, lockPath := DetectPackageManager(tempDir)
			assert.Equal(t, test.expectedManager, manager)
			
			// Check that the lock path is correct
			expectedFullPath := filepath.Join(tempDir, test.expectedLockPath)
			assert.Equal(t, expectedFullPath, lockPath)
		})
	}
}

func TestDetectPackageManager_NoLockFiles(t *testing.T) {
	// Create temporary directory with no lock files
	tempDir, err := os.MkdirTemp("", "npm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	manager, lockPath := DetectPackageManager(tempDir)
	assert.Equal(t, "", manager)
	assert.Equal(t, "", lockPath)
}

func TestDetectPackageManager_NestedDirectories(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "npm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create nested directory
	nestedDir := filepath.Join(tempDir, "nested", "project")
	err = os.MkdirAll(nestedDir, 0755)
	require.NoError(t, err)

	// Create lock file in parent directory
	lockPath := filepath.Join(tempDir, "package-lock.json")
	err = os.WriteFile(lockPath, []byte("{}"), 0644)
	require.NoError(t, err)

	// Test detection from nested directory
	manager, foundLockPath := DetectPackageManager(nestedDir)
	assert.Equal(t, "npm", manager)
	assert.Equal(t, lockPath, foundLockPath)
}

func TestDetectPackageManager_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		setupFunc       func(string) error
		expectedManager string
		expectedPath    string
	}{
		"empty directory": {
			setupFunc: func(dir string) error {
				return nil // No setup needed
			},
			expectedManager: "",
			expectedPath:    "",
		},
		"directory with package.json but no lock": {
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "test"}`), 0644)
			},
			expectedManager: "",
			expectedPath:    "",
		},
		"directory with node_modules but no lock": {
			setupFunc: func(dir string) error {
				return os.MkdirAll(filepath.Join(dir, "node_modules"), 0755)
			},
			expectedManager: "",
			expectedPath:    "",
		},
		"multiple bun lock files": {
			setupFunc: func(dir string) error {
				// Create both bun lock files, bun.lockb should take priority
				err := os.WriteFile(filepath.Join(dir, "bun.lock"), []byte(""), 0644)
				if err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "bun.lockb"), []byte(""), 0644)
			},
			expectedManager: "bun",
			expectedPath:    "bun.lockb",
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "npm-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			err = test.setupFunc(tempDir)
			require.NoError(t, err)

			manager, lockPath := DetectPackageManager(tempDir)
			assert.Equal(t, test.expectedManager, manager)
			
			if test.expectedPath != "" {
				expectedFullPath := filepath.Join(tempDir, test.expectedPath)
				assert.Equal(t, expectedFullPath, lockPath)
			} else {
				assert.Equal(t, "", lockPath)
			}
		})
	}
}

func TestDetectPackageManager_InvalidDirectory(t *testing.T) {
	// Test with non-existent directory
	manager, lockPath := DetectPackageManager("/non/existent/directory")
	assert.Equal(t, "", manager)
	assert.Equal(t, "", lockPath)
}

func TestDetectPackageManager_PermissionDenied(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "npm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a subdirectory with restricted permissions
	restrictedDir := filepath.Join(tempDir, "restricted")
	err = os.MkdirAll(restrictedDir, 0000) // No permissions
	require.NoError(t, err)
	defer os.Chmod(restrictedDir, 0755) // Restore permissions for cleanup

	// Test detection should handle permission errors gracefully
	manager, lockPath := DetectPackageManager(restrictedDir)
	assert.Equal(t, "", manager)
	assert.Equal(t, "", lockPath)
}

// Benchmark tests
func BenchmarkGet(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test-package", "version": "1.0.0"}`))
	}))
	defer server.Close()

	originalRegistry := os.Getenv("NPM_REGISTRY")
	os.Setenv("NPM_REGISTRY", server.URL)
	defer func() {
		if originalRegistry == "" {
			os.Unsetenv("NPM_REGISTRY")
		} else {
			os.Setenv("NPM_REGISTRY", originalRegistry)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Get("test-package", "1.0.0")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDetectPackageManager(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "npm-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a package-lock.json file
	lockPath := filepath.Join(tempDir, "package-lock.json")
	err = os.WriteFile(lockPath, []byte("{}"), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectPackageManager(tempDir)
	}
}