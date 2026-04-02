package project

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/sst/sst/v3/internal/util"
)

func TestWatchUnmarshalLegacyArray(t *testing.T) {
	var watch Watch
	err := json.Unmarshal([]byte(`["packages/api"]`), &watch)
	if err != nil {
		t.Fatal(err)
	}
	if len(watch.Paths) != 1 || watch.Paths[0] != "packages/api" {
		t.Fatalf("unexpected paths: %#v", watch.Paths)
	}
	if watch.Ignore != nil {
		t.Fatalf("unexpected ignore: %#v", watch.Ignore)
	}
	if !watch.legacyArray {
		t.Fatal("expected legacy array watch")
	}
}

func TestWatchUnmarshalLegacyArrayAllowsGlobs(t *testing.T) {
	var watch Watch
	err := json.Unmarshal([]byte(`["packages/*"]`), &watch)
	if err != nil {
		t.Fatal(err)
	}
	if len(watch.Paths) != 1 || watch.Paths[0] != "packages/*" {
		t.Fatalf("unexpected paths: %#v", watch.Paths)
	}
	if watch.Ignore != nil {
		t.Fatalf("unexpected ignore: %#v", watch.Ignore)
	}
	if !watch.legacyArray {
		t.Fatal("expected legacy array watch")
	}
}

func TestWatchUnmarshalObject(t *testing.T) {
	var watch Watch
	err := json.Unmarshal([]byte(`{"paths":["packages/api"],"ignore":[".env"]}`), &watch)
	if err != nil {
		t.Fatal(err)
	}
	if len(watch.Paths) != 1 || watch.Paths[0] != "packages/api" {
		t.Fatalf("unexpected paths: %#v", watch.Paths)
	}
	if len(watch.Ignore) != 1 || watch.Ignore[0] != ".env" {
		t.Fatalf("unexpected ignore: %#v", watch.Ignore)
	}
	if watch.legacyArray {
		t.Fatal("did not expect legacy array watch")
	}
}

func TestWatchUnmarshalObjectRejectsGlobs(t *testing.T) {
	var watch Watch
	err := json.Unmarshal([]byte(`{"paths":["packages/*"]}`), &watch)
	if err == nil {
		t.Fatal("expected error")
	}
	var readable *util.ReadableError
	if !errors.As(err, &readable) {
		t.Fatalf("expected readable error, got %T", err)
	}
	if err.Error() != "Watch path globs are only supported in the legacy array form: \"packages/*\"\nUse explicit directories in watch.paths instead: https://sst.dev/docs/reference/config/#watch" {
		t.Fatalf("unexpected error: %v", err)
	}
}
