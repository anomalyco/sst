package project

import (
	"encoding/json"
	"testing"
)

func TestValidateStateRetention(t *testing.T) {
	t.Parallel()

	for _, value := range []int{0, -1} {
		value := value
		if err := validateState(&State{Retention: &value}); err == nil {
			t.Fatalf("expected retention %d to fail validation", value)
		}
	}

	value := 30
	if err := validateState(&State{Retention: &value}); err != nil {
		t.Fatalf("expected positive retention to pass validation: %v", err)
	}
	if err := validateState(nil); err != nil {
		t.Fatalf("expected omitted state to pass validation: %v", err)
	}
}

func TestStateRetentionRejectsFractions(t *testing.T) {
	t.Parallel()

	var state State
	if err := json.Unmarshal([]byte(`{"retention":1.5}`), &state); err == nil {
		t.Fatal("expected fractional retention to fail decoding")
	}
}
