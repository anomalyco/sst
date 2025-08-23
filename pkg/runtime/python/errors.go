package python

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// Layout detection errors
	ErrorTypeLayoutDetection  ErrorType = "layout_detection"
	ErrorTypeHandlerNotFound  ErrorType = "handler_not_found"
	ErrorTypeWorkspaceInvalid ErrorType = "workspace_invalid"

	// Cache errors
	ErrorTypeCacheCorrupted  ErrorType = "cache_corrupted"
	ErrorTypeCachePermission ErrorType = "cache_permission"
	ErrorTypeCacheInvalid    ErrorType = "cache_invalid"

	// Build errors
	ErrorTypeBuildFailed        ErrorType = "build_failed"
	ErrorTypeDependencyFailed   ErrorType = "dependency_failed"
	ErrorTypeUVCommandFailed    ErrorType = "uv_command_failed"
	ErrorTypeBuildOutputMissing ErrorType = "build_output_missing"
	ErrorTypeExtractionFailed   ErrorType = "extraction_failed"
	ErrorTypeModuleMoveFailed   ErrorType = "module_move_failed"
	ErrorTypeValidationFailed   ErrorType = "validation_failed"
	ErrorTypeArtifactOversized  ErrorType = "artifact_oversized"
	ErrorTypeModuleMissing      ErrorType = "module_missing"

	// System errors
	ErrorTypeFileSystem ErrorType = "filesystem"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeTimeout    ErrorType = "timeout"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityInfo     ErrorSeverity = "info"
	ErrorSeverityWarning  ErrorSeverity = "warning"
	ErrorSeverityError    ErrorSeverity = "error"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// PythonRuntimeError represents a structured error with context and recovery information
type PythonRuntimeError struct {
	// Type categorizes the error
	Type ErrorType `json:"type"`

	// Severity indicates how critical the error is
	Severity ErrorSeverity `json:"severity"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// Context provides additional information about the error
	Context map[string]interface{} `json:"context,omitempty"`

	// Cause is the underlying error that caused this error
	Cause error `json:"-"`

	// Suggestions provides actionable advice for fixing the error
	Suggestions []string `json:"suggestions,omitempty"`

	// RecoveryActions lists possible recovery strategies
	RecoveryActions []RecoveryAction `json:"recoveryActions,omitempty"`

	// Timestamp when the error occurred
	Timestamp time.Time `json:"timestamp"`

	// Retryable indicates if this error can be retried
	Retryable bool `json:"retryable"`

	// RetryAfter suggests when to retry (for retryable errors)
	RetryAfter time.Duration `json:"retryAfter,omitempty"`
}

// RecoveryAction represents a possible recovery strategy
type RecoveryAction struct {
	// Name is a short identifier for the action
	Name string `json:"name"`

	// Description explains what the action does
	Description string `json:"description"`

	// Automatic indicates if this action can be performed automatically
	Automatic bool `json:"automatic"`

	// Command is the command to run (if applicable)
	Command string `json:"command,omitempty"`
}

// Error implements the error interface
func (e *PythonRuntimeError) Error() string {
	var parts []string

	// Add severity and type
	parts = append(parts, fmt.Sprintf("[%s:%s]", e.Severity, e.Type))

	// Add main message
	parts = append(parts, e.Message)

	// Add context if available
	if len(e.Context) > 0 {
		var contextParts []string
		for key, value := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
		}
		parts = append(parts, fmt.Sprintf("Context: %s", strings.Join(contextParts, ", ")))
	}

	// Add suggestions if available
	if len(e.Suggestions) > 0 {
		parts = append(parts, fmt.Sprintf("Suggestions: %s", strings.Join(e.Suggestions, "; ")))
	}

	return strings.Join(parts, " | ")
}

// Unwrap returns the underlying cause
func (e *PythonRuntimeError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns whether this error can be retried
func (e *PythonRuntimeError) IsRetryable() bool {
	return e.Retryable
}

// GetRetryAfter returns the suggested retry delay
func (e *PythonRuntimeError) GetRetryAfter() time.Duration {
	return e.RetryAfter
}

// NewPythonRuntimeError creates a new structured error
func NewPythonRuntimeError(errorType ErrorType, severity ErrorSeverity, message string) *PythonRuntimeError {
	return &PythonRuntimeError{
		Type:      errorType,
		Severity:  severity,
		Message:   message,
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
		Retryable: false,
	}
}

// WithCause adds the underlying cause
func (e *PythonRuntimeError) WithCause(cause error) *PythonRuntimeError {
	e.Cause = cause
	return e
}

// WithContext adds context information
func (e *PythonRuntimeError) WithContext(key string, value interface{}) *PythonRuntimeError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithSuggestion adds a suggestion for fixing the error
func (e *PythonRuntimeError) WithSuggestion(suggestion string) *PythonRuntimeError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithRecoveryAction adds a recovery action
func (e *PythonRuntimeError) WithRecoveryAction(action RecoveryAction) *PythonRuntimeError {
	e.RecoveryActions = append(e.RecoveryActions, action)
	return e
}

// WithRetry marks the error as retryable with a delay
func (e *PythonRuntimeError) WithRetry(retryAfter time.Duration) *PythonRuntimeError {
	e.Retryable = true
	e.RetryAfter = retryAfter
	return e
}

// Common error constructors

// NewLayoutDetectionError creates an error for layout detection failures
func NewLayoutDetectionError(message string, handler string) *PythonRuntimeError {
	return NewPythonRuntimeError(ErrorTypeLayoutDetection, ErrorSeverityError, message).
		WithContext("handler", handler).
		WithSuggestion("Check if the handler path is correct and the Python file exists").
		WithSuggestion("Ensure your project has a valid pyproject.toml file").
		WithRecoveryAction(RecoveryAction{
			Name:        "check_file",
			Description: "Verify the handler file exists and is accessible",
			Automatic:   false,
		})
}

// NewHandlerNotFoundError creates an error for missing handlers
func NewHandlerNotFoundError(handler string, searchPaths []string) *PythonRuntimeError {
	return NewPythonRuntimeError(ErrorTypeHandlerNotFound, ErrorSeverityError,
		fmt.Sprintf("Handler '%s' not found", handler)).
		WithContext("handler", handler).
		WithContext("searchPaths", searchPaths).
		WithSuggestion("Check if the handler file exists in the expected location").
		WithSuggestion("Verify the handler path is relative to your project root").
		WithRecoveryAction(RecoveryAction{
			Name:        "create_handler",
			Description: "Create the missing handler file",
			Automatic:   false,
		})
}

// NewCacheCorruptedError creates an error for corrupted cache
func NewCacheCorruptedError(cacheDir string, cause error) *PythonRuntimeError {
	return NewPythonRuntimeError(ErrorTypeCacheCorrupted, ErrorSeverityWarning,
		"Build cache is corrupted and will be cleared").
		WithCause(cause).
		WithContext("cacheDir", cacheDir).
		WithSuggestion("The cache will be automatically cleared and rebuilt").
		WithRecoveryAction(RecoveryAction{
			Name:        "clear_cache",
			Description: "Clear the corrupted cache directory",
			Automatic:   true,
			Command:     fmt.Sprintf("rm -rf %s", cacheDir),
		})
}

// NewBuildFailedError creates an error for build failures
func NewBuildFailedError(packageName string, cause error) *PythonRuntimeError {
	err := NewPythonRuntimeError(ErrorTypeBuildFailed, ErrorSeverityError,
		fmt.Sprintf("Failed to build package '%s'", packageName)).
		WithCause(cause).
		WithContext("package", packageName)

	// Add specific suggestions based on the cause
	if cause != nil {
		causeStr := cause.Error()
		if strings.Contains(causeStr, "permission") {
			err.WithSuggestion("Check file permissions in the build directory")
		}
		if strings.Contains(causeStr, "space") {
			err.WithSuggestion("Check available disk space")
		}
		if strings.Contains(causeStr, "network") {
			err.WithRetry(30 * time.Second).
				WithSuggestion("Check network connectivity and try again")
		}
	}

	return err
}

// NewUVCommandFailedError creates an error for UV command failures
func NewUVCommandFailedError(command string, args []string, exitCode int, stderr string) *PythonRuntimeError {
	return NewPythonRuntimeError(ErrorTypeUVCommandFailed, ErrorSeverityError,
		fmt.Sprintf("UV command failed: %s", command)).
		WithContext("command", command).
		WithContext("args", args).
		WithContext("exitCode", exitCode).
		WithContext("stderr", stderr).
		WithSuggestion("Check if UV is properly installed and accessible").
		WithSuggestion("Verify your pyproject.toml configuration is valid").
		WithRecoveryAction(RecoveryAction{
			Name:        "check_uv",
			Description: "Verify UV installation",
			Automatic:   false,
			Command:     "uv --version",
		})
}

// NewDependencyFailedError creates an error for dependency failures
func NewDependencyFailedError(dependency string, cause error) *PythonRuntimeError {
	return NewPythonRuntimeError(ErrorTypeDependencyFailed, ErrorSeverityError,
		fmt.Sprintf("Failed to resolve dependency '%s'", dependency)).
		WithCause(cause).
		WithContext("dependency", dependency).
		WithSuggestion("Check if the dependency name and version are correct").
		WithSuggestion("Verify network connectivity to package repositories").
		WithRetry(60 * time.Second)
}

// NewBuildOutputMissingError creates an error for missing build outputs
func NewBuildOutputMissingError(outputDir string, expectedFiles []string, actualFiles []string) *PythonRuntimeError {
	return NewPythonRuntimeError(ErrorTypeBuildOutputMissing, ErrorSeverityError,
		"Build completed but expected output files are missing").
		WithContext("outputDir", outputDir).
		WithContext("expectedFiles", expectedFiles).
		WithContext("actualFiles", actualFiles).
		WithSuggestion("Check if the 'uv build' command completed successfully").
		WithSuggestion("Verify your pyproject.toml configuration includes all necessary packages").
		WithSuggestion("Ensure your project structure follows Python packaging conventions").
		WithRecoveryAction(RecoveryAction{
			Name:        "rebuild_clean",
			Description: "Clean build directory and rebuild from scratch",
			Automatic:   false,
			Command:     "uv build --clean --all --sdist",
		})
}

// NewExtractionFailedError creates an error for tar.gz extraction failures
func NewExtractionFailedError(tarFile string, outputDir string, cause error) *PythonRuntimeError {
	err := NewPythonRuntimeError(ErrorTypeExtractionFailed, ErrorSeverityError,
		fmt.Sprintf("Failed to extract tar.gz file: %s", filepath.Base(tarFile))).
		WithCause(cause).
		WithContext("tarFile", tarFile).
		WithContext("outputDir", outputDir)

	// Add specific suggestions based on the cause
	if cause != nil {
		causeStr := strings.ToLower(cause.Error())
		if strings.Contains(causeStr, "permission") {
			err.WithSuggestion("Check write permissions in the output directory").
				WithSuggestion("Ensure the build process has sufficient privileges")
		} else if strings.Contains(causeStr, "space") {
			err.WithSuggestion("Check available disk space in the output directory").
				WithSuggestion("Clean up temporary files to free disk space")
		} else if strings.Contains(causeStr, "corrupted") || strings.Contains(causeStr, "invalid") {
			err.WithSuggestion("The tar.gz file may be corrupted - try rebuilding the package").
				WithRecoveryAction(RecoveryAction{
					Name:        "rebuild_package",
					Description: "Rebuild the corrupted package",
					Automatic:   false,
				})
		} else if strings.Contains(causeStr, "not found") {
			err.WithSuggestion("Verify the tar.gz file exists and is accessible").
				WithSuggestion("Check if the build process completed successfully")
		}
	}

	return err.WithRecoveryAction(RecoveryAction{
		Name:        "verify_tar",
		Description: "Verify tar.gz file integrity",
		Automatic:   false,
		Command:     fmt.Sprintf("tar -tzf %s", tarFile),
	})
}

// NewModuleMoveFailedError creates an error for module movement failures
func NewModuleMoveFailedError(sourceDir string, targetDir string, moduleName string, cause error) *PythonRuntimeError {
	err := NewPythonRuntimeError(ErrorTypeModuleMoveFailed, ErrorSeverityError,
		fmt.Sprintf("Failed to move module '%s' to target location", moduleName)).
		WithCause(cause).
		WithContext("sourceDir", sourceDir).
		WithContext("targetDir", targetDir).
		WithContext("moduleName", moduleName)

	// Add specific suggestions based on the cause
	if cause != nil {
		causeStr := strings.ToLower(cause.Error())
		if strings.Contains(causeStr, "permission") {
			err.WithSuggestion("Check write permissions in the target directory").
				WithSuggestion("Ensure the build process has sufficient privileges")
		} else if strings.Contains(causeStr, "cross-device") {
			err.WithSuggestion("Cross-device move detected - this will be handled automatically").
				WithContext("fallbackStrategy", "copy_and_remove")
		} else if strings.Contains(causeStr, "directory not empty") {
			err.WithSuggestion("Target directory exists and is not empty").
				WithSuggestion("The existing directory will be removed and replaced")
		} else if strings.Contains(causeStr, "space") {
			err.WithSuggestion("Check available disk space in the target directory")
		}
	}

	return err.WithRecoveryAction(RecoveryAction{
		Name:        "verify_paths",
		Description: "Verify source and target paths are accessible",
		Automatic:   false,
	})
}

// NewValidationFailedError creates an error for artifact validation failures
func NewValidationFailedError(validationType string, artifactDir string, details map[string]interface{}) *PythonRuntimeError {
	return NewPythonRuntimeError(ErrorTypeValidationFailed, ErrorSeverityError,
		fmt.Sprintf("Artifact validation failed: %s", validationType)).
		WithContext("validationType", validationType).
		WithContext("artifactDir", artifactDir).
		WithContext("details", details).
		WithSuggestion("Check the build logs for more details about what went wrong").
		WithSuggestion("Verify your project structure and dependencies are correct").
		WithRecoveryAction(RecoveryAction{
			Name:        "inspect_artifact",
			Description: "Inspect the artifact directory contents",
			Automatic:   false,
			Command:     fmt.Sprintf("ls -la %s", artifactDir),
		})
}

// NewArtifactOversizedError creates an error for oversized deployment artifacts
func NewArtifactOversizedError(artifactDir string, actualSize int64, maxSize int64, excessiveContent []string) *PythonRuntimeError {
	return NewPythonRuntimeError(ErrorTypeArtifactOversized, ErrorSeverityError,
		fmt.Sprintf("Deployment artifact is too large: %d bytes (max: %d bytes)", actualSize, maxSize)).
		WithContext("artifactDir", artifactDir).
		WithContext("actualSize", actualSize).
		WithContext("maxSize", maxSize).
		WithContext("excessiveContent", excessiveContent).
		WithSuggestion("Remove unnecessary files from your project (build cache, .git, etc.)").
		WithSuggestion("Check if entire project directories are being included incorrectly").
		WithSuggestion("Consider using .gitignore patterns to exclude non-essential files").
		WithRecoveryAction(RecoveryAction{
			Name:        "analyze_size",
			Description: "Analyze artifact contents by size",
			Automatic:   false,
			Command:     fmt.Sprintf("du -sh %s/*", artifactDir),
		}).
		WithRecoveryAction(RecoveryAction{
			Name:        "clean_artifact",
			Description: "Remove common unnecessary files",
			Automatic:   true,
		})
}

// NewModuleMissingError creates an error for missing Python modules
func NewModuleMissingError(moduleName string, handlerPath string, artifactDir string, expectedPaths []string) *PythonRuntimeError {
	return NewPythonRuntimeError(ErrorTypeModuleMissing, ErrorSeverityError,
		fmt.Sprintf("Required Python module '%s' is missing from deployment artifact", moduleName)).
		WithContext("moduleName", moduleName).
		WithContext("handlerPath", handlerPath).
		WithContext("artifactDir", artifactDir).
		WithContext("expectedPaths", expectedPaths).
		WithSuggestion("Check if the module was properly built and extracted").
		WithSuggestion("Verify your pyproject.toml includes the module in the build").
		WithSuggestion("Ensure the handler path matches your project structure").
		WithRecoveryAction(RecoveryAction{
			Name:        "list_modules",
			Description: "List all Python modules in the artifact",
			Automatic:   false,
			Command:     fmt.Sprintf("find %s -name '*.py' -type f", artifactDir),
		}).
		WithRecoveryAction(RecoveryAction{
			Name:        "rebuild_modules",
			Description: "Rebuild all modules from scratch",
			Automatic:   false,
		})
}

// ErrorRecoveryManager handles error recovery strategies
type ErrorRecoveryManager struct {
	// maxRetries is the maximum number of retry attempts
	maxRetries int

	// baseDelay is the base delay for exponential backoff
	baseDelay time.Duration

	// maxDelay is the maximum delay between retries
	maxDelay time.Duration
}

// NewErrorRecoveryManager creates a new error recovery manager
func NewErrorRecoveryManager() *ErrorRecoveryManager {
	return &ErrorRecoveryManager{
		maxRetries: 3,
		baseDelay:  1 * time.Second,
		maxDelay:   30 * time.Second,
	}
}

// RetryWithBackoff implements exponential backoff retry logic
func (erm *ErrorRecoveryManager) RetryWithBackoff(operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= erm.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := erm.baseDelay * time.Duration(1<<uint(attempt-1))
			if delay > erm.maxDelay {
				delay = erm.maxDelay
			}

			time.Sleep(delay)
		}

		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if pythonErr, ok := err.(*PythonRuntimeError); ok {
			if !pythonErr.IsRetryable() {
				return err // Not retryable, fail immediately
			}

			// Use custom retry delay if specified
			if pythonErr.GetRetryAfter() > 0 {
				time.Sleep(pythonErr.GetRetryAfter())
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", erm.maxRetries+1, lastErr)
}

// RecoverFromError attempts to recover from an error using available recovery actions
func (erm *ErrorRecoveryManager) RecoverFromError(err *PythonRuntimeError) error {
	for _, action := range err.RecoveryActions {
		if action.Automatic {
			if err := erm.executeRecoveryAction(action); err != nil {
				return fmt.Errorf("recovery action '%s' failed: %w", action.Name, err)
			}
		}
	}
	return nil
}

// executeRecoveryAction executes a recovery action
func (erm *ErrorRecoveryManager) executeRecoveryAction(action RecoveryAction) error {
	switch action.Name {
	case "clear_cache":
		// This would be implemented by the specific component
		return nil
	case "check_uv":
		// This would verify UV installation
		return nil
	default:
		return fmt.Errorf("unknown recovery action: %s", action.Name)
	}
}

// IsTransientError checks if an error is likely transient and retryable
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	transientPatterns := []string{
		"network",
		"timeout",
		"connection",
		"temporary",
		"busy",
		"locked",
		"unavailable",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// WrapError wraps a standard error into a PythonRuntimeError with appropriate type detection
func WrapError(err error, context string) *PythonRuntimeError {
	if err == nil {
		return nil
	}

	// If it's already a PythonRuntimeError, return as-is
	if pythonErr, ok := err.(*PythonRuntimeError); ok {
		return pythonErr
	}

	errStr := strings.ToLower(err.Error())

	// Detect error type based on error message
	var errorType ErrorType
	var severity ErrorSeverity = ErrorSeverityError

	switch {
	case strings.Contains(errStr, "not found") || strings.Contains(errStr, "no such file"):
		if strings.Contains(errStr, "tar.gz") || strings.Contains(errStr, "build output") {
			errorType = ErrorTypeBuildOutputMissing
		} else if strings.Contains(errStr, "module") {
			errorType = ErrorTypeModuleMissing
		} else {
			errorType = ErrorTypeHandlerNotFound
		}
	case strings.Contains(errStr, "permission"):
		errorType = ErrorTypeCachePermission
	case strings.Contains(errStr, "network") || strings.Contains(errStr, "connection"):
		errorType = ErrorTypeNetwork
		severity = ErrorSeverityWarning
	case strings.Contains(errStr, "timeout"):
		errorType = ErrorTypeTimeout
		severity = ErrorSeverityWarning
	case strings.Contains(errStr, "extract") || strings.Contains(errStr, "tar"):
		errorType = ErrorTypeExtractionFailed
	case strings.Contains(errStr, "move") || strings.Contains(errStr, "rename"):
		errorType = ErrorTypeModuleMoveFailed
	case strings.Contains(errStr, "validation") || strings.Contains(errStr, "validate"):
		errorType = ErrorTypeValidationFailed
	case strings.Contains(errStr, "oversized") || strings.Contains(errStr, "too large"):
		errorType = ErrorTypeArtifactOversized
	case strings.Contains(errStr, "build") || strings.Contains(errStr, "compile"):
		errorType = ErrorTypeBuildFailed
	default:
		errorType = ErrorTypeFileSystem
	}

	pythonErr := NewPythonRuntimeError(errorType, severity, err.Error()).
		WithCause(err).
		WithContext("originalContext", context)

	// Add retry capability for transient errors
	if IsTransientError(err) {
		pythonErr.WithRetry(5 * time.Second)
	}

	return pythonErr
}
