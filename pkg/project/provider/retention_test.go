package provider

import (
	"reflect"
	"testing"
)

func TestStaleUpdateIDs(t *testing.T) {
	t.Parallel()

	keys := []string{
		"snapshot/app/stage/fffffe987654000000000005.json",
		"snapshot/app/stage/fffffe987654000000000002.json",
		"snapshot/app/stage/fffffe987654000000000004.json",
		"snapshot/app/stage/fffffe987654000000000001.json",
		"snapshot/app/stage/fffffe987654000000000003.json",
	}

	got := staleUpdateIDs(keys, 3)
	want := []string{
		"fffffe987654000000000004",
		"fffffe987654000000000005",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestStaleUpdateIDsIgnoresUnknownKeys(t *testing.T) {
	t.Parallel()

	keys := []string{
		"snapshot/app/stage/fffffe987654000000000001.json",
		"snapshot/app/stage/fffffe987654000000000002.json",
		"snapshot/app/stage/not-an-update.json",
		"snapshot/app/stage/fffffe987654000000000001.json",
	}

	got := staleUpdateIDs(keys, 1)
	want := []string{"fffffe987654000000000002"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestStaleUpdateIDsWithinRetention(t *testing.T) {
	t.Parallel()

	keys := []string{
		"snapshot/app/stage/fffffe987654000000000001.json",
		"snapshot/app/stage/fffffe987654000000000002.json",
	}

	if got := staleUpdateIDs(keys, 2); got != nil {
		t.Fatalf("expected no stale updates, got %v", got)
	}
}
