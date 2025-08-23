package python

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ExtractionValidator validates tar.gz extraction operations
type ExtractionValidator struct {
	outputDir string
	logger    *slog.Logger
}

// NewExtractionValidator creates a new extraction validator
func NewExtractionValidator(outputDir string) *ExtractionValidator {
	return &ExtractionValidator{
		outputDir: outputDir,
		logger:    slog.Default(),
	}
}

// ValidateExtraction validates that a tar.gz file was extracted successfully
func (ev *ExtractionValidator) ValidateExtraction(tarGzFile string) error {
	ev.logger.Info("validating extraction", "tarGzFile", tarGzFile)

	// Get expected directory name from tar.gz file
	fileName := filepath.Base(tarGzFile)
	dirName := strings.TrimSuffix(fileName, ".tar.gz")
	extractedDir := filepath.Join(ev.outputDir, dirName)

	ev.logger.Debug("checking extracted directory",
		"tarGzFile", tarGzFile,
		"expectedDir", extractedDir)

	// Check if extracted directory exists
	info, err := os.Stat(extractedDir)
	if err != nil {
		if os.IsNotExist(err) {
			// List what was actually created to help with debugging
			entries, listErr := os.ReadDir(ev.outputDir)
			var actualFiles []string
			if listErr == nil {
				for _, entry := range entries {
					actualFiles = append(actualFiles, entry.Name())
				}
			}

			return &BuildValidationError{
				Stage:      "extract",
				Command:    fmt.Sprintf("tar -xzf %s -C %s", fileName, ev.outputDir),
				Files:      []string{tarGzFile},
				Expected:   []string{extractedDir},
				Actual:     actualFiles,
				Suggestion: fmt.Sprintf("Tar extraction of %s failed to create expected directory %s. Check if the tar.gz file is valid and the extraction command has proper permissions.", fileName, dirName),
			}
		}
		return fmt.Errorf("failed to stat extracted directory %s: %w", extractedDir, err)
	}

	if !info.IsDir() {
		return &BuildValidationError{
			Stage:      "extract",
			Command:    fmt.Sprintf("tar -xzf %s -C %s", fileName, ev.outputDir),
			Files:      []string{tarGzFile},
			Expected:   []string{"directory"},
			Actual:     []string{"file"},
			Suggestion: fmt.Sprintf("Expected %s to be a directory after extraction, but found a file instead. The tar.gz file may be corrupted.", extractedDir),
		}
	}

	// Validate directory contents
	entries, err := os.ReadDir(extractedDir)
	if err != nil {
		return fmt.Errorf("failed to read extracted directory %s: %w", extractedDir, err)
	}

	if len(entries) == 0 {
		return &BuildValidationError{
			Stage:      "extract",
			Command:    fmt.Sprintf("tar -xzf %s -C %s", fileName, ev.outputDir),
			Files:      []string{tarGzFile},
			Expected:   []string{"non-empty directory"},
			Actual:     []string{"empty directory"},
			Suggestion: fmt.Sprintf("Extracted directory %s is empty. The tar.gz file may be corrupted, empty, or the extraction process failed to copy files.", extractedDir),
		}
	}

	var extractedFiles []string
	var pythonFiles []string
	var directories []string
	for _, entry := range entries {
		extractedFiles = append(extractedFiles, entry.Name())
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		} else if strings.HasSuffix(entry.Name(), ".py") {
			pythonFiles = append(pythonFiles, entry.Name())
		}
	}

	ev.logger.Info("extraction validation successful",
		"tarGzFile", tarGzFile,
		"extractedDir", extractedDir,
		"totalFiles", len(extractedFiles),
		"pythonFiles", len(pythonFiles),
		"directories", len(directories),
		"contents", extractedFiles)

	// Validate that we have some Python-related content
	if len(pythonFiles) == 0 && len(directories) == 0 {
		ev.logger.Warn("extracted directory contains no Python files or subdirectories",
			"extractedDir", extractedDir,
			"contents", extractedFiles)
	}

	return nil
}

// ValidateExtractionContents performs detailed validation of extracted directory contents
func (ev *ExtractionValidator) ValidateExtractionContents(extractedDir string) error {
	ev.logger.Info("validating extraction contents", "extractedDir", extractedDir)

	// Check if directory exists
	info, err := os.Stat(extractedDir)
	if err != nil {
		return fmt.Errorf("extracted directory does not exist: %s", extractedDir)
	}

	if !info.IsDir() {
		return fmt.Errorf("expected directory but found file: %s", extractedDir)
	}

	// Read directory contents
	entries, err := os.ReadDir(extractedDir)
	if err != nil {
		return fmt.Errorf("failed to read extracted directory %s: %w", extractedDir, err)
	}

	// Analyze contents
	var files []string
	var directories []string
	var pythonFiles []string
	var hasSetupPy bool
	var hasPyProjectToml bool
	var hasSrcDir bool

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			directories = append(directories, name)
			if name == "src" {
				hasSrcDir = true
			}
		} else {
			files = append(files, name)
			if strings.HasSuffix(name, ".py") {
				pythonFiles = append(pythonFiles, name)
			}
			if name == "setup.py" {
				hasSetupPy = true
			}
			if name == "pyproject.toml" {
				hasPyProjectToml = true
			}
		}
	}

	ev.logger.Info("extraction contents analysis",
		"extractedDir", extractedDir,
		"totalEntries", len(entries),
		"files", len(files),
		"directories", len(directories),
		"pythonFiles", len(pythonFiles),
		"hasSetupPy", hasSetupPy,
		"hasPyProjectToml", hasPyProjectToml,
		"hasSrcDir", hasSrcDir,
		"allFiles", files,
		"allDirectories", directories)

	// Check for empty directory
	if len(entries) == 0 {
		return fmt.Errorf("extracted directory is empty: %s", extractedDir)
	}

	// Check that we have some files (not just directories)
	if len(files) == 0 && len(directories) > 0 {
		// Only directories, no files - this might be problematic
		ev.logger.Warn("extracted directory contains only subdirectories, no files",
			"extractedDir", extractedDir,
			"directories", directories)
	}

	// Validate that we have expected Python package structure
	if !hasSetupPy && !hasPyProjectToml {
		ev.logger.Warn("no setup.py or pyproject.toml found in extracted directory",
			"extractedDir", extractedDir,
			"files", files)
	}

	// If there's a src directory, validate its contents
	if hasSrcDir {
		srcDir := filepath.Join(extractedDir, "src")
		if err := ev.validateSrcDirectory(srcDir); err != nil {
			return fmt.Errorf("src directory validation failed: %w", err)
		}
	}

	return nil
}

// validateSrcDirectory validates the contents of a src directory
func (ev *ExtractionValidator) validateSrcDirectory(srcDir string) error {
	ev.logger.Debug("validating src directory", "srcDir", srcDir)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read src directory %s: %w", srcDir, err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("src directory is empty: %s", srcDir)
	}

	var packageDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			packageDirs = append(packageDirs, entry.Name())
		}
	}

	ev.logger.Debug("src directory contents",
		"srcDir", srcDir,
		"packageDirs", packageDirs,
		"count", len(packageDirs))

	if len(packageDirs) == 0 {
		return fmt.Errorf("no package directories found in src directory: %s", srcDir)
	}

	return nil
}

// getExpectedDirectoryName extracts the expected directory name from a tar.gz file path
func (ev *ExtractionValidator) getExpectedDirectoryName(tarGzFile string) string {
	fileName := filepath.Base(tarGzFile)
	return strings.TrimSuffix(fileName, ".tar.gz")
}

// validateDirectoryContents validates the contents of a directory
func (ev *ExtractionValidator) validateDirectoryContents(dirPath string) error {
	return ev.ValidateExtractionContents(dirPath)
}

// ValidateMultipleExtractions validates multiple tar.gz extractions
func (ev *ExtractionValidator) ValidateMultipleExtractions(tarGzFiles []string) error {
	ev.logger.Info("validating multiple extractions", "count", len(tarGzFiles))

	var errors []string
	for i, file := range tarGzFiles {
		ev.logger.Debug("validating extraction",
			"file", file,
			"index", i+1,
			"total", len(tarGzFiles))

		if err := ev.ValidateExtraction(file); err != nil {
			errors = append(errors, fmt.Sprintf("file %s: %v", file, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("extraction validation failed for %d files: %s", len(errors), strings.Join(errors, "; "))
	}

	ev.logger.Info("all extractions validated successfully", "count", len(tarGzFiles))
	return nil
}
