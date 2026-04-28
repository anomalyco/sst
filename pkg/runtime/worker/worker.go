package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/path"
	"github.com/sst/sst/v3/pkg/runtime"
	"github.com/sst/sst/v3/pkg/runtime/node"
)

type Runtime struct {
	contexts map[string]esbuild.BuildContext
	results  map[string]esbuild.BuildResult
	lock     sync.RWMutex
	unenv    map[string]*unenv
}

type Properties struct {
	AccountID     string              `json:"accountID"`
	ScriptName    string              `json:"scriptName"`
	Build         node.NodeProperties `json:"build"`
	Compatibility compatibility       `json:"compatibility"`
}

type compatibility struct {
	Date  string   `json:"date"`
	Flags []string `json:"flags"`
}

type unenv struct {
	Alias    map[string]string `json:"alias"`
	External []string          `json:"external"`
	Polyfill []string          `json:"polyfill"`
}

func New() *Runtime {
	return &Runtime{
		contexts: map[string]esbuild.BuildContext{},
		results:  map[string]esbuild.BuildResult{},
		lock:     sync.RWMutex{},
		unenv:    map[string]*unenv{},
	}
}

func (w *Runtime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	var properties Properties
	json.Unmarshal(input.Properties, &properties)
	build := properties.Build
	unenv, err := w.getUnenv(
		ctx,
		input.CfgPath,
		properties.Compatibility,
	)
	if err != nil {
		return nil, err
	}

	abs, err := filepath.Abs(input.Handler)
	if err != nil {
		return nil, err
	}
	// Windows paths must not be embedded raw in JS string literals: backslashes
	// are escape sequences (\p, \f, etc.). Forward slashes work for imports.
	importPath := filepath.ToSlash(abs)
	target := filepath.Join(input.Out(), input.Handler)

	slog.Info("loader info", "loader", build.Loader)

	loader := map[string]esbuild.Loader{
		".wasm": esbuild.LoaderBinary,
	}
	for key, value := range build.Loader {
		mapped, ok := node.LoaderMap[value]
		if !ok {
			continue
		}
		loader[key] = mapped
	}

	slog.Debug("esbuild options",
		"target", build.ESBuild.Target,
		"sourcemap", strings.Trim(string(build.ESBuild.Sourcemap), "\""),
		"keepNames", build.ESBuild.KeepNames != nil && *build.ESBuild.KeepNames,
		"define", build.ESBuild.Define,
		"banner", build.ESBuild.Banner,
		"external", build.ESBuild.External,
		"mainFields", build.ESBuild.MainFields,
		"conditions", build.ESBuild.Conditions,
	)
	external := uniqueStrings(
		[]string{"node:*", "cloudflare:workers"},
		unenv.External,
		build.ESBuild.External,
	)
	alias := stripAliasedExternals(unenv.Alias, external)
	options := esbuild.BuildOptions{
		Platform: esbuild.PlatformNode,
		Stdin: &esbuild.StdinOptions{
			Contents: fmt.Sprintf(`
      import * as _sst_user_module from "%s"
      import { wrapCloudflareHandler } from "sst/resource/cloudflare"
      export * from "%s"
      export default wrapCloudflareHandler(_sst_user_module.default)
      `, importPath, importPath),
			ResolveDir: filepath.Dir(abs),
			Loader:     esbuild.LoaderTS,
		},
		NodePaths: append([]string{
			filepath.Join(path.ResolvePlatformDir(input.CfgPath), "node_modules"),
		}, build.ESBuild.NodePaths...),
		Alias:             alias,
		Inject:            unenv.Polyfill,
		External:          external,
		Conditions:        build.ESBuild.ResolveConditions([]string{"workerd", "worker", "browser"}),
		Sourcemap:         build.ESBuild.ResolveSourcemap(esbuild.SourceMapNone),
		Loader:            loader,
		KeepNames:         build.ESBuild.ResolveKeepNames(true),
		Bundle:            true,
		Define:            build.ESBuild.Define,
		Splitting:         build.Splitting,
		Metafile:          true,
		Write:             true,
		Outfile:           target,
		MinifyWhitespace:  build.Minify,
		MinifySyntax:      build.Minify,
		MinifyIdentifiers: build.Minify,
		Target:            build.ESBuild.ResolveTarget(esbuild.ESNext),
		Format:            esbuild.FormatESModule,
		MainFields:        build.ESBuild.ResolveMainFields([]string{"module", "main"}),
		Banner: map[string]string{
			"js": func() string {
				defaultBanner := strings.Join([]string{
					`import { createRequire as topLevelCreateRequire } from 'module';`,
					`const require = topLevelCreateRequire("/");`,
				}, "\n")

				if banner, ok := build.ESBuild.Banner["js"]; ok {
					return banner + "\n" + defaultBanner
				}
				return defaultBanner
			}(),
		},
	}

	slog.Debug("esbuild resolved options",
		"target", options.Target,
		"sourcemap", options.Sourcemap,
		"keepNames", options.KeepNames,
		"define", options.Define,
		"external", options.External,
		"mainFields", options.MainFields,
		"conditions", options.Conditions,
	)
	contextKey := buildContextKey(input)
	w.lock.RLock()
	buildContext, ok := w.contexts[contextKey]
	w.lock.RUnlock()
	if !ok {
		prefix := input.FunctionID + "\x00"
		w.lock.Lock()
		for key, context := range w.contexts {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			context.Dispose()
			delete(w.contexts, key)
		}
		w.lock.Unlock()
		buildContext, _ = esbuild.Context(options)
		w.lock.Lock()
		w.contexts[contextKey] = buildContext
		w.lock.Unlock()
	}

	result := buildContext.Rebuild()
	if len(result.Errors) == 0 {
		w.lock.Lock()
		w.results[input.FunctionID] = result
		w.lock.Unlock()
	}
	errors := []string{}
	for _, error := range result.Errors {
		errors = append(errors, error.Text)
	}

	for _, error := range result.Errors {
		slog.Error("esbuild error", "error", error)
	}
	for _, warning := range result.Warnings {
		slog.Error("esbuild error", "error", warning)
	}

	return &runtime.BuildOutput{
		Handler: input.Handler,
		Errors:  errors,
	}, nil
}

func buildContextKey(input *runtime.BuildInput) string {
	return input.FunctionID + "\x00" + input.Handler + "\x00" + string(input.Properties)
}

func uniqueStrings(groups ...[]string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, group := range groups {
		for _, item := range group {
			if seen[item] {
				continue
			}
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func stripAliasedExternals(alias map[string]string, external []string) map[string]string {
	result := map[string]string{}
	maps.Copy(result, alias)
	for _, item := range external {
		delete(result, item)
	}
	return result
}

func (w *Runtime) getUnenv(ctx context.Context, cfgPath string, compatibility compatibility) (*unenv, error) {
	payload, err := json.Marshal(compatibility)
	if err != nil {
		return nil, err
	}
	key := string(payload)

	w.lock.RLock()
	if cached, ok := w.unenv[key]; ok {
		w.lock.RUnlock()
		return cached, nil
	}
	w.lock.RUnlock()

	cmd := process.CommandContext(
		ctx,
		"node",
		filepath.Join(path.ResolvePlatformDir(cfgPath), "src/runtime/worker/unenv.mjs"),
		string(payload),
	)
	cmd.Dir = path.ResolvePlatformDir(cfgPath)
	cmd.Env = []string{}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to load cloudflare unenv config: %w\n%s", err, output)
	}

	var result unenv
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to decode cloudflare unenv config: %w\n%s", err, output)
	}

	w.lock.Lock()
	w.unenv[key] = &result
	w.lock.Unlock()

	return &result, nil
}

func (w *Runtime) Match(runtime string) bool {
	return runtime == "worker"
}

func (w *Runtime) getFile(input *runtime.BuildInput) (string, bool) {
	dir := filepath.Dir(input.Handler)
	base := strings.Split(filepath.Base(input.Handler), ".")[0]
	for _, ext := range node.NODE_EXTENSIONS {
		file := filepath.Join(path.ResolveRootDir(input.CfgPath), dir, base+ext)
		if _, err := os.Stat(file); err == nil {
			return file, true
		}
	}
	return "", false
}

func (r *Runtime) ShouldRebuild(functionID string, file string) bool {
	r.lock.RLock()
	result, ok := r.results[functionID]
	r.lock.RUnlock()
	if !ok {
		return false
	}

	var meta = map[string]interface{}{}
	err := json.Unmarshal([]byte(result.Metafile), &meta)
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

func (r *Runtime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	return nil, fmt.Errorf("not implemented")
}

var NODE_BUILTINS = map[string]bool{
	"assert":              true,
	"async_hooks":         true,
	"buffer":              true,
	"child_process":       true,
	"cluster":             true,
	"console":             true,
	"constants":           true,
	"crypto":              true,
	"dgram":               true,
	"diagnostics_channel": true,
	"dns":                 true,
	"domain":              true,
	"events":              true,
	"fs":                  true,
	"http":                true,
	"http2":               true,
	"https":               true,
	"inspector":           true,
	"module":              true,
	"net":                 true,
	"os":                  true,
	"path":                true,
	"perf_hooks":          true,
	"process":             true,
	"punycode":            true,
	"querystring":         true,
	"readline":            true,
	"repl":                true,
	"stream":              true,
	"string_decoder":      true,
	"sys":                 true,
	"timers":              true,
	"tls":                 true,
	"trace_events":        true,
	"tty":                 true,
	"url":                 true,
	"util":                true,
	"v8":                  true,
	"vm":                  true,
	"wasi":                true,
	"worker_threads":      true,
	"zlib":                true,
}
