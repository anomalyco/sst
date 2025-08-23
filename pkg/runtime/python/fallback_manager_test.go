package python

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// MockLegacyBuilder implements LegacyPythonBuilder for testing
type MockLegacyBuilder struct {
	shouldFail bool
	result     *runtime.BuildOutput
}

func (m *MockLegacyBuilder) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	if m.shouldFail {
		return nil, errors.New("mock legacy builder failure")
	}
	if m.result != nil {
		return m.result, nil
	}
	return &runtime.BuildOutput{
		Handler: "mock_handler",
	}, nil
}

func TestNewFallbackManager(t *testing.T) {
	config := FallbackManagerConfig{
		EnableLegacyFallback: true,
		EnableCacheCleanup:   true,
		MaxFallbackAttempts:  5,
		LegacyBuilder:        &MockLegacyBuilder{},
	}

	fm := NewFallbackManager(config)

	if !fm.enableLegacyFallback {
		t.Error("Expected legacy fallback to be enabled")
	}

	if !fm.enableCacheCleanup {
		t.Error("Expected cache cleanup to be enabled")
	}

	if fm.maxFallbackAttempts != 5 {
		t.Errorf("Expected max fallback attempts 5, got %d", fm.maxFallbackAttempts)
	}

	if fm.legacyBuilder == nil {
		t.Error("Expected legacy builder to be set")
	}
}

func TestFallbackManager_ShouldFallback(t *testing.T) {
	fm := NewFallbackManager(FallbackManagerConfig{
		EnableLegacyFallback: true,
	})

	tests := []struct {
		name             string
		err              error
		expectedFallback bool
		expectedReason   FallbackReason
		expectedStrategy FallbackStrategy
	}{
		{
			name:             "no error",
			err:              nil,
			expectedFallback: false,
		},
		{
			name:             "layout detection error",
			err:              NewPythonRuntimeError(ErrorTypeLayoutDetection, ErrorSeverityError, "layout not supported"),
			expectedFallback: true,
			expectedReason:   FallbackReasonLayoutUnsupported,
			expectedStrategy: StrategySimpleLayout,
		},
		{
			name:             "cache corruption error",
			err:              NewPythonRuntimeError(ErrorTypeCacheCorrupted, ErrorSeverityError, "cache corrupted"),
			expectedFallback: true,
			expectedReason:   FallbackReasonCacheCorrupted,
			expectedStrategy: StrategyFullRebuild,
		},
		{
			name:             "build failure with legacy fallback",
			err:              NewPythonRuntimeError(ErrorTypeBuildFailed, ErrorSeverityError, "build failed"),
			expectedFallback: true,
			expectedReason:   FallbackReasonBuildError,
			expectedStrategy: StrategyLegacyBuilder,
		},
		{
			name:             "dependency error",
			err:              NewPythonRuntimeError(ErrorTypeDependencyFailed, ErrorSeverityError, "dependency error"),
			expectedFallback: true,
			expectedReason:   FallbackReasonDependencyError,
			expectedStrategy: StrategyNoOptimization,
		},
		{
			name:             "generic cache error",
			err:              errors.New("cache corrupted"),
			expectedFallback: true,
			expectedReason:   FallbackReasonCacheCorrupted,
			expectedStrategy: StrategyFullRebuild,
		},
		{
			name:             "generic layout error",
			err:              errors.New("layout not supported"),
			expectedFallback: true,
			expectedReason:   FallbackReasonLayoutUnsupported,
			expectedStrategy: StrategySimpleLayout,
		},
		{
			name:             "generic dependency error",
			err:              errors.New("uv failed to install"),
			expectedFallback: true,
			expectedReason:   FallbackReasonDependencyError,
			expectedStrategy: StrategyNoOptimization,
		},
		{
			name:             "unknown error with legacy fallback",
			err:              errors.New("unknown error"),
			expectedFallback: true,
			expectedReason:   FallbackReasonUnknownError,
			expectedStrategy: StrategyLegacyBuilder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldFallback, reason, strategy := fm.ShouldFallback(tt.err)

			if shouldFallback != tt.expectedFallback {
				t.Errorf("Expected fallback %v, got %v", tt.expectedFallback, shouldFallback)
			}

			if tt.expectedFallback {
				if reason != tt.expectedReason {
					t.Errorf("Expected reason %s, got %s", tt.expectedReason, reason)
				}

				if strategy != tt.expectedStrategy {
					t.Errorf("Expected strategy %s, got %s", tt.expectedStrategy, strategy)
				}
			}
		})
	}
}

func TestFallbackManager_ShouldFallback_NoLegacyFallback(t *testing.T) {
	fm := NewFallbackManager(FallbackManagerConfig{
		EnableLegacyFallback: false,
	})

	// Test that build failures fall back to full rebuild when legacy fallback is disabled
	shouldFallback, reason, strategy := fm.ShouldFallback(
		NewPythonRuntimeError(ErrorTypeBuildFailed, ErrorSeverityError, "build failed"))

	if !shouldFallback {
		t.Error("Expected fallback to be triggered")
	}

	if reason != FallbackReasonBuildError {
		t.Errorf("Expected reason %s, got %s", FallbackReasonBuildError, reason)
	}

	if strategy != StrategyFullRebuild {
		t.Errorf("Expected strategy %s, got %s", StrategyFullRebuild, strategy)
	}
}

func TestFallbackManager_ExecuteFallback_LegacyBuilder(t *testing.T) {
	mockBuilder := &MockLegacyBuilder{
		result: &runtime.BuildOutput{Handler: "legacy_handler"},
	}

	fm := NewFallbackManager(FallbackManagerConfig{
		EnableLegacyFallback: true,
		LegacyBuilder:        mockBuilder,
	})

	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Handler:    "test.handler",
	}

	result, err := fm.ExecuteFallback(
		context.Background(),
		input,
		FallbackReasonBuildError,
		StrategyLegacyBuilder,
		errors.New("original error"),
	)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Handler != "legacy_handler" {
		t.Errorf("Expected handler 'legacy_handler', got %s", result.Handler)
	}

	// Check that fallback was recorded
	history := fm.GetFallbackHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 fallback event, got %d", len(history))
	}

	if history[0].Reason != FallbackReasonBuildError {
		t.Errorf("Expected reason %s, got %s", FallbackReasonBuildError, history[0].Reason)
	}

	if history[0].Strategy != StrategyLegacyBuilder {
		t.Errorf("Expected strategy %s, got %s", StrategyLegacyBuilder, history[0].Strategy)
	}
}

func TestFallbackManager_ExecuteFallback_MaxAttempts(t *testing.T) {
	fm := NewFallbackManager(FallbackManagerConfig{
		MaxFallbackAttempts: 2,
	})

	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Handler:    "test.handler",
	}

	// Execute fallbacks up to the limit
	for i := 0; i < 2; i++ {
		_, err := fm.ExecuteFallback(
			context.Background(),
			input,
			FallbackReasonCacheCorrupted,
			StrategyFullRebuild,
			errors.New("test error"),
		)
		if err != nil {
			t.Fatalf("Unexpected error on attempt %d: %v", i+1, err)
		}
	}

	// This should fail due to max attempts
	_, err := fm.ExecuteFallback(
		context.Background(),
		input,
		FallbackReasonCacheCorrupted,
		StrategyFullRebuild,
		errors.New("test error"),
	)

	if err == nil {
		t.Error("Expected error due to max fallback attempts")
	}

	if len(fm.GetFallbackHistory()) != 2 {
		t.Errorf("Expected 2 fallback events, got %d", len(fm.GetFallbackHistory()))
	}
}

func TestFallbackManager_Callbacks(t *testing.T) {
	fm := NewFallbackManager(FallbackManagerConfig{})

	// Create a channel to receive events
	eventCh := make(chan FallbackEvent, 10)

	// Register callback
	fm.RegisterCallback(func(event FallbackEvent) {
		eventCh <- event
	})

	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Handler:    "test.handler",
	}

	// Execute a fallback
	_, err := fm.ExecuteFallback(
		context.Background(),
		input,
		FallbackReasonCacheCorrupted,
		StrategyFullRebuild,
		errors.New("test error"),
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Wait for event
	var event FallbackEvent
	select {
	case event = <-eventCh:
		// Got event
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for fallback event")
	}

	// Check event
	if event.Reason != FallbackReasonCacheCorrupted {
		t.Errorf("Expected reason %s, got %s", FallbackReasonCacheCorrupted, event.Reason)
	}

	if event.Strategy != StrategyFullRebuild {
		t.Errorf("Expected strategy %s, got %s", StrategyFullRebuild, event.Strategy)
	}
}

func TestFallbackManager_ClearHistory(t *testing.T) {
	fm := NewFallbackManager(FallbackManagerConfig{})

	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Handler:    "test.handler",
	}

	// Execute a fallback
	_, err := fm.ExecuteFallback(
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
	if len(fm.GetFallbackHistory()) == 0 {
		t.Error("Expected fallback history to have events")
	}

	// Clear history
	fm.ClearFallbackHistory()

	// Check that history is empty
	if len(fm.GetFallbackHistory()) != 0 {
		t.Error("Expected fallback history to be empty after clearing")
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		substrings []string
		expected   bool
	}{
		{
			name:       "contains one",
			s:          "this is a test string",
			substrings: []string{"test", "example"},
			expected:   true,
		},
		{
			name:       "contains multiple",
			s:          "this is a test string",
			substrings: []string{"test", "string"},
			expected:   true,
		},
		{
			name:       "contains none",
			s:          "this is a test string",
			substrings: []string{"example", "sample"},
			expected:   false,
		},
		{
			name:       "empty string",
			s:          "",
			substrings: []string{"test"},
			expected:   false,
		},
		{
			name:       "empty substrings",
			s:          "test string",
			substrings: []string{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrings)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
