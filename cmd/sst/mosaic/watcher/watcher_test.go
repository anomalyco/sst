package watcher

import (
	"os"
	"path/filepath"
	"testing"

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

func mustInfo(t *testing.T, path string) os.FileInfo {
	t.Helper()
	require.NoError(t, os.MkdirAll(path, 0755))
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info
}
