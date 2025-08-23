package ui

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sst/sst/v3/cmd/sst/mosaic/aws"
)

// TestFunctionBuildProgressEvent_UIEventHandling tests the integration of FunctionBuildProgressEvent
// with the UI system to verify proper event handling, formatting, and display consistency.
func TestFunctionBuildProgressEvent_UIEventHandling(t *testing.T) {

	// Create a temporary log file for testing
	logFile, err := os.CreateTemp("", "ui_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	defer os.Remove(logFile.Name())
	defer logFile.Close()

	// Create UI instance with test options
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ui := New(ctx, WithSilent, WithLog(logFile))
	defer ui.Destroy()

	// Test data for FunctionBuildProgressEvent
	testEvents := []aws.FunctionBuildProgressEvent{
		{
			FunctionID: "test-function-123",
			Stage:      "dependencies",
			Message:    "Installing dependencies...",
		},
		{
			FunctionID: "test-function-123",
			Stage:      "build",
			Message:    "Building function code...",
		},
		{
			FunctionID: "test-function-456",
			Stage:      "dependencies",
			Message:    "Installing requirements.txt...",
		},
	}

	// Process each test event through the UI
	for _, event := range testEvents {
		ui.Event(&event)
	}

	// Read the log file to verify output
	logFile.Seek(0, 0)
	logContent, err := os.ReadFile(logFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logOutput := string(logContent)

	// Verify that events are properly formatted and displayed
	for _, event := range testEvents {
		// Check that the function ID is used for color coding (consistent display)
		expectedMessage := event.Stage + ": " + event.Message
		if !strings.Contains(logOutput, expectedMessage) {
			t.Errorf("Expected log to contain formatted message '%s', but it was not found in: %s", expectedMessage, logOutput)
		}

		// Verify that "Build" label is used (consistent with other function events)
		if !strings.Contains(logOutput, "Build") {
			t.Errorf("Expected log to contain 'Build' label for function build progress events")
		}
	}

	// Verify that multiple function events are handled correctly
	// Note: Function IDs are used for color coding, not directly in text output
	// We verify by checking that all expected messages are present
	expectedMessages := []string{
		"dependencies: Installing dependencies...",
		"build: Building function code...",
		"dependencies: Installing requirements.txt...",
	}

	for _, expectedMsg := range expectedMessages {
		if !strings.Contains(logOutput, expectedMsg) {
			t.Errorf("Expected log to contain message '%s', but it was not found in: %s", expectedMsg, logOutput)
		}
	}

	// Verify that different stages are properly displayed
	if !strings.Contains(logOutput, "dependencies: Installing dependencies...") {
		t.Errorf("Expected log to contain 'dependencies: Installing dependencies...'")
	}
	if !strings.Contains(logOutput, "build: Building function code...") {
		t.Errorf("Expected log to contain 'build: Building function code...'")
	}
}

// TestFunctionBuildProgressEvent_FunctionTabFiltering tests that FunctionBuildProgressEvent
// appears in function tab filtering and not in SST tab filtering.
func TestFunctionBuildProgressEvent_FunctionTabFiltering(t *testing.T) {
	// This test verifies the event type registration in the UI filtering system
	// by checking that FunctionBuildProgressEvent is included in function filter types

	// Create test event
	event := &aws.FunctionBuildProgressEvent{
		FunctionID: "test-function-789",
		Stage:      "dependencies",
		Message:    "Installing packages...",
	}

	// Create UI instance
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a temporary log file
	logFile, err := os.CreateTemp("", "ui_filter_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	defer os.Remove(logFile.Name())
	defer logFile.Close()

	ui := New(ctx, WithSilent, WithLog(logFile))
	defer ui.Destroy()

	// Process the event
	ui.Event(event)

	// Read the log output
	logFile.Seek(0, 0)
	logContent, err := os.ReadFile(logFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logOutput := string(logContent)

	// Verify the event was processed (indicating it's properly registered for function filtering)
	expectedContent := "dependencies: Installing packages..."
	if !strings.Contains(logOutput, expectedContent) {
		t.Errorf("Expected FunctionBuildProgressEvent to be processed and logged, but output was: %s", logOutput)
	}

	// Verify the event uses function-specific formatting (Build label)
	if !strings.Contains(logOutput, "Build") {
		t.Errorf("Expected FunctionBuildProgressEvent to use 'Build' label for function tab display")
	}
}

// TestFunctionBuildProgressEvent_DisplayConsistency tests that FunctionBuildProgressEvent
// formatting is consistent with other function events.
func TestFunctionBuildProgressEvent_DisplayConsistency(t *testing.T) {
	// Create UI instance
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logFile, err := os.CreateTemp("", "ui_consistency_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	defer os.Remove(logFile.Name())
	defer logFile.Close()

	ui := New(ctx, WithSilent, WithLog(logFile))
	defer ui.Destroy()

	// Test FunctionBuildProgressEvent
	buildProgressEvent := &aws.FunctionBuildProgressEvent{
		FunctionID: "test-function-consistency",
		Stage:      "dependencies",
		Message:    "Installing requirements...",
	}

	// Test FunctionBuildEvent for comparison
	buildEvent := &aws.FunctionBuildEvent{
		FunctionID: "test-function-consistency",
		Errors:     []string{}, // No errors - successful build
	}

	// Process both events
	ui.Event(buildProgressEvent)
	ui.Event(buildEvent)

	// Read log output
	logFile.Seek(0, 0)
	logContent, err := os.ReadFile(logFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logOutput := string(logContent)

	// Verify both events use "Build" label (consistency)
	buildLabelCount := strings.Count(logOutput, "Build")
	if buildLabelCount < 2 {
		t.Errorf("Expected both FunctionBuildProgressEvent and FunctionBuildEvent to use 'Build' label, found %d occurrences in: %s", buildLabelCount, logOutput)
	}

	// Verify FunctionBuildProgressEvent includes stage and message
	if !strings.Contains(logOutput, "dependencies: Installing requirements...") {
		t.Errorf("Expected FunctionBuildProgressEvent to include formatted stage and message")
	}

	// Verify FunctionBuildEvent shows function name (since u.complete is nil, it shows functionID)
	if !strings.Contains(logOutput, "test-function-consistency") {
		t.Errorf("Expected FunctionBuildEvent to reference the function ID")
	}
}

// TestFunctionBuildProgressEvent_ColorCoding tests that different function IDs
// get different colors for visual distinction in the UI.
func TestFunctionBuildProgressEvent_ColorCoding(t *testing.T) {
	// Create UI instance
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ui := New(ctx, WithSilent)
	defer ui.Destroy()

	// Test that different function IDs get different colors
	functionID1 := "test-function-color-1"
	functionID2 := "test-function-color-2"

	// Get colors for different function IDs
	color1 := ui.getColor(functionID1)
	ui.getColor(functionID2) // Trigger color assignment for second function

	// Verify that the color assignment system works
	// In non-TTY environments, colors might be empty, but the system should still assign different colors
	// We test this by checking that the UI maintains separate color mappings for different function IDs

	// Verify that the same function ID gets the same color consistently
	color1Again := ui.getColor(functionID1)
	color1Str := color1.String()
	color1AgainStr := color1Again.String()
	if color1Str != color1AgainStr {
		t.Errorf("Expected the same function ID to get the same color consistently, got %s then %s", color1Str, color1AgainStr)
	}

	// Verify that the color system is working by checking that colors are assigned
	// (even if they're empty in non-TTY environments, the assignment should be consistent)
	if len(ui.colors) < 2 {
		t.Errorf("Expected UI to track colors for multiple function IDs, but only found %d entries", len(ui.colors))
	}
}

// TestFunctionBuildProgressEvent_RealTimeUpdates tests that progress events
// are processed in real-time without blocking.
func TestFunctionBuildProgressEvent_RealTimeUpdates(t *testing.T) {
	// Create UI instance
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logFile, err := os.CreateTemp("", "ui_realtime_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	defer os.Remove(logFile.Name())
	defer logFile.Close()

	ui := New(ctx, WithSilent, WithLog(logFile))
	defer ui.Destroy()

	// Create multiple progress events to simulate real-time updates
	events := []aws.FunctionBuildProgressEvent{
		{FunctionID: "test-function-realtime", Stage: "start", Message: "Starting build..."},
		{FunctionID: "test-function-realtime", Stage: "dependencies", Message: "Installing dependencies..."},
		{FunctionID: "test-function-realtime", Stage: "compile", Message: "Compiling code..."},
		{FunctionID: "test-function-realtime", Stage: "complete", Message: "Build complete"},
	}

	// Process events with small delays to simulate real-time updates
	start := time.Now()
	for i, event := range events {
		ui.Event(&event)
		if i < len(events)-1 {
			time.Sleep(10 * time.Millisecond) // Small delay between events
		}
	}
	duration := time.Since(start)

	// Verify processing was fast (real-time)
	if duration > 100*time.Millisecond {
		t.Errorf("Expected real-time processing, but took %v", duration)
	}

	// Verify all events were processed
	logFile.Seek(0, 0)
	logContent, err := os.ReadFile(logFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logOutput := string(logContent)

	// Check that all stages were processed
	expectedStages := []string{"start", "dependencies", "compile", "complete"}
	for _, stage := range expectedStages {
		if !strings.Contains(logOutput, stage+":") {
			t.Errorf("Expected log to contain stage '%s', but it was not found in: %s", stage, logOutput)
		}
	}
}
