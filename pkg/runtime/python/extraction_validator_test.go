package python

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractionValidator_ValidateExtraction(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) (string, error) // Returns tar.gz file path
		expectError bool
		errorType   string
	}{
		{
			name: "successful extraction validation",
			setupFunc: func(dir string) (string, error) {
				// Create tar.gz file
				tarGzFile := filepath.Join(dir, "package-1.0.0.tar.gz")
				if err := os.WriteFile(tarGzFile, []byte("content"), 0644); err != nil {
					return "", err
				}

				// Create corresponding extracted directory
				extractedDir := filepath.Join(dir, "package-1.0.0")
				if err := os.MkdirAll(extractedDir, 0755); err != nil {
					return "", err
				}

				// Add some files to the extracted directory
				files := []string{"setup.py", "module.py", "__init__.py"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(extractedDir, file), []byte("content"), 0644); err != nil {
						return "", err
					}
				}

				return tarGzFile, nil
			},
			expectError: false,
		},
		{
			name: "missing extracted directory",
			setupFunc: func(dir string) (string, error) {
				// Create tar.gz file but no extracted directory
				tarGzFile := filepath.Join(dir, "package-1.0.0.tar.gz")
				if err := os.WriteFile(tarGzFile, []byte("content"), 0644); err != nil {
					return "", err
				}
				return tarGzFile, nil
			},
			expectError: true,
			errorType:   "Build validation failed at stage 'extract'",
		},
		{
			name: "extracted path is file not directory",
			setupFunc: func(dir string) (string, error) {
				// Create tar.gz file
				tarGzFile := filepath.Join(dir, "package-1.0.0.tar.gz")
				if err := os.WriteFile(tarGzFile, []byte("content"), 0644); err != nil {
					return "", err
				}

				// Create file with expected directory name
				extractedFile := filepath.Join(dir, "package-1.0.0")
				if err := os.WriteFile(extractedFile, []byte("not a directory"), 0644); err != nil {
					return "", err
				}

				return tarGzFile, nil
			},
			expectError: true,
			errorType:   "Build validation failed at stage 'extract'",
		},
		{
			name: "empty extracted directory",
			setupFunc: func(dir string) (string, error) {
				// Create tar.gz file
				tarGzFile := filepath.Join(dir, "package-1.0.0.tar.gz")
				if err := os.WriteFile(tarGzFile, []byte("content"), 0644); err != nil {
					return "", err
				}

				// Create empty extracted directory
				extractedDir := filepath.Join(dir, "package-1.0.0")
				if err := os.MkdirAll(extractedDir, 0755); err != nil {
					return "", err
				}

				return tarGzFile, nil
			},
			expectError: true,
			errorType:   "Build validation failed at stage 'extract'",
		},
		{
			name: "complex package structure",
			setupFunc: func(dir string) (string, error) {
				// Create tar.gz file
				tarGzFile := filepath.Join(dir, "complex-package-2.1.0.tar.gz")
				if err := os.WriteFile(tarGzFile, []byte("content"), 0644); err != nil {
					return "", err
				}

				// Create extracted directory with complex structure
				extractedDir := filepath.Join(dir, "complex-package-2.1.0")
				if err := os.MkdirAll(extractedDir, 0755); err != nil {
					return "", err
				}

				// Create src subdirectory
				srcDir := filepath.Join(extractedDir, "src", "complex_package")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					return "", err
				}

				// Add files to various locations
				files := map[string]string{
					filepath.Join(extractedDir, "setup.py"):         "setup content",
					filepath.Join(extractedDir, "README.md"):        "readme content",
					filepath.Join(srcDir, "__init__.py"):            "init content",
					filepath.Join(srcDir, "module.py"):              "module content",
					filepath.Join(extractedDir, "tests", "test.py"): "test content",
				}

				for file, content := range files {
					if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
						return "", err
					}
					if err := os.WriteFile(file, []byte(content), 0644); err != nil {
						return "", err
					}
				}

				return tarGzFile, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "extraction_validator_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			tarGzFile, err := tt.setupFunc(tempDir)
			if err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create validator and test
			validator := NewExtractionValidator(tempDir)
			err = validator.ValidateExtraction(tarGzFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorType != "" && !strings.Contains(err.Error(), tt.errorType) {
					t.Errorf("Expected error type %s, but got: %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestExtractionValidator_GetExpectedDirectoryName(t *testing.T) {
	tests := []struct {
		name        string
		tarGzFile   string
		expectedDir string
	}{
		{
			name:        "simple package name",
			tarGzFile:   "/path/to/package-1.0.0.tar.gz",
			expectedDir: "package-1.0.0",
		},
		{
			name:        "complex package name with hyphens",
			tarGzFile:   "/path/to/my-complex-package-2.1.0.tar.gz",
			expectedDir: "my-complex-package-2.1.0",
		},
		{
			name:        "package with underscores",
			tarGzFile:   "/path/to/my_package_name-0.5.0.tar.gz",
			expectedDir: "my_package_name-0.5.0",
		},
		{
			name:        "package with pre-release version",
			tarGzFile:   "/path/to/package-1.0.0a1.tar.gz",
			expectedDir: "package-1.0.0a1",
		},
		{
			name:        "package with dev version",
			tarGzFile:   "/path/to/package-1.0.0.dev0.tar.gz",
			expectedDir: "package-1.0.0.dev0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewExtractionValidator("/tmp")
			dirName := validator.getExpectedDirectoryName(tt.tarGzFile)

			if dirName != tt.expectedDir {
				t.Errorf("Expected directory name %s, but got %s", tt.expectedDir, dirName)
			}
		})
	}
}

func TestExtractionValidator_ValidateDirectoryContents(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) error
		expectError bool
		errorType   string
	}{
		{
			name: "directory with files",
			setupFunc: func(dir string) error {
				files := []string{"setup.py", "module.py", "__init__.py"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("content"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectError: false,
		},
		{
			name: "empty directory",
			setupFunc: func(dir string) error {
				// Directory exists but is empty
				return nil
			},
			expectError: true,
			errorType:   "extracted directory is empty",
		},
		{
			name: "directory with subdirectories and files",
			setupFunc: func(dir string) error {
				// Create subdirectories
				subDirs := []string{"src", "tests", "docs"}
				for _, subDir := range subDirs {
					if err := os.MkdirAll(filepath.Join(dir, subDir), 0755); err != nil {
						return err
					}
				}

				// Create a package directory inside src
				packageDir := filepath.Join(dir, "src", "mypackage")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					return err
				}

				// Create files in various locations
				files := map[string]string{
					"setup.py":                  "setup content",
					"src/mypackage/__init__.py": "init content",
					"src/mypackage/module.py":   "module content",
					"tests/test.py":             "test content",
					"docs/README.md":            "readme content",
				}

				for file, content := range files {
					fullPath := filepath.Join(dir, file)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return err
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectError: false,
		},
		{
			name: "directory with only subdirectories (no files)",
			setupFunc: func(dir string) error {
				// Create subdirectories without src (to avoid src validation failure)
				subDirs := []string{"tests", "docs"}
				for _, subDir := range subDirs {
					if err := os.MkdirAll(filepath.Join(dir, subDir), 0755); err != nil {
						return err
					}
				}
				return nil
			},
			expectError: false, // This should not error anymore, just warn
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "validate_contents_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			if err := tt.setupFunc(tempDir); err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create validator and test
			validator := NewExtractionValidator(filepath.Dir(tempDir))
			err = validator.validateDirectoryContents(tempDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorType != "" && !strings.Contains(err.Error(), tt.errorType) {
					t.Errorf("Expected error type %s, but got: %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestExtractionValidator_Integration(t *testing.T) {
	// Integration test that validates the complete extraction validation process
	tempDir, err := os.MkdirTemp("", "extraction_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple tar.gz files and their extracted directories
	packages := []struct {
		name    string
		version string
		files   []string
	}{
		{
			name:    "functions",
			version: "1.0.0",
			files:   []string{"__init__.py", "handler.py", "utils.py"},
		},
		{
			name:    "core",
			version: "2.1.0",
			files:   []string{"__init__.py", "models.py", "services.py"},
		},
	}

	var tarGzFiles []string

	for _, pkg := range packages {
		// Create tar.gz file
		tarGzName := pkg.name + "-" + pkg.version + ".tar.gz"
		tarGzFile := filepath.Join(tempDir, tarGzName)
		if err := os.WriteFile(tarGzFile, []byte("fake tar.gz content"), 0644); err != nil {
			t.Fatalf("Failed to create tar.gz file: %v", err)
		}
		tarGzFiles = append(tarGzFiles, tarGzFile)

		// Create extracted directory
		extractedDir := filepath.Join(tempDir, pkg.name+"-"+pkg.version)
		if err := os.MkdirAll(extractedDir, 0755); err != nil {
			t.Fatalf("Failed to create extracted directory: %v", err)
		}

		// Create files in extracted directory
		for _, file := range pkg.files {
			filePath := filepath.Join(extractedDir, file)
			content := "# " + pkg.name + " " + file + "\nprint('Hello from " + pkg.name + "')\n"
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", file, err)
			}
		}
	}

	// Test validation of all extractions
	validator := NewExtractionValidator(tempDir)

	for _, tarGzFile := range tarGzFiles {
		if err := validator.ValidateExtraction(tarGzFile); err != nil {
			t.Errorf("Validation failed for %s: %v", tarGzFile, err)
		}
	}

	// Test with a missing extraction
	missingTarGz := filepath.Join(tempDir, "missing-1.0.0.tar.gz")
	if err := os.WriteFile(missingTarGz, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create missing tar.gz: %v", err)
	}

	err = validator.ValidateExtraction(missingTarGz)
	if err == nil {
		t.Errorf("Expected validation to fail for missing extraction, but it passed")
	}
}
