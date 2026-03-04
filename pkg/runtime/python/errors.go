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
	// File resolution errors
	ErrorTypeHandlerNotFound    ErrorType = "handler_not_found"
	ErrorTypeConfigurationError ErrorType = "configuration_error"
	ErrorTypeProjectStructure   ErrorType = "project_structure"

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

// NewConfigurationError creates an error for pyproject.toml and configuration issues
func NewConfigurationError(message string, configFile string, issue string, fix string) *PythonRuntimeError {
	err := NewPythonRuntimeError(ErrorTypeConfigurationError, ErrorSeverityError, message).
		WithContext("configFile", configFile).
		WithContext("issue", issue).
		WithContext("fix", fix)

	// Add specific suggestions based on the issue type
	switch {
	case strings.Contains(strings.ToLower(issue), "missing"):
		err.WithSuggestion("Create a pyproject.toml file in your project root").
			WithSuggestion("Use 'uv init' to create a basic project structure").
			WithRecoveryAction(RecoveryAction{
				Name:        "create_pyproject",
				Description: "Create a basic pyproject.toml file",
				Automatic:   false,
				Command:     "uv init",
			})
	case strings.Contains(strings.ToLower(issue), "invalid") || strings.Contains(strings.ToLower(issue), "parse"):
		err.WithSuggestion("Check the TOML syntax in your pyproject.toml file").
			WithSuggestion("Validate your pyproject.toml using an online TOML validator").
			WithRecoveryAction(RecoveryAction{
				Name:        "validate_toml",
				Description: "Validate TOML syntax",
				Automatic:   false,
			})
	case strings.Contains(strings.ToLower(issue), "name"):
		err.WithSuggestion("Add a project name to your pyproject.toml: [project] name = \"your-project-name\"").
			WithSuggestion("Ensure the project name follows Python package naming conventions")
	case strings.Contains(strings.ToLower(issue), "dependencies"):
		err.WithSuggestion("Check the dependencies section in your pyproject.toml").
			WithSuggestion("Use 'uv add <package>' to add dependencies correctly")
	}

	return err
}

// NewHandlerNotFoundError creates an error for missing handlers with comprehensive guidance
func NewHandlerNotFoundError(handler string, searchPaths []string, suggestions []string) *PythonRuntimeError {
	message := fmt.Sprintf("Python handler '%s' not found", handler)

	err := NewPythonRuntimeError(ErrorTypeHandlerNotFound, ErrorSeverityError, message).
		WithContext("handler", handler).
		WithContext("searchPaths", searchPaths)

	// Add search paths to the error message for clarity
	if len(searchPaths) > 0 {
		err.WithContext("searchedLocations", fmt.Sprintf("Searched in: %s", strings.Join(searchPaths, ", ")))
	}

	// Add specific suggestions based on common patterns
	if len(suggestions) > 0 {
		for _, suggestion := range suggestions {
			err.WithSuggestion(suggestion)
		}
	} else {
		// Improved default suggestions with more actionable guidance
		err.WithSuggestion(fmt.Sprintf("Create the handler file: %s", handler)).
			WithSuggestion("Ensure the handler path matches your SST configuration").
			WithSuggestion("Verify the file has a .py extension and contains the expected function").
			WithSuggestion("Check that the file is in your project root or a subdirectory")
	}

	// Add recovery actions with better descriptions
	err.WithRecoveryAction(RecoveryAction{
		Name:        "create_handler",
		Description: fmt.Sprintf("Create the missing handler file: %s", handler),
		Automatic:   false,
	}).WithRecoveryAction(RecoveryAction{
		Name:        "list_python_files",
		Description: "List all Python files in the project to find existing handlers",
		Automatic:   false,
		Command:     "find . -name '*.py' -type f | head -20",
	}).WithRecoveryAction(RecoveryAction{
		Name:        "check_sst_config",
		Description: "Verify your SST configuration matches your file structure",
		Automatic:   false,
	})

	return err
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
			err.WithRetry(NetworkRetryDelay).
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
		maxRetries: DefaultMaxRetries,
		baseDelay:  DefaultRetryDelay,
		maxDelay:   DefaultMaxRetryDelay,
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
		} else if strings.Contains(errStr, "handler") || strings.Contains(errStr, ".py") {
			errorType = ErrorTypeHandlerNotFound
		} else {
			errorType = ErrorTypeHandlerNotFound
		}
	case strings.Contains(errStr, "pyproject.toml") || strings.Contains(errStr, "configuration") || strings.Contains(errStr, "toml"):
		errorType = ErrorTypeConfigurationError
	case strings.Contains(errStr, "project structure") || strings.Contains(errStr, "workspace"):
		errorType = ErrorTypeProjectStructure
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
		pythonErr.WithRetry(TransientErrorRetryDelay)
	}

	return pythonErr
}

// GenerateHandlerSuggestions creates helpful suggestions for handler not found errors
func GenerateHandlerSuggestions(handler string, projectRoot string, searchPaths []string) []string {
	var suggestions []string

	// Extract handler components
	handlerDir := filepath.Dir(handler)
	handlerBase := strings.TrimSuffix(filepath.Base(handler), filepath.Ext(handler))

	// Primary suggestion: create the exact file specified
	suggestions = append(suggestions, fmt.Sprintf("Create the handler file: %s", handler))

	// If handler doesn't have .py extension, suggest adding it
	if !strings.HasSuffix(handler, ".py") {
		suggestions = append(suggestions, fmt.Sprintf("Add .py extension: %s.py", handler))
	}

	// Suggest checking common Python project directories
	commonDirs := []string{"src", "app", "functions", "lambda", "handlers"}
	for _, dir := range commonDirs {
		suggestedPath := filepath.Join(dir, handler)
		if !strings.HasSuffix(suggestedPath, ".py") {
			suggestedPath += ".py"
		}
		suggestions = append(suggestions, fmt.Sprintf("Check if file exists in %s/: %s", dir, suggestedPath))
	}

	// If handler has directory structure, suggest alternatives
	if handlerDir != "." && handlerDir != "" {
		flatHandler := handlerBase + ".py"
		suggestions = append(suggestions, fmt.Sprintf("Try without subdirectory: %s", flatHandler))

		// Suggest creating the directory structure
		suggestions = append(suggestions, fmt.Sprintf("Create directory structure: mkdir -p %s", handlerDir))
	}

	// Add general guidance
	suggestions = append(suggestions, "Ensure your handler path in sst.config.ts matches your actual file location")
	suggestions = append(suggestions, "Verify the Python file contains the expected function (e.g., 'def handler(event, context):')")

	return suggestions
}
