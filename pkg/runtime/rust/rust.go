package rust

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

	"github.com/BurntSushi/toml"
	"github.com/sst/sst/v3/internal/fs"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/runtime"
)

type Runtime struct {
	mut         sync.Mutex
	directories map[string]string
}

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

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(writer, w.stdout)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(writer, w.stderr)
	}()

	go func() {
		wg.Wait()
		defer writer.Close()
	}()

	return reader
}

func New() *Runtime {
	return &Runtime{
		directories: map[string]string{},
	}
}

func (r *Runtime) Match(runtime string) bool {
	return runtime == "rust"
}

type Properties struct {
	Architecture string `json:"architecture"`
}

type CargoToml struct {
	Package struct {
		Name string `toml:"name"`
	} `toml:"package"`
}

func (r *Runtime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	r.mut.Lock()
	defer r.mut.Unlock()
	var properties Properties
	json.Unmarshal(input.Properties, &properties)

	// split handler into path and function name
	parts := strings.Split(input.Handler, ".")
	handler := strings.Join(parts[:len(parts)-1], ".")

	// Locate cargo.toml/Cargo.toml
	cargotomlpath, err := fs.FindUp(handler, "cargo.toml")
	if err != nil {
		cargotomlpath, err = fs.FindUp(handler, "Cargo.toml")
		if err != nil {
			return nil, err
		}
	}

	// root of project
	root := filepath.Dir(cargotomlpath)

	// append args to the command
	args := []string{}
	env := os.Environ()
	if input.Dev {
		args = append(args, "build")
	} else {
		args = []string{"lambda", "build"}
		args = append(args, "--release")
		if properties.Architecture == "arm64" {
			args = append(args, "--arm64")
		}
	}

	cmd := process.Command("cargo", args...)
	cmd.Dir = root
	cmd.Env = env
	slog.Info("running cargo build", "cmd", cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &runtime.BuildOutput{
			Errors: []string{string(output)},
		}, nil
	}

	// get the default output of the crate
	var cargotoml CargoToml
	_, err = toml.DecodeFile(cargotomlpath, &cargotoml)
	if err != nil {
		return nil, err
	}

	// if binary name is not specified, use the package name
	name := cargotoml.Package.Name
	if len(parts) >= 2 {
		name = parts[len(parts)-1]
	}
	binary := filepath.Join(root, "target",
		map[bool]string{
			true:  filepath.Join("debug", name),
			false: filepath.Join("lambda", name, "bootstrap"),
		}[input.Dev],
	)
	out := filepath.Join(input.Out(), "bootstrap")

	r.directories[input.FunctionID], _ = filepath.Abs(root)

	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	source, err := os.Open(binary)
	if err != nil {
		return nil, fmt.Errorf("failed to open source binary: %w", err)
	}
	defer source.Close()

	destination, err := os.Create(out)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return nil, fmt.Errorf("failed to copy binary: %w", err)
	}

	if err := os.Chmod(out, 0755); err != nil {
		return nil, fmt.Errorf("failed to make binary executable: %w", err)
	}

	return &runtime.BuildOutput{
		Handler:    "bootstrap",
		Sourcemaps: []string{},
		Errors:     []string{},
		Out:        root,
	}, nil
}

func (r *Runtime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	cmd := process.Command(
		filepath.Join(input.Build.Out, input.Build.Handler),
	)
	slog.Info("running server binary", "server", input.Server)
	cmd.Env = input.Env
	cmd.Env = append(cmd.Env, "AWS_LAMBDA_RUNTIME_API=http://"+input.Server)
	cmd.Env = append(cmd.Env, "AWS_LAMBDA_FUNCTION_MEMORY_SIZE=1024")
	cmd.Dir = input.Build.Out
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	cmd.Start()
	return &Worker{
		stdout,
		stderr,
		cmd,
	}, nil
}

func (r *Runtime) ShouldRebuild(functionID string, file string) bool {
	// copied from go
	if !strings.HasSuffix(file, ".rs") {
		return false
	}
	match, ok := r.directories[functionID]
	if !ok {
		return false
	}
	slog.Info("checking if file needs to be rebuilt", "file", file, "match", match)
	rel, err := filepath.Rel(match, file)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
