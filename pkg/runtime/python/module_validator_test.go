package python

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleValidator_ValidateModulePlacement(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) (string, string, string, error) // Returns extractedDir, targetDir, moduleName
		expectError bool
		errorType   string
	}{
		{
			name: "successful module placement validation",
			setupFunc: func(dir string) (string, string, string, error) {
				moduleName := "functions"
				targetDir := filepath.Join(dir, moduleName)

				// Create target directory with proper module structure
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return "", "", "", err
				}

				// Create module files
				files := []string{"__init__.py", "handler.py", "utils.py"}
				for _, file := range files {
					content := "# " + file + "\nprint('Module content')\n"
					if err := os.WriteFile(filepath.Join(targetDir, file), []byte(content), 0644); err != nil {
						return "", "", "", err
					}
				}

				return "", targetDir, moduleName, nil
			},
			expectError: false,
		},
		{
			name: "missing target directory",
			setupFunc: func(dir string) (string, string, string, error) {
				moduleName := "functions"
				targetDir := filepath.Join(dir, moduleName)
				// Don't create the target directory
				return "", targetDir, moduleName, nil
			},
			expectError: true,
			errorType:   "target directory does not exist",
		},
		{
			name: "target path is file not directory",
			setupFunc: func(dir string) (string, string, string, error) {
				moduleName := "functions"
				targetDir := filepath.Join(dir, moduleName)

				// Create file instead of directory
				if err := os.WriteFile(targetDir, []byte("not a directory"), 0644); err != nil {
					return "", "", "", err
				}

				return "", targetDir, moduleName, nil
			},
			expectError: true,
			errorType:   "target path is not a directory",
		},
		{
			name: "empty target directory",
			setupFunc: func(dir string) (string, string, string, error) {
				moduleName := "functions"
				targetDir := filepath.Join(dir, moduleName)

				// Create empty directory
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return "", "", "", err
				}

				return "", targetDir, moduleName, nil
			},
			expectError: true,
			errorType:   "target directory is empty",
		},
		{
			name: "missing __init__.py file",
			setupFunc: func(dir string) (string, string, string, error) {
				moduleName := "functions"
				targetDir := filepath.Join(dir, moduleName)

				// Create target directory
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return "", "", "", err
				}

				// Create files but no __init__.py
				files := []string{"handler.py", "utils.py"}
				for _, file := range files {
					content := "# " + file + "\nprint('Module content')\n"
					if err := os.WriteFile(filepath.Join(targetDir, file), []byte(content), 0644); err != nil {
						return "", "", "", err
					}
				}

				return "", targetDir, moduleName, nil
			},
			expectError: true,
			errorType:   "__init__.py file is missing",
		},
		{
			name: "complex module structure with subdirectories",
			setupFunc: func(dir string) (string, string, string, error) {
				moduleName := "complex_module"
				targetDir := filepath.Join(dir, moduleName)

				// Create target directory
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return "", "", "", err
				}

				// Create complex structure
				structure := map[string]string{
					"__init__.py":          "# Main module init",
					"handler.py":           "# Main handler",
					"utils/__init__.py":    "# Utils package",
					"utils/helpers.py":     "# Helper functions",
					"models/__init__.py":   "# Models package",
					"models/user.py":       "# User model",
					"services/__init__.py": "# Services package",
					"services/auth.py":     "# Auth service",
				}

				for file, content := range structure {
					fullPath := filepath.Join(targetDir, file)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return "", "", "", err
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						return "", "", "", err
					}
				}

				return "", targetDir, moduleName, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "module_validator_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			extractedDir, targetDir, moduleName, err := tt.setupFunc(tempDir)
			if err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create validator and test
			validator := NewModuleValidator(tempDir)
			err = validator.ValidateModulePlacement(extractedDir, targetDir, moduleName)

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

func TestModuleValidator_ValidatePackageStructure(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) (string, error) // Returns module directory path
		expectError bool
		errorType   string
	}{
		{
			name: "valid package structure with __init__.py",
			setupFunc: func(dir string) (string, error) {
				moduleDir := filepath.Join(dir, "valid_package")
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					return "", err
				}

				files := []string{"__init__.py", "module.py", "handler.py"}
				for _, file := range files {
					content := "# " + file + "\nprint('Valid package')\n"
					if err := os.WriteFile(filepath.Join(moduleDir, file), []byte(content), 0644); err != nil {
						return "", err
					}
				}

				return moduleDir, nil
			},
			expectError: false,
		},
		{
			name: "missing __init__.py file",
			setupFunc: func(dir string) (string, error) {
				moduleDir := filepath.Join(dir, "invalid_package")
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					return "", err
				}

				// Create files but no __init__.py
				files := []string{"module.py", "handler.py"}
				for _, file := range files {
					content := "# " + file + "\nprint('Invalid package')\n"
					if err := os.WriteFile(filepath.Join(moduleDir, file), []byte(content), 0644); err != nil {
						return "", err
					}
				}

				return moduleDir, nil
			},
			expectError: true,
			errorType:   "__init__.py file is missing",
		},
		{
			name: "empty __init__.py file (should be valid)",
			setupFunc: func(dir string) (string, error) {
				moduleDir := filepath.Join(dir, "empty_init_package")
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					return "", err
				}

				// Create empty __init__.py
				if err := os.WriteFile(filepath.Join(moduleDir, "__init__.py"), []byte(""), 0644); err != nil {
					return "", err
				}

				// Create other files
				files := []string{"module.py", "handler.py"}
				for _, file := range files {
					content := "# " + file + "\nprint('Package with empty init')\n"
					if err := os.WriteFile(filepath.Join(moduleDir, file), []byte(content), 0644); err != nil {
						return "", err
					}
				}

				return moduleDir, nil
			},
			expectError: false,
		},
		{
			name: "nested package structure",
			setupFunc: func(dir string) (string, error) {
				moduleDir := filepath.Join(dir, "nested_package")
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					return "", err
				}

				// Create nested structure with __init__.py files
				structure := map[string]string{
					"__init__.py":             "# Main package",
					"module.py":               "# Main module",
					"subpackage/__init__.py":  "# Subpackage",
					"subpackage/submodule.py": "# Submodule",
					"utils/__init__.py":       "# Utils package",
					"utils/helpers.py":        "# Helper functions",
				}

				for file, content := range structure {
					fullPath := filepath.Join(moduleDir, file)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return "", err
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						return "", err
					}
				}

				return moduleDir, nil
			},
			expectError: false,
		},
		{
			name: "nested package with missing subpackage __init__.py",
			setupFunc: func(dir string) (string, error) {
				moduleDir := filepath.Join(dir, "broken_nested_package")
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					return "", err
				}

				// Create structure with missing subpackage __init__.py
				structure := map[string]string{
					"__init__.py":             "# Main package",
					"module.py":               "# Main module",
					"subpackage/submodule.py": "# Submodule (missing __init__.py)",
					"utils/__init__.py":       "# Utils package",
					"utils/helpers.py":        "# Helper functions",
				}

				for file, content := range structure {
					fullPath := filepath.Join(moduleDir, file)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return "", err
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						return "", err
					}
				}

				return moduleDir, nil
			},
			expectError: true,
			errorType:   "subpackage missing __init__.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "package_structure_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			moduleDir, err := tt.setupFunc(tempDir)
			if err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create validator and test
			validator := NewModuleValidator(tempDir)
			err = validator.ValidatePackageStructure(moduleDir)

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

func TestModuleValidator_EnsureInitPyFiles(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) (string, error) // Returns module directory path
		expectError bool
		checkFunc   func(string) error // Additional checks after operation
	}{
		{
			name: "create missing __init__.py in root",
			setupFunc: func(dir string) (string, error) {
				moduleDir := filepath.Join(dir, "module_without_init")
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					return "", err
				}

				// Create files but no __init__.py
				files := []string{"handler.py", "utils.py"}
				for _, file := range files {
					if err := os.WriteFile(filepath.Join(moduleDir, file), []byte("content"), 0644); err != nil {
						return "", err
					}
				}

				return moduleDir, nil
			},
			expectError: false,
			checkFunc: func(moduleDir string) error {
				// Check that __init__.py was created
				initPath := filepath.Join(moduleDir, "__init__.py")
				if _, err := os.Stat(initPath); os.IsNotExist(err) {
					return err
				}
				return nil
			},
		},
		{
			name: "preserve existing __init__.py",
			setupFunc: func(dir string) (string, error) {
				moduleDir := filepath.Join(dir, "module_with_init")
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					return "", err
				}

				// Create __init__.py with content
				initContent := "# Existing init file\n__version__ = '1.0.0'\n"
				if err := os.WriteFile(filepath.Join(moduleDir, "__init__.py"), []byte(initContent), 0644); err != nil {
					return "", err
				}

				// Create other files
				if err := os.WriteFile(filepath.Join(moduleDir, "handler.py"), []byte("content"), 0644); err != nil {
					return "", err
				}

				return moduleDir, nil
			},
			expectError: false,
			checkFunc: func(moduleDir string) error {
				// Check that existing content is preserved
				initPath := filepath.Join(moduleDir, "__init__.py")
				content, err := os.ReadFile(initPath)
				if err != nil {
					return err
				}
				if !strings.Contains(string(content), "__version__ = '1.0.0'") {
					return err
				}
				return nil
			},
		},
		{
			name: "create __init__.py in nested subdirectories",
			setupFunc: func(dir string) (string, error) {
				moduleDir := filepath.Join(dir, "nested_module")
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					return "", err
				}

				// Create nested structure without __init__.py files
				structure := []string{
					"handler.py",
					"utils/helpers.py",
					"models/user.py",
					"services/auth.py",
				}

				for _, file := range structure {
					fullPath := filepath.Join(moduleDir, file)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return "", err
					}
					if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
						return "", err
					}
				}

				return moduleDir, nil
			},
			expectError: false,
			checkFunc: func(moduleDir string) error {
				// Check that __init__.py files were created in all directories
				expectedInits := []string{
					"__init__.py",
					"utils/__init__.py",
					"models/__init__.py",
					"services/__init__.py",
				}

				for _, initFile := range expectedInits {
					initPath := filepath.Join(moduleDir, initFile)
					if _, err := os.Stat(initPath); os.IsNotExist(err) {
						return err
					}
				}
				return nil
			},
		},
		{
			name: "handle mixed existing and missing __init__.py files",
			setupFunc: func(dir string) (string, error) {
				moduleDir := filepath.Join(dir, "mixed_module")
				if err := os.MkdirAll(moduleDir, 0755); err != nil {
					return "", err
				}

				// Create structure with some __init__.py files existing
				structure := map[string]string{
					"__init__.py":       "# Root init exists",
					"handler.py":        "# Handler",
					"utils/__init__.py": "# Utils init exists",
					"utils/helpers.py":  "# Helpers",
					"models/user.py":    "# User model (no init)",
					"services/auth.py":  "# Auth service (no init)",
				}

				for file, content := range structure {
					fullPath := filepath.Join(moduleDir, file)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						return "", err
					}
					if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
						return "", err
					}
				}

				return moduleDir, nil
			},
			expectError: false,
			checkFunc: func(moduleDir string) error {
				// Check that missing __init__.py files were created
				expectedInits := []string{
					"__init__.py",          // Should exist (preserve)
					"utils/__init__.py",    // Should exist (preserve)
					"models/__init__.py",   // Should be created
					"services/__init__.py", // Should be created
				}

				for _, initFile := range expectedInits {
					initPath := filepath.Join(moduleDir, initFile)
					if _, err := os.Stat(initPath); os.IsNotExist(err) {
						return err
					}
				}

				// Check that existing content is preserved
				utilsInit := filepath.Join(moduleDir, "utils/__init__.py")
				content, err := os.ReadFile(utilsInit)
				if err != nil {
					return err
				}
				if !strings.Contains(string(content), "Utils init exists") {
					return err
				}

				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "ensure_init_test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			moduleDir, err := tt.setupFunc(tempDir)
			if err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			// Create validator and test
			validator := NewModuleValidator(tempDir)
			err = validator.EnsureInitPyFiles(moduleDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}

				// Run additional checks if provided
				if tt.checkFunc != nil {
					if err := tt.checkFunc(moduleDir); err != nil {
						t.Errorf("Additional check failed: %v", err)
					}
				}
			}
		})
	}
}

func TestModuleValidator_Integration(t *testing.T) {
	// Integration test that validates the complete module validation process
	tempDir, err := os.MkdirTemp("", "module_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a realistic module structure
	modules := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "functions",
			files: map[string]string{
				"__init__.py":        "# Functions package",
				"handler.py":         "def lambda_handler(event, context): pass",
				"utils.py":           "def helper_function(): pass",
				"models/__init__.py": "# Models subpackage",
				"models/user.py":     "class User: pass",
				"services/auth.py":   "def authenticate(): pass", // Missing __init__.py
			},
		},
		{
			name: "core",
			files: map[string]string{
				"database.py":       "def connect(): pass", // Missing __init__.py
				"config.py":         "CONFIG = {}",
				"utils/__init__.py": "# Core utils",
				"utils/helpers.py":  "def core_helper(): pass",
			},
		},
	}

	validator := NewModuleValidator(tempDir)

	for _, module := range modules {
		moduleDir := filepath.Join(tempDir, module.name)
		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			t.Fatalf("Failed to create module directory: %v", err)
		}

		// Create module files
		for file, content := range module.files {
			fullPath := filepath.Join(moduleDir, file)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Fatalf("Failed to create directory for %s: %v", file, err)
			}
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", file, err)
			}
		}

		// Ensure __init__.py files are created
		if err := validator.EnsureInitPyFiles(moduleDir); err != nil {
			t.Errorf("Failed to ensure __init__.py files for %s: %v", module.name, err)
		}

		// Validate package structure
		if err := validator.ValidatePackageStructure(moduleDir); err != nil {
			t.Errorf("Package structure validation failed for %s: %v", module.name, err)
		}

		// Validate module placement
		if err := validator.ValidateModulePlacement("", moduleDir, module.name); err != nil {
			t.Errorf("Module placement validation failed for %s: %v", module.name, err)
		}
	}

	// Verify that missing __init__.py files were created
	expectedInits := []string{
		"functions/__init__.py",          // Already existed
		"functions/models/__init__.py",   // Already existed
		"functions/services/__init__.py", // Should be created
		"core/__init__.py",               // Should be created
		"core/utils/__init__.py",         // Already existed
	}

	for _, initFile := range expectedInits {
		initPath := filepath.Join(tempDir, initFile)
		if _, err := os.Stat(initPath); os.IsNotExist(err) {
			t.Errorf("Expected __init__.py file not found: %s", initFile)
		}
	}
}
