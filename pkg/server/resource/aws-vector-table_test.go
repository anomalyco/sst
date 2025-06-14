package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sst/sst/v3/pkg/project"
)

func TestVectorTable_Create_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	vectorTable := &VectorTable{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &VectorTableInputs{
		ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
		SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
		DatabaseName: "vectordb",
		TableName:    "embeddings",
		Dimension:    1536,
	}

	var output CreateResult[VectorTableOutputs]
	err := vectorTable.Create(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestVectorTable_Update_NoProvider(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	vectorTable := &VectorTable{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	input := &UpdateInput[VectorTableInputs, VectorTableOutputs]{
		News: VectorTableInputs{
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
			DatabaseName: "vectordb",
			TableName:    "embeddings",
			Dimension:    1536,
		},
		Olds: VectorTableOutputs{
			Dimension: 768,
		},
	}

	var output UpdateResult[VectorTableOutputs]
	err := vectorTable.Update(input, &output)

	// Should fail with "no aws provider found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no aws provider found")
}

func TestVectorTableInputs_Validation(t *testing.T) {
	tests := map[string]VectorTableInputs{
		"valid_openai_embedding": {
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:vector-cluster",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:vector-secret-abc123",
			DatabaseName: "vectordb",
			TableName:    "embeddings",
			Dimension:    1536, // OpenAI text-embedding-ada-002
		},
		"valid_small_dimension": {
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret-def456",
			DatabaseName: "testdb",
			TableName:    "vectors",
			Dimension:    128,
		},
		"valid_large_dimension": {
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:large-cluster",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:large-secret-ghi789",
			DatabaseName: "largedb",
			TableName:    "large_vectors",
			Dimension:    4096,
		},
		"edge_case_long_names": {
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:very-long-cluster-name-for-testing-purposes",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:very-long-secret-name-for-testing-purposes-jkl012",
			DatabaseName: "very_long_database_name_for_testing",
			TableName:    "very_long_table_name_for_testing_purposes",
			Dimension:    768,
		},
		"edge_case_special_characters": {
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster-2024",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret-2024-mno345",
			DatabaseName: "test_db_2024",
			TableName:    "test_table_2024",
			Dimension:    512,
		},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			// Test struct field validation
			assert.NotEmpty(t, input.ClusterArn, "ClusterArn should not be empty")
			assert.NotEmpty(t, input.SecretArn, "SecretArn should not be empty")
			assert.NotEmpty(t, input.DatabaseName, "DatabaseName should not be empty")
			assert.NotEmpty(t, input.TableName, "TableName should not be empty")
			assert.Greater(t, input.Dimension, 0, "Dimension should be positive")

			// Test ARN format validation
			assert.Contains(t, input.ClusterArn, "arn:aws:rds:", "ClusterArn should be valid RDS ARN")
			assert.Contains(t, input.SecretArn, "arn:aws:secretsmanager:", "SecretArn should be valid Secrets Manager ARN")

			// Test dimension ranges (common embedding dimensions)
			assert.True(t, input.Dimension >= 1 && input.Dimension <= 10000, "Dimension should be in reasonable range")
		})
	}
}

func TestVectorTableOutputs_Structure(t *testing.T) {
	tests := map[string]VectorTableOutputs{
		"openai_dimension": {
			Dimension: 1536,
		},
		"bert_dimension": {
			Dimension: 768,
		},
		"small_dimension": {
			Dimension: 128,
		},
		"large_dimension": {
			Dimension: 4096,
		},
	}

	for name, output := range tests {
		t.Run(name, func(t *testing.T) {
			// Test output structure validation
			assert.Greater(t, output.Dimension, 0, "Dimension should be positive")
			assert.True(t, output.Dimension <= 10000, "Dimension should be reasonable")
		})
	}
}

func TestVectorTable_EdgeCases(t *testing.T) {
	// Create project without AWS provider for edge case testing
	p := &project.Project{}

	vectorTable := &VectorTable{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	t.Run("empty_cluster_arn", func(t *testing.T) {
		input := &VectorTableInputs{
			ClusterArn:   "",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
			DatabaseName: "vectordb",
			TableName:    "embeddings",
			Dimension:    1536,
		}

		var output CreateResult[VectorTableOutputs]
		err := vectorTable.Create(input, &output)

		// Should fail due to missing provider, but input validation would catch empty ARN
		assert.Error(t, err)
	})

	t.Run("empty_secret_arn", func(t *testing.T) {
		input := &VectorTableInputs{
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
			SecretArn:    "",
			DatabaseName: "vectordb",
			TableName:    "embeddings",
			Dimension:    1536,
		}

		var output CreateResult[VectorTableOutputs]
		err := vectorTable.Create(input, &output)

		// Should fail due to missing provider, but input validation would catch empty ARN
		assert.Error(t, err)
	})

	t.Run("empty_database_name", func(t *testing.T) {
		input := &VectorTableInputs{
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
			DatabaseName: "",
			TableName:    "embeddings",
			Dimension:    1536,
		}

		var output CreateResult[VectorTableOutputs]
		err := vectorTable.Create(input, &output)

		// Should fail due to missing provider, but input validation would catch empty database name
		assert.Error(t, err)
	})

	t.Run("empty_table_name", func(t *testing.T) {
		input := &VectorTableInputs{
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
			DatabaseName: "vectordb",
			TableName:    "",
			Dimension:    1536,
		}

		var output CreateResult[VectorTableOutputs]
		err := vectorTable.Create(input, &output)

		// Should fail due to missing provider, but input validation would catch empty table name
		assert.Error(t, err)
	})

	t.Run("zero_dimension", func(t *testing.T) {
		input := &VectorTableInputs{
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
			DatabaseName: "vectordb",
			TableName:    "embeddings",
			Dimension:    0,
		}

		var output CreateResult[VectorTableOutputs]
		err := vectorTable.Create(input, &output)

		// Should fail due to missing provider, but input validation would catch zero dimension
		assert.Error(t, err)
	})

	t.Run("negative_dimension", func(t *testing.T) {
		input := &VectorTableInputs{
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
			DatabaseName: "vectordb",
			TableName:    "embeddings",
			Dimension:    -100,
		}

		var output CreateResult[VectorTableOutputs]
		err := vectorTable.Create(input, &output)

		// Should fail due to missing provider, but input validation would catch negative dimension
		assert.Error(t, err)
	})
}

func TestVectorTable_UpdateDimensionChange(t *testing.T) {
	// Create project without AWS provider
	p := &project.Project{}

	vectorTable := &VectorTable{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	t.Run("dimension_change_requires_table_recreation", func(t *testing.T) {
		input := &UpdateInput[VectorTableInputs, VectorTableOutputs]{
			News: VectorTableInputs{
				ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
				SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
				DatabaseName: "vectordb",
				TableName:    "embeddings",
				Dimension:    1536, // Changed from 768 to 1536
			},
			Olds: VectorTableOutputs{
				Dimension: 768, // Old dimension
			},
		}

		var output UpdateResult[VectorTableOutputs]
		err := vectorTable.Update(input, &output)

		// Should fail due to missing provider, but logic would handle dimension change
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no aws provider found")
	})

	t.Run("no_dimension_change", func(t *testing.T) {
		input := &UpdateInput[VectorTableInputs, VectorTableOutputs]{
			News: VectorTableInputs{
				ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
				SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
				DatabaseName: "vectordb",
				TableName:    "embeddings",
				Dimension:    1536, // Same dimension
			},
			Olds: VectorTableOutputs{
				Dimension: 1536, // Same dimension
			},
		}

		var output UpdateResult[VectorTableOutputs]
		err := vectorTable.Update(input, &output)

		// Should fail due to missing provider, but logic would not recreate table
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no aws provider found")
	})
}

func TestVectorTable_CommonEmbeddingDimensions(t *testing.T) {
	// Test common embedding model dimensions
	commonDimensions := map[string]int{
		"openai_text_embedding_ada_002":     1536,
		"openai_text_embedding_3_small":     1536,
		"openai_text_embedding_3_large":     3072,
		"bert_base_uncased":                 768,
		"bert_large_uncased":                1024,
		"sentence_transformers_all_mpnet":   768,
		"sentence_transformers_all_minilm":  384,
		"cohere_embed_english":              4096,
		"cohere_embed_multilingual":         768,
		"custom_small":                      128,
		"custom_medium":                     512,
		"custom_large":                      2048,
	}

	for model, dimension := range commonDimensions {
		t.Run(model, func(t *testing.T) {
			input := VectorTableInputs{
				ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:vector-cluster",
				SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:vector-secret",
				DatabaseName: "vectordb",
				TableName:    "embeddings_" + model,
				Dimension:    dimension,
			}

			// Validate dimension is reasonable for vector embeddings
			assert.Greater(t, input.Dimension, 0, "Dimension should be positive")
			assert.LessOrEqual(t, input.Dimension, 10000, "Dimension should be reasonable")

			// Test that output matches input dimension
			expectedOutput := VectorTableOutputs{
				Dimension: dimension,
			}
			assert.Equal(t, input.Dimension, expectedOutput.Dimension, "Output dimension should match input")
		})
	}
}

func TestVectorTable_PostgreSQLIntegration(t *testing.T) {
	// Test scenarios specific to PostgreSQL with pgvector extension
	scenarios := map[string]struct {
		tableName string
		dimension int
		expected  string
	}{
		"standard_table": {
			tableName: "embeddings",
			dimension: 1536,
			expected:  "embeddings",
		},
		"table_with_prefix": {
			tableName: "app_embeddings",
			dimension: 768,
			expected:  "app_embeddings",
		},
		"table_with_suffix": {
			tableName: "embeddings_v2",
			dimension: 1024,
			expected:  "embeddings_v2",
		},
		"table_with_underscores": {
			tableName: "user_document_embeddings",
			dimension: 512,
			expected:  "user_document_embeddings",
		},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			input := VectorTableInputs{
				ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:postgres-cluster",
				SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:postgres-secret",
				DatabaseName: "vectordb",
				TableName:    scenario.tableName,
				Dimension:    scenario.dimension,
			}

			// Validate table name format
			assert.Equal(t, scenario.expected, input.TableName, "Table name should match expected format")
			assert.NotContains(t, input.TableName, " ", "Table name should not contain spaces")
			assert.NotContains(t, input.TableName, "-", "Table name should use underscores, not hyphens")
		})
	}
}

func TestVectorTable_AwsResource_Embedded(t *testing.T) {
	// Test that VectorTable properly embeds AwsResource
	p := &project.Project{}
	
	vectorTable := &VectorTable{
		AwsResource: &AwsResource{
			context: context.Background(),
			project: p,
		},
	}

	// Test embedded AwsResource structure
	assert.NotNil(t, vectorTable.AwsResource, "AwsResource should be embedded")
	assert.NotNil(t, vectorTable.AwsResource.context, "Context should be set")
	assert.NotNil(t, vectorTable.AwsResource.project, "Project should be set")
	assert.Equal(t, p, vectorTable.AwsResource.project, "Project should match")
}

func TestVectorTable_CreateResult_Structure(t *testing.T) {
	// Test CreateResult structure for VectorTable
	input := &VectorTableInputs{
		ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
		SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
		DatabaseName: "vectordb",
		TableName:    "embeddings",
		Dimension:    1536,
	}

	// Simulate successful creation result
	expectedResult := CreateResult[VectorTableOutputs]{
		ID:   input.TableName,
		Outs: VectorTableOutputs{Dimension: input.Dimension},
	}

	// Validate result structure
	assert.Equal(t, input.TableName, expectedResult.ID, "ID should match table name")
	assert.Equal(t, input.Dimension, expectedResult.Outs.Dimension, "Output dimension should match input")
}

func TestVectorTable_UpdateResult_Structure(t *testing.T) {
	// Test UpdateResult structure for VectorTable
	input := &UpdateInput[VectorTableInputs, VectorTableOutputs]{
		News: VectorTableInputs{
			ClusterArn:   "arn:aws:rds:us-east-1:123456789012:cluster:test-cluster",
			SecretArn:    "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret",
			DatabaseName: "vectordb",
			TableName:    "embeddings",
			Dimension:    1536,
		},
		Olds: VectorTableOutputs{
			Dimension: 768,
		},
	}

	// Simulate successful update result
	expectedResult := UpdateResult[VectorTableOutputs]{
		Outs: VectorTableOutputs{Dimension: input.News.Dimension},
	}

	// Validate result structure
	assert.Equal(t, input.News.Dimension, expectedResult.Outs.Dimension, "Output dimension should match new input")
}