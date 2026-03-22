package ruby

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/path"
	"github.com/sst/sst/v3/pkg/runtime"
)

type Worker struct {
	stdout io.ReadCloser
	stderr io.ReadCloser
	cmd    *exec.Cmd
}

func (w *Worker) Stop() {
	process.Kill(w.cmd.Process)
}

func (w *Worker) Logs() io.ReadCloser {
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		var wg sync.WaitGroup
		wg.Add(2)

		copyStream := func(dst io.Writer, src io.Reader, name string) {
			defer wg.Done()
			buf := make([]byte, 1024)
			for {
				n, err := src.Read(buf)
				if n > 0 {
					_, werr := dst.Write(buf[:n])
					if werr != nil {
						slog.Error("error writing to pipe", "stream", name, "err", werr)
						return
					}
				}
				if err != nil {
					if err != io.EOF {
						slog.Error("error reading from stream", "stream", name, "err", err)
					}
					return
				}
			}
		}

		go copyStream(writer, w.stdout, "stdout")
		go copyStream(writer, w.stderr, "stderr")

		wg.Wait()
	}()

	return reader
}

type RubyRuntime struct {
	lastBuiltHandler map[string]string
}

func New() *RubyRuntime {
	return &RubyRuntime{
		lastBuiltHandler: map[string]string{},
	}
}

func (r *RubyRuntime) Match(runtime string) bool {
	return strings.HasPrefix(runtime, "ruby")
}

func (r *RubyRuntime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	slog.Info("building ruby function", "handler", input.Handler)

	type Properties struct {
		Architecture string `json:"architecture"`
	}
	var props Properties
	if err := json.Unmarshal(input.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %v", err)
	}

	arch := props.Architecture
	if arch == "" {
		arch = "x86_64"
	}

	// 1. Prepare artifact directory
	out := input.Out()
	rootDir := path.ResolveRootDir(input.CfgPath)
	handlerDir := filepath.Dir(filepath.Join(rootDir, input.Handler))
	
	// Find Gemfile
	gemfilePath := ""
	currentDir := handlerDir
	for {
		if _, err := os.Stat(filepath.Join(currentDir, "Gemfile")); err == nil {
			gemfilePath = filepath.Join(currentDir, "Gemfile")
			break
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir || parent == rootDir {
			break
		}
		currentDir = parent
	}

	// Copy all files from handler directory (or workspace root if Gemfile found) to out
	copySource := handlerDir
	if gemfilePath != "" {
		copySource = filepath.Dir(gemfilePath)
	}
	
	err := exec.Command("cp", "-r", copySource+"/.", out).Run()
	if err != nil {
		return nil, fmt.Errorf("failed to copy source files: %v", err)
	}

	if input.IsContainer && !input.Dev {
		// Handle container build
		slog.Info("container build", "out", out)
		dockerfilePath := filepath.Join(copySource, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); err != nil {
			// Copy default Dockerfile
			defaultDockerfilePath := filepath.Join(path.ResolvePlatformDir(input.CfgPath), "/dist/dockerfiles/ruby.Dockerfile")
			err := copyFile(defaultDockerfilePath, filepath.Join(out, "Dockerfile"))
			if err != nil {
				return nil, fmt.Errorf("failed to copy default Dockerfile: %v", err)
			}
		} else {
			copyFile(dockerfilePath, filepath.Join(out, "Dockerfile"))
		}

		return &runtime.BuildOutput{
			Handler: input.Handler,
			Errors:  []string{},
		}, nil
	}

	// 2. Install dependencies if Gemfile exists
	if gemfilePath != "" {
		slog.Info("installing ruby dependencies", "dir", out)
		
		// For Ruby, we usually want to bundle into vendor/bundle
		cmd := process.CommandContext(ctx, "bundle", "config", "set", "--local", "deployment", "true")
		cmd.Dir = out
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to set bundle config: %v", err)
		}

		cmd = process.CommandContext(ctx, "bundle", "config", "set", "--local", "path", "vendor/bundle")
		cmd.Dir = out
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to set bundle path: %v", err)
		}

		cmd = process.CommandContext(ctx, "bundle", "install")
		cmd.Dir = out
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to run bundle install: %v\n%s", err, string(output))
		}
	}

	r.lastBuiltHandler[input.FunctionID] = input.Handler

	return &runtime.BuildOutput{
		Handler: input.Handler,
		Errors:  []string{},
	}, nil
}

func (r *RubyRuntime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	// Copy the ruby bridge
	bridgePath := filepath.Join(input.Build.Out, "index.rb")
	if _, err := os.Stat(bridgePath); os.IsNotExist(err) {
		err := copyFile(filepath.Join(path.ResolvePlatformDir(input.CfgPath), "/dist/ruby-runtime/index.rb"), bridgePath)
		if err != nil {
			return nil, fmt.Errorf("failed to copy ruby bridge: %v", err)
		}
	}

	// Run with bundle exec if Gemfile exists
	args := []string{"ruby", "index.rb", input.Build.Handler}
	if _, err := os.Stat(filepath.Join(input.Build.Out, "Gemfile")); err == nil {
		args = []string{"bundle", "exec", "ruby", "index.rb", input.Build.Handler}
	}

	cmd := process.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(input.Env, "AWS_LAMBDA_RUNTIME_API="+input.Server)
	cmd.Dir = input.Build.Out
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	slog.Info("starting ruby worker", "args", cmd.Args)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &Worker{
		stdout: stdout,
		stderr: stderr,
		cmd:    cmd,
	}, nil
}

func (r *RubyRuntime) ShouldRebuild(functionID string, file string) bool {
	return strings.HasSuffix(file, ".rb") || filepath.Base(file) == "Gemfile" || filepath.Base(file) == "Gemfile.lock"
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
