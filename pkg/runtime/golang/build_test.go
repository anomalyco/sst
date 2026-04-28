package golang

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testIntegrationEnv returns a minimal, hermetic env for `go list`.
// GOTOOLCHAIN=local prevents network downloads of mismatched
// toolchains, GOPROXY=off prevents module fetches, GOMODCACHE/GOCACHE
// are redirected so the host caches stay untouched.
func testIntegrationEnv(t *testing.T) []string {
	t.Helper()
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + t.TempDir(),
		"GOMODCACHE=" + filepath.Join(t.TempDir(), "modcache"),
		"GOCACHE=" + filepath.Join(t.TempDir(), "gocache"),
		"GOPROXY=off",
		"GOFLAGS=-mod=mod",
		"GOTOOLCHAIN=local",
	}
}

// requireGoToolchain skips when `go test -short` is set or when the
// `go` binary is not in PATH.
func requireGoToolchain(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test; skipped under -short")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go toolchain not available: %v", err)
	}
}

func TestCaptureDeps_SimpleMain(t *testing.T) {
	requireGoToolchain(t)

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test\n\ngo 1.22\n")
	main := filepath.Join(dir, "main.go")
	mustWriteFile(t, main, "package main\n\nfunc main() {}\n")

	r := New()
	files, err := r.captureDeps(context.Background(), dir, ".", testIntegrationEnv(t))
	if err != nil {
		t.Fatalf("captureDeps: %v", err)
	}

	if _, ok := files[main]; !ok {
		t.Errorf("expected main.go (%s) in captured files, got %v", main, keys(files))
	}
	for f := range files {
		if !strings.HasPrefix(f, dir) {
			t.Errorf("captured file outside module root: %s (root=%s)", f, dir)
		}
	}
}

func TestCaptureDeps_LocalDependency(t *testing.T) {
	requireGoToolchain(t)

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test\n\ngo 1.22\n")

	mustMkdirAll(t, filepath.Join(dir, "shared"))
	sharedFile := filepath.Join(dir, "shared", "lib.go")
	mustWriteFile(t, sharedFile, "package shared\n\nfunc Hello() string { return \"hi\" }\n")

	mainFile := filepath.Join(dir, "main.go")
	mustWriteFile(t, mainFile,
		"package main\n\nimport \"example.test/shared\"\n\nfunc main() { _ = shared.Hello() }\n")

	r := New()
	files, err := r.captureDeps(context.Background(), dir, ".", testIntegrationEnv(t))
	if err != nil {
		t.Fatalf("captureDeps: %v", err)
	}

	if _, ok := files[mainFile]; !ok {
		t.Errorf("expected main.go (%s) in captured files, got %v", mainFile, keys(files))
	}
	if _, ok := files[sharedFile]; !ok {
		t.Errorf("expected shared/lib.go (%s) in captured files, got %v", sharedFile, keys(files))
	}
}

func TestCaptureDeps_NoFilesOutsideModuleRoot(t *testing.T) {
	requireGoToolchain(t)

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test\n\ngo 1.22\n")
	mustWriteFile(t, filepath.Join(dir, "main.go"),
		"package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hi\") }\n")

	r := New()
	files, err := r.captureDeps(context.Background(), dir, ".", testIntegrationEnv(t))
	if err != nil {
		t.Fatalf("captureDeps: %v", err)
	}

	for f := range files {
		if !strings.HasPrefix(f, dir) {
			t.Errorf("file outside module root leaked into captured set: %s", f)
		}
	}
}

func TestCaptureDeps_NoTestFilesIncluded(t *testing.T) {
	requireGoToolchain(t)

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test\n\ngo 1.22\n")
	mustWriteFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")
	testFile := filepath.Join(dir, "main_test.go")
	mustWriteFile(t, testFile, "package main\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}\n")

	r := New()
	files, err := r.captureDeps(context.Background(), dir, ".", testIntegrationEnv(t))
	if err != nil {
		t.Fatalf("captureDeps: %v", err)
	}

	if _, ok := files[testFile]; ok {
		t.Errorf("test file %s leaked into captured set", testFile)
	}
}

func TestCaptureDeps_BrokenSourceFails(t *testing.T) {
	requireGoToolchain(t)

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test\n\ngo 1.22\n")
	mustWriteFile(t, filepath.Join(dir, "main.go"), "this is not valid go source")

	r := New()
	if _, err := r.captureDeps(context.Background(), dir, ".", testIntegrationEnv(t)); err == nil {
		t.Error("expected error on unparseable source, got nil")
	}
}

func TestCaptureDeps_SubPackageHandler(t *testing.T) {
	requireGoToolchain(t)

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "go.mod"), "module example.test\n\ngo 1.22\n")

	mustMkdirAll(t, filepath.Join(dir, "shared"))
	sharedFile := filepath.Join(dir, "shared", "lib.go")
	mustWriteFile(t, sharedFile, "package shared\n\nfunc Hello() string { return \"hi\" }\n")

	mustMkdirAll(t, filepath.Join(dir, "lambdas", "alpha"))
	alphaFile := filepath.Join(dir, "lambdas", "alpha", "main.go")
	mustWriteFile(t, alphaFile,
		"package main\n\nimport \"example.test/shared\"\n\nfunc main() { _ = shared.Hello() }\n")

	mustMkdirAll(t, filepath.Join(dir, "lambdas", "beta"))
	betaFile := filepath.Join(dir, "lambdas", "beta", "main.go")
	mustWriteFile(t, betaFile, "package main\n\nfunc main() {}\n")

	r := New()
	files, err := r.captureDeps(context.Background(), dir, "lambdas/alpha", testIntegrationEnv(t))
	if err != nil {
		t.Fatalf("captureDeps: %v", err)
	}

	if _, ok := files[alphaFile]; !ok {
		t.Errorf("expected alpha main.go in captured files, got %v", keys(files))
	}
	if _, ok := files[sharedFile]; !ok {
		t.Errorf("expected shared/lib.go in captured files, got %v", keys(files))
	}
	if _, ok := files[betaFile]; ok {
		t.Errorf("beta sibling main.go must NOT be in alpha's import graph: %s", betaFile)
	}
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
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
