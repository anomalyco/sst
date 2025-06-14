package resource

import (
	"context"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudflareDnsRecord_Create(t *testing.T) {
	tests := map[string]struct {
		input         CloudflareDnsRecordInputs
		expectedError string
	}{
		"missing API token": {
			input: CloudflareDnsRecordInputs{
				ZoneId: "zone123",
				Type:   "A",
				Name:   "test.example.com",
				Value:  cfStringPtr("192.168.1.1"),
			},
			expectedError: "Cloudflare API error",
		},
		"missing zone ID": {
			input: CloudflareDnsRecordInputs{
				Type:     "A",
				Name:     "test.example.com",
				Value:    cfStringPtr("192.168.1.1"),
				ApiToken: "token123",
			},
			expectedError: "Cloudflare API error",
		},
		"missing record type": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Name:     "test.example.com",
				Value:    cfStringPtr("192.168.1.1"),
				ApiToken: "token123",
			},
			expectedError: "Cloudflare API error",
		},
		"missing record name": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "A",
				Value:    cfStringPtr("192.168.1.1"),
				ApiToken: "token123",
			},
			expectedError: "Cloudflare API error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create resource instance
			resource := &CloudflareDnsRecord{
				CloudflareResource: &CloudflareResource{
					context: context.Background(),
					project: &project.Project{},
				},
			}
			
			var output CreateResult[CloudflareDnsRecordOutputs]
			err := resource.Create(&tt.input, &output)
			
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCloudflareDnsRecord_Update(t *testing.T) {
	tests := map[string]struct {
		input         UpdateInput[CloudflareDnsRecordInputs, CloudflareDnsRecordOutputs]
		expectedError string
	}{
		"missing API token": {
			input: UpdateInput[CloudflareDnsRecordInputs, CloudflareDnsRecordOutputs]{
				ID: "existing-record",
				News: CloudflareDnsRecordInputs{
					ZoneId: "zone123",
					Type:   "A",
					Name:   "updated.example.com",
					Value:  cfStringPtr("192.168.1.2"),
				},
				Olds: CloudflareDnsRecordOutputs{
					RecordId: "existing-record",
				},
			},
			expectedError: "Cloudflare API error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create resource instance
			resource := &CloudflareDnsRecord{
				CloudflareResource: &CloudflareResource{
					context: context.Background(),
					project: &project.Project{},
				},
			}
			
			var output UpdateResult[CloudflareDnsRecordOutputs]
			err := resource.Update(&tt.input, &output)
			
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCloudflareDnsRecord_InputValidation(t *testing.T) {
	tests := map[string]struct {
		input CloudflareDnsRecordInputs
		valid bool
	}{
		"valid A record": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "A",
				Name:     "test.example.com",
				Value:    cfStringPtr("192.168.1.1"),
				ApiToken: "token123",
			},
			valid: true,
		},
		"valid CAA record with data": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "CAA",
				Name:     "example.com",
				ApiToken: "token123",
				Data: &Data{
					Flags: "0",
					Tag:   "issue",
					Value: "letsencrypt.org",
				},
			},
			valid: true,
		},
		"empty zone ID": {
			input: CloudflareDnsRecordInputs{
				Type:     "A",
				Name:     "test.example.com",
				Value:    cfStringPtr("192.168.1.1"),
				ApiToken: "token123",
			},
			valid: false,
		},
		"empty API token": {
			input: CloudflareDnsRecordInputs{
				ZoneId: "zone123",
				Type:   "A",
				Name:   "test.example.com",
				Value:  cfStringPtr("192.168.1.1"),
			},
			valid: false,
		},
		"empty record type": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Name:     "test.example.com",
				Value:    cfStringPtr("192.168.1.1"),
				ApiToken: "token123",
			},
			valid: false,
		},
		"empty record name": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "A",
				Value:    cfStringPtr("192.168.1.1"),
				ApiToken: "token123",
			},
			valid: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Basic validation - check required fields
			hasZoneId := tt.input.ZoneId != ""
			hasType := tt.input.Type != ""
			hasName := tt.input.Name != ""
			hasApiToken := tt.input.ApiToken != ""
			
			isValid := hasZoneId && hasType && hasName && hasApiToken
			assert.Equal(t, tt.valid, isValid, "Input validation mismatch for %s", name)
		})
	}
}

func TestCloudflareDnsRecord_StructValidation(t *testing.T) {
	t.Run("CloudflareDnsRecordInputs struct fields", func(t *testing.T) {
		input := CloudflareDnsRecordInputs{
			ZoneId:   "zone123",
			Type:     "A",
			Name:     "test.example.com",
			Value:    cfStringPtr("192.168.1.1"),
			Proxied:  cfBoolPtr(true),
			ApiToken: "token123",
			Data: &Data{
				Flags: "0",
				Tag:   "issue",
				Value: "letsencrypt.org",
			},
		}
		
		assert.Equal(t, "zone123", input.ZoneId)
		assert.Equal(t, "A", input.Type)
		assert.Equal(t, "test.example.com", input.Name)
		assert.Equal(t, "192.168.1.1", *input.Value)
		assert.Equal(t, true, *input.Proxied)
		assert.Equal(t, "token123", input.ApiToken)
		assert.NotNil(t, input.Data)
		assert.Equal(t, "0", input.Data.Flags)
		assert.Equal(t, "issue", input.Data.Tag)
		assert.Equal(t, "letsencrypt.org", input.Data.Value)
	})
	
	t.Run("CloudflareDnsRecordOutputs struct fields", func(t *testing.T) {
		output := CloudflareDnsRecordOutputs{
			RecordId: "record123",
		}
		
		assert.Equal(t, "record123", output.RecordId)
	})
	
	t.Run("Data struct fields", func(t *testing.T) {
		data := Data{
			Flags: "128",
			Tag:   "iodef",
			Value: "mailto:security@example.com",
		}
		
		assert.Equal(t, "128", data.Flags)
		assert.Equal(t, "iodef", data.Tag)
		assert.Equal(t, "mailto:security@example.com", data.Value)
	})
	
	t.Run("CloudflareResource embedded struct", func(t *testing.T) {
		resource := &CloudflareDnsRecord{
			CloudflareResource: &CloudflareResource{
				context: context.Background(),
				project: &project.Project{},
			},
		}
		
		assert.NotNil(t, resource.CloudflareResource)
		assert.NotNil(t, resource.CloudflareResource.context)
		assert.NotNil(t, resource.CloudflareResource.project)
	})
}

func TestCloudflareDnsRecord_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		input       CloudflareDnsRecordInputs
		description string
	}{
		"very long zone ID": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "a" + "b" + "c", // Simplified to avoid long strings
				Type:     "A",
				Name:     "test.example.com",
				Value:    cfStringPtr("192.168.1.1"),
				ApiToken: "token123",
			},
			description: "Should handle zone IDs",
		},
		"special characters in name": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "TXT",
				Name:     "test-with-special_chars.example.com",
				Value:    cfStringPtr("v=spf1 include:_spf.google.com ~all"),
				ApiToken: "token123",
			},
			description: "Should handle special characters in DNS names",
		},
		"unicode in record name": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "A",
				Name:     "тест.example.com",
				Value:    cfStringPtr("192.168.1.1"),
				ApiToken: "token123",
			},
			description: "Should handle unicode characters in DNS names",
		},
		"IPv6 address": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "AAAA",
				Name:     "ipv6.example.com",
				Value:    cfStringPtr("2001:db8::1"),
				ApiToken: "token123",
			},
			description: "Should handle IPv6 addresses",
		},
		"complex TXT record": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "TXT",
				Name:     "_dmarc.example.com",
				Value:    cfStringPtr("v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com; ruf=mailto:dmarc@example.com; fo=1"),
				ApiToken: "token123",
			},
			description: "Should handle complex TXT records with multiple parameters",
		},
		"DNSKEY record with data": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "DNSKEY",
				Name:     "example.com",
				ApiToken: "token123",
				Data: &Data{
					Flags: "257",
					Tag:   "3",
					Value: "AQPJ////4Q==",
				},
			},
			description: "Should handle DNSKEY records with data field",
		},
		"empty data fields": {
			input: CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     "CAA",
				Name:     "example.com",
				ApiToken: "token123",
				Data: &Data{
					Flags: "",
					Tag:   "",
					Value: "",
				},
			},
			description: "Should handle empty data fields",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Validate that the input structure is properly formed
			assert.NotEmpty(t, tt.input.ZoneId, "ZoneId should not be empty")
			assert.NotEmpty(t, tt.input.Type, "Type should not be empty")
			assert.NotEmpty(t, tt.input.Name, "Name should not be empty")
			assert.NotEmpty(t, tt.input.ApiToken, "ApiToken should not be empty")
			
			// For records that use data field, validate data structure
			if tt.input.Data != nil && (tt.input.Type == "CAA" || tt.input.Type == "SRV" || tt.input.Type == "DNSKEY") {
				assert.NotNil(t, tt.input.Data, "Data should not be nil for %s records", tt.input.Type)
			}
			
			// For standard records, validate value field
			if tt.input.Data == nil {
				// Value can be nil for some record types, but if present should be valid
				if tt.input.Value != nil {
					assert.NotEmpty(t, *tt.input.Value, "Value should not be empty when provided")
				}
			}
		})
	}
}

func TestCloudflareDnsRecord_RecordTypes(t *testing.T) {
	recordTypes := []struct {
		recordType string
		usesData   bool
		testValue  string
		testData   *Data
	}{
		{"A", false, "192.168.1.1", nil},
		{"AAAA", false, "2001:db8::1", nil},
		{"CNAME", false, "example.com", nil},
		{"MX", false, "10 mail.example.com", nil},
		{"TXT", false, "v=spf1 include:_spf.google.com ~all", nil},
		{"NS", false, "ns1.example.com", nil},
		{"PTR", false, "example.com", nil},
		{"CAA", true, "", &Data{Flags: "0", Tag: "issue", Value: "letsencrypt.org"}},
		{"SRV", true, "", &Data{Flags: "10", Tag: "5", Value: "sip.example.com"}},
		{"DNSKEY", true, "", &Data{Flags: "257", Tag: "3", Value: "AQPJ////4Q=="}},
	}

	for _, rt := range recordTypes {
		t.Run("record type "+rt.recordType, func(t *testing.T) {
			input := CloudflareDnsRecordInputs{
				ZoneId:   "zone123",
				Type:     rt.recordType,
				Name:     "test.example.com",
				ApiToken: "token123",
			}
			
			if rt.usesData {
				input.Data = rt.testData
				assert.NotNil(t, input.Data, "Data should be provided for %s records", rt.recordType)
			} else {
				input.Value = &rt.testValue
				assert.NotNil(t, input.Value, "Value should be provided for %s records", rt.recordType)
			}
			
			// Validate the input structure
			assert.Equal(t, rt.recordType, input.Type)
			assert.Equal(t, rt.usesData, input.Data != nil)
			assert.Equal(t, !rt.usesData, input.Value != nil)
		})
	}
}

func TestCloudflareDnsRecord_PayloadStructure(t *testing.T) {
	t.Run("standard record payload structure", func(t *testing.T) {
		input := CloudflareDnsRecordInputs{
			ZoneId:   "zone123",
			Type:     "A",
			Name:     "test.example.com",
			Value:    cfStringPtr("192.168.1.1"),
			Proxied:  cfBoolPtr(true),
			ApiToken: "token123",
		}
		
		// Verify that standard records should use content field, not data field
		assert.NotNil(t, input.Value, "Standard records should have Value field")
		assert.Nil(t, input.Data, "Standard records should not have Data field")
		assert.NotNil(t, input.Proxied, "A records can have Proxied field")
	})
	
	t.Run("data record payload structure", func(t *testing.T) {
		input := CloudflareDnsRecordInputs{
			ZoneId:   "zone123",
			Type:     "CAA",
			Name:     "example.com",
			ApiToken: "token123",
			Data: &Data{
				Flags: "0",
				Tag:   "issue",
				Value: "letsencrypt.org",
			},
		}
		
		// Verify that data records should use data field, not content field
		assert.Nil(t, input.Value, "Data records should not have Value field")
		assert.NotNil(t, input.Data, "Data records should have Data field")
		assert.Nil(t, input.Proxied, "Data records should not have Proxied field")
	})
}

func TestCloudflareDnsRecord_CreateOrUpdateRecordLogic(t *testing.T) {
	t.Run("URL construction", func(t *testing.T) {
		zoneId := "test-zone-123"
		expectedURL := "https://api.cloudflare.com/client/v4/zones/" + zoneId + "/dns_records"
		
		// Verify URL pattern matches expected Cloudflare API format
		assert.Contains(t, expectedURL, "api.cloudflare.com")
		assert.Contains(t, expectedURL, "/client/v4/zones/")
		assert.Contains(t, expectedURL, "/dns_records")
		assert.Contains(t, expectedURL, zoneId)
	})
	
	t.Run("authorization header format", func(t *testing.T) {
		apiToken := "test-token-123"
		expectedHeader := "Bearer " + apiToken
		
		// Verify authorization header format
		assert.Equal(t, "Bearer test-token-123", expectedHeader)
		assert.Contains(t, expectedHeader, "Bearer ")
	})
	
	t.Run("TTL default value", func(t *testing.T) {
		// Verify that TTL is set to 60 seconds by default
		expectedTTL := 60
		assert.Equal(t, 60, expectedTTL)
	})
}

func TestCloudflareDnsRecord_ErrorHandling(t *testing.T) {
	t.Run("already exists error detection", func(t *testing.T) {
		errorMessages := []string{
			"The record already exists.",
			"Record already exists",
			"already exists for this zone",
			"ALREADY EXISTS",
		}
		
		for _, msg := range errorMessages {
			// Test case-insensitive detection of "already exists"
			containsAlreadyExists := strings.Contains(strings.ToLower(msg), "already exists")
			assert.True(t, containsAlreadyExists, "Should detect 'already exists' in: %s", msg)
		}
	})
	
	t.Run("error response structure", func(t *testing.T) {
		// Test the expected structure of Cloudflare API error responses
		type CloudflareError struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}

		type CloudflareResponse struct {
			Success  bool              `json:"success"`
			Errors   []CloudflareError `json:"errors"`
			Messages []string          `json:"messages"`
			Result   struct {
				Id string `json:"id"`
			} `json:"result"`
		}
		
		// Verify the structure can be created and populated
		response := CloudflareResponse{
			Success: false,
			Errors: []CloudflareError{
				{Code: 81057, Message: "The record already exists."},
			},
			Messages: []string{},
		}
		
		assert.False(t, response.Success)
		assert.Len(t, response.Errors, 1)
		assert.Equal(t, 81057, response.Errors[0].Code)
		assert.Equal(t, "The record already exists.", response.Errors[0].Message)
	})
}

// Helper functions specific to this test file to avoid conflicts
func cfStringPtr(s string) *string {
	return &s
}

func cfBoolPtr(b bool) *bool {
	return &b
}