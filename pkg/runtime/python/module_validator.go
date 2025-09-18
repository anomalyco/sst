package python

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ModuleValidator validates Python module placement and structure
type ModuleValidator struct {
	outputDir string
	logger    *slog.Logger
}

// NewModuleValidator creates a new module validator
func NewModuleValidator(outputDir string) *ModuleValidator {
	return &ModuleValidator{
		outputDir: outputDir,
		logger:    slog.Default(),
	}
}

// ValidateModulePlacement validates that modules are correctly placed in target directories
func (mv *ModuleValidator) ValidateModulePlacement(extractedDir, targetDir, moduleName string) error {
	mv.logger.Info("validating module placement",
		"extractedDir", extractedDir,
		"targetDir", targetDir,
		"moduleName", moduleName)

	// Check if target directory exists
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &BuildValidationError{
				Stage:      "move",
				Command:    fmt.Sprintf("move %s to %s", extractedDir, targetDir),
				Files:      []string{extractedDir},
				Expected:   []string{targetDir},
				Actual:     []string{"directory not found"},
				Suggestion: fmt.Sprintf("Module movement failed. Check if the move operation from %s to %s completed successfully.", extractedDir, targetDir),
			}
		}
		return fmt.Errorf("failed to stat target directory %s: %w", targetDir, err)
	}

	if !info.IsDir() {
		return &BuildValidationError{
			Stage:      "move",
			Command:    fmt.Sprintf("move %s to %s", extractedDir, targetDir),
			Files:      []string{extractedDir},
			Expected:   []string{"directory"},
			Actual:     []string{"file"},
			Suggestion: fmt.Sprintf("Expected %s to be a directory after module movement, but found a file instead", targetDir),
		}
	}

	// Validate directory contents
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("failed to read target directory %s: %w", targetDir, err)
	}

	if len(entries) == 0 {
		return &BuildValidationError{
			Stage:      "move",
			Command:    fmt.Sprintf("move %s to %s", extractedDir, targetDir),
			Files:      []string{extractedDir},
			Expected:   []string{"non-empty directory"},
			Actual:     []string{"empty directory"},
			Suggestion: fmt.Sprintf("Target directory %s is empty after module movement. The move operation may have failed.", targetDir),
		}
	}

	// Check for __init__.py file
	initPyPath := filepath.Join(targetDir, "__init__.py")
	hasInitPy := fileExists(initPyPath)

	var moduleFiles []string
	var pythonFiles []string
	var subdirectories []string

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			subdirectories = append(subdirectories, name)
		} else {
			moduleFiles = append(moduleFiles, name)
			if strings.HasSuffix(name, ".py") {
				pythonFiles = append(pythonFiles, name)
			}
		}
	}

	mv.logger.Info("module placement validation results",
		"targetDir", targetDir,
		"moduleName", moduleName,
		"totalFiles", len(moduleFiles),
		"pythonFiles", len(pythonFiles),
		"subdirectories", len(subdirectories),
		"hasInitPy", hasInitPy,
		"files", moduleFiles)

	// Validate that we have Python files
	if len(pythonFiles) == 0 && len(subdirectories) == 0 {
		return &BuildValidationError{
			Stage:      "move",
			Command:    fmt.Sprintf("move %s to %s", extractedDir, targetDir),
			Files:      []string{extractedDir},
			Expected:   []string{"Python files or subdirectories"},
			Actual:     moduleFiles,
			Suggestion: fmt.Sprintf("No Python files or subdirectories found in target directory %s. The module may not have been moved correctly.", targetDir),
		}
	}

	// Validate that __init__.py exists for Python packages
	if len(pythonFiles) > 1 && !hasInitPy {
		return &BuildValidationError{
			Stage:      "move",
			Command:    fmt.Sprintf("move %s to %s", extractedDir, targetDir),
			Files:      []string{extractedDir},
			Expected:   []string{"__init__.py file"},
			Actual:     []string{"missing __init__.py"},
			Suggestion: fmt.Sprintf("Python package in %s is missing __init__.py file. This file is required for Python packages.", targetDir),
		}
	}

	return nil
}

// ValidateInitPyFiles validates that __init__.py files are created where needed
func (mv *ModuleValidator) ValidateInitPyFiles(targetDir, moduleName string) error {
	mv.logger.Info("validating __init__.py files", "targetDir", targetDir, "moduleName", moduleName)

	// The __init__.py should be in the module directory, not the parent directory
	moduleDir := filepath.Join(targetDir, moduleName)
	initPyPath := filepath.Join(moduleDir, "__init__.py")

	// Check if __init__.py exists
	if !fileExists(initPyPath) {
		mv.logger.Warn("__init__.py file missing", "path", initPyPath)
		return &BuildValidationError{
			Stage:      "validate",
			Command:    "create __init__.py",
			Files:      []string{moduleDir},
			Expected:   []string{initPyPath},
			Actual:     []string{"file not found"},
			Suggestion: fmt.Sprintf("__init__.py file is missing in %s. This file is required for proper Python package structure.", moduleDir),
		}
	}

	// Validate that the file is readable
	info, err := os.Stat(initPyPath)
	if err != nil {
		return fmt.Errorf("failed to stat __init__.py file %s: %w", initPyPath, err)
	}

	mv.logger.Info("__init__.py validation successful",
		"path", initPyPath,
		"size", info.Size(),
		"mode", info.Mode())

	// Check for subdirectories that might also need __init__.py files
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		return fmt.Errorf("failed to read module directory %s: %w", moduleDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDirPath := filepath.Join(moduleDir, entry.Name())
			subInitPyPath := filepath.Join(subDirPath, "__init__.py")

			// Check if this subdirectory contains Python files
			if mv.containsPythonFiles(subDirPath) && !fileExists(subInitPyPath) {
				mv.logger.Warn("subdirectory missing __init__.py",
					"subDir", subDirPath,
					"expectedFile", subInitPyPath)
				return &BuildValidationError{
					Stage:      "validate",
					Command:    "create __init__.py",
					Files:      []string{subDirPath},
					Expected:   []string{subInitPyPath},
					Actual:     []string{"file not found"},
					Suggestion: fmt.Sprintf("subpackage missing __init__.py in %s. Python subpackages require __init__.py files.", subDirPath),
				}
			}
		}
	}

	return nil
}

// containsPythonFiles checks if a directory contains Python files
func (mv *ModuleValidator) containsPythonFiles(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
			return true
		}
	}
	return false
}

// ValidateModuleStructure validates the overall structure of a Python module
func (mv *ModuleValidator) ValidateModuleStructure(targetDir, moduleName string) error {
	mv.logger.Info("validating module structure", "targetDir", targetDir, "moduleName", moduleName)

	// Check if target directory exists
	if !fileExists(targetDir) {
		return fmt.Errorf("target directory does not exist: %s", targetDir)
	}

	// Validate __init__.py files
	if err := mv.ValidateInitPyFiles(targetDir, moduleName); err != nil {
		return fmt.Errorf("__init__.py validation failed: %w", err)
	}

	// Check for common Python module patterns
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("failed to read target directory %s: %w", targetDir, err)
	}

	var pythonFiles []string
	var packageDirs []string
	var otherFiles []string

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			packageDirs = append(packageDirs, name)
		} else if strings.HasSuffix(name, ".py") {
			pythonFiles = append(pythonFiles, name)
		} else {
			otherFiles = append(otherFiles, name)
		}
	}

	mv.logger.Info("module structure analysis",
		"targetDir", targetDir,
		"moduleName", moduleName,
		"pythonFiles", pythonFiles,
		"packageDirs", packageDirs,
		"otherFiles", otherFiles)

	// Validate that we have a proper module structure
	if len(pythonFiles) == 0 && len(packageDirs) == 0 {
		return &BuildValidationError{
			Stage:      "validate",
			Command:    "validate module structure",
			Files:      []string{targetDir},
			Expected:   []string{"Python files or package directories"},
			Actual:     otherFiles,
			Suggestion: fmt.Sprintf("Module %s in %s does not contain any Python files or package directories", moduleName, targetDir),
		}
	}

	return nil
}

// ValidateAllModules validates all modules in the output directory
func (mv *ModuleValidator) ValidateAllModules() error {
	mv.logger.Info("validating all modules", "outputDir", mv.outputDir)

	entries, err := os.ReadDir(mv.outputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory %s: %w", mv.outputDir, err)
	}

	var moduleCount int
	var validationErrors []string

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip common non-module directories
			if mv.isNonModuleDirectory(entry.Name()) {
				continue
			}

			modulePath := filepath.Join(mv.outputDir, entry.Name())
			moduleCount++

			mv.logger.Debug("validating module", "module", entry.Name(), "path", modulePath)

			if err := mv.ValidateModuleStructure(modulePath, entry.Name()); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("module %s: %v", entry.Name(), err))
			}
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("module validation failed for %d modules: %s", len(validationErrors), strings.Join(validationErrors, "; "))
	}

	mv.logger.Info("all modules validated successfully",
		"totalModules", moduleCount,
		"outputDir", mv.outputDir)

	return nil
}

// ValidatePackageStructure validates the structure of a Python package
func (mv *ModuleValidator) ValidatePackageStructure(packageDir string) error {
	return mv.ValidateModuleStructure(filepath.Dir(packageDir), filepath.Base(packageDir))
}

// EnsureInitPyFiles ensures that __init__.py files are created where needed
func (mv *ModuleValidator) EnsureInitPyFiles(moduleDir string) error {
	return mv.createInitPyFiles(filepath.Dir(moduleDir), filepath.Base(moduleDir))
}

// createInitPyFiles creates missing __init__.py files in the module and its subpackages
func (mv *ModuleValidator) createInitPyFiles(targetDir, moduleName string) error {
	mv.logger.Info("ensuring __init__.py files", "targetDir", targetDir, "moduleName", moduleName)

	// The __init__.py should be in the module directory
	moduleDir := filepath.Join(targetDir, moduleName)
	initPyPath := filepath.Join(moduleDir, "__init__.py")

	// Create __init__.py if it doesn't exist
	if !fileExists(initPyPath) {
		mv.logger.Info("creating missing __init__.py", "path", initPyPath)
		if err := os.WriteFile(initPyPath, []byte("# Auto-generated __init__.py\n"), 0644); err != nil {
			return fmt.Errorf("failed to create __init__.py file %s: %w", initPyPath, err)
		}
	}

	// Check for subdirectories that might also need __init__.py files
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		return fmt.Errorf("failed to read module directory %s: %w", moduleDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDirPath := filepath.Join(moduleDir, entry.Name())
			subInitPyPath := filepath.Join(subDirPath, "__init__.py")

			// Check if this subdirectory contains Python files
			if mv.containsPythonFiles(subDirPath) && !fileExists(subInitPyPath) {
				mv.logger.Info("creating missing __init__.py in subpackage", "path", subInitPyPath)
				if err := os.WriteFile(subInitPyPath, []byte("# Auto-generated __init__.py\n"), 0644); err != nil {
					return fmt.Errorf("failed to create __init__.py file %s: %w", subInitPyPath, err)
				}
			}
		}
	}

	return nil
}

// isNonModuleDirectory checks if a directory name represents a non-module directory
func (mv *ModuleValidator) isNonModuleDirectory(name string) bool {
	nonModuleDirs := []string{
		"__pycache__",
		".pytest_cache",
		".git",
		".sst",
		"node_modules",
	}

	for _, nonModuleDir := range nonModuleDirs {
		if name == nonModuleDir {
			return true
		}
	}

	// Skip .dist-info and .egg-info directories
	if strings.HasSuffix(name, ".dist-info") || strings.HasSuffix(name, ".egg-info") {
		return true
	}

	// Skip cache artifact directories that start with pkg_
	// These are created by the build cache system and are not actual Python modules
	if strings.HasPrefix(name, "pkg_") {
		return true
	}

	return false
}
