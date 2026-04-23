package watcher

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWatchDefaultsToProjectRoot(t *testing.T) {
	root := t.TempDir()
	roots, ignore, err := resolveWatch(root, project.Watch{})
	require.NoError(t, err)
	assert.Equal(t, []string{root}, roots)
	assert.False(t, isIgnored(root, ignore, filepath.Join(root, "sst.config.ts")))
}

func TestResolveWatchResolvesExternalIncludeRoots(t *testing.T) {
	workspace := t.TempDir()
	root := filepath.Join(workspace, "app")
	external := filepath.Join(workspace, "external-package")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "packages", "api"), 0755))
	require.NoError(t, os.MkdirAll(external, 0755))

	roots, _, err := resolveWatch(root, project.Watch{
		Paths: []string{"packages/api", "../external-package"},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{filepath.Join(root, "packages", "api"), external}, roots)
}

func TestResolveWatchExpandsLegacyArrayGlobs(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "packages", "api"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "packages", "web"), 0755))

	var watch project.Watch
	require.NoError(t, json.Unmarshal([]byte(`["packages/*"]`), &watch))

	roots, _, err := resolveWatch(root, watch)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{filepath.Join(root, "packages", "api"), filepath.Join(root, "packages", "web")}, roots)
}

func TestResolveWatchExpandsLegacyArrayFileGlobsToParentDirs(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "index.ts"), []byte("export {}\n"), 0644))

	var watch project.Watch
	require.NoError(t, json.Unmarshal([]byte(`["src/*"]`), &watch))

	roots, _, err := resolveWatch(root, watch)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(root, "src")}, roots)
}

func TestResolveWatchMatchesIgnorePaths(t *testing.T) {
	workspace := t.TempDir()
	root := filepath.Join(workspace, "app")
	external := filepath.Join(workspace, "external-package")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "packages", "api", "generated"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(external, "dist"), 0755))

	_, ignore, err := resolveWatch(root, project.Watch{
		Ignore: []string{"packages/api/generated", "../external-package/dist"},
	})
	require.NoError(t, err)

	generated := mustInfo(t, filepath.Join(root, "packages", "api", "generated"))
	dist := mustInfo(t, filepath.Join(external, "dist"))

	assert.True(t, shouldSkipDir(root, ignore, filepath.Join(root, "packages", "api", "generated"), generated))
	assert.True(t, shouldSkipDir(root, ignore, filepath.Join(external, "dist"), dist))
	assert.True(t, isIgnored(root, ignore, filepath.Join(root, "packages", "api", "generated", "index.ts")))
	assert.True(t, isIgnored(root, ignore, filepath.Join(external, "dist", "index.js")))
	assert.False(t, isIgnored(root, ignore, filepath.Join(root, "sst.config.ts")))
}

func TestResolveWatchMatchesIgnoreNamesAnywhere(t *testing.T) {
	root := t.TempDir()
	_, ignore, err := resolveWatch(root, project.Watch{
		Ignore: []string{".env", "*.egg-info"},
	})
	require.NoError(t, err)

	eggInfo := mustInfo(t, filepath.Join(root, "packages", "api", "foo.egg-info"))

	assert.True(t, isIgnored(root, ignore, filepath.Join(root, ".env")))
	assert.True(t, isIgnored(root, ignore, filepath.Join(root, "packages", "api", ".env")))
	assert.True(t, shouldSkipDir(root, ignore, filepath.Join(root, "packages", "api", "foo.egg-info"), eggInfo))
	assert.True(t, isIgnored(root, ignore, filepath.Join(root, "packages", "api", "foo.egg-info", "PKG-INFO")))
	assert.False(t, isIgnored(root, ignore, filepath.Join(root, "packages", "api", ".env.local")))
}

func TestResolveWatchSkipsBuiltInDirs(t *testing.T) {
	root := t.TempDir()
	_, ignore, err := resolveWatch(root, project.Watch{})
	require.NoError(t, err)

	hidden := mustInfo(t, filepath.Join(root, ".sst"))
	nodeModules := mustInfo(t, filepath.Join(root, "node_modules"))
	normal := mustInfo(t, filepath.Join(root, "src"))

	assert.True(t, shouldSkipDir(root, ignore, filepath.Join(root, ".sst"), hidden))
	assert.True(t, shouldSkipDir(root, ignore, filepath.Join(root, "node_modules"), nodeModules))
	assert.False(t, shouldSkipDir(root, ignore, filepath.Join(root, "src"), normal))
}

func TestStartDiscoversFilesInNewDirectories(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := bus.SubscribeAll()
	defer bus.Unsubscribe(events)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Start(ctx, WatchConfig{Root: root, Watch: project.Watch{}})
	}()

	waitForWatcherReady(t, filepath.Join(root, "watcher-ready.txt"), events)

	path := filepath.Join(root, "src", "newpkg", "handler.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte("package newpkg\n"), 0644))
	require.True(t, waitForFileChangedEvent(events, path, 5*time.Second), "expected file change for %s", path)

	cancel()
	require.NoError(t, <-errCh)
}

func TestStartWatchesNewDirectories(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := bus.SubscribeAll()
	defer bus.Unsubscribe(events)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Start(ctx, WatchConfig{Root: root, Watch: project.Watch{}})
	}()

	waitForWatcherReady(t, filepath.Join(root, "watcher-ready.txt"), events)

	dir := filepath.Join(root, "src", "newpkg")
	require.NoError(t, os.MkdirAll(dir, 0755))

	path := filepath.Join(dir, "handler.go")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		require.NoError(t, os.WriteFile(path, []byte(time.Now().String()), 0644))
		if waitForFileChangedEvent(events, path, 200*time.Millisecond) {
			cancel()
			require.NoError(t, <-errCh)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("expected file change for %s", path)
}

func TestStartPicksUpImmediateEditsAfterNewFileDiscovery(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := bus.SubscribeAll()
	defer bus.Unsubscribe(events)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Start(ctx, WatchConfig{Root: root, Watch: project.Watch{}})
	}()

	waitForWatcherReady(t, filepath.Join(root, "watcher-ready.txt"), events)

	dir := filepath.Join(root, "src", "newpkg")
	require.NoError(t, os.MkdirAll(dir, 0755))

	path := filepath.Join(dir, "handler.go")
	require.NoError(t, os.WriteFile(path, []byte("package newpkg\n"), 0644))
	require.True(t, waitForFileChangedEvent(events, path, 5*time.Second), "expected initial file change for %s", path)

	require.NoError(t, os.WriteFile(path, []byte("package newpkg\n\nfunc Handler() {}\n"), 0644))
	require.True(t, waitForFileChangedEvent(events, path, 1*time.Second), "expected immediate follow-up file change for %s", path)

	cancel()
	require.NoError(t, <-errCh)
}

func TestStartOnlyWatchesConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	watchedDir := filepath.Join(root, "packages", "api")
	require.NoError(t, os.MkdirAll(watchedDir, 0755))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := bus.SubscribeAll()
	defer bus.Unsubscribe(events)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Start(ctx, WatchConfig{
			Root: root,
			Watch: project.Watch{
				Paths: []string{"packages/api"},
			},
		})
	}()

	waitForWatcherReady(t, filepath.Join(watchedDir, "watcher-ready.txt"), events)

	unwatched := filepath.Join(root, "docs", "guide.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(unwatched), 0755))
	require.NoError(t, os.WriteFile(unwatched, []byte("# docs\n"), 0644))
	assert.False(t, waitForFileChangedEvent(events, unwatched, 750*time.Millisecond), "did not expect file change for %s", unwatched)

	watched := filepath.Join(watchedDir, "handler.go")
	require.NoError(t, os.WriteFile(watched, []byte("package api\n"), 0644))
	require.True(t, waitForFileChangedEvent(events, watched, 5*time.Second), "expected file change for %s", watched)

	cancel()
	require.NoError(t, <-errCh)
}

func mustInfo(t *testing.T, path string) os.FileInfo {
	t.Helper()
	require.NoError(t, os.MkdirAll(path, 0755))
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info
}

func waitForWatcherReady(t *testing.T, probe string, events <-chan interface{}) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(probe), 0755))
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		require.NoError(t, os.WriteFile(probe, []byte(time.Now().String()), 0644))
		if waitForFileChangedEvent(events, probe, 200*time.Millisecond) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for watcher startup")
}

func waitForFileChangedEvent(events <-chan interface{}, path string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case evt := <-events:
			changed, ok := evt.(*FileChangedEvent)
			if ok && changed.Path == path {
				return true
			}
		case <-deadline:
			return false
		}
	}
}
