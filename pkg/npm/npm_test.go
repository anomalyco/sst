package npm

import (
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

	wantTokens := map[string]string{
		"registry.npmjs.org":  "hardcoded-token",
		"custom.registry.com": "env-token",
		"another.host":        "plain",
	}
	for host, want := range wantTokens {
		if got := rc.tokens[host]; got != want {
			t.Errorf("tokens[%q] = %q, want %q", host, got, want)
		}
	}
	if len(rc.tokens) != len(wantTokens) {
		t.Errorf("len(tokens) = %d, want %d", len(rc.tokens), len(wantTokens))
	}
}

func TestParseNpmrcEmpty(t *testing.T) {
	rc := parseNpmrc("")
	if len(rc.tokens) != 0 {
		t.Errorf("expected no tokens, got %v", rc.tokens)
	}
	if rc.registry != "" {
		t.Errorf("expected empty registry, got %q", rc.registry)
	}
}

func TestParseNpmrcUnsetEnvVar(t *testing.T) {
	rc := parseNpmrc("//host/:_authToken=${UNSET_VAR_99999}")
	if got := rc.tokens["host"]; got != "" {
		t.Errorf("unset env var token = %q, want empty", got)
	}
}

func TestParseNpmrcRegistryOnly(t *testing.T) {
	rc := parseNpmrc("registry=https://my.registry.io")
	if rc.registry != "https://my.registry.io" {
		t.Errorf("registry = %q, want %q", rc.registry, "https://my.registry.io")
	}
	if len(rc.tokens) != 0 {
		t.Errorf("expected no tokens, got %v", rc.tokens)
	}
}
