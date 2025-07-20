package python

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// DeprecationWarning represents a warning about deprecated patterns
type DeprecationWarning struct {
	// Pattern is the deprecated pattern detected
	Pattern string `json:"pattern"`

	// Message is the warning message
	Message string `json:"message"`

	// Suggestion is the recommended migration path
	Suggestion string `json:"suggestion"`

	// Severity indicates how urgent the migration is
	Severity DeprecationSeverity `json:"severity"`

	// Timestamp is when the warning was issued
	Timestamp time.Time `json:"timestamp"`

	// Context contains additional information
	Context map[string]interface{} `json:"context,omitempty"`
}

// DeprecationSeverity represents the severity of a deprecation warning
type DeprecationSeverity string

const (
	SeverityInfo     DeprecationSeverity = "info"
	SeverityWarning  DeprecationSeverity = "warning"
	SeverityError    DeprecationSeverity = "error"
	SeverityCritical DeprecationSeverity = "critical"
)

// DeprecationCallback is called when deprecation warnings are issued
type DeprecationCallback func(warning DeprecationWarning)

// DeprecationChecker checks for deprecated patterns and issues warnings
type DeprecationChecker struct {
	// callbacks are functions to call when warnings are issued
	callbacks []DeprecationCallback

	// enableWarnings controls whether warnings are issued
	enableWarnings bool

	// warningHistory tracks issued warnings to avoid duplicates
	warningHistory map[string]time.Time
}

// NewDeprecationChecker creates a new deprecation checker
func NewDeprecationChecker(enableWarnings bool) *DeprecationChecker {
	return &DeprecationChecker{
		enableWarnings: enableWarnings,
		warningHistory: make(map[string]time.Time),
	}
}

// RegisterCallback adds a callback for deprecation warnings
func (dc *DeprecationChecker) RegisterCallback(callback DeprecationCallback) {
	dc.callbacks = append(dc.callbacks, callback)
}

// CheckLayout checks for deprecated layout patterns
func (dc *DeprecationChecker) CheckLayout(layout *LayoutInfo) {
	if !dc.enableWarnings || layout == nil {
		return
	}

	// Check for deprecated workspace patterns
	if layout.Type == LayoutTypeWorkspace {
		dc.checkWorkspacePatterns(layout)
	}

	// Check for deprecated flat patterns
	if layout.Type == LayoutTypeFlat {
		dc.checkFlatPatterns(layout)
	}

	// Check for deprecated nested patterns
	if layout.Type == LayoutTypeNested {
		dc.checkNestedPatterns(layout)
	}
}

// checkWorkspacePatterns checks for deprecated workspace patterns
func (dc *DeprecationChecker) checkWorkspacePatterns(layout *LayoutInfo) {
	// Check for old-style workspace structure
	if strings.Contains(layout.WorkspaceDir, "src") && strings.Contains(layout.WorkspaceDir, layout.PackageName) {
		dc.issueWarning(DeprecationWarning{
			Pattern:    "old_workspace_structure",
			Message:    "Using deprecated workspace structure with src/{package} pattern",
			Suggestion: "Consider migrating to a flatter structure or using modern Python project layouts",
			Severity:   SeverityWarning,
			Context: map[string]interface{}{
				"workspaceDir": layout.WorkspaceDir,
				"packageName":  layout.PackageName,
			},
		})
	}

	// Check for missing pyproject.toml
	if layout.PyprojectPath == "" {
		dc.issueWarning(DeprecationWarning{
			Pattern:    "missing_pyproject",
			Message:    "No pyproject.toml found in workspace",
			Suggestion: "Add a pyproject.toml file to define your project configuration and dependencies",
			Severity:   SeverityInfo,
			Context: map[string]interface{}{
				"workspaceDir": layout.WorkspaceDir,
			},
		})
	}
}

// checkFlatPatterns checks for deprecated flat patterns
func (dc *DeprecationChecker) checkFlatPatterns(layout *LayoutInfo) {
	// Check for requirements.txt usage
	if layout.RequirementsPath != "" {
		dc.issueWarning(DeprecationWarning{
			Pattern:    "requirements_txt_usage",
			Message:    "Using requirements.txt instead of pyproject.toml",
			Suggestion: "Migrate to pyproject.toml for better dependency management and modern Python tooling",
			Severity:   SeverityWarning,
			Context: map[string]interface{}{
				"requirementsPath": layout.RequirementsPath,
			},
		})
	}

	// Check for setup.py usage
	if layout.SetupPyPath != "" {
		dc.issueWarning(DeprecationWarning{
			Pattern:    "setup_py_usage",
			Message:    "Using setup.py instead of pyproject.toml",
			Suggestion: "Migrate to pyproject.toml for modern Python packaging standards",
			Severity:   SeverityWarning,
			Context: map[string]interface{}{
				"setupPyPath": layout.SetupPyPath,
			},
		})
	}
}

// checkNestedPatterns checks for deprecated nested patterns
func (dc *DeprecationChecker) checkNestedPatterns(layout *LayoutInfo) {
	// Check for overly complex nesting
	pathDepth := strings.Count(layout.HandlerPath, "/")
	if pathDepth > 3 {
		dc.issueWarning(DeprecationWarning{
			Pattern:    "deep_nesting",
			Message:    "Handler is deeply nested which may impact performance",
			Suggestion: "Consider flattening your project structure for better maintainability",
			Severity:   SeverityInfo,
			Context: map[string]interface{}{
				"handlerPath": layout.HandlerPath,
				"depth":       pathDepth,
			},
		})
	}
}

// CheckDependencies checks for deprecated dependency patterns
func (dc *DeprecationChecker) CheckDependencies(dependencies *DependencyInfo) {
	if !dc.enableWarnings || dependencies == nil {
		return
	}

	// Check for deprecated Python versions
	if dependencies.PythonVersion != "" {
		dc.checkPythonVersion(dependencies.PythonVersion)
	}

	// Check for deprecated packages
	dc.checkDeprecatedPackages(dependencies.Dependencies)
}

// checkPythonVersion checks for deprecated Python versions
func (dc *DeprecationChecker) checkPythonVersion(version string) {
	// Check for Python 3.8 and below (deprecated)
	if strings.HasPrefix(version, "3.8") || strings.HasPrefix(version, "3.7") || strings.HasPrefix(version, "3.6") {
		dc.issueWarning(DeprecationWarning{
			Pattern:    "deprecated_python_version",
			Message:    fmt.Sprintf("Python %s is deprecated and will lose support soon", version),
			Suggestion: "Upgrade to Python 3.9 or later for continued support and security updates",
			Severity:   SeverityError,
			Context: map[string]interface{}{
				"pythonVersion": version,
			},
		})
	}

	// Check for Python 3.9 (approaching deprecation)
	if strings.HasPrefix(version, "3.9") {
		dc.issueWarning(DeprecationWarning{
			Pattern:    "approaching_deprecated_python_version",
			Message:    fmt.Sprintf("Python %s will be deprecated soon", version),
			Suggestion: "Consider upgrading to Python 3.11 or 3.12 for better performance and features",
			Severity:   SeverityWarning,
			Context: map[string]interface{}{
				"pythonVersion": version,
			},
		})
	}
}

// checkDeprecatedPackages checks for deprecated packages
func (dc *DeprecationChecker) checkDeprecatedPackages(dependencies map[string]string) {
	deprecatedPackages := map[string]string{
		"flask":       "Consider using FastAPI for better performance and modern async support",
		"django<4.0":  "Upgrade to Django 4.0+ for security updates and new features",
		"requests":    "Consider using httpx for async support and better performance",
		"urllib3<2.0": "Upgrade to urllib3 2.0+ for security fixes",
		"setuptools":  "Consider using build and pip-tools for modern packaging",
	}

	for pkg, version := range dependencies {
		if suggestion, isDeprecated := deprecatedPackages[pkg]; isDeprecated {
			dc.issueWarning(DeprecationWarning{
				Pattern:    "deprecated_package",
				Message:    fmt.Sprintf("Package %s is deprecated or has better alternatives", pkg),
				Suggestion: suggestion,
				Severity:   SeverityInfo,
				Context: map[string]interface{}{
					"package": pkg,
					"version": version,
				},
			})
		}
	}
}

// CheckBuildPatterns checks for deprecated build patterns
func (dc *DeprecationChecker) CheckBuildPatterns(buildInfo map[string]interface{}) {
	if !dc.enableWarnings || buildInfo == nil {
		return
	}

	// Check for deprecated build tools
	if buildTool, exists := buildInfo["buildTool"]; exists {
		if buildTool == "setuptools" {
			dc.issueWarning(DeprecationWarning{
				Pattern:    "deprecated_build_tool",
				Message:    "Using setuptools directly is deprecated",
				Suggestion: "Use build backend in pyproject.toml (e.g., hatchling, setuptools-scm)",
				Severity:   SeverityWarning,
				Context: map[string]interface{}{
					"buildTool": buildTool,
				},
			})
		}
	}

	// Check for deprecated packaging patterns
	if packagingPattern, exists := buildInfo["packagingPattern"]; exists {
		if packagingPattern == "setup.py" {
			dc.issueWarning(DeprecationWarning{
				Pattern:    "deprecated_packaging",
				Message:    "Using setup.py for packaging is deprecated",
				Suggestion: "Migrate to pyproject.toml with a modern build backend",
				Severity:   SeverityWarning,
				Context: map[string]interface{}{
					"packagingPattern": packagingPattern,
				},
			})
		}
	}
}

// issueWarning issues a deprecation warning
func (dc *DeprecationChecker) issueWarning(warning DeprecationWarning) {
	// Check if we've already issued this warning recently
	warningKey := fmt.Sprintf("%s:%s", warning.Pattern, warning.Message)
	if lastIssued, exists := dc.warningHistory[warningKey]; exists {
		// Don't issue the same warning more than once per hour
		if time.Since(lastIssued) < time.Hour {
			return
		}
	}

	// Record the warning
	warning.Timestamp = time.Now()
	dc.warningHistory[warningKey] = warning.Timestamp

	// Log the warning
	logLevel := slog.LevelInfo
	switch warning.Severity {
	case SeverityWarning:
		logLevel = slog.LevelWarn
	case SeverityError, SeverityCritical:
		logLevel = slog.LevelError
	}

	slog.Log(nil, logLevel, "deprecation warning",
		"pattern", warning.Pattern,
		"message", warning.Message,
		"suggestion", warning.Suggestion,
		"severity", string(warning.Severity))

	// Notify callbacks
	dc.notifyCallbacks(warning)
}

// notifyCallbacks notifies all registered callbacks
func (dc *DeprecationChecker) notifyCallbacks(warning DeprecationWarning) {
	for _, callback := range dc.callbacks {
		go callback(warning)
	}
}

// GetWarningHistory returns the warning history
func (dc *DeprecationChecker) GetWarningHistory() map[string]time.Time {
	return dc.warningHistory
}

// ClearWarningHistory clears the warning history
func (dc *DeprecationChecker) ClearWarningHistory() {
	dc.warningHistory = make(map[string]time.Time)
}
