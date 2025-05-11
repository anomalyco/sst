package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/sst/sst/v3/internal/util"
	"github.com/sst/sst/v3/pkg/flag"
	"github.com/sst/sst/v3/pkg/global"
	"github.com/sst/sst/v3/pkg/js"
	"github.com/sst/sst/v3/pkg/npm"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/path"
	"golang.org/x/sync/errgroup"
)

type ErrProviderVersionTooLow struct {
	Name    string
	Version string
	Needed  string
}

func (err *ErrProviderVersionTooLow) Error() string {
	return "ErrorProviderVersionTooLow"
}

type ErrPluginNotFound struct {
	Name    string
	Version string
}

func (err *ErrPluginNotFound) Error() string {
	return "ErrPluginNotFound"
}

type platformPackageJson struct {
	js.PackageJson
	SST map[string]*pluginEntry `json:"sst,omitempty"`
}

type pluginEntry struct {
	Alias   string `json:"alias"`
	Package string `json:"package"`
}

type pluginPackageJson struct {
	js.PackageJson
	Pulumi *struct {
		Name             string `json:"name,omitempty"`
		Version          string `json:"version,omitempty"`
		Parameterization *struct {
			Name string `json:"name,omitempty"`
		} `json:"parameterization,omitempty"`
	} `json:"pulumi,omitempty"`
	SST *struct {
		Name string `json:"name,omitempty"`
	} `json:"sst,omitempty"`
}

func (p *Project) loadPlugins() error {
	var pkg platformPackageJson
	pkgFile, err := os.OpenFile(path.ResolvePlatformPackageJson(p.config), os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		defer pkgFile.Close()
		err = json.NewDecoder(pkgFile).Decode(&pkg)
	}
	pkg.Type = "module"
	if pkg.SST == nil {
		pkg.SST = map[string]*pluginEntry{}
	}
	if pkg.Dependencies == nil {
		pkg.Dependencies = map[string]string{}
	}

	dirty := []*Plugin{}
	for name, plugin := range p.plugins {
		match := pkg.SST[name]
		if match != nil {
			version := pkg.PackageJson.Dependencies[match.Package]
			if plugin.Version == "latest" || version == plugin.Version {
				plugin.Alias = match.Alias
				plugin.Package = match.Package
				continue
			}
		}
		dirty = append(dirty, plugin)
	}

	if len(dirty) > 0 {
		slog.Info("dirty plugins", "plugins", slices.Collect(util.Map(slices.Values(dirty), func(item *Plugin) string {
			return item.Name
		})))
		var wg errgroup.Group
		var lock sync.Mutex
		for _, plugin := range dirty {
			wg.Go(func() error {
				name, version, err := FindPlugin(plugin.Name, plugin.Version)
				if err != nil {
					return err
				}
				lock.Lock()
				defer lock.Unlock()
				pkg.Dependencies[name] = version
				pkg.SST[plugin.Name] = &pluginEntry{
					Package: name,
				}
				return nil
			})
		}

		err := wg.Wait()
		if err != nil {
			return err
		}

		pkgFile.Truncate(0)
		pkgFile.Seek(0, 0)
		err = json.NewEncoder(pkgFile).Encode(pkg)
		if err != nil {
			return fmt.Errorf("failed to write package.json: %w", err)
		}

		err = p.installPlugins()
		if err != nil {
			return err
		}

		for _, plugin := range dirty {
			entry := pkg.SST[plugin.Name]
			path := filepath.Join(p.PathPlatformDir(), "node_modules", entry.Package, "package.json")
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			var pluginPkg pluginPackageJson
			err = json.NewDecoder(file).Decode(&pluginPkg)
			if err != nil {
				return err
			}
			if pluginPkg.SST != nil {
				plugin.Alias = pluginPkg.SST.Name
			}
			if pluginPkg.Pulumi != nil {
				plugin.Alias = pluginPkg.Pulumi.Name
				if pluginPkg.Pulumi.Parameterization != nil {
					plugin.Alias = pluginPkg.Pulumi.Parameterization.Name
				}
				if plugin.Alias == "" || plugin.Alias == "terraform-provider" {
					plugin.Alias = pluginPkg.Name
					plugin.Alias = strings.ReplaceAll(plugin.Alias, "@sst-provider", "")
					plugin.Alias = strings.ReplaceAll(plugin.Alias, "/", "")
					plugin.Alias = strings.ReplaceAll(plugin.Alias, "@", "")
					plugin.Alias = strings.ReplaceAll(plugin.Alias, "pulumi", "")
				}
			}
			plugin.Package = entry.Package
			pkg.SST[plugin.Name].Alias = plugin.Alias
		}
		pkgFile.Truncate(0)
		pkgFile.Seek(0, 0)
		err = json.NewEncoder(pkgFile).Encode(pkg)
		if err != nil {
			return err
		}

		err = p.writeTypes()
		if err != nil {
			return err
		}

	}

	return nil
}

func FindPlugin(name string, version string) (string, string, error) {
	if strings.HasPrefix(version, "link:") {
		return strings.TrimPrefix(version, "link:"), version, nil
	}
	for _, prefix := range []string{"@sst-provider/", "sst-plugin-", "@pulumi/", "@pulumiverse/", "pulumi-", "@", ""} {
		pkg, err := npm.Get(prefix+name, version)
		if err != nil {
			continue
		}
		if pkg.Other["sst"] == nil && pkg.Other["pulumi"] == nil {
			continue
		}
		return pkg.Name, pkg.Version, nil
	}
	return "", "", &ErrPluginNotFound{Name: name, Version: version}
}

func (p *Project) installPlugins() error {
	slog.Info("fetching deps")
	manager := global.BunPath()
	if flag.SST_NO_BUN {
		manager = "npm"
	}
	cmd := process.Command(manager, "install")
	cmd.Dir = p.PathPlatformDir()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New("failed to run bun install " + string(output))
	}
	return nil
}

func (p *Project) writeTypes() error {
	slog.Info("writing types")
	typesPath := filepath.Join(p.PathPlatformDir(), "config.d.ts")
	file, err := os.OpenFile(typesPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString(`import { AppInput, AppOutput, Config } from "sst-plugin/config"` + "\n")

	for _, plugin := range p.plugins {
		file.WriteString(`import * as _` + plugin.Alias + ` from "` + plugin.Package + `";` + "\n")
	}

	file.WriteString("\n\n")

	file.WriteString(`declare global {` + "\n")
	for _, plugin := range p.plugins {
		splits := strings.Split(plugin.Alias, ".")
		for _, part := range splits {
			file.WriteString(`  export namespace ` + part + ` {` + "\n")
		}
		file.WriteString(`export * from "` + plugin.Package + `";` + "\n")
		for range splits {
			file.WriteString(`  }` + "\n")
		}
	}
	file.WriteString(`  interface Providers {` + "\n")
	file.WriteString(`    /** @deprecated` + "\n")
	file.WriteString(`     * Use ` + "`plugins`" + ` instead.` + "\n")
	file.WriteString(`     */` + "\n")
	file.WriteString(`    providers?: {` + "\n")
	for _, plugin := range p.plugins {
		if plugin.Hidden {
			continue
		}
		file.WriteString(`      "` + plugin.Name + `"?:  (_` + plugin.Alias + `.ProviderArgs & { version?: string }) | boolean | string;` + "\n")
	}
	file.WriteString(`    }` + "\n")
	file.WriteString(`    plugins?: {` + "\n")
	for _, plugin := range p.plugins {
		if plugin.Hidden {
			continue
		}
		file.WriteString(`      "` + plugin.Name + `"?:  { version?: string, config?: _` + plugin.Alias + `.ProviderArgs   } | string;` + "\n")
	}
	file.WriteString(`    }` + "\n")
	file.WriteString(`  }` + "\n")
	file.WriteString(`  export const $config: (` + "\n")
	file.WriteString(`    input: Omit<Config, "app"> & {` + "\n")
	file.WriteString(`      app(input: AppInput): Promise<Omit<AppOutput, "plugins" | "providers"> & Providers> | (Omit<AppOutput, "providers" | "plugins"> & Providers);` + "\n")
	file.WriteString(`    },` + "\n")
	file.WriteString(`  ) => Config;` + "\n")
	file.WriteString(`}` + "\n")

	return nil
}
