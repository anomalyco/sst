package project

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/sst/sst/v3/pkg/project/provider"
)

func (p *Project) EnvFor(ctx context.Context, complete *CompleteEvent, name string) (map[string]string, error) {
	log := slog.Default().With("service", "project.env").With("resource", name)
	dev := complete.Devs[name]
	env := map[string]string{}
	if dev.Aws != nil && dev.Aws.Role != "" {
		log.Info("loading aws credentials", "role", dev.Aws.Role)
		prov, _ := p.Provider("aws")
		awsProvider := prov.(*provider.AwsProvider)
		stsClient := sts.NewFromConfig(awsProvider.Config())
		sessionName := "sst-dev"
		result, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         &dev.Aws.Role,
			RoleSessionName: &sessionName,
			DurationSeconds: awssdk.Int32(3600),
		})
		if err == nil {
			env["AWS_ACCESS_KEY_ID"] = *result.Credentials.AccessKeyId
			env["AWS_SECRET_ACCESS_KEY"] = *result.Credentials.SecretAccessKey
			env["AWS_SESSION_TOKEN"] = *result.Credentials.SessionToken
			env["AWS_REGION"] = awsProvider.Config().Region
		}

		if err != nil {
			log.Error("failed to load aws credentials", "err", err)
		}
	}
	log.Info("dev", "links", dev.Links)
	for _, resource := range dev.Links {
		value := complete.Links[resource].Properties
		jsonValue, _ := json.Marshal(value)
		env["SST_RESOURCE_"+resource] = string(jsonValue)
	}
	env["SST_RESOURCE_App"] = fmt.Sprintf(`{"name": "%s", "stage": "%s" }`, p.App().Name, p.App().Stage)
	for key, value := range dev.Environment {
		env[key] = value
	}
	if wranglerPath, err := p.generateWranglerFile(complete, dev); err != nil {
		log.Error("failed to write cloudflare wrangler file", "err", err)
	} else if wranglerPath != "" {
		env["SST_WRANGLER_PATH"] = wranglerPath
	}
	// Pass Cloudflare credentials to the child process for remote bindings
	for _, key := range []string{"CLOUDFLARE_API_TOKEN", "CLOUDFLARE_API_KEY", "CLOUDFLARE_EMAIL", "CLOUDFLARE_DEFAULT_ACCOUNT_ID"} {
		if val := os.Getenv(key); val != "" {
			env[key] = val
		}
	}
	return env, nil
}

// Generate hidden wrangler config for bindings to work during `sst dev`
func (p *Project) generateWranglerFile(complete *CompleteEvent, dev Dev) (string, error) {
	prov, ok := p.Provider("cloudflare")
	if !ok {
		return "", nil
	}

	vars := map[string]interface{}{}
	for key, value := range dev.Environment {
		vars[key] = value
	}

	kvNamespaces := []map[string]interface{}{}
	r2Buckets := []map[string]interface{}{}
	d1Databases := []map[string]interface{}{}
	hyperdrives := []map[string]interface{}{}
	services := []map[string]interface{}{}
	queueProducers := []map[string]interface{}{}
	var ai map[string]interface{}
	var versionMetadata map[string]interface{}
	hasBindings := false

	for _, resource := range dev.Links {
		link, ok := complete.Links[resource]
		if !ok {
			continue
		}

		for _, include := range link.Include {
			if include.Type != "cloudflare.binding" {
				continue
			}

			binding, _ := include.Other["binding"].(string)
			properties, _ := include.Other["properties"].(map[string]interface{})
			hasBindings = true

			switch binding {
			case "aiBindings":
				ai = map[string]interface{}{
					"binding": resource,
					"remote":  true,
				}
			case "kvNamespaceBindings":
				kvNamespaces = append(kvNamespaces, map[string]interface{}{
					"binding": resource,
					"id":      stringValue(properties["namespaceId"]),
					"remote":  true,
				})
			case "secretTextBindings", "plainTextBindings":
				vars[resource] = stringValue(properties["text"])
			case "serviceBindings":
				services = append(services, map[string]interface{}{
					"binding": resource,
					"service": stringValue(properties["service"]),
					"remote":  true,
				})
			case "queueBindings":
				queueProducers = append(queueProducers, map[string]interface{}{
					"binding": resource,
					"queue":   stringValue(properties["queueName"]),
					"remote":  true,
				})
			case "r2BucketBindings":
				r2Buckets = append(r2Buckets, map[string]interface{}{
					"binding":     resource,
					"bucket_name": stringValue(properties["bucketName"]),
					"remote":      true,
				})
			case "d1DatabaseBindings":
				d1Databases = append(d1Databases, map[string]interface{}{
					"binding":     resource,
					"database_id": stringValue(properties["id"]),
					"remote":      true,
				})
			case "hyperdriveBindings":
				hyperdrives = append(hyperdrives, map[string]interface{}{
					"binding": resource,
					"id":      stringValue(properties["id"]),
				})
			case "versionMetadataBindings":
				versionMetadata = map[string]interface{}{
					"binding": resource,
				}
			}
		}
	}

	if !hasBindings && len(vars) == 0 {
		return "", nil
	}

	config := map[string]interface{}{
		"name": sanitizeWranglerName("sst-dev-" + dev.Name),
	}
	if dev.Cloudflare != nil && dev.Cloudflare.Compatibility != nil {
		if dev.Cloudflare.Compatibility.Date != nil {
			config["compatibility_date"] = *dev.Cloudflare.Compatibility.Date
		}
		if dev.Cloudflare.Compatibility.Flags != nil {
			config["compatibility_flags"] = dev.Cloudflare.Compatibility.Flags
		}
	}

	providerEnv, err := prov.Env()
	if err != nil {
		return "", err
	}
	if accountID := providerEnv["CLOUDFLARE_DEFAULT_ACCOUNT_ID"]; accountID != "" {
		config["account_id"] = accountID
	}
	if len(vars) > 0 {
		config["vars"] = vars
	}
	if len(kvNamespaces) > 0 {
		config["kv_namespaces"] = kvNamespaces
	}
	if len(r2Buckets) > 0 {
		config["r2_buckets"] = r2Buckets
	}
	if len(d1Databases) > 0 {
		config["d1_databases"] = d1Databases
	}
	if len(hyperdrives) > 0 {
		config["hyperdrive"] = hyperdrives
	}
	if len(services) > 0 {
		config["services"] = services
	}
	if len(queueProducers) > 0 {
		config["queues"] = map[string]interface{}{
			"producers": queueProducers,
		}
	}
	if ai != nil {
		config["ai"] = ai
	}
	if versionMetadata != nil {
		config["version_metadata"] = versionMetadata
	}

	wranglerDir := filepath.Join(p.PathWorkingDir(), "wrangler", p.App().Stage)
	os.MkdirAll(wranglerDir, 0o755)
	configPath := filepath.Join(wranglerDir, dev.Name+".jsonc")

	contents, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	if existing, err := os.ReadFile(configPath); err == nil && bytes.Equal(existing, contents) {
		return configPath, nil
	}
	if err := os.WriteFile(configPath, contents, 0o644); err != nil {
		return "", err
	}

	return configPath, nil
}

func stringValue(input interface{}) string {
	if value, ok := input.(string); ok {
		return value
	}
	return ""
}

var wranglerNameRegex = regexp.MustCompile(`[^a-z0-9-]+`)

func sanitizeWranglerName(input string) string {
	value := strings.ToLower(input)
	value = wranglerNameRegex.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "sst-dev"
	}
	return value
}
