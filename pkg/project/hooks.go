package project

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/flag"
	"github.com/sst/sst/v3/pkg/js"
	"github.com/sst/sst/v3/pkg/process"
)

const HookDeployAfter = "deploy.after"

type DeployHookResult struct {
	Success   bool                   `json:"success"`
	Errors    []DeployHookError      `json:"errors"`
	Outputs   map[string]interface{} `json:"outputs"`
	Resources int                    `json:"resources"`
	App       DeployHookApp          `json:"app"`
}

type DeployHookError struct {
	Message string   `json:"message"`
	URN     string   `json:"urn,omitempty"`
	Help    []string `json:"help,omitempty"`
}

type DeployHookApp struct {
	Name  string `json:"name"`
	Stage string `json:"stage"`
}

func (p *Project) RunDeployHook(complete *CompleteEvent) error {
	log := slog.Default().With("service", "project.hooks")

	if complete == nil {
		log.Warn("skipping deploy.after hook: complete event is nil")
		return nil
	}

	log.Info("running deploy.after hook")

	// Set up timeout context
	timeout := 30 * time.Second
	if flag.SST_HOOK_TIMEOUT != "" {
		seconds, err := strconv.Atoi(flag.SST_HOOK_TIMEOUT)
		if err != nil {
			log.Warn("invalid SST_HOOK_TIMEOUT value, using default",
				"value", flag.SST_HOOK_TIMEOUT,
				"default", timeout,
				"err", err)
		} else if seconds <= 0 {
			log.Warn("SST_HOOK_TIMEOUT must be positive, using default",
				"value", seconds,
				"default", timeout)
		} else {
			timeout = time.Duration(seconds) * time.Second
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	bus.Publish(&HookStartEvent{Hook: HookDeployAfter})

	// Build the hook result from CompleteEvent
	hookErrors := make([]DeployHookError, len(complete.Errors))
	for i, err := range complete.Errors {
		hookErrors[i] = DeployHookError{
			Message: err.Message,
			URN:     err.URN,
			Help:    err.Help,
		}
	}

	result := DeployHookResult{
		Success:   len(complete.Errors) == 0,
		Errors:    hookErrors,
		Outputs:   complete.Outputs,
		Resources: len(complete.Resources),
		App: DeployHookApp{
			Name:  p.app.Name,
			Stage: p.app.Stage,
		},
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		bus.Publish(&HookErrorEvent{Hook: HookDeployAfter, Error: err.Error()})
		return fmt.Errorf("failed to marshal hook result: %w", err)
	}

	outfile := filepath.Join(p.PathPlatformDir(), fmt.Sprintf("hook.deploy.%v.mjs", time.Now().UnixMilli()))

	buildResult, err := js.Build(js.EvalOptions{
		Dir:     p.PathRoot(),
		Outfile: outfile,
		Define: map[string]string{
			"$hookResult": string(resultBytes),
		},
		Code: fmt.Sprintf(`
import mod from '%s';
const result = $hookResult;
if (mod.hooks?.deploy?.after) {
	await mod.hooks.deploy.after(result);
}
`,
			filepath.ToSlash(p.PathConfig()),
		),
	})
	if err != nil {
		bus.Publish(&HookErrorEvent{Hook: HookDeployAfter, Error: err.Error()})
		return fmt.Errorf("failed to build hook: %w", err)
	}
	if !flag.SST_NO_CLEANUP {
		defer js.Cleanup(buildResult)
	}

	log.Info("executing deploy.after hook", "outfile", outfile, "timeout", timeout)
	node := process.CommandContext(ctx, "node", "--no-warnings", outfile)
	node.Env = os.Environ()
	output, err := node.CombinedOutput()
	if err != nil {
		var errMsg string
		switch ctx.Err() {
		case context.Canceled:
			errMsg = fmt.Sprintf("hook was canceled\n%s", string(output))
		case context.DeadlineExceeded:
			errMsg = fmt.Sprintf("hook timed out after %v\n%s", timeout, string(output))
		default:
			errMsg = fmt.Sprintf("hook execution failed: %s\n%s", err.Error(), string(output))
		}
		bus.Publish(&HookErrorEvent{Hook: HookDeployAfter, Error: errMsg})
		log.Error("deploy.after hook failed", "err", err, "output", string(output))
		return fmt.Errorf("hook %s failed: %s", HookDeployAfter, errMsg)
	}

	// Log any output from the hook
	if len(output) > 0 {
		log.Info("deploy.after hook output", "output", string(output))
	}

	bus.Publish(&HookCompleteEvent{Hook: HookDeployAfter})
	log.Info("deploy.after hook completed")
	return nil
}
