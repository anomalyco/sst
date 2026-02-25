package npm

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestParseNpmrc(t *testing.T) {
	t.Setenv("MY_TOKEN", "env-token")

	input := `# comment
//registry.npmjs.org/:_authToken=hardcoded-token
//custom.registry.com/:_authToken=${MY_TOKEN}

registry=https://custom.registry.com
not-a-token-line=value
//another.host/:_authToken=plain
`
	rc := parseNpmrc(input)

	if rc.registry != "https://custom.registry.com" {
		t.Errorf("registry = %q, want %q", rc.registry, "https://custom.registry.com")
	}

	wantAuths := map[string]Auth{
		"//registry.npmjs.org/":  {token: "hardcoded-token", scheme: "Bearer"},
		"//custom.registry.com/": {token: "env-token", scheme: "Bearer"},
		"//another.host/":        {token: "plain", scheme: "Bearer"},
	}
	for key, want := range wantAuths {
		got := rc.auths[key]
		if got.token != want.token || got.scheme != want.scheme {
			t.Errorf("auths[%q] = %+v, want %+v", key, got, want)
		}
	}
	if len(rc.auths) != len(wantAuths) {
		t.Errorf("len(auths) = %d, want %d", len(rc.auths), len(wantAuths))
	}
}

func TestParseNpmrcBasicAuth(t *testing.T) {
	input := `//private.registry.com/:_auth=dXNlcjpwYXNz
//token.registry.com/:_authToken=my-bearer-token
`
	rc := parseNpmrc(input)

	if got := rc.auths["//private.registry.com/"]; got.token != "dXNlcjpwYXNz" || got.scheme != "Basic" {
		t.Errorf("basic auth = %+v, want {token: dXNlcjpwYXNz, scheme: Basic}", got)
	}
	if got := rc.auths["//token.registry.com/"]; got.token != "my-bearer-token" || got.scheme != "Bearer" {
		t.Errorf("bearer auth = %+v, want {token: my-bearer-token, scheme: Bearer}", got)
	}
}

func TestParseNpmrcWhitespaceAroundEquals(t *testing.T) {
	input := `//registry.example.com/:_authToken = spaced-token
//basic.example.com/:_auth = base64value
`
	rc := parseNpmrc(input)

	if got := rc.auths["//registry.example.com/"]; got.token != "spaced-token" || got.scheme != "Bearer" {
		t.Errorf("spaced bearer = %+v, want {token: spaced-token, scheme: Bearer}", got)
	}
	if got := rc.auths["//basic.example.com/"]; got.token != "base64value" || got.scheme != "Basic" {
		t.Errorf("spaced basic = %+v, want {token: base64value, scheme: Basic}", got)
	}
}

func TestParseNpmrcEmpty(t *testing.T) {
	rc := parseNpmrc("")
	if len(rc.auths) != 0 {
		t.Errorf("expected no auths, got %v", rc.auths)
	}
	if rc.registry != "" {
		t.Errorf("expected empty registry, got %q", rc.registry)
	}
}

func TestParseNpmrcUnsetEnvVar(t *testing.T) {
	rc := parseNpmrc("//host/:_authToken=${UNSET_VAR_99999}")
	if got := rc.auths["//host/"]; got.token != "" {
		t.Errorf("unset env var token = %q, want empty", got.token)
	}
}

func TestParseNpmrcUsernamePassword(t *testing.T) {
	b64Pass := base64.StdEncoding.EncodeToString([]byte("s3cret"))
	input := "//private.registry.com/:username=myuser\n//private.registry.com/:_password=" + b64Pass + "\n"
	rc := parseNpmrc(input)

	got := rc.auths["//private.registry.com/"]
	wantToken := base64.StdEncoding.EncodeToString([]byte("myuser:s3cret"))
	if got.token != wantToken || got.scheme != "Basic" {
		t.Errorf("username/password auth = %+v, want {token: %s, scheme: Basic}", got, wantToken)
	}
}

func TestParseNpmrcUsernamePasswordSkippedWhenAuthTokenExists(t *testing.T) {
	b64Pass := base64.StdEncoding.EncodeToString([]byte("pass"))
	input := "//host.com/:_authToken=my-token\n//host.com/:username=user\n//host.com/:_password=" + b64Pass + "\n"
	rc := parseNpmrc(input)

	got := rc.auths["//host.com/"]
	if got.token != "my-token" || got.scheme != "Bearer" {
		t.Errorf("should prefer _authToken, got %+v", got)
	}
}

func TestParseNpmrcRegistryOnly(t *testing.T) {
	rc := parseNpmrc("registry=https://my.registry.io")
	if rc.registry != "https://my.registry.io" {
		t.Errorf("registry = %q, want %q", rc.registry, "https://my.registry.io")
	}
	if len(rc.auths) != 0 {
		t.Errorf("expected no auths, got %v", rc.auths)
	}
}

func TestParseNpmrcUsernameWithoutPassword(t *testing.T) {
	rc := parseNpmrc("//host.com/:username=lonely-user\n")
	if len(rc.auths) != 0 {
		t.Errorf("username without password should produce no auth, got %v", rc.auths)
	}
}

func TestParseNpmrcPasswordWithoutUsername(t *testing.T) {
	b64Pass := base64.StdEncoding.EncodeToString([]byte("orphan"))
	rc := parseNpmrc("//host.com/:_password=" + b64Pass + "\n")
	if len(rc.auths) != 0 {
		t.Errorf("password without username should produce no auth, got %v", rc.auths)
	}
}

func TestParseNpmrcInvalidBase64Password(t *testing.T) {
	rc := parseNpmrc("//host.com/:username=user\n//host.com/:_password=%%%not-base64\n")
	if len(rc.auths) != 0 {
		t.Errorf("invalid base64 password should be skipped, got %v", rc.auths)
	}
}

func TestParseNpmrcBasicAuthOverridesUsernamePassword(t *testing.T) {
	b64Pass := base64.StdEncoding.EncodeToString([]byte("pass"))
	input := "//host.com/:_auth=direct-basic\n//host.com/:username=user\n//host.com/:_password=" + b64Pass + "\n"
	rc := parseNpmrc(input)

	got := rc.auths["//host.com/"]
	if got.token != "direct-basic" || got.scheme != "Basic" {
		t.Errorf("should prefer _auth, got %+v", got)
	}
}

func TestParseNpmrcRegistryWithTrailingSlash(t *testing.T) {
	rc := parseNpmrc("registry=https://my.registry.io/")
	if rc.registry != "https://my.registry.io/" {
		t.Errorf("registry = %q, want %q", rc.registry, "https://my.registry.io/")
	}
}

func TestParseNpmrcRegistryWithWhitespace(t *testing.T) {
	rc := parseNpmrc("registry = https://my.registry.io")
	if rc.registry != "https://my.registry.io" {
		t.Errorf("registry = %q, want %q", rc.registry, "https://my.registry.io")
	}
}

func TestFindAuth(t *testing.T) {
	auths := map[string]Auth{
		"//registry.example.com/": {token: "host-token", scheme: "Bearer"},
	}
	got := findAuth("https://registry.example.com", auths)
	if got.token != "host-token" || got.scheme != "Bearer" {
		t.Errorf("host match = %+v, want {token: host-token, scheme: Bearer}", got)
	}
}

func TestFindAuthPathMatch(t *testing.T) {
	auths := map[string]Auth{
		"//registry.example.com/custom/path/": {token: "path-token", scheme: "Bearer"},
		"//registry.example.com/":             {token: "host-token", scheme: "Bearer"},
	}
	// should match the longer path first
	got := findAuth("https://registry.example.com/custom/path", auths)
	if got.token != "path-token" {
		t.Errorf("path match = %+v, want path-token", got)
	}
	// different path should fall back to host
	got = findAuth("https://registry.example.com/other", auths)
	if got.token != "host-token" {
		t.Errorf("fallback match = %+v, want host-token", got)
	}
}

func TestFindAuthNoMatch(t *testing.T) {
	auths := map[string]Auth{
		"//other.host/": {token: "other-token", scheme: "Bearer"},
	}
	got := findAuth("https://registry.example.com", auths)
	if got.token != "" {
		t.Errorf("no match should return empty, got %+v", got)
	}
}

func TestFindAuthWithPort(t *testing.T) {
	auths := map[string]Auth{
		"//registry.example.com:8080/": {token: "port-token", scheme: "Bearer"},
	}
	got := findAuth("https://registry.example.com:8080", auths)
	if got.token != "port-token" {
		t.Errorf("port match = %+v, want port-token", got)
	}
}

func TestFindAuthRegistryWithTrailingSlash(t *testing.T) {
	auths := map[string]Auth{
		"//registry.example.com/": {token: "slash-token", scheme: "Bearer"},
	}
	got := findAuth("https://registry.example.com/", auths)
	if got.token != "slash-token" {
		t.Errorf("trailing slash = %+v, want slash-token", got)
	}
}

func TestFindAuthEmptyAuths(t *testing.T) {
	got := findAuth("https://registry.example.com", map[string]Auth{})
	if got.token != "" {
		t.Errorf("empty auths should return empty, got %+v", got)
	}
}

func TestLoadRegistryDefaults(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("NPM_REGISTRY", "")

	origDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(origDir)

	reg := LoadRegistry()
	if reg.url != "https://registry.npmjs.org" {
		t.Errorf("default registry = %q, want https://registry.npmjs.org", reg.url)
	}
	if reg.auth.token != "" {
		t.Errorf("default auth should be empty, got %+v", reg.auth)
	}
}

func TestLoadRegistryNpmRegistryEnvVar(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("NPM_REGISTRY", "https://env.registry.io")

	// write an .npmrc with a different registry to confirm env wins
	os.WriteFile(filepath.Join(tmp, ".npmrc"), []byte("registry=https://file.registry.io\n//env.registry.io/:_authToken=env-tok\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(origDir)

	reg := LoadRegistry()
	if reg.url != "https://env.registry.io" {
		t.Errorf("env registry = %q, want https://env.registry.io", reg.url)
	}
	if reg.auth.token != "env-tok" {
		t.Errorf("env auth token = %q, want env-tok", reg.auth.token)
	}
}
