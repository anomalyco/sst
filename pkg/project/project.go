package project

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/sst/sst/v3/internal/fs"
	"github.com/sst/sst/v3/internal/util"
	"github.com/sst/sst/v3/pkg/flag"
	"github.com/sst/sst/v3/pkg/js"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/provider"
	"github.com/sst/sst/v3/pkg/runtime"
	"github.com/sst/sst/v3/pkg/runtime/golang"
	"github.com/sst/sst/v3/pkg/runtime/node"
	"github.com/sst/sst/v3/pkg/runtime/python"
	"github.com/sst/sst/v3/pkg/runtime/rust"
	"github.com/sst/sst/v3/pkg/runtime/worker"
)

type App struct {
	Name    string `json:"name"`
	Stage   string `json:"stage"`
	Removal string `json:"removal"`
	Home    string `json:"home"`
	Version string `json:"version"`
	Protect bool   `json:"protect"`
}

type Project struct {
	version         string
	root            string
	config          string
	app             *App
	home            provider.Home
	env             map[string]string
	plugins         map[string]*Plugin
	loadedProviders map[string]provider.Provider
	Runtime         *runtime.Collection
}

type Plugin struct {
	Name    string                 `json:"name"`
	Version string                 `json:"version"`
	Config  map[string]interface{} `json:"config"`

	Package string `json:"package"`
	Alias   string `json:"alias"`
	Hidden  bool   `json:"hidden"`
}

func Discover() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cfgPath, err := fs.FindUp(cwd, "sst.config.ts")
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(ResolveWorkingDir(cfgPath), 0755)
	if err != nil {
		return "", err
	}
	return cfgPath, nil
}

func ResolveWorkingDir(cfgPath string) string {
	return filepath.Join(filepath.Dir(cfgPath), ".sst")
}

func ResolvePlatformDir(cfgPath string) string {
	return filepath.Join(ResolveWorkingDir(cfgPath), "platform")
}

func ResolveLogDir(cfgPath string) string {
	return filepath.Join(ResolveWorkingDir(cfgPath), "log")
}

type ProjectConfig struct {
	Version string
	Stage   string
	Config  string
}

var ErrInvalidStageName = fmt.Errorf("ErrInvalidStageName")
var ErrInvalidAppName = fmt.Errorf("ErrInvalidAppName")
var ErrAppNameChanged = fmt.Errorf("ErrAppNameChanged")
var ErrV2Config = fmt.Errorf("ErrV2Config")
var ErrVersionInvalid = fmt.Errorf("ErrVersionInvalid")

type ErrVersionMismatch struct {
	Needed   string
	Received string
}

func (err *ErrVersionMismatch) Error() string {
	return "ErrorVersionMismatch"
}

type ErrBuildFailed struct {
	msg    string
	Errors []api.Message
}

func (err *ErrBuildFailed) Error() string {
	return err.msg
}

var InvalidStageRegex = regexp.MustCompile(`[^a-zA-Z0-9-]`)
var InvalidAppRegex = regexp.MustCompile(`^[^a-zA-Z]|[^a-zA-Z0-9-]`)

func New(input *ProjectConfig) (*Project, error) {
	if InvalidStageRegex.MatchString(input.Stage) {
		return nil, ErrInvalidStageName
	}
	rootPath := filepath.Dir(input.Config)
	proj := &Project{
		version: input.Version,
		root:    rootPath,
		plugins: map[string]*Plugin{},
		config:  input.Config,
		env:     map[string]string{},
		Runtime: runtime.NewCollection(
			input.Config,
			node.New(input.Version),
			worker.New(),
			python.New(),
			golang.New(),
			rust.New(),
		),
	}
	workdir := proj.PathWorkingDir()
	_, err := os.Stat(workdir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		err := os.Mkdir(workdir, 0755)
		if err != nil {
			return nil, err
		}
	}
	inputBytes, err := json.Marshal(map[string]string{
		"stage": input.Stage,
	})
	buildResult, err := js.Build(
		js.EvalOptions{
			Dir:    proj.PathRoot(),
			Banner: `function $config(input) { return input }`,
			Define: map[string]string{
				"$input": string(inputBytes),
			},
			Code: fmt.Sprintf(`
import mod from '%s';
if (mod.stacks || mod.config) {
  console.log("~v2")
  process.exit(0)
}
console.log("~j" + JSON.stringify(await mod.app({
  stage: $input.stage || undefined,
})))`,
				filepath.ToSlash(input.Config)),
		},
	)
	if err != nil {
		if buildResult.Errors != nil {
			return nil, &ErrBuildFailed{msg: err.Error(), Errors: buildResult.Errors}
		}
		return nil, err
	}
	defer js.Cleanup(buildResult)
	slog.Info("evaluating config")
	node := process.Command("node", "--no-warnings", string(buildResult.OutputFiles[1].Path))
	output, err := node.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error evaluating config: %w\n%s", err, output)
	}
	var parsed struct {
		Name      string                 `json:"name"`
		Stage     string                 `json:"stage"`
		Removal   string                 `json:"removal"`
		Providers map[string]interface{} `json:"providers"`
		Plugins   map[string]interface{} `json:"plugins"`
		Home      string                 `json:"home"`
		Version   string                 `json:"version"`
		Protect   bool                   `json:"protect"`
		// Deprecated: Backend is now Home
		Backend string `json:"backend"`
		// Deprecated: RemovalPolicy is now Removal
		RemovalPolicy string `json:"removalPolicy"`
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "~v2" {
			return nil, ErrV2Config
		}
		if strings.HasPrefix(line, "~j") {
			err = json.Unmarshal([]byte(line[2:]), &parsed)
			if err != nil {
				return nil, err
			}
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	proj.app = &App{
		Name:    parsed.Name,
		Stage:   input.Stage,
		Removal: parsed.Removal,
		Home:    parsed.Home,
		Version: parsed.Version,
		Protect: parsed.Protect,
	}

	if proj.app.Name == "" {
		return nil, fmt.Errorf("Project name is required")
	}

	if proj.app.Home == "" {
		return nil, util.NewReadableError(nil, `You must specify a "home" provider in the project configuration file.`)
	}

	if InvalidAppRegex.MatchString(proj.app.Name) {
		return nil, ErrInvalidAppName
	}

	if proj.app.Removal == "" {
		proj.app.Removal = "retain"
	}

	if parsed.RemovalPolicy != "" {
		return nil, util.NewReadableError(nil, `The "removalPolicy" has been renamed to "removal"`)
	}

	if proj.app.Version != "" && input.Version != "dev" {
		constraint, err := semver.NewConstraint(proj.app.Version)
		if err != nil {
			return nil, ErrVersionInvalid
		}
		version, err := semver.NewVersion(input.Version)
		if err != nil {
			return nil, ErrVersionInvalid
		}
		if !constraint.Check(version) {
			return nil, &ErrVersionMismatch{Needed: input.Version, Received: proj.app.Version}
		}
	}

	if proj.app.Removal != "remove" && proj.app.Removal != "retain" && proj.app.Removal != "retain-all" {
		return nil, fmt.Errorf("Removal must be one of: remove, retain, retain-all")
	}

	// Check if app name has changed by comparing the folder name inside ".pulumi/stacks"
	// and the app name in the config file.
	stacksDir := filepath.Join(proj.PathWorkingDir(), ".pulumi", "stacks")
	files, err := os.ReadDir(stacksDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		files = []os.DirEntry{}
	}
	if len(files) > 0 {
		appName := files[0].Name()
		if appName != proj.app.Name {
			return nil, ErrAppNameChanged
		}
	}

	if parsed.Plugins != nil {
		for name, args := range parsed.Plugins {
			plugin := Plugin{
				Name:    name,
				Config:  map[string]interface{}{},
				Version: "latest",
			}

			if argsString, ok := args.(string); ok {
				plugin.Version = argsString
			}

			if argsMap, ok := args.(map[string]interface{}); ok {
				if alias, ok := argsMap["alias"].(string); ok {
					plugin.Alias = alias
				}
				if config, ok := argsMap["config"].(map[string]interface{}); ok {
					plugin.Config = config
				}
				if version, ok := argsMap["version"].(string); ok {
					plugin.Version = version
				}
			}
			proj.plugins[name] = &plugin
		}
	}

	// deprecated
	if parsed.Providers != nil {
		for name, args := range parsed.Providers {
			plugin := Plugin{
				Name:    name,
				Config:  map[string]interface{}{},
				Version: "latest",
			}

			if argsString, ok := args.(string); ok {
				plugin.Version = argsString
			}

			if argsMap, ok := args.(map[string]interface{}); ok {
				if version, ok := argsMap["version"].(string); ok {
					plugin.Version = version
				}
				delete(argsMap, "version")
				plugin.Config = argsMap
			}
			proj.plugins[name] = &plugin
		}
	}

	if _, ok := proj.plugins[proj.app.Home]; !ok && proj.app.Home != "local" {
		proj.plugins[proj.app.Home] = &Plugin{
			Name:    proj.app.Home,
			Config:  map[string]interface{}{},
			Version: "latest",
		}
	}

	proj.plugins["sst-plugin"] = &Plugin{
		Name:    "sst-plugin",
		Version: "0.0.12",
		Hidden:  true,
		Config:  map[string]interface{}{},
	}
	proj.plugins["sst-plugin"].Version = "link:sst-plugin"

	err = proj.loadPlugins()
	if err != nil {
		return nil, err
	}

	return proj, nil
}

func (proj *Project) LoadHome() error {
	slog.Info("loading home")
	loadedProviders := make(map[string]provider.Provider)

	for _, plugin := range proj.plugins {
		var match provider.Provider
		switch plugin.Name {
		case "cloudflare":
			match = &provider.CloudflareProvider{}
		case "aws":
			match = provider.NewAwsProvider()
		}
		if match == nil {
			continue
		}
		err := match.Init(proj.app.Name, proj.app.Stage, plugin.Config)
		if err != nil {
			return util.NewReadableError(err, plugin.Name+": "+err.Error())
		}
		env, err := match.Env()
		if err != nil {
			return err
		}
		for key, value := range env {
			proj.env[key] = value
		}
		loadedProviders[plugin.Name] = match
	}

	var home provider.Home

	switch proj.app.Home {
	case "local":
		home = provider.NewLocalHome()
	case "aws":
		home = provider.NewAwsHome(loadedProviders["aws"].(*provider.AwsProvider))
	case "cloudflare":
		home = provider.NewCloudflareHome(loadedProviders["cloudflare"].(*provider.CloudflareProvider))
	default:
		return fmt.Errorf("Home provider %s is invalid", proj.app.Home)
	}

	err := home.Bootstrap()
	if err != nil {
		return fmt.Errorf("Error initializing %s:\n   %w", proj.app.Home, err)
	}
	proj.home = home
	proj.loadedProviders = loadedProviders
	return nil
}

func (p Project) getPath(path ...string) string {
	paths := append([]string{p.PathWorkingDir()}, path...)
	return filepath.Join(paths...)
}

func (p Project) PathWorkingDir() string {
	return filepath.Join(p.root, ".sst")
}

func (p Project) PathPlatformDir() string {
	return filepath.Join(p.PathWorkingDir(), "platform")
}

func (p Project) PathPlatformSST() string {
	return filepath.Join(p.PathPlatformDir(), "node_modules", "@sst", "platform")
}

func (p Project) PathRoot() string {
	return p.root
}

func (p Project) PathConfig() string {
	return p.config
}

func (p Project) Version() string {
	return p.version
}

func (p Project) App() *App {
	return p.app
}

func (p Project) Backend() provider.Home {
	return p.home
}

func (p Project) Env() map[string]string {
	return p.env
}

func (p *Project) Provider(name string) (provider.Provider, bool) {
	result, ok := p.loadedProviders[name]
	return result, ok
}

func (p *Project) Cleanup() error {
	if flag.SST_NO_CLEANUP {
		return nil
	}
	return nil
}

func (p *Project) Plugins() map[string]*Plugin {
	return p.plugins
}

func (p *Project) PathLog(name string) string {
	if name == "" {
		return filepath.Join(p.PathWorkingDir(), "log")
	}
	return filepath.Join(p.PathWorkingDir(), "log", name+".log")
}
