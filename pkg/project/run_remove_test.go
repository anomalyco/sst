package project

import (
	"context"
	"errors"
	"testing"
)

func TestRemoveRefreshPreservesProtection(t *testing.T) {
	for _, refresh := range []bool{false, true} {
		// No backend is configured: protection must be checked before any
		// provider or Pulumi operation, even when refresh is requested.
		p := &Project{app: &App{Protect: true}}
		err := p.Run(context.Background(), &StackInput{Command: "remove", Refresh: refresh})
		if !errors.Is(err, ErrProtectedStage) {
			t.Fatalf("refresh=%v: got %v, want ErrProtectedStage", refresh, err)
		}
	}
}
