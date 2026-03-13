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

	var result DiffOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if len(result.Changes) != 0 {
		t.Fatalf("expected 0 changes, got %d", len(result.Changes))
	}
}

func TestRenderDiffJSON_WithChanges(t *testing.T) {
	outputs := []*apitype.ResOutputsEvent{
		{
			Metadata: apitype.StepEventMetadata{
				URN:  ("urn:pulumi:dev::app::aws:ecs/service:Service::MyService"),
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
				URN:  ("urn:pulumi:dev::app::aws:s3/bucket:Bucket::MyBucket"),
				Type: "aws:s3/bucket:Bucket",
				Op:   apitype.OpCreate,
			},
		},
		{
			Metadata: apitype.StepEventMetadata{
				URN:  ("urn:pulumi:dev::app::aws:lambda/function:Function::OldFn"),
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

	var result DiffOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	// OpSame should be filtered out
	if len(result.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(result.Changes))
	}

	update := result.Changes[0]
	if update.Operation != "update" {
		t.Errorf("expected operation 'update', got %q", update.Operation)
	}
	if update.URN != "urn:pulumi:dev::app::aws:ecs/service:Service::MyService" {
		t.Errorf("unexpected URN: %s", update.URN)
	}
	if update.Type != "aws:ecs/service:Service" {
		t.Errorf("unexpected type: %s", update.Type)
	}
	if len(update.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(update.Properties))
	}
	// Properties are sorted alphabetically
	if update.Properties[0].Path != "healthCheckGracePeriodSeconds" {
		t.Errorf("expected first property 'healthCheckGracePeriodSeconds', got %q", update.Properties[0].Path)
	}
	if update.Properties[0].Kind != "update" {
		t.Errorf("expected kind 'update', got %q", update.Properties[0].Kind)
	}
	if update.Properties[1].Path != "tags" {
		t.Errorf("expected second property 'tags', got %q", update.Properties[1].Path)
	}
	if update.Properties[1].Kind != "add" {
		t.Errorf("expected kind 'add', got %q", update.Properties[1].Kind)
	}

	create := result.Changes[1]
	if create.Operation != "create" {
		t.Errorf("expected operation 'create', got %q", create.Operation)
	}
	if len(create.Properties) != 0 {
		t.Errorf("expected 0 properties for create without detailed diff, got %d", len(create.Properties))
	}
}

func TestOpToString(t *testing.T) {
	cases := []struct {
		op   apitype.OpType
		want string
	}{
		{apitype.OpCreate, "create"},
		{apitype.OpUpdate, "update"},
		{apitype.OpDelete, "delete"},
		{apitype.OpReplace, "replace"},
		{apitype.OpImport, "import"},
		{apitype.OpSame, ""},
	}
	for _, tc := range cases {
		got := opToString(tc.op)
		if got != tc.want {
			t.Errorf("opToString(%v) = %q, want %q", tc.op, got, tc.want)
		}
	}
}

func TestDiffKindToString(t *testing.T) {
	cases := []struct {
		kind apitype.DiffKind
		want string
	}{
		{apitype.DiffAdd, "add"},
		{apitype.DiffDelete, "delete"},
		{apitype.DiffUpdate, "update"},
		{apitype.DiffAddReplace, "add-replace"},
		{apitype.DiffUpdateReplace, "update-replace"},
		{apitype.DiffDeleteReplace, "delete-replace"},
	}
	for _, tc := range cases {
		got := diffKindToString(tc.kind)
		if got != tc.want {
			t.Errorf("diffKindToString(%v) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}
