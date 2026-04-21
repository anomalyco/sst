package python_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sst/sst/v3/pkg/project/common"
	"github.com/sst/sst/v3/pkg/types/python"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Run("no pyproject.toml returns nil", func(t *testing.T) {
		dir := t.TempDir()
		err := python.Generate(dir, common.Links{}, nil)
		require.NoError(t, err)
	})

	t.Run("creates sst.pyi with correct types", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]"), 0644)

		links := common.Links{
			"MyBucket": {
				Properties: map[string]interface{}{
					"name": "my-bucket",
					"arn":  "arn:aws:s3:::my-bucket",
				},
			},
		}

		err := python.Generate(dir, links, nil)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(dir, "sst.pyi"))
		require.NoError(t, err)

		out := string(content)
		assert.Contains(t, out, "class Resource:")
		assert.Contains(t, out, "class MyBucket:")
		assert.Contains(t, out, "arn: str")
		assert.Contains(t, out, "name: str")
		assert.Contains(t, out, "class App:")
		assert.Contains(t, out, "name: str")
		assert.Contains(t, out, "stage: str")
	})

	t.Run("fields are sorted", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]"), 0644)

		links := common.Links{
			"Db": {
				Properties: map[string]interface{}{
					"z_field": "z",
					"a_field": "a",
				},
			},
		}

		err := python.Generate(dir, links, nil)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(dir, "sst.pyi"))
		require.NoError(t, err)

		out := string(content)
		aIdx := indexOf(out, "a_field")
		zIdx := indexOf(out, "z_field")
		assert.Greater(t, aIdx, -1)
		assert.Less(t, aIdx, zIdx)
	})

	t.Run("nested pyproject.toml", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "services", "api")
		os.MkdirAll(sub, 0755)
		os.WriteFile(filepath.Join(sub, "pyproject.toml"), []byte("[project]"), 0644)

		err := python.Generate(dir, common.Links{}, nil)
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(sub, "sst.pyi"))
	})

	t.Run("ignores configured directories", func(t *testing.T) {
		dir := t.TempDir()
		ignored := filepath.Join(dir, "services", "legacy")
		included := filepath.Join(dir, "services", "api")
		os.MkdirAll(ignored, 0755)
		os.MkdirAll(included, 0755)
		os.WriteFile(filepath.Join(ignored, "pyproject.toml"), []byte("[project]"), 0644)
		os.WriteFile(filepath.Join(included, "pyproject.toml"), []byte("[project]"), 0644)

		err := python.Generate(dir, common.Links{}, []string{"services/legacy"})
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(included, "sst.pyi"))
		_, err = os.Stat(filepath.Join(ignored, "sst.pyi"))
		assert.True(t, os.IsNotExist(err))
	})
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
