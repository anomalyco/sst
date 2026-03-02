package project

import (
	"strings"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

func makeProviderResource(provider string, version string) apitype.ResourceV3 {
	return apitype.ResourceV3{
		Type: tokens.Type("pulumi:providers:" + provider),
		Outputs: map[string]interface{}{
			"version": version,
		},
	}
}

func TestCheckProviderUpgrade_VercelV1ToV4(t *testing.T) {
	p := &Project{
		lock: ProviderLock{
			{Name: "@pulumiverse/vercel", Version: "4.6.0"},
		},
	}

	resources := []apitype.ResourceV3{
		makeProviderResource("@pulumiverse/vercel", "1.11.0"),
	}

	messages := p.checkProviderUpgrade(resources)
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if !strings.Contains(messages[0], "Vercel") {
		t.Fatalf("expected message to mention Vercel, got: %s", messages[0])
	}
}

func TestCheckProviderUpgrade_VercelAlreadyOnV4(t *testing.T) {
	p := &Project{
		lock: ProviderLock{
			{Name: "@pulumiverse/vercel", Version: "4.6.0"},
		},
	}

	resources := []apitype.ResourceV3{
		makeProviderResource("@pulumiverse/vercel", "4.5.0"),
	}

	messages := p.checkProviderUpgrade(resources)
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}
}

func TestCheckProviderUpgrade_NoVercelProvider(t *testing.T) {
	p := &Project{
		lock: ProviderLock{
			{Name: "@pulumiverse/vercel", Version: "4.6.0"},
		},
	}

	resources := []apitype.ResourceV3{
		makeProviderResource("aws", "6.5.0"),
	}

	messages := p.checkProviderUpgrade(resources)
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}
}

func TestCheckProviderUpgrade_AWSV6ToV7(t *testing.T) {
	p := &Project{
		lock: ProviderLock{
			{Name: "aws", Version: "7.0.0"},
		},
	}

	resources := []apitype.ResourceV3{
		makeProviderResource("aws", "6.52.0"),
	}

	messages := p.checkProviderUpgrade(resources)
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if !strings.Contains(messages[0], "AWS") {
		t.Fatalf("expected message to mention AWS, got: %s", messages[0])
	}
}

func TestCheckProviderUpgrade_BothProviders(t *testing.T) {
	p := &Project{
		lock: ProviderLock{
			{Name: "aws", Version: "7.0.0"},
			{Name: "@pulumiverse/vercel", Version: "4.6.0"},
		},
	}

	resources := []apitype.ResourceV3{
		makeProviderResource("aws", "6.52.0"),
		makeProviderResource("@pulumiverse/vercel", "1.11.0"),
	}

	messages := p.checkProviderUpgrade(resources)
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
}
