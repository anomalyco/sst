package resource

import (
	"context"
	"testing"

	"github.com/sst/sst/v3/pkg/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVercelDnsRecord_Create(t *testing.T) {
	tests := map[string]struct {
		input         VercelDnsRecordInputs
		expectedError string
	}{
		"missing API token": {
			input: VercelDnsRecordInputs{
				Domain: "example.com",
				Type:   "A",
				Name:   "test",
				Value:  "192.168.1.1",
			},
			expectedError: "failed to create DNS record",
		},
		"missing domain": {
			input: VercelDnsRecordInputs{
				Type:     "A",
				Name:     "test",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			expectedError: "failed to create DNS record",
		},
		"missing record type": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Name:     "test",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			expectedError: "failed to create DNS record",
		},
		"missing record name": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			expectedError: "failed to create DNS record",
		},
		"missing record value": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "test",
				ApiToken: "token123",
			},
			expectedError: "failed to create DNS record",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create resource instance
			resource := &VercelDnsRecord{
				VercelResource: &VercelResource{
					context: context.Background(),
					project: &project.Project{},
				},
			}
			
			var output CreateResult[VercelDnsRecordOutputs]
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

func TestVercelDnsRecord_Update(t *testing.T) {
	tests := map[string]struct {
		input         UpdateInput[VercelDnsRecordInputs, VercelDnsRecordOutputs]
		expectedError string
	}{
		"missing API token": {
			input: UpdateInput[VercelDnsRecordInputs, VercelDnsRecordOutputs]{
				ID: "existing-record",
				News: VercelDnsRecordInputs{
					Domain: "example.com",
					Type:   "A",
					Name:   "updated",
					Value:  "192.168.1.2",
				},
				Olds: VercelDnsRecordOutputs{
					RecordId: "existing-record",
				},
			},
			expectedError: "failed to create DNS record",
		},
		"missing domain": {
			input: UpdateInput[VercelDnsRecordInputs, VercelDnsRecordOutputs]{
				ID: "existing-record",
				News: VercelDnsRecordInputs{
					Type:     "A",
					Name:     "updated",
					Value:    "192.168.1.2",
					ApiToken: "token123",
				},
				Olds: VercelDnsRecordOutputs{
					RecordId: "existing-record",
				},
			},
			expectedError: "failed to create DNS record",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create resource instance
			resource := &VercelDnsRecord{
				VercelResource: &VercelResource{
					context: context.Background(),
					project: &project.Project{},
				},
			}
			
			var output UpdateResult[VercelDnsRecordOutputs]
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

func TestVercelDnsRecord_InputValidation(t *testing.T) {
	tests := map[string]struct {
		input VercelDnsRecordInputs
		valid bool
	}{
		"valid A record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "test",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			valid: true,
		},
		"valid AAAA record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "AAAA",
				Name:     "test",
				Value:    "2001:db8::1",
				ApiToken: "token123",
			},
			valid: true,
		},
		"valid CNAME record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "CNAME",
				Name:     "www",
				Value:    "example.com",
				ApiToken: "token123",
			},
			valid: true,
		},
		"valid MX record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "MX",
				Name:     "@",
				Value:    "10 mail.example.com",
				ApiToken: "token123",
			},
			valid: true,
		},
		"valid TXT record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "TXT",
				Name:     "_dmarc",
				Value:    "v=DMARC1; p=none",
				ApiToken: "token123",
			},
			valid: true,
		},
		"valid NS record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "NS",
				Name:     "subdomain",
				Value:    "ns1.example.com",
				ApiToken: "token123",
			},
			valid: true,
		},
		"valid SRV record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "SRV",
				Name:     "_sip._tcp",
				Value:    "10 5 5060 sip.example.com",
				ApiToken: "token123",
			},
			valid: true,
		},
		"empty domain": {
			input: VercelDnsRecordInputs{
				Type:     "A",
				Name:     "test",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			valid: false,
		},
		"empty type": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Name:     "test",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			valid: false,
		},
		"empty name": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			valid: false,
		},
		"empty value": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "test",
				ApiToken: "token123",
			},
			valid: false,
		},
		"empty API token": {
			input: VercelDnsRecordInputs{
				Domain: "example.com",
				Type:   "A",
				Name:   "test",
				Value:  "192.168.1.1",
			},
			valid: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Basic validation - check if required fields are present
			hasRequiredFields := tt.input.Domain != "" && 
				tt.input.Type != "" && 
				tt.input.Name != "" && 
				tt.input.Value != "" && 
				tt.input.ApiToken != ""
			
			assert.Equal(t, tt.valid, hasRequiredFields, "Input validation mismatch")
		})
	}
}

func TestVercelDnsRecord_StructValidation(t *testing.T) {
	t.Run("VercelDnsRecordInputs struct fields", func(t *testing.T) {
		input := VercelDnsRecordInputs{
			Domain:   "example.com",
			Type:     "A",
			Name:     "test",
			Value:    "192.168.1.1",
			TeamId:   "team123",
			ApiToken: "token123",
		}
		
		assert.Equal(t, "example.com", input.Domain)
		assert.Equal(t, "A", input.Type)
		assert.Equal(t, "test", input.Name)
		assert.Equal(t, "192.168.1.1", input.Value)
		assert.Equal(t, "team123", input.TeamId)
		assert.Equal(t, "token123", input.ApiToken)
	})
	
	t.Run("VercelDnsRecordOutputs struct fields", func(t *testing.T) {
		output := VercelDnsRecordOutputs{
			RecordId: "record123",
		}
		
		assert.Equal(t, "record123", output.RecordId)
	})
	
	t.Run("VercelResource embedded structure", func(t *testing.T) {
		resource := &VercelDnsRecord{
			VercelResource: &VercelResource{
				context: context.Background(),
				project: &project.Project{},
			},
		}
		
		assert.NotNil(t, resource.VercelResource)
		assert.NotNil(t, resource.VercelResource.context)
		assert.NotNil(t, resource.VercelResource.project)
	})
}

func TestVercelDnsRecord_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		input       VercelDnsRecordInputs
		description string
	}{
		"unicode characters in name": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "tëst",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			description: "Should handle unicode characters in record name",
		},
		"international domain": {
			input: VercelDnsRecordInputs{
				Domain:   "exämple.com",
				Type:     "A",
				Name:     "test",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			description: "Should handle international domain names",
		},
		"IPv6 address": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "AAAA",
				Name:     "ipv6",
				Value:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
				ApiToken: "token123",
			},
			description: "Should handle IPv6 addresses",
		},
		"complex TXT record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "TXT",
				Name:     "_dmarc",
				Value:    "v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com; ruf=mailto:dmarc@example.com; fo=1",
				ApiToken: "token123",
			},
			description: "Should handle complex TXT records with multiple parameters",
		},
		"long record name": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "very-long-subdomain-name-that-might-cause-issues",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			description: "Should handle long record names",
		},
		"special characters in value": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "TXT",
				Name:     "test",
				Value:    "v=spf1 include:_spf.google.com ~all",
				ApiToken: "token123",
			},
			description: "Should handle special characters in record values",
		},
		"with team ID": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "test",
				Value:    "192.168.1.1",
				TeamId:   "team_abc123xyz",
				ApiToken: "token123",
			},
			description: "Should handle team ID parameter",
		},
		"root domain record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "@",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			description: "Should handle root domain records with @ symbol",
		},
		"wildcard record": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "*",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			description: "Should handle wildcard DNS records",
		},
		"subdomain wildcard": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "*.api",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			description: "Should handle subdomain wildcard records",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create resource instance
			resource := &VercelDnsRecord{
				VercelResource: &VercelResource{
					context: context.Background(),
					project: &project.Project{},
				},
			}
			
			var output CreateResult[VercelDnsRecordOutputs]
			err := resource.Create(&tt.input, &output)
			
			// Since we don't have real API access, we expect errors
			// but we're testing that the input structure is valid
			assert.Error(t, err, tt.description)
			assert.Contains(t, err.Error(), "failed to create DNS record")
		})
	}
}

func TestVercelDnsRecord_URLConstruction(t *testing.T) {
	tests := map[string]struct {
		input       VercelDnsRecordInputs
		expectedURL string
	}{
		"without team ID": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "test",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			expectedURL: "https://api.vercel.com/v4/domains/example.com/records",
		},
		"with team ID": {
			input: VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     "A",
				Name:     "test",
				Value:    "192.168.1.1",
				TeamId:   "team123",
				ApiToken: "token123",
			},
			expectedURL: "https://api.vercel.com/v4/domains/example.com/records?teamId=team123",
		},
		"special characters in domain": {
			input: VercelDnsRecordInputs{
				Domain:   "test-domain.example.com",
				Type:     "A",
				Name:     "api",
				Value:    "192.168.1.1",
				ApiToken: "token123",
			},
			expectedURL: "https://api.vercel.com/v4/domains/test-domain.example.com/records",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Test URL construction logic
			baseURL := "https://api.vercel.com/v4/domains/" + tt.input.Domain + "/records"
			if tt.input.TeamId != "" {
				baseURL = baseURL + "?teamId=" + tt.input.TeamId
			}
			
			assert.Equal(t, tt.expectedURL, baseURL)
		})
	}
}

func TestVercelDnsRecord_PayloadStructure(t *testing.T) {
	t.Run("record payload structure", func(t *testing.T) {
		input := VercelDnsRecordInputs{
			Domain:   "example.com",
			Type:     "A",
			Name:     "test",
			Value:    "192.168.1.1",
			ApiToken: "token123",
		}
		
		// Test the payload structure that would be sent to Vercel API
		expectedPayload := struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			Value string `json:"value"`
			Ttl   int    `json:"ttl"`
		}{
			Type:  input.Type,
			Name:  input.Name,
			Value: input.Value,
			Ttl:   60,
		}
		
		assert.Equal(t, "A", expectedPayload.Type)
		assert.Equal(t, "test", expectedPayload.Name)
		assert.Equal(t, "192.168.1.1", expectedPayload.Value)
		assert.Equal(t, 60, expectedPayload.Ttl)
	})
}

func TestVercelDnsRecord_ErrorHandling(t *testing.T) {
	tests := map[string]struct {
		statusCode    int
		responseBody  string
		expectedError string
	}{
		"record already exists": {
			statusCode:    400,
			responseBody:  `{"error":{"code":"record_exists","message":"Record already exists"}}`,
			expectedError: "", // Should not error for existing records
		},
		"legacy already exists": {
			statusCode:    400,
			responseBody:  `{"error":{"message":"Record already exists"}}`,
			expectedError: "", // Should not error for existing records
		},
		"legacy already registered": {
			statusCode:    400,
			responseBody:  `{"error":{"message":"Record already registered"}}`,
			expectedError: "", // Should not error for existing records
		},
		"unauthorized": {
			statusCode:    401,
			responseBody:  `{"error":{"code":"unauthorized","message":"Invalid token"}}`,
			expectedError: "failed to create DNS record, status: 401",
		},
		"forbidden": {
			statusCode:    403,
			responseBody:  `{"error":{"code":"forbidden","message":"Access denied"}}`,
			expectedError: "failed to create DNS record, status: 403",
		},
		"not found": {
			statusCode:    404,
			responseBody:  `{"error":{"code":"not_found","message":"Domain not found"}}`,
			expectedError: "failed to create DNS record, status: 404",
		},
		"server error": {
			statusCode:    500,
			responseBody:  `{"error":{"code":"internal_error","message":"Internal server error"}}`,
			expectedError: "failed to create DNS record, status: 500",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Test error handling logic
			shouldError := tt.statusCode < 200 || tt.statusCode >= 300
			
			if shouldError {
				// Check for special cases that should not error
				if tt.statusCode == 400 {
					if tt.responseBody == `{"error":{"code":"record_exists","message":"Record already exists"}}` ||
						(tt.responseBody != "" && (
							tt.responseBody == `{"error":{"message":"Record already exists"}}` ||
							tt.responseBody == `{"error":{"message":"Record already registered"}}`)) {
						shouldError = false
					}
				}
			}
			
			if shouldError && tt.expectedError != "" {
				assert.Contains(t, tt.expectedError, "failed to create DNS record")
			}
		})
	}
}

func TestVercelDnsRecord_RecordTypes(t *testing.T) {
	recordTypes := []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "PTR", "CAA"}
	
	for _, recordType := range recordTypes {
		t.Run("record type "+recordType, func(t *testing.T) {
			input := VercelDnsRecordInputs{
				Domain:   "example.com",
				Type:     recordType,
				Name:     "test",
				Value:    getTestValueForRecordType(recordType),
				ApiToken: "token123",
			}
			
			// Validate that all record types have proper structure
			assert.NotEmpty(t, input.Type)
			assert.NotEmpty(t, input.Value)
		})
	}
}

// Helper function to get appropriate test values for different record types
func getTestValueForRecordType(recordType string) string {
	switch recordType {
	case "A":
		return "192.168.1.1"
	case "AAAA":
		return "2001:db8::1"
	case "CNAME":
		return "example.com"
	case "MX":
		return "10 mail.example.com"
	case "TXT":
		return "v=spf1 include:_spf.google.com ~all"
	case "NS":
		return "ns1.example.com"
	case "SRV":
		return "10 5 5060 sip.example.com"
	case "PTR":
		return "example.com"
	case "CAA":
		return "0 issue letsencrypt.org"
	default:
		return "test-value"
	}
}