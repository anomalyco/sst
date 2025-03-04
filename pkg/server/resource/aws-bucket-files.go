package resource

import (
	"bytes"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BucketFiles struct {
	*AwsResource
}

type BucketFile struct {
	Source       string  `json:"source"`
	Key          string  `json:"key"`
	CacheControl *string `json:"cacheControl,omitempty"`
	ContentType  string  `json:"contentType"`
	Hash         *string `json:"hash,omitempty"`
}

type BucketFilesInputs struct {
	BucketName string       `json:"bucketName"`
	Files      []BucketFile `json:"files"`
	Purge      bool         `json:"purge"`
	Region     string       `json:"region"`
}

type BucketFilesOutputs struct {
	BucketName string       `json:"bucketName,omitempty"`
	Files      []BucketFile `json:"files,omitempty"`
	Purge      bool         `json:"purge,omitempty"`
	Region     string       `json:"region,omitempty"`
}

func (r *BucketFiles) Create(input *BucketFilesInputs, output *CreateResult[BucketFilesOutputs]) error {
	cfg, err := r.config()
	if err != nil {
		return err
	}
	cfg.Region = input.Region
	s3Client := s3.NewFromConfig(cfg)

	if err := r.upload(s3Client, input.BucketName, input.Files, nil); err != nil {
		return err
	}

	*output = CreateResult[BucketFilesOutputs]{
		ID: "files",
		Outs: BucketFilesOutputs{
			BucketName: input.BucketName,
			Files:      input.Files,
			Purge:      input.Purge,
			Region:     input.Region,
		},
	}
	return nil
}

func (r *BucketFiles) Update(input *UpdateInput[BucketFilesInputs, BucketFilesOutputs], output *UpdateResult[BucketFilesOutputs]) error {
	cfg, err := r.config()
	if err != nil {
		return err
	}
	cfg.Region = input.News.Region
	s3Client := s3.NewFromConfig(cfg)

	oldFiles := input.Olds.Files
	if input.News.BucketName != input.Olds.BucketName {
		oldFiles = nil
	}

	if err := r.upload(s3Client, input.News.BucketName, input.News.Files, oldFiles); err != nil {
		return err
	}

	if input.News.Purge {
		if err := r.purge(s3Client, input.News.BucketName, input.News.Files, oldFiles); err != nil {
			return err
		}
	}

	*output = UpdateResult[BucketFilesOutputs]{
		Outs: BucketFilesOutputs{
			BucketName: input.News.BucketName,
			Files:      input.News.Files,
			Purge:      input.News.Purge,
			Region:     input.News.Region,
		},
	}
	return nil
}

func (r *BucketFiles) Delete(input *DeleteInput[BucketFilesOutputs], output *int) error {
	if input.Outs.BucketName == "" || len(input.Outs.Files) == 0 {
		return nil
	}

	cfg, err := r.config()
	if err != nil {
		return err
	}
	// input.Outs.Region can be empty if last deployed in version is 3.2.59 or earlier
	if input.Outs.Region != "" {
		cfg.Region = input.Outs.Region
	}
	s3Client := s3.NewFromConfig(cfg)

	return r.purge(s3Client, input.Outs.BucketName, nil, input.Outs.Files)
}

func (r *BucketFiles) upload(client *s3.Client, bucketName string, files []BucketFile, oldFiles []BucketFile) error {
	// Create map of existing files
	oldFilesMap := make(map[string]BucketFile)
	for _, f := range oldFiles {
		oldFilesMap[f.Key] = f
	}

	// Create channels for work distribution and error collection
	filesChan := make(chan BucketFile)
	errChan := make(chan error, len(files))
	var wg sync.WaitGroup

	// Start worker pool (10 workers)
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each worker processes files from the channel
			for file := range filesChan {
				// write start timestamp with nanoseconds to file
				content, err := os.ReadFile(file.Source)
				if err != nil {
					errChan <- err
					return
				}

				_, err = client.PutObject(r.context, &s3.PutObjectInput{
					Bucket:       aws.String(bucketName),
					Key:          aws.String(file.Key),
					Body:         bytes.NewReader(content),
					CacheControl: file.CacheControl,
					ContentType:  aws.String(file.ContentType),
				})
				if err != nil {
					errChan <- err
				}
			}
		}()
	}

	// Send files that need uploading to the channel
	go func() {
		for _, file := range files {
			oldFile, exists := oldFilesMap[file.Key]
			if exists && oldFile.Hash != nil && *oldFile.Hash == *file.Hash &&
				oldFile.CacheControl == file.CacheControl &&
				oldFile.ContentType == file.ContentType {
				continue
			}
			filesChan <- file
		}
		close(filesChan)
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *BucketFiles) purge(client *s3.Client, bucketName string, files []BucketFile, oldFiles []BucketFile) error {
	newFileKeys := make(map[string]bool)
	for _, f := range files {
		newFileKeys[f.Key] = true
	}

	for _, oldFile := range oldFiles {
		if !newFileKeys[oldFile.Key] {
			_, err := client.DeleteObject(r.context, &s3.DeleteObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(oldFile.Key),
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

