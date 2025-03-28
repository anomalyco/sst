package resource

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore/types"
)

type KvRoutesUpdate struct {
	*AwsResource
}

type KvRoutesUpdateInputs struct {
	Store     string `json:"store"`
	Key       string `json:"key"`
	Entry     string `json:"entry"`
	Namespace string `json:"namespace"`
}

type KvRoutesUpdateOutputs struct {
	Store     string   `json:"store,omitempty"`
	Key       string   `json:"key,omitempty"`
	Entry     string   `json:"entry,omitempty"`
	Namespace string   `json:"namespace,omitempty"`
}

func (r *KvRoutesUpdate) Create(input *KvRoutesUpdateInputs, output *CreateResult[KvRoutesUpdateOutputs]) error {
	// get client
	cfg, err := r.config()
	if err != nil {
		return err
	}
	client := cloudfrontkeyvaluestore.NewFromConfig(cfg)

	// Get all routes
	fullKey := input.Namespace + ":" + input.Key
	existingRoutes, etag, err := r.getRoutesWithEtag(client, input.Store, fullKey)
	if err != nil {
		return err
	}
	
	// Append route if not exists
	if !existsRoute(existingRoutes, input.Entry) {
		routes := append(existingRoutes, input.Entry)
		err = r.setRoutes(client, input.Store, etag, fullKey, routes)
		// retry if etag pre-condition failed
		if err != nil {
			var preconditionFailedErr *types.ValidationException
			if errors.As(err, &preconditionFailedErr) && preconditionFailedErr.Message != nil && strings.Contains(*preconditionFailedErr.Message, "Pre-Condition failed") {
				// sleep for a random time between 100 and 500ms
				time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)
				return r.Create(input, output)
			}
			return err
		}
	}
	
	*output = CreateResult[KvRoutesUpdateOutputs]{
		ID: fmt.Sprintf("%s:%s:%s", input.Store, input.Namespace, input.Key),
		Outs: KvRoutesUpdateOutputs{
			Store:     input.Store,
			Key:       input.Key,
			Entry:     input.Entry,
			Namespace: input.Namespace,
		},
	}
	return nil
}

func (r *KvRoutesUpdate) Update(input *UpdateInput[KvRoutesUpdateInputs, KvRoutesUpdateOutputs], output *UpdateResult[KvRoutesUpdateOutputs]) error {
	// Handle store, namespace, or key has changed, handle as a new create
	if input.News.Store != input.Olds.Store ||
		input.News.Namespace != input.Olds.Namespace ||
		input.News.Key != input.Olds.Key {
		result := CreateResult[KvRoutesUpdateOutputs]{}
		
		// First, delete the old entry if it exists
		deleteInput := DeleteInput[KvRoutesUpdateOutputs]{
			ID: input.ID,
			Outs: KvRoutesUpdateOutputs{
				Store:     input.Olds.Store,
				Key:       input.Olds.Key,
				Entry:     input.Olds.Entry,
				Namespace: input.Olds.Namespace,
			},
		}
		var dummy int
		if err := r.Delete(&deleteInput, &dummy); err != nil {
			return err
		}
		
		// Then create the new entry
		if err := r.Create(&input.News, &result); err != nil {
			return err
		}
		
		*output = UpdateResult[KvRoutesUpdateOutputs]{
			Outs: result.Outs,
		}
		return nil
	}
	
	// get client
	cfg, err := r.config()
	if err != nil {
		return err
	}
	client := cloudfrontkeyvaluestore.NewFromConfig(cfg)

	// Get all routes
	fullKey := input.News.Namespace + ":" + input.News.Key
	existingEntries, etag, err := r.getRoutesWithEtag(client, input.News.Store, fullKey)
	if err != nil {
		return err
	}
	
	// Remove the old entry and add new
	entries := removeRoute(existingEntries, input.Olds.Entry)
	if !existsRoute(entries, input.News.Entry) {
		entries = append(entries, input.News.Entry)
	}
	
	// Save routes
	err = r.setRoutes(client, input.News.Store, etag, fullKey, entries)
	if err != nil {
		var preconditionFailedErr *types.ValidationException
		if errors.As(err, &preconditionFailedErr) && preconditionFailedErr.Message != nil && strings.Contains(*preconditionFailedErr.Message, "Pre-Condition failed") {
			// sleep for a random time between 100 and 500ms
			time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)
			return r.Update(input, output)
		}
		return err
	}
	
	*output = UpdateResult[KvRoutesUpdateOutputs]{
		Outs: KvRoutesUpdateOutputs{
			Store:     input.News.Store,
			Key:       input.News.Key,
			Entry:     input.News.Entry,
			Namespace: input.News.Namespace,
		},
	}
	return nil
}

func (r *KvRoutesUpdate) Delete(input *DeleteInput[KvRoutesUpdateOutputs], output *int) error {
	// get client
	cfg, err := r.config()
	if err != nil {
		return err
	}
	client := cloudfrontkeyvaluestore.NewFromConfig(cfg)

	// Get all routes
	fullKey := input.Outs.Namespace + ":" + input.Outs.Key
	existingRoutes, etag, err := r.getRoutesWithEtag(client, input.Outs.Store, fullKey)
	if err != nil {
		return err
	}

	// Remove route
	routes := removeRoute(existingRoutes, input.Outs.Entry)

	// Route not found
	if len(routes) == len(existingRoutes) {
		return nil
	}

	if len(routes) == 0 {
		err = r.deleteKey(client, input.Outs.Store, etag, fullKey)
	} else {
		err = r.setRoutes(client, input.Outs.Store, etag, fullKey, routes)
	}
	if err != nil {
		var preconditionFailedErr *types.ValidationException
		if errors.As(err, &preconditionFailedErr) && preconditionFailedErr.Message != nil && strings.Contains(*preconditionFailedErr.Message, "Pre-Condition failed") {
			// sleep for a random time between 100 and 500ms
			time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)
			return r.Delete(input, output)
		}
		return err
	}

	return nil
}

func (r *KvRoutesUpdate) getRoutesWithEtag(client *cloudfrontkeyvaluestore.Client, storeARN, key string) ([]string, string, error) {
	// get etag
	descResp, err := client.DescribeKeyValueStore(r.context, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: aws.String(storeARN),
	})
	if err != nil {
		return nil, "", err
	}
	etag := *descResp.ETag

	// get routes
	getResp, err := client.GetKey(r.context, &cloudfrontkeyvaluestore.GetKeyInput{
		KvsARN: aws.String(storeARN),
		Key:    aws.String(key),
	})
	
	if err != nil {
		var notFoundErr *types.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			// Key not found, return empty entries
			return []string{}, etag, nil
		}
		// other error, propagate
		return nil, etag, err
	}
	
	if getResp.Value == nil {
		// no value
		return []string{}, etag, nil
	}
	
	// Parse routes array
	var entries []string
	if err := json.Unmarshal([]byte(*getResp.Value), &entries); err != nil {
		return nil, etag, fmt.Errorf("failed to unmarshal existing entries: %w", err)
	}
	
	return entries, etag, nil
}

func (r *KvRoutesUpdate) setRoutes(client *cloudfrontkeyvaluestore.Client, storeARN string, etag string, key string, entries []string) error {
	entriesJSON, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}
	entriesStr := string(entriesJSON)

	// update
	_, err = client.UpdateKeys(r.context, &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  aws.String(storeARN),
		IfMatch: aws.String(etag),
		Puts: []types.PutKeyRequestListItem{
			{
				Key:   aws.String(key),
				Value: aws.String(entriesStr),
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *KvRoutesUpdate) deleteKey(client *cloudfrontkeyvaluestore.Client, storeARN, etag string, key string) error {
	// delete
	_, err := client.UpdateKeys(r.context, &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  aws.String(storeARN),
		IfMatch: aws.String(etag),
		Deletes: []types.DeleteKeyRequestListItem{
			{
				Key: aws.String(key),
			},
		},
	})

	// retry if etag pre-condition failed
	if err != nil {
		return err
	}

	return nil
}

func existsRoute(entries []string, entry string) bool {
	for _, e := range entries {
		if e == entry {
			return true
		}
	}
	return false
}

func removeRoute(entries []string, entryToRemove string) []string {
	var result []string
	for _, v := range entries {
		if v != entryToRemove {
			result = append(result, v)
		}
	}
	return result
}
