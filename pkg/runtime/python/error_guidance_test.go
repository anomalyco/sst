package python

import (
	"errors"
	"strings"
	"testing"
)

// TestHandlerSuggestions tests that handler not found errors provide actionable suggestions
func TestHandlerSuggestions(t *testing.T) {
	tests := []struct {
		name                string
		handler             string
		projectRoot         string
		searchPaths         []string
		expectedSuggestions []string
	}{
		{
			name:        "simple handler",
			handler:     "main.py",
			projectRoot: "/project",
			searchPaths: []string{"/project"},
			expectedSuggestions: []string{
				"Create the handler file: main.py",
				"Ensure your handler path in sst.config.ts matches your actual file location",
			},
		},
		{
			name:        "handler without extension",
			handler:     "handler",
			projectRoot: "/project",
			searchPaths: []string{"/project"},
			expectedSuggestions: []string{
				"Add .py extension: handler.py",
			},
		},
		{
			name:        "nested handler",
			handler:     "api/users.py",
			projectRoot: "/project",
			searchPaths: []string{"/project"},
			expectedSuggestions: []string{
				"Try without subdirectory: users.py",
				"Create directory structure: mkdir -p api",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := GenerateHandlerSuggestions(tt.handler, tt.projectRoot, tt.searchPaths)

			// Check that expected suggestions are present
			for _, expectedSuggestion := range tt.expectedSuggestions {
				found := false
				for _, suggestion := range suggestions {
					if strings.Contains(suggestion, expectedSuggestion) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected suggestion containing '%s' not found in: %v", expectedSuggestion, suggestions)
				}
			}

			// Verify suggestions are actionable (contain action verbs)
			actionVerbs := []string{"create", "add", "check", "verify", "ensure", "try"}
			hasActionable := false
			for _, suggestion := range suggestions {
				lowerSuggestion := strings.ToLower(suggestion)
				for _, verb := range actionVerbs {
					if strings.Contains(lowerSuggestion, verb) {
						hasActionable = true
						break
					}
				}
				if hasActionable {
					break
				}
			}

			if !hasActionable {
				t.Errorf("Suggestions should be actionable: %v", suggestions)
			}
		})
	}
}

// TestConfigurationSuggestions tests that configuration errors provide helpful guidance
func TestConfigurationSuggestions(t *testing.T) {
	tests := []struct {
		name                string
		issue               string
		expectedSuggestions []string
	}{
		{
			name:  "missing pyproject.toml",
			issue: "missing",
			expectedSuggestions: []string{
				"Create a pyproject.toml file in your project root directory",
				"Run 'uv init' to create a basic Python project structure",
			},
		},
		{
			name:  "invalid TOML syntax",
			issue: "invalid",
			expectedSuggestions: []string{
				"Check TOML syntax - ensure proper quotes and brackets",
				"Validate your TOML at https://www.toml-lint.com/",
			},
		},
		{
			name:  "missing project name",
			issue: "name",
			expectedSuggestions: []string{
				"Add project name: [project] name = \"your-project-name\"",
				"Use lowercase with hyphens for project names",
			},
		},
		{
			name:  "dependencies format error",
			issue: "dependencies",
			expectedSuggestions: []string{
				"Fix dependencies format: dependencies = [\"package>=1.0.0\"]",
				"Use 'uv add package-name' to add dependencies correctly",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := GenerateConfigurationSuggestions("/project/pyproject.toml", tt.issue)

			// Check that expected suggestions are present
			for _, expectedSuggestion := range tt.expectedSuggestions {
				found := false
				for _, suggestion := range suggestions {
					if strings.Contains(suggestion, expectedSuggestion) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected suggestion containing '%s' not found in: %v", expectedSuggestion, suggestions)
				}
			}
		})
	}
}

// TestErrorMessageQuality tests that error messages are clear and don't contain layout-specific terms
func TestErrorMessageQuality(t *testing.T) {
	tests := []struct {
		name        string
		createError func() *PythonRuntimeError
	}{
		{
			name: "handler not found error",
			createError: func() *PythonRuntimeError {
				return NewHandlerNotFoundError("main.py", []string{"/project"}, nil)
			},
		},
		{
			name: "configuration error",
			createError: func() *PythonRuntimeError {
				return NewConfigurationError("Invalid pyproject.toml", "/project/pyproject.toml", "parse error", "fix syntax")
			},
		},
		{
			name: "project structure error",
			createError: func() *PythonRuntimeError {
				return NewProjectStructureError("Invalid structure", "/project", "main.py")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createError()

			// Should not contain legacy layout-specific terms
			errorStr := strings.ToLower(err.Error())
			legacyTerms := []string{"workspace layout", "flat layout", "nested layout"}
			for _, term := range legacyTerms {
				if strings.Contains(errorStr, term) {
					t.Errorf("Error message should not contain legacy layout-specific term '%s': %s", term, errorStr)
				}
			}

			// Should contain actionable guidance
			if len(err.Suggestions) == 0 {
				t.Error("Error should provide suggestions")
			}

			// Should have context information
			if len(err.Context) == 0 {
				t.Error("Error should provide context")
			}

			// Suggestions should be actionable
			actionVerbs := []string{"create", "add", "check", "verify", "ensure", "run", "use", "fix"}
			hasActionableSuggestion := false
			for _, suggestion := range err.Suggestions {
				lowerSuggestion := strings.ToLower(suggestion)
				for _, verb := range actionVerbs {
					if strings.Contains(lowerSuggestion, verb) {
						hasActionableSuggestion = true
						break
					}
				}
				if hasActionableSuggestion {
					break
				}
			}

			if !hasActionableSuggestion {
				t.Errorf("Error suggestions should be actionable: %v", err.Suggestions)
			}
		})
	}
}

// TestLegacyErrorDetection tests that we can identify old layout-specific errors
func TestLegacyErrorDetection(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg string
		isLegacy bool
	}{

		{
			name:     "workspace invalid error",
			errorMsg: "workspace invalid configuration",
			isLegacy: true,
		},
		{
			name:     "regular error",
			errorMsg: "file not found",
			isLegacy: false,
		},
		{
			name:     "nil error",
			errorMsg: "",
			isLegacy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errorMsg != "" {
				err = errors.New(tt.errorMsg)
			}

			got := IsLegacyError(err)
			if got != tt.isLegacy {
				t.Errorf("IsLegacyError() = %v, want %v", got, tt.isLegacy)
			}
		})
	}
}

// TestErrorConversionImprovement tests that converted errors have better guidance
func TestErrorConversionImprovement(t *testing.T) {
	tests := []struct {
		name          string
		originalError string
		handler       string
		projectRoot   string
		expectedType  ErrorType
	}{
		{
			name:          "handler not found",
			originalError: "handler main.py not found",
			handler:       "main.py",
			projectRoot:   "/project",
			expectedType:  ErrorTypeHandlerNotFound,
		},
		{
			name:          "configuration issue",
			originalError: "pyproject.toml invalid",
			handler:       "main.py",
			projectRoot:   "/project",
			expectedType:  ErrorTypeConfigurationError,
		},
		{
			name:          "project structure issue",
			originalError: "workspace structure invalid",
			handler:       "main.py",
			projectRoot:   "/project",
			expectedType:  ErrorTypeProjectStructure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalErr := errors.New(tt.originalError)
			convertedErr := ConvertLegacyErrorToModernError(originalErr, tt.handler, tt.projectRoot)

			// Should have the expected error type
			if convertedErr.Type != tt.expectedType {
				t.Errorf("Expected error type %s, got %s", tt.expectedType, convertedErr.Type)
			}

			// Should have actionable suggestions
			if len(convertedErr.Suggestions) == 0 {
				t.Error("Converted error should have suggestions")
			}

			// Should have context information
			if len(convertedErr.Context) == 0 {
				t.Error("Converted error should have context")
			}

			// Should not contain legacy layout-specific terms in suggestions
			for _, suggestion := range convertedErr.Suggestions {
				lowerSuggestion := strings.ToLower(suggestion)
				legacyTerms := []string{"layout detection", "layout type", "layout classification"}
				for _, term := range legacyTerms {
					if strings.Contains(lowerSuggestion, term) {
						t.Errorf("Converted error suggestion should not contain legacy layout-specific term '%s': %s", term, suggestion)
					}
				}
			}
		})
	}
}
