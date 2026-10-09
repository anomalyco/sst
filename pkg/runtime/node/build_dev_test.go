package node

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

func devBuildInput(t *testing.T, dir string, name string) *runtime.BuildInput {
	t.Helper()
	file := filepath.Join(dir, name+".ts")
	err := os.WriteFile(file, []byte("export const handler = async () => \""+name+"\";\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return &runtime.BuildInput{
		CfgPath:    filepath.Join(dir, "sst.config.ts"),
		Dev:        true,
		FunctionID: name,
		Handler:    name + ".handler",
		Runtime:    "nodejs20.x",
		Properties: json.RawMessage(`{}`),
	}
}

func TestDevBuildContextsAreBounded(t *testing.T) {
	dir := t.TempDir()
	r := New("dev")
	r.contextCap = 2

	for i := 0; i < 5; i++ {
		input := devBuildInput(t, dir, fmt.Sprintf("fn%d", i))
		output, err := r.Build(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if len(output.Errors) > 0 {
			t.Fatal(output.Errors)
		}
	}

	r.contextsLock.Lock()
	got := len(r.contexts)
	r.contextsLock.Unlock()
	if got > 2 {
		t.Errorf("live esbuild contexts = %d, want <= 2", got)
	}
}

func TestDevBuildShouldRebuildTracksInputs(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := New("dev")
	r.contextCap = 1

	first := devBuildInput(t, dir, "first")
	second := devBuildInput(t, dir, "second")
	for _, input := range []*runtime.BuildInput{first, second} {
		output, err := r.Build(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if len(output.Errors) > 0 {
			t.Fatal(output.Errors)
		}
	}

	// dependency tracking must survive context eviction
	if !r.ShouldRebuild("first", filepath.Join(dir, "first.ts")) {
		t.Error("expected ShouldRebuild to be true for first.ts after eviction")
	}
	if !r.ShouldRebuild("second", filepath.Join(dir, "second.ts")) {
		t.Error("expected ShouldRebuild to be true for second.ts")
	}
	if r.ShouldRebuild("first", filepath.Join(dir, "second.ts")) {
		t.Error("expected ShouldRebuild to be false for unrelated file")
	}

	// rebuilding an evicted function must still work
	output, buildErr := r.Build(context.Background(), first)
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	if len(output.Errors) > 0 {
		t.Fatal(output.Errors)
	}
}
