package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRenderDiffJSON_NoChanges(t *testing.T) {
	out := captureStdout(t, func() {
		if err := renderDiffJSON(nil); err != nil {
			t.Fatal(err)
		}
	})

	var result []apitype.StepEventMetadata
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 changes, got %d", len(result))
	}
}

func TestRenderDiffJSON_WithChanges(t *testing.T) {
	outputs := []*apitype.ResOutputsEvent{
		{
			Metadata: apitype.StepEventMetadata{
				URN:  "urn:pulumi:dev::app::aws:ecs/service:Service::MyService",
				Type: "aws:ecs/service:Service",
				Op:   apitype.OpUpdate,
				DetailedDiff: map[string]apitype.PropertyDiff{
					"healthCheckGracePeriodSeconds": {Kind: apitype.DiffUpdate},
					"tags":                          {Kind: apitype.DiffAdd},
				},
				New: &apitype.StepEventStateMetadata{
					Outputs: map[string]interface{}{
						"healthCheckGracePeriodSeconds": 35,
						"tags":                          map[string]interface{}{"env": "dev"},
					},
				},
			},
		},
		{
			Metadata: apitype.StepEventMetadata{
				URN:  "urn:pulumi:dev::app::aws:s3/bucket:Bucket::MyBucket",
				Type: "aws:s3/bucket:Bucket",
				Op:   apitype.OpCreate,
			},
		},
		{
			Metadata: apitype.StepEventMetadata{
				URN:  "urn:pulumi:dev::app::aws:ecs/service:Service::Replacement",
				Type: "aws:ecs/service:Service",
				Op:   apitype.OpCreateReplacement,
			},
		},
		{
			Metadata: apitype.StepEventMetadata{
				URN:  "urn:pulumi:dev::app::aws:lambda/function:Function::OldFn",
				Type: "aws:lambda/function:Function",
				Op:   apitype.OpSame,
			},
		},
	}

	out := captureStdout(t, func() {
		if err := renderDiffJSON(outputs); err != nil {
			t.Fatal(err)
		}
	})

	var result []apitype.StepEventMetadata
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	// OpSame should be filtered out, but replacement steps should be kept.
	if len(result) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(result))
	}

	update := result[0]
	if update.Op != apitype.OpUpdate {
		t.Errorf("expected op 'update', got %q", update.Op)
	}
	if update.URN != "urn:pulumi:dev::app::aws:ecs/service:Service::MyService" {
		t.Errorf("unexpected URN: %s", update.URN)
	}
	if update.Type != "aws:ecs/service:Service" {
		t.Errorf("unexpected type: %s", update.Type)
	}
	if len(update.DetailedDiff) != 2 {
		t.Fatalf("expected 2 detailed diff entries, got %d", len(update.DetailedDiff))
	}
	if update.DetailedDiff["healthCheckGracePeriodSeconds"].Kind != apitype.DiffUpdate {
		t.Errorf("expected diff kind 'update', got %q", update.DetailedDiff["healthCheckGracePeriodSeconds"].Kind)
	}
	if update.DetailedDiff["tags"].Kind != apitype.DiffAdd {
		t.Errorf("expected diff kind 'add', got %q", update.DetailedDiff["tags"].Kind)
	}

	create := result[1]
	if create.Op != apitype.OpCreate {
		t.Errorf("expected op 'create', got %q", create.Op)
	}

	replacement := result[2]
	if replacement.Op != apitype.OpCreateReplacement {
		t.Errorf("expected op 'create-replacement', got %q", replacement.Op)
	}
}
