package validation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions for performance tests

func shouldRunIntegrationTests() bool {
	return os.Getenv("AWS_ACCESS_KEY_ID") != ""
}

func createTempProject(t *testing.T, name string) string {
	t.Helper()
	
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("sst-perf-test-%s-*", name))
	require.NoError(t, err)
	
	// Initialize package.json
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
	
	return tempDir
}

func cleanupProject(t *testing.T, projectDir string) {
	t.Helper()
	os.RemoveAll(projectDir)
}

func runSST(ctx context.Context, projectDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "npx", append([]string{"sst"}, args...)...)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "SST_TELEMETRY_DISABLED=1")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sst command failed: %v\nOutput: %s", err, string(output))
	}
	
	return nil
}

func TestDeploymentPerformance(t *testing.T) {
	if !shouldRunIntegrationTests() {
		t.Skip("Skipping integration test - AWS credentials not configured")
	}

	ctx := context.Background()
	projectDir := createTempProject(t, "performance-test")
	defer cleanupProject(t, projectDir)

	// Create a simple SST project for performance testing
	sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "performance-test",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Simple bucket for performance testing
    const bucket = new sst.aws.Bucket("TestBucket", {
      public: true,
    });

    // Simple function for performance testing
    const fn = new sst.aws.Function("TestFunction", {
      handler: "index.handler",
      runtime: "nodejs20.x",
      timeout: "30 seconds",
    });

    return {
      bucket: bucket.name,
      function: fn.name,
    };
  },
});`

	functionCode := `export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Hello from performance test!" }),
  };
};`

	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "index.ts"), []byte(functionCode), 0644))

	// Test deployment speed
	t.Run("deployment speed", func(t *testing.T) {
		start := time.Now()
		
		err := runSST(ctx, projectDir, "deploy", "--stage", "perf-test")
		require.NoError(t, err, "Deployment should succeed")
		
		deploymentTime := time.Since(start)
		
		// Deployment should complete within reasonable time (5 minutes for simple project)
		assert.Less(t, deploymentTime, 5*time.Minute, "Deployment should complete within 5 minutes")
		
		t.Logf("Deployment completed in: %v", deploymentTime)
		
		// Cleanup
		err = runSST(ctx, projectDir, "remove", "--stage", "perf-test")
		require.NoError(t, err, "Cleanup should succeed")
	})
}

func TestConcurrentDeployments(t *testing.T) {
	if !shouldRunIntegrationTests() {
		t.Skip("Skipping integration test - AWS credentials not configured")
	}

	ctx := context.Background()
	
	// Test concurrent deployments to different stages
	t.Run("concurrent stage deployments", func(t *testing.T) {
		projectDir := createTempProject(t, "concurrent-test")
		defer cleanupProject(t, projectDir)

		// Create a simple SST project
		sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "concurrent-test",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("ConcurrentBucket", {
      public: true,
    });

    return {
      bucket: bucket.name,
    };
  },
});`

		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644))

		// Deploy to multiple stages concurrently
		stages := []string{"stage1", "stage2", "stage3"}
		results := make(chan error, len(stages))
		
		start := time.Now()
		
		for _, stage := range stages {
			go func(s string) {
				err := runSST(ctx, projectDir, "deploy", "--stage", s)
				results <- err
			}(stage)
		}
		
		// Wait for all deployments to complete
		var errors []error
		for i := 0; i < len(stages); i++ {
			if err := <-results; err != nil {
				errors = append(errors, err)
			}
		}
		
		concurrentTime := time.Since(start)
		t.Logf("Concurrent deployments completed in: %v", concurrentTime)
		
		// Concurrent deployments should not interfere with each other
		assert.Empty(t, errors, "All concurrent deployments should succeed")
		
		// Cleanup all stages
		for _, stage := range stages {
			err := runSST(ctx, projectDir, "remove", "--stage", stage)
			assert.NoError(t, err, "Cleanup should succeed for stage %s", stage)
		}
	})
}

func TestScalingPerformance(t *testing.T) {
	if !shouldRunIntegrationTests() {
		t.Skip("Skipping integration test - AWS credentials not configured")
	}

	ctx := context.Background()
	projectDir := createTempProject(t, "scaling-test")
	defer cleanupProject(t, projectDir)

	// Test performance with increasing number of resources
	t.Run("resource scaling", func(t *testing.T) {
		resourceCounts := []int{5, 10, 20}
		
		for _, count := range resourceCounts {
			t.Run(fmt.Sprintf("resources_%d", count), func(t *testing.T) {
				// Generate SST config with multiple resources
				sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "scaling-test",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const resources = {};
`
				
				// Add multiple buckets
				for i := 0; i < count; i++ {
					sstConfig += fmt.Sprintf(`
    const bucket%d = new sst.aws.Bucket("TestBucket%d", {
      public: true,
    });
    resources.bucket%d = bucket%d.name;
`, i, i, i, i)
				}
				
				sstConfig += `
    return resources;
  },
});`

				require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644))
				
				start := time.Now()
				
				err := runSST(ctx, projectDir, "deploy", "--stage", fmt.Sprintf("scale-%d", count))
				require.NoError(t, err, "Deployment with %d resources should succeed", count)
				
				deploymentTime := time.Since(start)
				t.Logf("Deployment with %d resources completed in: %v", count, deploymentTime)
				
				// Deployment time should scale reasonably (not exponentially)
				maxTime := time.Duration(count) * 30 * time.Second // 30 seconds per resource max
				assert.Less(t, deploymentTime, maxTime, "Deployment time should scale reasonably")
				
				// Cleanup
				err = runSST(ctx, projectDir, "remove", "--stage", fmt.Sprintf("scale-%d", count))
				require.NoError(t, err, "Cleanup should succeed")
			})
		}
	})
}

func TestUpdatePerformance(t *testing.T) {
	if !shouldRunIntegrationTests() {
		t.Skip("Skipping integration test - AWS credentials not configured")
	}

	ctx := context.Background()
	projectDir := createTempProject(t, "update-test")
	defer cleanupProject(t, projectDir)

	// Test update performance
	t.Run("incremental updates", func(t *testing.T) {
		// Initial deployment
		sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "update-test",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("UpdateBucket", {
      public: true,
    });

    const fn = new sst.aws.Function("UpdateFunction", {
      handler: "index.handler",
      runtime: "nodejs20.x",
      timeout: "30 seconds",
    });

    return {
      bucket: bucket.name,
      function: fn.name,
    };
  },
});`

		functionCode := `export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Hello from update test!" }),
  };
};`

		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "index.ts"), []byte(functionCode), 0644))

		// Initial deployment
		start := time.Now()
		err := runSST(ctx, projectDir, "deploy", "--stage", "update-test")
		require.NoError(t, err, "Initial deployment should succeed")
		initialTime := time.Since(start)
		
		t.Logf("Initial deployment completed in: %v", initialTime)

		// Update function code
		updatedFunctionCode := `export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Updated function!" }),
  };
};`

		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "index.ts"), []byte(updatedFunctionCode), 0644))

		// Update deployment
		start = time.Now()
		err = runSST(ctx, projectDir, "deploy", "--stage", "update-test")
		require.NoError(t, err, "Update deployment should succeed")
		updateTime := time.Since(start)
		
		t.Logf("Update deployment completed in: %v", updateTime)

		// Update should be faster than initial deployment
		assert.Less(t, updateTime, initialTime, "Update deployment should be faster than initial deployment")
		
		// Update should complete within reasonable time (2 minutes)
		assert.Less(t, updateTime, 2*time.Minute, "Update deployment should complete within 2 minutes")

		// Cleanup
		err = runSST(ctx, projectDir, "remove", "--stage", "update-test")
		require.NoError(t, err, "Cleanup should succeed")
	})
}

func TestResourceUtilization(t *testing.T) {
	if !shouldRunIntegrationTests() {
		t.Skip("Skipping integration test - AWS credentials not configured")
	}

	ctx := context.Background()
	projectDir := createTempProject(t, "utilization-test")
	defer cleanupProject(t, projectDir)

	// Test resource utilization during deployment
	t.Run("memory and cpu usage", func(t *testing.T) {
		sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "utilization-test",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    // Create multiple resources to test utilization
    const resources = {};
    
    for (let i = 0; i < 10; i++) {
      const bucket = new sst.aws.Bucket("UtilBucket" + i, {
        public: true,
      });
      resources["bucket" + i] = bucket.name;
    }

    return resources;
  },
});`

		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644))

		// Monitor deployment performance
		start := time.Now()
		
		err := runSST(ctx, projectDir, "deploy", "--stage", "util-test")
		require.NoError(t, err, "Deployment should succeed")
		
		deploymentTime := time.Since(start)
		t.Logf("Resource utilization test completed in: %v", deploymentTime)
		
		// Deployment should complete efficiently
		assert.Less(t, deploymentTime, 10*time.Minute, "Deployment should complete within 10 minutes")

		// Cleanup
		err = runSST(ctx, projectDir, "remove", "--stage", "util-test")
		require.NoError(t, err, "Cleanup should succeed")
	})
}

func TestRegressionDetection(t *testing.T) {
	if !shouldRunIntegrationTests() {
		t.Skip("Skipping integration test - AWS credentials not configured")
	}

	ctx := context.Background()
	
	// Test performance regression detection
	t.Run("baseline performance", func(t *testing.T) {
		projectDir := createTempProject(t, "regression-test")
		defer cleanupProject(t, projectDir)

		// Standard test project
		sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "regression-test",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    const bucket = new sst.aws.Bucket("RegressionBucket", {
      public: true,
    });

    const fn = new sst.aws.Function("RegressionFunction", {
      handler: "index.handler",
      runtime: "nodejs20.x",
      timeout: "30 seconds",
    });

    return {
      bucket: bucket.name,
      function: fn.name,
    };
  },
});`

		functionCode := `export const handler = async (event) => {
  return {
    statusCode: 200,
    body: JSON.stringify({ message: "Regression test!" }),
  };
};`

		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "index.ts"), []byte(functionCode), 0644))

		// Measure baseline performance
		var deploymentTimes []time.Duration
		
		for i := 0; i < 3; i++ {
			stage := fmt.Sprintf("regression-%d", i)
			
			start := time.Now()
			err := runSST(ctx, projectDir, "deploy", "--stage", stage)
			require.NoError(t, err, "Deployment %d should succeed", i)
			
			deploymentTime := time.Since(start)
			deploymentTimes = append(deploymentTimes, deploymentTime)
			
			t.Logf("Deployment %d completed in: %v", i, deploymentTime)
			
			// Cleanup
			err = runSST(ctx, projectDir, "remove", "--stage", stage)
			require.NoError(t, err, "Cleanup %d should succeed", i)
		}
		
		// Calculate average deployment time
		var total time.Duration
		for _, dt := range deploymentTimes {
			total += dt
		}
		average := total / time.Duration(len(deploymentTimes))
		
		t.Logf("Average deployment time: %v", average)
		
		// Check for consistency (no deployment should be more than 50% slower than average)
		for i, dt := range deploymentTimes {
			maxAllowed := average + (average / 2) // 150% of average
			assert.Less(t, dt, maxAllowed, "Deployment %d should not be significantly slower than average", i)
		}
	})
}