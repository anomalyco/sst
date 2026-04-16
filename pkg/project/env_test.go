package project

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/sst/sst/v3/pkg/project/provider"
)

type testProvider struct {
	env map[string]string
}

func (t testProvider) Init(app string, stage string, args map[string]interface{}) error {
	return nil
}

func (t testProvider) Env() (map[string]string, error) {
	return t.env, nil
}

func TestGenerateWranglerFileDefaultsCompatibilityFlags(t *testing.T) {
	project := &Project{
		root: t.TempDir(),
		app: &App{
			Stage: "test",
		},
		loadedProviders: map[string]provider.Provider{
			"cloudflare": testProvider{},
		},
	}

	config := generateWranglerConfig(t, project, Dev{
		Name: "MyWeb",
		Environment: map[string]string{
			"FOO": "bar",
		},
		Cloudflare: &DevCloudflare{
			Compatibility: &DevCloudflareCompatibility{
				Date:  stringPtr("2025-05-05"),
				Flags: []string{"nodejs_compat"},
			},
		},
	})

	if config.CompatibilityDate != "2025-05-05" {
		t.Fatalf("expected compatibility date %q, got %q", "2025-05-05", config.CompatibilityDate)
	}
	if !reflect.DeepEqual(config.CompatibilityFlags, []string{"nodejs_compat"}) {
		t.Fatalf("expected compatibility flags %v, got %v", []string{"nodejs_compat"}, config.CompatibilityFlags)
	}
}

func TestGenerateWranglerFileUsesDevCompatibility(t *testing.T) {
	project := &Project{
		root: t.TempDir(),
		app: &App{
			Stage: "test",
		},
		loadedProviders: map[string]provider.Provider{
			"cloudflare": testProvider{},
		},
	}

	date := "2026-01-01"
	config := generateWranglerConfig(t, project, Dev{
		Name: "MyWeb",
		Environment: map[string]string{
			"FOO": "bar",
		},
		Cloudflare: &DevCloudflare{
			Compatibility: &DevCloudflareCompatibility{
				Date:  &date,
				Flags: []string{"nodejs_compat_v2", "no_global_navigator"},
			},
		},
	})

	if config.CompatibilityDate != date {
		t.Fatalf("expected compatibility date %q, got %q", date, config.CompatibilityDate)
	}
	expectedFlags := []string{"nodejs_compat_v2", "no_global_navigator"}
	if !reflect.DeepEqual(config.CompatibilityFlags, expectedFlags) {
		t.Fatalf("expected compatibility flags %v, got %v", expectedFlags, config.CompatibilityFlags)
	}
}

func TestGenerateWranglerFileOmitsCompatibilityWhenMissing(t *testing.T) {
	project := &Project{
		root: t.TempDir(),
		app: &App{
			Stage: "test",
		},
		loadedProviders: map[string]provider.Provider{
			"cloudflare": testProvider{},
		},
	}

	config := generateWranglerConfigMap(t, project, Dev{
		Name: "MyWeb",
		Environment: map[string]string{
			"FOO": "bar",
		},
	})

	if _, ok := config["compatibility_date"]; ok {
		t.Fatalf("expected compatibility_date to be omitted")
	}
	if _, ok := config["compatibility_flags"]; ok {
		t.Fatalf("expected compatibility_flags to be omitted")
	}
}

type wranglerConfig struct {
	CompatibilityDate  string   `json:"compatibility_date"`
	CompatibilityFlags []string `json:"compatibility_flags"`
}

func stringPtr(input string) *string {
	return &input
}

func generateWranglerConfig(t *testing.T, project *Project, dev Dev) wranglerConfig {
	t.Helper()

	contents := generateWranglerConfigBytes(t, project, dev)

	var config wranglerConfig
	if err := json.Unmarshal(contents, &config); err != nil {
		t.Fatal(err)
	}

	return config
}

func generateWranglerConfigMap(t *testing.T, project *Project, dev Dev) map[string]interface{} {
	t.Helper()

	contents := generateWranglerConfigBytes(t, project, dev)

	var config map[string]interface{}
	if err := json.Unmarshal(contents, &config); err != nil {
		t.Fatal(err)
	}

	return config
}

func generateWranglerConfigBytes(t *testing.T, project *Project, dev Dev) []byte {
	t.Helper()

	path, err := project.generateWranglerFile(&CompleteEvent{}, dev)
	if err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return contents
}
