package python

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOutputValidator_ValidateBuildOutputs(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) error
		expectError bool
		errorType   string
	}{
		{
			name: "valid build outputs with tar.gz files",
			setupFunc: func(dir string) error {
				// Create some tar.gz files
				files := []string{"package1-1.0.0.tar.gz", "package2-2.1.0.tar.gz"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("fake tar.gz content"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectError: false,
		},
		{
			name: "missing output directory",
			setupFunc: func(dir string) error {
				// Remove the directory
				return os.RemoveAll(dir)
			},
			expectError: true,
			errorType:   "output directory does not exist",
		},
		{
			name: "no tar.gz files found",
			setupFunc: func(dir string) error {
				// Create some other files but no tar.gz
				files := []string{"requirements.txt", "setup.py", "README.md"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("content"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectError: true,
			errorType:   "BuildValidationError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "build_validator_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			if err := tt.setupFunc(tempDir); err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create validator and test
			validator := NewBuildOutputValidator(tempDir)
			err = validator.ValidateBuildOutputs()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorType != "" {
					// Check if the error is of the expected type
					if tt.errorType == "BuildValidationError" {
						if _, ok := err.(*BuildValidationError); !ok {
							t.Errorf("Expected error type %s, but got: %T", tt.errorType, err)
						}
					} else if !strings.Contains(err.Error(), tt.errorType) {
						t.Errorf("Expected error type %s, but got: %v", tt.errorType, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestBuildOutputValidator_ListTarGzFiles(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(string) error
		expectedCount int
		expectError   bool
	}{
		{
			name: "multiple tar.gz files",
			setupFunc: func(dir string) error {
				files := []string{
					"package1-1.0.0.tar.gz",
					"package2-2.1.0.tar.gz",
					"core-0.5.0.tar.gz",
				}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("fake content"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "single tar.gz file",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "single-1.0.0.tar.gz"), []byte("content"), 0644)
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "no tar.gz files",
			setupFunc: func(dir string) error {
				// Create other files
				files := []string{"requirements.txt", "setup.py"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("content"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedCount: 0,
			expectError:   true,
		},
		{
			name: "tar.gz files in subdirectory",
			setupFunc: func(dir string) error {
				subDir := filepath.Join(dir, "dist")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(subDir, "package-1.0.0.tar.gz"), []byte("content"), 0644)
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "corrupted tar.gz file (directory with .tar.gz name)",
			setupFunc: func(dir string) error {
				// Create a directory with .tar.gz extension (invalid)
				return os.MkdirAll(filepath.Join(dir, "invalid.tar.gz"), 0755)
			},
			expectedCount: 0,
			expectError:   false, // Should filter out invalid files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "list_tarGz_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			if err := tt.setupFunc(tempDir); err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create validator and test
			validator := NewBuildOutputValidator(tempDir)
			files, err := validator.ListTarGzFiles()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(files) != tt.expectedCount {
					t.Errorf("Expected %d files, but got %d: %v", tt.expectedCount, len(files), files)
				}
			}
		})
	}
}

func TestBuildOutputValidator_ValidateExtractionResults(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) ([]string, error)
		expectError bool
		errorType   string
	}{
		{
			name: "successful extraction results",
			setupFunc: func(dir string) ([]string, error) {
				// Create tar.gz file and corresponding extracted directory
				tarGzFile := filepath.Join(dir, "package-1.0.0.tar.gz")
				if err := os.WriteFile(tarGzFile, []byte("content"), 0644); err != nil {
					return nil, err
				}

				// Create extracted directory
				extractedDir := filepath.Join(dir, "package-1.0.0")
				if err := os.MkdirAll(extractedDir, 0755); err != nil {
					return nil, err
				}

				// Add some content to extracted directory
				if err := os.WriteFile(filepath.Join(extractedDir, "setup.py"), []byte("setup content"), 0644); err != nil {
					return nil, err
				}

				return []string{tarGzFile}, nil
			},
			expectError: false,
		},
		{
			name: "missing extracted directory",
			setupFunc: func(dir string) ([]string, error) {
				// Create tar.gz file but no extracted directory
				tarGzFile := filepath.Join(dir, "package-1.0.0.tar.gz")
				if err := os.WriteFile(tarGzFile, []byte("content"), 0644); err != nil {
					return nil, err
				}
				return []string{tarGzFile}, nil
			},
			expectError: true,
			errorType:   "BuildValidationError",
		},
		{
			name: "extracted path is file not directory",
			setupFunc: func(dir string) ([]string, error) {
				// Create tar.gz file
				tarGzFile := filepath.Join(dir, "package-1.0.0.tar.gz")
				if err := os.WriteFile(tarGzFile, []byte("content"), 0644); err != nil {
					return nil, err
				}

				// Create file instead of directory with expected name
				extractedFile := filepath.Join(dir, "package-1.0.0")
				if err := os.WriteFile(extractedFile, []byte("not a directory"), 0644); err != nil {
					return nil, err
				}

				return []string{tarGzFile}, nil
			},
			expectError: true,
			errorType:   "BuildValidationError",
		},
		{
			name: "multiple successful extractions",
			setupFunc: func(dir string) ([]string, error) {
				var tarGzFiles []string

				packages := []string{"package1-1.0.0", "package2-2.1.0"}
				for _, pkg := range packages {
					// Create tar.gz file
					tarGzFile := filepath.Join(dir, pkg+".tar.gz")
					if err := os.WriteFile(tarGzFile, []byte("content"), 0644); err != nil {
						return nil, err
					}
					tarGzFiles = append(tarGzFiles, tarGzFile)

					// Create extracted directory
					extractedDir := filepath.Join(dir, pkg)
					if err := os.MkdirAll(extractedDir, 0755); err != nil {
						return nil, err
					}

					// Add content
					if err := os.WriteFile(filepath.Join(extractedDir, "module.py"), []byte("module content"), 0644); err != nil {
						return nil, err
					}
				}

				return tarGzFiles, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "validate_extraction_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			tarGzFiles, err := tt.setupFunc(tempDir)
			if err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create validator and test
			validator := NewBuildOutputValidator(tempDir)
			err = validator.ValidateExtractionResults(tarGzFiles)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorType != "" {
					// Check if the error is of the expected type
					if tt.errorType == "BuildValidationError" {
						if _, ok := err.(*BuildValidationError); !ok {
							t.Errorf("Expected error type %s, but got: %T", tt.errorType, err)
						}
					} else if !strings.Contains(err.Error(), tt.errorType) {
						t.Errorf("Expected error type %s, but got: %v", tt.errorType, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestBuildValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		error    *BuildValidationError
		expected []string // Substrings that should be in the error message
	}{
		{
			name: "complete error with all fields",
			error: &BuildValidationError{
				Stage:      "build",
				Command:    "uv build --all --sdist",
				Files:      []string{"file1.tar.gz", "file2.tar.gz"},
				Expected:   []string{"*.tar.gz files"},
				Actual:     []string{"requirements.txt", "setup.py"},
				Suggestion: "Check if uv build command completed successfully",
			},
			expected: []string{
				"Build validation failed at stage 'build'",
				"Command: uv build --all --sdist",
				"Files: file1.tar.gz, file2.tar.gz",
				"Expected: *.tar.gz files",
				"Actual: requirements.txt, setup.py",
				"Suggestion: Check if uv build command completed successfully",
			},
		},
		{
			name: "minimal error with only stage",
			error: &BuildValidationError{
				Stage: "extract",
			},
			expected: []string{
				"Build validation failed at stage 'extract'",
			},
		},
		{
			name: "error with command and suggestion only",
			error: &BuildValidationError{
				Stage:      "move",
				Command:    "tar -xzf package.tar.gz",
				Suggestion: "Check file permissions",
			},
			expected: []string{
				"Build validation failed at stage 'move'",
				"Command: tar -xzf package.tar.gz",
				"Suggestion: Check file permissions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorMsg := tt.error.Error()

			for _, expectedSubstring := range tt.expected {
				if !strings.Contains(errorMsg, expectedSubstring) {
					t.Errorf("Expected error message to contain '%s', but got: %s", expectedSubstring, errorMsg)
				}
			}
		})
	}
}

func TestBuildOutputValidator_findTarGzFilesAlternative(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(string) error
		expectedCount int
		expectError   bool
	}{
		{
			name: "find files via directory scan",
			setupFunc: func(dir string) error {
				files := []string{"package1-1.0.0.tar.gz", "package2-2.1.0.tar.gz"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("content"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "find files in subdirectories",
			setupFunc: func(dir string) error {
				// Create subdirectory with tar.gz files
				subDir := filepath.Join(dir, "dist")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}

				files := []string{"package1-1.0.0.tar.gz", "package2-2.1.0.tar.gz"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(subDir, file), []byte("content"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "no tar.gz files found anywhere",
			setupFunc: func(dir string) error {
				// Create various files but no tar.gz
				files := []string{"requirements.txt", "setup.py", "README.md"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("content"), 0644); err != nil {
						return err
					}
				}

				// Create subdirectory with non-tar.gz files
				subDir := filepath.Join(dir, "src")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(subDir, "module.py"), []byte("content"), 0644)
			},
			expectedCount: 0,
			expectError:   true,
		},
		{
			name: "mixed files and directories",
			setupFunc: func(dir string) error {
				// Create tar.gz in root
				if err := os.WriteFile(filepath.Join(dir, "root-1.0.0.tar.gz"), []byte("content"), 0644); err != nil {
					return err
				}

				// Create subdirectory with tar.gz
				subDir := filepath.Join(dir, "build")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(subDir, "sub-2.0.0.tar.gz"), []byte("content"), 0644); err != nil {
					return err
				}

				// Create other files
				return os.WriteFile(filepath.Join(dir, "other.txt"), []byte("content"), 0644)
			},
			expectedCount: 2,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "alternative_search_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			if err := tt.setupFunc(tempDir); err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create validator and test
			validator := NewBuildOutputValidator(tempDir)
			files, err := validator.findTarGzFilesAlternative()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(files) != tt.expectedCount {
					t.Errorf("Expected %d files, but got %d: %v", tt.expectedCount, len(files), files)
				}
			}
		})
	}
}
