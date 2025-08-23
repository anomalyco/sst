package python

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/sst/sst/v3/cmd/sst/mosaic/ui/common"
	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/events"
)

// BuildProgressReporter handles enhanced progress reporting for Python builds
type BuildProgressReporter struct {
	functionID string
	startTime  time.Time
	stages     map[string]time.Time
}

// NewBuildProgressReporter creates a new progress reporter
func NewBuildProgressReporter(functionID string) *BuildProgressReporter {
	return &BuildProgressReporter{
		functionID: functionID,
		startTime:  time.Now(),
		stages:     make(map[string]time.Time),
	}
}

// ReportStage reports progress for a build stage with timing information
func (bpr *BuildProgressReporter) ReportStage(stage, message string) {
	bpr.stages[stage] = time.Now()

	// Calculate elapsed time since build start
	elapsed := time.Since(bpr.startTime)

	// Enhanced message with timing
	enhancedMessage := fmt.Sprintf("%s (%.1fs)", message, elapsed.Seconds())

	if bpr.functionID != "" {
		bus.Publish(&events.FunctionBuildProgressEvent{
			FunctionID: bpr.functionID,
			Stage:      stage,
			Message:    enhancedMessage,
		})
	} else {
		bus.Publish(&common.StdoutEvent{
			Line: fmt.Sprintf("🔨 Build %s: %s", stage, enhancedMessage),
		})
	}

	slog.Info("build progress",
		"functionID", bpr.functionID,
		"stage", stage,
		"message", message,
		"elapsed", elapsed.Seconds())
}

// ReportSuccess reports successful completion of a stage with details
func (bpr *BuildProgressReporter) ReportSuccess(stage, message string, details map[string]interface{}) {
	elapsed := time.Since(bpr.startTime)

	// Create detailed success message
	var detailParts []string
	for key, value := range details {
		detailParts = append(detailParts, fmt.Sprintf("%s=%v", key, value))
	}

	enhancedMessage := message
	if len(detailParts) > 0 {
		enhancedMessage = fmt.Sprintf("%s (%s) (%.1fs)", message, strings.Join(detailParts, ", "), elapsed.Seconds())
	} else {
		enhancedMessage = fmt.Sprintf("%s (%.1fs)", message, elapsed.Seconds())
	}

	if bpr.functionID != "" {
		bus.Publish(&events.FunctionBuildProgressEvent{
			FunctionID: bpr.functionID,
			Stage:      stage,
			Message:    enhancedMessage,
		})
	} else {
		bus.Publish(&common.StdoutEvent{
			Line: fmt.Sprintf("✅ Build %s: %s", stage, enhancedMessage),
		})
	}

	slog.Info("build success",
		"functionID", bpr.functionID,
		"stage", stage,
		"message", message,
		"details", details,
		"elapsed", elapsed.Seconds())
}

// ReportWarning reports a warning that doesn't cause build failure
func (bpr *BuildProgressReporter) ReportWarning(stage, message string, details map[string]interface{}) {
	elapsed := time.Since(bpr.startTime)

	enhancedMessage := fmt.Sprintf("⚠️  %s (%.1fs)", message, elapsed.Seconds())

	if bpr.functionID != "" {
		bus.Publish(&events.FunctionBuildProgressEvent{
			FunctionID: bpr.functionID,
			Stage:      stage,
			Message:    enhancedMessage,
		})
	} else {
		bus.Publish(&common.StdoutEvent{
			Line: fmt.Sprintf("⚠️  Build %s: %s", stage, enhancedMessage),
		})
	}

	slog.Warn("build warning",
		"functionID", bpr.functionID,
		"stage", stage,
		"message", message,
		"details", details,
		"elapsed", elapsed.Seconds())
}

// ReportModuleIncluded reports successful inclusion of a Python module
func (bpr *BuildProgressReporter) ReportModuleIncluded(moduleName string, sourcePath string, targetPath string) {
	details := map[string]interface{}{
		"module": moduleName,
		"source": filepath.Base(sourcePath),
		"target": filepath.Base(targetPath),
	}

	message := fmt.Sprintf("Module '%s' included successfully", moduleName)
	bpr.ReportSuccess("modules", message, details)
}

// ReportArtifactSummary reports a summary of what was included in the deployment package
func (bpr *BuildProgressReporter) ReportArtifactSummary(artifactDir string, summary ArtifactSummary) {
	details := map[string]interface{}{
		"modules":     summary.ModuleCount,
		"files":       summary.FileCount,
		"size":        fmt.Sprintf("%.1fMB", float64(summary.TotalSize)/(1024*1024)),
		"pythonFiles": summary.PythonFileCount,
	}

	message := "Deployment artifact created"
	bpr.ReportSuccess("summary", message, details)
}

// ReportBuildComplete reports final build completion with comprehensive summary
func (bpr *BuildProgressReporter) ReportBuildComplete(summary BuildSummary) {
	totalElapsed := time.Since(bpr.startTime)

	details := map[string]interface{}{
		"duration": fmt.Sprintf("%.1fs", totalElapsed.Seconds()),
		"modules":  summary.ModulesIncluded,
		"size":     fmt.Sprintf("%.1fMB", float64(summary.ArtifactSize)/(1024*1024)),
		"stages":   len(bpr.stages),
	}

	message := "Build completed successfully"
	bpr.ReportSuccess("complete", message, details)

	// Log stage timing breakdown
	slog.Info("build stage timing breakdown",
		"functionID", bpr.functionID,
		"totalDuration", totalElapsed.Seconds(),
		"stages", bpr.getStageTimings())
}

// getStageTimings returns timing information for all stages
func (bpr *BuildProgressReporter) getStageTimings() map[string]float64 {
	timings := make(map[string]float64)

	var previousTime time.Time = bpr.startTime

	// Calculate duration for each stage
	for stage, stageTime := range bpr.stages {
		duration := stageTime.Sub(previousTime)
		timings[stage] = duration.Seconds()
		previousTime = stageTime
	}

	return timings
}

// ArtifactSummary contains information about the deployment artifact
type ArtifactSummary struct {
	ModuleCount     int   `json:"moduleCount"`
	FileCount       int   `json:"fileCount"`
	PythonFileCount int   `json:"pythonFileCount"`
	TotalSize       int64 `json:"totalSize"`
}

// BuildSummary contains comprehensive build information
type BuildSummary struct {
	ModulesIncluded []string `json:"modulesIncluded"`
	ArtifactSize    int64    `json:"artifactSize"`
	BuildDuration   float64  `json:"buildDuration"`
	StagesCompleted int      `json:"stagesCompleted"`
}

// ValidateAndReportModules validates modules and reports their inclusion
func (bpr *BuildProgressReporter) ValidateAndReportModules(artifactDir string, expectedModules []string) error {
	bpr.ReportStage("validate-modules", "Validating Python modules...")

	var includedModules []string
	var missingModules []string

	for _, moduleName := range expectedModules {
		modulePath := filepath.Join(artifactDir, moduleName)

		// Check if module directory exists
		if fileExists(modulePath) {
			// Check if it's a proper Python module
			initPyPath := filepath.Join(modulePath, "__init__.py")
			if fileExists(initPyPath) {
				includedModules = append(includedModules, moduleName)
				bpr.ReportModuleIncluded(moduleName, modulePath, modulePath)
			} else {
				bpr.ReportWarning("validate-modules",
					fmt.Sprintf("Module '%s' missing __init__.py file", moduleName),
					map[string]interface{}{"module": moduleName, "path": modulePath})
				includedModules = append(includedModules, moduleName) // Still count as included
			}
		} else {
			missingModules = append(missingModules, moduleName)
		}
	}

	if len(missingModules) > 0 {
		return NewModuleMissingError(
			strings.Join(missingModules, ", "),
			"", // handler path will be filled by caller
			artifactDir,
			expectedModules,
		)
	}

	bpr.ReportSuccess("validate-modules",
		"All required modules validated successfully",
		map[string]interface{}{
			"included": len(includedModules),
			"total":    len(expectedModules),
		})

	return nil
}
