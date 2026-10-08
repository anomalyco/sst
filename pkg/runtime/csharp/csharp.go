package csharp

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/runtime"
)

// Runtime builds and runs .NET Lambda functions.
//
// It targets the managed Lambda runtimes (`dotnet8`, `dotnet10`) and expects the
// handler to point at a project directory, a `.csproj` file, or a single-file
// app (`app.cs`, .NET 10 SDK and later). Two project styles are supported:
//
//   - Executable assemblies (OutputType Exe using Amazon.Lambda.RuntimeSupport).
//     The handler is just the path, e.g. `./src` or `./src/Api.cs`, and the
//     Lambda handler string becomes the assembly name.
//   - Class libraries. The handler is `{path}::{Namespace.Class}::{Method}` and the
//     Lambda handler string becomes `{Assembly}::{Namespace.Class}::{Method}`.
//
// Live dev (`sst dev`) is only available for executable assemblies since those
// can be started directly and speak the Lambda Runtime API on their own.
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
	return strings.HasPrefix(runtime, "dotnet")
}

type Properties struct {
	Architecture string `json:"architecture"`
}

// Handler is the parsed form of the `handler` string.
type Handler struct {
	// Path is the project directory, .csproj file or file-based .cs app the user pointed at.
	Path string
	// Type is the fully qualified class name for class library handlers.
	Type string
	// Method is the method name for class library handlers.
	Method string
}

// IsExecutable reports whether the handler refers to an executable assembly
// rather than a class library method.
func (h Handler) IsExecutable() bool {
	return h.Type == ""
}

// ParseHandler splits the handler into its path and optional `::Type::Method`
// suffix.
func ParseHandler(handler string) (Handler, error) {
	parts := strings.Split(handler, "::")
	switch len(parts) {
	case 1:
		return Handler{Path: parts[0]}, nil
	case 3:
		if parts[1] == "" || parts[2] == "" {
			return Handler{}, fmt.Errorf("invalid .NET handler %q: expected {path}::{Namespace.Class}::{Method}", handler)
		}
		return Handler{Path: parts[0], Type: parts[1], Method: parts[2]}, nil
	default:
		return Handler{}, fmt.Errorf("invalid .NET handler %q: expected {path} or {path}::{Namespace.Class}::{Method}", handler)
	}
}

// ResolveProject locates what `dotnet publish` should build for the handler
// path. The path can be a `.csproj`, a single-file `.cs` app, or a directory
// containing exactly one project file.
func ResolveProject(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("handler path %q not found: %w", path, err)
	}
	if !info.IsDir() {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".csproj", ".cs":
			return path, nil
		}
		return "", fmt.Errorf("handler path %q must be a directory, a .csproj file or a file-based .cs app", path)
	}
	matches, err := filepath.Glob(filepath.Join(path, "*.csproj"))
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no .csproj found in %q", path)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple .csproj files found in %q, point the handler at one of them", path)
	}
}

// Project is the subset of build settings the runtime needs from a `.csproj`
// or from the `#:` directives of a file-based app.
type Project struct {
	// FileBased is true for single-file apps built directly from a `.cs` file.
	FileBased    bool
	sdk          string
	assemblyName string
	outputType   string
}

type csproj struct {
	Sdk  string `xml:"Sdk,attr"`
	Sdks []struct {
		Name string `xml:"Name,attr"`
	} `xml:"Sdk"`
	PropertyGroups []struct {
		AssemblyName string `xml:"AssemblyName"`
		OutputType   string `xml:"OutputType"`
	} `xml:"PropertyGroup"`
}

// executableSdks build an executable unless <OutputType> says otherwise.
var executableSdks = map[string]bool{
	"microsoft.net.sdk.web":    true,
	"microsoft.net.sdk.worker": true,
}

// ReadProject parses the parts of a project we care about.
func ReadProject(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(filepath.Ext(path), ".cs") {
		return readFileBasedApp(data), nil
	}
	var parsed csproj
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	project := &Project{sdk: parsed.Sdk}
	for _, sdk := range parsed.Sdks {
		if project.sdk == "" {
			project.sdk = sdk.Name
		}
	}
	for _, group := range parsed.PropertyGroups {
		if project.assemblyName == "" {
			project.assemblyName = strings.TrimSpace(group.AssemblyName)
		}
		if project.outputType == "" {
			project.outputType = strings.TrimSpace(group.OutputType)
		}
	}
	return project, nil
}

// readFileBasedApp picks up `#:property Name=Value` directives. File-based apps
// are always executables.
func readFileBasedApp(data []byte) *Project {
	project := &Project{FileBased: true}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#:property") {
			continue
		}
		kv := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(line, "#:property")), "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(kv[0]), "AssemblyName") {
			project.assemblyName = strings.TrimSpace(kv[1])
		}
	}
	return project
}

// AssemblyName returns the explicit AssemblyName or falls back to the project
// or source file name, mirroring MSBuild's default.
func (p *Project) AssemblyName(path string) string {
	if p.assemblyName != "" {
		return p.assemblyName
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// IsExecutable reports whether the project builds an executable. MSBuild
// defaults to a class library when <OutputType> is not set, except for the Web
// and Worker SDKs which default to an executable.
func (p *Project) IsExecutable() bool {
	if p.FileBased {
		return true
	}
	switch strings.ToLower(p.outputType) {
	case "exe", "winexe":
		return true
	case "":
		// The Sdk attribute may carry a version, e.g. "Microsoft.NET.Sdk.Web/1.0".
		sdk, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(p.sdk)), "/")
		return executableSdks[sdk]
	}
	return false
}

// PublishArgs builds the `dotnet publish` arguments. Dev builds are Debug builds
// for the host machine. Deploy builds are framework-dependent Release builds for
// the Lambda architecture so the managed runtime can load them. Native AOT is
// switched off in both cases: the managed runtimes can't run AOT output, and
// file-based apps enable it by default.
func PublishArgs(project string, out string, dev bool, architecture string) []string {
	args := []string{"publish", project, "--nologo", "-o", out, "-p:PublishAot=false"}
	if dev {
		return append(args, "-c", "Debug")
	}
	rid := "linux-x64"
	if architecture == "arm64" {
		rid = "linux-arm64"
	}
	return append(args,
		"-c", "Release",
		"-r", rid,
		"--self-contained", "false",
		"-p:GenerateRuntimeConfigurationFiles=true",
		// The Lambda handler is the assembly name, so the native apphost is dead weight.
		"-p:UseAppHost=false",
	)
}

func (r *Runtime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	r.mut.Lock()
	defer r.mut.Unlock()
	var properties Properties
	json.Unmarshal(input.Properties, &properties)

	handler, err := ParseHandler(input.Handler)
	if err != nil {
		return nil, err
	}
	projectPath, err := ResolveProject(handler.Path)
	if err != nil {
		return nil, err
	}
	// The handler is relative to the app root but publish runs from the project
	// directory, so resolve it first.
	projectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}
	project, err := ReadProject(projectPath)
	if err != nil {
		return nil, err
	}
	assembly := project.AssemblyName(projectPath)

	if handler.IsExecutable() && !project.IsExecutable() {
		return &runtime.BuildOutput{
			Errors: []string{fmt.Sprintf(
				"%s is a class library. Either set <OutputType>Exe</OutputType> and use Amazon.Lambda.RuntimeSupport, or point the handler at a method: \"%s::{Namespace.Class}::{Method}\"",
				filepath.Base(projectPath), input.Handler,
			)},
		}, nil
	}
	if !handler.IsExecutable() && input.Dev {
		return &runtime.BuildOutput{
			Errors: []string{fmt.Sprintf(
				"sst dev needs an executable assembly to run %s locally. Set <OutputType>Exe</OutputType>, bootstrap it with Amazon.Lambda.RuntimeSupport and set the handler to \"%s\"",
				filepath.Base(projectPath), handler.Path,
			)},
		}, nil
	}

	root := filepath.Dir(projectPath)
	dotnet, err := exec.LookPath("dotnet")
	if err != nil {
		return &runtime.BuildOutput{
			Errors: []string{
				"dotnet was not found on your PATH. Install the .NET SDK from https://dotnet.microsoft.com/download (10 or later for file-based apps) and restart sst.",
			},
		}, nil
	}
	if project.FileBased {
		if major, version, err := sdkVersion(dotnet, root); err == nil && major < 10 {
			return &runtime.BuildOutput{
				Errors: []string{fmt.Sprintf(
					"%s is a file-based app, which needs the .NET 10 SDK or later. `dotnet --version` reports %s.",
					filepath.Base(projectPath), version,
				)},
			}, nil
		}
	}
	cmd := process.Command(dotnet, PublishArgs(projectPath, input.Out(), input.Dev, properties.Architecture)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"DOTNET_CLI_TELEMETRY_OPTOUT=1",
		"DOTNET_NOLOGO=1",
		"DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1",
	)
	slog.Info("running dotnet publish", "cmd", cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return &runtime.BuildOutput{
			Errors: []string{message},
		}, nil
	}
	r.directories[input.FunctionID], _ = filepath.Abs(root)

	lambdaHandler := assembly
	if !handler.IsExecutable() {
		lambdaHandler = assembly + "::" + handler.Type + "::" + handler.Method
	}
	return &runtime.BuildOutput{
		Handler:    lambdaHandler,
		Sourcemaps: []string{},
		Errors:     []string{},
		Out:        root,
	}, nil
}

// sdkVersion reports the SDK that `dotnet --version` resolves to from the given
// directory, which honors any global.json on the way up.
func sdkVersion(dotnet string, dir string) (int, string, error) {
	cmd := process.Command(dotnet, "--version")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DOTNET_CLI_TELEMETRY_OPTOUT=1", "DOTNET_NOLOGO=1")
	output, err := cmd.Output()
	if err != nil {
		return 0, "", err
	}
	return parseMajor(strings.TrimSpace(string(output)))
}

// parseMajor extracts the major version from a version string like "10.0.401".
func parseMajor(version string) (int, string, error) {
	var major int
	if _, err := fmt.Sscanf(version, "%d.", &major); err != nil {
		return 0, version, fmt.Errorf("unexpected dotnet version %q", version)
	}
	return major, version, nil
}

func (r *Runtime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	if strings.Contains(input.Build.Handler, "::") {
		return nil, fmt.Errorf("sst dev only supports executable .NET assemblies, got handler %q", input.Build.Handler)
	}
	cmd := process.Command(
		"dotnet",
		filepath.Join(input.Build.Out, input.Build.Handler+".dll"),
	)
	slog.Info("running dotnet", "server", input.Server, "cmd", cmd.Args)
	cmd.Env = input.Env
	// Amazon.Lambda.RuntimeSupport prepends http:// itself.
	cmd.Env = append(cmd.Env, "AWS_LAMBDA_RUNTIME_API="+input.Server)
	cmd.Env = append(cmd.Env, "DOTNET_CLI_TELEMETRY_OPTOUT=1")
	cmd.Dir = input.Build.Out
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Worker{
		stdout,
		stderr,
		cmd,
	}, nil
}

var rebuildExtensions = map[string]bool{
	".cs":      true,
	".csproj":  true,
	".props":   true,
	".targets": true,
}

func (r *Runtime) ShouldRebuild(functionID string, file string) bool {
	if !rebuildExtensions[strings.ToLower(filepath.Ext(file))] {
		return false
	}
	match, ok := r.directories[functionID]
	if !ok {
		return false
	}
	rel, err := filepath.Rel(match, file)
	if err != nil || strings.HasPrefix(rel, "..") {
		return false
	}
	// MSBuild writes generated .cs files and publish output under obj/ and bin/.
	// Rebuilding on those would loop forever.
	for _, segment := range strings.Split(filepath.ToSlash(rel), "/") {
		if segment == "bin" || segment == "obj" {
			return false
		}
	}
	slog.Info("checking if file needs to be rebuilt", "file", file, "match", match)
	return true
}

// ShouldRunEagerly returns true for .NET - workers restart immediately after
// rebuild. Like Go and Rust, rebuilds are scoped to the project directory that
// changed.
func (r *Runtime) ShouldRunEagerly() bool {
	return true
}
