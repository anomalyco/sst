package python

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// BuildOutputValidator validates the outputs of the build process
type BuildOutputValidator struct {
	outputDir string
	logger    *slog.Logger
}

// NewBuildOutputValidator creates a new build output validator
func NewBuildOutputValidator(outputDir string) *BuildOutputValidator {
	return &BuildOutputValidator{
		outputDir: outputDir,
		logger:    slog.Default(),
	}
}

// ValidateBuildOutputs validates that the uv build command produced expected outputs
func (bov *BuildOutputValidator) ValidateBuildOutputs() error {
	bov.logger.Info("validating build outputs", "outputDir", bov.outputDir)

	// Check if output directory exists
	if _, err := os.Stat(bov.outputDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory does not exist: %s", bov.outputDir)
	}

	// List all files in output directory for diagnostic purposes
	entries, err := os.ReadDir(bov.outputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory %s: %w", bov.outputDir, err)
	}

	var allFiles []string
	for _, entry := range entries {
		allFiles = append(allFiles, entry.Name())
	}

	bov.logger.Info("output directory contents during validation",
		"dir", bov.outputDir,
		"files", allFiles,
		"count", len(allFiles))

	// Find tar.gz files
	tarGzFiles, err := bov.ListTarGzFiles()
	if err != nil {
		// If it's already a BuildValidationError, return it directly
		if _, ok := err.(*BuildValidationError); ok {
			return err
		}
		return fmt.Errorf("failed to list tar.gz files: %w", err)
	}

	if len(tarGzFiles) == 0 {
		return &BuildValidationError{
			Stage:      "build",
			Command:    "uv build --all --sdist",
			Expected:   []string{"*.tar.gz files"},
			Actual:     allFiles,
			Suggestion: "Check if uv build command completed successfully and created source distribution files",
		}
	}

	bov.logger.Info("build output validation successful",
		"tarGzFiles", tarGzFiles,
		"count", len(tarGzFiles))

	return nil
}

// ListTarGzFiles finds and validates tar.gz files in the output directory
func (bov *BuildOutputValidator) ListTarGzFiles() ([]string, error) {
	globPattern := filepath.Join(bov.outputDir, "*.tar.gz")
	bov.logger.Info("searching for tar.gz files", "pattern", globPattern)

	files, err := filepath.Glob(globPattern)
	if err != nil {
		bov.logger.Error("failed to glob tar.gz files",
			"pattern", globPattern,
			"error", err)
		return nil, fmt.Errorf("failed to glob tar.gz files with pattern %s: %w", globPattern, err)
	}

	// If glob fails to find files, try alternative discovery methods
	if len(files) == 0 {
		bov.logger.Warn("glob pattern found no tar.gz files, trying alternative discovery methods",
			"pattern", globPattern)

		alternativeFiles, err := bov.findTarGzFilesAlternative()
		if err != nil {
			bov.logger.Error("alternative tar.gz discovery failed", "error", err)
			// If it's already a BuildValidationError, return it directly
			if _, ok := err.(*BuildValidationError); ok {
				return nil, err
			}
			return nil, fmt.Errorf("both glob and alternative discovery failed: %w", err)
		}
		files = alternativeFiles
	}

	// Convert to relative paths for cleaner logging
	var relativeFiles []string
	for _, file := range files {
		relPath, err := filepath.Rel(bov.outputDir, file)
		if err != nil {
			relPath = filepath.Base(file)
		}
		relativeFiles = append(relativeFiles, relPath)
	}

	bov.logger.Info("tar.gz file search results",
		"pattern", globPattern,
		"files", relativeFiles,
		"count", len(files))

	// Validate that files actually exist and are readable
	var validFiles []string
	for _, file := range files {
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			validFiles = append(validFiles, file)
			bov.logger.Debug("validated tar.gz file",
				"file", file,
				"size", info.Size())
		} else {
			bov.logger.Warn("invalid tar.gz file found",
				"file", file,
				"error", err)
		}
	}

	if len(validFiles) != len(files) {
		bov.logger.Warn("some tar.gz files failed validation",
			"found", len(files),
			"valid", len(validFiles))
	}

	return validFiles, nil
}

// findTarGzFilesAlternative provides alternative methods to find tar.gz files
func (bov *BuildOutputValidator) findTarGzFilesAlternative() ([]string, error) {
	bov.logger.Info("using alternative tar.gz file discovery", "outputDir", bov.outputDir)

	// Method 1: Read directory and filter for .tar.gz files
	entries, err := os.ReadDir(bov.outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read output directory: %w", err)
	}

	var tarGzFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tar.gz") {
			fullPath := filepath.Join(bov.outputDir, entry.Name())
			tarGzFiles = append(tarGzFiles, fullPath)
			bov.logger.Debug("found tar.gz file via directory scan",
				"file", entry.Name(),
				"fullPath", fullPath)
		}
	}

	// Method 2: Check for common patterns in subdirectories
	bov.logger.Info("checking subdirectories for tar.gz files")
	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(bov.outputDir, entry.Name())
			subPattern := filepath.Join(subDir, "*.tar.gz")

			subFiles, err := filepath.Glob(subPattern)
			if err != nil {
				bov.logger.Debug("failed to glob subdirectory",
					"subDir", subDir,
					"error", err)
				continue
			}

			if len(subFiles) > 0 {
				tarGzFiles = append(tarGzFiles, subFiles...)
				bov.logger.Info("found tar.gz files in subdirectory",
					"subDir", subDir,
					"files", len(subFiles))
			}
		}
	}

	if len(tarGzFiles) == 0 {
		// Method 3: Check if uv build might have used a different output location
		bov.logger.Warn("no tar.gz files found in expected locations, checking for build artifacts")

		// Log all files for debugging
		var allFiles []string
		for _, entry := range entries {
			allFiles = append(allFiles, entry.Name())
		}

		return nil, &BuildValidationError{
			Stage:      "build",
			Command:    "uv build --all --sdist",
			Files:      []string{bov.outputDir},
			Expected:   []string{"*.tar.gz files"},
			Actual:     allFiles,
			Suggestion: "uv build command may not have created source distribution files. Check if the command completed successfully and verify the output directory is correct.",
		}
	}

	bov.logger.Info("alternative discovery successful",
		"files", len(tarGzFiles))
	return tarGzFiles, nil
}

// ValidateExtractionResults validates that tar.gz extraction was successful
func (bov *BuildOutputValidator) ValidateExtractionResults(tarGzFiles []string) error {
	bov.logger.Info("validating extraction results", "tarGzFiles", len(tarGzFiles))

	for _, file := range tarGzFiles {
		// Get expected directory name from tar.gz file
		fileName := filepath.Base(file)
		dirName := strings.TrimSuffix(fileName, ".tar.gz")
		expectedDir := filepath.Join(bov.outputDir, dirName)

		bov.logger.Debug("checking extraction result",
			"tarGzFile", file,
			"expectedDir", expectedDir)

		// Check if extracted directory exists
		if info, err := os.Stat(expectedDir); err != nil {
			return &BuildValidationError{
				Stage:      "extract",
				Command:    fmt.Sprintf("tar -xzf %s", fileName),
				Files:      []string{file},
				Expected:   []string{expectedDir},
				Actual:     []string{"directory not found"},
				Suggestion: fmt.Sprintf("Check if tar extraction of %s completed successfully", fileName),
			}
		} else if !info.IsDir() {
			return &BuildValidationError{
				Stage:      "extract",
				Command:    fmt.Sprintf("tar -xzf %s", fileName),
				Files:      []string{file},
				Expected:   []string{"directory"},
				Actual:     []string{"file"},
				Suggestion: fmt.Sprintf("Expected %s to be a directory after extraction", expectedDir),
			}
		}

		// Validate directory contents
		entries, err := os.ReadDir(expectedDir)
		if err != nil {
			return fmt.Errorf("failed to read extracted directory %s: %w", expectedDir, err)
		}

		var extractedFiles []string
		for _, entry := range entries {
			extractedFiles = append(extractedFiles, entry.Name())
		}

		bov.logger.Info("extraction validation successful",
			"tarGzFile", file,
			"extractedDir", expectedDir,
			"contents", extractedFiles,
			"count", len(extractedFiles))
	}

	return nil
}

// BuildValidationError represents a build validation error with detailed context
type BuildValidationError struct {
	Stage      string   `json:"stage"`      // "build", "extract", "move", "validate"
	Command    string   `json:"command"`    // command that failed
	Files      []string `json:"files"`      // relevant files
	Expected   []string `json:"expected"`   // what was expected
	Actual     []string `json:"actual"`     // what was found
	Suggestion string   `json:"suggestion"` // how to fix
}

func (e *BuildValidationError) Error() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Build validation failed at stage '%s'", e.Stage))

	if e.Command != "" {
		parts = append(parts, fmt.Sprintf("Command: %s", e.Command))
	}

	if len(e.Files) > 0 {
		parts = append(parts, fmt.Sprintf("Files: %s", strings.Join(e.Files, ", ")))
	}

	if len(e.Expected) > 0 {
		parts = append(parts, fmt.Sprintf("Expected: %s", strings.Join(e.Expected, ", ")))
	}

	if len(e.Actual) > 0 {
		parts = append(parts, fmt.Sprintf("Actual: %s", strings.Join(e.Actual, ", ")))
	}

	if e.Suggestion != "" {
		parts = append(parts, fmt.Sprintf("Suggestion: %s", e.Suggestion))
	}

	return strings.Join(parts, "; ")
}
