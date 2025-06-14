package state_test

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/sst/sst/v3/pkg/state"
)

func TestRemove(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		checkpoint *apitype.CheckpointV3
		wantMuts   int
	}{
		{
			name:   "remove single resource",
			target: "test-resource",
			checkpoint: &apitype.CheckpointV3{
				Latest: &apitype.DeploymentV3{
					Resources: []apitype.ResourceV3{
						{
							URN: resource.URN("urn:pulumi:test::test::test::test-resource"),
						},
					},
				},
			},
			wantMuts: 1,
		},
		{
			name:   "remove non-existent resource",
			target: "non-existent",
			checkpoint: &apitype.CheckpointV3{
				Latest: &apitype.DeploymentV3{
					Resources: []apitype.ResourceV3{
						{
							URN: resource.URN("urn:pulumi:test::test::test::test-resource"),
						},
					},
				},
			},
			wantMuts: 0,
		},
		{
			name:   "remove multiple resources with same name",
			target: "test-resource",
			checkpoint: &apitype.CheckpointV3{
				Latest: &apitype.DeploymentV3{
					Resources: []apitype.ResourceV3{
						{
							URN: resource.URN("urn:pulumi:test::test::test::test-resource"),
						},
						{
							URN: resource.URN("urn:pulumi:test::test::other::test-resource"),
						},
					},
				},
			},
			wantMuts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalResourceCount := len(tt.checkpoint.Latest.Resources)
			mutations := state.Remove(tt.target, tt.checkpoint)
			
			if len(mutations) < tt.wantMuts {
				t.Errorf("Remove() returned %d mutations, want at least %d", len(mutations), tt.wantMuts)
			}

			// Check that resources were actually removed from checkpoint
			removedCount := 0
			for _, mut := range mutations {
				if mut.Remove != nil {
					removedCount++
				}
			}

			expectedResourceCount := originalResourceCount - removedCount
			if len(tt.checkpoint.Latest.Resources) != expectedResourceCount {
				t.Errorf("Expected %d resources after removal, got %d", expectedResourceCount, len(tt.checkpoint.Latest.Resources))
			}
		})
	}
}

func TestRepair(t *testing.T) {
	tests := []struct {
		name       string
		checkpoint *apitype.CheckpointV3
		wantMuts   int
	}{
		{
			name: "repair missing parent dependency",
			checkpoint: &apitype.CheckpointV3{
				Latest: &apitype.DeploymentV3{
					Resources: []apitype.ResourceV3{
						{
							URN:    resource.URN("urn:pulumi:test::test::test::child"),
							Parent: resource.URN("urn:pulumi:test::test::test::missing-parent"),
						},
					},
				},
			},
			wantMuts: 1,
		},
		{
			name: "repair missing resource dependency",
			checkpoint: &apitype.CheckpointV3{
				Latest: &apitype.DeploymentV3{
					Resources: []apitype.ResourceV3{
						{
							URN: resource.URN("urn:pulumi:test::test::test::resource"),
							Dependencies: []resource.URN{
								resource.URN("urn:pulumi:test::test::test::missing-dep"),
							},
						},
					},
				},
			},
			wantMuts: 1,
		},
		{
			name: "repair missing property dependency",
			checkpoint: &apitype.CheckpointV3{
				Latest: &apitype.DeploymentV3{
					Resources: []apitype.ResourceV3{
						{
							URN: resource.URN("urn:pulumi:test::test::test::resource"),
							PropertyDependencies: map[resource.PropertyKey][]resource.URN{
								"prop1": {
									resource.URN("urn:pulumi:test::test::test::missing-prop-dep"),
								},
							},
						},
					},
				},
			},
			wantMuts: 1,
		},
		{
			name: "no repair needed",
			checkpoint: &apitype.CheckpointV3{
				Latest: &apitype.DeploymentV3{
					Resources: []apitype.ResourceV3{
						{
							URN: resource.URN("urn:pulumi:test::test::test::parent"),
						},
						{
							URN:    resource.URN("urn:pulumi:test::test::test::child"),
							Parent: resource.URN("urn:pulumi:test::test::test::parent"),
							Dependencies: []resource.URN{
								resource.URN("urn:pulumi:test::test::test::parent"),
							},
							PropertyDependencies: map[resource.PropertyKey][]resource.URN{
								"prop1": {
									resource.URN("urn:pulumi:test::test::test::parent"),
								},
							},
						},
					},
				},
			},
			wantMuts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutations := state.Repair(tt.checkpoint)
			
			if len(mutations) != tt.wantMuts {
				t.Errorf("Repair() returned %d mutations, want %d", len(mutations), tt.wantMuts)
			}

			// Verify mutation types are correct
			for _, mut := range mutations {
				if mut.Remove == nil && mut.RemoveDependency == nil && mut.RemoveProperty == nil {
					t.Error("Mutation has no valid operation")
				}
			}
		})
	}
}

func TestMutationTypes(t *testing.T) {
	t.Run("MutationRemove", func(t *testing.T) {
		mut := state.Mutation{
			Remove: &state.MutationRemove{
				Resource: resource.URN("urn:pulumi:test::test::test::resource"),
				Index:    0,
			},
		}
		
		if mut.Remove == nil {
			t.Error("Expected Remove mutation to be set")
		}
		if mut.Remove.Resource == "" {
			t.Error("Expected Resource URN to be set")
		}
	})

	t.Run("MutationRemoveDependency", func(t *testing.T) {
		mut := state.Mutation{
			RemoveDependency: &state.MutationRemoveDependency{
				Resource:   resource.URN("urn:pulumi:test::test::test::resource"),
				Dependency: resource.URN("urn:pulumi:test::test::test::dependency"),
			},
		}
		
		if mut.RemoveDependency == nil {
			t.Error("Expected RemoveDependency mutation to be set")
		}
		if mut.RemoveDependency.Resource == "" {
			t.Error("Expected Resource URN to be set")
		}
		if mut.RemoveDependency.Dependency == "" {
			t.Error("Expected Dependency URN to be set")
		}
	})

	t.Run("MutationRemoveProperty", func(t *testing.T) {
		mut := state.Mutation{
			RemoveProperty: &state.MutationRemoveProperty{
				Resource:   resource.URN("urn:pulumi:test::test::test::resource"),
				Dependency: resource.URN("urn:pulumi:test::test::test::dependency"),
				Property:   resource.PropertyKey("prop1"),
			},
		}
		
		if mut.RemoveProperty == nil {
			t.Error("Expected RemoveProperty mutation to be set")
		}
		if mut.RemoveProperty.Resource == "" {
			t.Error("Expected Resource URN to be set")
		}
		if mut.RemoveProperty.Dependency == "" {
			t.Error("Expected Dependency URN to be set")
		}
		if mut.RemoveProperty.Property == "" {
			t.Error("Expected Property key to be set")
		}
	})
}

func TestRepairAppliesMutations(t *testing.T) {
	checkpoint := &apitype.CheckpointV3{
		Latest: &apitype.DeploymentV3{
			Resources: []apitype.ResourceV3{
				{
					URN: resource.URN("urn:pulumi:test::test::test::resource1"),
					Dependencies: []resource.URN{
						resource.URN("urn:pulumi:test::test::test::missing-dep"),
						resource.URN("urn:pulumi:test::test::test::resource2"),
					},
					PropertyDependencies: map[resource.PropertyKey][]resource.URN{
						"prop1": {
							resource.URN("urn:pulumi:test::test::test::missing-prop-dep"),
							resource.URN("urn:pulumi:test::test::test::resource2"),
						},
					},
				},
				{
					URN: resource.URN("urn:pulumi:test::test::test::resource2"),
				},
			},
		},
	}

	mutations := state.Repair(checkpoint)

	// Should have mutations for missing dependency and missing property dependency
	if len(mutations) != 2 {
		t.Errorf("Expected 2 mutations, got %d", len(mutations))
	}

	// Verify that the mutations were applied to the checkpoint
	resource1 := checkpoint.Latest.Resources[0]
	
	// Check that missing dependency was removed
	for _, dep := range resource1.Dependencies {
		if dep == resource.URN("urn:pulumi:test::test::test::missing-dep") {
			t.Error("Missing dependency was not removed")
		}
	}

	// Check that missing property dependency was removed
	prop1Deps := resource1.PropertyDependencies["prop1"]
	for _, dep := range prop1Deps {
		if dep == resource.URN("urn:pulumi:test::test::test::missing-prop-dep") {
			t.Error("Missing property dependency was not removed")
		}
	}

	// Check that valid dependencies remain
	foundValidDep := false
	for _, dep := range resource1.Dependencies {
		if dep == resource.URN("urn:pulumi:test::test::test::resource2") {
			foundValidDep = true
		}
	}
	if !foundValidDep {
		t.Error("Valid dependency was incorrectly removed")
	}
}