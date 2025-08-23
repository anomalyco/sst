package python

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

func TestIncrementalBuilder_FallbackIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Track fallback events
	var fallbackEvents []FallbackEvent

	// Create incremental builder with fallbacks enabled
	config := IncrementalBuilderConfig{
		CacheDir:                  tempDir,
		ArtifactDir:               tempDir,
		MaxCacheAge:               time.Hour,
		MaxCacheSize:              100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:      false,
		MaxParallelBuilds:         1,
		EnableProgressReporting:   true,
		EnableBuildOptimization:   true,
		EnableFallbacks:           true,
		EnableLegacyFallback:      false,
		EnableDeprecationWarnings: true,
		FallbackCallback: func(event FallbackEvent) {
			fallbackEvents = append(fallbackEvents, event)
		},
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Check that fallback manager was initialized
	if builder.fallbackManager == nil {
		t.Fatal("Fallback manager should be initialized")
	}

	// Check that deprecation checker was initialized
	if builder.deprecationChecker == nil {
		t.Fatal("Deprecation checker should be initialized")
	}
}

func TestIncrementalBuilder_FallbackDisabled(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create incremental builder with fallbacks disabled
	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             tempDir,
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
		EnableFallbacks:         false, // Disabled
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Check that fallback manager was not initialized
	if builder.fallbackManager != nil {
		t.Error("Fallback manager should not be initialized when disabled")
	}
}

func TestIncrementalBuilder_FallbackWithLegacyBuilder(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock legacy builder
	mockLegacyBuilder := &MockLegacyBuilder{
		result: &runtime.BuildOutput{Handler: "legacy_handler"},
	}

	// Track fallback events
	var fallbackEvents []FallbackEvent

	// Create incremental builder with legacy fallback enabled
	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             tempDir,
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
		EnableFallbacks:         true,
		EnableLegacyFallback:    true,
		LegacyBuilder:           mockLegacyBuilder,
		FallbackCallback: func(event FallbackEvent) {
			fallbackEvents = append(fallbackEvents, event)
		},
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Check that fallback manager has legacy builder
	if builder.fallbackManager.legacyBuilder == nil {
		t.Error("Fallback manager should have legacy builder")
	}

	if !builder.fallbackManager.enableLegacyFallback {
		t.Error("Legacy fallback should be enabled")
	}
}

func TestIncrementalBuilder_DeprecationWarnings(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Track deprecation warnings
	var warnings []DeprecationWarning

	// Create incremental builder with deprecation warnings enabled
	config := IncrementalBuilderConfig{
		CacheDir:                  tempDir,
		ArtifactDir:               tempDir,
		MaxCacheAge:               time.Hour,
		MaxCacheSize:              100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:      false,
		MaxParallelBuilds:         1,
		EnableProgressReporting:   true,
		EnableBuildOptimization:   true,
		EnableFallbacks:           true,
		EnableDeprecationWarnings: true,
		DeprecationCallback: func(warning DeprecationWarning) {
			warnings = append(warnings, warning)
		},
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Test deprecation checking
	layout := &LayoutInfo{
		Type:          LayoutTypeWorkspace,
		WorkspaceDir:  "/test/src/mypackage",
		PackageName:   "mypackage",
		PyprojectPath: "/test/pyproject.toml", // Add pyproject to avoid missing_pyproject warning
	}

	builder.deprecationChecker.CheckLayout(layout)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that deprecation warning was issued
	if len(warnings) == 0 {
		t.Error("Expected deprecation warning to be issued")
	}

	if len(warnings) > 0 {
		warning := warnings[0]
		if warning.Pattern != "old_workspace_structure" {
			t.Errorf("Expected pattern 'old_workspace_structure', got %s", warning.Pattern)
		}
	}
}

func TestIncrementalBuilder_FallbackHistory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create incremental builder with fallbacks enabled
	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             tempDir,
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
		EnableFallbacks:         true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Check initial history is empty
	history := builder.GetFallbackHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty fallback history, got %d events", len(history))
	}

	// Simulate a fallback
	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Handler:    "test.handler",
	}

	_, err = builder.fallbackManager.ExecuteFallback(
		context.Background(),
		input,
		FallbackReasonCacheCorrupted,
		StrategyFullRebuild,
		errors.New("test error"),
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that history has events
	history = builder.GetFallbackHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 fallback event, got %d", len(history))
	}

	// Clear history
	builder.ClearFallbackHistory()

	// Check that history is empty
	history = builder.GetFallbackHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty fallback history after clearing, got %d events", len(history))
	}
}

func TestDeprecationChecker_PythonVersionWarnings(t *testing.T) {
	checker := NewDeprecationChecker(true)

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	// Test deprecated Python version
	dependencies := &DependencyInfo{
		PythonVersion: "3.8.10",
		Dependencies:  map[string]string{},
	}

	checker.CheckDependencies(dependencies)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that warning was issued
	if len(warnings) == 0 {
		t.Error("Expected deprecation warning for Python 3.8")
	}

	if len(warnings) > 0 {
		warning := warnings[0]
		if warning.Pattern != "deprecated_python_version" {
			t.Errorf("Expected pattern 'deprecated_python_version', got %s", warning.Pattern)
		}
		if warning.Severity != SeverityError {
			t.Errorf("Expected severity 'error', got %s", warning.Severity)
		}
	}
}

func TestDeprecationChecker_PackageWarnings(t *testing.T) {
	checker := NewDeprecationChecker(true)

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	// Test deprecated packages
	dependencies := &DependencyInfo{
		PythonVersion: "3.11.0",
		Dependencies: map[string]string{
			"flask":    "2.0.0",
			"requests": "2.28.0",
		},
	}

	checker.CheckDependencies(dependencies)

	// Give callbacks time to execute
	time.Sleep(200 * time.Millisecond)

	// Check that warnings were issued
	if len(warnings) < 2 {
		t.Errorf("Expected at least 2 deprecation warnings, got %d", len(warnings))
		for i, w := range warnings {
			t.Logf("Warning %d: %s - %s (context: %v)", i+1, w.Pattern, w.Message, w.Context)
		}
	}

	// Check for flask warning
	foundFlaskWarning := false
	foundRequestsWarning := false
	for _, warning := range warnings {
		if warning.Pattern == "deprecated_package" {
			if pkg, exists := warning.Context["package"]; exists {
				if pkg == "flask" {
					foundFlaskWarning = true
				}
				if pkg == "requests" {
					foundRequestsWarning = true
				}
			}
		}
	}

	if !foundFlaskWarning {
		t.Error("Expected flask deprecation warning")
	}

	if !foundRequestsWarning {
		t.Error("Expected requests deprecation warning")
	}
}
