package npm

import (
	"bufio"
	"encoding/base64"
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
var authTokenRegex = regexp.MustCompile(`^(//.+?):_authToken\s*=\s*(.+)$`)
var authRegex = regexp.MustCompile(`^(//.+?):_auth\s*=\s*(.+)$`)
var usernameRegex = regexp.MustCompile(`^(//.+?):username\s*=\s*(.+)$`)
var passwordRegex = regexp.MustCompile(`^(//.+?):_password\s*=\s*(.+)$`)
var registryRegex = regexp.MustCompile(`^registry\s*=\s*(.+)$`)

type Auth struct {
	token  string
	scheme string // "Bearer" or "Basic"
}

type npmrc struct {
	registry string
	auths    map[string]Auth
}

func expandEnvVars(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		return os.Getenv(envVarRegex.FindStringSubmatch(match)[1])
	})
}

func parseNpmrc(content string) npmrc {
	result := npmrc{auths: make(map[string]Auth)}
	usernames := make(map[string]string)
	passwords := make(map[string]string)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if matches := authTokenRegex.FindStringSubmatch(line); matches != nil {
			result.auths[matches[1]] = Auth{
				token:  strings.TrimSpace(expandEnvVars(matches[2])),
				scheme: "Bearer",
			}
			continue
		}
		if matches := authRegex.FindStringSubmatch(line); matches != nil {
			result.auths[matches[1]] = Auth{
				token:  strings.TrimSpace(expandEnvVars(matches[2])),
				scheme: "Basic",
			}
			continue
		}
		if matches := usernameRegex.FindStringSubmatch(line); matches != nil {
			usernames[matches[1]] = strings.TrimSpace(expandEnvVars(matches[2]))
			continue
		}
		if matches := passwordRegex.FindStringSubmatch(line); matches != nil {
			passwords[matches[1]] = strings.TrimSpace(expandEnvVars(matches[2]))
			continue
		}
		if matches := registryRegex.FindStringSubmatch(line); matches != nil {
			result.registry = strings.TrimSpace(matches[1])
		}
	}

	// username + _password produces Basic auth (password is base64-encoded in .npmrc)
	for host, username := range usernames {
		if _, exists := result.auths[host]; exists {
			continue
		}
		if password, ok := passwords[host]; ok {
			decoded, err := base64.StdEncoding.DecodeString(password)
			if err != nil {
				continue
			}
			result.auths[host] = Auth{
				token:  base64.StdEncoding.EncodeToString([]byte(username + ":" + string(decoded))),
				scheme: "Basic",
			}
		}
	}

	return result
}

type Registry struct {
	url  string
	auth Auth
}

func LoadRegistry() Registry {
	var merged npmrc
	merged.auths = make(map[string]Auth)

	merge := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		rc := parseNpmrc(string(data))
		if rc.registry != "" {
			merged.registry = rc.registry
		}
		for k, v := range rc.auths {
			merged.auths[k] = v
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

	auth := findAuth(registry, merged.auths)

	return Registry{url: registry, auth: auth}
}

// findAuth walks up the registry URL path to find the longest matching
// auth key, mirroring npm's regFromURI behavior.
// e.g. for https://registry.example.com/some/path it checks:
//
//	//registry.example.com/some/path
//	//registry.example.com/some/
//	//registry.example.com/some
//	//registry.example.com/
//	//registry.example.com
func findAuth(registryURL string, auths map[string]Auth) Auth {
	parsed, err := url.Parse(registryURL)
	if err != nil {
		return Auth{}
	}
	path := parsed.Path
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	regKey := "//" + parsed.Host + path
	for len(regKey) > len("//") {
		if auth, ok := auths[regKey]; ok {
			return auth
		}
		if regKey[len(regKey)-1] == '/' {
			// remove trailing slash: //host/path/ -> //host/path
			regKey = regKey[:len(regKey)-1]
		} else {
			// remove last segment: //host/path -> //host/
			i := strings.LastIndex(regKey, "/")
			if i <= 1 {
				break
			}
			regKey = regKey[:i+1]
		}
	}
	return Auth{}
}

type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Pulumi  *struct {
		Name             string `json:"name"`
		Version          string `json:"version"`
		Parameterization *struct {
			Name string `json:"name"`
		} `json:"parameterization"`
	}
}

func Get(registry Registry, name string, version string) (*Package, error) {
	slog.Info("getting package", "name", name, "version", version)
	pkgURL := fmt.Sprintf("%s/%s/%s", registry.url, name, version)

	req, err := http.NewRequest("GET", pkgURL, nil)
	if err != nil {
		return nil, err
	}

	if registry.auth.token != "" {
		req.Header.Set("Authorization", registry.auth.scheme+" "+registry.auth.token)
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
