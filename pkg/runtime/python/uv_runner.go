package python

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sst/sst/v3/pkg/process"
)

// UvCommandRunner executes UV commands efficiently with caching and optimization
type UvCommandRunner struct {
	// buildCache provides access to build cache for command result caching
	buildCache *BuildCache

	// mutex protects concurrent access
	mutex sync.RWMutex

	// config stores runner configuration
	config UvCommandRunnerConfig

	// commandCache caches command results
	commandCache map[string]*CommandResult

	// cacheTimeout determines how long command results are cached
	cacheTimeout time.Duration
}

// UvCommandRunnerConfig configures the UV command runner
type UvCommandRunnerConfig struct {
	// BuildCache for caching command results
	BuildCache *BuildCache

	// EnableCaching enables caching of command results
	EnableCaching bool

	// EnableProgressReport enables progress reporting for commands
	EnableProgressReport bool

	// CommandTimeout is the timeout for individual commands
	CommandTimeout time.Duration

	// MaxRetries is the maximum number of retries for failed commands
	MaxRetries int

	// RetryDelay is the delay between retries
	RetryDelay time.Duration
}

// CommandResult represents the result of a UV command execution
type CommandResult struct {
	// Command is the command that was executed
	Command string

	// Args are the command arguments
	Args []string

	// WorkingDir is the directory where the command was executed
	WorkingDir string

	// ExitCode is the command exit code
	ExitCode int

	// Stdout contains the command stdout
	Stdout string

	// Stderr contains the command stderr
	Stderr string

	// Duration is how long the command took to execute
	Duration time.Duration

	// ExecutedAt is when the command was executed
	ExecutedAt time.Time

	// Success indicates if the command was successful
	Success bool

	// Cached indicates if this result was retrieved from cache
	Cached bool

	// InputFileHashes contains hashes of files that affect this command's output
	InputFileHashes map[string]string
}

// UvBuildCommand represents a UV build command
type UvBuildCommand struct {
	// PackageName is the name of the package to build
	PackageName string

	// PackageDir is the directory containing the package
	PackageDir string

	// WorkspaceDir is the workspace directory for multi-package builds
	WorkspaceDir string

	// OutputDir is the directory where build artifacts should be placed
	OutputDir string

	// SourceFiles contains the source files for the package
	SourceFiles []string

	// Dependencies contains the package dependencies
	Dependencies []string

	// BuildType specifies the type of build (sdist, wheel, etc.)
	BuildType string

	// Architecture specifies the target architecture
	Architecture string

	// AllPackages builds all packages in the workspace
	AllPackages bool

	// ExtraArgs contains additional arguments for the build command
	ExtraArgs []string
}

// UvSyncCommand represents a UV sync command
type UvSyncCommand struct {
	// WorkspaceDir is the workspace directory
	WorkspaceDir string

	// Packages specifies which packages to sync (empty for all)
	Packages []string

	// AllPackages indicates if all packages should be synced
	AllPackages bool

	// NoDev excludes development dependencies
	NoDev bool

	// ExtraArgs contains additional arguments for the sync command
	ExtraArgs []string

	// TargetVenv specifies a custom venv path (uses UV_PROJECT_ENVIRONMENT)
	// If empty, uses the default .venv in WorkspaceDir
	TargetVenv string
}

// UvExportCommand represents a UV export command
type UvExportCommand struct {
	// WorkspaceDir is the workspace directory
	WorkspaceDir string

	// PackageName is the package to export dependencies for
	PackageName string

	// OutputFile is the file to write requirements to
	OutputFile string

	// NoEmitWorkspace excludes workspace dependencies
	NoEmitWorkspace bool

	// NoDev excludes development dependencies
	NoDev bool

	// AllPackages exports dependencies for all packages in the workspace
	AllPackages bool

	// ExtraArgs contains additional arguments for the export command
	ExtraArgs []string
}

// UvInstallCommand represents a UV pip install command
type UvInstallCommand struct {
	// WorkingDir is the directory to run the command in
	WorkingDir string

	// RequirementsFile is the requirements file to install from
	RequirementsFile string

	// TargetDir is the target directory for installation
	TargetDir string

	// PythonPlatform specifies the target platform
	PythonPlatform string

	// Architecture specifies the target architecture
	Architecture string

	// ExtraArgs contains additional arguments for the install command
	ExtraArgs []string
}

// NewUvCommandRunner creates a new UV command runner
func NewUvCommandRunner(config UvCommandRunnerConfig) *UvCommandRunner {
	if config.CommandTimeout == 0 {
		config.CommandTimeout = 1 * time.Minute // Reduced for faster development builds
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	return &UvCommandRunner{
		buildCache:   config.BuildCache,
		config:       config,
		commandCache: make(map[string]*CommandResult),
		cacheTimeout: 10 * time.Minute,
	}
}

// ExecuteBuildCommand executes a UV build command for a single package
func (ur *UvCommandRunner) ExecuteBuildCommand(ctx context.Context, cmd *UvBuildCommand) error {
	// Add nil checks to prevent segfaults
	if ur == nil {
		return fmt.Errorf("UV command runner is nil")
	}
	if cmd == nil {
		return fmt.Errorf("UV build command is nil")
	}

	// Check if we can use cached results
	if ur.config.EnableCaching {
		cacheKey := ur.generateBuildCacheKey(cmd)
		if cached := ur.getCachedResult(cacheKey, cmd.WorkspaceDir); cached != nil && cached.Success {
			slog.Info("using cached UV build result",
				"package", cmd.PackageName,
				"cached", true)
			return nil
		}
	}

	// Construct UV build command
	args := []string{"build"}

	// Add package specification
	if cmd.AllPackages {
		args = append(args, "--all")
	} else if cmd.PackageName != "" {
		args = append(args, "--package="+cmd.PackageName)
	}

	// Add build type
	if cmd.BuildType != "" {
		switch cmd.BuildType {
		case "sdist":
			args = append(args, "--sdist")
		case "wheel":
			args = append(args, "--wheel")
		default:
			args = append(args, "--sdist") // Default to sdist
		}
	} else {
		args = append(args, "--sdist")
	}

	// Add output directory
	if cmd.OutputDir != "" {
		args = append(args, "--out-dir="+cmd.OutputDir)
	}

	// Add performance optimization flags for faster builds
	// Note: --no-build-isolation removed as it prevents build dependencies from being installed
	args = append(args, "--no-sources") // Don't include source files in wheel (faster)

	// Add extra arguments
	args = append(args, cmd.ExtraArgs...)

	// Determine working directory - use workspace directory for all packages, package directory for single package
	workingDir := cmd.PackageDir
	if cmd.AllPackages && cmd.WorkspaceDir != "" {
		workingDir = cmd.WorkspaceDir
	} else if workingDir == "" {
		workingDir = "."
	}

	// Log the exact command being executed for debugging
	slog.Info("about to execute UV build command",
		"package", cmd.PackageName,
		"command", "uv "+strings.Join(args, " "),
		"workingDir", workingDir,
		"buildType", cmd.BuildType)

	// Execute command with enhanced error handling
	result, err := ur.executeCommand(ctx, "uv", args, workingDir)
	if err != nil {
		slog.Error("UV build command failed",
			"package", cmd.PackageName,
			"command", "uv "+strings.Join(args, " "),
			"workingDir", workingDir,
			"error", err)
		return NewUVCommandFailedError("uv", args, -1, err.Error()).
			WithContext("package", cmd.PackageName).
			WithContext("workingDir", workingDir)
	}

	slog.Info("UV build command completed successfully",
		"package", cmd.PackageName,
		"duration", result.Duration)

	if !result.Success {
		return NewUVCommandFailedError("uv", args, result.ExitCode, result.Stderr).
			WithContext("package", cmd.PackageName).
			WithContext("workingDir", workingDir).
			WithContext("duration", result.Duration.String())
	}

	slog.Info("UV build completed",
		"package", cmd.PackageName,
		"duration", result.Duration,
		"cached", result.Cached)

	return nil
}

// ExecuteSyncCommand executes a UV sync command with optimization
func (ur *UvCommandRunner) ExecuteSyncCommand(ctx context.Context, cmd *UvSyncCommand) error {
	// Check if sync is needed based on file changes
	if ur.config.EnableCaching && !ur.shouldExecuteSync(cmd) {
		slog.Info("skipping UV sync - no changes detected",
			"workspaceDir", cmd.WorkspaceDir)
		return nil
	}

	// Construct UV sync command
	args := []string{"sync"}

	// Add package specifications
	if cmd.AllPackages {
		args = append(args, "--all-packages")
	} else if len(cmd.Packages) > 0 {
		for _, pkg := range cmd.Packages {
			args = append(args, "--package="+pkg)
		}
	}

	// Add dev dependency exclusion
	if cmd.NoDev {
		args = append(args, "--no-dev")
	}

	// Add extra arguments
	args = append(args, cmd.ExtraArgs...)

	// Execute command with progress reporting
	if ur.config.EnableProgressReport {
		slog.Info("executing UV sync",
			"workspaceDir", cmd.WorkspaceDir,
			"packages", len(cmd.Packages),
			"allPackages", cmd.AllPackages,
			"targetVenv", cmd.TargetVenv)
	}

	// Execute command with custom venv if specified
	var result *CommandResult
	var err error
	if cmd.TargetVenv != "" {
		// Use UV_PROJECT_ENVIRONMENT to sync to a build-specific venv
		// This prevents modifying the user's development .venv
		result, err = ur.executeCommandWithEnv(ctx, "uv", args, cmd.WorkspaceDir, map[string]string{
			"UV_PROJECT_ENVIRONMENT": cmd.TargetVenv,
		})
	} else {
		result, err = ur.executeCommand(ctx, "uv", args, cmd.WorkspaceDir)
	}
	if err != nil {
		return fmt.Errorf("UV sync failed: %w", err)
	}

	if !result.Success {
		return ur.createDetailedSyncError(result, cmd)
	}

	// Update sync state for future optimization
	if ur.config.EnableCaching {
		ur.updateSyncState(cmd)
	}

	slog.Info("UV sync completed",
		"workspaceDir", cmd.WorkspaceDir,
		"duration", result.Duration,
		"cached", result.Cached)

	return nil
}

// ExecuteExportCommand executes a UV export command with optimization
func (ur *UvCommandRunner) ExecuteExportCommand(ctx context.Context, cmd *UvExportCommand) error {
	// Check if export is needed based on file changes
	if ur.config.EnableCaching && !ur.shouldExecuteExport(cmd) {
		slog.Info("skipping UV export - no changes detected",
			"outputFile", cmd.OutputFile)
		return nil
	}

	// Construct UV export command
	args := []string{"export"}

	// Add package specification
	if cmd.AllPackages {
		args = append(args, "--all-packages")
	} else if cmd.PackageName != "" {
		args = append(args, "--package="+cmd.PackageName)
	}

	// Add output file
	if cmd.OutputFile != "" {
		args = append(args, "--output-file="+cmd.OutputFile)
	}

	// Add workspace exclusion
	if cmd.NoEmitWorkspace {
		args = append(args, "--no-emit-workspace")
	}

	// Add dev dependency exclusion
	if cmd.NoDev {
		args = append(args, "--no-dev")
	}

	// Add extra arguments
	args = append(args, cmd.ExtraArgs...)

	// Execute command with progress reporting
	if ur.config.EnableProgressReport {
		slog.Info("executing UV export",
			"packageName", cmd.PackageName,
			"outputFile", cmd.OutputFile,
			"workspaceDir", cmd.WorkspaceDir)
	}

	// Execute command
	result, err := ur.executeCommand(ctx, "uv", args, cmd.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("UV export failed: %w", err)
	}

	if !result.Success {
		return ur.createDetailedExportError(result, cmd)
	}

	// Update export state for future optimization
	if ur.config.EnableCaching {
		ur.updateExportState(cmd)
	}

	slog.Info("UV export completed",
		"outputFile", cmd.OutputFile,
		"duration", result.Duration,
		"cached", result.Cached)

	return nil
}

// shouldExecuteExport determines if an export command should be executed
func (ur *UvCommandRunner) shouldExecuteExport(cmd *UvExportCommand) bool {
	if ur.buildCache == nil {
		return true // Always export if no cache available
	}

	// Check if output file exists and is newer than dependencies
	if cmd.OutputFile != "" {
		outputInfo, err := os.Stat(cmd.OutputFile)
		if err != nil {
			// Output file doesn't exist, need to export
			return true
		}

		// Check if dependency files are newer than output
		dependencyFiles := []string{
			filepath.Join(cmd.WorkspaceDir, "pyproject.toml"),
			filepath.Join(cmd.WorkspaceDir, "uv.lock"),
		}

		for _, depFile := range dependencyFiles {
			if depInfo, err := os.Stat(depFile); err == nil {
				if depInfo.ModTime().After(outputInfo.ModTime()) {
					slog.Debug("export needed due to dependency change",
						"dependencyFile", depFile,
						"outputFile", cmd.OutputFile)
					return true
				}
			}
		}

		// Output is up to date
		return false
	}

	// No output file specified, always export
	return true
}

// updateExportState updates the export state after a successful export
func (ur *UvCommandRunner) updateExportState(cmd *UvExportCommand) {
	exportStateKey := ur.generateExportStateKey(cmd)

	// Create a dummy result to track export time
	result := &CommandResult{
		Command:    "uv",
		Args:       []string{"export"},
		WorkingDir: cmd.WorkspaceDir,
		ExecutedAt: time.Now(),
		Success:    true,
	}

	ur.cacheResult(exportStateKey, result, cmd.WorkspaceDir)
}

// generateExportStateKey generates a key for tracking export state
func (ur *UvCommandRunner) generateExportStateKey(cmd *UvExportCommand) string {
	return fmt.Sprintf("export:%s:%s:%v:%v",
		cmd.WorkspaceDir,
		cmd.PackageName,
		cmd.NoEmitWorkspace,
		cmd.NoDev)
}

// createDetailedExportError creates a detailed error message for export failures
func (ur *UvCommandRunner) createDetailedExportError(result *CommandResult, cmd *UvExportCommand) error {
	errorMsg := fmt.Sprintf("UV export failed with exit code %d", result.ExitCode)

	// Add context about the command
	if cmd.PackageName != "" {
		errorMsg += fmt.Sprintf(" (exporting package: %s)", cmd.PackageName)
	}

	if cmd.OutputFile != "" {
		errorMsg += fmt.Sprintf(" to file: %s", cmd.OutputFile)
	}

	// Add workspace context
	errorMsg += fmt.Sprintf(" in workspace: %s", cmd.WorkspaceDir)

	// Add stderr output if available
	if result.Stderr != "" {
		errorMsg += fmt.Sprintf("\nError output: %s", result.Stderr)
	}

	// Add suggestions based on common error patterns
	if strings.Contains(result.Stderr, "package") && strings.Contains(result.Stderr, "not found") {
		errorMsg += "\nSuggestion: Check if the package name is correct and exists in the workspace"
	} else if strings.Contains(result.Stderr, "permission") {
		errorMsg += "\nSuggestion: Check file permissions for the output directory"
	} else if strings.Contains(result.Stderr, "lock") {
		errorMsg += "\nSuggestion: Run 'uv sync' first to ensure dependencies are resolved"
	}

	return fmt.Errorf(errorMsg)
}

// ExecuteInstallCommand executes a UV pip install command with optimization
func (ur *UvCommandRunner) ExecuteInstallCommand(ctx context.Context, cmd *UvInstallCommand) error {
	// Check if install is needed based on file changes
	if ur.config.EnableCaching && !ur.shouldExecuteInstall(cmd) {
		slog.Info("skipping UV pip install - no changes detected",
			"targetDir", cmd.TargetDir)
		return nil
	}

	// Construct UV pip install command
	args := []string{"pip", "install"}

	// Add requirements file
	if cmd.RequirementsFile != "" {
		args = append(args, "-r", cmd.RequirementsFile)
	}

	// Add target directory
	if cmd.TargetDir != "" {
		args = append(args, "--target", cmd.TargetDir)
	}

	// Add platform specification
	if cmd.PythonPlatform != "" {
		args = append(args, "--python-platform", cmd.PythonPlatform)
	}

	// Add extra arguments
	args = append(args, cmd.ExtraArgs...)

	// Execute command with progress reporting
	if ur.config.EnableProgressReport {
		slog.Info("executing UV pip install",
			"requirementsFile", cmd.RequirementsFile,
			"targetDir", cmd.TargetDir,
			"platform", cmd.PythonPlatform)
	}

	// Execute command
	result, err := ur.executeCommand(ctx, "uv", args, cmd.WorkingDir)
	if err != nil {
		return fmt.Errorf("UV pip install failed: %w", err)
	}

	if !result.Success {
		return ur.createDetailedInstallError(result, cmd)
	}

	// Update install state for future optimization
	if ur.config.EnableCaching {
		ur.updateInstallState(cmd)
	}

	slog.Info("UV pip install completed",
		"targetDir", cmd.TargetDir,
		"duration", result.Duration,
		"cached", result.Cached)

	return nil
}

// shouldExecuteInstall determines if an install command should be executed
func (ur *UvCommandRunner) shouldExecuteInstall(cmd *UvInstallCommand) bool {
	if ur.buildCache == nil {
		return true // Always install if no cache available
	}

	// Check if requirements file has changed since last install
	if cmd.RequirementsFile != "" && cmd.TargetDir != "" {
		// Check if target directory exists and has content
		targetInfo, err := os.Stat(cmd.TargetDir)
		if err != nil || !targetInfo.IsDir() {
			// Target doesn't exist, need to install
			return true
		}

		// Check if requirements file is newer than target directory
		if reqInfo, err := os.Stat(cmd.RequirementsFile); err == nil {
			if reqInfo.ModTime().After(targetInfo.ModTime()) {
				slog.Debug("install needed due to requirements change",
					"requirementsFile", cmd.RequirementsFile,
					"targetDir", cmd.TargetDir)
				return true
			}
		}

		// Check if target directory is empty
		if isEmpty, err := ur.isDirectoryEmpty(cmd.TargetDir); err == nil && isEmpty {
			return true
		}

		// Target is up to date
		return false
	}

	// No requirements file or target dir specified, always install
	return true
}

// isDirectoryEmpty checks if a directory is empty
func (ur *UvCommandRunner) isDirectoryEmpty(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// updateInstallState updates the install state after a successful install
func (ur *UvCommandRunner) updateInstallState(cmd *UvInstallCommand) {
	installStateKey := ur.generateInstallStateKey(cmd)

	// Create a dummy result to track install time
	result := &CommandResult{
		Command:    "uv",
		Args:       []string{"pip", "install"},
		WorkingDir: cmd.WorkingDir,
		ExecutedAt: time.Now(),
		Success:    true,
	}

	ur.cacheResult(installStateKey, result, cmd.WorkingDir)
}

// generateInstallStateKey generates a key for tracking install state
func (ur *UvCommandRunner) generateInstallStateKey(cmd *UvInstallCommand) string {
	return fmt.Sprintf("install:%s:%s:%s:%s",
		cmd.WorkingDir,
		cmd.RequirementsFile,
		cmd.TargetDir,
		cmd.PythonPlatform)
}

// createDetailedInstallError creates a detailed error message for install failures
func (ur *UvCommandRunner) createDetailedInstallError(result *CommandResult, cmd *UvInstallCommand) error {
	errorMsg := fmt.Sprintf("UV pip install failed with exit code %d", result.ExitCode)

	// Add context about the command
	if cmd.RequirementsFile != "" {
		errorMsg += fmt.Sprintf(" (installing from: %s)", cmd.RequirementsFile)
	}

	if cmd.TargetDir != "" {
		errorMsg += fmt.Sprintf(" to target: %s", cmd.TargetDir)
	}

	if cmd.PythonPlatform != "" {
		errorMsg += fmt.Sprintf(" for platform: %s", cmd.PythonPlatform)
	}

	// Add stderr output if available
	if result.Stderr != "" {
		errorMsg += fmt.Sprintf("\nError output: %s", result.Stderr)
	}

	// Add suggestions based on common error patterns
	if strings.Contains(result.Stderr, "No such file or directory") {
		errorMsg += "\nSuggestion: Check if the requirements file exists and is readable"
	} else if strings.Contains(result.Stderr, "permission") {
		errorMsg += "\nSuggestion: Check write permissions for the target directory"
	} else if strings.Contains(result.Stderr, "network") || strings.Contains(result.Stderr, "timeout") {
		errorMsg += "\nSuggestion: Check network connectivity and try again"
	} else if strings.Contains(result.Stderr, "platform") {
		errorMsg += "\nSuggestion: Check if the specified platform is supported"
	} else if strings.Contains(result.Stderr, "version") && strings.Contains(result.Stderr, "not found") {
		errorMsg += "\nSuggestion: Check if all package versions in requirements.txt are available"
	}

	return fmt.Errorf(errorMsg)
}

// executeCommand executes a command with caching and retry logic
func (ur *UvCommandRunner) executeCommand(ctx context.Context, command string, args []string, workingDir string) (*CommandResult, error) {
	// Generate cache key
	cacheKey := ur.generateCacheKey(command, args, workingDir)

	// Check cache if enabled
	if ur.config.EnableCaching {
		if cached := ur.getCachedResult(cacheKey, workingDir); cached != nil {
			slog.Debug("using cached command result",
				"command", command,
				"args", strings.Join(args, " "))
			return cached, nil
		}
	}

	// Execute command with retries
	var result *CommandResult
	var err error

	for attempt := 0; attempt <= ur.config.MaxRetries; attempt++ {
		if attempt > 0 {
			slog.Info("retrying command",
				"command", command,
				"attempt", attempt+1,
				"maxRetries", ur.config.MaxRetries+1)

			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(ur.config.RetryDelay * time.Duration(attempt)):
			}
		}

		result, err = ur.executeCommandOnce(ctx, command, args, workingDir)
		if err == nil && result.Success {
			break
		}

		if attempt < ur.config.MaxRetries {
			exitCode := -1
			if result != nil {
				exitCode = result.ExitCode
			}
			slog.Warn("command failed, will retry",
				"command", command,
				"error", err,
				"exitCode", exitCode)
		}
	}

	if err != nil {
		return nil, err
	}

	// Cache successful results
	if ur.config.EnableCaching && result.Success {
		ur.cacheResult(cacheKey, result, workingDir)
	}

	return result, nil
}

// executeCommandWithEnv executes a command with custom environment variables
func (ur *UvCommandRunner) executeCommandWithEnv(ctx context.Context, command string, args []string, workingDir string, env map[string]string) (*CommandResult, error) {
	startTime := time.Now()

	// Create command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, ur.config.CommandTimeout)
	defer cancel()

	cmd := process.CommandContext(cmdCtx, command, args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set up environment with custom vars
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	slog.Info("executing UV command with custom env",
		"command", command,
		"args", strings.Join(args, " "),
		"workingDir", workingDir,
		"customEnv", env)

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	result := &CommandResult{
		Command:    command,
		Args:       args,
		WorkingDir: workingDir,
		Stdout:     string(output),
		Stderr:     "",
		ExitCode:   0,
		Duration:   duration,
		ExecutedAt: time.Now(),
		Success:    err == nil,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Stderr = string(output)
		return result, fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, string(output))
	}

	return result, nil
}

// executeCommandOnce executes a command once without retries
func (ur *UvCommandRunner) executeCommandOnce(ctx context.Context, command string, args []string, workingDir string) (*CommandResult, error) {
	// Add nil checks to prevent segfaults
	if ur == nil {
		return nil, fmt.Errorf("UV command runner is nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}

	startTime := time.Now()

	// Create command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, ur.config.CommandTimeout)
	defer cancel()

	cmd := process.CommandContext(cmdCtx, command, args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set up environment
	cmd.Env = os.Environ()

	slog.Info("executing UV command",
		"command", command,
		"args", strings.Join(args, " "),
		"workingDir", workingDir,
		"timeout", ur.config.CommandTimeout)

	// Add detailed debugging for the exact command being run
	slog.Info("UV command details",
		"fullCommand", fmt.Sprintf("%s %s", command, strings.Join(args, " ")),
		"workingDir", workingDir,
		"env", fmt.Sprintf("PATH=%s", os.Getenv("PATH")))

	// Check if working directory exists and is accessible
	if workingDir != "" {
		if stat, err := os.Stat(workingDir); err != nil {
			slog.Error("working directory issue", "workingDir", workingDir, "error", err)
		} else {
			slog.Info("working directory info", "workingDir", workingDir, "isDir", stat.IsDir())
		}
	}

	// Execute command with progress reporting
	if ur.config.EnableProgressReport && strings.Contains(strings.Join(args, " "), "build") {
		// Report that UV build is starting
		slog.Info("UV build command starting", "command", command, "args", strings.Join(args, " "))
	}

	// Add a goroutine to log progress every 30 seconds to detect hangs
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				slog.Info("UV command still running",
					"command", command,
					"elapsed", time.Since(startTime),
					"workingDir", workingDir)
			}
		}
	}()

	slog.Info("about to call cmd.CombinedOutput()")
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// Signal the progress goroutine to stop
	close(done)

	slog.Info("cmd.CombinedOutput() returned", "duration", duration, "success", err == nil)

	if ur.config.EnableProgressReport && strings.Contains(strings.Join(args, " "), "build") {
		// Report build completion with timing
		slog.Info("UV build command finished", "duration", duration, "success", err == nil)
	}

	result := &CommandResult{
		Command:    command,
		Args:       args,
		WorkingDir: workingDir,
		Duration:   duration,
		ExecutedAt: startTime,
		Cached:     false,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Success = false
		result.Stderr = string(output)
	} else {
		result.ExitCode = 0
		result.Success = true
		result.Stdout = string(output)
	}

	slog.Info("UV command completed",
		"command", command,
		"args", strings.Join(args, " "),
		"duration", duration,
		"success", result.Success,
		"command", command,
		"duration", duration,
		"success", result.Success,
		"exitCode", result.ExitCode)

	if ur.config.EnableProgressReport && len(result.Stdout) > 0 {
		slog.Debug("command output", "stdout", result.Stdout)
	}

	if !result.Success && len(result.Stderr) > 0 {
		slog.Error("command error output", "stderr", result.Stderr)
	}

	return result, nil
}

// generateCacheKey generates a cache key for a command
func (ur *UvCommandRunner) generateCacheKey(command string, args []string, workingDir string) string {
	key := fmt.Sprintf("%s:%s:%s", command, strings.Join(args, ":"), workingDir)
	return key
}

// generateBuildCacheKey generates a cache key for a build command
func (ur *UvCommandRunner) generateBuildCacheKey(cmd *UvBuildCommand) string {
	key := fmt.Sprintf("build:%s:%s:%s:%s",
		cmd.PackageName,
		cmd.PackageDir,
		cmd.BuildType,
		cmd.OutputDir)
	return key
}

// getCachedResult retrieves a cached command result
func (ur *UvCommandRunner) getCachedResult(cacheKey string, workingDir string) *CommandResult {
	ur.mutex.RLock()
	defer ur.mutex.RUnlock()

	if cached, exists := ur.commandCache[cacheKey]; exists {
		// Validate cache using content hashes instead of time
		if ur.isCacheValid(cached, workingDir) {
			// Create a copy with cached flag set
			result := *cached
			result.Cached = true
			return &result
		}
		// Cache invalid, remove it
		delete(ur.commandCache, cacheKey)
	}

	return nil
}

// cacheResult caches a command result with content hashes
func (ur *UvCommandRunner) cacheResult(cacheKey string, result *CommandResult, workingDir string) {
	ur.mutex.Lock()
	defer ur.mutex.Unlock()

	// Calculate hashes of relevant input files
	inputHashes, err := ur.calculateInputFileHashes(workingDir)
	if err != nil {
		slog.Warn("failed to calculate input file hashes for cache", "error", err)
		inputHashes = make(map[string]string)
	}

	// Create a copy to cache with input hashes
	cached := *result
	cached.InputFileHashes = inputHashes
	ur.commandCache[cacheKey] = &cached

	// Cleanup old entries if cache is getting too large
	if len(ur.commandCache) > 100 {
		ur.cleanupCache()
	}
}

// isCacheValid validates cache entry using content hashes instead of time
func (ur *UvCommandRunner) isCacheValid(cached *CommandResult, workingDir string) bool {
	if cached.InputFileHashes == nil {
		return false
	}

	// Calculate current hashes of input files
	currentHashes, err := ur.calculateInputFileHashes(workingDir)
	if err != nil {
		return false
	}

	// Compare cached hashes with current hashes
	for filePath, cachedHash := range cached.InputFileHashes {
		if currentHash, exists := currentHashes[filePath]; !exists || currentHash != cachedHash {
			return false
		}
	}

	// Check if any new relevant files have been added
	for filePath := range currentHashes {
		if _, exists := cached.InputFileHashes[filePath]; !exists {
			return false
		}
	}

	return true
}

// calculateInputFileHashes calculates hashes of files that affect UV command output
func (ur *UvCommandRunner) calculateInputFileHashes(workingDir string) (map[string]string, error) {
	hashes := make(map[string]string)

	// Files that typically affect UV command output
	relevantFiles := []string{
		"pyproject.toml",
		"uv.lock",
		"requirements.txt",
		"requirements-dev.txt",
		"dev-requirements.txt",
		"poetry.lock",
		"Pipfile.lock",
	}

	for _, fileName := range relevantFiles {
		filePath := filepath.Join(workingDir, fileName)
		if _, err := os.Stat(filePath); err == nil {
			hash, err := ur.calculateFileHash(filePath)
			if err != nil {
				slog.Warn("failed to hash file", "file", filePath, "error", err)
				continue
			}
			hashes[fileName] = hash
		}
	}

	return hashes, nil
}

// calculateFileHash calculates SHA256 hash of a file
func (ur *UvCommandRunner) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// cleanupCache removes old entries from the command cache
func (ur *UvCommandRunner) cleanupCache() {
	// Remove entries older than cache timeout (keep some time-based cleanup for memory management)
	cutoff := time.Now().Add(-ur.cacheTimeout)

	for key, result := range ur.commandCache {
		if result.ExecutedAt.Before(cutoff) {
			delete(ur.commandCache, key)
		}
	}
}

// ClearCache clears the command cache
func (ur *UvCommandRunner) ClearCache() {
	ur.mutex.Lock()
	defer ur.mutex.Unlock()

	ur.commandCache = make(map[string]*CommandResult)
}

// GetCacheStats returns statistics about the command cache
func (ur *UvCommandRunner) GetCacheStats() *UvCommandCacheStats {
	ur.mutex.RLock()
	defer ur.mutex.RUnlock()

	stats := &UvCommandCacheStats{
		TotalEntries: len(ur.commandCache),
		CacheTimeout: ur.cacheTimeout,
	}

	// Calculate detailed statistics
	var totalCommands, cachedCommands int
	var totalDuration time.Duration

	for _, result := range ur.commandCache {
		totalCommands++
		totalDuration += result.Duration

		if result.Cached {
			cachedCommands++
		}

		// Count command types
		if len(result.Args) > 0 {
			switch result.Args[0] {
			case "sync":
				stats.SyncCommands++
			case "build":
				stats.BuildCommands++
			case "export":
				stats.ExportCommands++
			case "pip":
				if len(result.Args) > 1 && result.Args[1] == "install" {
					stats.InstallCommands++
				}
			}
		}
	}

	if totalCommands > 0 {
		stats.HitRate = float64(cachedCommands) / float64(totalCommands)
		stats.AverageDuration = totalDuration / time.Duration(totalCommands)
	}

	stats.TotalDuration = totalDuration
	stats.SkippedCommands = cachedCommands // Commands that were skipped due to caching

	return stats
}

// UvCommandCacheStats contains statistics about the UV command cache
type UvCommandCacheStats struct {
	TotalEntries    int           `json:"totalEntries"`
	HitRate         float64       `json:"hitRate"`
	CacheTimeout    time.Duration `json:"cacheTimeout"`
	SyncCommands    int           `json:"syncCommands"`
	BuildCommands   int           `json:"buildCommands"`
	ExportCommands  int           `json:"exportCommands"`
	InstallCommands int           `json:"installCommands"`
	SkippedCommands int           `json:"skippedCommands"`
	TotalDuration   time.Duration `json:"totalDuration"`
	AverageDuration time.Duration `json:"averageDuration"`
}

// shouldExecuteSync determines if a sync command should be executed
func (ur *UvCommandRunner) shouldExecuteSync(cmd *UvSyncCommand) bool {
	if ur.buildCache == nil {
		return true // Always sync if no cache available
	}

	// Check if relevant files have changed since last sync
	syncStateKey := ur.generateSyncStateKey(cmd)
	if lastSync, exists := ur.getSyncState(syncStateKey); exists {
		// Check if pyproject.toml or uv.lock have changed
		relevantFiles := []string{
			filepath.Join(cmd.WorkspaceDir, "pyproject.toml"),
			filepath.Join(cmd.WorkspaceDir, "uv.lock"),
		}

		for _, file := range relevantFiles {
			if ur.hasFileChangedSince(file, lastSync) {
				slog.Debug("sync needed due to file change",
					"file", file,
					"lastSync", lastSync)
				return true
			}
		}

		// No relevant changes detected
		return false
	}

	// No previous sync state, need to sync
	return true
}

// generateSyncStateKey generates a key for tracking sync state
func (ur *UvCommandRunner) generateSyncStateKey(cmd *UvSyncCommand) string {
	key := fmt.Sprintf("sync:%s:%v:%v", cmd.WorkspaceDir, cmd.AllPackages, cmd.NoDev)
	if len(cmd.Packages) > 0 {
		key += ":" + strings.Join(cmd.Packages, ",")
	}
	return key
}

// getSyncState retrieves the last sync time for a command
func (ur *UvCommandRunner) getSyncState(key string) (time.Time, bool) {
	ur.mutex.RLock()
	defer ur.mutex.RUnlock()

	if result, exists := ur.commandCache[key]; exists {
		return result.ExecutedAt, true
	}
	return time.Time{}, false
}

// updateSyncState updates the sync state after a successful sync
func (ur *UvCommandRunner) updateSyncState(cmd *UvSyncCommand) {
	syncStateKey := ur.generateSyncStateKey(cmd)

	// Create a dummy result to track sync time
	result := &CommandResult{
		Command:    "uv",
		Args:       []string{"sync"},
		WorkingDir: cmd.WorkspaceDir,
		ExecutedAt: time.Now(),
		Success:    true,
	}

	ur.cacheResult(syncStateKey, result, cmd.WorkspaceDir)
}

// hasFileChangedSince checks if a file has changed since the given time
func (ur *UvCommandRunner) hasFileChangedSince(filePath string, since time.Time) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		// File doesn't exist or can't be accessed, consider it changed
		return true
	}

	return info.ModTime().After(since)
}

// createDetailedSyncError creates a detailed error message for sync failures
func (ur *UvCommandRunner) createDetailedSyncError(result *CommandResult, cmd *UvSyncCommand) error {
	errorMsg := fmt.Sprintf("UV sync failed with exit code %d", result.ExitCode)

	// Add context about the command
	if cmd.AllPackages {
		errorMsg += " (syncing all packages)"
	} else if len(cmd.Packages) > 0 {
		errorMsg += fmt.Sprintf(" (syncing packages: %s)", strings.Join(cmd.Packages, ", "))
	}

	// Add workspace context
	errorMsg += fmt.Sprintf(" in workspace: %s", cmd.WorkspaceDir)

	// Add stderr output if available
	if result.Stderr != "" {
		errorMsg += fmt.Sprintf("\nError output: %s", result.Stderr)
	}

	// Add suggestions based on common error patterns
	if strings.Contains(result.Stderr, "No such file or directory") {
		errorMsg += "\nSuggestion: Check if pyproject.toml exists and is properly configured"
	} else if strings.Contains(result.Stderr, "dependency") && strings.Contains(result.Stderr, "not found") {
		errorMsg += "\nSuggestion: Check if all dependencies are available and properly specified"
	} else if strings.Contains(result.Stderr, "lock") {
		errorMsg += "\nSuggestion: Try removing uv.lock and running sync again"
	}

	return fmt.Errorf(errorMsg)
}
