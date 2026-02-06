package npm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sst/sst/v3/internal/fs"
)

var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)
var authTokenRegex = regexp.MustCompile(`^//([^/:]+).*:_authToken=(.+)$`)
var registryRegex = regexp.MustCompile(`^registry\s*=\s*(.+)$`)

type npmrc struct {
	registry string
	tokens   map[string]string
}

func parseNpmrc(content string) npmrc {
	result := npmrc{tokens: make(map[string]string)}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if matches := authTokenRegex.FindStringSubmatch(line); matches != nil {
			token := envVarRegex.ReplaceAllStringFunc(matches[2], func(match string) string {
				return os.Getenv(envVarRegex.FindStringSubmatch(match)[1])
			})
			result.tokens[matches[1]] = token
			continue
		}
		if matches := registryRegex.FindStringSubmatch(line); matches != nil {
			result.registry = strings.TrimSpace(matches[1])
		}
	}
	return result
}

type npmRegistry struct {
	url   string
	token string
}

func loadNpmRegistry() npmRegistry {
	var merged npmrc
	merged.tokens = make(map[string]string)

	merge := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		rc := parseNpmrc(string(data))
		if rc.registry != "" {
			merged.registry = rc.registry
		}
		for k, v := range rc.tokens {
			merged.tokens[k] = v
		}
	}

	// Load from ~/.npmrc
	if home, err := os.UserHomeDir(); err == nil {
		merge(filepath.Join(home, ".npmrc"))
	}

	// Load from project .npmrc (overrides home)
	if npmrcPath, err := fs.FindUp(".", ".npmrc"); err == nil {
		merge(npmrcPath)
	}

	// NPM_REGISTRY env var takes priority
	registry := os.Getenv("NPM_REGISTRY")
	if registry == "" {
		registry = merged.registry
	}
	if registry == "" {
		registry = "https://registry.npmjs.org"
	}

	var token string
	if parsed, err := url.Parse(registry); err == nil {
		token = merged.tokens[parsed.Host]
	}

	return npmRegistry{url: registry, token: token}
}

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
	registry := loadNpmRegistry()
	pkgURL := fmt.Sprintf("%s/%s/%s", registry.url, name, version)

	req, err := http.NewRequest("GET", pkgURL, nil)
	if err != nil {
		return nil, err
	}

	if registry.token != "" {
		req.Header.Set("Authorization", "Bearer "+registry.token)
	}

	resp, err := http.DefaultClient.Do(req)
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
