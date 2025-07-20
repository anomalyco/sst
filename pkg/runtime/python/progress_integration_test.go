package python

import (
	"testing"
	"time"
)

func TestIncrementalBuilder_ProgressReporting(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create incremental builder with progress reporting enabled
	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             tempDir,
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Check that progress reporter was initialized
	if builder.progressReporter == nil {
		t.Fatal("Progress reporter should be initialized")
	}

	// Check initial progress
	progress := builder.GetBuildProgress()
	if progress["progress"].(int) != 0 {
		t.Errorf("Expected initial progress 0, got %v", progress["progress"])
	}
}

func TestIncrementalBuilder_ProgressCallback(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a channel to receive progress events
	eventCh := make(chan ProgressEvent, 100)

	// Create incremental builder with progress callback
	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             tempDir,
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
		ProgressCallback: func(event ProgressEvent) {
			eventCh <- event
		},
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Simulate a build stage
	builder.progressReporter.StartStage(StageInit, "Testing progress callback")

	// Wait for event
	var event ProgressEvent
	select {
	case event = <-eventCh:
		// Got event
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for progress event")
	}

	// Check event
	if event.Stage != StageInit {
		t.Errorf("Expected event stage %s, got %s", StageInit, event.Stage)
	}

	if event.Status != StatusRunning {
		t.Errorf("Expected event status %s, got %s", StatusRunning, event.Status)
	}

	if event.Message != "Testing progress callback" {
		t.Errorf("Expected message 'Testing progress callback', got %s", event.Message)
	}
}

func TestIncrementalBuilder_ProgressStages(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Track progress events
	var events []ProgressEvent

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             tempDir,
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
		ProgressCallback: func(event ProgressEvent) {
			events = append(events, event)
		},
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Simulate build stages
	builder.progressReporter.StartStage(StageInit, "Starting build")
	builder.progressReporter.CompleteStage(StageInit, "Build started")

	builder.progressReporter.StartStage(StageLayoutDetect, "Detecting layout")
	builder.progressReporter.CompleteStage(StageLayoutDetect, "Layout detected")

	builder.progressReporter.StartStage(StageDependencies, "Analyzing dependencies")
	builder.progressReporter.CompleteStage(StageDependencies, "Dependencies analyzed")

	builder.progressReporter.Complete("Build completed", nil)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that we received events for all stages
	if len(events) < 7 { // At least 7 events (start/complete for 3 stages + complete)
		t.Errorf("Expected at least 7 events, got %d", len(events))
	}

	// Check final progress
	progress := builder.GetBuildProgress()
	if progress["progress"].(int) != 100 {
		t.Errorf("Expected final progress 100, got %v", progress["progress"])
	}
}

func TestIncrementalBuilder_ProgressFailure(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Track progress events
	var events []ProgressEvent

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             tempDir,
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
		ProgressCallback: func(event ProgressEvent) {
			events = append(events, event)
		},
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Simulate build failure
	builder.progressReporter.StartStage(StageLayoutDetect, "Detecting layout")
	builder.progressReporter.FailStage(StageLayoutDetect, "Layout detection failed",
		NewPythonRuntimeError(ErrorTypeLayoutDetection, "test error", nil))
	builder.progressReporter.Fail("Build failed",
		NewPythonRuntimeError(ErrorTypeLayoutDetection, "test error", nil))

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that we received failure events
	if len(events) < 3 { // At least 3 events (start, fail stage, fail build)
		t.Errorf("Expected at least 3 events, got %d", len(events))
	}

	// Check that the last event is a failure
	lastEvent := events[len(events)-1]
	if lastEvent.Stage != StageFailed {
		t.Errorf("Expected last event stage %s, got %s", StageFailed, lastEvent.Stage)
	}

	if lastEvent.Status != StatusFailed {
		t.Errorf("Expected last event status %s, got %s", StatusFailed, lastEvent.Status)
	}
}

func TestIncrementalBuilder_ProgressCaching(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Track progress events
	var events []ProgressEvent

	config := IncrementalBuilderConfig{
		CacheDir:                tempDir,
		ArtifactDir:             tempDir,
		MaxCacheAge:             time.Hour,
		MaxCacheSize:            100 * 1024 * 1024, // 100MB
		EnableParallelBuilds:    false,
		MaxParallelBuilds:       1,
		EnableProgressReporting: true,
		EnableBuildOptimization: true,
		ProgressCallback: func(event ProgressEvent) {
			events = append(events, event)
		},
	}

	builder, err := NewIncrementalBuilder(config)
	if err != nil {
		t.Fatalf("Failed to create incremental builder: %v", err)
	}

	// Simulate cached build
	builder.progressReporter.StartStage(StageInit, "Starting build")
	builder.progressReporter.MarkCached(StageLayoutDetect, "Using cached layout")
	builder.progressReporter.MarkCached(StageDependencies, "Using cached dependencies")
	builder.progressReporter.MarkCached(StageBuildPackages, "Using cached packages")
	builder.progressReporter.Complete("Build completed using cache", map[string]interface{}{
		"cached": true,
	})

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that we received cached events
	cachedEvents := 0
	for _, event := range events {
		if event.Status == StatusCached {
			cachedEvents++
		}
	}

	if cachedEvents < 3 {
		t.Errorf("Expected at least 3 cached events, got %d", cachedEvents)
	}

	// Check final progress
	progress := builder.GetBuildProgress()
	if progress["progress"].(int) != 100 {
		t.Errorf("Expected final progress 100, got %v", progress["progress"])
	}
}

func TestProgressReporter_TimeEstimation(t *testing.T) {
	reporter := NewProgressReporter(time.Minute, false)

	// Check initial estimate
	initialEstimate := reporter.GetEstimatedTimeRemaining()
	if initialEstimate != time.Minute {
		t.Errorf("Expected initial estimate %v, got %v", time.Minute, initialEstimate)
	}

	// Simulate some progress
	reporter.StartStage(StageLayoutDetect, "Detecting layout")
	reporter.CompleteStage(StageLayoutDetect, "Layout detected")

	// Check updated estimate
	updatedEstimate := reporter.GetEstimatedTimeRemaining()
	if updatedEstimate >= time.Minute {
		t.Errorf("Expected updated estimate < %v, got %v", time.Minute, updatedEstimate)
	}

	// Complete build
	reporter.Complete("Build completed", nil)

	// Check final estimate
	finalEstimate := reporter.GetEstimatedTimeRemaining()
	if finalEstimate != 0 {
		t.Errorf("Expected final estimate 0, got %v", finalEstimate)
	}
}
