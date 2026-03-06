package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sst/sst/v3/internal/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindUp(t *testing.T) {
	t.Run("file in current dir", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target.txt")
		os.WriteFile(target, []byte("x"), 0644)

		found, err := fs.FindUp(dir, "target.txt")
		require.NoError(t, err)
		assert.Equal(t, target, found)
	})

	t.Run("file in parent dir", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target.txt")
		os.WriteFile(target, []byte("x"), 0644)
		child := filepath.Join(dir, "sub")
		os.Mkdir(child, 0755)

		found, err := fs.FindUp(child, "target.txt")
		require.NoError(t, err)
		assert.Equal(t, target, found)
	})

	t.Run("file not found", func(t *testing.T) {
		dir := t.TempDir()
		_, err := fs.FindUp(dir, "nonexistent.txt")
		assert.Error(t, err)
	})
}

func TestFindDown(t *testing.T) {
	t.Run("finds nested file", func(t *testing.T) {
		dir := t.TempDir()
		nested := filepath.Join(dir, "a", "b")
		os.MkdirAll(nested, 0755)
		target := filepath.Join(nested, "config.json")
		os.WriteFile(target, []byte("{}"), 0644)

		results := fs.FindDown(dir, "config.json")
		assert.Equal(t, []string{target}, results)
	})

	t.Run("skips node_modules", func(t *testing.T) {
		dir := t.TempDir()
		nm := filepath.Join(dir, "node_modules")
		os.MkdirAll(nm, 0755)
		os.WriteFile(filepath.Join(nm, "pkg.json"), []byte("{}"), 0644)

		results := fs.FindDown(dir, "pkg.json")
		assert.Empty(t, results)
	})

	t.Run("skips dot dirs", func(t *testing.T) {
		dir := t.TempDir()
		dotDir := filepath.Join(dir, ".hidden")
		os.MkdirAll(dotDir, 0755)
		os.WriteFile(filepath.Join(dotDir, "f.txt"), []byte("x"), 0644)

		results := fs.FindDown(dir, "f.txt")
		assert.Empty(t, results)
	})

	t.Run("skips git submodules", func(t *testing.T) {
		dir := t.TempDir()

		// Normal nested dir — should be found
		normal := filepath.Join(dir, "app")
		os.MkdirAll(normal, 0755)
		os.WriteFile(filepath.Join(normal, "package.json"), []byte("{}"), 0644)

		// Submodule dir (has .git file) — should be skipped
		sub := filepath.Join(dir, "submod")
		os.MkdirAll(sub, 0755)
		os.WriteFile(filepath.Join(sub, ".git"), []byte("gitdir: ../../.git/modules/submod"), 0644)
		os.WriteFile(filepath.Join(sub, "package.json"), []byte("{}"), 0644)

		results := fs.FindDown(dir, "package.json")
		assert.Equal(t, []string{filepath.Join(normal, "package.json")}, results)
	})

	t.Run("does not skip dirs with .git directory", func(t *testing.T) {
		dir := t.TempDir()

		// Nested repo clone (has .git directory, not file) — should be walked
		nested := filepath.Join(dir, "nested-repo")
		os.MkdirAll(filepath.Join(nested, ".git"), 0755)
		os.WriteFile(filepath.Join(nested, "package.json"), []byte("{}"), 0644)

		results := fs.FindDown(dir, "package.json")
		assert.Equal(t, []string{filepath.Join(nested, "package.json")}, results)
	})

	t.Run("not found returns empty", func(t *testing.T) {
		dir := t.TempDir()
		results := fs.FindDown(dir, "nope.txt")
		assert.Empty(t, results)
	})
}

func TestIsGitSubmodule(t *testing.T) {
	t.Run("git file returns true", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../../.git/modules/foo"), 0644)
		assert.True(t, fs.IsGitSubmodule(dir))
	})

	t.Run("git directory returns false", func(t *testing.T) {
		dir := t.TempDir()
		os.Mkdir(filepath.Join(dir, ".git"), 0755)
		assert.False(t, fs.IsGitSubmodule(dir))
	})

	t.Run("no git entry returns false", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, fs.IsGitSubmodule(dir))
	})
}

func TestExists(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "file.txt")
		os.WriteFile(f, []byte("x"), 0644)
		assert.True(t, fs.Exists(f))
	})

	t.Run("non-existing", func(t *testing.T) {
		assert.False(t, fs.Exists("/tmp/nonexistent_sst_test_file"))
	})

	t.Run("existing dir", func(t *testing.T) {
		dir := t.TempDir()
		assert.True(t, fs.Exists(dir))
	})
}
