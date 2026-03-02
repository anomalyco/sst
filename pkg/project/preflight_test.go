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

func TestCheckProviderUpgrade_AWSV6ToV7(t *testing.T) {
	p := &Project{
		lock: ProviderLock{
			{Name: "aws", Package: "@pulumi/aws", Alias: "aws", Version: "7.0.0"},
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

func TestCheckProviderUpgrade_AWSAlreadyOnV7(t *testing.T) {
	p := &Project{
		lock: ProviderLock{
			{Name: "aws", Package: "@pulumi/aws", Alias: "aws", Version: "7.0.0"},
		},
	}

	resources := []apitype.ResourceV3{
		makeProviderResource("aws", "7.0.0"),
	}

	messages := p.checkProviderUpgrade(resources)
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}
}

func TestCheckProviderUpgrade_NoMatchingProvider(t *testing.T) {
	p := &Project{
		lock: ProviderLock{
			{Name: "aws", Package: "@pulumi/aws", Alias: "aws", Version: "7.0.0"},
		},
	}

	resources := []apitype.ResourceV3{
		makeProviderResource("vercel", "1.11.0"),
	}

	messages := p.checkProviderUpgrade(resources)
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}
}
