package project

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sst/sst/v3/pkg/project/common"
)

func TestEnvForIncludesWranglerPath(t *testing.T) {
	project := &Project{
		app: &App{
			Name:  "my-app",
			Stage: "test",
		},
	}

	complete := &CompleteEvent{
		Devs: Devs{
			"MyWeb": {
				Name:        "MyWeb",
				Environment: map[string]string{"FOO": "bar"},
				Links:       []string{"MyKv"},
				Cloudflare: &DevCloudflare{
					Path: "/tmp/MyWeb.jsonc",
				},
			},
		},
		Links: common.Links{
			"MyKv": {
				Properties: map[string]interface{}{
					"namespaceId": "1234",
				},
			},
		},
	}

	env, err := project.EnvFor(context.Background(), complete, "MyWeb")
	if err != nil {
		t.Fatal(err)
	}

	if got := env["SST_WRANGLER_PATH"]; got != "/tmp/MyWeb.jsonc" {
		t.Fatalf("expected SST_WRANGLER_PATH to be set, got %q", got)
	}
	if got := env["FOO"]; got != "bar" {
		t.Fatalf("expected FOO to be preserved, got %q", got)
	}

	var resource map[string]interface{}
	if err := json.Unmarshal([]byte(env["SST_RESOURCE_MyKv"]), &resource); err != nil {
		t.Fatal(err)
	}
	if got := resource["namespaceId"]; got != "1234" {
		t.Fatalf("expected linked resource data, got %#v", resource)
	}
}

func TestEnvForOmitsWranglerPathWhenMissing(t *testing.T) {
	project := &Project{
		app: &App{
			Name:  "my-app",
			Stage: "test",
		},
	}

	complete := &CompleteEvent{
		Devs: Devs{
			"MyWeb": {
				Name: "MyWeb",
			},
		},
	}

	env, err := project.EnvFor(context.Background(), complete, "MyWeb")
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := env["SST_WRANGLER_PATH"]; ok {
		t.Fatalf("expected SST_WRANGLER_PATH to be omitted")
	}
}
