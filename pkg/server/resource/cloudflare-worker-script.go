package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type WorkerScript struct {
	*CloudflareResource
}

type WorkerScriptInputs struct {
	AccountId  string `json:"accountId"`
	ApiToken   string `json:"apiToken"`
	ScriptName string `json:"scriptName"`
	Content    struct {
		Filename string `json:"filename"`
		Hash     string `json:"hash"`
	} `json:"content"`
	Assets     struct {
		Jwt string `json:"jwt,omitempty"`
		Config struct {
			Headers string `json:"_headers,omitempty"`
			Redirects string `json:"_redirects,omitempty"`
			HtmlHandling string `json:"htmlHandling,omitempty"`
			NotFoundHandling string `json:"notFoundHandling,omitempty"`
			RunWorkerFirst bool `json:"runWorkerFirst,omitempty"`
		} `json:"config,omitempty"`
	} `json:"assets,omitempty"`
	Bindings   []struct {
		Type       string `json:"type"`
		Name       string `json:"name"`
		BucketName string `json:"bucketName,omitempty"`
		ClassName  string `json:"className,omitempty"`
		NamespaceId string `json:"namespaceId,omitempty"`
		QueueName  string `json:"queueName,omitempty"`
		ScriptName string `json:"scriptName,omitempty"`
		SecretName string `json:"secretName,omitempty"`
		Service    string `json:"service,omitempty"`
		Text       string `json:"text,omitempty"`
	} `json:"bindings,omitempty"`
	BodyPart   string `json:"bodyPart,omitempty"`
	CompatibilityDate string `json:"compatibilityDate,omitempty"`
	CompatibilityFlags []string `json:"compatibilityFlags,omitempty"`
	KeepAssets         bool `json:"keepAssets,omitempty"`
	KeepBindings       []string `json:"keepBindings,omitempty"`
	Logpush            bool `json:"logpush,omitempty"`
	MainModule         string `json:"mainModule,omitempty"`
	Migrations         struct {
		DeletedClasses []string `json:"deletedClasses,omitempty"`
		NewClasses     []string `json:"newClasses,omitempty"`
		NewSqliteClasses []string `json:"newSqliteClasses,omitempty"`
		NewTag string `json:"newTag,omitempty"`
		OldTag string `json:"oldTag,omitempty"`
		RenamedClasses []struct {
			From string `json:"from,omitempty"`
			To string `json:"to,omitempty"`
		} `json:"renamedClasses,omitempty"`
		Steps []struct {
			DeletedClasses []string `json:"deletedClasses,omitempty"`
			NewClasses     []string `json:"newClasses,omitempty"`
			NewSqliteClasses []string `json:"newSqliteClasses,omitempty"`
			RenamedClasses []struct {
				From string `json:"from,omitempty"`
				To string `json:"to,omitempty"`
			} `json:"renamedClasses,omitempty"`
			TransferredClasses []struct {
				From string `json:"from,omitempty"`
				FromScript string `json:"fromScript,omitempty"`
				To string `json:"to,omitempty"`
			} `json:"transferredClasses,omitempty"`
		} `json:"steps,omitempty"`
		TransferredClasses []struct {
			From string `json:"from,omitempty"`
			FromScript string `json:"fromScript,omitempty"`
			To string `json:"to,omitempty"`
		} `json:"transferredClasses,omitempty"`
	} `json:"migrations,omitempty"`
	Observability struct {
		Enabled bool `json:"enabled,omitempty"`
		HeapSamplingRate int `json:"heapSamplingRate,omitempty"`
	} `json:"observability,omitempty"`
	Placement struct {
		LastAnalyzedAt string `json:"lastAnalyzedAt,omitempty"`
		Mode string `json:"mode,omitempty"`
		Status string `json:"status,omitempty"`
	} `json:"placement,omitempty"`
	TailConsumers []struct {
		Service    string `json:"service"`
		Environment string `json:"environment,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	} `json:"tailConsumers,omitempty"`
	UsageModel string `json:"usageModel,omitempty"`
}

type WorkerScriptOutputs struct {
	AccountId  string `json:"accountId"`
	ApiToken   string `json:"apiToken"`
	ScriptName string `json:"scriptName"`
}

func (r *WorkerScript) Create(input *WorkerScriptInputs, output *CreateResult[WorkerScriptOutputs]) error {
	err := r.handleUpdate(input)
	if err != nil {
		return err
	}

	*output = CreateResult[WorkerScriptOutputs]{
		ID:   "script",
		Outs: WorkerScriptOutputs{
			AccountId: input.AccountId,
			ApiToken: input.ApiToken,
			ScriptName: input.ScriptName,
		},
	}
	return nil
}

func (r *WorkerScript) Update(input *UpdateInput[WorkerScriptInputs, WorkerScriptOutputs], output *UpdateResult[WorkerScriptOutputs]) error {
	err := r.handleUpdate(&input.News)
	if err != nil {
		return err
	}

	*output = UpdateResult[WorkerScriptOutputs]{
		Outs: WorkerScriptOutputs{
			AccountId: input.News.AccountId,
			ApiToken: input.News.ApiToken,
			ScriptName: input.News.ScriptName,
		},
	}
	return nil
}

func (r *WorkerScript) Delete(input *DeleteInput[WorkerScriptOutputs], output *int) error {
	err := r.handleDelete(&input.Outs)
	if err != nil {
		return err
	}

	return nil
}

func (r *WorkerScript) handleUpdate(input *WorkerScriptInputs) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file content to form data
	fileContent, err := os.ReadFile(input.Content.Filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", input.Content.Filename, err)
	}

	contentType := "application/javascript"
	if input.MainModule != "" {
		input.MainModule = input.Content.Hash
		contentType = "application/javascript+module"
	}

	contentPart, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, input.Content.Hash, input.Content.Hash)},
		"Content-Type":        []string{contentType},
	})
	if err != nil {
		return fmt.Errorf("failed to create form part %s: %w", input.Content.Hash, err)
	}

	_, err = contentPart.Write([]byte(fileContent))
	if err != nil {
		return fmt.Errorf("failed to write file content for %s: %w", input.Content.Hash, err)
	}

	// Add metadata to form data
	metadataPart, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": []string{`form-data; name="metadata"`},
		"Content-Type":        []string{"application/json"},
	})
	if err != nil {
		return fmt.Errorf("failed to create form part metadata: %w", err)
	}

	metadata, err := json.Marshal(buildMetadata(input))
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = metadataPart.Write(metadata)
	if err != nil {
		return fmt.Errorf("failed to write file content for metadata: %w", err)
	}

	// Close writer
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/workers/scripts/%s", input.AccountId, input.ScriptName)

	req, err := http.NewRequest("PUT", url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+writer.Boundary())
	req.Header.Set("Authorization", "Bearer "+input.ApiToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// print out response body as a string
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to initialize assets upload: HTTP %d %s", resp.StatusCode, string(responseBody))
	}
	return nil
}

func buildMetadata(input *WorkerScriptInputs) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Assets
	if input.Assets.Jwt != "" || input.Assets.Config.Headers != "" || input.Assets.Config.Redirects != "" || input.Assets.Config.HtmlHandling != "" || input.Assets.Config.NotFoundHandling != "" || input.Assets.Config.RunWorkerFirst {
		assets := make(map[string]interface{})
		if input.Assets.Jwt != "" {
			assets["jwt"] = input.Assets.Jwt
		}
		config := make(map[string]interface{})
		if input.Assets.Config.Headers != "" {
			config["_headers"] = input.Assets.Config.Headers
		}
		if input.Assets.Config.Redirects != "" {
			config["_redirects"] = input.Assets.Config.Redirects
		}
		if input.Assets.Config.HtmlHandling != "" {
			config["html_handling"] = input.Assets.Config.HtmlHandling
		}
		if input.Assets.Config.NotFoundHandling != "" {
			config["not_found_handling"] = input.Assets.Config.NotFoundHandling
		}
		if input.Assets.Config.RunWorkerFirst {
			config["run_worker_first"] = input.Assets.Config.RunWorkerFirst
		}
		if len(config) > 0 {
			assets["config"] = config
		}
		if len(assets) > 0 {
			metadata["assets"] = assets
		}
	}

	// Convert bindings slice
	if len(input.Bindings) > 0 {
		bindings := make([]map[string]interface{}, 0, len(input.Bindings))
		for _, binding := range input.Bindings {
			bindingMap := make(map[string]interface{})
			bindingMap["type"] = binding.Type
			bindingMap["name"] = binding.Name
			if binding.BucketName != "" {
				bindingMap["bucket_name"] = binding.BucketName
			}
			if binding.ClassName != "" {
				bindingMap["class_name"] = binding.ClassName
			}
			if binding.NamespaceId != "" {
				bindingMap["namespace_id"] = binding.NamespaceId
			}
			if binding.QueueName != "" {
				bindingMap["queue_name"] = binding.QueueName
			}
			if binding.ScriptName != "" {
				bindingMap["script_name"] = binding.ScriptName
			}
			if binding.SecretName != "" {
				bindingMap["secret_name"] = binding.SecretName
			}
			if binding.Service != "" {
				bindingMap["service"] = binding.Service
			}
			if binding.Text != "" {
				bindingMap["text"] = binding.Text
			}
			bindings = append(bindings, bindingMap)
		}
		if len(bindings) > 0 {
			metadata["bindings"] = bindings
		}
	}

	// Simple string fields
	if input.BodyPart != "" {
		metadata["body_part"] = input.BodyPart
	}
	if input.CompatibilityDate != "" {
		metadata["compatibility_date"] = input.CompatibilityDate
	}
	if len(input.CompatibilityFlags) > 0 {
		metadata["compatibility_flags"] = input.CompatibilityFlags
	}
	if input.KeepAssets {
		metadata["keep_assets"] = input.KeepAssets
	}
	if len(input.KeepBindings) > 0 {
		metadata["keep_bindings"] = input.KeepBindings
	}
	if input.Logpush {
		metadata["logpush"] = input.Logpush
	}
	if input.MainModule != "" {
		metadata["main_module"] = input.MainModule
	}

	// Convert renamed classes slice
	renamedClasses := make([]map[string]interface{}, 0, len(input.Migrations.RenamedClasses))
	for _, renamedClass := range input.Migrations.RenamedClasses {
		renamedClassMap := make(map[string]interface{})
		if renamedClass.From != "" {
			renamedClassMap["from"] = renamedClass.From
		}
		if renamedClass.To != "" {
			renamedClassMap["to"] = renamedClass.To
		}
		if len(renamedClassMap) > 0 {
			renamedClasses = append(renamedClasses, renamedClassMap)
		}
	}

	// Convert steps slice
	steps := make([]map[string]interface{}, 0, len(input.Migrations.Steps))
	for _, step := range input.Migrations.Steps {
		stepMap := make(map[string]interface{})
		
		if len(step.DeletedClasses) > 0 {
			stepMap["deleted_classes"] = step.DeletedClasses
		}
		if len(step.NewClasses) > 0 {
			stepMap["new_classes"] = step.NewClasses
		}
		if len(step.NewSqliteClasses) > 0 {
			stepMap["new_sqlite_classes"] = step.NewSqliteClasses
		}

		stepRenamedClasses := make([]map[string]interface{}, 0, len(step.RenamedClasses))
		for _, renamedClass := range step.RenamedClasses {
			renamedClassMap := make(map[string]interface{})
			if renamedClass.From != "" {
				renamedClassMap["from"] = renamedClass.From
			}
			if renamedClass.To != "" {
				renamedClassMap["to"] = renamedClass.To
			}
			if len(renamedClassMap) > 0 {
				stepRenamedClasses = append(stepRenamedClasses, renamedClassMap)
			}
		}
		if len(stepRenamedClasses) > 0 {
			stepMap["renamed_classes"] = stepRenamedClasses
		}

		stepTransferredClasses := make([]map[string]interface{}, 0, len(step.TransferredClasses))
		for _, transferredClass := range step.TransferredClasses {
			transferredClassMap := make(map[string]interface{})
			if transferredClass.From != "" {
				transferredClassMap["from"] = transferredClass.From
			}
			if transferredClass.FromScript != "" {
				transferredClassMap["from_script"] = transferredClass.FromScript
			}
			if transferredClass.To != "" {
				transferredClassMap["to"] = transferredClass.To
			}
			if len(transferredClassMap) > 0 {
				stepTransferredClasses = append(stepTransferredClasses, transferredClassMap)
			}
		}
		if len(stepTransferredClasses) > 0 {
			stepMap["transferred_classes"] = stepTransferredClasses
		}

		if len(stepMap) > 0 {
			steps = append(steps, stepMap)
		}
	}

	// Convert transferred classes slice
	transferredClasses := make([]map[string]interface{}, 0, len(input.Migrations.TransferredClasses))
	for _, transferredClass := range input.Migrations.TransferredClasses {
		transferredClassMap := make(map[string]interface{})
		if transferredClass.From != "" {
			transferredClassMap["from"] = transferredClass.From
		}
		if transferredClass.FromScript != "" {
			transferredClassMap["from_script"] = transferredClass.FromScript
		}
		if transferredClass.To != "" {
			transferredClassMap["to"] = transferredClass.To
		}
		if len(transferredClassMap) > 0 {
			transferredClasses = append(transferredClasses, transferredClassMap)
		}
	}

	// Migrations
	if len(input.Migrations.DeletedClasses) > 0 || len(input.Migrations.NewClasses) > 0 || len(input.Migrations.NewSqliteClasses) > 0 || input.Migrations.NewTag != "" || input.Migrations.OldTag != "" || len(renamedClasses) > 0 || len(steps) > 0 || len(transferredClasses) > 0 {
		migrations := make(map[string]interface{})
		if len(input.Migrations.DeletedClasses) > 0 {
			migrations["deleted_classes"] = input.Migrations.DeletedClasses
		}
		if len(input.Migrations.NewClasses) > 0 {
			migrations["new_classes"] = input.Migrations.NewClasses
		}
		if len(input.Migrations.NewSqliteClasses) > 0 {
			migrations["new_sqlite_classes"] = input.Migrations.NewSqliteClasses
		}
		if input.Migrations.NewTag != "" {
			migrations["new_tag"] = input.Migrations.NewTag
		}
		if input.Migrations.OldTag != "" {
			migrations["old_tag"] = input.Migrations.OldTag
		}
		if len(renamedClasses) > 0 {
			migrations["renamed_classes"] = renamedClasses
		}
		if len(steps) > 0 {
			migrations["steps"] = steps
		}
		if len(transferredClasses) > 0 {
			migrations["transferred_classes"] = transferredClasses
		}
		if len(migrations) > 0 {
			metadata["migrations"] = migrations
		}
	}

	// Observability
	if input.Observability.Enabled || input.Observability.HeapSamplingRate > 0 {
		observability := make(map[string]interface{})
		if input.Observability.Enabled {
			observability["enabled"] = input.Observability.Enabled
		}
		if input.Observability.HeapSamplingRate > 0 {
			observability["heap_sampling_rate"] = input.Observability.HeapSamplingRate
		}
		metadata["observability"] = observability
	}

	// Placement
	if input.Placement.LastAnalyzedAt != "" || input.Placement.Mode != "" || input.Placement.Status != "" {
		placement := make(map[string]interface{})
		if input.Placement.LastAnalyzedAt != "" {
			placement["last_analyzed_at"] = input.Placement.LastAnalyzedAt
		}
		if input.Placement.Mode != "" {
			placement["mode"] = input.Placement.Mode
		}
		if input.Placement.Status != "" {
			placement["status"] = input.Placement.Status
		}
		metadata["placement"] = placement
	}

	// Convert tail consumers slice
	if len(input.TailConsumers) > 0 {
		tailConsumers := make([]map[string]interface{}, 0, len(input.TailConsumers))
		for _, tailConsumer := range input.TailConsumers {
			tailConsumerMap := make(map[string]interface{})
			tailConsumerMap["service"] = tailConsumer.Service
			if tailConsumer.Environment != "" {
				tailConsumerMap["environment"] = tailConsumer.Environment
			}
			if tailConsumer.Namespace != "" {
				tailConsumerMap["namespace"] = tailConsumer.Namespace
			}
			tailConsumers = append(tailConsumers, tailConsumerMap)
		}
		if len(tailConsumers) > 0 {
			metadata["tail_consumers"] = tailConsumers
		}
	}

	// Usage model
	if input.UsageModel != "" {
		metadata["usage_model"] = input.UsageModel
	}

	return metadata
}

func (r *WorkerScript) handleDelete(input *WorkerScriptOutputs) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/workers/scripts/%s", input.AccountId, input.ScriptName)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+input.ApiToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete script: HTTP %d %s", resp.StatusCode, string(responseBody))
	}
	return nil
}
