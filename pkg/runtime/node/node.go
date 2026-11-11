package node

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/sst/sst/v3/pkg/flag"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/path"
	"github.com/sst/sst/v3/pkg/runtime"
	"golang.org/x/sync/semaphore"
)

var LoaderMap = map[string]esbuild.Loader{
	"js":      esbuild.LoaderJS,
	"jsx":     esbuild.LoaderJSX,
	"ts":      esbuild.LoaderTS,
	"tsx":     esbuild.LoaderTSX,
	"css":     esbuild.LoaderCSS,
	"json":    esbuild.LoaderJSON,
	"text":    esbuild.LoaderText,
	"base64":  esbuild.LoaderBase64,
	"file":    esbuild.LoaderFile,
	"dataurl": esbuild.LoaderDataURL,
	"binary":  esbuild.LoaderBinary,
}

var LoaderToString = []string{
	"none",
	"base64",
	"binary",
	"copy",
	"css",
	"dataurl",
	"default",
	"empty",
	"file",
	"global-css",
	"js",
	"json",
	"json",
	"jsx",
	"local-css",
	"text",
	"ts",
	"ts",
	"tsx",
}

// DefaultContextCap is the default number of esbuild build contexts kept
// alive in dev mode. Each context caches the parsed dependency graph of one
// function, which can retain hundreds of MB for large graphs, so keeping one
// per function grows memory without bound as functions are built.
const DefaultContextCap = 4

type buildContextEntry struct {
	context  esbuild.BuildContext
	lastUsed int64
	building int
}

type Runtime struct {
	version      string
	concurrency  *semaphore.Weighted
	contextCap   int
	contextsLock sync.Mutex
	contexts     map[string]*buildContextEntry
	contextClock int64
	inputs       map[string]map[string]bool
}

func New(version string) *Runtime {
	weight := int64(4)
	if flag.SST_BUILD_CONCURRENCY_FUNCTION != "" {
		weight, _ = strconv.ParseInt(flag.SST_BUILD_CONCURRENCY_FUNCTION, 10, 64)
	} else if flag.SST_BUILD_CONCURRENCY != "" {
		weight, _ = strconv.ParseInt(flag.SST_BUILD_CONCURRENCY, 10, 64)
	}

	contextCap := DefaultContextCap
	if flag.SST_BUILD_CONTEXT_CACHE != "" {
		if parsed, err := strconv.Atoi(flag.SST_BUILD_CONTEXT_CACHE); err == nil && parsed >= 0 {
			contextCap = parsed
		}
	}

	return &Runtime{
		version:     version,
		concurrency: semaphore.NewWeighted(weight),
		contextCap:  contextCap,
		contexts:    map[string]*buildContextEntry{},
		inputs:      map[string]map[string]bool{},
	}
}

// acquireContext returns the cached build context for a function, creating
// one if needed. The entry is pinned until releaseContext is called so it
// cannot be evicted mid-build.
func (r *Runtime) acquireContext(functionID string, options esbuild.BuildOptions) (esbuild.BuildContext, error) {
	r.contextsLock.Lock()
	if entry, ok := r.contexts[functionID]; ok {
		entry.building++
		r.contextClock++
		entry.lastUsed = r.contextClock
		r.contextsLock.Unlock()
		return entry.context, nil
	}
	r.contextsLock.Unlock()

	// create outside the lock; plugin setup can spawn subprocesses
	created, ctxErr := esbuild.Context(options)
	if ctxErr != nil {
		return nil, ctxErr
	}

	r.contextsLock.Lock()
	if entry, ok := r.contexts[functionID]; ok {
		// lost a race with a concurrent build of the same function
		entry.building++
		r.contextClock++
		entry.lastUsed = r.contextClock
		r.contextsLock.Unlock()
		created.Dispose()
		return entry.context, nil
	}
	r.contextClock++
	r.contexts[functionID] = &buildContextEntry{
		context:  created,
		lastUsed: r.contextClock,
		building: 1,
	}
	r.contextsLock.Unlock()
	return created, nil
}

// releaseContext unpins a function's build context and evicts the least
// recently used contexts beyond the configured cap.
func (r *Runtime) releaseContext(functionID string) {
	var evicted []esbuild.BuildContext
	r.contextsLock.Lock()
	if entry, ok := r.contexts[functionID]; ok {
		entry.building--
	}
	for len(r.contexts) > r.contextCap {
		oldestID := ""
		var oldest *buildContextEntry
		for id, entry := range r.contexts {
			if entry.building > 0 {
				continue
			}
			if oldest == nil || entry.lastUsed < oldest.lastUsed {
				oldestID = id
				oldest = entry
			}
		}
		if oldest == nil {
			break
		}
		delete(r.contexts, oldestID)
		evicted = append(evicted, oldest.context)
	}
	r.contextsLock.Unlock()
	for _, context := range evicted {
		context.Dispose()
	}
}

// storeInputs records the absolute paths of a build's inputs so ShouldRebuild
// can match changed files without retaining the full build result.
func (r *Runtime) storeInputs(functionID string, metafile string) {
	var meta struct {
		Inputs map[string]json.RawMessage `json:"inputs"`
	}
	if err := json.Unmarshal([]byte(metafile), &meta); err != nil {
		return
	}
	paths := make(map[string]bool, len(meta.Inputs))
	for key := range meta.Inputs {
		absPath, err := filepath.Abs(key)
		if err != nil {
			continue
		}
		paths[absPath] = true
	}
	r.contextsLock.Lock()
	r.inputs[functionID] = paths
	r.contextsLock.Unlock()
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

type NodeProperties struct {
	Loader       map[string]string `json:"loader"`
	Install      map[string]string `json:"install"`
	Banner       string            `json:"banner"`
	ESBuild      ESBuildOptions    `json:"esbuild"`
	Minify       bool              `json:"minify"`
	Format       string            `json:"format"`
	Target       string            `json:"target"`
	SourceMap    *bool             `json:"sourceMap"`
	Splitting    bool              `json:"splitting"`
	Plugins      string            `json:"plugins"`
	Architecture string            `json:"architecture"`
}

type ESBuildOptions struct {
	Target     string            `json:"target"`
	Sourcemap  json.RawMessage   `json:"sourcemap"`
	KeepNames  *bool             `json:"keepNames"`
	Define     map[string]string `json:"define"`
	Banner     map[string]string `json:"banner"`
	External   []string          `json:"external"`
	NodePaths  []string          `json:"nodePaths"`
	MainFields []string          `json:"mainFields"`
	Conditions []string          `json:"conditions"`
}

func (o *ESBuildOptions) ResolveTarget(fallback esbuild.Target) esbuild.Target {
	if t, ok := esTargetMap[strings.ToLower(o.Target)]; ok {
		return t
	}
	return fallback
}

func (o *ESBuildOptions) ResolveSourcemap(fallback esbuild.SourceMap) esbuild.SourceMap {
	if len(o.Sourcemap) == 0 {
		return fallback
	}
	var str string
	if json.Unmarshal(o.Sourcemap, &str) == nil {
		if s, ok := esSourcemapMap[strings.ToLower(str)]; ok {
			return s
		}
		return fallback
	}
	var b bool
	if json.Unmarshal(o.Sourcemap, &b) == nil {
		if b {
			return esbuild.SourceMapLinked
		}
		return esbuild.SourceMapNone
	}
	return fallback
}

func (o *ESBuildOptions) ResolveKeepNames(fallback bool) bool {
	if o.KeepNames != nil {
		return *o.KeepNames
	}
	return fallback
}

func (o *ESBuildOptions) ResolveMainFields(fallback []string) []string {
	if len(o.MainFields) > 0 {
		return o.MainFields
	}
	return fallback
}

func (o *ESBuildOptions) ResolveConditions(fallback []string) []string {
	if len(o.Conditions) > 0 {
		return o.Conditions
	}
	return fallback
}

var esTargetMap = map[string]esbuild.Target{
	"esnext": esbuild.ESNext,
	"es5":    esbuild.ES5,
	"es6":    esbuild.ES2015,
	"es2015": esbuild.ES2015,
	"es2016": esbuild.ES2016,
	"es2017": esbuild.ES2017,
	"es2018": esbuild.ES2018,
	"es2019": esbuild.ES2019,
	"es2020": esbuild.ES2020,
	"es2021": esbuild.ES2021,
	"es2022": esbuild.ES2022,
	"es2023": esbuild.ES2023,
}

var esSourcemapMap = map[string]esbuild.SourceMap{
	"inline":   esbuild.SourceMapInline,
	"linked":   esbuild.SourceMapLinked,
	"external": esbuild.SourceMapExternal,
	"both":     esbuild.SourceMapInlineAndExternal,
}

var NODE_EXTENSIONS = []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs"}

var EDITOR_ENV = []string{
	"VSCODE_INSPECTOR_OPTIONS",
	"JB_IDE_HOST",
	"JB_IDE_PORT",
	"JB_INTERPRETER",
	"JB_NODE_DEBUG_CONNECTION_GATEWAY_HOST",
	"JB_NODE_DEBUG_CONNECTION_GATEWAY_PORT",
	"JETBRAINS_NODE_BIND_HOST",
	"JETBRAINS_NODE_DEBUGGER_ATTACH_TO_HELPERS",
	"JETBRAINS_NODE_DEBUGGER_VERBOSE_LOGGING",
}

func (r *Runtime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	cmd := process.Command(
		"node",
		"--enable-source-maps",
		"--no-warnings",
		filepath.Join(
			path.ResolvePlatformDir(input.CfgPath),
			"/dist/nodejs-runtime/index.js",
		),
		filepath.Join(input.Build.Out, input.Build.Handler),
		input.WorkerID,
	)
	cmd.Env = input.Env
	cmd.Env = append(cmd.Env, "NODE_OPTIONS="+os.Getenv("NODE_OPTIONS"))
	for _, key := range EDITOR_ENV {
		if value := os.Getenv(key); value != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	cmd.Env = append(cmd.Env, "AWS_LAMBDA_RUNTIME_API="+input.Server)
	slog.Info("starting worker", "server", input.Server)
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

func (r *Runtime) Match(runtime string) bool {
	return strings.HasPrefix(runtime, "node")
}

// ShouldRunEagerly returns true for Node.js - workers restart immediately after rebuild.
// Node.js uses esbuild's metafile for precise per-function dependency tracking,
// so only functions that actually import a changed file will rebuild.
func (r *Runtime) ShouldRunEagerly() bool {
	return true
}

func (r *Runtime) getFile(input *runtime.BuildInput) (string, bool) {
	dir := filepath.Dir(input.Handler)
	fileSplit := strings.Split(filepath.Base(input.Handler), ".")
	base := strings.Join(fileSplit[:len(fileSplit)-1], ".")
	for _, ext := range NODE_EXTENSIONS {
		file := filepath.Join(dir, base+ext)
		if !filepath.IsAbs(file) {
			file = filepath.Join(path.ResolveRootDir(input.CfgPath), file)
		}
		if _, err := os.Stat(file); err == nil {
			return file, true
		}
	}
	return "", false
}

func (r *Runtime) ShouldRebuild(functionID string, file string) bool {
	r.contextsLock.Lock()
	defer r.contextsLock.Unlock()
	paths, ok := r.inputs[functionID]
	if !ok {
		return false
	}
	return paths[file]
}
