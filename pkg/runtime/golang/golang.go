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
	files              map[string]map[string]struct{}
	pkgDirs            map[string]map[string]struct{}
	gomodPaths         map[string]string
	gomodcacheMut      sync.Mutex
	gomodcache         string
	gomodcacheResolved bool
	// gomodcacheOverride is a test seam set at construction time and
	// never mutated afterwards, so it can be read without the mutex.
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
		files:      map[string]map[string]struct{}{},
		pkgDirs:    map[string]map[string]struct{}{},
		gomodPaths: map[string]string{},
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
	slog.Debug("running go build", "cmd", cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &runtime.BuildOutput{
			Errors: []string{string(output)},
		}, nil
	}

	absRoot, _ := filepath.Abs(root)
	absGomod, _ := filepath.Abs(gomod)
	deps, depErr := r.captureDeps(ctx, absRoot, src, env)

	r.mut.Lock()
	r.gomodPaths[input.FunctionID] = absGomod
	// On capture failure keep the previous graph (if any): the next file
	// edit on a still-tracked file should still rebuild correctly. A
	// successful build below replaces it; until then we degrade
	// gracefully rather than disabling rebuild detection entirely.
	if depErr == nil {
		r.files[input.FunctionID] = deps.files
		r.pkgDirs[input.FunctionID] = deps.pkgDirs
	}
	r.mut.Unlock()

	if depErr != nil {
		slog.Warn("failed to capture go deps; keeping previous graph for rebuild detection",
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
// of the given Lambda. Fires when the file is in the handler's
// captured import graph (covers .go, cgo and //go:embed assets), when
// it is the handler's go.mod/go.sum (those shift the resolved
// dependency set), or when it is a new .go file inside a directory
// that already belongs to the captured graph (e.g. a util.go just
// added to a tracked package).
func (r *Runtime) ShouldRebuild(functionID string, file string) bool {
	r.mut.RLock()
	files, ok := r.files[functionID]
	pkgDirs := r.pkgDirs[functionID]
	gomod := r.gomodPaths[functionID]
	r.mut.RUnlock()
	if !ok {
		return false
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return false
	}
	if _, found := files[abs]; found {
		slog.Info("rebuilding go function", "fn", functionID, "file", abs, "reason", "file_in_graph")
		return true
	}
	if gomod != "" {
		gomodDir := filepath.Dir(gomod)
		if abs == gomod || abs == filepath.Join(gomodDir, "go.sum") {
			slog.Info("rebuilding go function", "fn", functionID, "file", abs, "reason", "gomod_change")
			return true
		}
	}
	if strings.HasSuffix(abs, ".go") {
		for dir := range pkgDirs {
			if isUnderDir(abs, dir) {
				slog.Info("rebuilding go function", "fn", functionID, "file", abs, "reason", "new_file_in_pkg")
				return true
			}
		}
	}
	return false
}

// capturedDeps holds the result of a single captureDeps run: the set
// of source files that compile into the handler's binary, plus the
// set of package directories those files live in (used to detect new
// files added to an already-tracked package between Builds).
type capturedDeps struct {
	files   map[string]struct{}
	pkgDirs map[string]struct{}
}

func (r *Runtime) captureDeps(ctx context.Context, root, src string, env []string) (capturedDeps, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-deps", "-json", "./"+src)
	cmd.Dir = root
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return capturedDeps{}, err
	}
	return parseGoListOutput(strings.NewReader(string(out)), r.resolveGoModCache(env))
}

func parseGoListOutput(r io.Reader, gomodcache string) (capturedDeps, error) {
	type pkgInfo struct {
		Standard   bool
		Goroot     bool
		Dir        string
		GoFiles    []string
		CgoFiles   []string
		EmbedFiles []string
	}

	deps := capturedDeps{
		files:   make(map[string]struct{}),
		pkgDirs: make(map[string]struct{}),
	}
	dec := json.NewDecoder(r)
	for {
		var p pkgInfo
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return capturedDeps{}, err
		}
		if p.Standard || p.Goroot {
			continue
		}
		if gomodcache != "" && isUnderDir(p.Dir, gomodcache) {
			continue
		}
		deps.pkgDirs[p.Dir] = struct{}{}
		all := make([]string, 0, len(p.GoFiles)+len(p.CgoFiles)+len(p.EmbedFiles))
		all = append(all, p.GoFiles...)
		all = append(all, p.CgoFiles...)
		all = append(all, p.EmbedFiles...)
		for _, name := range all {
			deps.files[filepath.Join(p.Dir, name)] = struct{}{}
		}
	}
	return deps, nil
}

func (r *Runtime) resolveGoModCache(env []string) string {
	if r.gomodcacheOverride != "" {
		return r.gomodcacheOverride
	}
	r.gomodcacheMut.Lock()
	defer r.gomodcacheMut.Unlock()
	if r.gomodcacheResolved {
		return r.gomodcache
	}
	cmd := exec.Command("go", "env", "GOMODCACHE")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		slog.Warn("failed to resolve GOMODCACHE; will retry on next build", "err", err)
		return ""
	}
	r.gomodcache = strings.TrimSpace(string(out))
	r.gomodcacheResolved = true
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
