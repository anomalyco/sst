package python

import (
	"fmt"
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
	ErrorTypeBuildFailed      ErrorType = "build_failed"
	ErrorTypeDependencyFailed ErrorType = "dependency_failed"
	ErrorTypeUVCommandFailed  ErrorType = "uv_command_failed"

	// System errors
	ErrorTypeFileSystem ErrorType = "filesystem"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeTimeout    ErrorType = "timeout"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityInfo     ErrorSeverity = "info"
	SeverityWarning  ErrorSeverity = "warning"
	SeverityError    ErrorSeverity = "error"
	SeverityCritical ErrorSeverity = "critical"
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
	return NewPythonRuntimeError(ErrorTypeLayoutDetection, SeverityError, message).
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
	return NewPythonRuntimeError(ErrorTypeHandlerNotFound, SeverityError,
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
	return NewPythonRuntimeError(ErrorTypeCacheCorrupted, SeverityWarning,
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
	err := NewPythonRuntimeError(ErrorTypeBuildFailed, SeverityError,
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
	return NewPythonRuntimeError(ErrorTypeUVCommandFailed, SeverityError,
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
	return NewPythonRuntimeError(ErrorTypeDependencyFailed, SeverityError,
		fmt.Sprintf("Failed to resolve dependency '%s'", dependency)).
		WithCause(cause).
		WithContext("dependency", dependency).
		WithSuggestion("Check if the dependency name and version are correct").
		WithSuggestion("Verify network connectivity to package repositories").
		WithRetry(60 * time.Second)
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
	var severity ErrorSeverity = SeverityError

	switch {
	case strings.Contains(errStr, "not found") || strings.Contains(errStr, "no such file"):
		errorType = ErrorTypeHandlerNotFound
	case strings.Contains(errStr, "permission"):
		errorType = ErrorTypeCachePermission
	case strings.Contains(errStr, "network") || strings.Contains(errStr, "connection"):
		errorType = ErrorTypeNetwork
		severity = SeverityWarning
	case strings.Contains(errStr, "timeout"):
		errorType = ErrorTypeTimeout
		severity = SeverityWarning
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
