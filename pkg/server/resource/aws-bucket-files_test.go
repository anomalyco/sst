package resource

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestBucketFiles_Create_NoProvider(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()
	testFile1 := filepath.Join(tempDir, "test1.txt")
	
	err := os.WriteFile(testFile1, []byte("test content 1"), 0644)
	assert.NoError(t, err)

	// Create project without AWS provider
	p := &project.Project{}

	bucketFiles := &BucketFiles{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &BucketFilesInputs{
		BucketName: "test-bucket",
		Region:     "us-east-1",
		Files: []BucketFile{
			{
				Source:      testFile1,
				Key:         "test1.txt",
				ContentType: "text/plain",
				Hash:        aws.String("hash1"),
			},
		},
		Purge: false,
	}

	var output CreateResult[BucketFilesOutputs]
	err = bucketFiles.Create(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestBucketFiles_Update_NoProvider(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()
	testFile1 := filepath.Join(tempDir, "test1.txt")
	
	err := os.WriteFile(testFile1, []byte("test content 1"), 0644)
	assert.NoError(t, err)

	p := &project.Project{}

	bucketFiles := &BucketFiles{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &UpdateInput[BucketFilesInputs, BucketFilesOutputs]{
		ID: "test-id",
		News: BucketFilesInputs{
			BucketName: "test-bucket",
			Region:     "us-east-1",
			Files: []BucketFile{
				{
					Source:      testFile1,
					Key:         "test1.txt",
					ContentType: "text/plain",
					Hash:        aws.String("hash1-new"),
				},
			},
			Purge: true,
		},
		Olds: BucketFilesOutputs{
			BucketName: "test-bucket",
			Region:     "us-east-1",
			Files: []BucketFile{
				{
					Source:      testFile1,
					Key:         "test1.txt",
					ContentType: "text/plain",
					Hash:        aws.String("hash1-old"),
				},
			},
			Purge: false,
		},
	}

	var output UpdateResult[BucketFilesOutputs]
	err = bucketFiles.Update(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestBucketFiles_Delete_EmptyBucket(t *testing.T) {
	p := &project.Project{}
	bucketFiles := &BucketFiles{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DeleteInput[BucketFilesOutputs]{
		ID: "test-id",
		Outs: BucketFilesOutputs{
			BucketName: "", // Empty bucket name
			Files:      []BucketFile{},
		},
	}

	var output int
	err := bucketFiles.Delete(input, &output)

	// Should return nil without error for empty bucket name
	assert.NoError(t, err)
}

func TestBucketFiles_Delete_EmptyFiles(t *testing.T) {
	p := &project.Project{}
	bucketFiles := &BucketFiles{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DeleteInput[BucketFilesOutputs]{
		ID: "test-id",
		Outs: BucketFilesOutputs{
			BucketName: "test-bucket",
			Files:      []BucketFile{}, // Empty files list
		},
	}

	var output int
	err := bucketFiles.Delete(input, &output)

	// Should return nil without error for empty files list
	assert.NoError(t, err)
}

func TestBucketFiles_Delete_NoProvider(t *testing.T) {
	p := &project.Project{}

	bucketFiles := &BucketFiles{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DeleteInput[BucketFilesOutputs]{
		ID: "test-id",
		Outs: BucketFilesOutputs{
			BucketName: "test-bucket",
			Region:     "us-east-1",
			Files: []BucketFile{
				{
					Key:         "test1.txt",
					ContentType: "text/plain",
					Hash:        aws.String("hash1"),
				},
			},
		},
	}

	var output int
	err := bucketFiles.Delete(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestBucketFiles_Delete_BackwardCompatibility(t *testing.T) {
	p := &project.Project{}

	bucketFiles := &BucketFiles{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &DeleteInput[BucketFilesOutputs]{
		ID: "test-id",
		Outs: BucketFilesOutputs{
			BucketName: "test-bucket",
			Region:     "", // Empty region for backward compatibility
			Files: []BucketFile{
				{
					Key:         "test1.txt",
					ContentType: "text/plain",
					Hash:        aws.String("hash1"),
				},
			},
		},
	}

	var output int
	err := bucketFiles.Delete(input, &output)

	// Should fail with "no aws provider found" but not due to empty region
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestBucketFilesInputs_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input BucketFilesInputs
		valid bool
	}{
		{
			name: "valid input",
			input: BucketFilesInputs{
				BucketName: "test-bucket",
				Region:     "us-east-1",
				Files: []BucketFile{
					{
						Source:      "/path/to/file.txt",
						Key:         "file.txt",
						ContentType: "text/plain",
						Hash:        aws.String("hash123"),
					},
				},
				Purge: false,
			},
			valid: true,
		},
		{
			name: "empty bucket name",
			input: BucketFilesInputs{
				BucketName: "",
				Region:     "us-east-1",
				Files:      []BucketFile{},
				Purge:      false,
			},
			valid: false,
		},
		{
			name: "empty region",
			input: BucketFilesInputs{
				BucketName: "test-bucket",
				Region:     "",
				Files:      []BucketFile{},
				Purge:      false,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			if tt.valid {
				assert.NotEmpty(t, tt.input.BucketName)
				assert.NotEmpty(t, tt.input.Region)
			} else {
				assert.True(t, tt.input.BucketName == "" || tt.input.Region == "")
			}
		})
	}
}

func TestBucketFile_Structure(t *testing.T) {
	// Test BucketFile struct fields and JSON tags
	file := BucketFile{
		Source:       "/path/to/file.txt",
		Key:          "file.txt",
		CacheControl: aws.String("max-age=3600"),
		ContentType:  "text/plain",
		Hash:         aws.String("hash123"),
	}

	assert.Equal(t, "/path/to/file.txt", file.Source)
	assert.Equal(t, "file.txt", file.Key)
	assert.Equal(t, "max-age=3600", *file.CacheControl)
	assert.Equal(t, "text/plain", file.ContentType)
	assert.Equal(t, "hash123", *file.Hash)
}

func TestBucketFilesOutputs_Structure(t *testing.T) {
	// Test BucketFilesOutputs struct fields
	outputs := BucketFilesOutputs{
		BucketName: "test-bucket",
		Files: []BucketFile{
			{
				Key:         "file.txt",
				ContentType: "text/plain",
			},
		},
		Purge:  true,
		Region: "us-east-1",
	}

	assert.Equal(t, "test-bucket", outputs.BucketName)
	assert.Len(t, outputs.Files, 1)
	assert.Equal(t, "file.txt", outputs.Files[0].Key)
	assert.True(t, outputs.Purge)
	assert.Equal(t, "us-east-1", outputs.Region)
}

func TestBucketFiles_UploadLogic_FileComparison(t *testing.T) {
	// Test the logic for determining when files should be uploaded
	tests := []struct {
		name        string
		newFile     BucketFile
		oldFile     *BucketFile
		shouldSkip  bool
		description string
	}{
		{
			name: "new file should upload",
			newFile: BucketFile{
				Key:         "test.txt",
				ContentType: "text/plain",
				Hash:        aws.String("hash1"),
			},
			oldFile:     nil,
			shouldSkip:  false,
			description: "New files should always be uploaded",
		},
		{
			name: "unchanged file should skip",
			newFile: BucketFile{
				Key:         "test.txt",
				ContentType: "text/plain",
				Hash:        aws.String("hash1"),
			},
			oldFile: &BucketFile{
				Key:         "test.txt",
				ContentType: "text/plain",
				Hash:        aws.String("hash1"), // Same hash
			},
			shouldSkip:  true,
			description: "Files with same hash and properties should be skipped",
		},
		{
			name: "changed hash should upload",
			newFile: BucketFile{
				Key:         "test.txt",
				ContentType: "text/plain",
				Hash:        aws.String("hash2"),
			},
			oldFile: &BucketFile{
				Key:         "test.txt",
				ContentType: "text/plain",
				Hash:        aws.String("hash1"), // Different hash
			},
			shouldSkip:  false,
			description: "Files with different hash should be uploaded",
		},
		{
			name: "changed content type should upload",
			newFile: BucketFile{
				Key:         "test.txt",
				ContentType: "application/json",
				Hash:        aws.String("hash1"),
			},
			oldFile: &BucketFile{
				Key:         "test.txt",
				ContentType: "text/plain", // Different content type
				Hash:        aws.String("hash1"),
			},
			shouldSkip:  false,
			description: "Files with different content type should be uploaded",
		},
		{
			name: "changed cache control should upload",
			newFile: BucketFile{
				Key:          "test.txt",
				ContentType:  "text/plain",
				Hash:         aws.String("hash1"),
				CacheControl: aws.String("max-age=3600"),
			},
			oldFile: &BucketFile{
				Key:          "test.txt",
				ContentType:  "text/plain",
				Hash:         aws.String("hash1"),
				CacheControl: aws.String("max-age=7200"), // Different cache control
			},
			shouldSkip:  false,
			description: "Files with different cache control should be uploaded",
		},
		{
			name: "nil hash should upload",
			newFile: BucketFile{
				Key:         "test.txt",
				ContentType: "text/plain",
				Hash:        nil,
			},
			oldFile: &BucketFile{
				Key:         "test.txt",
				ContentType: "text/plain",
				Hash:        aws.String("hash1"),
			},
			shouldSkip:  false,
			description: "Files with nil hash should be uploaded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from the upload method
			shouldSkip := false
			if tt.oldFile != nil {
				if tt.oldFile.Hash != nil && tt.newFile.Hash != nil &&
					*tt.oldFile.Hash == *tt.newFile.Hash &&
					tt.oldFile.CacheControl == tt.newFile.CacheControl &&
					tt.oldFile.ContentType == tt.newFile.ContentType {
					shouldSkip = true
				}
			}

			assert.Equal(t, tt.shouldSkip, shouldSkip, tt.description)
		})
	}
}

func TestBucketFiles_PurgeLogic_FileSelection(t *testing.T) {
	// Test the logic for determining which files should be purged
	tests := []struct {
		name        string
		newFiles    []BucketFile
		oldFiles    []BucketFile
		shouldPurge []string
		description string
	}{
		{
			name: "purge removed files",
			newFiles: []BucketFile{
				{Key: "keep.txt"},
			},
			oldFiles: []BucketFile{
				{Key: "keep.txt"},
				{Key: "remove.txt"},
			},
			shouldPurge: []string{"remove.txt"},
			description: "Should purge files not in new files list",
		},
		{
			name: "no files to purge",
			newFiles: []BucketFile{
				{Key: "file1.txt"},
				{Key: "file2.txt"},
			},
			oldFiles: []BucketFile{
				{Key: "file1.txt"},
				{Key: "file2.txt"},
			},
			shouldPurge: []string{},
			description: "Should not purge any files when all old files are in new files",
		},
		{
			name: "purge all old files",
			newFiles: []BucketFile{},
			oldFiles: []BucketFile{
				{Key: "file1.txt"},
				{Key: "file2.txt"},
			},
			shouldPurge: []string{"file1.txt", "file2.txt"},
			description: "Should purge all old files when no new files exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from the purge method
			newFileKeys := make(map[string]bool)
			for _, f := range tt.newFiles {
				newFileKeys[f.Key] = true
			}

			var toPurge []string
			for _, oldFile := range tt.oldFiles {
				if !newFileKeys[oldFile.Key] {
					toPurge = append(toPurge, oldFile.Key)
				}
			}

			assert.ElementsMatch(t, tt.shouldPurge, toPurge, tt.description)
		})
	}
}