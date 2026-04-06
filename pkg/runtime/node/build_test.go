package node

import (
	"testing"

	"github.com/sst/sst/v3/pkg/js"
)

func TestResolveInstallVersion(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		install map[string]string
		deps    map[string]string
		want    string
	}{
		{
			name:    "explicit version overrides package json",
			pkg:     "sharp",
			install: map[string]string{"sharp": "0.33.5"},
			deps:    map[string]string{"sharp": "0.32.6"},
			want:    "0.33.5",
		},
		{
			name:    "wildcard falls back to package json",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			deps:    map[string]string{"sharp": "0.32.6"},
			want:    "0.32.6",
		},
		{
			name:    "missing package falls back to wildcard",
			pkg:     "sharp",
			install: map[string]string{"sharp": "*"},
			want:    "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveInstallVersion(tt.pkg, tt.install, js.PackageJson{Dependencies: tt.deps})
			if got != tt.want {
				t.Fatalf("resolveInstallVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
