package resource

import (
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
		if err := r.purge(client, input.News.Store, input.News.Namespace, input.News.Entries, oldEntries); err != nil {
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
	if input.Outs.Store == "" || len(input.Outs.Entries) == 0 {
		return nil
	}

	cfg, err := r.config()
	if err != nil {
		return err
	}
	client := cloudfrontkeyvaluestore.NewFromConfig(cfg)

	return r.purge(client, input.Outs.Store, input.Outs.Namespace, nil, input.Outs.Entries)
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

	// Get store ETag
	descResp, err := client.DescribeKeyValueStore(r.context, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: aws.String(store),
	})
	if err != nil {
		return err
	}
	etag := descResp.ETag

	// Process in batches of 50
	const batchSize = 50
	for i := 0; i < len(keysToUpdate); i += batchSize {
		end := min(i+batchSize, len(keysToUpdate))
		
		// Create put items
		putItems := make([]types.PutKeyRequestListItem, 0, end-i)
		for _, kv := range keysToUpdate[i:end] {
			putItems = append(putItems, types.PutKeyRequestListItem{
				Key:   aws.String(namespace + ":" + kv.key),
				Value: aws.String(kv.value),
			})
		}
		
		// Upload batch
		resp, err := client.UpdateKeys(r.context, &cloudfrontkeyvaluestore.UpdateKeysInput{
			KvsARN:  aws.String(store),
			IfMatch: etag,
			Puts:    putItems,
		})
		if err != nil {
			return err
		}
		
		// Update ETag for next batch
		etag = resp.ETag
	}

	return nil
}

func (r *KvKeys) purge(client *cloudfrontkeyvaluestore.Client, store string, namespace string, entries map[string]string, oldEntries map[string]string) error {
	// Collect keys to delete
	var keysToDelete []string
	for oldKey := range oldEntries {
		if _, exists := entries[oldKey]; !exists {
			keysToDelete = append(keysToDelete, oldKey)
		}
	}

	if len(keysToDelete) == 0 {
		return nil
	}

	// Get store ETag
	descResp, err := client.DescribeKeyValueStore(r.context, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: aws.String(store),
	})
	if err != nil {
		return err
	}
	etag := descResp.ETag

	// Process in batches of 50
	const batchSize = 50
	for i := 0; i < len(keysToDelete); i += batchSize {
		end := min(i+batchSize, len(keysToDelete))
		
		// Create delete items
		deleteItems := make([]types.DeleteKeyRequestListItem, 0, end-i)
		for _, key := range keysToDelete[i:end] {
			deleteItems = append(deleteItems, types.DeleteKeyRequestListItem{
				Key: aws.String(namespace + ":" + key),
			})
		}

		// Delete batch
		resp, err := client.UpdateKeys(r.context, &cloudfrontkeyvaluestore.UpdateKeysInput{
			KvsARN:  aws.String(store),
			IfMatch: etag,
			Deletes: deleteItems,
		})
		if err != nil {
			return err
		}
		
		// Update ETag for next batch
		etag = resp.ETag
	}

	return nil
}

