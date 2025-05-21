package js

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/zeebo/xxh3"
)

var ErrTopLevelImport = fmt.Errorf("ErrTopLevelImport")

type EvalOptions struct {
	External    []string
	Dir         string
	ResourceDir string
	Outfile     string
	Code        string
	Env         []string
	Banner      string
	Inject      []string
	Define      map[string]string
}

type PackageJson struct {
	Name            string                 `json:"name,omitempty"`
	Version         string                 `json:"version,omitempty"`
	Type            string                 `json:"type,omitempty"`
	Dependencies    map[string]string      `json:"dependencies,omitempty"`
	DevDependencies map[string]string      `json:"devDependencies,omitempty"`
	Overrides       map[string]string      `json:"overrides,omitempty"`
	Exports         map[string]interface{} `json:"exports,omitempty"`
	Other           map[string]any         `json:"-"`
}

type Metafile struct {
	Inputs map[string]struct {
		Bytes   int `json:"bytes"`
		Imports []struct {
			Path        string `json:"path"`
			Kind        string `json:"kind"`
			External    bool   `json:"external,omitempty"`
			Original    string `json:"original,omitempty"`
			Namespace   string `json:"namespace,omitempty"`
			SideEffects bool   `json:"sideEffects,omitempty"`
		} `json:"imports"`
	} `json:"inputs"`
	Outputs map[string]struct {
		Bytes  int `json:"bytes"`
		Inputs map[string]struct {
			BytesInOutput int `json:"bytesInOutput"`
		} `json:"inputs"`
		Exports    []string `json:"exports"`
		Entrypoint string   `json:"entrypoint"`
	} `json:"outputs"`
}

func Build(input EvalOptions) (esbuild.BuildResult, error) {
	outfile := input.Outfile
	if outfile == "" {
		outfile = filepath.Join(input.Dir, ".sst", "platform", fmt.Sprintf("sst.config.%v.mjs", time.Now().UnixMilli()))
	}
	slog.Info("esbuild building", "out", outfile)
	var err error
	result := esbuild.Build(esbuild.BuildOptions{
		Banner: map[string]string{
			"js": `
import { createRequire as topLevelCreateRequire } from 'module';
const require = topLevelCreateRequire(import.meta.url);
import { fileURLToPath as topLevelFileUrlToPath, URL as topLevelURL } from "url"
const __filename = topLevelFileUrlToPath(import.meta.url)
const __dirname = topLevelFileUrlToPath(new topLevelURL(".", import.meta.url))
` + input.Banner,
		},
		MainFields: []string{"module", "main"},
		Format:     esbuild.FormatESModule,
		Platform:   esbuild.PlatformNode,
		Target:     esbuild.ES2023,
		Sourcemap:  esbuild.SourceMapLinked,
		Stdin: &esbuild.StdinOptions{
			Contents:   input.Code,
			ResolveDir: input.Dir,
			Sourcefile: "eval.ts",
			Loader:     esbuild.LoaderTS,
		},
		NodePaths: []string{
			filepath.Join(input.Dir, ".sst", "platform", "node_modules"),
		},
		Plugins: []esbuild.Plugin{
			{
				Name: "DisallowImports",
				Setup: func(build esbuild.PluginBuild) {
					build.OnResolve(esbuild.OnResolveOptions{Filter: ".*"}, func(args esbuild.OnResolveArgs) (esbuild.OnResolveResult, error) {
						if filepath.Base(args.Importer) == "sst.config.ts" && args.Kind == esbuild.ResolveJSImportStatement {
							err = ErrTopLevelImport
							return esbuild.OnResolveResult{}, ErrTopLevelImport
						}
						return esbuild.OnResolveResult{}, nil
					})
				},
			},
			{
				Name: "ExtractResources",
				Setup: func(build esbuild.PluginBuild) {
					pattern := regexp.MustCompile(`(export\s+const\s+)([a-zA-Z0-9_]+)(\s*=\s*sst\.resource\s*\()(\s*\{)`)
					dir := filepath.Join(filepath.Dir(outfile), "resource")
					os.RemoveAll(dir)
					os.MkdirAll(dir, 0755)

					build.OnLoad(esbuild.OnLoadOptions{Filter: `\.([jt]sx?|mjs|cjs)$`},
						func(args esbuild.OnLoadArgs) (esbuild.OnLoadResult, error) {
							content, err := os.ReadFile(args.Path)
							if err != nil {
								return esbuild.OnLoadResult{}, fmt.Errorf("failed to read file: %v", err)
							}

							hash := fmt.Sprint(xxh3.Hash(content))
							fileContent := string(content)
							found := false
							processedContent := pattern.ReplaceAllStringFunc(fileContent, func(match string) string {
								found = true
								submatch := pattern.FindStringSubmatch(match)
								if len(submatch) < 5 {
									return match
								}

								constPart := submatch[1]
								resourceName := submatch[2]

								return fmt.Sprintf(`%s%s = resource({"type": "%s", "__source": { type: "dynamic", hash: "%s", export: "%s" }, `,
									constPart, resourceName, resourceName, hash, resourceName)
							})
							if found {
								processedContent = "import { resource } from \"sst-plugin\";\n" + processedContent
								result := esbuild.Build(esbuild.BuildOptions{
									AbsWorkingDir: filepath.Dir(args.Path),
									External:      []string{"sst-plugin"},
									MainFields:    []string{"module", "main"},
									Stdin: &esbuild.StdinOptions{
										Sourcefile: args.Path,
										ResolveDir: filepath.Dir(args.Path),
										Contents:   processedContent,
										Loader:     esbuild.LoaderTS,
									},
									Outfile: filepath.Join(dir, hash+".mjs"),
									Bundle:  true,
									Write:   true,
									Format:  esbuild.FormatESModule,
									Target:  esbuild.ES2023,
								})
								if len(result.Errors) > 0 {
									return esbuild.OnLoadResult{}, fmt.Errorf("failed to build resource: %v", result.Errors)
								}
							}

							return esbuild.OnLoadResult{
								Contents: &processedContent,
								Loader:   esbuild.LoaderTS,
							}, nil
						})
				},
			},
		},
		External: input.External,
		Define:   input.Define,
		Inject:   input.Inject,
		Outfile:  outfile,
		Write:    true,
		Bundle:   true,
		Metafile: true,
	})
	if err != nil {
		return esbuild.BuildResult{}, err
	}
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			slog.Error("esbuild error", "text", err.Text)
		}
		return result, fmt.Errorf("%s", FormatError(result.Errors))
	}
	slog.Info("esbuild built", "outfile", outfile)

	analysis := esbuild.AnalyzeMetafile(result.Metafile, esbuild.AnalyzeMetafileOptions{
		Verbose: true,
	})
	os.WriteFile(filepath.Join(input.Dir, ".sst", "esbuild.json"), []byte(analysis), 0644)

	return result, nil
}

func FormatError(input []esbuild.Message) string {
	lines := []string{}
	for _, err := range input {
		if err.Location == nil {
			lines = append(lines, fmt.Sprintf("%v", err.Text))
			continue
		}
		lines = append(lines, fmt.Sprintf("%v:%v:%v: %v", err.Location.File, err.Location.Line, err.Location.Column, err.Text))
	}
	return strings.Join(lines, "\n")
}

func Cleanup(result esbuild.BuildResult) {
	for _, file := range result.OutputFiles {
		os.Remove(file.Path)
	}
}
