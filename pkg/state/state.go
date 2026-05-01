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

// RepairResult describes the changes performed by Repair. Pruned holds
// per-resource changes applied by Snapshot.Prune (URN rewrites + removed
// dangling references). Toposort runs unconditionally and is not reported.
type RepairResult struct {
	Pruned []deploy.PruneResult
}

func (r RepairResult) IsEmpty() bool { return len(r.Pruned) == 0 }

// RemoveResult describes the changes performed by Remove.
type RemoveResult struct {
	// Removed is the set of URNs deleted (target + cascaded dependents).
	Removed []resource.URN
	// Pruned holds any post-delete cleanup applied by Snapshot.Prune.
	Pruned []deploy.PruneResult
}

func (r RemoveResult) IsEmpty() bool { return len(r.Removed) == 0 && len(r.Pruned) == 0 }

// Repair fixes structural integrity issues in the given checkpoint by
// topologically sorting resources and pruning dangling references (Provider,
// Parent, Dependencies, PropertyDependencies, DeletedWith, ReplaceWith).
// Resources whose Parent URN no longer exists are unparented (URN rewritten),
// not deleted. The checkpoint is mutated in place.
func Repair(ctx context.Context, passphrase string, checkpoint *apitype.CheckpointV3) (RepairResult, error) {
	os.Setenv("PULUMI_CONFIG_PASSPHRASE", passphrase)
	snap, err := stack.DeserializeCheckpoint(ctx, &defaultSecretsProvider{passphrase: passphrase}, checkpoint)
	if err != nil {
		return RepairResult{}, fmt.Errorf("deserialize checkpoint: %w", err)
	}

	if err := snap.Toposort(); err != nil {
		return RepairResult{}, fmt.Errorf("toposort: %w", err)
	}
	result := RepairResult{Pruned: snap.Prune()}

	if result.IsEmpty() {
		return result, nil
	}

	depl, err := stack.SerializeDeployment(ctx, snap, false)
	if err != nil {
		return RepairResult{}, fmt.Errorf("serialize deployment: %w", err)
	}
	checkpoint.Latest = depl
	return result, nil
}

// Remove deletes every resource whose URN.Name() == target, cascading to
// dependents (children, dependsOn, property deps, DeletedWith, ReplaceWith),
// then prunes leftover dangling references and pending operations. Protected
// resources are not deleted; encountering one returns an error. The checkpoint
// is mutated in place.
func Remove(ctx context.Context, passphrase string, target string, checkpoint *apitype.CheckpointV3) (RemoveResult, error) {
	os.Setenv("PULUMI_CONFIG_PASSPHRASE", passphrase)
	snap, err := stack.DeserializeCheckpoint(ctx, &defaultSecretsProvider{passphrase: passphrase}, checkpoint)
	if err != nil {
		return RemoveResult{}, fmt.Errorf("deserialize checkpoint: %w", err)
	}

	var targets []*resource.State
	for _, r := range snap.Resources {
		if r.URN.Name() == target {
			targets = append(targets, r)
		}
	}
	if len(targets) == 0 {
		return RemoveResult{}, fmt.Errorf("no resource named %q in state", target)
	}

	before := make(map[resource.URN]struct{}, len(snap.Resources))
	for _, r := range snap.Resources {
		before[r.URN] = struct{}{}
	}

	onProtected := func(r *resource.State) error {
		return fmt.Errorf("resource %s is protected; remove protection first via 'sst state edit'", r.URN)
	}
	for _, t := range targets {
		// A previous iteration's cascade may have already removed this target.
		if len(edit.LocateResource(snap, t.URN)) == 0 {
			continue
		}
		if err := edit.DeleteResource(snap, t, onProtected, true); err != nil {
			var dep edit.ResourceHasDependenciesError
			if errors.As(err, &dep) {
				return RemoveResult{}, fmt.Errorf("could not delete %s: has dependents", t.URN)
			}
			return RemoveResult{}, err
		}
	}

	after := make(map[resource.URN]struct{}, len(snap.Resources))
	for _, r := range snap.Resources {
		after[r.URN] = struct{}{}
	}
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

	result := RemoveResult{Removed: removed, Pruned: snap.Prune()}
	if result.IsEmpty() {
		return result, nil
	}

	depl, err := stack.SerializeDeployment(ctx, snap, false)
	if err != nil {
		return RemoveResult{}, fmt.Errorf("serialize deployment: %w", err)
	}
	checkpoint.Latest = depl
	return result, nil
}
