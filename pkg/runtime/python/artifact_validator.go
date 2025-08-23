package python

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactValidator validates deployment artifacts for both size and content
type ArtifactValidator struct {
	artifactDir     string
	expectedModules []string
	maxSizeBytes    int64
	logger          *slog.Logger
}

// ArtifactValidationResult contains the results of artifact validation
type ArtifactValidationResult struct {
	Success            bool     `json:"success"`
	TotalSize          int64    `json:"totalSize"`
	FileCount          int      `json:"fileCount"`
	PythonModules      []string `json:"pythonModules"`
	MissingModules     []string `json:"missingModules"`
	DependencyPackages []string `json:"dependencyPackages"`
	HandlerCompatible  bool     `json:"handlerCompatible"`
	SizeWithinLimits   bool     `json:"sizeWithinLimits"`
	ErrorMessages      []string `json:"errorMessages"`
	WarningMessages    []string `json:"warningMessages"`
}

// NewArtifactValidator creates a new artifact validator
func NewArtifactValidator(artifactDir string, expectedModules []string, maxSizeBytes int64) *ArtifactValidator {
	return &ArtifactValidator{
		artifactDir:     artifactDir,
		expectedModules: expectedModules,
		maxSizeBytes:    maxSizeBytes,
		logger:          slog.Default(),
	}
}

// ValidateArtifact performs comprehensive validation of the deployment artifact
func (av *ArtifactValidator) ValidateArtifact(handlerPath string) (*ArtifactValidationResult, error) {
	av.logger.Info("starting comprehensive artifact validation",
		"artifactDir", av.artifactDir,
		"expectedModules", av.expectedModules,
		"handlerPath", handlerPath,
		"maxSizeBytes", av.maxSizeBytes)

	result := &ArtifactValidationResult{
		Success:            true,
		ErrorMessages:      []string{},
		WarningMessages:    []string{},
		PythonModules:      []string{},
		MissingModules:     []string{},
		DependencyPackages: []string{},
	}

	// 1. Validate artifact directory exists
	if err := av.validateArtifactDirectory(); err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("artifact directory validation failed: %v", err))
		return result, err
	}

	// 2. Analyze artifact contents
	if err := av.analyzeArtifactContents(result); err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("artifact content analysis failed: %v", err))
		return result, err
	}

	// 3. Validate module presence
	if err := av.validateModulePresence(result); err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("module presence validation failed: %v", err))
	}

	// 4. Validate module structure
	if err := av.validateModuleStructure(result); err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("module structure validation failed: %v", err))
	}

	// 5. Validate handler path compatibility
	if err := av.validateHandlerPathCompatibility(handlerPath, result); err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("handler path validation failed: %v", err))
	}

	// 6. Validate artifact size
	if err := av.validateArtifactSize(result); err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("artifact size validation failed: %v", err))
	}

	// 7. Log comprehensive validation results
	av.logValidationResults(result)

	av.logger.Info("artifact validation completed",
		"success", result.Success,
		"errorCount", len(result.ErrorMessages),
		"warningCount", len(result.WarningMessages))

	return result, nil
}

// validateArtifactDirectory validates that the artifact directory exists and is accessible
func (av *ArtifactValidator) validateArtifactDirectory() error {
	av.logger.Debug("validating artifact directory", "dir", av.artifactDir)

	info, err := os.Stat(av.artifactDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("artifact directory does not exist: %s", av.artifactDir)
		}
		return fmt.Errorf("failed to access artifact directory %s: %w", av.artifactDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("artifact path is not a directory: %s", av.artifactDir)
	}

	return nil
}

// analyzeArtifactContents analyzes the contents of the artifact directory
func (av *ArtifactValidator) analyzeArtifactContents(result *ArtifactValidationResult) error {
	av.logger.Debug("analyzing artifact contents", "dir", av.artifactDir)

	var totalSize int64
	var fileCount int
	var pythonModules []string
	var dependencyPackages []string
	var allFiles []string

	err := filepath.Walk(av.artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			av.logger.Warn("error walking artifact directory",
				"path", path,
				"error", err)
			return nil // Continue walking despite errors
		}

		// Skip the root directory itself
		if path == av.artifactDir {
			return nil
		}

		relPath, err := filepath.Rel(av.artifactDir, path)
		if err != nil {
			av.logger.Warn("failed to get relative path",
				"path", path,
				"error", err)
			return nil
		}

		if info.IsDir() {
			// Check if this is a Python module directory
			if av.isPythonModule(path) {
				moduleName := info.Name()
				pythonModules = append(pythonModules, moduleName)
				av.logger.Debug("found Python module", "module", moduleName, "path", relPath)
			}

			// Check if this is a dependency package directory
			if av.isDependencyPackage(info.Name()) {
				dependencyPackages = append(dependencyPackages, info.Name())
				av.logger.Debug("found dependency package", "package", info.Name(), "path", relPath)
			}
		} else {
			// Count files and calculate total size
			fileCount++
			totalSize += info.Size()
			allFiles = append(allFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk artifact directory: %w", err)
	}

	// Update result with analysis
	result.TotalSize = totalSize
	result.FileCount = fileCount
	result.PythonModules = pythonModules
	result.DependencyPackages = dependencyPackages

	av.logger.Info("artifact content analysis completed",
		"totalSize", totalSize,
		"fileCount", fileCount,
		"pythonModules", len(pythonModules),
		"dependencyPackages", len(dependencyPackages),
		"modules", pythonModules,
		"dependencies", dependencyPackages)

	return nil
}

// isPythonModule checks if a directory is a Python module
func (av *ArtifactValidator) isPythonModule(dirPath string) bool {
	// Check for __init__.py file
	initPyPath := filepath.Join(dirPath, "__init__.py")
	if fileExists(initPyPath) {
		return true
	}

	// Check if directory contains Python files
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

// isDependencyPackage checks if a directory name represents a dependency package
func (av *ArtifactValidator) isDependencyPackage(name string) bool {
	// Common patterns for dependency packages
	dependencyPatterns := []string{
		".dist-info",
		".egg-info",
		"__pycache__",
	}

	for _, pattern := range dependencyPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	// Check for common Python package names (this is a basic heuristic)
	// In a real implementation, you might want to check against a known list
	// or use other heuristics to identify dependency packages
	return false
}

// validateModulePresence validates that all expected modules are present
func (av *ArtifactValidator) validateModulePresence(result *ArtifactValidationResult) error {
	av.logger.Debug("validating module presence", "expectedModules", av.expectedModules)

	var missingModules []string

	for _, expectedModule := range av.expectedModules {
		found := false
		for _, actualModule := range result.PythonModules {
			if actualModule == expectedModule {
				found = true
				break
			}
		}

		if !found {
			missingModules = append(missingModules, expectedModule)
			av.logger.Warn("expected module not found",
				"module", expectedModule,
				"availableModules", result.PythonModules)
		}
	}

	result.MissingModules = missingModules

	if len(missingModules) > 0 {
		return &BuildValidationError{
			Stage:      "validate",
			Command:    "module presence validation",
			Files:      []string{av.artifactDir},
			Expected:   av.expectedModules,
			Actual:     result.PythonModules,
			Suggestion: fmt.Sprintf("Missing modules: %s. Check if the build process correctly extracted and placed all required modules.", strings.Join(missingModules, ", ")),
		}
	}

	av.logger.Info("module presence validation successful",
		"expectedModules", av.expectedModules,
		"foundModules", result.PythonModules)

	return nil
}

// validateModuleStructure validates that modules have proper package structure
func (av *ArtifactValidator) validateModuleStructure(result *ArtifactValidationResult) error {
	av.logger.Debug("validating module structure", "modules", result.PythonModules)

	var structureErrors []string

	for _, moduleName := range result.PythonModules {
		modulePath := filepath.Join(av.artifactDir, moduleName)

		// Validate __init__.py file
		initPyPath := filepath.Join(modulePath, "__init__.py")
		if !fileExists(initPyPath) {
			structureErrors = append(structureErrors, fmt.Sprintf("module %s missing __init__.py", moduleName))
			result.WarningMessages = append(result.WarningMessages, fmt.Sprintf("Module %s is missing __init__.py file", moduleName))
		}

		// Validate that module contains Python files
		if !av.containsPythonFiles(modulePath) {
			structureErrors = append(structureErrors, fmt.Sprintf("module %s contains no Python files", moduleName))
			result.WarningMessages = append(result.WarningMessages, fmt.Sprintf("Module %s contains no Python files", moduleName))
		}

		av.logger.Debug("module structure validation",
			"module", moduleName,
			"hasInitPy", fileExists(initPyPath),
			"hasPythonFiles", av.containsPythonFiles(modulePath))
	}

	if len(structureErrors) > 0 {
		av.logger.Warn("module structure validation found issues",
			"errors", structureErrors)
		// Don't fail validation for structure issues, just warn
	}

	av.logger.Info("module structure validation completed",
		"modules", len(result.PythonModules),
		"structureErrors", len(structureErrors))

	return nil
}

// containsPythonFiles checks if a directory contains Python files
func (av *ArtifactValidator) containsPythonFiles(dirPath string) bool {
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

// validateHandlerPathCompatibility validates that the handler path will work with deployed modules
func (av *ArtifactValidator) validateHandlerPathCompatibility(handlerPath string, result *ArtifactValidationResult) error {
	av.logger.Debug("validating handler path compatibility", "handlerPath", handlerPath)

	if handlerPath == "" {
		result.WarningMessages = append(result.WarningMessages, "Handler path is empty")
		return nil
	}

	// Parse handler path to extract module name
	var expectedModule string
	var handlerParts []string

	if strings.Contains(handlerPath, "/") {
		// Workspace layout: functions/handler.lambda_handler
		handlerParts = strings.Split(handlerPath, "/")
		if len(handlerParts) == 0 {
			return fmt.Errorf("invalid handler path format: %s", handlerPath)
		}
		expectedModule = handlerParts[0]
	} else {
		// Flat layout: handler.lambda_handler or module.handler.lambda_handler
		// For flat layouts, we need to check if the handler path contains a module prefix
		dotParts := strings.Split(handlerPath, ".")
		if len(dotParts) >= 3 {
			// Format: module.handler.lambda_handler
			expectedModule = dotParts[0]
			handlerParts = []string{expectedModule, strings.Join(dotParts[1:len(dotParts)-1], ".")}
		} else if len(dotParts) == 2 {
			// Format: handler.lambda_handler (no module prefix)
			// In this case, we need to find which module contains the handler file
			handlerFile := dotParts[0] + ".py"
			for _, module := range result.PythonModules {
				handlerFilePath := filepath.Join(av.artifactDir, module, handlerFile)
				if fileExists(handlerFilePath) {
					expectedModule = module
					handlerParts = []string{module, dotParts[0]}
					break
				}
			}
			if expectedModule == "" {
				// If no module contains the handler file, this might be an error
				// but we'll let the later validation catch it
				expectedModule = dotParts[0] // This will likely fail validation
				handlerParts = []string{expectedModule}
			}
		} else {
			return fmt.Errorf("invalid handler path format: %s", handlerPath)
		}
	}

	// Check if the expected module exists in the artifact
	moduleFound := false
	for _, module := range result.PythonModules {
		if module == expectedModule {
			moduleFound = true
			break
		}
	}

	result.HandlerCompatible = moduleFound

	if !moduleFound {
		return &BuildValidationError{
			Stage:      "validate",
			Command:    "handler path validation",
			Files:      []string{av.artifactDir},
			Expected:   []string{expectedModule},
			Actual:     result.PythonModules,
			Suggestion: fmt.Sprintf("Handler path '%s' expects module '%s' but it was not found in the artifact. Available modules: %s", handlerPath, expectedModule, strings.Join(result.PythonModules, ", ")),
		}
	}

	// Validate that the handler file exists within the module
	if len(handlerParts) > 1 {
		handlerFile := strings.Join(handlerParts[1:], "/") + ".py"
		handlerFilePath := filepath.Join(av.artifactDir, expectedModule, handlerFile)

		if !fileExists(handlerFilePath) {
			result.WarningMessages = append(result.WarningMessages, fmt.Sprintf("Handler file %s not found in module %s", handlerFile, expectedModule))
			av.logger.Warn("handler file not found",
				"handlerPath", handlerPath,
				"expectedFile", handlerFilePath,
				"module", expectedModule)
		} else {
			av.logger.Debug("handler file found",
				"handlerPath", handlerPath,
				"handlerFile", handlerFilePath)
		}
	}

	av.logger.Info("handler path compatibility validation completed",
		"handlerPath", handlerPath,
		"expectedModule", expectedModule,
		"compatible", result.HandlerCompatible)

	return nil
}

// validateArtifactSize validates that the artifact size is within limits
func (av *ArtifactValidator) validateArtifactSize(result *ArtifactValidationResult) error {
	av.logger.Debug("validating artifact size",
		"totalSize", result.TotalSize,
		"maxSizeBytes", av.maxSizeBytes)

	result.SizeWithinLimits = av.maxSizeBytes <= 0 || result.TotalSize <= av.maxSizeBytes

	if !result.SizeWithinLimits {
		return &BuildValidationError{
			Stage:      "validate",
			Command:    "artifact size validation",
			Files:      []string{av.artifactDir},
			Expected:   []string{fmt.Sprintf("size <= %d bytes", av.maxSizeBytes)},
			Actual:     []string{fmt.Sprintf("size = %d bytes", result.TotalSize)},
			Suggestion: fmt.Sprintf("Artifact size (%d bytes) exceeds the maximum limit (%d bytes). Consider excluding unnecessary files or reducing dependencies.", result.TotalSize, av.maxSizeBytes),
		}
	}

	av.logger.Info("artifact size validation completed",
		"totalSize", result.TotalSize,
		"maxSizeBytes", av.maxSizeBytes,
		"withinLimits", result.SizeWithinLimits)

	return nil
}

// logValidationResults logs comprehensive validation results for debugging
func (av *ArtifactValidator) logValidationResults(result *ArtifactValidationResult) {
	av.logger.Info("comprehensive artifact validation results",
		"artifactDir", av.artifactDir,
		"success", result.Success,
		"totalSize", result.TotalSize,
		"fileCount", result.FileCount,
		"pythonModules", result.PythonModules,
		"moduleCount", len(result.PythonModules),
		"missingModules", result.MissingModules,
		"missingCount", len(result.MissingModules),
		"dependencyPackages", result.DependencyPackages,
		"dependencyCount", len(result.DependencyPackages),
		"handlerCompatible", result.HandlerCompatible,
		"sizeWithinLimits", result.SizeWithinLimits,
		"errorCount", len(result.ErrorMessages),
		"warningCount", len(result.WarningMessages))

	// Log errors
	for i, errMsg := range result.ErrorMessages {
		av.logger.Error("validation error",
			"index", i+1,
			"message", errMsg)
	}

	// Log warnings
	for i, warnMsg := range result.WarningMessages {
		av.logger.Warn("validation warning",
			"index", i+1,
			"message", warnMsg)
	}

	// Log detailed module information
	for _, module := range result.PythonModules {
		modulePath := filepath.Join(av.artifactDir, module)
		if info, err := os.Stat(modulePath); err == nil {
			av.logger.Debug("module details",
				"module", module,
				"path", modulePath,
				"isDir", info.IsDir(),
				"hasInitPy", fileExists(filepath.Join(modulePath, "__init__.py")))
		}
	}
}

// ListArtifactContents returns a detailed list of artifact contents for debugging
func (av *ArtifactValidator) ListArtifactContents() ([]string, error) {
	av.logger.Debug("listing artifact contents", "dir", av.artifactDir)

	var contents []string

	err := filepath.Walk(av.artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == av.artifactDir {
			return nil
		}

		relPath, err := filepath.Rel(av.artifactDir, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			contents = append(contents, fmt.Sprintf("DIR:  %s/", relPath))
		} else {
			contents = append(contents, fmt.Sprintf("FILE: %s (%d bytes)", relPath, info.Size()))
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list artifact contents: %w", err)
	}

	av.logger.Info("artifact contents listed",
		"dir", av.artifactDir,
		"totalItems", len(contents))

	return contents, nil
}

// DeploymentPackageValidationResult contains the results of deployment package validation
type DeploymentPackageValidationResult struct {
	Success               bool     `json:"success"`
	PackageSize           int64    `json:"packageSize"`
	SizeWithinLimits      bool     `json:"sizeWithinLimits"`
	ContainsSourceModules bool     `json:"containsSourceModules"`
	ContainsDependencies  bool     `json:"containsDependencies"`
	HandlerPathValid      bool     `json:"handlerPathValid"`
	ErrorMessages         []string `json:"errorMessages"`
	WarningMessages       []string `json:"warningMessages"`
	PackageContents       []string `json:"packageContents"`
}

// ValidateDeploymentPackage validates the final deployment package (zip file)
func (av *ArtifactValidator) ValidateDeploymentPackage(packagePath string) error {
	av.logger.Info("validating deployment package", "packagePath", packagePath)

	// Check if package file exists
	info, err := os.Stat(packagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("deployment package does not exist: %s", packagePath)
		}
		return fmt.Errorf("failed to access deployment package %s: %w", packagePath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("deployment package path is a directory, not a file: %s", packagePath)
	}

	packageSize := info.Size()

	// Validate package size
	if av.maxSizeBytes > 0 && packageSize > av.maxSizeBytes {
		return &BuildValidationError{
			Stage:      "validate",
			Command:    "deployment package validation",
			Files:      []string{packagePath},
			Expected:   []string{fmt.Sprintf("size <= %d bytes", av.maxSizeBytes)},
			Actual:     []string{fmt.Sprintf("size = %d bytes", packageSize)},
			Suggestion: fmt.Sprintf("Deployment package size (%d bytes) exceeds the maximum limit (%d bytes). The package is too large for Lambda deployment.", packageSize, av.maxSizeBytes),
		}
	}

	av.logger.Info("deployment package validation completed",
		"packagePath", packagePath,
		"packageSize", packageSize,
		"maxSizeBytes", av.maxSizeBytes,
		"valid", true)

	return nil
}

// ValidateDeploymentPackageComprehensive performs comprehensive validation of the deployment package
func (av *ArtifactValidator) ValidateDeploymentPackageComprehensive(packagePath, handlerPath string) (*DeploymentPackageValidationResult, error) {
	av.logger.Info("starting comprehensive deployment package validation",
		"packagePath", packagePath,
		"handlerPath", handlerPath)

	result := &DeploymentPackageValidationResult{
		Success:         true,
		ErrorMessages:   []string{},
		WarningMessages: []string{},
		PackageContents: []string{},
	}

	// 1. Validate package file exists and is accessible
	info, err := os.Stat(packagePath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Success = false
			result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("deployment package does not exist: %s", packagePath))
			return result, fmt.Errorf("deployment package does not exist: %s", packagePath)
		}
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("failed to access deployment package: %v", err))
		return result, fmt.Errorf("failed to access deployment package %s: %w", packagePath, err)
	}

	if info.IsDir() {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, "deployment package path is a directory, not a file")
		return result, fmt.Errorf("deployment package path is a directory, not a file: %s", packagePath)
	}

	result.PackageSize = info.Size()

	// 2. Validate package size
	result.SizeWithinLimits = av.maxSizeBytes <= 0 || result.PackageSize <= av.maxSizeBytes
	if !result.SizeWithinLimits {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages,
			fmt.Sprintf("package size (%d bytes) exceeds limit (%d bytes)", result.PackageSize, av.maxSizeBytes))
	}

	// 3. Validate that the artifact directory contains expected content
	// (This validates the source before it gets zipped)
	artifactResult, err := av.ValidateArtifact(handlerPath)
	if err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("artifact validation failed: %v", err))
		return result, fmt.Errorf("artifact validation failed: %w", err)
	}

	// Copy relevant validation results
	result.ContainsSourceModules = len(artifactResult.PythonModules) > 0
	result.ContainsDependencies = len(artifactResult.DependencyPackages) > 0
	result.HandlerPathValid = artifactResult.HandlerCompatible

	// Add any errors or warnings from artifact validation
	result.ErrorMessages = append(result.ErrorMessages, artifactResult.ErrorMessages...)
	result.WarningMessages = append(result.WarningMessages, artifactResult.WarningMessages...)

	// 4. Validate package structure requirements
	if err := av.validatePackageStructureRequirements(result, handlerPath); err != nil {
		result.Success = false
		result.ErrorMessages = append(result.ErrorMessages, fmt.Sprintf("package structure validation failed: %v", err))
	}

	// 5. Log comprehensive results
	av.logDeploymentPackageResults(result, packagePath, handlerPath)

	av.logger.Info("comprehensive deployment package validation completed",
		"packagePath", packagePath,
		"success", result.Success,
		"packageSize", result.PackageSize,
		"sizeWithinLimits", result.SizeWithinLimits,
		"containsSourceModules", result.ContainsSourceModules,
		"containsDependencies", result.ContainsDependencies,
		"handlerPathValid", result.HandlerPathValid)

	return result, nil
}

// validatePackageStructureRequirements validates that the package meets deployment requirements
func (av *ArtifactValidator) validatePackageStructureRequirements(result *DeploymentPackageValidationResult, handlerPath string) error {
	av.logger.Debug("validating package structure requirements", "handlerPath", handlerPath)

	// Check that source modules are present
	if !result.ContainsSourceModules {
		return &BuildValidationError{
			Stage:      "validate",
			Command:    "package structure validation",
			Files:      []string{av.artifactDir},
			Expected:   []string{"Python source modules"},
			Actual:     []string{"no source modules found"},
			Suggestion: "The deployment package does not contain any Python source modules. Check if the build process correctly extracted and included your application code.",
		}
	}

	// Validate handler path compatibility
	if !result.HandlerPathValid {
		return &BuildValidationError{
			Stage:      "validate",
			Command:    "handler path validation",
			Files:      []string{av.artifactDir},
			Expected:   []string{fmt.Sprintf("module for handler path: %s", handlerPath)},
			Actual:     []string{"handler module not found"},
			Suggestion: fmt.Sprintf("The handler path '%s' is not compatible with the modules in the deployment package. Ensure the handler module is properly included.", handlerPath),
		}
	}

	// Check for both source and dependencies (for non-container builds)
	if result.ContainsSourceModules && !result.ContainsDependencies {
		result.WarningMessages = append(result.WarningMessages,
			"Package contains source modules but no dependency packages. This may be expected for container builds or if no external dependencies are used.")
	}

	// Validate reasonable package size
	if result.PackageSize == 0 {
		return &BuildValidationError{
			Stage:      "validate",
			Command:    "package size validation",
			Files:      []string{av.artifactDir},
			Expected:   []string{"non-empty package"},
			Actual:     []string{"empty package"},
			Suggestion: "The deployment package is empty. Check if the build process completed successfully and created the package file.",
		}
	}

	// Warn about very small packages (likely missing content)
	if result.PackageSize < 1024 { // Less than 1KB
		result.WarningMessages = append(result.WarningMessages,
			fmt.Sprintf("Package size is very small (%d bytes). This may indicate missing content.", result.PackageSize))
	}

	// Warn about very large packages (approaching limits)
	if av.maxSizeBytes > 0 && result.PackageSize > av.maxSizeBytes*8/10 { // 80% of limit
		result.WarningMessages = append(result.WarningMessages,
			fmt.Sprintf("Package size (%d bytes) is approaching the limit (%d bytes). Consider optimizing dependencies or excluding unnecessary files.", result.PackageSize, av.maxSizeBytes))
	}

	return nil
}

// logDeploymentPackageResults logs comprehensive deployment package validation results
func (av *ArtifactValidator) logDeploymentPackageResults(result *DeploymentPackageValidationResult, packagePath, handlerPath string) {
	av.logger.Info("comprehensive deployment package validation results",
		"packagePath", packagePath,
		"handlerPath", handlerPath,
		"success", result.Success,
		"packageSize", result.PackageSize,
		"sizeWithinLimits", result.SizeWithinLimits,
		"containsSourceModules", result.ContainsSourceModules,
		"containsDependencies", result.ContainsDependencies,
		"handlerPathValid", result.HandlerPathValid,
		"errorCount", len(result.ErrorMessages),
		"warningCount", len(result.WarningMessages))

	// Log errors
	for i, errMsg := range result.ErrorMessages {
		av.logger.Error("deployment package validation error",
			"index", i+1,
			"message", errMsg)
	}

	// Log warnings
	for i, warnMsg := range result.WarningMessages {
		av.logger.Warn("deployment package validation warning",
			"index", i+1,
			"message", warnMsg)
	}

	// Log size information in human-readable format
	sizeKB := float64(result.PackageSize) / 1024
	sizeMB := sizeKB / 1024

	if sizeMB >= 1 {
		av.logger.Info("deployment package size",
			"bytes", result.PackageSize,
			"MB", fmt.Sprintf("%.2f", sizeMB))
	} else {
		av.logger.Info("deployment package size",
			"bytes", result.PackageSize,
			"KB", fmt.Sprintf("%.2f", sizeKB))
	}

	// Log validation summary
	if result.Success {
		av.logger.Info("deployment package validation summary: PASSED",
			"packagePath", packagePath,
			"allChecks", "passed")
	} else {
		av.logger.Warn("deployment package validation summary: FAILED",
			"packagePath", packagePath,
			"failedChecks", len(result.ErrorMessages))
	}
}
