package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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
	return "provider version too low"
}

func (p *Project) writePackageJson() error {
	slog.Info("writing package.json")
	file, err := os.OpenFile(filepath.Join(p.PathPlatformDir(), "package.json"), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	pkg := js.PackageJson{
		Dependencies: map[string]string{},
	}
	json.NewDecoder(file).Decode(&pkg)

	for _, plugin := range p.plugins {
		slog.Info("adding dependency", "name", plugin.Name)
		pkg.Dependencies[plugin.Package] = plugin.Version
	}
	pkg.Dependencies["@pulumi/pulumi"] = global.PULUMI_VERSION

	if err := file.Truncate(0); err != nil {
		return err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	err = json.NewEncoder(file).Encode(pkg)
	if err != nil {
		return err
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

	file.WriteString(`import "@sst/platform/global.d.ts"` + "\n")
	file.WriteString(`import { AppInput, App, Config } from "@sst/platform/config"` + "\n")

	for _, plugin := range p.plugins {
		file.WriteString(`import * as _` + plugin.Alias + ` from "` + plugin.Package + `";` + "\n")
	}

	file.WriteString("\n\n")

	file.WriteString(`declare global {` + "\n")
	for _, plugin := range p.plugins {
		file.WriteString(`  // @ts-expect-error` + "\n")
		file.WriteString(`  export import ` + plugin.Alias + ` = _` + plugin.Alias + "\n")
	}
	file.WriteString(`  interface Providers {` + "\n")
	file.WriteString(`    /** @deprecated` + "\n")
	file.WriteString(`     * Use ` + "`plugins`" + ` instead.` + "\n")
	file.WriteString(`     */` + "\n")
	file.WriteString(`    providers?: {` + "\n")
	for _, plugin := range p.plugins {
		file.WriteString(`      "` + plugin.Name + `"?:  (_` + plugin.Alias + `.ProviderArgs & { version?: string }) | boolean | string;` + "\n")
	}
	file.WriteString(`    }` + "\n")
	file.WriteString(`    plugins?: {` + "\n")
	for _, plugin := range p.plugins {
		file.WriteString(`      "` + plugin.Name + `"?:  { version?: string, alias?: string, config?: _` + plugin.Alias + `.ProviderArgs   } | string;` + "\n")
	}
	file.WriteString(`    }` + "\n")
	file.WriteString(`  }` + "\n")
	file.WriteString(`  export const $config: (` + "\n")
	file.WriteString(`    input: Omit<Config, "app"> & {` + "\n")
	file.WriteString(`      app(input: AppInput): Promise<Omit<App, "providers"> & Providers> | (Omit<App, "providers"> & Providers);` + "\n")
	file.WriteString(`    },` + "\n")
	file.WriteString(`  ) => Config;` + "\n")
	file.WriteString(`}` + "\n")

	return nil
}

func (p *Project) fetchDeps() error {
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

func resolvePlugins(plugins []*Plugin) (map[string]*Plugin, error) {
	var wg errgroup.Group
	results := make(chan *Plugin, len(plugins))
	for _, plugin := range plugins {
		n := plugin.Name
		version := plugin.Version
		if version == "" {
			version = "latest"
		}
		wg.Go(func() error {
			result, err := FindProvider(n, version)
			if err != nil {
				return err
			}
			results <- result
			return nil
		})
	}
	err := wg.Wait()
	if err != nil {
		return nil, err
	}
	close(results)
	result := map[string]*Plugin{}
	for item := range results {
		result[item.Name] = item
	}
	return result, nil
}

func FindProvider(name string, version string) (*Plugin, error) {
	for _, prefix := range []string{"@sst-provider/", "@pulumi/", "@pulumiverse/", "pulumi-", "@", ""} {
		pkg, err := npm.Get(prefix+name, version)
		if err != nil {
			continue
		}
		if pkg.Pulumi == nil && pkg.SST == nil {
			continue
		}
		alias := pkg.Pulumi.Name
		if pkg.SST != nil {
			alias = pkg.SST.Name
		}
		if alias == "" || alias == "terraform-provider" {
			alias = pkg.Name
			alias = strings.ReplaceAll(alias, "@sst-provider", "")
			alias = strings.ReplaceAll(alias, "/", "")
			alias = strings.ReplaceAll(alias, "@", "")
			alias = strings.ReplaceAll(alias, "pulumi", "")
		}
		alias = strings.ReplaceAll(alias, "-", "")
		return &Plugin{
			Name:    name,
			Package: pkg.Name,
			Version: pkg.Version,
			Alias:   alias,
		}, nil
	}
	return nil, fmt.Errorf("provider %s not found", name)
}

func (p *Project) writeProviderLock() error {
	lockPath := path.ResolvePluginLock(p.PathConfig())
	data, err := json.MarshalIndent(p.plugins, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(lockPath, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
