package golang

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sst/sst/v3/internal/fs"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/runtime"
)

// Runtime implements [runtime.Runtime] for Go-based Lambda functions.
// It builds each handler with `go build` and tracks the per-handler
// import graph so [Runtime.ShouldRebuild] only fires for files that
// actually compile into the handler's binary.
type Runtime struct {
	mut                sync.RWMutex
	directories        map[string]string
	files              map[string]map[string]struct{}
	gomodcache         string
	gomodcacheOnce     sync.Once
	gomodcacheOverride string
}

// Worker is a running Lambda handler binary started by [Runtime.Run].
type Worker struct {
	stdout io.ReadCloser
	stderr io.ReadCloser
	cmd    *exec.Cmd
}

// Stop terminates the running worker process.
func (w *Worker) Stop() {
	process.Kill(w.cmd.Process)
}

// Logs returns a single reader that streams the worker's stdout and
// stderr interleaved.
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

// New returns a Runtime with empty per-function state.
func New() *Runtime {
	return &Runtime{
		directories: map[string]string{},
		files:       map[string]map[string]struct{}{},
	}
}

// Match reports whether this Runtime handles functions declared with
// `runtime: "go"`.
func (r *Runtime) Match(runtime string) bool {
	return runtime == "go"
}

// Properties is the runtime-specific configuration block in the SST
// component output for a Go function.
type Properties struct {
	Architecture string `json:"architecture"`
}

func goarchFromArchitecture(arch string) string {
	if arch == "arm64" {
		return "arm64"
	}
	return "amd64"
}

// Build compiles the handler with `go build` and captures its
// transitive import graph for use by ShouldRebuild. A failed compile is
// returned as a non-nil BuildOutput with errors; a failed graph capture
// is logged and disables ShouldRebuild for that function until the
// next successful build.
func (r *Runtime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	var properties Properties
	json.Unmarshal(input.Properties, &properties)

	gomod, err := fs.FindUp(input.Handler, "go.mod")
	if err != nil {
		return nil, err
	}
	root := filepath.Dir(gomod)
	src, _ := filepath.Rel(root, input.Handler)
	out := filepath.Join(input.Out(), "bootstrap")
	args := []string{"build"}
	env := os.Environ()
	if !input.Dev {
		args = append(args, "-ldflags", "-s -w")
		env = append(env, "CGO_ENABLED=0")
		env = append(env, "GOOS=linux")
		env = append(env, "GOARCH="+goarchFromArchitecture(properties.Architecture))
	}
	// "./" prefix forces relative path resolution; without it `go build`
	// treats a sub-path like "commands/connect" as a stdlib package name.
	args = append(args, "-o", out, "./"+src)
	cmd := process.Command("go", args...)
	cmd.Dir = root
	cmd.Env = env
	slog.Info("running go build", "cmd", cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &runtime.BuildOutput{
			Errors: []string{string(output)},
		}, nil
	}

	absRoot, _ := filepath.Abs(root)
	deps, depErr := r.captureDeps(ctx, absRoot, src, env)

	r.mut.Lock()
	r.directories[input.FunctionID] = absRoot
	if depErr != nil {
		delete(r.files, input.FunctionID)
	} else {
		r.files[input.FunctionID] = deps
	}
	r.mut.Unlock()

	if depErr != nil {
		slog.Warn("failed to capture go deps; ShouldRebuild disabled until next build",
			"fn", input.FunctionID,
			"err", depErr,
			"root", absRoot,
			"src", src,
			"goflags", os.Getenv("GOFLAGS"),
			"gomodcache", r.resolveGoModCache(env),
		)
	}

	return &runtime.BuildOutput{
		Handler:    "bootstrap",
		Sourcemaps: []string{},
		Errors:     []string{},
		Out:        root,
	}, nil
}

// Run starts the previously-built handler binary as a worker.
func (r *Runtime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	cmd := process.Command(
		filepath.Join(input.Build.Out, input.Build.Handler),
	)
	slog.Info("running go run", "server", input.Server)
	cmd.Env = input.Env
	cmd.Env = append(cmd.Env, "AWS_LAMBDA_RUNTIME_API="+input.Server)
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

// ShouldRebuild reports whether a file change must trigger a rebuild
// of the given Lambda. Returns true only when the file is part of the
// handler's transitive import graph captured by the last Build.
func (r *Runtime) ShouldRebuild(functionID string, file string) bool {
	if !strings.HasSuffix(file, ".go") {
		return false
	}
	r.mut.RLock()
	files, ok := r.files[functionID]
	r.mut.RUnlock()
	if !ok {
		return false
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return false
	}
	if _, found := files[abs]; !found {
		return false
	}
	slog.Info("rebuilding go function", "fn", functionID, "file", abs)
	return true
}

func (r *Runtime) captureDeps(ctx context.Context, root, src string, env []string) (map[string]struct{}, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-deps", "-json", "./"+src)
	cmd.Dir = root
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseGoListOutput(strings.NewReader(string(out)), r.resolveGoModCache(env))
}

func parseGoListOutput(r io.Reader, gomodcache string) (map[string]struct{}, error) {
	type pkgInfo struct {
		Standard   bool
		Goroot     bool
		Dir        string
		GoFiles    []string
		CgoFiles   []string
		EmbedFiles []string
	}

	files := make(map[string]struct{})
	dec := json.NewDecoder(r)
	for {
		var p pkgInfo
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if p.Standard || p.Goroot {
			continue
		}
		if gomodcache != "" && isUnderDir(p.Dir, gomodcache) {
			continue
		}
		all := make([]string, 0, len(p.GoFiles)+len(p.CgoFiles)+len(p.EmbedFiles))
		all = append(all, p.GoFiles...)
		all = append(all, p.CgoFiles...)
		all = append(all, p.EmbedFiles...)
		for _, name := range all {
			files[filepath.Join(p.Dir, name)] = struct{}{}
		}
	}
	return files, nil
}

func (r *Runtime) resolveGoModCache(env []string) string {
	if r.gomodcacheOverride != "" {
		return r.gomodcacheOverride
	}
	r.gomodcacheOnce.Do(func() {
		cmd := exec.Command("go", "env", "GOMODCACHE")
		cmd.Env = env
		out, err := cmd.Output()
		if err != nil {
			return
		}
		r.gomodcache = strings.TrimSpace(string(out))
	})
	return r.gomodcache
}

func isUnderDir(path, dir string) bool {
	cleanPath := filepath.Clean(path)
	cleanDir := filepath.Clean(dir)
	if cleanPath == cleanDir {
		return true
	}
	rel, err := filepath.Rel(cleanDir, cleanPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
