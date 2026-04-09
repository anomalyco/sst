package ui

import (
	"testing"
)

func TestCompactWorkflowLogLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{
			name:     "plain text passthrough",
			input:    "Executing step 1: Hello from the invoker",
			expected: "",
			ok:       false,
		},
		{
			name:     "workflow string message",
			input:    `{"requestId":"a8220e1b-1f9b-4529-a40f-9e3419bc494f","timestamp":"2026-04-07T15:20:59.866Z","level":"INFO","executionArn":"arn:aws:lambda:us-east-1:123456789012:function:workflow","operationId":"c4ca4238a0b92382","attempt":1,"message":"Executing step 1: Hello from the invoker"}`,
			expected: "Executing step 1: Hello from the invoker",
			ok:       true,
		},
		{
			name:     "workflow object message",
			input:    `{"requestId":"a8220e1b-1f9b-4529-a40f-9e3419bc494f","timestamp":"2026-04-07T15:21:01.611Z","level":"INFO","executionArn":"arn:aws:lambda:us-east-1:123456789012:function:workflow","operationId":"eccbc87e4b5ce2fe","attempt":1,"message":{"callbackResult":"{\"message\":\"Hello from the invoker\"}","step1":"Hello from the invoker"}}`,
			expected: `{"callbackResult":"{\"message\":\"Hello from the invoker\"}","step1":"Hello from the invoker"}`,
			ok:       true,
		},
		{
			name:     "python workflow message",
			input:    `{"timestamp":"2026-04-07T15:21:01.611Z","level":"INFO","execution_arn":"arn:aws:lambda:us-east-1:123456789012:function:workflow","message":{"step":"complete","ok":true},"requestId":"req-123"}`,
			expected: `{"ok":true,"step":"complete"}`,
			ok:       true,
		},
		{
			name:     "raw user json passthrough",
			input:    `{"timestamp":"2026-04-07T15:21:01.611Z","level":"INFO","message":{"step":"complete","ok":true},"requestId":"req-123"}`,
			expected: "",
			ok:       false,
		},
		{
			name:     "custom json with execution arn passthrough",
			input:    `{"executionArn":"arn:aws:lambda:us-east-1:123456789012:function:workflow","message":{"step":"complete","ok":true}}`,
			expected: "",
			ok:       false,
		},
		{
			name:     "missing request id passthrough",
			input:    `{"timestamp":"2026-04-07T15:21:01.611Z","level":"INFO","executionArn":"arn:aws:lambda:us-east-1:123456789012:function:workflow","message":"hello"}`,
			expected: "",
			ok:       false,
		},
		{
			name:     "malformed json passthrough",
			input:    `{"timestamp":"2026-04-07T15:21:01.611Z","message":`,
			expected: "",
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, ok := compactWorkflowLogLine(tt.input)
			if ok != tt.ok {
				t.Fatalf("expected ok=%v, got %v", tt.ok, ok)
			}
			if actual != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestFormatFunctionLogLine(t *testing.T) {
	line := `{"timestamp":"2026-04-07T15:20:59.866Z","level":"INFO","executionArn":"arn:aws:lambda:us-east-1:123456789012:function:workflow","message":"Executing step 1: Hello from the invoker","requestId":"req-123"}`

	u := &UI{}

	if actual := u.formatFunctionLogLine(line); actual != "Executing step 1: Hello from the invoker" {
		t.Fatalf("expected compacted workflow log, got %q", actual)
	}

	rawJSON := `{"timestamp":"2026-04-07T15:20:59.866Z","level":"INFO","message":{"user":true},"requestId":"req-123"}`
	if actual := u.formatFunctionLogLine(rawJSON); actual != rawJSON {
		t.Fatalf("expected raw user json to be unchanged, got %q", actual)
	}
}
