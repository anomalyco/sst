package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/runtime"
	"github.com/sst/sst/v3/pkg/runtime/python"
)

func main() {
	// Subscribe to FunctionBuildProgressEvent
	eventCh := bus.Subscribe(&python.FunctionBuildProgressEvent{})

	// Create a Python runtime
	pythonRuntime := &python.PythonRuntime{}

	// Create a test build input
	buildInput := &runtime.BuildInput{
		FunctionID: "test-function-debug",
		CfgPath:    ".",
		Runtime:    "python3.12",
		Properties: []byte(`{"architecture": "x86_64"}`),
	}

	fmt.Println("Starting build test...")

	// Start listening for events in a goroutine
	go func() {
		for {
			select {
			case event := <-eventCh:
				if buildProgressEvent, ok := event.(*python.FunctionBuildProgressEvent); ok {
					fmt.Printf("Received FunctionBuildProgressEvent: FunctionID=%s, Stage=%s, Message=%s\n",
						buildProgressEvent.FunctionID, buildProgressEvent.Stage, buildProgressEvent.Message)
				}
			case <-time.After(10 * time.Second):
				fmt.Println("Timeout waiting for events")
				return
			}
		}
	}()

	// Try to build
	ctx := context.Background()
	_, err := pythonRuntime.CreateBuildAsset(ctx, buildInput)
	if err != nil {
		fmt.Printf("Build failed: %v\n", err)
	} else {
		fmt.Println("Build completed")
	}

	// Wait a bit for events
	time.Sleep(2 * time.Second)
	fmt.Println("Test completed")
}
