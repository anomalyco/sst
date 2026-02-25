package resource

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore/types"
)

type KvKeys struct {
	*AwsResource
}

type KvKeysInputs struct {
	Store     string            `json:"store"`
	Entries   map[string]string `json:"entries"`
	Purge     bool              `json:"purge"`
	Namespace string            `json:"namespace"`
}

type KvKeysOutputs struct {
	Store     string            `json:"store,omitempty"`
	Entries   map[string]string `json:"entries,omitempty"`
	Purge     bool              `json:"purge,omitempty"`
	Namespace string            `json:"namespace,omitempty"`
}

func (r *KvKeys) Create(input *KvKeysInputs, output *CreateResult[KvKeysOutputs]) error {
	cfg, err := r.config()
	if err != nil {
		return err
	}
	client := cloudfrontkeyvaluestore.NewFromConfig(cfg)

	if err := r.upload(client, input.Store, input.Namespace, input.Entries, nil); err != nil {
		return err
	}

	*output = CreateResult[KvKeysOutputs]{
		ID: "entries",
		Outs: KvKeysOutputs{
			Store:     input.Store,
			Entries:   input.Entries,
			Purge:     input.Purge,
			Namespace: input.Namespace,
		},
	}
	return nil
}

func (r *KvKeys) Update(input *UpdateInput[KvKeysInputs, KvKeysOutputs], output *UpdateResult[KvKeysOutputs]) error {
	cfg, err := r.config()
	if err != nil {
		return err
	}
	client := cloudfrontkeyvaluestore.NewFromConfig(cfg)

	oldEntries := input.Olds.Entries
	if input.News.Store != input.Olds.Store {
		oldEntries = nil
	}

	if err := r.upload(client, input.News.Store, input.News.Namespace, input.News.Entries, oldEntries); err != nil {
		return err
	}

	if input.News.Purge {
		if err := r.purge(client, input.News.Store, input.News.Namespace, input.News.Entries); err != nil {
			return err
		}
	}

	*output = UpdateResult[KvKeysOutputs]{
		Outs: KvKeysOutputs{
			Store:     input.News.Store,
			Entries:   input.News.Entries,
			Purge:     input.News.Purge,
			Namespace: input.News.Namespace,
		},
	}
	return nil
}

func (r *KvKeys) Delete(input *DeleteInput[KvKeysOutputs], output *int) error {
	if input.Outs.Store == "" {
		return nil
	}

	cfg, err := r.config()
	if err != nil {
		return err
	}
	client := cloudfrontkeyvaluestore.NewFromConfig(cfg)

	return r.purge(client, input.Outs.Store, input.Outs.Namespace, nil)
}

func (r *KvKeys) upload(client *cloudfrontkeyvaluestore.Client, store string, namespace string, entries map[string]string, oldEntries map[string]string) error {
	if len(entries) == 0 {
		return nil
	}

	// Identify keys that need to be updated (new keys or changed values)
	var keysToUpdate []struct {
		key   string
		value string
	}
	for k, v := range entries {
		oldValue, exists := oldEntries[k]
		if !exists || oldValue != v {
			keysToUpdate = append(keysToUpdate, struct {
				key   string
				value string
			}{k, v})
		}
	}

	// If no keys need updating, return early
	if len(keysToUpdate) == 0 {
		return nil
	}

	// Prepare put items
	putItems := make([]types.PutKeyRequestListItem, 0, len(keysToUpdate))
	for _, kv := range keysToUpdate {
		putItems = append(putItems, types.PutKeyRequestListItem{
			Key:   aws.String(namespace + ":" + kv.key),
			Value: aws.String(kv.value),
		})
	}

	// Update keys with retry support (passing nil for etag to fetch a new one)
	return r.updateKeys(client, store, nil, putItems, nil)
}

func (r *KvKeys) purge(client *cloudfrontkeyvaluestore.Client, store string, namespace string, entries map[string]string) error {
	// Collect all keys under the namespace that are not in entries
	var keysToDelete []string
	var nextToken *string = nil

	// Use pagination to get all keys
	for {
		listResp, err := client.ListKeys(r.context, &cloudfrontkeyvaluestore.ListKeysInput{
			KvsARN:    aws.String(store),
			NextToken: nextToken,
		})
		if err != nil {
			var notFoundErr *types.ResourceNotFoundException
			if errors.As(err, &notFoundErr) {
				return nil
			}
			return err
		}

		// Process keys from this page
		for _, item := range listResp.Items {
			// Check if the key has the namespace prefix
			if strings.HasPrefix(*item.Key, namespace+":") {
				// Extract the key without namespace prefix
				keyWithoutNamespace := strings.TrimPrefix(*item.Key, namespace+":")
				
				// Skip keys that exist in entries
				if entries != nil {
					if _, exists := entries[keyWithoutNamespace]; exists {
						continue
					}
				}
				
				keysToDelete = append(keysToDelete, *item.Key)
			}
		}

		// Check if there are more keys to retrieve
		if listResp.NextToken == nil {
			break
		}
		nextToken = listResp.NextToken
	}

	if len(keysToDelete) == 0 {
		return nil
	}

	// Prepare delete items
	deleteItems := make([]types.DeleteKeyRequestListItem, 0, len(keysToDelete))
	for _, key := range keysToDelete {
		deleteItems = append(deleteItems, types.DeleteKeyRequestListItem{
			Key: aws.String(key),
		})
	}

	// Update keys with retry support (passing nil for etag to fetch a new one)
	return r.updateKeys(client, store, nil, nil, deleteItems)
}

func (r *KvKeys) updateKeys(client *cloudfrontkeyvaluestore.Client, store string, etag *string, putItems []types.PutKeyRequestListItem, deleteItems []types.DeleteKeyRequestListItem) error {
	// If etag is nil, fetch a new one
	if etag == nil {
		descResp, err := client.DescribeKeyValueStore(r.context, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
			KvsARN: aws.String(store),
		})
		if err != nil {
			return err
		}
		etag = descResp.ETag
	}

	// Process in batches of 50
	const batchSize = 50
	
	// Determine if we have items to process in this call
	hasPuts := len(putItems) > 0
	hasDeletes := len(deleteItems) > 0
	
	// If there are no items to process, we're done
	if !hasPuts && !hasDeletes {
		return nil
	}
	
	// Get the puts and deletes for this batch
	var putBatch []types.PutKeyRequestListItem
	var deleteBatch []types.DeleteKeyRequestListItem
	var remainingPuts []types.PutKeyRequestListItem
	var remainingDeletes []types.DeleteKeyRequestListItem
	
	// Calculate how many items we can include in this batch
	availableBatchSize := batchSize
	
	// Process puts up to batch size
	if hasPuts {
		if len(putItems) <= availableBatchSize {
			putBatch = putItems
			remainingPuts = nil
		} else {
			putBatch = putItems[:availableBatchSize]
			remainingPuts = putItems[availableBatchSize:]
		}
		
		// Adjust available batch size for deletes
		availableBatchSize -= len(putBatch)
	}
	
	// Process deletes with remaining batch size
	if hasDeletes && availableBatchSize > 0 {
		if len(deleteItems) <= availableBatchSize {
			deleteBatch = deleteItems
			remainingDeletes = nil
		} else {
			deleteBatch = deleteItems[:availableBatchSize]
			remainingDeletes = deleteItems[availableBatchSize:]
		}
	} else if hasDeletes {
		// No space in this batch, all deletes go to next batch
		remainingDeletes = deleteItems
	}
	
	// Prepare update input
	input := &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  aws.String(store),
		IfMatch: etag,
	}
	
	// Add puts if we have any
	if len(putBatch) > 0 {
		input.Puts = putBatch
	}
	
	// Add deletes if we have any
	if len(deleteBatch) > 0 {
		input.Deletes = deleteBatch
	}
	
	// Update keys
	resp, err := client.UpdateKeys(r.context, input)
	
	// Handle errors
	if err != nil {
		var preconditionFailedErr *types.ValidationException
		if errors.As(err, &preconditionFailedErr) && 
		   preconditionFailedErr.Message != nil && 
		   strings.Contains(*preconditionFailedErr.Message, "Pre-Condition failed") {
			
			// Sleep for a random time between 100 and 500ms to reduce contention
			time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)
			
			// Retry with nil etag (to force fetching a new one) and the same batch
			return r.updateKeys(client, store, nil, putItems, deleteItems)
		}
		return err
	}
	
	// Process any remaining items
	if len(remainingPuts) > 0 || len(remainingDeletes) > 0 {
		return r.updateKeys(client, store, resp.ETag, remainingPuts, remainingDeletes)
	}
	
	return nil
}