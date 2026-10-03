package node

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/js"
	"github.com/sst/sst/v3/pkg/runtime"
)

func TestBuildTarget(t *testing.T) {
	const resource = `export function handler() {
  using resource = { [Symbol.dispose]() {} };
  return resource;
}`
	tests := []struct {
		name    string
		runtime string
		target  string
		source  string
		syntax  string
		retain  bool
	}{
		{"node26 native using", "nodejs26.x", "", resource, "using resource", true},
		{"node26 lowers accessor", "nodejs26.x", "", `export class Handler { accessor value = 1 }`, "accessor value", false},
		{"node22 lowers using", "nodejs22.x", "", resource, "using resource", false},
		{"explicit esnext overrides node22", "nodejs22.x", "ESNext", resource, "using resource", true},
		{"explicit es2022 overrides node26", "nodejs26.x", "es2022", resource, "using resource", false},
		{"unknown target uses node26", "nodejs26.x", "unknown", resource, "using resource", true},
		{"unknown target uses node22", "nodejs22.x", "unknown", resource, "using resource", false},
		{"node12 lowers optional chaining", "nodejs12.x", "", `export const handler = (event) => event?.value`, "?.", false},
	}
	for _, format := range []string{"esm", "cjs"} {
		for _, tt := range tests {
			t.Run(format+"/"+tt.name, func(t *testing.T) {
				code := buildTargetFixture(t, tt.runtime, format, tt.target, tt.source)
				if retained := strings.Contains(code, tt.syntax); retained != tt.retain {
					t.Fatalf("retained syntax %q = %v, want %v:\n%s", tt.syntax, retained, tt.retain, code)
				}
			})
		}
	}
	t.Run("esm/node20 top level await", func(t *testing.T) {
		code := buildTargetFixture(t, "nodejs20.x", "esm", "", `export const handler = await Promise.resolve(() => 42)`)
		if !strings.Contains(code, "await Promise.resolve") {
			t.Fatalf("top level await missing:\n%s", code)
		}
	})
}

func buildTargetFixture(t *testing.T, nodeRuntime, format, target, source string) string {
	t.Helper()
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "handler.js"), source)
	properties, err := json.Marshal(NodeProperties{
		Format:  format,
		ESBuild: ESBuildOptions{Target: target},
	})
	if err != nil {
		t.Fatal(err)
	}
	input := &runtime.BuildInput{
		CfgPath:    filepath.Join(dir, "sst.config.ts"),
		FunctionID: "target",
		Handler:    filepath.Join(dir, "handler.handler"),
		Runtime:    nodeRuntime,
		Properties: properties,
	}
	output, err := New("dev").Build(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Errors) > 0 {
		t.Fatalf("build errors: %v", output.Errors)
	}
	extension := ".mjs"
	if format == "cjs" {
		extension = ".cjs"
	}
	code, err := os.ReadFile(filepath.Join(input.Out(), "bundle"+extension))
	if err != nil {
		t.Fatal(err)
	}
	return string(code)
}

func TestResolveInstallVersion(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		install map[string]string
		setup   func(t *testing.T) (string, js.PackageJson)
		want    string
		wantErr string
	}{
		{
			name:    "explicit version overrides package json",
			pkg:     "sharp",
			install: map[string]string{"sharp": "0.33.5"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				return "", js.PackageJson{Dependencies: map[string]string{"sharp": "0.32.6"}}
			},
			want: "0.33.5",
		},
		{
			name:    "wildcard falls back to package json",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				return "", js.PackageJson{Dependencies: map[string]string{"sharp": "0.32.6"}}
			},
			want: "0.32.6",
		},
		{
			name:    "missing package falls back to wildcard",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				return "", js.PackageJson{}
			},
			want: "*",
		},
		{
			name:    "catalog spec resolves from pnpm workspace",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				dir := t.TempDir()
				pkgDir := filepath.Join(dir, "packages", "functions")
				mustMkdirAll(t, pkgDir)
				mustWriteFile(t, filepath.Join(dir, "pnpm-workspace.yaml"), "catalogs:\n  shared:\n    sharp: 0.32.6\n")
				return pkgDir, js.PackageJson{Dependencies: map[string]string{"sharp": "catalog:shared"}}
			},
			want: "0.32.6",
		},
		{
			name:    "explicit catalog spec resolves from pnpm workspace",
			pkg:     "sharp",
			install: map[string]string{"sharp": "catalog:shared"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				dir := t.TempDir()
				pkgDir := filepath.Join(dir, "packages", "functions")
				mustMkdirAll(t, pkgDir)
				mustWriteFile(t, filepath.Join(dir, "pnpm-workspace.yaml"), "catalogs:\n  shared:\n    sharp: 0.32.6\n")
				return pkgDir, js.PackageJson{}
			},
			want: "0.32.6",
		},
		{
			name:    "missing catalog entry returns error",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				dir := t.TempDir()
				pkgDir := filepath.Join(dir, "packages", "functions")
				mustMkdirAll(t, pkgDir)
				mustWriteFile(t, filepath.Join(dir, "pnpm-workspace.yaml"), "catalogs:\n  shared:\n    other: 1.0.0\n")
				return pkgDir, js.PackageJson{Dependencies: map[string]string{"sharp": "catalog:shared"}}
			},
			wantErr: "no matching catalog entry exists",
		},
		{
			name:    "catalog spec resolves from bun top level catalog",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				dir := t.TempDir()
				pkgDir := filepath.Join(dir, "packages", "functions")
				mustMkdirAll(t, pkgDir)
				mustWriteFile(t, filepath.Join(dir, "package.json"), `{
	"catalog": {
		"sharp": "0.32.6"
	}
}`)
				return pkgDir, js.PackageJson{Dependencies: map[string]string{"sharp": "catalog:"}}
			},
			want: "0.32.6",
		},
		{
			name:    "explicit catalog spec resolves from bun top level catalogs",
			pkg:     "sharp",
			install: map[string]string{"sharp": "catalog:shared"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				dir := t.TempDir()
				pkgDir := filepath.Join(dir, "packages", "functions")
				mustMkdirAll(t, pkgDir)
				mustWriteFile(t, filepath.Join(dir, "package.json"), `{
	"catalogs": {
		"shared": {
			"sharp": "0.32.6"
		}
	}
}`)
				return pkgDir, js.PackageJson{}
			},
			want: "0.32.6",
		},
		{
			name:    "catalog spec resolves from bun workspaces catalog",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				dir := t.TempDir()
				pkgDir := filepath.Join(dir, "packages", "functions")
				mustMkdirAll(t, pkgDir)
				mustWriteFile(t, filepath.Join(dir, "package.json"), `{
	"workspaces": {
		"packages": ["packages/*"],
		"catalog": {
			"sharp": "0.32.6"
		}
	}
}`)
				return pkgDir, js.PackageJson{Dependencies: map[string]string{"sharp": "catalog:"}}
			},
			want: "0.32.6",
		},
		{
			name:    "explicit catalog spec resolves from bun workspaces catalogs",
			pkg:     "sharp",
			install: map[string]string{"sharp": "catalog:shared"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				dir := t.TempDir()
				pkgDir := filepath.Join(dir, "packages", "functions")
				mustMkdirAll(t, pkgDir)
				mustWriteFile(t, filepath.Join(dir, "package.json"), `{
	"workspaces": {
		"packages": ["packages/*"],
		"catalogs": {
			"shared": {
				"sharp": "0.32.6"
			}
		}
	}
}`)
				return pkgDir, js.PackageJson{}
			},
			want: "0.32.6",
		},
		{
			name:    "catalog spec resolves from bun top level catalog with workspaces array",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				dir := t.TempDir()
				pkgDir := filepath.Join(dir, "packages", "functions")
				mustMkdirAll(t, pkgDir)
				mustWriteFile(t, filepath.Join(dir, "package.json"), `{
	"workspaces": ["packages/*"],
	"catalog": {
		"sharp": "0.32.6"
	}
}`)
				return pkgDir, js.PackageJson{Dependencies: map[string]string{"sharp": "catalog:"}}
			},
			want: "0.32.6",
		},
		{
			name:    "missing bun catalog entry returns error",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				dir := t.TempDir()
				pkgDir := filepath.Join(dir, "packages", "functions")
				mustMkdirAll(t, pkgDir)
				mustWriteFile(t, filepath.Join(dir, "package.json"), `{
	"catalogs": {
		"shared": {
			"other": "1.0.0"
		}
	}
}`)
				return pkgDir, js.PackageJson{Dependencies: map[string]string{"sharp": "catalog:shared"}}
			},
			wantErr: "no matching catalog entry exists in",
		},
		{
			name:    "workspace spec returns error",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				return "", js.PackageJson{Dependencies: map[string]string{"sharp": "workspace:^"}}
			},
			wantErr: "found \"workspace:^\" using \"workspace:\"",
		},
		{
			name:    "explicit workspace spec returns error",
			pkg:     "sharp",
			install: map[string]string{"sharp": "workspace:^"},
			setup: func(t *testing.T) (string, js.PackageJson) {
				return "", js.PackageJson{}
			},
			wantErr: "found \"workspace:^\" using \"workspace:\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, packageJSON := tt.setup(t)
			got, err := resolveInstallVersion(tt.pkg, tt.install, dir, packageJSON)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("resolveInstallVersion() error = nil, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("resolveInstallVersion() error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveInstallVersion() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveInstallVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
