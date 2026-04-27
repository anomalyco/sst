package worker

import (
	"testing"
)

func TestStripAliasedExternals(t *testing.T) {
	original := map[string]string{
		"http":      "unenv/node/http",
		"node:http": "unenv/node/http",
		"buffer":    "buffer",
	}

	got := stripAliasedExternals(original, []string{"http", "node:http", "node:*"})

	if _, ok := got["http"]; ok {
		t.Fatalf("http alias should be removed")
	}
	if _, ok := got["node:http"]; ok {
		t.Fatalf("node:http alias should be removed")
	}
	if got["buffer"] != "buffer" {
		t.Fatalf("buffer alias = %q, want %q", got["buffer"], "buffer")
	}
	if original["http"] != "unenv/node/http" {
		t.Fatalf("original alias map was mutated")
	}
}
