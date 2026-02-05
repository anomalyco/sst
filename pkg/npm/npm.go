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

func expandEnvVars(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		varName := envVarRegex.FindStringSubmatch(match)[1]
		return os.Getenv(varName)
	})
}

func parseNpmrc(path string) map[string]string {
	tokens := make(map[string]string)
	file, err := os.Open(path)
	if err != nil {
		return tokens
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		matches := authTokenRegex.FindStringSubmatch(line)
		if matches != nil {
			host := matches[1]
			token := expandEnvVars(matches[2])
			tokens[host] = token
		}
	}
	return tokens
}

func loadAuthTokens() map[string]string {
	tokens := make(map[string]string)

	// Load from ~/.npmrc
	if home, err := os.UserHomeDir(); err == nil {
		for k, v := range parseNpmrc(filepath.Join(home, ".npmrc")) {
			tokens[k] = v
		}
	}

	// Load from project .npmrc (overrides home)
	if npmrc, err := fs.FindUp(".", ".npmrc"); err == nil {
		for k, v := range parseNpmrc(npmrc) {
			tokens[k] = v
		}
	}

	return tokens
}

func getAuthToken(registryURL string, tokens map[string]string) string {
	parsed, err := url.Parse(registryURL)
	if err != nil {
		return ""
	}
	return tokens[parsed.Host]
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
	baseURL := os.Getenv("NPM_REGISTRY")
	if baseURL == "" {
		baseURL = "https://registry.npmjs.org"
	}
	pkgURL := fmt.Sprintf("%s/%s/%s", baseURL, name, version)

	req, err := http.NewRequest("GET", pkgURL, nil)
	if err != nil {
		return nil, err
	}

	tokens := loadAuthTokens()
	if token := getAuthToken(baseURL, tokens); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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
