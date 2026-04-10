package main

import (
	"errors"
	"testing"

	"github.com/sst/sst/v3/internal/util"
)

func TestConfirmRepairMutations_AllowsSnapshotOnlyRecovery(t *testing.T) {
	t.Parallel()

	err := confirmRepairMutations("fe628cc5182389b648c70130", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRepairSnapshotFlagRequired(t *testing.T) {
	t.Parallel()

	cause := errors.New("EOF")
	err := repairSnapshotFlagRequired(nil, cause)
	if err == nil {
		t.Fatal("expected error")
	}
	var readable *util.ReadableError
	if !errors.As(err, &readable) {
		t.Fatalf("expected readable error, got %T", err)
	}
	if readable.Error() != "State is missing or corrupted. Re-run `sst state repair --dangerously-revert` to restore the latest valid snapshot. This can orphan or recreate resources." {
		t.Fatalf("unexpected error: %q", readable.Error())
	}
}
