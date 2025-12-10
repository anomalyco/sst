package resource

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WorkerAssets struct {
	*CloudflareResource
}

type WorkerAssetsInputs struct {
	Manifest   AssetManifest `json:"manifest"`
	Directory  string `json:"directory"`
	AccountId  string `json:"accountId"`
	ApiToken   string `json:"apiToken"`
	ScriptName string `json:"scriptName"`
}

type WorkerAssetsOutputs struct {
	Manifest   AssetManifest `json:"manifest"`
	Directory  string `json:"directory"`
	AccountId  string `json:"accountId"`
	ScriptName string `json:"scriptName"`
	Jwt        string `json:"jwt"`
}

type AssetManifest map[string]AssetEntry

type AssetEntry struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
	ContentType string `json:"contentType"`
}

type InitializeAssetsResponse struct {
	Buckets [][]string `json:"buckets"`
	Jwt     string     `json:"jwt"`
}

type UploadResponse struct {
	Jwt string `json:"jwt"`
}


func (r *WorkerAssets) Create(input *WorkerAssetsInputs, output *CreateResult[WorkerAssetsOutputs]) error {
	jwt, err := r.handleUpload(input.Manifest, input.Directory, input.AccountId, input.ScriptName, input.ApiToken)
	if err != nil {
		return err
	}

	*output = CreateResult[WorkerAssetsOutputs]{
		ID:   "assets",
		Outs: WorkerAssetsOutputs{
			Manifest:   input.Manifest,
			Directory:  input.Directory,
			AccountId:  input.AccountId,
			ScriptName: input.ScriptName,
			Jwt:        jwt,
		},
	}
	return nil
}

func (r *WorkerAssets) Update(input *UpdateInput[WorkerAssetsInputs, WorkerAssetsOutputs], output *UpdateResult[WorkerAssetsOutputs]) error {
	jwt, err := r.handleUpload(input.News.Manifest, input.News.Directory, input.News.AccountId, input.News.ScriptName, input.News.ApiToken)
	if err != nil {
		return err
	}

	*output = UpdateResult[WorkerAssetsOutputs]{
		Outs: WorkerAssetsOutputs{
			Manifest:   input.News.Manifest,
			Directory:  input.News.Directory,
			AccountId:  input.News.AccountId,
			ScriptName: input.News.ScriptName,
			Jwt:        jwt,
		},
	}
	return nil
}

func (r *WorkerAssets) handleUpload(manifest AssetManifest, directory, accountId, scriptName, apiToken string) (string, error) {

	// Initialize assets upload session
	initResponse, err := r.uploadAssetManifest(accountId, scriptName, apiToken, manifest)
	if err != nil {
		return "", err
	}

	// No files to upload
	totalFilesToUpload := 0
	for _, bucket := range initResponse.Buckets {
		totalFilesToUpload += len(bucket)
	}
	if totalFilesToUpload == 0 {
		return initResponse.Jwt, nil
	}

	// Create channels for work distribution and error collection
	bucketsChan := make(chan []string)
	errChan := make(chan error, len(initResponse.Buckets))
	jwtChan := make(chan string, len(initResponse.Buckets))
	var wg sync.WaitGroup

	// Start worker pool (3 workers)
	numWorkers := 3
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each worker processes buckets from the channel
			for hashes := range bucketsChan {
				jwt, err := r.uploadAssets(manifest, directory, accountId, apiToken, hashes, initResponse.Jwt)
				if err != nil {
					errChan <- fmt.Errorf("bucket %v upload failed: %w", hashes, err)
					return
				}

				if jwt != "" {
					jwtChan <- jwt
				}
			}
		}()
	}

	// Send buckets to the channel
	go func() {
		for _, bucketHashes := range initResponse.Buckets {
			bucketsChan <- bucketHashes
		}
		close(bucketsChan)
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(errChan)
	close(jwtChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return "", err
		}
	}

	// Get completion JWT (from the last bucket response)
	var completionJwt string
	for jwt := range jwtChan {
		completionJwt = jwt
	}

	if completionJwt == "" {
		return "", fmt.Errorf("failed to complete asset upload - no completion JWT received")
	}

	return completionJwt, nil
}

func (r *WorkerAssets) uploadAssetManifest(accountId, scriptName, apiToken string, manifest AssetManifest) (*InitializeAssetsResponse, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/workers/scripts/%s/assets-upload-session", accountId, scriptName)

	jsonBody, err := json.Marshal(map[string]interface{}{
		"manifest": manifest,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// print out response body as a string
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to initialize assets upload: HTTP %d %s", resp.StatusCode, string(responseBody))
	}

	var result struct {
		Result InitializeAssetsResponse `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Result.Jwt == "" {
		return nil, fmt.Errorf("failed to initialize assets upload: no JWT received")
	}

	return &result.Result, nil
}

func (r *WorkerAssets) uploadAssets(manifest AssetManifest, directory, accountId, apiToken string, hashes []string, jwt string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, hash := range hashes {
		// Find the file path for this hash in the manifest
		var fileKey string
		var contentType string
		for path, entry := range manifest {
			if entry.Hash == hash {
				fileKey = path
				contentType = entry.ContentType
				break
			}
		}
		if fileKey == "" {
			return "", fmt.Errorf("hash %s not found in manifest", hash)
		}
		
		// Read file content
		absFilePath := filepath.Join(directory, fileKey)
		fileContent, err := os.ReadFile(absFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", absFilePath, err)
		}
		
		// Base64 encode the file content
		encodedContent := base64.StdEncoding.EncodeToString(fileContent)
		
		// Create form field with content type header
		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, hash, hash)},
			"Content-Type":        []string{contentType},
		})
		if err != nil {
			return "", fmt.Errorf("failed to create form part %s: %w", hash, err)
		}

		_, err = part.Write([]byte(encodedContent))
		if err != nil {
			return "", fmt.Errorf("failed to write encoded content for %s: %w", hash, err)
		}
	}
	err := writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/workers/assets/upload?base64=true", accountId)

	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+writer.Boundary())
	req.Header.Set("Authorization", "Bearer "+jwt)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// API returns:
	// - 202 Accepted if there are more buckets to upload
	// - 201 Created if all buckets have been uploaded
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		responseBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload assets: HTTP %d %s", resp.StatusCode, string(responseBody))
	}

	// Decode response
	var result struct {
		Result UploadResponse `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Result.Jwt, nil
}