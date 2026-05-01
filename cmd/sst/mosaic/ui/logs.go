package ui

import (
	"encoding/json"
)

func compactWorkflowLogLine(line string) (string, bool) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(line), &parsed); err != nil {
		return "", false
	}

	if !isWorkflowLogEnvelope(parsed) {
		return "", false
	}

	message, ok := parsed["message"]
	if !ok {
		return "", false
	}

	switch value := message.(type) {
	case string:
		return value, true
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return "", false
		}
		return string(data), true
	}
}

func isWorkflowLogEnvelope(parsed map[string]any) bool {
	if _, hasMessage := parsed["message"]; !hasMessage {
		return false
	}

	if !hasStringField(parsed, "timestamp") {
		return false
	}

	if !hasStringField(parsed, "level") {
		return false
	}

	if !hasStringField(parsed, "requestId") && !hasStringField(parsed, "AWSrequestId") {
		return false
	}

	if hasStringField(parsed, "executionArn") {
		return true
	}

	if hasStringField(parsed, "execution_arn") {
		return true
	}

	return false
}

func hasStringField(parsed map[string]any, key string) bool {
	value, ok := parsed[key].(string)
	return ok && value != ""
}

func (u *UI) formatFunctionLogLine(line string) string {
	formatted, ok := compactWorkflowLogLine(line)
	if !ok {
		return line
	}

	return formatted
}
