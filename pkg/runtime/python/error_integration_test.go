package python

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

func TestIncrementalBuilder_ErrorHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "error_handling_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create incremental builder
	config := IncrementalBuilderConfig{
		CacheDir:                filepath.Join(tempDir, "cache"),
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            1024 * 1024,
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Test with invalid handler (should trigger layout detection error)
	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Handler:    "nonexistent/handler.py",
		CfgPath:    tempDir,
		Properties: []byte(`{"architecture": "x86_64"}`),
	}

	// Create the output directory that BuildInput.Out() will return
	outputDir := input.Out()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	ctx := context.Background()
	_, err = builder.Build(ctx, input)

	// Verify we get a structured error
	if err == nil {
		t.Fatal("Expected error for invalid handler, got nil")
	}

	// Check if it's a PythonRuntimeError
	pythonErr, ok := err.(*PythonRuntimeError)
	if !ok {
		t.Fatalf("Expected PythonRuntimeError, got %T: %v", err, err)
	}

	// Verify error properties
	if pythonErr.Type != ErrorTypeLayoutDetection {
		t.Errorf("Expected error type %s, got %s", ErrorTypeLayoutDetection, pythonErr.Type)
	}

	if pythonErr.Severity != SeverityError {
		t.Errorf("Expected severity %s, got %s", SeverityError, pythonErr.Severity)
	}

	// Verify context is present
	if pythonErr.Context["handler"] != input.Handler {
		t.Error("Expected handler context to be set")
	}

	// Verify suggestions are present
	if len(pythonErr.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	t.Logf("Error handled correctly: %v", pythonErr)
}

func TestIncrementalBuilder_ErrorRecovery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "error_recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create incremental builder
	config := IncrementalBuilderConfig{
		CacheDir:                filepath.Join(tempDir, "cache"),
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            1024 * 1024,
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Test cache corruption recovery
	corruptedErr := NewCacheCorruptedError(config.CacheDir, os.ErrNotExist)
	recoveryErr := builder.RecoverFromError(corruptedErr)

	if recoveryErr != nil {
		t.Errorf("Expected successful recovery from cache corruption, got: %v", recoveryErr)
	}

	// Test build failure recovery
	buildErr := NewBuildFailedError("test-package", os.ErrPermission)
	recoveryErr = builder.RecoverFromError(buildErr)

	// Build failures should return the original error (no automatic recovery)
	if recoveryErr != buildErr {
		t.Error("Expected build failure recovery to return original error")
	}

	// Test layout detection failure recovery
	layoutErr := NewLayoutDetectionError("layout failed", "handler.py")
	recoveryErr = builder.RecoverFromError(layoutErr)

	// Layout failures should return the original error (no automatic recovery for now)
	if recoveryErr != layoutErr {
		t.Error("Expected layout detection failure recovery to return original error")
	}
}

func TestDependencyCache_ErrorHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dependency_cache_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create build cache
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          filepath.Join(tempDir, "build"),
		MaxAge:            time.Hour,
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create dependency cache
	cache, err := NewDependencyCache(DependencyCacheConfig{
		CacheDir:              filepath.Join(tempDir, "deps"),
		BuildCache:            buildCache,
		MaxCacheSize:          1024 * 1024,
		MaxCacheAge:           time.Hour,
		EnableSharedCache:     true,
		EnableIntegrityChecks: true,
	})
	if err != nil {
		t.Fatalf("Failed to create dependency cache: %v", err)
	}

	// Test with non-existent requirements file
	_, err = cache.GetCachedDependencies("nonexistent.txt", "x86_64", tempDir)

	// Verify we get a structured error
	if err == nil {
		t.Fatal("Expected error for non-existent requirements file, got nil")
	}

	// Check if it's a PythonRuntimeError
	pythonErr, ok := err.(*PythonRuntimeError)
	if !ok {
		t.Fatalf("Expected PythonRuntimeError, got %T: %v", err, err)
	}

	// Verify error has context and suggestions
	if len(pythonErr.Context) == 0 {
		t.Error("Expected error context to be provided")
	}

	if len(pythonErr.Suggestions) == 0 {
		t.Error("Expected error suggestions to be provided")
	}

	t.Logf("Dependency cache error handled correctly: %v", pythonErr)
}

func TestUVCommandRunner_ErrorHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "uv_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create build cache
	buildCache, err := NewBuildCache(BuildCacheConfig{
		CacheDir:          filepath.Join(tempDir, "build"),
		MaxAge:            time.Hour,
		MaxSize:           100,
		EnablePersistence: false,
	})
	if err != nil {
		t.Fatalf("Failed to create build cache: %v", err)
	}

	// Create UV runner
	runner := NewUvCommandRunner(UvCommandRunnerConfig{
		BuildCache:           buildCache,
		EnableCaching:        true,
		EnableProgressReport: true,
		CommandTimeout:       5 * time.Second,
		MaxRetries:           1,
		RetryDelay:           100 * time.Millisecond,
	})

	// Test with invalid command (should fail)
	buildCmd := &UvBuildCommand{
		PackageName: "nonexistent-package",
		PackageDir:  "/nonexistent/path",
		OutputDir:   tempDir,
		BuildType:   "sdist",
	}

	ctx := context.Background()
	err = runner.ExecuteBuildCommand(ctx, buildCmd)

	// Verify we get a structured error
	if err == nil {
		t.Fatal("Expected error for invalid UV command, got nil")
	}

	// Check if it's a PythonRuntimeError
	pythonErr, ok := err.(*PythonRuntimeError)
	if !ok {
		t.Fatalf("Expected PythonRuntimeError, got %T: %v", err, err)
	}

	// Verify error properties
	if pythonErr.Type != ErrorTypeUVCommandFailed {
		t.Errorf("Expected error type %s, got %s", ErrorTypeUVCommandFailed, pythonErr.Type)
	}

	// Verify context is present
	if pythonErr.Context["package"] != buildCmd.PackageName {
		t.Error("Expected package context to be set")
	}

	// Verify suggestions are present
	if len(pythonErr.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	t.Logf("UV command error handled correctly: %v", pythonErr)
}

func TestErrorRecoveryManager(t *testing.T) {
	manager := NewErrorRecoveryManager()

	// Test successful operation after retry
	attempts := 0
	err := manager.RetryWithBackoff(func() error {
		attempts++
		if attempts < 2 {
			return NewPythonRuntimeError(ErrorTypeNetwork, SeverityWarning, "network error").
				WithRetry(10 * time.Millisecond)
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected successful retry, got error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	// Test non-retryable error
	attempts = 0
	nonRetryableErr := NewPythonRuntimeError(ErrorTypeHandlerNotFound, SeverityError, "not found")
	err = manager.RetryWithBackoff(func() error {
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

func TestWrapError_Integration(t *testing.T) {
	// Test wrapping various error types
	tests := []struct {
		name         string
		originalErr  error
		context      string
		expectedType ErrorType
	}{
		{
			name:         "file not found",
			originalErr:  os.ErrNotExist,
			context:      "handler lookup",
			expectedType: ErrorTypeFileSystem, // os.ErrNotExist maps to filesystem error
		},
		{
			name:         "permission denied",
			originalErr:  os.ErrPermission,
			context:      "cache access",
			expectedType: ErrorTypeCachePermission,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedErr := WrapError(tt.originalErr, tt.context)

			if wrappedErr == nil {
				t.Fatal("Expected wrapped error, got nil")
			}

			if wrappedErr.Type != tt.expectedType {
				t.Errorf("Expected error type %s, got %s", tt.expectedType, wrappedErr.Type)
			}

			if wrappedErr.Context["originalContext"] != tt.context {
				t.Error("Expected original context to be preserved")
			}

			if wrappedErr.Cause != tt.originalErr {
				t.Error("Expected original error to be preserved as cause")
			}
		})
	}
}
