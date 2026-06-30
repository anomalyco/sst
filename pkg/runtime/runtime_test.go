package runtime_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRuntime struct {
	matchFn func(string) bool
}

func (m *mockRuntime) Match(r string) bool {
	return m.matchFn(r)
}
func (m *mockRuntime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	return nil, nil
}
func (m *mockRuntime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	return nil, nil
}
func (m *mockRuntime) ShouldRebuild(functionID string, path string) bool {
	return false
}
func (m *mockRuntime) ShouldRunEagerly() bool {
	return true
}

func TestBuildInputOut(t *testing.T) {
	cfgPath := filepath.Join("/project", "sst.config.ts")
	workingDir := filepath.Join("/project", ".sst")

	t.Run("dev mode", func(t *testing.T) {
		input := &runtime.BuildInput{
			CfgPath:    cfgPath,
			Dev:        true,
			FunctionID: "myFunc",
		}
		expected := filepath.Join(workingDir, "artifacts", "myFunc-dev")
		assert.Equal(t, expected, input.Out())
	})

	t.Run("prod mode", func(t *testing.T) {
		input := &runtime.BuildInput{
			CfgPath:    cfgPath,
			Dev:        false,
			FunctionID: "myFunc",
		}
		expected := filepath.Join(workingDir, "artifacts", "myFunc-src")
		assert.Equal(t, expected, input.Out())
	})
}

func TestCollectionRuntime(t *testing.T) {
	t.Run("matching runtime found", func(t *testing.T) {
		mr := &mockRuntime{matchFn: func(r string) bool { return r == "nodejs" }}
		c := runtime.NewCollection("cfg", mr)

		rt, ok := c.Runtime("nodejs")
		require.True(t, ok)
		assert.Equal(t, mr, rt)
	})

	t.Run("no match", func(t *testing.T) {
		mr := &mockRuntime{matchFn: func(r string) bool { return false }}
		c := runtime.NewCollection("cfg", mr)

		_, ok := c.Runtime("python")
		assert.False(t, ok)
	})

	t.Run("empty collection", func(t *testing.T) {
		c := runtime.NewCollection("cfg")

		_, ok := c.Runtime("anything")
		assert.False(t, ok)
	})
}

func TestCollectionBuildEncryptedResourceFileWithBundle(t *testing.T) {
	// A 32-byte AES-256 key (all zeroes is fine for testing purposes).
	encryptionKey := base64.StdEncoding.EncodeToString(make([]byte, 32))

	t.Run("writes per-function subdirectory when bundle is set", func(t *testing.T) {
		bundleDir := t.TempDir()

		mr := &mockRuntime{matchFn: func(r string) bool { return r == "nodejs" }}
		c := runtime.NewCollection("cfg", mr)

		input := &runtime.BuildInput{
			FunctionID:    "my-function",
			Handler:       "index.handler",
			Bundle:        bundleDir,
			Runtime:       "nodejs",
			EncryptionKey: encryptionKey,
			Links:         map[string]json.RawMessage{},
		}

		_, err := c.Build(context.Background(), input)
		require.NoError(t, err)

		// Per-function file lives under .sst/<FunctionID>/ so concurrent
		// Build calls sharing the same bundle directory don't race, and the
		// uploaded zip for each function can exclude every sibling's subtree.
		perFunctionPath := filepath.Join(bundleDir, ".sst", "my-function", "resource.enc")
		_, err = os.Stat(perFunctionPath)
		assert.NoError(t, err, "expected %s to exist", perFunctionPath)

		// Default top-level filename is reserved for the non-bundle
		// (per-function artifact directory) path.
		defaultPath := filepath.Join(bundleDir, "resource.enc")
		_, err = os.Stat(defaultPath)
		assert.True(t, os.IsNotExist(err), "default resource.enc should not exist when bundle is set")
	})

	t.Run("distinct functions sharing a bundle write to distinct subdirs", func(t *testing.T) {
		bundleDir := t.TempDir()

		mr := &mockRuntime{matchFn: func(r string) bool { return r == "nodejs" }}
		c := runtime.NewCollection("cfg", mr)

		for _, id := range []string{"fn-a", "fn-b"} {
			input := &runtime.BuildInput{
				FunctionID:    id,
				Handler:       "index.handler",
				Bundle:        bundleDir,
				Runtime:       "nodejs",
				EncryptionKey: encryptionKey,
				Links:         map[string]json.RawMessage{},
			}
			_, err := c.Build(context.Background(), input)
			require.NoError(t, err)
		}

		for _, id := range []string{"fn-a", "fn-b"} {
			p := filepath.Join(bundleDir, ".sst", id, "resource.enc")
			_, err := os.Stat(p)
			assert.NoError(t, err, "expected %s to exist", p)
		}
	})
}
