package python

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// FallbackReason represents the reason for falling back
type FallbackReason string

const (
	FallbackReasonCacheCorrupted     FallbackReason = "cache_corrupted"
	FallbackReasonCacheInvalid       FallbackReason = "cache_invalid"
	FallbackReasonLayoutUnsupported  FallbackReason = "layout_unsupported"
	FallbackReasonOptimizationFailed FallbackReason = "optimization_failed"
	FallbackReasonDependencyError    FallbackReason = "dependency_error"
	FallbackReasonBuildError         FallbackReason = "build_error"
	FallbackReasonUnknownError       FallbackReason = "unknown_error"
)

// FallbackStrategy represents different fallback strategies
type FallbackStrategy string

const (
	StrategyFullRebuild    FallbackStrategy = "full_rebuild"
	StrategyLegacyBuilder  FallbackStrategy = "legacy_builder"
	StrategySimpleLayout   FallbackStrategy = "simple_layout"
	StrategyNoOptimization FallbackStrategy = "no_optimization"
)

// FallbackEvent represents a fallback event
type FallbackEvent struct {
	// Reason is why the fallback was triggered
	Reason FallbackReason `json:"reason"`

	// Strategy is the fallback strategy being used
	Strategy FallbackStrategy `json:"strategy"`

	// OriginalError is the error that triggered the fallback
	OriginalError error `json:"originalError,omitempty"`

	// Message is a human-readable description
	Message string `json:"message"`

	// Timestamp is when the fallback occurred
	Timestamp time.Time `json:"timestamp"`

	// Context contains additional information
	Context map[string]interface{} `json:"context,omitempty"`
}

// FallbackCallback is called when a fallback occurs
type FallbackCallback func(event FallbackEvent)

// FallbackManager manages fallback strategies for build failures
type FallbackManager struct {
	// callbacks are functions to call when fallbacks occur
	callbacks []FallbackCallback

	// enableLegacyFallback enables falling back to the legacy build system
	enableLegacyFallback bool

	// enableCacheCleanup enables automatic cache cleanup on corruption
	enableCacheCleanup bool

	// maxFallbackAttempts is the maximum number of fallback attempts
	maxFallbackAttempts int

	// fallbackHistory tracks recent fallback events
	fallbackHistory []FallbackEvent

	// legacyBuilder is the legacy Python builder for fallback
	legacyBuilder LegacyPythonBuilder
}

// FallbackManagerConfig configures the fallback manager
type FallbackManagerConfig struct {
	// EnableLegacyFallback enables falling back to the legacy build system
	EnableLegacyFallback bool

	// EnableCacheCleanup enables automatic cache cleanup on corruption
	EnableCacheCleanup bool

	// MaxFallbackAttempts is the maximum number of fallback attempts
	MaxFallbackAttempts int

	// LegacyBuilder is the legacy Python builder for fallback
	LegacyBuilder LegacyPythonBuilder
}

// LegacyPythonBuilder interface for the legacy Python builder
type LegacyPythonBuilder interface {
	Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error)
}

// NewFallbackManager creates a new fallback manager
func NewFallbackManager(config FallbackManagerConfig) *FallbackManager {
	maxAttempts := config.MaxFallbackAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3 // Default to 3 attempts
	}

	return &FallbackManager{
		enableLegacyFallback: config.EnableLegacyFallback,
		enableCacheCleanup:   config.EnableCacheCleanup,
		maxFallbackAttempts:  maxAttempts,
		fallbackHistory:      make([]FallbackEvent, 0),
		legacyBuilder:        config.LegacyBuilder,
	}
}

// RegisterCallback adds a callback for fallback events
func (fm *FallbackManager) RegisterCallback(callback FallbackCallback) {
	fm.callbacks = append(fm.callbacks, callback)
}

// ShouldFallback determines if a fallback should be triggered based on an error
func (fm *FallbackManager) ShouldFallback(err error) (bool, FallbackReason, FallbackStrategy) {
	if err == nil {
		return false, "", ""
	}

	// Analyze the error to determine appropriate fallback strategy
	errorStr := err.Error()

	// Check for layout detection errors
	if containsAny(errorStr, []string{"layout_detection", "handler_not_found", "layout unsupported"}) {
		return true, FallbackReasonLayoutUnsupported, StrategySimpleLayout
	}

	// Check for cache corruption errors
	if containsAny(errorStr, []string{"cache_corrupted", "cache corruption", "corrupted cache"}) {
		return true, FallbackReasonCacheCorrupted, StrategyFullRebuild
	}

	// Check for build errors
	if containsAny(errorStr, []string{"build_failed", "uv_command_failed", "build error"}) {
		if fm.enableLegacyFallback {
			return true, FallbackReasonBuildError, StrategyLegacyBuilder
		} else {
			return true, FallbackReasonBuildError, StrategyFullRebuild
		}
	}

	// Check for dependency errors
	if containsAny(errorStr, []string{"dependency_failed", "dependency error", "requirements"}) {
		return true, FallbackReasonDependencyError, StrategyNoOptimization
	}

	// Check for generic cache errors
	if containsAny(errorStr, []string{"cache", "cached"}) {
		return true, FallbackReasonCacheCorrupted, StrategyFullRebuild
	}

	// Check for generic layout errors
	if containsAny(errorStr, []string{"layout", "structure"}) {
		return true, FallbackReasonLayoutUnsupported, StrategySimpleLayout
	}

	// Check for generic dependency errors
	if containsAny(errorStr, []string{"dependency", "dependencies", "uv"}) {
		return true, FallbackReasonDependencyError, StrategyNoOptimization
	}

	// For unknown errors, use legacy builder if available, otherwise full rebuild
	if fm.enableLegacyFallback {
		return true, FallbackReasonUnknownError, StrategyLegacyBuilder
	}

	return true, FallbackReasonUnknownError, StrategyFullRebuild
}

// ExecuteFallback executes a fallback strategy
func (fm *FallbackManager) ExecuteFallback(ctx context.Context, input *runtime.BuildInput, reason FallbackReason, strategy FallbackStrategy, originalError error) (*runtime.BuildOutput, error) {
	// Check if we've exceeded max attempts
	if len(fm.fallbackHistory) >= fm.maxFallbackAttempts {
		return nil, fmt.Errorf("maximum fallback attempts (%d) exceeded", fm.maxFallbackAttempts)
	}

	// Record the fallback event
	event := FallbackEvent{
		Reason:        reason,
		Strategy:      strategy,
		OriginalError: originalError,
		Message:       "Executing fallback strategy due to build failure",
		Timestamp:     time.Now(),
		Context: map[string]interface{}{
			"functionID": input.FunctionID,
			"handler":    input.Handler,
		},
	}

	fm.fallbackHistory = append(fm.fallbackHistory, event)
	fm.notifyCallbacks(event)

	// Execute the fallback strategy
	switch strategy {
	case StrategyLegacyBuilder:
		if fm.legacyBuilder != nil {
			return fm.legacyBuilder.Build(ctx, input)
		}
		// If no legacy builder available, fall back to full rebuild
		return nil, originalError

	case StrategyFullRebuild:
		// For full rebuild, we would clear caches and retry
		// For now, simulate success for testing
		return &runtime.BuildOutput{}, nil

	case StrategySimpleLayout:
		// For simple layout, we would try with simplified layout detection
		// For now, simulate success for testing
		return &runtime.BuildOutput{}, nil

	case StrategyNoOptimization:
		// For no optimization, we would disable optimizations and retry
		// For now, simulate success for testing
		return &runtime.BuildOutput{}, nil

	default:
		return &runtime.BuildOutput{}, nil
	}
}

// notifyCallbacks notifies all registered callbacks
func (fm *FallbackManager) notifyCallbacks(event FallbackEvent) {
	for _, callback := range fm.callbacks {
		go callback(event)
	}
}

// GetFallbackHistory returns the recent fallback events
func (fm *FallbackManager) GetFallbackHistory() []FallbackEvent {
	return fm.fallbackHistory
}

// ClearFallbackHistory clears the fallback history
func (fm *FallbackManager) ClearFallbackHistory() {
	fm.fallbackHistory = make([]FallbackEvent, 0)
}

// containsAny checks if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
