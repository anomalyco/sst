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

type Runtime struct {
	version     string
	contexts    sync.Map
	results     sync.Map
	concurrency *semaphore.Weighted
}

func New(version string) *Runtime {
	weight := int64(4)
	if flag.SST_BUILD_CONCURRENCY_FUNCTION != "" {
		weight, _ = strconv.ParseInt(flag.SST_BUILD_CONCURRENCY_FUNCTION, 10, 64)
	} else if flag.SST_BUILD_CONCURRENCY != "" {
		weight, _ = strconv.ParseInt(flag.SST_BUILD_CONCURRENCY, 10, 64)
	}

	return &Runtime{
		contexts:    sync.Map{},
		results:     sync.Map{},
		version:     version,
		concurrency: semaphore.NewWeighted(weight),
	}
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
	Install      []string          `json:"install"`
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

func (r *Runtime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	cmd := process.Command(
		"node",
		"--enable-source-maps",
		filepath.Join(
			path.ResolvePlatformDir(input.CfgPath),
			"/dist/nodejs-runtime/index.js",
		),
		filepath.Join(input.Build.Out, input.Build.Handler),
		input.WorkerID,
	)
	cmd.Env = input.Env
	cmd.Env = append(cmd.Env, "NODE_OPTIONS="+os.Getenv("NODE_OPTIONS"))
	cmd.Env = append(cmd.Env, "VSCODE_INSPECTOR_OPTIONS="+os.Getenv("VSCODE_INSPECTOR_OPTIONS"))
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
	result, ok := r.results.Load(functionID)
	if !ok {
		return false
	}

	var meta = map[string]interface{}{}
	err := json.Unmarshal([]byte(result.(esbuild.BuildResult).Metafile), &meta)
	if err != nil {
		return false
	}
	for key := range meta["inputs"].(map[string]interface{}) {
		absPath, err := filepath.Abs(key)
		if err != nil {
			continue
		}
		if absPath == file {
			return true
		}
	}

	return false
}
