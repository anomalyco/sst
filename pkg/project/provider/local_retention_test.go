package provider

import (
	"bytes"
	"os"
	"testing"
)

func TestLocalRetentionPrunesUpdateHistory(t *testing.T) {
	t.Parallel()

	home := &LocalHome{root: t.TempDir()}
	app := "my-app"
	stage := "production"
	updateIDs := []string{
		"fffffe987654000000000001",
		"fffffe987654000000000002",
		"fffffe987654000000000003",
		"fffffe987654000000000004",
		"fffffe987654000000000005",
	}

	for _, updateID := range updateIDs {
		for _, kind := range []string{"snapshot", "eventlog", "update"} {
			if err := home.putData(kind, app, stage+"/"+updateID, bytes.NewReader([]byte(updateID))); err != nil {
				t.Fatalf("writing %s failed: %v", kind, err)
			}
		}
	}
	if err := home.putData("app", app, stage, bytes.NewReader([]byte("current"))); err != nil {
		t.Fatalf("writing current state failed: %v", err)
	}

	if err := home.prune(app, stage, 3); err != nil {
		t.Fatalf("prune failed: %v", err)
	}

	for i, updateID := range updateIDs {
		for _, kind := range []string{"snapshot", "eventlog", "update"} {
			_, err := os.Stat(home.pathForData(kind, app, stage+"/"+updateID))
			if i < 3 && err != nil {
				t.Fatalf("expected retained %s/%s: %v", kind, updateID, err)
			}
			if i >= 3 && !os.IsNotExist(err) {
				t.Fatalf("expected pruned %s/%s, got %v", kind, updateID, err)
			}
		}
	}
	if _, err := os.Stat(home.pathForData("app", app, stage)); err != nil {
		t.Fatalf("expected current state to remain: %v", err)
	}
}
