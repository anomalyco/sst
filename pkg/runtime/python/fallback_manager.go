package python

import (
	"context"
	"fmt"
	"log/slog"
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
		enableLegacyFallback:  config.EnableLegacyFallback,
		enableCacheCleanup:    config.EnableCacheCleanup,
		maxFallbackAttempts:   maxAttempts,
		fallbackHistory:       make([]FallbackEvent, 0),
		legacyBuilder:         config.LegacyBuilder,
	}
}\n\n// RegisterCallback adds a callback for fallback events\nfunc (fm *FallbackManager) RegisterCallback(callback FallbackCallback) {\n\tfm.callbacks = append(fm.callbacks, callback)\n}\n\n// ShouldFallback determines if a fallback should be triggered based on an error\nfunc (fm *FallbackManager) ShouldFallback(err error) (bool, FallbackReason, FallbackStrategy) {\n\tif err == nil {\n\t\treturn false, \"\", \"\"\n\t}\n\t\n\t// Check if this is a Python runtime error with specific handling\n\tif pythonErr, ok := err.(*PythonRuntimeError); ok {\n\t\tswitch pythonErr.Type {\n\t\tcase ErrorTypeLayoutDetection:\n\t\t\treturn true, FallbackReasonLayoutUnsupported, StrategySimpleLayout\n\t\tcase ErrorTypeCacheCorruption:\n\t\t\treturn true, FallbackReasonCacheCorrupted, StrategyFullRebuild\n\t\tcase ErrorTypeCacheInvalid:\n\t\t\treturn true, FallbackReasonCacheInvalid, StrategyFullRebuild\n\t\tcase ErrorTypeDependencyResolution:\n\t\t\treturn true, FallbackReasonDependencyError, StrategyNoOptimization\n\t\tcase ErrorTypeBuildFailure:\n\t\t\tif fm.enableLegacyFallback {\n\t\t\t\treturn true, FallbackReasonBuildError, StrategyLegacyBuilder\n\t\t\t}\n\t\t\treturn true, FallbackReasonBuildError, StrategyFullRebuild\n\t\t}\n\t}\n\t\n\t// Check for specific error patterns\n\terrorMsg := err.Error()\n\t\n\t// Cache-related errors\n\tif containsAny(errorMsg, []string{\"cache corrupted\", \"invalid cache\", \"cache error\"}) {\n\t\treturn true, FallbackReasonCacheCorrupted, StrategyFullRebuild\n\t}\n\t\n\t// Layout-related errors\n\tif containsAny(errorMsg, []string{\"layout not supported\", \"unsupported structure\", \"handler not found\"}) {\n\t\treturn true, FallbackReasonLayoutUnsupported, StrategySimpleLayout\n\t}\n\t\n\t// Dependency-related errors\n\tif containsAny(errorMsg, []string{\"dependency error\", \"uv failed\", \"package not found\"}) {\n\t\treturn true, FallbackReasonDependencyError, StrategyNoOptimization\n\t}\n\t\n\t// Generic build errors - use legacy builder if available\n\tif fm.enableLegacyFallback {\n\t\treturn true, FallbackReasonUnknownError, StrategyLegacyBuilder\n\t}\n\t\n\t// Default to full rebuild\n\treturn true, FallbackReasonUnknownError, StrategyFullRebuild\n}\n\n// ExecuteFallback executes a fallback strategy\nfunc (fm *FallbackManager) ExecuteFallback(ctx context.Context, input *runtime.BuildInput, reason FallbackReason, strategy FallbackStrategy, originalError error) (*runtime.BuildOutput, error) {\n\t// Check if we've exceeded max fallback attempts\n\tif len(fm.fallbackHistory) >= fm.maxFallbackAttempts {\n\t\treturn nil, fmt.Errorf(\"exceeded maximum fallback attempts (%d): %w\", fm.maxFallbackAttempts, originalError)\n\t}\n\t\n\t// Create fallback event\n\tevent := FallbackEvent{\n\t\tReason:        reason,\n\t\tStrategy:      strategy,\n\t\tOriginalError: originalError,\n\t\tTimestamp:     time.Now(),\n\t\tContext: map[string]interface{}{\n\t\t\t\"functionID\": input.FunctionID,\n\t\t\t\"handler\":    input.Handler,\n\t\t},\n\t}\n\t\n\t// Execute the appropriate fallback strategy\n\tvar result *runtime.BuildOutput\n\tvar err error\n\t\n\tswitch strategy {\n\tcase StrategyFullRebuild:\n\t\tevent.Message = \"Falling back to full rebuild due to \" + string(reason)\n\t\tresult, err = fm.executeFullRebuild(ctx, input)\n\t\t\n\tcase StrategyLegacyBuilder:\n\t\tevent.Message = \"Falling back to legacy builder due to \" + string(reason)\n\t\tresult, err = fm.executeLegacyBuilder(ctx, input)\n\t\t\n\tcase StrategySimpleLayout:\n\t\tevent.Message = \"Falling back to simple layout detection due to \" + string(reason)\n\t\tresult, err = fm.executeSimpleLayout(ctx, input)\n\t\t\n\tcase StrategyNoOptimization:\n\t\tevent.Message = \"Falling back to build without optimizations due to \" + string(reason)\n\t\tresult, err = fm.executeNoOptimization(ctx, input)\n\t\t\n\tdefault:\n\t\treturn nil, fmt.Errorf(\"unknown fallback strategy: %s\", strategy)\n\t}\n\t\n\t// Record the fallback event\n\tfm.fallbackHistory = append(fm.fallbackHistory, event)\n\t\n\t// Notify callbacks\n\tfm.notifyCallbacks(event)\n\t\n\t// Log the fallback\n\tif err != nil {\n\t\tslog.Error(\"fallback strategy failed\",\n\t\t\t\"strategy\", string(strategy),\n\t\t\t\"reason\", string(reason),\n\t\t\t\"error\", err,\n\t\t\t\"originalError\", originalError)\n\t} else {\n\t\tslog.Warn(\"executed fallback strategy\",\n\t\t\t\"strategy\", string(strategy),\n\t\t\t\"reason\", string(reason),\n\t\t\t\"functionID\", input.FunctionID)\n\t}\n\t\n\treturn result, err\n}\n\n// executeFullRebuild performs a full rebuild without any optimizations\nfunc (fm *FallbackManager) executeFullRebuild(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {\n\tslog.Info(\"executing full rebuild fallback\", \"functionID\", input.FunctionID)\n\t\n\t// Clean up cache if enabled\n\tif fm.enableCacheCleanup {\n\t\tif err := fm.cleanupCache(input.FunctionID); err != nil {\n\t\t\tslog.Warn(\"failed to cleanup cache during fallback\", \"error\", err)\n\t\t}\n\t}\n\t\n\t// Create a simple incremental builder without optimizations\n\tconfig := IncrementalBuilderConfig{\n\t\tCacheDir:                \"/tmp/sst-fallback-cache\",\n\t\tArtifactDir:             input.Out(),\n\t\tMaxCacheAge:             time.Hour,\n\t\tMaxCacheSize:            100 * 1024 * 1024, // 100MB\n\t\tEnableParallelBuilds:    false,\n\t\tMaxParallelBuilds:       1,\n\t\tEnableProgressReporting: false,\n\t\tEnableBuildOptimization: false, // Disable optimizations\n\t}\n\t\n\tbuilder, err := NewIncrementalBuilder(config)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"failed to create fallback builder: %w\", err)\n\t}\n\t\n\treturn builder.Build(ctx, input)\n}\n\n// executeLegacyBuilder uses the legacy Python builder\nfunc (fm *FallbackManager) executeLegacyBuilder(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {\n\tif fm.legacyBuilder == nil {\n\t\treturn nil, fmt.Errorf(\"legacy builder not available\")\n\t}\n\t\n\tslog.Info(\"executing legacy builder fallback\", \"functionID\", input.FunctionID)\n\t\n\treturn fm.legacyBuilder.Build(ctx, input)\n}\n\n// executeSimpleLayout uses simple layout detection\nfunc (fm *FallbackManager) executeSimpleLayout(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {\n\tslog.Info(\"executing simple layout fallback\", \"functionID\", input.FunctionID)\n\t\n\t// Create a builder with simple layout detection\n\tconfig := IncrementalBuilderConfig{\n\t\tCacheDir:                \"/tmp/sst-fallback-cache\",\n\t\tArtifactDir:             input.Out(),\n\t\tMaxCacheAge:             time.Hour,\n\t\tMaxCacheSize:            100 * 1024 * 1024, // 100MB\n\t\tEnableParallelBuilds:    false,\n\t\tMaxParallelBuilds:       1,\n\t\tEnableProgressReporting: false,\n\t\tEnableBuildOptimization: true,\n\t}\n\t\n\tbuilder, err := NewIncrementalBuilder(config)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"failed to create simple layout builder: %w\", err)\n\t}\n\t\n\t// Override layout detector with simple detection\n\tbuilder.layoutDetector = NewSimpleLayoutDetector()\n\t\n\treturn builder.Build(ctx, input)\n}\n\n// executeNoOptimization builds without optimizations\nfunc (fm *FallbackManager) executeNoOptimization(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {\n\tslog.Info(\"executing no optimization fallback\", \"functionID\", input.FunctionID)\n\t\n\t// Create a builder without optimizations\n\tconfig := IncrementalBuilderConfig{\n\t\tCacheDir:                \"/tmp/sst-fallback-cache\",\n\t\tArtifactDir:             input.Out(),\n\t\tMaxCacheAge:             time.Hour,\n\t\tMaxCacheSize:            100 * 1024 * 1024, // 100MB\n\t\tEnableParallelBuilds:    false,\n\t\tMaxParallelBuilds:       1,\n\t\tEnableProgressReporting: false,\n\t\tEnableBuildOptimization: false, // Disable all optimizations\n\t}\n\t\n\tbuilder, err := NewIncrementalBuilder(config)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"failed to create no-optimization builder: %w\", err)\n\t}\n\t\n\treturn builder.Build(ctx, input)\n}\n\n// cleanupCache removes corrupted cache entries\nfunc (fm *FallbackManager) cleanupCache(functionID string) error {\n\tslog.Info(\"cleaning up corrupted cache\", \"functionID\", functionID)\n\t\n\t// This would implement cache cleanup logic\n\t// For now, we'll just log the action\n\treturn nil\n}\n\n// GetFallbackHistory returns the recent fallback events\nfunc (fm *FallbackManager) GetFallbackHistory() []FallbackEvent {\n\treturn fm.fallbackHistory\n}\n\n// ClearFallbackHistory clears the fallback history\nfunc (fm *FallbackManager) ClearFallbackHistory() {\n\tfm.fallbackHistory = make([]FallbackEvent, 0)\n}\n\n// notifyCallbacks notifies all registered callbacks\nfunc (fm *FallbackManager) notifyCallbacks(event FallbackEvent) {\n\tfor _, callback := range fm.callbacks {\n\t\tgo callback(event)\n\t}\n}\n\n// containsAny checks if a string contains any of the given substrings\nfunc containsAny(s string, substrings []string) bool {\n\tfor _, substring := range substrings {\n\t\tif len(s) >= len(substring) {\n\t\t\tfor i := 0; i <= len(s)-len(substring); i++ {\n\t\t\t\tif s[i:i+len(substring)] == substring {\n\t\t\t\t\treturn true\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n\treturn false\n}\n\n// NewSimpleLayoutDetector creates a simple layout detector for fallback\nfunc NewSimpleLayoutDetector() *LayoutDetector {\n\t// Create a layout detector with simple, conservative settings\n\tconfig := LayoutDetectorConfig{\n\t\tCacheTimeout: time.Minute, // Short cache timeout\n\t}\n\t\n\tdetector, _ := NewLayoutDetector(config)\n\treturn detector\n}"