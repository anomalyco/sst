package state

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/pulumi/pulumi/pkg/v3/resource/deploy"
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

// Mutation records a single change made to the state.
type Mutation struct {
	Remove           *MutationRemove
	RemoveDependency *MutationRemoveDependency
	RemoveProperty   *MutationRemoveProperty
}

// MutationRemove records the removal of a resource.
type MutationRemove struct {
	Resource resource.URN
	Index    int
}

// MutationRemoveDependency records the removal of a dependency reference.
type MutationRemoveDependency struct {
	Resource   resource.URN
	Dependency resource.URN
}

// MutationRemoveProperty records the removal of a property dependency reference.
type MutationRemoveProperty struct {
	Resource   resource.URN
	Dependency resource.URN
	Property   resource.PropertyKey
}

// Remove deletes every resource in the snapshot whose URN.Name() == target,
// then removes any orphaned children (resources whose parent was deleted) and
// prunes dangling dependency and property-dependency references. The
// checkpoint is mutated in place.
func Remove(target string, checkpoint *apitype.CheckpointV3) []Mutation {
	result := []Mutation{}
	for resourceIndex := len(checkpoint.Latest.Resources) - 1; resourceIndex >= 0; resourceIndex-- {
		resource := checkpoint.Latest.Resources[resourceIndex]
		if resource.URN.Name() == target {
			checkpoint.Latest.Resources = append(checkpoint.Latest.Resources[:resourceIndex], checkpoint.Latest.Resources[resourceIndex+1:]...)
			result = append(result, Mutation{
				Remove: &MutationRemove{
					Resource: resource.URN,
				},
			})
		}
	}

	// Clean up orphaned children and dangling references.
	resources := map[resource.URN]bool{}
	for _, item := range checkpoint.Latest.Resources {
		resources[item.URN] = true
	}
	var repairs []Mutation
	for _, resource := range checkpoint.Latest.Resources {
		if resource.Parent != "" {
			if _, ok := resources[resource.Parent]; !ok {
				repairs = append(repairs, Mutation{
					Remove: &MutationRemove{
						Resource: resource.URN,
					},
				})
				delete(resources, resource.URN)
				continue
			}
		}
		for _, dependency := range resource.Dependencies {
			if _, ok := resources[dependency]; !ok {
				repairs = append(repairs, Mutation{
					RemoveDependency: &MutationRemoveDependency{
						Resource:   resource.URN,
						Dependency: dependency,
					},
				})
			}
		}
		for key, dependencies := range resource.PropertyDependencies {
			for _, dependency := range dependencies {
				if _, ok := resources[dependency]; !ok {
					repairs = append(repairs, Mutation{
						RemoveProperty: &MutationRemoveProperty{
							Resource:   resource.URN,
							Dependency: dependency,
							Property:   key,
						},
					})
				}
			}
		}
	}

	for _, mut := range repairs {
		if mut.Remove != nil {
			checkpoint.Latest.Resources = slices.DeleteFunc(checkpoint.Latest.Resources, func(item apitype.ResourceV3) bool {
				return item.URN == mut.Remove.Resource
			})
		}
		if mut.RemoveDependency != nil {
			index := slices.IndexFunc(checkpoint.Latest.Resources, func(item apitype.ResourceV3) bool {
				return item.URN == mut.RemoveDependency.Resource
			})
			if index >= 0 {
				checkpoint.Latest.Resources[index].Dependencies = slices.DeleteFunc(checkpoint.Latest.Resources[index].Dependencies, func(item resource.URN) bool {
					return item == mut.RemoveDependency.Dependency
				})
			}
		}
		if mut.RemoveProperty != nil {
			index := slices.IndexFunc(checkpoint.Latest.Resources, func(item apitype.ResourceV3) bool {
				return item.URN == mut.RemoveProperty.Resource
			})
			if index >= 0 {
				properties := checkpoint.Latest.Resources[index].PropertyDependencies[mut.RemoveProperty.Property]
				checkpoint.Latest.Resources[index].PropertyDependencies[mut.RemoveProperty.Property] = slices.DeleteFunc(properties, func(item resource.URN) bool {
					return item == mut.RemoveProperty.Dependency
				})
			}
		}
	}

	return append(result, repairs...)
}
