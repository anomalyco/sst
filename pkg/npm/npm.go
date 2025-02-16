package npm

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/sst/sst/v3/internal/fs"
)

type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Pulumi  *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
}

func Get(name string, version string) (*Package, error) {
	slog.Info("getting package", "name", name, "version", version)
	baseUrl := os.Getenv("NPM_REGISTRY")
	if baseUrl == "" {
		baseUrl = "https://registry.npmjs.org"
	}
	url := fmt.Sprintf("%s/%s/%s", baseUrl, name, version)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch package: %s", resp.Status)
	}
	var data Package
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func DetectPackageManager(dir string) (string, string) {
	options := []struct {
		search string
		name   string
	}{
		{
			search: "package-lock.json",
			name:   "npm",
		},
		{
			search: "yarn.lock",
			name:   "yarn",
		},
		{
			search: "pnpm-lock.yaml",
			name:   "pnpm",
		},
		{
			search: "bun.lockb",
			name:   "bun",
		},
		{
			search: "bun.lock",
			name:   "bun",
		},
	}
	for _, option := range options {
		lock, err := fs.FindUp(dir, option.search)
		if err != nil {
			continue
		}
		return option.name, lock
	}
	return "", ""
}
