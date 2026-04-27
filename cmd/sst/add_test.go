package main

import (
	"testing"

	"github.com/sst/sst/v3/pkg/project"
)

func TestGetAliasName(t *testing.T) {
	tests := []struct {
		name  string
		entry *project.ProviderLockEntry
		want  string
	}{
		{
			name: "simple provider",
			entry: &project.ProviderLockEntry{
				Name:    "stripe",
				Package: "pulumi-stripe",
				Alias:   "stripe",
			},
			want: "stripe",
		},
		{
			name: "strip official suffix",
			entry: &project.ProviderLockEntry{
				Name:    "stripe-official",
				Package: "@sst-provider/stripe-official",
				Alias:   "stripe",
			},
			want: "stripe",
		},
		{
			name: "strip community suffix from alias",
			entry: &project.ProviderLockEntry{
				Name:    "@scope/pulumi-foo-community",
				Package: "@scope/pulumi-foo-community",
				Alias:   "foocommunity",
			},
			want: "foo",
		},
		{
			name: "package input still uses alias",
			entry: &project.ProviderLockEntry{
				Name:    "@paynearme/pulumi-jetstream",
				Package: "@paynearme/pulumi-jetstream",
				Alias:   "jetstream",
			},
			want: "jetstream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAliasName(tt.entry); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
