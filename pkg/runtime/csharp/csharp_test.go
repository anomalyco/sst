package csharp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

func TestMatch(t *testing.T) {
	r := New()
	for _, name := range []string{"dotnet8", "dotnet10"} {
		if !r.Match(name) {
			t.Errorf("expected %q to match", name)
		}
	}
	for _, name := range []string{"nodejs24.x", "go", "rust", "python3.13", "provided.al2023"} {
		if r.Match(name) {
			t.Errorf("expected %q not to match", name)
		}
	}
}

func TestParseHandler(t *testing.T) {
	h, err := ParseHandler("./src")
	if err != nil {
		t.Fatal(err)
	}
	if h.Path != "./src" || !h.IsExecutable() {
		t.Errorf("unexpected handler %+v", h)
	}

	h, err = ParseHandler("./src::My.Functions.Handler::Handle")
	if err != nil {
		t.Fatal(err)
	}
	if h.Path != "./src" || h.Type != "My.Functions.Handler" || h.Method != "Handle" || h.IsExecutable() {
		t.Errorf("unexpected handler %+v", h)
	}

	for _, bad := range []string{"./src::Type", "./src::Type::Method::Extra", "./src::::Method"} {
		if _, err := ParseHandler(bad); err == nil {
			t.Errorf("expected %q to fail", bad)
		}
	}
}

func writeProject(t *testing.T, dir string, name string, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveProject(t *testing.T) {
	root := t.TempDir()

	single := filepath.Join(root, "single")
	want := writeProject(t, single, "Api.csproj", "<Project/>")
	got, err := ResolveProject(single)
	if err != nil || got != want {
		t.Errorf("ResolveProject(dir) = %q, %v; want %q", got, err, want)
	}
	got, err = ResolveProject(want)
	if err != nil || got != want {
		t.Errorf("ResolveProject(file) = %q, %v; want %q", got, err, want)
	}

	if _, err := ResolveProject(filepath.Join(root, "empty-"+t.Name())); err == nil {
		t.Error("expected missing path to fail")
	}
	empty := filepath.Join(root, "empty")
	os.MkdirAll(empty, 0755)
	if _, err := ResolveProject(empty); err == nil {
		t.Error("expected directory without csproj to fail")
	}

	multi := filepath.Join(root, "multi")
	writeProject(t, multi, "A.csproj", "<Project/>")
	writeProject(t, multi, "B.csproj", "<Project/>")
	if _, err := ResolveProject(multi); err == nil {
		t.Error("expected directory with two csproj files to fail")
	}

	app := writeProject(t, filepath.Join(root, "app"), "Api.cs", "Console.WriteLine();")
	got, err = ResolveProject(app)
	if err != nil || got != app {
		t.Errorf("ResolveProject(file-based app) = %q, %v; want %q", got, err, app)
	}

	other := writeProject(t, filepath.Join(root, "other"), "README.md", "")
	if _, err := ResolveProject(other); err == nil {
		t.Error("expected non-project file to fail")
	}
}

func TestReadProject(t *testing.T) {
	root := t.TempDir()

	exe := writeProject(t, root, "MyFunction.csproj", `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>`)
	p, err := ReadProject(exe)
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsExecutable() {
		t.Error("expected Exe project to be executable")
	}
	if got := p.AssemblyName(exe); got != "MyFunction" {
		t.Errorf("AssemblyName = %q, want MyFunction", got)
	}

	lib := writeProject(t, root, "Lib.csproj", `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <PropertyGroup Condition="'$(Configuration)' == 'Release'">
    <AssemblyName>Custom.Assembly</AssemblyName>
  </PropertyGroup>
</Project>`)
	p, err = ReadProject(lib)
	if err != nil {
		t.Fatal(err)
	}
	if p.IsExecutable() {
		t.Error("expected project without OutputType to be a class library")
	}
	if got := p.AssemblyName(lib); got != "Custom.Assembly" {
		t.Errorf("AssemblyName = %q, want Custom.Assembly", got)
	}

	web := writeProject(t, root, "Web.csproj", `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net10.0</TargetFramework>
  </PropertyGroup>
</Project>`)
	p, err = ReadProject(web)
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsExecutable() {
		t.Error("expected Web SDK project without OutputType to be executable")
	}

	webLib := writeProject(t, root, "WebLib.csproj", `<Project Sdk="Microsoft.NET.Sdk.Web/1.0">
  <PropertyGroup>
    <OutputType>Library</OutputType>
  </PropertyGroup>
</Project>`)
	p, err = ReadProject(webLib)
	if err != nil {
		t.Fatal(err)
	}
	if p.IsExecutable() {
		t.Error("expected explicit Library OutputType to win over the Web SDK default")
	}

	worker := writeProject(t, root, "Worker.csproj", `<Project>
  <Sdk Name="Microsoft.NET.Sdk.Worker" />
  <PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup>
</Project>`)
	p, err = ReadProject(worker)
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsExecutable() {
		t.Error("expected Worker SDK element project to be executable")
	}

	broken := writeProject(t, root, "Broken.csproj", "<Project><PropertyGroup>")
	if _, err := ReadProject(broken); err == nil {
		t.Error("expected malformed csproj to fail")
	}

	app := writeProject(t, root, "Api.cs", `#:property PublishAot=false
#:package Amazon.Lambda.RuntimeSupport@2.2.0

Console.WriteLine("hi");
`)
	p, err = ReadProject(app)
	if err != nil {
		t.Fatal(err)
	}
	if !p.FileBased || !p.IsExecutable() {
		t.Error("expected file-based app to be an executable")
	}
	if got := p.AssemblyName(app); got != "Api" {
		t.Errorf("AssemblyName = %q, want Api", got)
	}

	named := writeProject(t, root, "tool.cs", `#:property   AssemblyName = MyTool
#:property TargetFramework=net10.0
`)
	p, err = ReadProject(named)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.AssemblyName(named); got != "MyTool" {
		t.Errorf("AssemblyName = %q, want MyTool", got)
	}
}

func TestPublishArgs(t *testing.T) {
	dev := strings.Join(PublishArgs("a.csproj", "/out", true, "x86_64"), " ")
	if !strings.Contains(dev, "-c Debug") || strings.Contains(dev, "-r ") || !strings.Contains(dev, "PublishAot=false") {
		t.Errorf("unexpected dev args: %s", dev)
	}

	x64 := strings.Join(PublishArgs("a.csproj", "/out", false, "x86_64"), " ")
	for _, want := range []string{"publish a.csproj", "-o /out", "-c Release", "-r linux-x64", "--self-contained false", "GenerateRuntimeConfigurationFiles=true", "UseAppHost=false", "PublishAot=false"} {
		if !strings.Contains(x64, want) {
			t.Errorf("expected %q in %s", want, x64)
		}
	}

	arm := strings.Join(PublishArgs("a.csproj", "/out", false, "arm64"), " ")
	if !strings.Contains(arm, "-r linux-arm64") {
		t.Errorf("unexpected arm64 args: %s", arm)
	}
}

func TestShouldRebuild(t *testing.T) {
	r := New()
	root := filepath.Join(t.TempDir(), "src")
	r.directories["fn"] = root

	yes := []string{
		filepath.Join(root, "Function.cs"),
		filepath.Join(root, "Nested", "Handler.cs"),
		filepath.Join(root, "MyFunction.csproj"),
		filepath.Join(root, "Directory.Build.props"),
	}
	for _, f := range yes {
		if !r.ShouldRebuild("fn", f) {
			t.Errorf("expected rebuild for %s", f)
		}
	}

	no := []string{
		filepath.Join(root, "obj", "Debug", "net8.0", "MyFunction.GlobalUsings.g.cs"),
		filepath.Join(root, "obj", "MyFunction.csproj.nuget.g.props"),
		filepath.Join(root, "bin", "Debug", "net8.0", "MyFunction.dll"),
		filepath.Join(root, "README.md"),
		filepath.Join(filepath.Dir(root), "other", "Function.cs"),
	}
	for _, f := range no {
		if r.ShouldRebuild("fn", f) {
			t.Errorf("expected no rebuild for %s", f)
		}
	}

	if r.ShouldRebuild("unknown", filepath.Join(root, "Function.cs")) {
		t.Error("expected no rebuild for unknown function")
	}
}

func TestBuildRejectsClassLibraryWithoutMethod(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src")
	writeProject(t, dir, "Lib.csproj", `<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>`)

	out, err := New().Build(context.Background(), &runtime.BuildInput{
		FunctionID: "fn",
		Handler:    dir,
		Runtime:    "dotnet8",
		CfgPath:    filepath.Join(root, "sst.config.ts"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Errors) != 1 || !strings.Contains(out.Errors[0], "class library") {
		t.Errorf("expected class library error, got %+v", out)
	}
}

func TestBuildRejectsClassLibraryInDev(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src")
	writeProject(t, dir, "Lib.csproj", `<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>`)

	out, err := New().Build(context.Background(), &runtime.BuildInput{
		FunctionID: "fn",
		Handler:    dir + "::My.Ns.Function::Handler",
		Runtime:    "dotnet8",
		Dev:        true,
		CfgPath:    filepath.Join(root, "sst.config.ts"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Errors) != 1 || !strings.Contains(out.Errors[0], "sst dev") {
		t.Errorf("expected dev mode error, got %+v", out)
	}
}

func TestBuildReportsMissingDotnet(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	dir := filepath.Join(root, "src")
	writeProject(t, dir, "App.csproj", `<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><OutputType>Exe</OutputType></PropertyGroup></Project>`)

	out, err := New().Build(context.Background(), &runtime.BuildInput{
		FunctionID: "fn",
		Handler:    dir,
		Runtime:    "dotnet10",
		CfgPath:    filepath.Join(root, "sst.config.ts"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Errors) != 1 || !strings.Contains(out.Errors[0], "dotnet was not found on your PATH") {
		t.Errorf("expected missing dotnet error, got %+v", out)
	}
}

func TestParseMajor(t *testing.T) {
	for version, want := range map[string]int{"10.0.401": 10, "8.0.425": 8, "9.0.100-preview.1": 9} {
		major, _, err := parseMajor(version)
		if err != nil || major != want {
			t.Errorf("parseMajor(%q) = %d, %v; want %d", version, major, err, want)
		}
	}
	if _, _, err := parseMajor("garbage"); err == nil {
		t.Error("expected garbage version to fail")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("could not find repo root")
		}
		dir = parent
	}
}

// requireDotnet skips the test unless a .NET SDK of at least the given major
// version is installed.
func requireDotnet(t *testing.T, major int) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("dotnet"); err != nil {
		t.Skip("dotnet not installed")
	}
	out, err := exec.Command("dotnet", "--list-sdks").Output()
	if err != nil {
		t.Skipf("dotnet --list-sdks failed: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		var found int
		if _, err := fmt.Sscanf(line, "%d.", &found); err == nil && found >= major {
			return
		}
	}
	t.Skipf(".NET %d SDK not installed", major)
}

// invoke runs the built function in dev mode against a fake Lambda Runtime API,
// feeds it one event and returns what the function posted back. Errors are
// prefixed so assertions on the body fail with a useful message.
func invoke(t *testing.T, r *Runtime, build *runtime.BuildOutput, event string, env []string) string {
	t.Helper()
	responses := make(chan string, 1)
	done := make(chan struct{})
	invoked := false
	mux := http.NewServeMux()
	mux.HandleFunc("/lambda/w1/2018-06-01/runtime/invocation/next", func(w http.ResponseWriter, req *http.Request) {
		if invoked {
			<-done
			return
		}
		invoked = true
		w.Header().Set("Lambda-Runtime-Aws-Request-Id", "req-1")
		w.Header().Set("Lambda-Runtime-Deadline-Ms", "9999999999999")
		w.Header().Set("Lambda-Runtime-Invoked-Function-Arn", "arn:aws:lambda:us-east-1:123456789012:function:test")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(event))
	})
	mux.HandleFunc("/lambda/w1/2018-06-01/runtime/invocation/req-1/response", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		responses <- string(body)
		w.WriteHeader(http.StatusAccepted)
	})
	mux.HandleFunc("/lambda/w1/2018-06-01/runtime/invocation/req-1/error", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		responses <- "ERROR: " + string(body)
		w.WriteHeader(http.StatusAccepted)
	})
	mux.HandleFunc("/lambda/w1/2018-06-01/runtime/init/error", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		responses <- "INIT ERROR: " + string(body)
		w.WriteHeader(http.StatusAccepted)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(done) })
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))

	worker, err := r.Run(context.Background(), &runtime.RunInput{
		Runtime:    "dotnet",
		Server:     "127.0.0.1:" + port + "/lambda/w1",
		FunctionID: "fn",
		WorkerID:   "w1",
		Build:      build,
		Env: append(append(os.Environ(),
			`SST_RESOURCE_App={"name":"example","stage":"test"}`,
			"AWS_LAMBDA_FUNCTION_NAME=example-test-fn",
			"AWS_LAMBDA_FUNCTION_MEMORY_SIZE=1024",
		), env...),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(worker.Stop)
	var logs strings.Builder
	go io.Copy(&logs, worker.Logs())

	select {
	case body := <-responses:
		return body
	case <-time.After(90 * time.Second):
		t.Fatalf("timed out waiting for the function to respond, logs:\n%s", logs.String())
		return ""
	}
}

// devBuild builds an example in dev mode and returns the output with Out set,
// the way the runtime collection would.
func devBuild(t *testing.T, r *Runtime, cfg string, id string, handler string, rt string) *runtime.BuildOutput {
	t.Helper()
	out, err := r.Build(context.Background(), &runtime.BuildInput{
		FunctionID: id,
		Handler:    handler,
		Runtime:    rt,
		Dev:        true,
		CfgPath:    cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Errors) > 0 {
		t.Fatalf("build errors: %s", strings.Join(out.Errors, "\n"))
	}
	out.Out = (&runtime.BuildInput{CfgPath: cfg, FunctionID: id, Dev: true}).Out()
	return out
}

// TestExampleBuildAndRun publishes the example function with the real dotnet SDK,
// runs it in dev mode against a fake Lambda Runtime API and checks the
// invocation round trip. It also checks a deploy build produces the files the
// managed Lambda runtime needs. Skipped when dotnet is not installed.
func TestExampleBuildAndRun(t *testing.T) {
	requireDotnet(t, 8)
	example := filepath.Join(findRepoRoot(t), "examples", "aws-lambda-dotnet")
	if _, err := os.Stat(example); err != nil {
		t.Skipf("example not found: %v", err)
	}
	// Keep artifacts out of the example's .sst directory.
	cfg := filepath.Join(t.TempDir(), "sst.config.ts")
	// The CLI runs from the app root and handlers are relative to it.
	t.Chdir(example)

	r := New()

	t.Run("deploy build", func(t *testing.T) {
		out, err := r.Build(context.Background(), &runtime.BuildInput{
			FunctionID: "deploy",
			Handler:    "./src",
			Runtime:    "dotnet8",
			Properties: json.RawMessage(`{"architecture":"arm64"}`),
			CfgPath:    cfg,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Errors) > 0 {
			t.Fatalf("build errors: %s", strings.Join(out.Errors, "\n"))
		}
		if out.Handler != "MyFunction" {
			t.Errorf("Handler = %q, want MyFunction", out.Handler)
		}
		dir := (&runtime.BuildInput{CfgPath: cfg, FunctionID: "deploy"}).Out()
		for _, f := range []string{"MyFunction.dll", "MyFunction.deps.json", "MyFunction.runtimeconfig.json", "Amazon.Lambda.RuntimeSupport.dll", "SST.Sdk.dll"} {
			if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
				t.Errorf("missing %s in publish output", f)
			}
		}
	})

	t.Run("dev build and run", func(t *testing.T) {
		out := devBuild(t, r, cfg, "dev", "./src", "dotnet8")

		if !r.ShouldRebuild("dev", filepath.Join(example, "src", "Function.cs")) {
			t.Error("expected source change to trigger rebuild")
		}
		if r.ShouldRebuild("dev", filepath.Join(example, "src", "obj", "Debug", "net8.0", "MyFunction.GlobalUsings.g.cs")) {
			t.Error("expected generated file not to trigger rebuild")
		}

		body := invoke(t, r, out, `{"hello":"world"}`, []string{
			`SST_RESOURCE_MyBucket={"name":"my-test-bucket"}`,
		})
		if body != `"my-test-bucket"` {
			t.Errorf("unexpected response body %s", body)
		}
	})
}

// TestDotnet10Examples builds the .NET 10 examples, a file-based app with a
// function URL and an ASP.NET Core project behind API Gateway V2, runs them in
// dev mode and sends one event through the fake Runtime API. Both payloads use
// the HTTP API v2 shape. Needs the .NET 10 SDK.
func TestDotnet10Examples(t *testing.T) {
	requireDotnet(t, 10)
	root := findRepoRoot(t)

	event := func(path string) string {
		return `{
		"version": "2.0",
		"routeKey": "$default",
		"rawPath": "` + path + `",
		"rawQueryString": "",
		"headers": {"host": "example.com", "user-agent": "test"},
		"requestContext": {
			"accountId": "123456789012",
			"apiId": "api",
			"domainName": "example.com",
			"http": {"method": "GET", "path": "` + path + `", "protocol": "HTTP/1.1", "sourceIp": "127.0.0.1", "userAgent": "test"},
			"requestId": "req-1",
			"routeKey": "$default",
			"stage": "$default",
			"time": "12/Sep/2026:00:00:00 +0000",
			"timeEpoch": 1789000000000
		},
		"isBase64Encoded": false
	}`
	}

	cases := []struct {
		name     string
		example  string
		handler  string
		assembly string
		path     string
		env      []string
		want     string
	}{
		{
			name:     "file-based app",
			example:  "aws-lambda-dotnet-file-based",
			handler:  "./src/Api.cs",
			assembly: "Api",
			path:     "/",
			// The home route reports the linked table name without calling DynamoDB.
			env:  []string{`SST_RESOURCE_DotnetTable={"name":"test-notes-table"}`, "AWS_REGION=us-east-1"},
			want: `"table":"test-notes-table"`,
		},
		{
			name:     "project",
			example:  "aws-dotnet-project-based-apigw",
			handler:  "./src",
			assembly: "Api",
			path:     "/hello",
			want:     "hello world",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			example := filepath.Join(root, "examples", tc.example)
			if _, err := os.Stat(example); err != nil {
				t.Skipf("example not found: %v", err)
			}
			cfg := filepath.Join(t.TempDir(), "sst.config.ts")
			t.Chdir(example)
			r := New()

			out := devBuild(t, r, cfg, "api", tc.handler, "dotnet10")
			if out.Handler != tc.assembly {
				t.Errorf("Handler = %q, want %q", out.Handler, tc.assembly)
			}
			if !r.ShouldRebuild("api", filepath.Join(example, "src", "anything.cs")) {
				t.Error("expected source change to trigger rebuild")
			}

			body := invoke(t, r, out, event(tc.path), tc.env)
			var response struct {
				StatusCode int    `json:"statusCode"`
				Body       string `json:"body"`
			}
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				t.Fatalf("response is not an HTTP API v2 response: %s", body)
			}
			if response.StatusCode != 200 || !strings.Contains(response.Body, tc.want) {
				t.Errorf("unexpected response %s", body)
			}

			// A deploy build must produce a framework-dependent layout the
			// managed dotnet10 runtime can load.
			deploy, err := r.Build(context.Background(), &runtime.BuildInput{
				FunctionID: "deploy",
				Handler:    tc.handler,
				Runtime:    "dotnet10",
				Properties: json.RawMessage(`{"architecture":"x86_64"}`),
				CfgPath:    cfg,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(deploy.Errors) > 0 {
				t.Fatalf("deploy build errors: %s", strings.Join(deploy.Errors, "\n"))
			}
			dir := (&runtime.BuildInput{CfgPath: cfg, FunctionID: "deploy"}).Out()
			for _, f := range []string{tc.assembly + ".dll", tc.assembly + ".deps.json", tc.assembly + ".runtimeconfig.json", "SST.Sdk.dll"} {
				if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
					t.Errorf("missing %s in publish output", f)
				}
			}
			for _, f := range []string{tc.assembly, "web.config"} {
				if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
					t.Errorf("unexpected %s in publish output", f)
				}
			}
		})
	}
}
