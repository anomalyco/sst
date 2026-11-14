package worker

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

func TestStripAliasedExternals(t *testing.T) {
	original := map[string]string{
		"http":      "unenv/node/http",
		"node:http": "unenv/node/http",
		"buffer":    "buffer",
	}

	got := stripAliasedExternals(original, []string{"http", "node:http", "node:*"})

	if _, ok := got["http"]; ok {
		t.Fatalf("http alias should be removed")
	}
	if _, ok := got["node:http"]; ok {
		t.Fatalf("node:http alias should be removed")
	}
	if got["buffer"] != "buffer" {
		t.Fatalf("buffer alias = %q, want %q", got["buffer"], "buffer")
	}
	if original["http"] != "unenv/node/http" {
		t.Fatalf("original alias map was mutated")
	}
}

func TestUnenvAliasInteropPlugin(t *testing.T) {
	dir := t.TempDir()
	shimDir := filepath.Join(dir, "node_modules", "unenv", "npm")
	if err := os.MkdirAll(shimDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shimDir, "inherits.js"), []byte(`
export default function inherits() {}
export const named = "named export"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node_modules", "unenv", "package.json"), []byte(`{"type":"module","exports":{"./npm/inherits":"./npm/inherits.js"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	alias := map[string]string{"inherits": "unenv/npm/inherits"}
	result := esbuild.Build(esbuild.BuildOptions{
		Stdin: &esbuild.StdinOptions{
			Contents: `
import inheritsImport from "inherits"
const inherits = require("inherits")
if (typeof inheritsImport !== "function") throw new Error("ESM alias behavior changed")
if (typeof inherits !== "function") throw new Error("default export was not a function")
if (inherits.named !== "named export") throw new Error("named export was not copied")
`,
			ResolveDir: dir,
		},
		Alias:     alias,
		Bundle:    true,
		Format:    esbuild.FormatCommonJS,
		NodePaths: []string{filepath.Join(dir, "node_modules")},
		Plugins:   []esbuild.Plugin{unenvAliasInteropPlugin(alias, dir)},
		Write:     false,
	})
	if len(result.Errors) > 0 {
		t.Fatalf("bundle failed: %v", result.Errors)
	}

	cmd := exec.Command("node", "-")
	cmd.Stdin = strings.NewReader(string(result.OutputFiles[0].Contents))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bundle execution failed: %v\n%s", err, output)
	}
}
