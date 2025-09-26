package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/sst/sst/v3/internal/util"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/provider"
)

func CmdShell(c *cli.Cli) error {
	// Debug output to file for Windows debugging
	var debugFile *os.File
	if runtime.GOOS == "windows" {
		debugFile, _ = os.Create("sst-shell-debug.log")
		if debugFile != nil {
			defer debugFile.Close()
			fmt.Fprintf(debugFile, "=== SST Shell Debug Log ===\n")
			fmt.Fprintf(debugFile, "Time: %s\n", time.Now().Format(time.RFC3339))
			fmt.Fprintf(debugFile, "Platform: %s\n", runtime.GOOS)
		}
	}

	p, err := c.InitProject()
	if err != nil {
		return err
	}
	defer p.Cleanup()

	var args []string
	for _, arg := range c.Arguments() {
		args = append(args, arg)
	}

	if debugFile != nil {
		fmt.Fprintf(debugFile, "\n=== Command Arguments ===\n")
		fmt.Fprintf(debugFile, "Raw arguments: %v\n", args)
	}
	cwd, _ := os.Getwd()
	currentDir := cwd
	for {
		nodeBinPath := filepath.Join(currentDir, "node_modules", ".bin")
		newPath := nodeBinPath + string(os.PathListSeparator) + os.Getenv("PATH")
		os.Setenv("PATH", newPath)
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}
	if len(args) == 0 {
		switch runtime.GOOS {
		case "windows":
			args = append(args, "cmd")
		default:
			args = append(args, "sh")
		}
	}

	// On Windows, when executing commands like cross-env that manage their own environment,
	// bypass cmd.exe and execute directly when possible
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" && len(args) > 0 && args[0] != "cmd" {
		// Try to find the executable directly
		if execPath, err := exec.LookPath(args[0]); err == nil {
			if debugFile != nil {
				fmt.Fprintf(debugFile, "Found executable: %s\n", execPath)
			}
			// Use exec.Command directly instead of process.Command to avoid potential issues
			cmd = exec.Command(execPath, args[1:]...)
			// Track it manually since we're not using process.Command
			// (Note: this skips the process tracking in the process package)
		} else {
			if debugFile != nil {
				fmt.Fprintf(debugFile, "Could not find executable %s, using process.Command\n", args[0])
			}
			cmd = process.Command(args[0], args[1:]...)
		}
	} else {
		cmd = process.Command(args[0], args[1:]...)
	}
	// Initialize with current environment variables
	cmd.Env = os.Environ()
	// Add SST-specific environment variables
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("PS1=%s/%s> ", p.App().Name, p.App().Stage),
	)
	complete, err := p.GetCompleted(c.Context)
	if err != nil {
		return err
	}

	// Debug logging
	if debugFile != nil {
		fmt.Fprintf(debugFile, "\n=== Resource Links ===\n")
		fmt.Fprintf(debugFile, "Found %d links in completed state\n", len(complete.Links))
		if _, exists := complete.Links["activityTable"]; exists {
			fmt.Fprintf(debugFile, "activityTable: FOUND\n")
		} else {
			fmt.Fprintf(debugFile, "activityTable: NOT FOUND\n")
			fmt.Fprintf(debugFile, "Available links:\n")
			for name := range complete.Links {
				fmt.Fprintf(debugFile, "  - %s\n", name)
			}
		}
	}

	target := c.String("target")
	if target != "" {
		cmd.Env = append(cmd.Env, c.Env()...)
		env, err := p.EnvFor(c.Context, complete, target)
		if err != nil {
			return err
		}
		for key, value := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}
	if target == "" {
		// Add SST resource environment variables
		for resource, value := range complete.Links {
			jsonValue, err := json.Marshal(value.Properties)
			if err != nil {
				return err
			}
			envVar := fmt.Sprintf("SST_RESOURCE_%s=%s", resource, string(jsonValue))
			cmd.Env = append(cmd.Env, envVar)

			// Debug logging
			if debugFile != nil {
				fmt.Fprintf(debugFile, "Setting SST_RESOURCE_%s (length: %d)\n", resource, len(jsonValue))
			}
		}
		appEnv := fmt.Sprintf("SST_RESOURCE_App=%s", fmt.Sprintf(`{"name": "%s", "stage": "%s" }`, p.App().Name, p.App().Stage))
		cmd.Env = append(cmd.Env, appEnv)

		if debugFile != nil {
			fmt.Fprintf(debugFile, "Setting SST_RESOURCE_App\n")
		}

		aws, ok := p.Provider("aws")
		if ok {
			// Remove AWS_PROFILE from environment
			filteredEnv := []string{}
			for _, envVar := range cmd.Env {
				if !strings.HasPrefix(envVar, "AWS_PROFILE=") {
					filteredEnv = append(filteredEnv, envVar)
				}
			}
			cmd.Env = filteredEnv

			provider := aws.(*provider.AwsProvider)
			cfg := provider.Config()
			creds, err := cfg.Credentials.Retrieve(c.Context)
			if err != nil {
				return err
			}
			cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", creds.AccessKeyID))
			cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", creds.SecretAccessKey))
			cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SESSION_TOKEN=%s", creds.SessionToken))
			if cfg.Region != "" {
				cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_REGION=%s", cfg.Region))
			}
		}
	}
	// Debug: Verify environment variables are set
	if debugFile != nil {
		fmt.Fprintf(debugFile, "\n=== Final Environment ===\n")
		sstResourceCount := 0
		totalEnvSize := 0
		for _, env := range cmd.Env {
			totalEnvSize += len(env) + 1 // +1 for null terminator
			if strings.HasPrefix(env, "SST_RESOURCE_") {
				sstResourceCount++
				// Log each SST_RESOURCE variable
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					fmt.Fprintf(debugFile, "ENV: %s = %d bytes\n", parts[0], len(parts[1]))
				}
			}
		}
		fmt.Fprintf(debugFile, "Total env vars: %d\n", len(cmd.Env))
		fmt.Fprintf(debugFile, "SST_RESOURCE vars: %d\n", sstResourceCount)
		fmt.Fprintf(debugFile, "Total environment size: %d bytes\n", totalEnvSize)
		if runtime.GOOS == "windows" && totalEnvSize > 32767 {
			fmt.Fprintf(debugFile, "WARNING: Environment size exceeds Windows limit (32KB)\n")
		}
		fmt.Fprintf(debugFile, "\n=== Executing Command ===\n")
		fmt.Fprintf(debugFile, "Command: %s\n", args[0])
		fmt.Fprintf(debugFile, "Arguments: %v\n", args[1:])
		debugFile.Sync() // Force write to disk before executing command
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		return util.NewReadableError(err, err.Error())
	}
	return nil
}
