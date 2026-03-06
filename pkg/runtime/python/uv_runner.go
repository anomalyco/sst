package python

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sst/sst/v3/pkg/process"
)

// UvCommandRunner executes uv commands
type UvCommandRunner struct {
	commandTimeout time.Duration
}

// UvCommandRunnerConfig configures the UV command runner
type UvCommandRunnerConfig struct {
	// CommandTimeout is the timeout for individual commands
	CommandTimeout time.Duration
}

// CommandResult represents the result of a UV command execution
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
	Success  bool
}

// UvBuildCommand represents a UV build command
type UvBuildCommand struct {
	PackageName  string
	PackageDir   string
	WorkspaceDir string
	OutputDir    string
	BuildType    string
	AllPackages  bool
	ExtraArgs    []string
}

// UvExportCommand represents a UV export command
type UvExportCommand struct {
	WorkspaceDir    string
	PackageName     string
	OutputFile      string
	NoEmitWorkspace bool
	NoDev           bool
	NoEditable      bool
	NoEmitProject   bool
	AllPackages     bool
	ExtraArgs       []string
}

// NewUvCommandRunner creates a new UV command runner
func NewUvCommandRunner(config UvCommandRunnerConfig) *UvCommandRunner {
	timeout := config.CommandTimeout
	if timeout == 0 {
		timeout = 1 * time.Minute
	}
	return &UvCommandRunner{commandTimeout: timeout}
}

// ExecuteBuildCommand executes a UV build command for a single package
func (ur *UvCommandRunner) ExecuteBuildCommand(ctx context.Context, cmd *UvBuildCommand) error {
	if ur == nil {
		return fmt.Errorf("UV command runner is nil")
	}
	if cmd == nil {
		return fmt.Errorf("UV build command is nil")
	}

	args := []string{"build"}

	if cmd.AllPackages {
		args = append(args, "--all")
	} else if cmd.PackageName != "" {
		args = append(args, "--package="+cmd.PackageName)
	}

	if cmd.BuildType == "wheel" {
		args = append(args, "--wheel")
	} else {
		args = append(args, "--sdist")
	}

	if cmd.OutputDir != "" {
		args = append(args, "--out-dir="+cmd.OutputDir)
	}

	args = append(args, "--no-sources")
	args = append(args, cmd.ExtraArgs...)

	workingDir := cmd.PackageDir
	if cmd.AllPackages && cmd.WorkspaceDir != "" {
		workingDir = cmd.WorkspaceDir
	} else if workingDir == "" {
		workingDir = "."
	}

	result, err := ur.executeCommand(ctx, "uv", args, workingDir)
	if err != nil {
		slog.Error("UV build command failed",
			"package", cmd.PackageName,
			"command", "uv "+strings.Join(args, " "),
			"error", err)
		return NewUVCommandFailedError("uv", args, -1, err.Error())
	}

	if !result.Success {
		return NewUVCommandFailedError("uv", args, result.ExitCode, result.Stderr)
	}

	return nil
}

// ExecuteExportCommand executes a UV export command
func (ur *UvCommandRunner) ExecuteExportCommand(ctx context.Context, cmd *UvExportCommand) error {
	args := []string{"export"}

	if cmd.AllPackages {
		args = append(args, "--all-packages")
	} else if cmd.PackageName != "" {
		args = append(args, "--package="+cmd.PackageName)
	}

	if cmd.OutputFile != "" {
		args = append(args, "--output-file="+cmd.OutputFile)
	}
	if cmd.NoEmitWorkspace {
		args = append(args, "--no-emit-workspace")
	}
	if cmd.NoEditable {
		args = append(args, "--no-editable")
	}
	if cmd.NoEmitProject {
		args = append(args, "--no-emit-project")
	}
	if cmd.NoDev {
		args = append(args, "--no-dev")
	}

	args = append(args, cmd.ExtraArgs...)

	result, err := ur.executeCommand(ctx, "uv", args, cmd.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("UV export failed: %w", err)
	}

	if !result.Success {
		return ur.createDetailedExportError(result, cmd)
	}

	return nil
}

// createDetailedExportError creates a detailed error message for export failures
func (ur *UvCommandRunner) createDetailedExportError(result *CommandResult, cmd *UvExportCommand) error {
	errorMsg := fmt.Sprintf("UV export failed with exit code %d", result.ExitCode)

	if cmd.PackageName != "" {
		errorMsg += fmt.Sprintf(" (exporting package: %s)", cmd.PackageName)
	}
	if cmd.OutputFile != "" {
		errorMsg += fmt.Sprintf(" to file: %s", cmd.OutputFile)
	}
	errorMsg += fmt.Sprintf(" in workspace: %s", cmd.WorkspaceDir)

	if result.Stderr != "" {
		errorMsg += fmt.Sprintf("\nError output: %s", result.Stderr)
	}

	if strings.Contains(result.Stderr, "package") && strings.Contains(result.Stderr, "not found") {
		errorMsg += "\nSuggestion: Check if the package name is correct and exists in the workspace"
	} else if strings.Contains(result.Stderr, "lock") {
		errorMsg += "\nSuggestion: Run 'uv sync' first to ensure dependencies are resolved"
	}

	return fmt.Errorf("%s", errorMsg)
}

// executeCommand executes a command with timeout and progress logging
func (ur *UvCommandRunner) executeCommand(ctx context.Context, command string, args []string, workingDir string) (*CommandResult, error) {
	if ur == nil {
		return nil, fmt.Errorf("UV command runner is nil")
	}

	startTime := time.Now()

	cmdCtx, cancel := context.WithTimeout(ctx, ur.commandTimeout)
	defer cancel()

	cmd := process.CommandContext(cmdCtx, command, args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	cmd.Env = os.Environ()

	// Log progress every 30 seconds to detect hangs
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				slog.Warn("UV command still running",
					"command", command,
					"elapsed", time.Since(startTime),
					"workingDir", workingDir)
			}
		}
	}()

	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)
	close(done)

	result := &CommandResult{
		Duration: duration,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Success = false
		result.Stderr = string(output)
		slog.Error("command error output", "stderr", result.Stderr)
	} else {
		result.ExitCode = 0
		result.Success = true
		result.Stdout = string(output)
	}

	return result, nil
}
