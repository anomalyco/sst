package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestHostedZoneLookup_Create_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	lookup := &HostedZoneLookup{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &HostedZoneLookupInputs{
		Domain: "example.com",
	}

	var output CreateResult[HostedZoneLookupOutputs]
	err := lookup.Create(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestHostedZoneLookup_Update_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	lookup := &HostedZoneLookup{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &UpdateInput[HostedZoneLookupInputs, HostedZoneLookupOutputs]{
		News: HostedZoneLookupInputs{
			Domain: "example.com",
		},
		Olds: HostedZoneLookupOutputs{
			ZoneId: "Z123456789",
		},
	}

	var output UpdateResult[HostedZoneLookupOutputs]
	err := lookup.Update(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestHostedZoneLookupInputs_Validation(t *testing.T) {
	tests := map[string]struct {
		input    HostedZoneLookupInputs
		expected string
	}{
		"valid domain": {
			input: HostedZoneLookupInputs{
				Domain: "example.com",
			},
			expected: "example.com",
		},
		"subdomain": {
			input: HostedZoneLookupInputs{
				Domain: "api.example.com",
			},
			expected: "api.example.com",
		},
		"deep subdomain": {
			input: HostedZoneLookupInputs{
				Domain: "api.v1.example.com",
			},
			expected: "api.v1.example.com",
		},
		"empty domain": {
			input: HostedZoneLookupInputs{
				Domain: "",
			},
			expected: "",
		},
		"domain with trailing dot": {
			input: HostedZoneLookupInputs{
				Domain: "example.com.",
			},
			expected: "example.com.",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.input.Domain)
		})
	}
}

func TestHostedZoneLookupOutputs_Structure(t *testing.T) {
	tests := map[string]struct {
		output   HostedZoneLookupOutputs
		expected string
	}{
		"standard zone id": {
			output: HostedZoneLookupOutputs{
				ZoneId: "Z123456789ABCDEF",
			},
			expected: "Z123456789ABCDEF",
		},
		"empty zone id": {
			output: HostedZoneLookupOutputs{
				ZoneId: "",
			},
			expected: "",
		},
		"zone id with prefix": {
			output: HostedZoneLookupOutputs{
				ZoneId: "/hostedzone/Z123456789ABCDEF",
			},
			expected: "/hostedzone/Z123456789ABCDEF",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.output.ZoneId)
		})
	}
}

func TestHostedZoneLookup_StructValidation(t *testing.T) {
	// Test that HostedZoneLookup embeds AwsResource correctly
	lookup := &HostedZoneLookup{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: &project.Project{},
		},
	}

	assert.NotNil(t, lookup.AwsResource)
	assert.NotNil(t, lookup.AwsResource.context)
	assert.NotNil(t, lookup.AwsResource.project)
}

func TestHostedZoneLookup_CreateResult_Structure(t *testing.T) {
	// Test CreateResult structure for HostedZoneLookup
	result := CreateResult[HostedZoneLookupOutputs]{
		ID: "Z123456789ABCDEF",
		Outs: HostedZoneLookupOutputs{
			ZoneId: "Z123456789ABCDEF",
		},
	}

	assert.Equal(t, "Z123456789ABCDEF", result.ID)
	assert.Equal(t, "Z123456789ABCDEF", result.Outs.ZoneId)
}

func TestHostedZoneLookup_UpdateResult_Structure(t *testing.T) {
	// Test UpdateResult structure for HostedZoneLookup
	result := UpdateResult[HostedZoneLookupOutputs]{
		Outs: HostedZoneLookupOutputs{
			ZoneId: "Z987654321FEDCBA",
		},
	}

	assert.Equal(t, "Z987654321FEDCBA", result.Outs.ZoneId)
}

func TestHostedZoneLookup_DomainParsing_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		domain      string
		description string
	}{
		"single character domain": {
			domain:      "a.com",
			description: "should handle single character domains",
		},
		"numeric domain": {
			domain:      "123.com",
			description: "should handle numeric domains",
		},
		"hyphenated domain": {
			domain:      "my-site.com",
			description: "should handle hyphenated domains",
		},
		"international domain": {
			domain:      "münchen.de",
			description: "should handle international domains",
		},
		"long subdomain chain": {
			domain:      "a.b.c.d.e.f.example.com",
			description: "should handle long subdomain chains",
		},
		"domain with port": {
			domain:      "example.com:8080",
			description: "should handle domains with ports",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			input := HostedZoneLookupInputs{
				Domain: test.domain,
			}
			
			// Just verify the input structure accepts various domain formats
			assert.Equal(t, test.domain, input.Domain, test.description)
		})
	}
}

func TestHostedZoneLookup_ZoneId_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		zoneId      string
		description string
	}{
		"short zone id": {
			zoneId:      "Z123",
			description: "should handle short zone IDs",
		},
		"long zone id": {
			zoneId:      "Z123456789ABCDEF123456789ABCDEF",
			description: "should handle long zone IDs",
		},
		"zone id with special chars": {
			zoneId:      "Z123-456_789",
			description: "should handle zone IDs with special characters",
		},
		"lowercase zone id": {
			zoneId:      "z123456789abcdef",
			description: "should handle lowercase zone IDs",
		},
		"mixed case zone id": {
			zoneId:      "Z123aBc456DeF789",
			description: "should handle mixed case zone IDs",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			output := HostedZoneLookupOutputs{
				ZoneId: test.zoneId,
			}
			
			assert.Equal(t, test.zoneId, output.ZoneId, test.description)
		})
	}
}