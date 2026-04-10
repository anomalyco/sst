package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sst/sst/v3/internal/util"
)

func TestExportReturnsReadableErrorForEmptyStateFile(t *testing.T) {
	t.Parallel()

	workdir := &PulumiWorkdir{
		path: t.TempDir(),
		project: &Project{
			app: &App{Name: "app", Stage: "dev"},
		},
	}

	statePath := workdir.state()
	err := os.MkdirAll(filepath.Dir(statePath), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(statePath, []byte{}, 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = workdir.Export()
	if err == nil {
		t.Fatal("expected export to fail for an empty state file")
	}

	var readable *util.ReadableError
	if !errors.As(err, &readable) {
		t.Fatalf("expected readable error, got %T: %v", err, err)
	}
	if readable.Error() != "State file is empty or corrupted" {
		t.Fatalf("unexpected readable error message: %q", readable.Error())
	}
}
