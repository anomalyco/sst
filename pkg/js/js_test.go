package js_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/sst/sst/v3/pkg/js"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatError(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		assert.Equal(t, "", js.FormatError([]esbuild.Message{}))
	})

	t.Run("no location", func(t *testing.T) {
		msgs := []esbuild.Message{{Text: "something broke"}}
		assert.Equal(t, "something broke", js.FormatError(msgs))
	})

	t.Run("with location", func(t *testing.T) {
		msgs := []esbuild.Message{{
			Text:     "unexpected token",
			Location: &esbuild.Location{File: "app.ts", Line: 10, Column: 5},
		}}
		assert.Equal(t, "app.ts:10:5: unexpected token", js.FormatError(msgs))
	})

	t.Run("multiple errors", func(t *testing.T) {
		msgs := []esbuild.Message{
			{Text: "err1"},
			{Text: "err2", Location: &esbuild.Location{File: "b.ts", Line: 2, Column: 3}},
		}
		assert.Equal(t, "err1\nb.ts:2:3: err2", js.FormatError(msgs))
	})
}

func TestBuildGlobalsDoNotCollideWithDependencyBindings(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, ".sst", "platform")
	providerDir := filepath.Join(dir, "node_modules", "provider")
	dependencyDir := filepath.Join(dir, "node_modules", "collision")
	require.NoError(t, os.MkdirAll(outDir, 0755))
	for _, packageDir := range []string{providerDir, dependencyDir} {
		require.NoError(t, os.MkdirAll(packageDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(packageDir, "package.json"),
			[]byte(`{"type":"module"}`),
			0644,
		))
	}
	require.NoError(t, os.WriteFile(
		filepath.Join(providerDir, "index.js"),
		[]byte(`export const name = "provider";`),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dependencyDir, "index.js"),
		[]byte(`
export const postgresql = "dependency";
export default postgresql;
`),
		0644,
	))

	outfile := filepath.Join(outDir, "config.mjs")
	_, err := js.Build(js.EvalOptions{
		Dir:     dir,
		Outfile: outfile,
		Code:    `import value from "collision"; console.log(postgresql.name, value);`,
		Globals: `export * as postgresql from "provider";`,
	})
	require.NoError(t, err)

	output, err := exec.Command("node", outfile).CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Equal(t, "provider dependency\n", string(output))
}
