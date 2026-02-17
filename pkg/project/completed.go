package project

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/sst/sst/v3/pkg/id"
	"github.com/sst/sst/v3/pkg/project/common"
	"github.com/sst/sst/v3/pkg/project/provider"
	"github.com/sst/sst/v3/pkg/state"
)

func (p *Project) GetCompleted(ctx context.Context) (*CompleteEvent, error) {
	passphrase, err := provider.Passphrase(p.home, p.app.Name, p.app.Stage)
	if err != nil {
		return nil, err
	}
	workdir, err := p.NewWorkdir(id.Descending())
	if err != nil {
		return nil, err
	}
	defer workdir.Cleanup()
	_, err = workdir.Pull()
	if err != nil {
		return nil, err
	}
	return getCompletedEvent(ctx, passphrase, workdir)
}

func getCompletedEvent(ctx context.Context, passphrase string, workdir *PulumiWorkdir) (*CompleteEvent, error) {
	complete := &CompleteEvent{
		Links:       common.Links{},
		Versions:    map[string]int{},
		ImportDiffs: map[string][]ImportDiff{},
		Devs:        Devs{},
		Tunnels:     map[string]Tunnel{},
		Hints:       map[string]string{},
		Outputs:     map[string]interface{}{},
		Tasks:       map[string]Task{},
		Errors:      []Error{},
		Finished:    false,
		Resources:   []apitype.ResourceV3{},
	}
	checkpoint, err := workdir.Export()
	if err != nil {
		return nil, err
	}
	decrypted, err := state.Decrypt(ctx, passphrase, checkpoint)
	if err != nil {
		return nil, err
	}
	deployment := decrypted.Latest
	if len(deployment.Resources) == 0 {
		return complete, nil
	}
	complete.Resources = deployment.Resources

	for _, resource := range complete.Resources {
		outputs, ok := parsePlaintext(resource.Outputs).(map[string]interface{})
		if !ok {
			continue
		}
		if resource.URN.Type().Module().Package().Name() == "sst" {
			if resource.Type == "sst:sst:Version" {
				target, targetOk := outputs["target"].(string)
				version, versionOk := outputs["version"].(float64)
				if targetOk && versionOk {
					complete.Versions[target] = int(version)
				}
			}

			if resource.Type != "sst:sst:Version" {
				name := resource.URN.Name()
				_, ok := complete.Versions[name]
				if !ok {
					complete.Versions[name] = 1
				}
			}
		}
		if match, ok := outputs["_dev"].(map[string]interface{}); ok {
			data, _ := json.Marshal(match)
			var entry Dev
			json.Unmarshal(data, &entry)
			entry.Name = resource.URN.Name()
			complete.Devs[entry.Name] = entry
		}

		if match, ok := outputs["_task"].(map[string]interface{}); ok {
			data, _ := json.Marshal(match)
			var entry Task
			json.Unmarshal(data, &entry)
			entry.Name = resource.URN.Name()
			complete.Tasks[entry.Name] = entry
		}

		if match, ok := outputs["_tunnel"].(map[string]interface{}); ok {
			ip, ipOk := match["ip"].(string)
			username, usernameOk := match["username"].(string)
			privateKey, privateKeyOk := match["privateKey"].(string)
			if !ipOk || !usernameOk || !privateKeyOk {
				continue
			}
			tunnel := Tunnel{
				IP:         ip,
				Username:   username,
				PrivateKey: privateKey,
				Subnets:    []string{},
			}
			if subnets, ok := match["subnets"].([]interface{}); ok {
				for _, subnet := range subnets {
					if s, ok := subnet.(string); ok {
						tunnel.Subnets = append(tunnel.Subnets, s)
					}
				}
				complete.Tunnels[resource.URN.Name()] = tunnel
			}
		}

		if hint, ok := outputs["_hint"].(string); ok {
			complete.Hints[string(resource.URN)] = hint
		}

		if resource.Type == "sst:sst:LinkRef" {
			target, targetOk := outputs["target"].(string)
			properties, propertiesOk := outputs["properties"].(map[string]interface{})
			if !targetOk || !propertiesOk {
				continue
			}
			link := common.Link{
				Properties: properties,
				Include:    []common.LinkInclude{},
			}
			if includeSlice, ok := outputs["include"].([]interface{}); ok {
				for _, inc := range includeSlice {
					incMap, ok := inc.(map[string]interface{})
					if !ok {
						continue
					}
					incType, ok := incMap["type"].(string)
					if !ok {
						continue
					}
					link.Include = append(link.Include, common.LinkInclude{
						Type:  incType,
						Other: incMap,
					})
				}
			}
			complete.Links[target] = link
		}
	}

	if outputs, ok := parsePlaintext(deployment.Resources[0].Outputs).(map[string]interface{}); ok {
		for key, value := range outputs {
			if strings.HasPrefix(key, "_") {
				continue
			}
			complete.Outputs[key] = value
		}
	}

	return complete, nil
}

func parsePlaintext(input interface{}) interface{} {
	switch cast := input.(type) {
	case apitype.SecretV1:
		var parsed any
		json.Unmarshal([]byte(cast.Plaintext), &parsed)
		return parsed
	case *apitype.SecretV1:
		var parsed any
		json.Unmarshal([]byte(cast.Plaintext), &parsed)
		return parsed
	case map[string]interface{}:
		for key, value := range cast {
			cast[key] = parsePlaintext(value)
		}
		return cast
	case []interface{}:
		for i, value := range cast {
			cast[i] = parsePlaintext(value)
		}
		return cast
	default:
		return cast
	}
}
