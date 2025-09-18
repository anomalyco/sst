package python

import (
	"errors"
	"strings"
	"testing"
)

func TestNewHandlerNotFoundError(t *testing.T) {
	tests := []struct {
		name        string
		handler     string
		searchPaths []string
		suggestions []string
		wantMessage string
		wantContext map[string]interface{}
	}{
		{
			name:        "basic handler not found",
			handler:     "main.py",
			searchPaths: []string{"/project", "/project/src"},
			suggestions: []string{"Create the handler file at: main.py"},
			wantMessage: "Python handler 'main.py' not found",
			wantContext: map[string]interface{}{
				"handler":     "main.py",
				"searchPaths": []string{"/project", "/project/src"},
			},
		},
		{
			name:        "nested handler not found",
			handler:     "api/handlers/main.py",
			searchPaths: []string{"/project", "/project/src", "/project/app"},
			suggestions: []string{"Create the handler file at: api/handlers/main.py", "Check directory structure"},
			wantMessage: "Python handler 'api/handlers/main.py' not found",
			wantContext: map[string]interface{}{
				"handler":     "api/handlers/main.py",
				"searchPaths": []string{"/project", "/project/src", "/project/app"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewHandlerNotFoundError(tt.handler, tt.searchPaths, tt.suggestions)

			// Check error type
			if err.Type != ErrorTypeHandlerNotFound {
				t.Errorf("Expected error type %s, got %s", ErrorTypeHandlerNotFound, err.Type)
			}

			// Check message
			if err.Message != tt.wantMessage {
				t.Errorf("Expected message %q, got %q", tt.wantMessage, err.Message)
			}

			// Check context
			for key, expectedValue := range tt.wantContext {
				if actualValue, exists := err.Context[key]; !exists {
					t.Errorf("Expected context key %s to exist", key)
				} else {
					// Handle slice comparison
					if key == "searchPaths" {
						expectedSlice := expectedValue.([]string)
						actualSlice := actualValue.([]string)
						if len(expectedSlice) != len(actualSlice) {
							t.Errorf("Expected searchPaths length %d, got %d", len(expectedSlice), len(actualSlice))
						}
						for i, expected := range expectedSlice {
							if i < len(actualSlice) && actualSlice[i] != expected {
								t.Errorf("Expected searchPaths[%d] = %s, got %s", i, expected, actualSlice[i])
							}
						}
					} else if actualValue != expectedValue {
						t.Errorf("Expected context[%s] = %v, got %v", key, expectedValue, actualValue)
					}
				}
			}

			// Check suggestions
			if len(tt.suggestions) > 0 {
				if len(err.Suggestions) == 0 {
					t.Error("Expected suggestions to be present")
				}
				for _, expectedSuggestion := range tt.suggestions {
					found := false
					for _, actualSuggestion := range err.Suggestions {
						if actualSuggestion == expectedSuggestion {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected suggestion %q not found in %v", expectedSuggestion, err.Suggestions)
					}
				}
			}

			// Check recovery actions
			if len(err.RecoveryActions) == 0 {
				t.Error("Expected recovery actions to be present")
			}

			// Verify error message format
			errorStr := err.Error()
			if !strings.Contains(errorStr, tt.wantMessage) {
				t.Errorf("Error string should contain message: %s", errorStr)
			}
		})
	}
}

func TestNewConfigurationError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		configFile  string
		issue       string
		fix         string
		wantType    ErrorType
		wantContext map[string]interface{}
	}{
		{
			name:       "missing pyproject.toml",
			message:    "pyproject.toml not found",
			configFile: "/project/pyproject.toml",
			issue:      "missing configuration file",
			fix:        "create pyproject.toml",
			wantType:   ErrorTypeConfigurationError,
			wantContext: map[string]interface{}{
				"configFile": "/project/pyproject.toml",
				"issue":      "missing configuration file",
				"fix":        "create pyproject.toml",
			},
		},
		{
			name:       "invalid TOML syntax",
			message:    "failed to parse pyproject.toml",
			configFile: "/project/pyproject.toml",
			issue:      "invalid TOML syntax",
			fix:        "fix TOML formatting",
			wantType:   ErrorTypeConfigurationError,
			wantContext: map[string]interface{}{
				"configFile": "/project/pyproject.toml",
				"issue":      "invalid TOML syntax",
				"fix":        "fix TOML formatting",
			},
		},
		{
			name:       "missing project name",
			message:    "project name is required",
			configFile: "/project/pyproject.toml",
			issue:      "missing project name",
			fix:        "add project name to [project] section",
			wantType:   ErrorTypeConfigurationError,
			wantContext: map[string]interface{}{
				"configFile": "/project/pyproject.toml",
				"issue":      "missing project name",
				"fix":        "add project name to [project] section",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewConfigurationError(tt.message, tt.configFile, tt.issue, tt.fix)

			// Check error type
			if err.Type != tt.wantType {
				t.Errorf("Expected error type %s, got %s", tt.wantType, err.Type)
			}

			// Check message
			if err.Message != tt.message {
				t.Errorf("Expected message %q, got %q", tt.message, err.Message)
			}

			// Check context
			for key, expectedValue := range tt.wantContext {
				if actualValue, exists := err.Context[key]; !exists {
					t.Errorf("Expected context key %s to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected context[%s] = %v, got %v", key, expectedValue, actualValue)
				}
			}

			// Check that suggestions are generated based on issue type
			if len(err.Suggestions) == 0 {
				t.Error("Expected suggestions to be generated")
			}

			// Verify specific suggestions based on issue type
			issueType := strings.ToLower(tt.issue)
			switch {
			case strings.Contains(issueType, "missing"):
				found := false
				for _, suggestion := range err.Suggestions {
					if strings.Contains(strings.ToLower(suggestion), "create") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected suggestion about creating file for missing issue")
				}
			case strings.Contains(issueType, "invalid") || strings.Contains(issueType, "syntax"):
				found := false
				for _, suggestion := range err.Suggestions {
					if strings.Contains(strings.ToLower(suggestion), "syntax") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected suggestion about syntax for invalid issue")
				}
			case strings.Contains(issueType, "name"):
				found := false
				for _, suggestion := range err.Suggestions {
					if strings.Contains(strings.ToLower(suggestion), "project") && strings.Contains(strings.ToLower(suggestion), "name") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected suggestion about project name")
				}
			}
		})
	}
}

func TestNewProjectStructureError(t *testing.T) {
	err := NewProjectStructureError("Invalid project structure", "/project", "handler.py")

	// Check error type
	if err.Type != ErrorTypeProjectStructure {
		t.Errorf("Expected error type %s, got %s", ErrorTypeProjectStructure, err.Type)
	}

	// Check context
	expectedContext := map[string]interface{}{
		"projectRoot": "/project",
		"handlerPath": "handler.py",
	}

	for key, expectedValue := range expectedContext {
		if actualValue, exists := err.Context[key]; !exists {
			t.Errorf("Expected context key %s to exist", key)
		} else if actualValue != expectedValue {
			t.Errorf("Expected context[%s] = %v, got %v", key, expectedValue, actualValue)
		}
	}

	// Check suggestions
	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be present")
	}

	// Check recovery actions
	if len(err.RecoveryActions) == 0 {
		t.Error("Expected recovery actions to be present")
	}
}

func TestGenerateHandlerSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		handler     string
		projectRoot string
		searchPaths []string
		wantCount   int
		wantContain []string
	}{
		{
			name:        "simple handler",
			handler:     "main.py",
			projectRoot: "/project",
			searchPaths: []string{"/project"},
			wantCount:   5, // At least 5 suggestions
			wantContain: []string{"Create the handler file: main.py"},
		},
		{
			name:        "handler without extension",
			handler:     "main",
			projectRoot: "/project",
			searchPaths: []string{"/project"},
			wantCount:   5,
			wantContain: []string{"Add .py extension: main.py"},
		},
		{
			name:        "nested handler",
			handler:     "api/main.py",
			projectRoot: "/project",
			searchPaths: []string{"/project"},
			wantCount:   5,
			wantContain: []string{"Try without subdirectory: main.py"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := GenerateHandlerSuggestions(tt.handler, tt.projectRoot, tt.searchPaths)

			if len(suggestions) < tt.wantCount {
				t.Errorf("Expected at least %d suggestions, got %d", tt.wantCount, len(suggestions))
			}

			for _, expectedContent := range tt.wantContain {
				found := false
				for _, suggestion := range suggestions {
					if strings.Contains(suggestion, expectedContent) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected suggestion containing %q not found in %v", expectedContent, suggestions)
				}
			}
		})
	}
}

func TestGenerateConfigurationSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		issue       string
		wantContain []string
	}{
		{
			name:       "missing file",
			configFile: "pyproject.toml",
			issue:      "missing configuration file",
			wantContain: []string{
				"Create a pyproject.toml file",
				"uv init",
			},
		},
		{
			name:       "invalid syntax",
			configFile: "pyproject.toml",
			issue:      "invalid TOML syntax",
			wantContain: []string{
				"TOML syntax",
				"validate",
			},
		},
		{
			name:       "missing name",
			configFile: "pyproject.toml",
			issue:      "missing project name",
			wantContain: []string{
				"project name",
				"[project]",
			},
		},
		{
			name:       "dependencies issue",
			configFile: "pyproject.toml",
			issue:      "invalid dependencies format",
			wantContain: []string{
				"dependencies",
				"uv add",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := GenerateConfigurationSuggestions(tt.configFile, tt.issue)

			if len(suggestions) == 0 {
				t.Error("Expected suggestions to be generated")
			}

			for _, expectedContent := range tt.wantContain {
				found := false
				for _, suggestion := range suggestions {
					if strings.Contains(strings.ToLower(suggestion), strings.ToLower(expectedContent)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected suggestion containing %q not found in %v", expectedContent, suggestions)
				}
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		context   string
		wantType  ErrorType
		wantRetry bool
	}{
		{
			name:      "handler not found error",
			err:       errors.New("handler main.py not found"),
			context:   "file resolution",
			wantType:  ErrorTypeHandlerNotFound,
			wantRetry: false,
		},
		{
			name:      "pyproject.toml error",
			err:       errors.New("failed to parse pyproject.toml"),
			context:   "configuration",
			wantType:  ErrorTypeConfigurationError,
			wantRetry: false,
		},
		{
			name:      "network error",
			err:       errors.New("network connection failed"),
			context:   "dependency resolution",
			wantType:  ErrorTypeNetwork,
			wantRetry: true,
		},
		{
			name:      "build error",
			err:       errors.New("build compilation failed"),
			context:   "build process",
			wantType:  ErrorTypeBuildFailed,
			wantRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedErr := WrapError(tt.err, tt.context)

			if wrappedErr == nil {
				t.Fatal("Expected wrapped error, got nil")
			}

			if wrappedErr.Type != tt.wantType {
				t.Errorf("Expected error type %s, got %s", tt.wantType, wrappedErr.Type)
			}

			if wrappedErr.IsRetryable() != tt.wantRetry {
				t.Errorf("Expected retryable %v, got %v", tt.wantRetry, wrappedErr.IsRetryable())
			}

			if wrappedErr.Cause != tt.err {
				t.Error("Expected original error to be preserved as cause")
			}

			if !strings.Contains(wrappedErr.Error(), tt.err.Error()) {
				t.Error("Expected wrapped error to contain original error message")
			}
		})
	}
}

func TestIsLegacyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "project resolution error",
			err:  errors.New("project resolution failed"),
			want: true,
		},
		{
			name: "handler not found error",
			err:  errors.New("handler not found"),
			want: true,
		},
		{
			name: "workspace invalid error",
			err:  errors.New("workspace invalid configuration"),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("file not found"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLegacyError(tt.err)
			if got != tt.want {
				t.Errorf("IsLegacyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertLegacyErrorToModernError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		handler     string
		projectRoot string
		wantType    ErrorType
		wantNil     bool
	}{
		{
			name:        "handler not found error",
			err:         errors.New("project resolution failed: handler main.py not found"),
			handler:     "main.py",
			projectRoot: "/project",
			wantType:    ErrorTypeHandlerNotFound,
			wantNil:     false,
		},
		{
			name:        "pyproject.toml configuration error",
			err:         errors.New("project resolution failed: pyproject.toml invalid"),
			handler:     "main.py",
			projectRoot: "/project",
			wantType:    ErrorTypeConfigurationError,
			wantNil:     false,
		},
		{
			name:        "project structure error",
			err:         errors.New("project structure invalid"),
			handler:     "main.py",
			projectRoot: "/project",
			wantType:    ErrorTypeProjectStructure,
			wantNil:     false,
		},
		{
			name:        "nil error",
			err:         nil,
			handler:     "main.py",
			projectRoot: "/project",
			wantType:    "",
			wantNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertLegacyErrorToModernError(tt.err, tt.handler, tt.projectRoot)

			if tt.wantNil {
				if got != nil {
					t.Errorf("Expected nil error, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("Expected error, got nil")
			}

			if got.Type != tt.wantType {
				t.Errorf("Expected error type %s, got %s", tt.wantType, got.Type)
			}

			// Check that context is properly set
			if _, exists := got.Context["handler"]; !exists && tt.wantType == ErrorTypeHandlerNotFound {
				t.Error("Expected handler context for handler not found error")
			}

			if _, exists := got.Context["projectRoot"]; !exists && tt.wantType == ErrorTypeProjectStructure {
				t.Error("Expected projectRoot context for project structure error")
			}
		})
	}
}

func TestErrorMessageClarity(t *testing.T) {
	// Test that error messages are clear and actionable
	tests := []struct {
		name     string
		errorFn  func() *PythonRuntimeError
		wantKeys []string // Keys that should be present in the error message
	}{
		{
			name: "handler not found error clarity",
			errorFn: func() *PythonRuntimeError {
				return NewHandlerNotFoundError("api/main.py", []string{"/project", "/project/src"}, nil)
			},
			wantKeys: []string{"handler", "not found", "api/main.py"},
		},
		{
			name: "configuration error clarity",
			errorFn: func() *PythonRuntimeError {
				return NewConfigurationError("Invalid pyproject.toml", "/project/pyproject.toml", "missing name", "add project name")
			},
			wantKeys: []string{"pyproject.toml", "Invalid"},
		},
		{
			name: "project structure error clarity",
			errorFn: func() *PythonRuntimeError {
				return NewProjectStructureError("Project structure is invalid", "/project", "handler.py")
			},
			wantKeys: []string{"Project structure", "invalid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFn()
			errorMsg := err.Error()

			for _, key := range tt.wantKeys {
				if !strings.Contains(strings.ToLower(errorMsg), strings.ToLower(key)) {
					t.Errorf("Error message should contain %q: %s", key, errorMsg)
				}
			}

			// Check that suggestions are actionable (contain verbs)
			actionVerbs := []string{"create", "check", "add", "use", "try", "ensure", "verify"}
			hasActionableSuggestion := false
			for _, suggestion := range err.Suggestions {
				suggestionLower := strings.ToLower(suggestion)
				for _, verb := range actionVerbs {
					if strings.Contains(suggestionLower, verb) {
						hasActionableSuggestion = true
						break
					}
				}
				if hasActionableSuggestion {
					break
				}
			}

			if !hasActionableSuggestion {
				t.Errorf("Error should have at least one actionable suggestion with action verbs: %v", err.Suggestions)
			}
		})
	}
}
