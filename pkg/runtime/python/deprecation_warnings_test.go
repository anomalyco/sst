package python

import (
	"testing"
	"time"
)

func TestNewDeprecationChecker(t *testing.T) {
	checker := NewDeprecationChecker(true)

	if !checker.enableWarnings {
		t.Error("Expected warnings to be enabled")
	}

	if checker.warningHistory == nil {
		t.Error("Expected warning history to be initialized")
	}
}

func TestDeprecationChecker_CheckLayout_WorkspacePatterns(t *testing.T) {
	checker := NewDeprecationChecker(true)

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	// Test old workspace structure
	layout := &LayoutInfo{
		Type:         LayoutTypeWorkspace,
		WorkspaceDir: "/project/src/mypackage",
		PackageName:  "mypackage",
	}

	checker.CheckLayout(layout)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that warning was issued
	if len(warnings) == 0 {
		t.Error("Expected deprecation warning for old workspace structure")
	}

	if len(warnings) > 0 {
		warning := warnings[0]
		if warning.Pattern != "old_workspace_structure" {
			t.Errorf("Expected pattern 'old_workspace_structure', got %s", warning.Pattern)
		}
		if warning.Severity != SeverityWarning {
			t.Errorf("Expected severity 'warning', got %s", warning.Severity)
		}
	}
}

func TestDeprecationChecker_CheckLayout_MissingPyproject(t *testing.T) {
	checker := NewDeprecationChecker(true)

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	// Test missing pyproject.toml
	layout := &LayoutInfo{
		Type:          LayoutTypeWorkspace,
		WorkspaceDir:  "/project",
		PackageName:   "mypackage",
		PyprojectPath: "", // Missing
	}

	checker.CheckLayout(layout)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that warning was issued
	if len(warnings) == 0 {
		t.Error("Expected deprecation warning for missing pyproject.toml")
	}

	if len(warnings) > 0 {
		warning := warnings[0]
		if warning.Pattern != "missing_pyproject" {
			t.Errorf("Expected pattern 'missing_pyproject', got %s", warning.Pattern)
		}
		if warning.Severity != SeverityInfo {
			t.Errorf("Expected severity 'info', got %s", warning.Severity)
		}
	}
}

func TestDeprecationChecker_CheckLayout_FlatPatterns(t *testing.T) {
	checker := NewDeprecationChecker(true)

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	// Test requirements.txt usage
	layout := &LayoutInfo{
		Type:             LayoutTypeFlat,
		WorkspaceDir:     "/project",
		PackageName:      "mypackage",
		RequirementsPath: "/project/requirements.txt",
	}

	checker.CheckLayout(layout)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that warning was issued
	if len(warnings) == 0 {
		t.Error("Expected deprecation warning for requirements.txt usage")
	}

	if len(warnings) > 0 {
		warning := warnings[0]
		if warning.Pattern != "requirements_txt_usage" {
			t.Errorf("Expected pattern 'requirements_txt_usage', got %s", warning.Pattern)
		}
		if warning.Severity != SeverityWarning {
			t.Errorf("Expected severity 'warning', got %s", warning.Severity)
		}
	}
}

func TestDeprecationChecker_CheckLayout_NestedPatterns(t *testing.T) {
	checker := NewDeprecationChecker(true)

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	// Test deep nesting
	layout := &LayoutInfo{
		Type:         LayoutTypeNested,
		WorkspaceDir: "/project",
		PackageName:  "mypackage",
		HandlerPath:  "/project/src/app/handlers/api/v1/handler.py", // Deep nesting
	}

	checker.CheckLayout(layout)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that warning was issued
	if len(warnings) == 0 {
		t.Error("Expected deprecation warning for deep nesting")
	}

	if len(warnings) > 0 {
		warning := warnings[0]
		if warning.Pattern != "deep_nesting" {
			t.Errorf("Expected pattern 'deep_nesting', got %s", warning.Pattern)
		}
		if warning.Severity != SeverityInfo {
			t.Errorf("Expected severity 'info', got %s", warning.Severity)
		}
	}
}

func TestDeprecationChecker_CheckDependencies_PythonVersions(t *testing.T) {
	checker := NewDeprecationChecker(true)

	tests := []struct {
		name             string
		pythonVersion    string
		expectedPattern  string
		expectedSeverity DeprecationSeverity
		shouldWarn       bool
	}{
		{
			name:             "Python 3.7 (deprecated)",
			pythonVersion:    "3.7.12",
			expectedPattern:  "deprecated_python_version",
			expectedSeverity: SeverityError,
			shouldWarn:       true,
		},
		{
			name:             "Python 3.8 (deprecated)",
			pythonVersion:    "3.8.10",
			expectedPattern:  "deprecated_python_version",
			expectedSeverity: SeverityError,
			shouldWarn:       true,
		},
		{
			name:             "Python 3.9 (approaching deprecation)",
			pythonVersion:    "3.9.16",
			expectedPattern:  "approaching_deprecated_python_version",
			expectedSeverity: SeverityWarning,
			shouldWarn:       true,
		},
		{
			name:          "Python 3.11 (current)",
			pythonVersion: "3.11.0",
			shouldWarn:    false,
		},
		{
			name:          "Python 3.12 (current)",
			pythonVersion: "3.12.0",
			shouldWarn:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track warnings
			var warnings []DeprecationWarning
			checker.RegisterCallback(func(warning DeprecationWarning) {
				warnings = append(warnings, warning)
			})

			// Clear previous warnings
			checker.ClearWarningHistory()

			dependencies := &DependencyInfo{
				PythonVersion: tt.pythonVersion,
				Dependencies:  map[string]string{},
			}

			checker.CheckDependencies(dependencies)

			// Give callbacks time to execute
			time.Sleep(100 * time.Millisecond)

			if tt.shouldWarn {
				if len(warnings) == 0 {
					t.Errorf("Expected deprecation warning for Python %s", tt.pythonVersion)
					return
				}

				warning := warnings[0]
				if warning.Pattern != tt.expectedPattern {
					t.Errorf("Expected pattern '%s', got '%s'", tt.expectedPattern, warning.Pattern)
				}
				if warning.Severity != tt.expectedSeverity {
					t.Errorf("Expected severity '%s', got '%s'", tt.expectedSeverity, warning.Severity)
				}
			} else {
				if len(warnings) > 0 {
					t.Errorf("Expected no warnings for Python %s, got %d", tt.pythonVersion, len(warnings))
				}
			}
		})
	}
}

func TestDeprecationChecker_CheckDependencies_Packages(t *testing.T) {
	checker := NewDeprecationChecker(true)

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	dependencies := &DependencyInfo{
		PythonVersion: "3.11.0",
		Dependencies: map[string]string{
			"flask":      "2.0.0",
			"requests":   "2.28.0",
			"fastapi":    "0.95.0", // Not deprecated
			"setuptools": "65.0.0",
		},
	}

	checker.CheckDependencies(dependencies)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that warnings were issued for deprecated packages
	expectedWarnings := []string{"flask", "requests", "setuptools"}
	actualWarnings := make(map[string]bool)

	for _, warning := range warnings {
		if warning.Pattern == "deprecated_package" {
			if pkg, exists := warning.Context["package"]; exists {
				actualWarnings[pkg.(string)] = true
			}
		}
	}

	for _, expectedPkg := range expectedWarnings {
		if !actualWarnings[expectedPkg] {
			t.Errorf("Expected deprecation warning for package '%s'", expectedPkg)
		}
	}

	// Check that no warning was issued for fastapi
	if actualWarnings["fastapi"] {
		t.Error("Did not expect deprecation warning for fastapi")
	}
}

func TestDeprecationChecker_CheckBuildPatterns(t *testing.T) {
	checker := NewDeprecationChecker(true)

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	buildInfo := map[string]interface{}{
		"buildTool":        "setuptools",
		"packagingPattern": "setup.py",
	}

	checker.CheckBuildPatterns(buildInfo)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Check that warnings were issued
	if len(warnings) < 2 {
		t.Errorf("Expected at least 2 warnings, got %d", len(warnings))
	}

	foundBuildToolWarning := false
	foundPackagingWarning := false

	for _, warning := range warnings {
		if warning.Pattern == "deprecated_build_tool" {
			foundBuildToolWarning = true
		}
		if warning.Pattern == "deprecated_packaging" {
			foundPackagingWarning = true
		}
	}

	if !foundBuildToolWarning {
		t.Error("Expected deprecated build tool warning")
	}

	if !foundPackagingWarning {
		t.Error("Expected deprecated packaging warning")
	}
}

func TestDeprecationChecker_WarningDeduplication(t *testing.T) {
	checker := NewDeprecationChecker(true)

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	layout := &LayoutInfo{
		Type:         LayoutTypeWorkspace,
		WorkspaceDir: "/project/src/mypackage",
		PackageName:  "mypackage",
	}

	// Check layout multiple times
	checker.CheckLayout(layout)
	checker.CheckLayout(layout)
	checker.CheckLayout(layout)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Should only get one warning due to deduplication
	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning due to deduplication, got %d", len(warnings))
	}
}

func TestDeprecationChecker_DisabledWarnings(t *testing.T) {
	checker := NewDeprecationChecker(false) // Disabled

	// Track warnings
	var warnings []DeprecationWarning
	checker.RegisterCallback(func(warning DeprecationWarning) {
		warnings = append(warnings, warning)
	})

	layout := &LayoutInfo{
		Type:         LayoutTypeWorkspace,
		WorkspaceDir: "/project/src/mypackage",
		PackageName:  "mypackage",
	}

	checker.CheckLayout(layout)

	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)

	// Should not get any warnings when disabled
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings when disabled, got %d", len(warnings))
	}
}
