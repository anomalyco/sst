package python

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewPythonRuntimeError(t *testing.T) {
	err := NewPythonRuntimeError(ErrorTypeHandlerNotFound, ErrorSeverityError, "test error")

	if err.Type != ErrorTypeHandlerNotFound {
		t.Errorf("Expected type %s, got %s", ErrorTypeHandlerNotFound, err.Type)
	}

	if err.Severity != ErrorSeverityError {
		t.Errorf("Expected severity %s, got %s", ErrorSeverityError, err.Severity)
	}

	if err.Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", err.Message)
	}

	if err.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if err.Retryable {
		t.Error("Expected error to not be retryable by default")
	}
}

func TestPythonRuntimeError_WithMethods(t *testing.T) {
	originalErr := errors.New("original error")

	err := NewPythonRuntimeError(ErrorTypeBuildFailed, ErrorSeverityError, "build failed").
		WithCause(originalErr).
		WithContext("package", "test-package").
		WithSuggestion("Check your code").
		WithRetry(5 * time.Second)

	if err.Cause != originalErr {
		t.Error("Expected cause to be set")
	}

	if err.Context["package"] != "test-package" {
		t.Error("Expected context to be set")
	}

	if len(err.Suggestions) != 1 || err.Suggestions[0] != "Check your code" {
		t.Error("Expected suggestion to be set")
	}

	if !err.Retryable {
		t.Error("Expected error to be retryable")
	}

	if err.RetryAfter != 5*time.Second {
		t.Error("Expected retry delay to be set")
	}
}

func TestPythonRuntimeError_Error(t *testing.T) {
	err := NewPythonRuntimeError(ErrorTypeBuildFailed, ErrorSeverityError, "build failed").
		WithContext("package", "test-package").
		WithSuggestion("Check your code")

	errorStr := err.Error()

	if !strings.Contains(errorStr, "[error:build_failed]") {
		t.Error("Expected error string to contain severity and type")
	}

	if !strings.Contains(errorStr, "build failed") {
		t.Error("Expected error string to contain message")
	}

	if !strings.Contains(errorStr, "package=test-package") {
		t.Error("Expected error string to contain context")
	}

	if !strings.Contains(errorStr, "Check your code") {
		t.Error("Expected error string to contain suggestions")
	}
}

func TestPythonRuntimeError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := NewPythonRuntimeError(ErrorTypeBuildFailed, ErrorSeverityError, "build failed").
		WithCause(originalErr)

	if err.Unwrap() != originalErr {
		t.Error("Expected Unwrap to return original error")
	}
}

func TestNewConfigurationErrorLegacy(t *testing.T) {
	err := NewConfigurationError("configuration failed", "pyproject.toml", "missing name", "add project name")

	if err.Type != ErrorTypeConfigurationError {
		t.Error("Expected configuration error type")
	}

	if err.Context["configFile"] != "pyproject.toml" {
		t.Error("Expected configFile context to be set")
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	if len(err.RecoveryActions) == 0 {
		t.Error("Expected recovery actions to be provided")
	}
}

func TestNewHandlerNotFoundErrorLegacy(t *testing.T) {
	searchPaths := []string{"/path1", "/path2"}
	suggestions := []string{"Create the handler file"}
	err := NewHandlerNotFoundError("handler.py", searchPaths, suggestions)

	if err.Type != ErrorTypeHandlerNotFound {
		t.Error("Expected handler not found error type")
	}

	if err.Context["handler"] != "handler.py" {
		t.Error("Expected handler context to be set")
	}

	if err.Context["searchPaths"] == nil {
		t.Error("Expected search paths context to be set")
	}
}

func TestNewBuildFailedError(t *testing.T) {
	err := NewBuildFailedError("test-package", errors.New("build error"))

	if err.Type != ErrorTypeBuildFailed {
		t.Error("Expected build failed error type")
	}

	if err.Context["package"] != "test-package" {
		t.Error("Expected package context to be set")
	}
}

func TestNewUVCommandFailedError(t *testing.T) {
	args := []string{"build", "--package=test"}
	err := NewUVCommandFailedError("uv", args, 1, "command failed")

	if err.Type != ErrorTypeUVCommandFailed {
		t.Error("Expected UV command failed error type")
	}

	if err.Context["command"] != "uv" {
		t.Error("Expected command context to be set")
	}

	if err.Context["exitCode"] != 1 {
		t.Error("Expected exit code context to be set")
	}

	if err.Context["stderr"] != "command failed" {
		t.Error("Expected stderr context to be set")
	}
}

func TestErrorRecoveryManager_RetryWithBackoff(t *testing.T) {
	erm := NewErrorRecoveryManager()

	// Test successful operation
	attempts := 0
	err := erm.RetryWithBackoff(func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected operation to succeed after retry, got error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestErrorRecoveryManager_RetryWithBackoff_NonRetryable(t *testing.T) {
	erm := NewErrorRecoveryManager()

	// Test non-retryable error
	attempts := 0
	nonRetryableErr := NewPythonRuntimeError(ErrorTypeHandlerNotFound, ErrorSeverityError, "not found")

	err := erm.RetryWithBackoff(func() error {
		attempts++
		return nonRetryableErr
	})

	if err != nonRetryableErr {
		t.Error("Expected non-retryable error to be returned immediately")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestErrorRecoveryManager_RetryWithBackoff_MaxRetries(t *testing.T) {
	erm := NewErrorRecoveryManager()

	// Test max retries exceeded
	attempts := 0
	retryableErr := NewPythonRuntimeError(ErrorTypeNetwork, ErrorSeverityWarning, "network error").
		WithRetry(1 * time.Millisecond)

	err := erm.RetryWithBackoff(func() error {
		attempts++
		return retryableErr
	})

	if err == nil {
		t.Error("Expected error after max retries exceeded")
	}

	expectedAttempts := erm.maxRetries + 1
	if attempts != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attempts)
	}
}

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		transient bool
	}{
		{
			name:      "nil error",
			err:       nil,
			transient: false,
		},
		{
			name:      "network error",
			err:       errors.New("network connection failed"),
			transient: true,
		},
		{
			name:      "timeout error",
			err:       errors.New("operation timeout"),
			transient: true,
		},
		{
			name:      "file not found error",
			err:       errors.New("file not found"),
			transient: false,
		},
		{
			name:      "temporary failure",
			err:       errors.New("temporary failure"),
			transient: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransientError(tt.err)
			if result != tt.transient {
				t.Errorf("Expected IsTransientError(%v) = %v, got %v", tt.err, tt.transient, result)
			}
		})
	}
}

func TestWrapErrorLegacy(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		context       string
		expectedType  ErrorType
		expectedRetry bool
	}{
		{
			name:          "nil error",
			err:           nil,
			context:       "test",
			expectedType:  "",
			expectedRetry: false,
		},
		{
			name:          "file not found",
			err:           errors.New("file not found"),
			context:       "test",
			expectedType:  ErrorTypeHandlerNotFound,
			expectedRetry: false,
		},
		{
			name:          "network error",
			err:           errors.New("network connection failed"),
			context:       "test",
			expectedType:  ErrorTypeNetwork,
			expectedRetry: true,
		},
		{
			name:          "permission error",
			err:           errors.New("permission denied"),
			context:       "test",
			expectedType:  ErrorTypeCachePermission,
			expectedRetry: false,
		},
		{
			name:          "timeout error",
			err:           errors.New("operation timeout"),
			context:       "test",
			expectedType:  ErrorTypeTimeout,
			expectedRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.context)

			if tt.err == nil {
				if result != nil {
					t.Error("Expected nil result for nil error")
				}
				return
			}

			if result == nil {
				t.Error("Expected non-nil result for non-nil error")
				return
			}

			if result.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, result.Type)
			}

			if result.Retryable != tt.expectedRetry {
				t.Errorf("Expected retryable %v, got %v", tt.expectedRetry, result.Retryable)
			}

			if result.Context["originalContext"] != tt.context {
				t.Error("Expected original context to be preserved")
			}
		})
	}
}

func TestWrapError_AlreadyPythonRuntimeError(t *testing.T) {
	originalErr := NewPythonRuntimeError(ErrorTypeBuildFailed, ErrorSeverityError, "build failed")
	result := WrapError(originalErr, "test")

	if result != originalErr {
		t.Error("Expected WrapError to return the same PythonRuntimeError")
	}
}
