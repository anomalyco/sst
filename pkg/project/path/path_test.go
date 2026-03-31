package path_test

import (
	"path/filepath"
	"testing"

	"github.com/sst/sst/v3/pkg/project/path"
	"github.com/stretchr/testify/assert"
)

func TestPathResolvers(t *testing.T) {
	cfgPath := filepath.Join("/foo", "bar", "sst.config.ts")
	root := filepath.Join("/foo", "bar")
	working := filepath.Join(root, ".sst")

	tests := []struct {
		name     string
		fn       func(string) string
		expected string
	}{
		{"ResolveRootDir", path.ResolveRootDir, root},
		{"ResolveWorkingDir", path.ResolveWorkingDir, working},
		{"ResolvePlatformDir", path.ResolvePlatformDir, filepath.Join(working, "platform")},
		{"ResolveLogDir", path.ResolveLogDir, filepath.Join(working, "log")},
		{"ResolveProviderLock", path.ResolveProviderLock, filepath.Join(working, "provider-lock.json")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.fn(cfgPath))
		})
	}
}
