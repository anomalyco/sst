package node

import (
	"encoding/json"
	"testing"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

func TestNodePropertiesUnmarshal(t *testing.T) {
	payload := `{
		"loader": {".png": "file"},
		"install": ["sharp"],
		"minify": true,
		"splitting": false,
		"esbuild": {
			"target": "es2022",
			"sourcemap": "linked",
			"keepNames": false,
			"define": {"process.env.NODE_ENV": "\"production\""},
			"banner": {"js": "// banner"},
			"external": ["aws-sdk"],
			"nodePaths": ["/custom/path"],
			"mainFields": ["main"],
			"conditions": ["esbuild", "node"]
		}
	}`

	var props NodeProperties
	if err := json.Unmarshal([]byte(payload), &props); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got := props.ESBuild.ResolveTarget(esbuild.ESNext); got != esbuild.ES2022 {
		t.Errorf("Target = %v, want ES2022", got)
	}
	if got := props.ESBuild.ResolveSourcemap(esbuild.SourceMapNone); got != esbuild.SourceMapLinked {
		t.Errorf("Sourcemap = %v, want SourceMapLinked", got)
	}
	if got := props.ESBuild.ResolveKeepNames(true); got != false {
		t.Errorf("KeepNames = %v, want false", got)
	}
	if got := props.ESBuild.Define["process.env.NODE_ENV"]; got != `"production"` {
		t.Errorf("Define = %v, want \"production\"", got)
	}
	if got := props.ESBuild.Banner["js"]; got != "// banner" {
		t.Errorf("Banner = %v, want // banner", got)
	}
	if len(props.ESBuild.External) != 1 || props.ESBuild.External[0] != "aws-sdk" {
		t.Errorf("External = %v, want [aws-sdk]", props.ESBuild.External)
	}
	if len(props.ESBuild.NodePaths) != 1 || props.ESBuild.NodePaths[0] != "/custom/path" {
		t.Errorf("NodePaths = %v, want [/custom/path]", props.ESBuild.NodePaths)
	}
	if got := props.ESBuild.ResolveMainFields([]string{"module", "main"}); len(got) != 1 || got[0] != "main" {
		t.Errorf("MainFields = %v, want [main]", got)
	}
	if got := props.ESBuild.ResolveConditions([]string{"workerd"}); len(got) != 2 || got[0] != "esbuild" || got[1] != "node" {
		t.Errorf("Conditions = %v, want [esbuild node]", got)
	}
}

func TestResolveTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		fallback esbuild.Target
		want     esbuild.Target
	}{
		{"empty uses fallback", "", esbuild.ESNext, esbuild.ESNext},
		{"esnext", "esnext", esbuild.ES2020, esbuild.ESNext},
		{"es2020", "es2020", esbuild.ESNext, esbuild.ES2020},
		{"es6 alias", "es6", esbuild.ESNext, esbuild.ES2015},
		{"case insensitive", "ESNext", esbuild.ES2020, esbuild.ESNext},
		{"unknown uses fallback", "es9999", esbuild.ES2023, esbuild.ES2023},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &ESBuildOptions{Target: tt.target}
			if got := o.ResolveTarget(tt.fallback); got != tt.want {
				t.Errorf("ResolveTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveSourcemap(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		fallback esbuild.SourceMap
		want     esbuild.SourceMap
	}{
		{"empty uses fallback", "", esbuild.SourceMapLinked, esbuild.SourceMapLinked},
		{"linked string", `"linked"`, esbuild.SourceMapNone, esbuild.SourceMapLinked},
		{"inline string", `"inline"`, esbuild.SourceMapNone, esbuild.SourceMapInline},
		{"external string", `"external"`, esbuild.SourceMapNone, esbuild.SourceMapExternal},
		{"both string", `"both"`, esbuild.SourceMapNone, esbuild.SourceMapInlineAndExternal},
		{"case insensitive", `"Linked"`, esbuild.SourceMapNone, esbuild.SourceMapLinked},
		{"bool true", `true`, esbuild.SourceMapNone, esbuild.SourceMapLinked},
		{"bool false disables", `false`, esbuild.SourceMapLinked, esbuild.SourceMapNone},
		{"bool false ignores fallback", `false`, esbuild.SourceMapNone, esbuild.SourceMapNone},
		{"unknown string uses fallback", `"bogus"`, esbuild.SourceMapLinked, esbuild.SourceMapLinked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &ESBuildOptions{}
			if tt.raw != "" {
				o.Sourcemap = json.RawMessage(tt.raw)
			}
			if got := o.ResolveSourcemap(tt.fallback); got != tt.want {
				t.Errorf("ResolveSourcemap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveKeepNames(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }
	tests := []struct {
		name     string
		keep     *bool
		fallback bool
		want     bool
	}{
		{"nil uses fallback true", nil, true, true},
		{"nil uses fallback false", nil, false, false},
		{"explicit true", boolPtr(true), false, true},
		{"explicit false", boolPtr(false), true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &ESBuildOptions{KeepNames: tt.keep}
			if got := o.ResolveKeepNames(tt.fallback); got != tt.want {
				t.Errorf("ResolveKeepNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveMainFields(t *testing.T) {
	fallback := []string{"module", "main"}
	tests := []struct {
		name string
		val  []string
		want []string
	}{
		{"nil uses fallback", nil, fallback},
		{"empty uses fallback", []string{}, fallback},
		{"override", []string{"main"}, []string{"main"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &ESBuildOptions{MainFields: tt.val}
			got := o.ResolveMainFields(fallback)
			if len(got) != len(tt.want) {
				t.Errorf("ResolveMainFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveConditions(t *testing.T) {
	fallback := []string{"workerd", "worker", "browser"}
	tests := []struct {
		name string
		val  []string
		want []string
	}{
		{"nil uses fallback", nil, fallback},
		{"empty uses fallback", []string{}, fallback},
		{"override", []string{"esbuild", "node"}, []string{"esbuild", "node"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &ESBuildOptions{Conditions: tt.val}
			got := o.ResolveConditions(fallback)
			if len(got) != len(tt.want) {
				t.Errorf("ResolveConditions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodePropertiesUnmarshalBoolSourcemap(t *testing.T) {
	payload := `{"esbuild": {"sourcemap": true}}`
	var props NodeProperties
	json.Unmarshal([]byte(payload), &props)

	if got := props.ESBuild.ResolveSourcemap(esbuild.SourceMapNone); got != esbuild.SourceMapLinked {
		t.Errorf("Sourcemap with bool true = %v, want SourceMapLinked", got)
	}
}

func TestNodePropertiesUnmarshalBoolSourcemapFalse(t *testing.T) {
	payload := `{"esbuild": {"sourcemap": false}}`
	var props NodeProperties
	json.Unmarshal([]byte(payload), &props)

	if got := props.ESBuild.ResolveSourcemap(esbuild.SourceMapLinked); got != esbuild.SourceMapNone {
		t.Errorf("Sourcemap with bool false = %v, want SourceMapNone", got)
	}
}

func TestNodePropertiesUnmarshalEmpty(t *testing.T) {
	payload := `{}`
	var props NodeProperties
	json.Unmarshal([]byte(payload), &props)

	if got := props.ESBuild.ResolveTarget(esbuild.ESNext); got != esbuild.ESNext {
		t.Errorf("Target = %v, want ESNext fallback", got)
	}
	if got := props.ESBuild.ResolveSourcemap(esbuild.SourceMapLinked); got != esbuild.SourceMapLinked {
		t.Errorf("Sourcemap = %v, want SourceMapLinked fallback", got)
	}
	if got := props.ESBuild.ResolveKeepNames(true); got != true {
		t.Errorf("KeepNames = %v, want true fallback", got)
	}
	if got := props.ESBuild.ResolveMainFields([]string{"module", "main"}); len(got) != 2 {
		t.Errorf("MainFields = %v, want fallback [module main]", got)
	}
	if got := props.ESBuild.ResolveConditions([]string{"workerd"}); len(got) != 1 || got[0] != "workerd" {
		t.Errorf("Conditions = %v, want fallback [workerd]", got)
	}
}
