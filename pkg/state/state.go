package state

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/pulumi/pulumi/pkg/v3/resource/deploy"
	"github.com/pulumi/pulumi/pkg/v3/resource/edit"
	"github.com/pulumi/pulumi/pkg/v3/resource/stack"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
)

// RepairResult describes the changes performed by Repair.
type RepairResult struct {
	// Reordered is true if Toposort changed the order of resources in the snapshot.
	Reordered bool
	// Pruned holds per-resource changes applied by Prune (URN rewrites + removed
	// dangling references).
	Pruned []deploy.PruneResult
}

// IsEmpty reports whether the repair made no changes.
func (r RepairResult) IsEmpty() bool {
	return !r.Reordered && len(r.Pruned) == 0
}

// RemoveResult describes the changes performed by Remove.
type RemoveResult struct {
	// Removed is the set of URNs that were deleted (target + cascaded dependents).
	Removed []resource.URN
	// Pruned holds any post-delete cleanup applied by Prune.
	Pruned []deploy.PruneResult
}

// IsEmpty reports whether the remove made no changes.
func (r RemoveResult) IsEmpty() bool {
	return len(r.Removed) == 0 && len(r.Pruned) == 0
}

// Repair fixes structural integrity issues in the given checkpoint by:
//  1. Topologically sorting resources so that dependencies precede dependents.
//  2. Pruning dangling references (Provider, Parent, Dependencies,
//     PropertyDependencies, DeletedWith, ReplaceWith). Resources whose Parent
//     URN no longer exists are unparented (their URN is rewritten) rather than
//     deleted.
//
// The checkpoint is mutated in place. Secrets are preserved (re-encrypted on
// the way out using the stack passphrase).
func Repair(ctx context.Context, passphrase string, checkpoint *apitype.CheckpointV3) (RepairResult, error) {
	snap, err := loadSnapshot(ctx, passphrase, checkpoint)
	if err != nil {
		return RepairResult{}, err
	}

	beforeOrder := urnOrder(snap.Resources)
	if err := snap.Toposort(); err != nil {
		return RepairResult{}, fmt.Errorf("toposort: %w", err)
	}
	afterOrder := urnOrder(snap.Resources)
	pruned := snap.Prune()

	result := RepairResult{
		Reordered: !sameOrder(beforeOrder, afterOrder),
		Pruned:    pruned,
	}

	if result.IsEmpty() {
		return result, nil
	}

	if err := saveSnapshot(ctx, snap, checkpoint); err != nil {
		return RepairResult{}, err
	}
	return result, nil
}

// Remove deletes every resource in the snapshot whose URN.Name() == target,
// cascading to dependents (children, dependsOn, property dependencies,
// DeletedWith, ReplaceWith). After deletion it runs a Prune to clean up any
// references that became dangling. Protected resources are not deleted; they
// produce a clear error.
//
// The checkpoint is mutated in place.
func Remove(ctx context.Context, passphrase string, target string, checkpoint *apitype.CheckpointV3) (RemoveResult, error) {
	snap, err := loadSnapshot(ctx, passphrase, checkpoint)
	if err != nil {
		return RemoveResult{}, err
	}

	var matches []*resource.State
	for _, r := range snap.Resources {
		if r.URN.Name() == target {
			matches = append(matches, r)
		}
	}
	if len(matches) == 0 {
		return RemoveResult{}, fmt.Errorf("no resource named %q in state", target)
	}

	before := urnSet(snap.Resources)
	onProtected := func(r *resource.State) error {
		return fmt.Errorf("resource %s is protected; remove protection first via 'sst state edit'", r.URN)
	}
	for _, m := range matches {
		// Resources may have been removed by an earlier iteration's cascade; skip
		// those that are no longer present in the snapshot.
		if !contains(snap.Resources, m) {
			continue
		}
		if err := edit.DeleteResource(snap, m, onProtected, true); err != nil {
			var dep edit.ResourceHasDependenciesError
			if errors.As(err, &dep) {
				return RemoveResult{}, fmt.Errorf("could not delete %s: has dependents", m.URN)
			}
			return RemoveResult{}, err
		}
	}

	after := urnSet(snap.Resources)
	var removed []resource.URN
	for u := range before {
		if _, ok := after[u]; !ok {
			removed = append(removed, u)
		}
	}

	// Drop pending operations referencing removed resources.
	if len(snap.PendingOperations) > 0 {
		filtered := snap.PendingOperations[:0]
		for _, op := range snap.PendingOperations {
			if _, ok := after[op.Resource.URN]; ok {
				filtered = append(filtered, op)
			}
		}
		snap.PendingOperations = filtered
	}

	pruned := snap.Prune()

	result := RemoveResult{Removed: removed, Pruned: pruned}
	if result.IsEmpty() {
		return result, nil
	}

	if err := saveSnapshot(ctx, snap, checkpoint); err != nil {
		return RemoveResult{}, err
	}
	return result, nil
}

// loadSnapshot deserializes a CheckpointV3 into an in-memory deploy.Snapshot
// using the stack passphrase to decrypt secrets.
func loadSnapshot(ctx context.Context, passphrase string, checkpoint *apitype.CheckpointV3) (*deploy.Snapshot, error) {
	os.Setenv("PULUMI_CONFIG_PASSPHRASE", passphrase)
	sp := &defaultSecretsProvider{passphrase: passphrase}
	snap, err := stack.DeserializeCheckpoint(ctx, sp, checkpoint)
	if err != nil {
		return nil, fmt.Errorf("deserialize checkpoint: %w", err)
	}
	return snap, nil
}

// saveSnapshot serializes the snapshot back into checkpoint.Latest, keeping
// secrets encrypted via the snapshot's SecretsManager.
func saveSnapshot(ctx context.Context, snap *deploy.Snapshot, checkpoint *apitype.CheckpointV3) error {
	depl, err := stack.SerializeDeployment(ctx, snap, false)
	if err != nil {
		return fmt.Errorf("serialize deployment: %w", err)
	}
	checkpoint.Latest = depl
	return nil
}

func urnSet(resources []*resource.State) map[resource.URN]struct{} {
	out := make(map[resource.URN]struct{}, len(resources))
	for _, r := range resources {
		out[r.URN] = struct{}{}
	}
	return out
}

func urnOrder(resources []*resource.State) []resource.URN {
	out := make([]resource.URN, len(resources))
	for i, r := range resources {
		out[i] = r.URN
	}
	return out
}

func sameOrder(a, b []resource.URN) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(haystack []*resource.State, needle *resource.State) bool {
	for _, r := range haystack {
		if r == needle {
			return true
		}
	}
	return false
}
