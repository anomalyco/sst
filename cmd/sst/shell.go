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
			cmd.Env = append(cmd.Env, fmt.Sprintf("SST_RESOURCE_%s=%s", resource, string(jsonValue)))
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("SST_RESOURCE_App=%s", fmt.Sprintf(`{"name": "%s", "stage": "%s" }`, p.App().Name, p.App().Stage)))

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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		return util.NewReadableError(err, err.Error())
	}
	return nil
}
