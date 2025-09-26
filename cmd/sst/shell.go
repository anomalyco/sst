package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/sst/sst/v3/internal/util"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/provider"
)

func CmdShell(c *cli.Cli) error {
	p, err := c.InitProject()
	if err != nil {
		return err
	}
	defer p.Cleanup()

	var args []string
	for _, arg := range c.Arguments() {
		args = append(args, arg)
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
	cmd := process.Command(
		args[0],
		args[1:]...,
	)
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

	// Debug logging for Windows issue
	fmt.Fprintf(os.Stderr, "[DEBUG] Platform: %s\n", runtime.GOOS)
	fmt.Fprintf(os.Stderr, "[DEBUG] Found %d links in completed state\n", len(complete.Links))
	if _, exists := complete.Links["activityTable"]; exists {
		fmt.Fprintf(os.Stderr, "[DEBUG] activityTable found in links\n")
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] activityTable NOT found in links\n")
		fmt.Fprintf(os.Stderr, "[DEBUG] Available links:\n")
		for name := range complete.Links {
			fmt.Fprintf(os.Stderr, "[DEBUG]   - %s\n", name)
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
			fmt.Fprintf(os.Stderr, "[DEBUG] Adding env var SST_RESOURCE_%s (length: %d)\n", resource, len(jsonValue))
		}
		appEnv := fmt.Sprintf("SST_RESOURCE_App=%s", fmt.Sprintf(`{"name": "%s", "stage": "%s" }`, p.App().Name, p.App().Stage))
		cmd.Env = append(cmd.Env, appEnv)

		fmt.Fprintf(os.Stderr, "[DEBUG] Adding env var SST_RESOURCE_App\n")

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
	sstResourceCount := 0
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "SST_RESOURCE_") {
			sstResourceCount++
		}
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] Total env vars: %d, SST_RESOURCE vars: %d\n", len(cmd.Env), sstResourceCount)
	fmt.Fprintf(os.Stderr, "[DEBUG] Executing command: %s %v\n", args[0], args[1:])

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		return util.NewReadableError(err, err.Error())
	}
	return nil
}
