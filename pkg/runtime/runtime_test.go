package runtime_test

import (
	"context"
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

func TestCollectionBuildCustomBundle(t *testing.T) {
	projectDir := t.TempDir()
	absoluteBundle := filepath.Join(t.TempDir(), "dist")
	tests := []struct {
		name   string
		bundle string
		want   string
	}{
		{
			name:   "relative path",
			bundle: "./dist",
			want:   filepath.Join(projectDir, "dist"),
		},
		{
			name:   "absolute path",
			bundle: absoluteBundle,
			want:   absoluteBundle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &runtime.BuildInput{
				CfgPath: filepath.Join(projectDir, "sst.config.ts"),
				Bundle:  tt.bundle,
				Handler: "lambda.handler",
			}

			result, err := runtime.NewCollection(input.CfgPath).Build(context.Background(), input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result.Out)
			assert.True(t, filepath.IsAbs(result.Out))
			assert.Equal(t, filepath.Join(tt.want, "lambda.handler"), filepath.Join(result.Out, result.Handler))
		})
	}
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
