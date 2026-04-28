package golang

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func absPath(parts ...string) string {
	return string(filepath.Separator) + filepath.Join(parts...)
}

func TestProperties_JSONFieldName(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{"arm64", `{"architecture": "arm64"}`, "arm64"},
		{"x86_64", `{"architecture": "x86_64"}`, "x86_64"},
		{"empty", `{}`, ""},
		{"extra fields ignored", `{"architecture": "arm64", "memory": 512}`, "arm64"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got Properties
			if err := json.Unmarshal([]byte(tt.payload), &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got.Architecture != tt.want {
				t.Errorf("Architecture = %q, want %q", got.Architecture, tt.want)
			}
		})
	}
}

func TestGoarchFromArchitecture(t *testing.T) {
	tests := []struct {
		arch string
		want string
	}{
		{"arm64", "arm64"},
		{"x86_64", "amd64"},
		{"", "amd64"},
		{"unknown-arch", "amd64"},
	}
	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			t.Parallel()
			if got := goarchFromArchitecture(tt.arch); got != tt.want {
				t.Errorf("goarchFromArchitecture(%q) = %q, want %q", tt.arch, got, tt.want)
			}
		})
	}
}

func TestShouldRebuild_NonGoFile(t *testing.T) {
	t.Parallel()
	r := New()
	r.files["fn"] = map[string]struct{}{absPath("abs", "path", "handler.go"): {}}

	tests := []struct {
		name string
		file string
	}{
		{"txt", absPath("abs", "path", "handler.txt")},
		{"md", absPath("abs", "path", "README.md")},
		{"go.bak", absPath("abs", "path", "handler.go.bak")},
		{"no extension", absPath("abs", "path", "handler")},
		// HasSuffix is case-sensitive; ".GO" must NOT match ".go".
		{"uppercase extension", absPath("abs", "path", "handler.GO")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if r.ShouldRebuild("fn", tt.file) {
				t.Errorf("expected false for non-.go file %q", tt.file)
			}
		})
	}
}

func TestShouldRebuild_NoCapturedGraph(t *testing.T) {
	t.Parallel()
	r := New()
	if r.ShouldRebuild("fn", absPath("abs", "path", "handler.go")) {
		t.Error("expected false when no graph has been captured")
	}
}

func TestShouldRebuild_FileInGraph(t *testing.T) {
	t.Parallel()
	r := New()
	dir := t.TempDir()
	handler := filepath.Join(dir, "handler.go")
	r.files["fn"] = map[string]struct{}{handler: {}}

	if !r.ShouldRebuild("fn", handler) {
		t.Errorf("expected true for file in graph: %s", handler)
	}
}

func TestShouldRebuild_FileNotInGraph(t *testing.T) {
	t.Parallel()
	r := New()
	dir := t.TempDir()
	handler := filepath.Join(dir, "handler.go")
	other := filepath.Join(dir, "other.go")
	r.files["fn"] = map[string]struct{}{handler: {}}

	if r.ShouldRebuild("fn", other) {
		t.Errorf("expected false for file not in graph: %s", other)
	}
}

func TestShouldRebuild_RelativePathIsResolved(t *testing.T) {
	r := New()
	dir := t.TempDir()
	handler := filepath.Join(dir, "handler.go")
	mustWriteFile(t, handler, "package main\n")
	r.files["fn"] = map[string]struct{}{handler: {}}

	t.Chdir(dir)
	if !r.ShouldRebuild("fn", "handler.go") {
		t.Errorf("expected true for relative path \"handler.go\" with cwd=%s", dir)
	}
}

func TestRuntime_ConcurrentShouldRebuild(t *testing.T) {
	t.Parallel()
	r := New()
	dir := t.TempDir()
	handler := filepath.Join(dir, "handler.go")
	r.files["fn"] = map[string]struct{}{handler: {}}

	const goroutines = 50
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range 100 {
				r.ShouldRebuild("fn", handler)
			}
		}()
	}

	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			fn := "fn" + string(rune('A'+i%26))
			for range 50 {
				r.mut.Lock()
				r.files[fn] = map[string]struct{}{handler: {}}
				r.mut.Unlock()
			}
		}(i)
	}

	// Writer that hits the same key the readers query.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100 {
			r.mut.Lock()
			r.files["fn"] = map[string]struct{}{handler: {}}
			r.mut.Unlock()
		}
	}()

	wg.Wait()
}

func TestIsUnderDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		dir  string
		want bool
	}{
		{"descendant", absPath("a", "b", "c"), absPath("a"), true},
		{"direct child", absPath("a", "b", "c"), absPath("a", "b"), true},
		{"identity", absPath("a", "b", "c"), absPath("a", "b", "c"), true},
		{"deeper than dir", absPath("a", "b", "c"), absPath("a", "b", "c", "d"), false},
		{"unrelated", absPath("a", "b", "c"), absPath("x"), false},
		{"prefix-only must not match", absPath("a", "b", "c"), absPath("a", "b", "cd"), false},
		{"trailing separator", absPath("a", "b", "c") + string(filepath.Separator), absPath("a", "b"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isUnderDir(tt.path, tt.dir); got != tt.want {
				t.Errorf("isUnderDir(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
			}
		})
	}
}

func TestParseGoListOutput_FiltersStdlibAndGOMODCACHE(t *testing.T) {
	t.Parallel()
	gomodcache := absPath("home", "user", "go", "pkg", "mod")
	localDir := absPath("repo", "services")
	stdlibDir := absPath("usr", "local", "go", "src", "fmt")
	modcacheDir := filepath.Join(gomodcache, "github.com", "foo", "bar@v1.0.0")
	siblingDir := absPath("workspace", "shared-replace")

	stream := strings.Join([]string{
		`{"Standard": false, "Goroot": false, "Dir": ` + jsonStr(localDir) + `, "GoFiles": ["main.go"]}`,
		`{"Standard": true,  "Goroot": true,  "Dir": ` + jsonStr(stdlibDir) + `, "GoFiles": ["print.go"]}`,
		`{"Standard": false, "Goroot": false, "Dir": ` + jsonStr(modcacheDir) + `, "GoFiles": ["lib.go"]}`,
		`{"Standard": false, "Goroot": false, "Dir": ` + jsonStr(siblingDir) + `, "GoFiles": ["replaced.go"]}`,
	}, "\n")

	files, err := parseGoListOutput(strings.NewReader(stream), gomodcache)
	if err != nil {
		t.Fatalf("parseGoListOutput: %v", err)
	}

	wantKept := []string{
		filepath.Join(localDir, "main.go"),
		filepath.Join(siblingDir, "replaced.go"),
	}
	for _, want := range wantKept {
		if _, ok := files[want]; !ok {
			t.Errorf("expected %q in result, got %v", want, keys(files))
		}
	}
	for f := range files {
		if strings.HasPrefix(f, gomodcache) {
			t.Errorf("module-cache file leaked into result: %s", f)
		}
		if strings.HasPrefix(f, stdlibDir) {
			t.Errorf("stdlib file leaked into result: %s", f)
		}
	}
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
