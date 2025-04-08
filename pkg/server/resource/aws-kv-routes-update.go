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

const (
	chunkSize = 1000
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

	// get etag
	etag, err := r.getEtag(client, input.Store)
	if err != nil {
		return err
	}

	// get routes
	fullKey := input.Namespace + ":" + input.Key
	existingRoutes, chunkNum, err := r.getRoutes(client, input.Store, fullKey)
	if err != nil {
		// check etag to see if this happened b/c routes were updated in the meantime
		newEtag, err := r.getEtag(client, input.Store)
		if err == nil && newEtag != etag {
			// sleep for a random time between 100 and 500ms
			time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)
			return r.Create(input, output)
		}
		return err
	}
	
	// append route if not exists
	routes := existingRoutes
	if !existsRoute(routes, input.Entry) {
		routes = append(routes, input.Entry)
	}

	// set routes
	err = r.setRoutes(client, input.Store, etag, fullKey, routes, chunkNum)
	if err != nil {
		var preconditionFailedErr *types.ValidationException
		if errors.As(err, &preconditionFailedErr) && preconditionFailedErr.Message != nil && strings.Contains(*preconditionFailedErr.Message, "Pre-Condition failed") {
			// sleep for a random time between 100 and 500ms
			time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)
			return r.Create(input, output)
		}
		return err
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

	// get etag
	etag, err := r.getEtag(client, input.News.Store)
	if err != nil {
		return err
	}

	// get routes
	fullKey := input.News.Namespace + ":" + input.News.Key
	existingEntries, chunkNum, err := r.getRoutes(client, input.News.Store, fullKey)
	if err != nil {
		// check etag to see if this happened b/c routes were updated in the meantime
		newEtag, err := r.getEtag(client, input.News.Store)
		if err == nil && newEtag != etag {
			// sleep for a random time between 100 and 500ms
			time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)
			return r.Update(input, output)
		}
		return err
	}
	
	// Remove the old entry and add new
	entries := removeRoute(existingEntries, input.Olds.Entry)
	if !existsRoute(entries, input.News.Entry) {
		entries = append(entries, input.News.Entry)
	}
	
	// Save routes
	err = r.setRoutes(client, input.News.Store, etag, fullKey, entries, chunkNum)
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

	// get etag
	etag, err := r.getEtag(client, input.Outs.Store)
	if err != nil {
		return err
	}

	// get routes
	fullKey := input.Outs.Namespace + ":" + input.Outs.Key
	existingRoutes, chunkNum, err := r.getRoutes(client, input.Outs.Store, fullKey)
	if err != nil {
		// check etag to see if this happened b/c routes were updated in the meantime
		newEtag, err := r.getEtag(client, input.Outs.Store)
		if err == nil && newEtag != etag {
			// sleep for a random time between 100 and 500ms
			time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)
			return r.Delete(input, output)
		}
		return err
	}

	// Remove route
	routes := removeRoute(existingRoutes, input.Outs.Entry)

	// Route not found - DO NOT do optimizations like this b/c existingRoutes might not
	// contain the latest data. Etag might have changed from when etag was first retrieved.
	//if len(routes) == len(existingRoutes) {
	//	return nil
	//}

	if len(routes) == 0 {
		err = r.deleteKey(client, input.Outs.Store, etag, fullKey, chunkNum)
	} else {
		err = r.setRoutes(client, input.Outs.Store, etag, fullKey, routes, chunkNum)
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

func (r *KvRoutesUpdate) getEtag(client *cloudfrontkeyvaluestore.Client, storeARN string) (string, error) {
	descResp, err := client.DescribeKeyValueStore(r.context, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: aws.String(storeARN),
	})
	if err != nil {
		return "", err
	}
	return *descResp.ETag, nil
}

func (r *KvRoutesUpdate) getRoutes(client *cloudfrontkeyvaluestore.Client, storeARN, key string) ([]string, int, error) {
	chunkNum := 1

	getResp, err := client.GetKey(r.context, &cloudfrontkeyvaluestore.GetKeyInput{
		KvsARN: aws.String(storeARN),
		Key:    aws.String(key),
	})
	
	if err != nil {
		var notFoundErr *types.ResourceNotFoundException
		if errors.As(err, &notFoundErr) {
			// Key not found, return empty entries
			return []string{}, chunkNum, nil
		}
		// other error, propagate
		return nil, chunkNum, err
	}
	
	if getResp.Value == nil {
		// no value
		return []string{}, chunkNum, nil
	}
	
	entriesData := *getResp.Value

	// Check if the data is chunked by trying to parse a metadata object first
	var metadata map[string]int
	if err := json.Unmarshal([]byte(*getResp.Value), &metadata); err == nil {
		if parts, ok := metadata["parts"]; ok && parts > 0 {
			chunkNum = parts

			// This is chunked data, we need to retrieve and concatenate all chunks
			entriesData = ""
			
			// Retrieve all chunks
			for i := 0; i < parts; i++ {
				chunkResp, err := client.GetKey(r.context, &cloudfrontkeyvaluestore.GetKeyInput{
					KvsARN: aws.String(storeARN),
					Key:    aws.String(fmt.Sprintf("%s:%d", key, i)),
				})
				
				if err != nil {
					return nil, chunkNum, fmt.Errorf("failed to retrieve chunk %d: %w", i, err)
				}
				
				if chunkResp.Value == nil {
					return nil, chunkNum, fmt.Errorf("chunk %d value is missing", i)
				}
				
				entriesData += *chunkResp.Value
			}
		}
	}
	
	// Parse routes array
	var entries []string
	if err := json.Unmarshal([]byte(entriesData), &entries); err != nil {
		return nil, chunkNum, fmt.Errorf("failed to unmarshal existing entries: %w", err)
	}
	
	return entries, chunkNum, nil
}

func (r *KvRoutesUpdate) setRoutes(client *cloudfrontkeyvaluestore.Client, storeARN string, etag string, key string, entries []string, oldChunkNum int) error {
	newChunkNum := 1
	puts := []types.PutKeyRequestListItem{}
	deletes := []types.DeleteKeyRequestListItem{}

	// Build new entries
	entriesJSON, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}
	entriesStr := string(entriesJSON)

	// Check if the string is longer than chunkSize
	if len(entriesStr) > chunkSize {
		// Calculate number of chunks needed
		newChunkNum= (len(entriesStr) + chunkSize - 1) / chunkSize // Ceiling division
		
		// Create a metadata entry to store the number of parts
		metadataMap := map[string]int{"parts": newChunkNum}
		metadataJSON, err := json.Marshal(metadataMap)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		
		// Create multiple puts for each chunk
		puts = append(puts, types.PutKeyRequestListItem{
			Key:   aws.String(key),
			Value: aws.String(string(metadataJSON)),
		})
		
		// Split the string into chunks
		for i := 0; i < newChunkNum; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if end > len(entriesStr) {
				end = len(entriesStr)
			}
			
			puts = append(puts, types.PutKeyRequestListItem{
				Key:   aws.String(fmt.Sprintf("%s:%d", key, i)),
				Value: aws.String(entriesStr[start:end]),
			})
		}
	} else {
		// For smaller strings, use the original approach
		puts = append(puts, types.PutKeyRequestListItem{
			Key:   aws.String(key),
			Value: aws.String(entriesStr),
		})
	}

	// Delete extra chunks if previous chunk # is greater than new chunk #
	if oldChunkNum > newChunkNum {
		for i := newChunkNum; i < oldChunkNum; i++ {
			deletes = append(deletes, types.DeleteKeyRequestListItem{
				Key: aws.String(fmt.Sprintf("%s:%d", key, i)),
			})
		}

		if newChunkNum == 1 {
			deletes = append(deletes, types.DeleteKeyRequestListItem{
				Key: aws.String(fmt.Sprintf("%s:0", key)),
			})
		}
	}

	// update
	_, err = client.UpdateKeys(r.context, &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  aws.String(storeARN),
		IfMatch: aws.String(etag),
		Puts:    puts,
		Deletes: deletes,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *KvRoutesUpdate) deleteKey(client *cloudfrontkeyvaluestore.Client, storeARN, etag string, key string, oldChunkNum int) error {
	deletes := []types.DeleteKeyRequestListItem{}
	deletes = append(deletes, types.DeleteKeyRequestListItem{
		Key: aws.String(key),
	})
	
	// Add all chunk keys to delete
	if oldChunkNum > 1 {
		for i := 0; i < oldChunkNum; i++ {
			deletes = append(deletes, types.DeleteKeyRequestListItem{
				Key: aws.String(fmt.Sprintf("%s:%d", key, i)),
			})
		}
	}
	
	// Not chunked or key not found, proceed with normal delete
	_, err := client.UpdateKeys(r.context, &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  aws.String(storeARN),
		IfMatch: aws.String(etag),
		Deletes: deletes,
	})
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