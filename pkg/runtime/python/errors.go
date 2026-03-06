package python

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeHandlerNotFound    ErrorType = "handler_not_found"
	ErrorTypeConfigurationError ErrorType = "configuration_error"
	ErrorTypeBuildFailed        ErrorType = "build_failed"
	ErrorTypeUVCommandFailed    ErrorType = "uv_command_failed"
	ErrorTypeFileSystem         ErrorType = "filesystem"
)

// PythonRuntimeError represents a structured error with context
type PythonRuntimeError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *PythonRuntimeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// NewConfigurationError creates an error for configuration issues
func NewConfigurationError(message string, configFile string, issue string, fix string) *PythonRuntimeError {
	return &PythonRuntimeError{
		Type:    ErrorTypeConfigurationError,
		Message: fmt.Sprintf("%s (file: %s, issue: %s, fix: %s)", message, configFile, issue, fix),
	}
}

// NewHandlerNotFoundError creates an error for missing handlers
func NewHandlerNotFoundError(handler string, searchPaths []string, suggestions []string) *PythonRuntimeError {
	return &PythonRuntimeError{
		Type:    ErrorTypeHandlerNotFound,
		Message: fmt.Sprintf("handler '%s' not found (searched: %s)", handler, strings.Join(searchPaths, ", ")),
	}
}

// NewBuildFailedError creates an error for build failures
func NewBuildFailedError(packageName string, cause error) *PythonRuntimeError {
	return &PythonRuntimeError{
		Type:    ErrorTypeBuildFailed,
		Message: fmt.Sprintf("failed to build package '%s'", packageName),
		Cause:   cause,
	}
}

// NewUVCommandFailedError creates an error for UV command failures
func NewUVCommandFailedError(command string, args []string, exitCode int, stderr string) *PythonRuntimeError {
	return &PythonRuntimeError{
		Type:    ErrorTypeUVCommandFailed,
		Message: fmt.Sprintf("uv %s failed (exit %d): %s", strings.Join(args, " "), exitCode, stderr),
	}
}

// WrapError wraps a standard error into a PythonRuntimeError
func WrapError(err error, context string) *PythonRuntimeError {
	if err == nil {
		return nil
	}
	if pythonErr, ok := err.(*PythonRuntimeError); ok {
		return pythonErr
	}
	return &PythonRuntimeError{
		Type:    ErrorTypeFileSystem,
		Message: fmt.Sprintf("%s: %v", context, err),
		Cause:   err,
	}
}

// GenerateHandlerSuggestions creates suggestions for handler not found errors
func GenerateHandlerSuggestions(handler string, projectRoot string, searchPaths []string) []string {
	suggestions := []string{
		fmt.Sprintf("Create the handler file: %s", handler),
	}
	if !strings.HasSuffix(handler, ".py") {
		suggestions = append(suggestions, fmt.Sprintf("Add .py extension: %s.py", handler))
	}
	handlerDir := filepath.Dir(handler)
	if handlerDir != "." && handlerDir != "" {
		suggestions = append(suggestions, fmt.Sprintf("Try without subdirectory: %s.py", filepath.Base(strings.TrimSuffix(handler, filepath.Ext(handler)))))
	}
	return suggestions
}
