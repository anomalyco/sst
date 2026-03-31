package resource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sst/sst/v3/pkg/project"
)

// Base resource for Vercel providers
type VercelResource struct {
	context context.Context
	project *project.Project
}

type VercelDnsRecord struct {
	*VercelResource
}

type VercelDnsRecordInputs struct {
	Domain    string `json:"domain"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	TeamId    string `json:"teamId,omitempty"`
	ApiToken  string `json:"apiToken"`
}

type VercelDnsRecordOutputs struct {
	RecordId string `json:"recordId"`
}


func (r *VercelDnsRecord) Create(input *VercelDnsRecordInputs, output *CreateResult[VercelDnsRecordOutputs]) error {
	recordId, err := r.createOrUpdateRecord(input)
	if err != nil {
		return err
	}

	*output = CreateResult[VercelDnsRecordOutputs]{
		ID:   recordId,
		Outs: VercelDnsRecordOutputs{RecordId: recordId},
	}
	return nil
}

func (r *VercelDnsRecord) Update(input *UpdateInput[VercelDnsRecordInputs, VercelDnsRecordOutputs], output *UpdateResult[VercelDnsRecordOutputs]) error {
	recordId, err := r.createOrUpdateRecord(&input.News)
	if err != nil {
		return err
	}

	*output = UpdateResult[VercelDnsRecordOutputs]{
		Outs: VercelDnsRecordOutputs{RecordId: recordId},
	}
	return nil
}

func (r *VercelDnsRecord) createOrUpdateRecord(input *VercelDnsRecordInputs) (string, error) {
	// Construct the URL with teamId if available
	url := fmt.Sprintf("https://api.vercel.com/v4/domains/%s/records", input.Domain)
	if input.TeamId != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, input.TeamId)
	}
	
	// Use the input type, name, and value for the DNS record
	recordPayload := struct {
		Type     string   `json:"type"`
		Name     string   `json:"name"`
		Value    string   `json:"value"`
		Ttl      int      `json:"ttl"`
	}{
		Type:     input.Type,
		Name:     input.Name,
		Value:    input.Value,
		Ttl:      60,
	}
	
	payloadBytes, err := json.Marshal(recordPayload)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+input.ApiToken)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	// Check if the response was successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Check for "record_exists" error
		errorMsg := string(body)
		
		// Parse error response
		var errorResponse struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		
		if err := json.Unmarshal(body, &errorResponse); err == nil && 
		   errorResponse.Error.Code == "record_exists" {
			// If the record already exists, we'll treat this as a success
			// We can attempt to find the existing record ID, but for now we'll just return a placeholder
			return "existing-record", nil
		}
		
		// Also check for legacy error messages for backward compatibility
		if resp.StatusCode == 400 && (bytes.Contains(body, []byte("already exists")) || 
		                            bytes.Contains(body, []byte("already registered"))) {
			return "existing-record", nil
		}
		
		return "", fmt.Errorf("failed to create DNS record, status: %d, response: %s", resp.StatusCode, errorMsg)
	}
	
	// Parse the response
	var apiResponse struct {
		Uid       string `json:"uid"`
	}
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return "", err
	}
	
	return apiResponse.Uid, nil
}