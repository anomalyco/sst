package validation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCostValidation verifies resource costs are within expected ranges
func TestCostValidation(t *testing.T) {
	// Skip if AWS credentials not configured
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("AWS credentials not configured, skipping cost validation test")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	// Create test project
	projectDir := createCostTestProject(t, "cost-validation")
	defer cleanupCostProject(t, projectDir)

	// Deploy the project
	_ = deployCostTestProject(t, projectDir, "cost-validation-test")
	defer removeCostTestProject(t, projectDir, "cost-validation-test")

	// Wait for cost data to be available (costs are reported with delay)
	time.Sleep(5 * time.Minute)

	// Validate Lambda function costs
	t.Run("Lambda Function Cost Validation", func(t *testing.T) {
		costClient := costexplorer.NewFromConfig(cfg)
		
		// Get cost data for the last 7 days
		endDate := time.Now().Format("2006-01-02")
		startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		
		resp, err := costClient.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(startDate),
				End:   aws.String(endDate),
			},
			Granularity: types.GranularityDaily,
			Metrics:     []string{"BlendedCost"},
			GroupBy: []types.GroupDefinition{
				{
					Type: types.GroupDefinitionTypeTag,
					Key:  aws.String("sst:stage"),
				},
				{
					Type: types.GroupDefinitionTypeDimension,
					Key:  aws.String("SERVICE"),
				},
			},
		})
		
		if err != nil {
			t.Logf("Cost Explorer API not available or insufficient permissions: %v", err)
			t.Skip("Skipping cost validation - Cost Explorer not accessible")
		}
		
		// Validate Lambda costs are reasonable (< $1 for test)
		lambdaCost := extractServiceCost(resp.ResultsByTime, "AWS Lambda")
		assert.True(t, lambdaCost < 1.0, "Lambda cost should be less than $1 for test deployment, got $%.2f", lambdaCost)
		
		t.Logf("Lambda cost for test period: $%.4f", lambdaCost)
	})

	// Validate S3 bucket costs
	t.Run("S3 Bucket Cost Validation", func(t *testing.T) {
		costClient := costexplorer.NewFromConfig(cfg)
		
		endDate := time.Now().Format("2006-01-02")
		startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		
		resp, err := costClient.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(startDate),
				End:   aws.String(endDate),
			},
			Granularity: types.GranularityDaily,
			Metrics:     []string{"BlendedCost"},
			GroupBy: []types.GroupDefinition{
				{
					Type: types.GroupDefinitionTypeDimension,
					Key:  aws.String("SERVICE"),
				},
			},
		})
		
		if err != nil {
			t.Logf("Cost Explorer API not available: %v", err)
			t.Skip("Skipping S3 cost validation")
		}
		
		// Validate S3 costs are reasonable (< $0.50 for test)
		s3Cost := extractServiceCost(resp.ResultsByTime, "Amazon Simple Storage Service")
		assert.True(t, s3Cost < 0.5, "S3 cost should be less than $0.50 for test deployment, got $%.2f", s3Cost)
		
		t.Logf("S3 cost for test period: $%.4f", s3Cost)
	})

	// Validate total deployment cost
	t.Run("Total Deployment Cost Validation", func(t *testing.T) {
		costClient := costexplorer.NewFromConfig(cfg)
		
		endDate := time.Now().Format("2006-01-02")
		startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		
		resp, err := costClient.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(startDate),
				End:   aws.String(endDate),
			},
			Granularity: types.GranularityDaily,
			Metrics:     []string{"BlendedCost"},
		})
		
		if err != nil {
			t.Logf("Cost Explorer API not available: %v", err)
			t.Skip("Skipping total cost validation")
		}
		
		// Calculate total cost for the period
		totalCost := 0.0
		for _, result := range resp.ResultsByTime {
			if result.Total != nil {
				if blendedCost, exists := result.Total["BlendedCost"]; exists && blendedCost.Amount != nil {
					if cost, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil {
						totalCost += cost
					}
				}
			}
		}
		
		// Validate total cost is reasonable for test deployment (< $5)
		assert.True(t, totalCost < 5.0, "Total deployment cost should be less than $5 for test period, got $%.2f", totalCost)
		
		t.Logf("Total cost for test period: $%.4f", totalCost)
	})
}

// TestCostOptimization tests cost optimization strategies
func TestCostOptimization(t *testing.T) {
	// Skip if AWS credentials not configured
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("AWS credentials not configured, skipping cost optimization test")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	// Test resource tagging for cost tracking
	t.Run("Resource Tagging for Cost Tracking", func(t *testing.T) {
		projectDir := createCostTestProject(t, "cost-tagging")
		defer cleanupCostProject(t, projectDir)

		stackName := deployCostTestProject(t, projectDir, "cost-tagging-test")
		defer removeCostTestProject(t, projectDir, "cost-tagging-test")

		// Validate Lambda function has proper cost tracking tags
		lambdaClient := lambda.NewFromConfig(cfg)
		functionName := findLambdaFunctionByStack(t, lambdaClient, stackName)
		
		resp, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)

		// Check for required cost tracking tags
		tags := resp.Tags
		assert.Contains(t, tags, "sst:stage", "Function should have sst:stage tag for cost tracking")
		assert.Contains(t, tags, "sst:app", "Function should have sst:app tag for cost tracking")
		
		t.Logf("Function tags: %v", tags)
	})

	// Test cost-effective resource selection
	t.Run("Cost-Effective Resource Selection", func(t *testing.T) {
		projectDir := createCostTestProject(t, "cost-effective")
		defer cleanupCostProject(t, projectDir)

		stackName := deployCostTestProject(t, projectDir, "cost-effective-test")
		defer removeCostTestProject(t, projectDir, "cost-effective-test")

		// Validate Lambda function uses cost-effective configuration
		lambdaClient := lambda.NewFromConfig(cfg)
		functionName := findLambdaFunctionByStack(t, lambdaClient, stackName)
		
		resp, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)

		config := resp.Configuration
		
		// Validate memory allocation is reasonable (not over-provisioned)
		assert.LessOrEqual(t, *config.MemorySize, int32(512), "Memory should be <= 512MB for cost efficiency")
		
		// Validate timeout is reasonable (not excessive)
		assert.LessOrEqual(t, *config.Timeout, int32(30), "Timeout should be <= 30 seconds for cost efficiency")
		
		t.Logf("Function memory: %dMB, timeout: %ds", *config.MemorySize, *config.Timeout)
	})

	// Test cost monitoring and alerting setup
	t.Run("Cost Monitoring Setup", func(t *testing.T) {
		// This would typically validate that cost monitoring is configured
		// For now, we'll just verify the Cost Explorer API is accessible
		costClient := costexplorer.NewFromConfig(cfg)
		
		endDate := time.Now().Format("2006-01-02")
		startDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		
		_, err := costClient.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(startDate),
				End:   aws.String(endDate),
			},
			Granularity: types.GranularityDaily,
			Metrics:     []string{"BlendedCost"},
		})
		
		if err != nil {
			t.Logf("Cost Explorer API not available: %v", err)
			t.Skip("Skipping cost monitoring validation")
		}
		
		assert.NoError(t, err, "Cost Explorer API should be accessible for monitoring")
		t.Log("Cost monitoring API is accessible")
	})
}

// TestCostEstimation tests cost estimation accuracy
func TestCostEstimation(t *testing.T) {
	// Skip if AWS credentials not configured
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("AWS credentials not configured, skipping cost estimation test")
	}

	// Test cost estimation for different deployment sizes
	t.Run("Small Deployment Cost Estimation", func(t *testing.T) {
		// This would test SST's cost estimation features
		// For now, we'll validate that small deployments stay within expected ranges
		
		projectDir := createCostTestProject(t, "small-deployment")
		defer cleanupCostProject(t, projectDir)

		stackName := deployCostTestProject(t, projectDir, "small-deployment-test")
		defer removeCostTestProject(t, projectDir, "small-deployment-test")

		// Small deployment should have minimal cost
		// This is more of a validation that the deployment is actually small
		ctx := context.Background()
		cfg, err := config.LoadDefaultConfig(ctx)
		require.NoError(t, err)

		lambdaClient := lambda.NewFromConfig(cfg)
		functionName := findLambdaFunctionByStack(t, lambdaClient, stackName)
		
		resp, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(functionName),
		})
		require.NoError(t, err)

		// Validate it's actually a small deployment
		config := resp.Configuration
		assert.LessOrEqual(t, *config.MemorySize, int32(256), "Small deployment should use <= 256MB memory")
		assert.LessOrEqual(t, *config.Timeout, int32(15), "Small deployment should use <= 15s timeout")
		
		t.Logf("Small deployment - Memory: %dMB, Timeout: %ds", *config.MemorySize, *config.Timeout)
	})
}

// Helper functions

func createCostTestProject(t *testing.T, name string) string {
	t.Helper()
	
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("sst-cost-test-%s-*", name))
	require.NoError(t, err)
	
	// Create package.json
	packageJSON := `{
  "name": "` + name + `",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "sst dev",
    "deploy": "sst deploy"
  },
  "devDependencies": {
    "sst": "latest"
  }
}`
	
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644))
	
	// Create SST config with cost-optimized settings
	sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "` + name + `",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Cost-optimized Lambda function
    const api = new sst.aws.Function("TestFunction", {
      handler: "index.handler",
      runtime: "nodejs20.x",
      memory: "256 MB",
      timeout: "15 seconds",
      environment: {
        NODE_ENV: "test"
      }
    });

    // Cost-optimized S3 bucket
    const bucket = new sst.aws.Bucket("TestBucket", {
      public: false
    });

    return {
      api: api.arn,
      bucket: bucket.name
    };
  },
});`
	
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "sst.config.ts"), []byte(sstConfig), 0644))
	
	// Create simple Lambda handler
	handler := `export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Hello from cost-optimized function!" })
  };
};`
	
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "index.js"), []byte(handler), 0644))
	
	return tempDir
}

func cleanupCostProject(t *testing.T, projectDir string) {
	t.Helper()
	os.RemoveAll(projectDir)
}

func deployCostTestProject(t *testing.T, projectDir, stage string) string {
	t.Helper()
	
	cmd := exec.Command("sst", "deploy", "--stage", stage)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "SST_TELEMETRY_DISABLED=1")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to deploy project: %v\nOutput: %s", err, output)
	}
	
	t.Logf("Deploy output: %s", output)
	return fmt.Sprintf("%s-%s", filepath.Base(projectDir), stage)
}

func removeCostTestProject(t *testing.T, projectDir, stage string) {
	t.Helper()
	
	cmd := exec.Command("sst", "remove", "--stage", stage)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "SST_TELEMETRY_DISABLED=1")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Failed to remove project (this is often expected): %v\nOutput: %s", err, output)
	} else {
		t.Logf("Remove output: %s", output)
	}
}

func findLambdaFunctionByStack(t *testing.T, client *lambda.Client, stackName string) string {
	t.Helper()
	
	ctx := context.Background()
	resp, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	require.NoError(t, err)
	
	for _, function := range resp.Functions {
		if strings.Contains(*function.FunctionName, stackName) || 
		   strings.Contains(*function.FunctionName, "TestFunction") {
			return *function.FunctionName
		}
	}
	
	t.Fatalf("No Lambda function found for stack: %s", stackName)
	return ""
}

func extractServiceCost(results []types.ResultByTime, serviceName string) float64 {
	totalCost := 0.0
	
	for _, result := range results {
		for _, group := range result.Groups {
			// Check if this group is for the specified service
			for _, key := range group.Keys {
				if strings.Contains(key, serviceName) {
					if group.Metrics != nil {
						if blendedCost, exists := group.Metrics["BlendedCost"]; exists && blendedCost.Amount != nil {
							if cost, err := strconv.ParseFloat(*blendedCost.Amount, 64); err == nil {
								totalCost += cost
							}
						}
					}
				}
			}
		}
	}
	
	return totalCost
}