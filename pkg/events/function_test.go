package events

import (
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/bus"
)

func TestFunctionBuildProgressEvent(t *testing.T) {
	// Subscribe to the event
	eventCh := bus.Subscribe(&FunctionBuildProgressEvent{})

	// Create and publish an event
	testEvent := &FunctionBuildProgressEvent{
		FunctionID: "test-function",
		Stage:      "dependencies",
		Message:    "Installing packages...",
	}

	bus.Publish(testEvent)

	// Wait for the event
	select {
	case event := <-eventCh:
		receivedEvent, ok := event.(*FunctionBuildProgressEvent)
		if !ok {
			t.Fatalf("Expected *FunctionBuildProgressEvent, got %T", event)
		}

		if receivedEvent.FunctionID != testEvent.FunctionID {
			t.Errorf("Expected FunctionID %s, got %s", testEvent.FunctionID, receivedEvent.FunctionID)
		}

		if receivedEvent.Stage != testEvent.Stage {
			t.Errorf("Expected Stage %s, got %s", testEvent.Stage, receivedEvent.Stage)
		}

		if receivedEvent.Message != testEvent.Message {
			t.Errorf("Expected Message %s, got %s", testEvent.Message, receivedEvent.Message)
		}

	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for event")
	}
}
