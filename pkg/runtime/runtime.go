package runtime

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sst/sst/v3/pkg/project/path"
)

type Runtime interface {
	Match(runtime string) bool
	Build(ctx context.Context, input *BuildInput) (*BuildOutput, error)
	Run(ctx context.Context, input *RunInput) (Worker, error)
	ShouldRebuild(functionID string, path string) bool
	// ShouldRunEagerly controls whether workers are started immediately after a rebuild
	// or lazily on first invocation.
	//
	// Background: When a file changes, SST stops all affected workers and rebuilds them.
	// By default (returning true), workers are restarted immediately after rebuild.
	// This works well for runtimes like Node.js where esbuild provides precise per-function
	// dependency tracking, so only a few functions rebuild on each change.
	//
	// For Python, we lack precise dependency tracking - a change to shared library code
	// triggers rebuilds for ALL 50+ functions. Starting all workers immediately causes:
	// - 50+ processes competing for CPU/memory during startup
	// - ~2 second startup time per worker
	// - Long delays before the system is responsive again
	//
	// By returning false, Python opts into lazy startup: workers are stopped and marked
	// as needing rebuild, but only actually start when invoked. This means only the
	// functions you're actively using restart immediately.
	ShouldRunEagerly() bool
}

type Worker interface {
	Stop()
	Logs() io.ReadCloser
}

type BuildInput struct {
	CfgPath       string
	Dev           bool                       `json:"dev"`
	FunctionID    string                     `json:"functionID"`
	Handler       string                     `json:"handler"`
	Bundle        string                     `json:"bundle"`
	Runtime       string                     `json:"runtime"`
	Properties    json.RawMessage            `json:"properties"`
	Links         map[string]json.RawMessage `json:"links"`
	EncryptionKey string                     `json:"encryptionKey"`
	CopyFiles     []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"copyFiles"`
	IsContainer bool `json:"isContainer,omitempty"`
}

func (input *BuildInput) Out() string {
	suffix := "-src"
	if input.Dev {
		suffix = "-dev"
	}
	return filepath.Join(path.ResolveWorkingDir(input.CfgPath), "artifacts", input.FunctionID+suffix)
}

type BuildOutput struct {
	Out        string   `json:"out"`
	Handler    string   `json:"handler"`
	Errors     []string `json:"errors"`
	Sourcemaps []string `json:"sourcemaps"`
}

type RunInput struct {
	CfgPath    string
	Runtime    string
	Server     string
	FunctionID string
	WorkerID   string
	Build      *BuildOutput
	Env        []string
}

type Collection struct {
	runtimes []Runtime
	cfgPath  string
	targets  map[string]*BuildInput
}

func NewCollection(platform string, runtimes ...Runtime) *Collection {
	return &Collection{
		runtimes: runtimes,
		cfgPath:  platform,
		targets:  map[string]*BuildInput{},
	}
}

func (c *Collection) Runtime(input string) (Runtime, bool) {
	for _, runtime := range c.runtimes {
		if runtime.Match(input) {
			return runtime, true
		}
	}
	return nil, false
}

func (c *Collection) Build(ctx context.Context, input *BuildInput) (*BuildOutput, error) {
	slog.Info("building function", "runtime", input.Runtime, "functionID", input.FunctionID)
	defer slog.Info("function built", "runtime", input.Runtime, "functionID", input.FunctionID)
	out := input.Out()
	var result *BuildOutput

	if input.Bundle != "" {
		out = input.Bundle
		result = &BuildOutput{
			Handler: input.Handler,
			Errors:  []string{},
		}
	}

	if input.Bundle == "" {
		err := os.RemoveAll(out)
		if err != nil {
			return nil, err
		}
		err = os.MkdirAll(out, 0755)
		if err != nil {
			return nil, err
		}
		runtime, ok := c.Runtime(input.Runtime)
		if !ok {
			return nil, fmt.Errorf("Runtime not found: %v", input.Runtime)
		}
		result, err = runtime.Build(ctx, input)
		if err != nil {
			return nil, err
		}
	}

	result.Out = out
	if result.Sourcemaps == nil {
		result.Sourcemaps = []string{}
	}

	if len(input.CopyFiles) > 0 {
		for _, item := range input.CopyFiles {
			from, err := filepath.Abs(item.From)
			if err != nil {
				return nil, err
			}
			var dest string
			if item.To != "" {
				dest = filepath.Join(out, item.To)
			} else {
				dest = filepath.Join(out, item.From)
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return nil, err
			}
			if input.Dev {
				if err := os.Symlink(from, dest); err != nil {
					if !os.IsExist(err) {
						return nil, err
					}
				}
			}
			// copying files still happens in node
			if !input.Dev && false {
				sourceFile, err := os.Open(from)
				if err != nil {
					return nil, err
				}
				defer sourceFile.Close()
				destFile, err := os.Create(dest)
				if err != nil {
					return nil, err
				}
				defer destFile.Close()
				_, err = io.Copy(destFile, sourceFile)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if input.EncryptionKey != "" {
		key, err := base64.StdEncoding.DecodeString(input.EncryptionKey)
		if err != nil {
			return nil, err
		}
		json, err := json.Marshal(input.Links)
		if err != nil {
			return nil, err
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		ciphertext := gcm.Seal(nil, make([]byte, 12), json, nil)
		// When a shared bundle is used, multiple functions share the same output
		// directory (result.Out == input.Bundle). Writing "resource.enc" from
		// concurrent Runtime.Build calls races: partial writes can leave a
		// truncated or mixed ciphertext that fails AES-GCM authentication on
		// the Lambda's first cold start (observed as `Decipheriv` errors at
		// runtime initialization).
		//
		// Namespace the file under a per-function subdirectory so each function
		// writes to its own path (eliminating the race) AND each function's
		// uploaded zip can exclude every sibling's subtree (the .sst/ prefix
		// gives a single, stable glob target for the zipper to filter on). The
		// Lambda reads its own file via SST_KEY_FILE, which the platform
		// component points at the matching path.
		resourcePath := filepath.Join(result.Out, "resource.enc")
		if input.Bundle != "" {
			resourcePath = filepath.Join(result.Out, ".sst", input.FunctionID, "resource.enc")
			if err := os.MkdirAll(filepath.Dir(resourcePath), 0755); err != nil {
				return nil, err
			}
		}
		err = os.WriteFile(resourcePath, ciphertext, 0644)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c *Collection) Run(ctx context.Context, input *RunInput) (Worker, error) {
	slog.Info("running function", "runtime", input.Runtime, "functionID", input.FunctionID)
	runtime, ok := c.Runtime(input.Runtime)
	input.Env = append(input.Env, "SST_LIVE=true")
	input.Env = append(input.Env, "SST_DEV=true")
	if !ok {
		return nil, fmt.Errorf("runtime not found")
	}
	return runtime.Run(ctx, input)
}

func (c *Collection) ShouldRebuild(runtime string, functionID string, file string) bool {
	slog.Info("checking if function should be rebuilt", "runtime", runtime, "functionID", functionID, "file", file, "runtime", runtime)
	r, ok := c.Runtime(runtime)
	if !ok {
		return false
	}
	result := r.ShouldRebuild(functionID, file)
	slog.Info("should rebuild", "result", result, "functionID", functionID)
	return result
}

func (c *Collection) ShouldRunEagerly(runtime string) bool {
	r, ok := c.Runtime(runtime)
	if !ok {
		return true // Default to eager for unknown runtimes
	}
	return r.ShouldRunEagerly()
}

func (c *Collection) AddTarget(input *BuildInput) {
	input.CfgPath = c.cfgPath
	c.targets[input.FunctionID] = input
}
