package examples

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sst/sst/v3/test/integration/pulumi/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudflareWorkerExample(t *testing.T) {
	defer helpers.CleanupTestArtifacts()

	// Skip if Cloudflare credentials not configured
	apiToken := os.Getenv("SST_TEST_CLOUDFLARE_API_TOKEN")
	accountID := os.Getenv("SST_TEST_CLOUDFLARE_ACCOUNT_ID")
	if apiToken == "" || accountID == "" {
		t.Skip("Skipping Cloudflare Worker example test: SST_TEST_CLOUDFLARE_API_TOKEN and SST_TEST_CLOUDFLARE_ACCOUNT_ID not set")
	}

	// Create test project directory
	projectDir := t.TempDir()
	defer helpers.CleanupTestProject(projectDir)

	// Create a simple Cloudflare Worker example project
	createCloudflareWorkerExample(t, projectDir)

	// Set up environment variables
	os.Setenv("CLOUDFLARE_API_TOKEN", apiToken)
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", accountID)

	// Deploy the project
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	
	stackName := fmt.Sprintf("cloudflare-worker-example-%d", time.Now().Unix())
	err := helpers.DeployProject(ctx, projectDir, stackName)
	require.NoError(t, err, "Failed to deploy Cloudflare Worker example")

	// Clean up after test
	defer func() {
		err := helpers.RemoveProject(ctx, projectDir, stackName)
		if err != nil {
			t.Logf("Warning: Failed to clean up project: %v", err)
		}
	}()

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, stackName)
	require.NoError(t, err, "Failed to get stack outputs")

	// Validate worker URL exists
	workerURL, exists := outputs["url"]
	require.True(t, exists, "Worker URL not found in outputs")
	require.NotEmpty(t, workerURL, "Worker URL is empty")

	// Test worker endpoint
	t.Run("Worker Health Check", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/health", workerURL))
		require.NoError(t, err, "Failed to call worker health endpoint")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health check should return 200")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		assert.Contains(t, string(body), "healthy", "Health check should return healthy status")
	})

	// Test worker API endpoint
	t.Run("Worker API Endpoint", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/hello", workerURL))
		require.NoError(t, err, "Failed to call worker API endpoint")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "API endpoint should return 200")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		assert.Contains(t, string(body), "Hello", "API should return greeting")
	})

	// Test worker default route
	t.Run("Worker Default Route", func(t *testing.T) {
		resp, err := http.Get(workerURL.(string))
		require.NoError(t, err, "Failed to call worker default route")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Default route should return 200")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		assert.Contains(t, string(body), "Cloudflare Worker", "Default route should identify as Cloudflare Worker")
	})
}

func TestCloudflareWorkerExampleUpdate(t *testing.T) {
	defer helpers.CleanupTestArtifacts()

	// Skip if Cloudflare credentials not configured
	apiToken := os.Getenv("SST_TEST_CLOUDFLARE_API_TOKEN")
	accountID := os.Getenv("SST_TEST_CLOUDFLARE_ACCOUNT_ID")
	if apiToken == "" || accountID == "" {
		t.Skip("Skipping Cloudflare Worker example update test: SST_TEST_CLOUDFLARE_API_TOKEN and SST_TEST_CLOUDFLARE_ACCOUNT_ID not set")
	}

	// Create test project directory
	projectDir := t.TempDir()
	defer helpers.CleanupTestProject(projectDir)

	// Create initial Cloudflare Worker example project
	createCloudflareWorkerExample(t, projectDir)

	// Set up environment variables
	os.Setenv("CLOUDFLARE_API_TOKEN", apiToken)
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", accountID)

	// Deploy the initial project
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	
	stackName := fmt.Sprintf("cloudflare-worker-example-update-%d", time.Now().Unix())
	err := helpers.DeployProject(ctx, projectDir, stackName)
	require.NoError(t, err, "Failed to deploy initial Cloudflare Worker example")

	// Clean up after test
	defer func() {
		err := helpers.RemoveProject(ctx, projectDir, stackName)
		if err != nil {
			t.Logf("Warning: Failed to clean up project: %v", err)
		}
	}()

	// Update the worker code
	updateCloudflareWorkerExample(t, projectDir)

	// Deploy the updated project
	err = helpers.DeployProject(ctx, projectDir, stackName)
	require.NoError(t, err, "Failed to deploy updated Cloudflare Worker example")

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, stackName)
	require.NoError(t, err, "Failed to get stack outputs")

	// Validate worker URL exists
	workerURL, exists := outputs["url"]
	require.True(t, exists, "Worker URL not found in outputs")
	require.NotEmpty(t, workerURL, "Worker URL is empty")

	// Test updated worker endpoint
	t.Run("Updated Worker API", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/hello", workerURL))
		require.NoError(t, err, "Failed to call updated worker API endpoint")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Updated API endpoint should return 200")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		assert.Contains(t, string(body), "Updated", "API should return updated greeting")
	})

	// Test new endpoint added in update
	t.Run("New Worker Endpoint", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/version", workerURL))
		require.NoError(t, err, "Failed to call new worker endpoint")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "New endpoint should return 200")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		assert.Contains(t, string(body), "v2", "New endpoint should return version info")
	})
}

func TestCloudflareWorkerExampleRollback(t *testing.T) {
	defer helpers.CleanupTestArtifacts()

	// Skip if Cloudflare credentials not configured
	apiToken := os.Getenv("SST_TEST_CLOUDFLARE_API_TOKEN")
	accountID := os.Getenv("SST_TEST_CLOUDFLARE_ACCOUNT_ID")
	if apiToken == "" || accountID == "" {
		t.Skip("Skipping Cloudflare Worker example rollback test: SST_TEST_CLOUDFLARE_API_TOKEN and SST_TEST_CLOUDFLARE_ACCOUNT_ID not set")
	}

	// Create test project directory
	projectDir := t.TempDir()
	defer helpers.CleanupTestProject(projectDir)

	// Create initial Cloudflare Worker example project
	createCloudflareWorkerExample(t, projectDir)

	// Set up environment variables
	os.Setenv("CLOUDFLARE_API_TOKEN", apiToken)
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", accountID)

	// Deploy the initial project
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	
	stackName := fmt.Sprintf("cloudflare-worker-example-rollback-%d", time.Now().Unix())
	err := helpers.DeployProject(ctx, projectDir, stackName)
	require.NoError(t, err, "Failed to deploy initial Cloudflare Worker example")

	// Clean up after test
	defer func() {
		err := helpers.RemoveProject(ctx, projectDir, stackName)
		if err != nil {
			t.Logf("Warning: Failed to clean up project: %v", err)
		}
	}()

	// Create a breaking change
	createBrokenCloudflareWorkerExample(t, projectDir)

	// Attempt to deploy the broken version (should fail or have issues)
	err = helpers.DeployProject(ctx, projectDir, stackName)
	// Note: This might succeed but the worker might not function correctly

	// Rollback to the original version
	createCloudflareWorkerExample(t, projectDir)

	// Deploy the rollback
	err = helpers.DeployProject(ctx, projectDir, stackName)
	require.NoError(t, err, "Failed to rollback Cloudflare Worker example")

	// Get stack outputs
	outputs, err := helpers.GetStackOutputs(ctx, projectDir, stackName)
	require.NoError(t, err, "Failed to get stack outputs")

	// Validate worker URL exists
	workerURL, exists := outputs["url"]
	require.True(t, exists, "Worker URL not found in outputs")
	require.NotEmpty(t, workerURL, "Worker URL is empty")

	// Test that rollback worked
	t.Run("Rollback Verification", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/health", workerURL))
		require.NoError(t, err, "Failed to call worker health endpoint after rollback")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health check should work after rollback")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		assert.Contains(t, string(body), "healthy", "Health check should return healthy status after rollback")
	})
}

// Helper function to create a basic Cloudflare Worker example project
func createCloudflareWorkerExample(t *testing.T, projectDir string) {
	// Create package.json
	packageJSON := `{
  "name": "cloudflare-worker-example",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "sst dev",
    "deploy": "sst deploy"
  },
  "dependencies": {
    "sst": "latest"
  }
}`
	err := os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "Failed to create package.json")

	// Create sst.config.ts
	sstConfig := `/// <reference path="./.sst/platform/config.d.ts" />

export default $config({
  app(input) {
    return {
      name: "cloudflare-worker-example",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "cloudflare",
    };
  },
  async run() {
    const worker = new sst.cloudflare.Worker("MyWorker", {
      handler: "./src/index.ts",
      url: true,
    });

    return {
      url: worker.url,
    };
  },
});`
	err = os.WriteFile(filepath.Join(projectDir, "sst.config.ts"), []byte(sstConfig), 0644)
	require.NoError(t, err, "Failed to create sst.config.ts")

	// Create src directory
	srcDir := filepath.Join(projectDir, "src")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err, "Failed to create src directory")

	// Create worker handler
	workerHandler := `export default {
  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url);
    
    // Health check endpoint
    if (url.pathname === '/health') {
      return new Response(JSON.stringify({ status: 'healthy', timestamp: new Date().toISOString() }), {
        headers: { 'Content-Type': 'application/json' },
      });
    }
    
    // API endpoint
    if (url.pathname === '/api/hello') {
      return new Response(JSON.stringify({ message: 'Hello from Cloudflare Worker!' }), {
        headers: { 'Content-Type': 'application/json' },
      });
    }
    
    // Default route
    return new Response(
      JSON.stringify({ 
        message: 'Cloudflare Worker Example',
        path: url.pathname,
        method: request.method,
        timestamp: new Date().toISOString()
      }),
      {
        headers: { 'Content-Type': 'application/json' },
      }
    );
  },
};`
	err = os.WriteFile(filepath.Join(srcDir, "index.ts"), []byte(workerHandler), 0644)
	require.NoError(t, err, "Failed to create worker handler")

	// Create tsconfig.json
	tsConfig := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true,
    "esModuleInterop": true,
    "allowJs": true,
    "strict": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "types": ["@cloudflare/workers-types"]
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules"]
}`
	err = os.WriteFile(filepath.Join(projectDir, "tsconfig.json"), []byte(tsConfig), 0644)
	require.NoError(t, err, "Failed to create tsconfig.json")
}

// Helper function to update the Cloudflare Worker example
func updateCloudflareWorkerExample(t *testing.T, projectDir string) {
	srcDir := filepath.Join(projectDir, "src")

	// Update worker handler with new functionality
	updatedWorkerHandler := `export default {
  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url);
    
    // Health check endpoint
    if (url.pathname === '/health') {
      return new Response(JSON.stringify({ status: 'healthy', timestamp: new Date().toISOString() }), {
        headers: { 'Content-Type': 'application/json' },
      });
    }
    
    // Updated API endpoint
    if (url.pathname === '/api/hello') {
      return new Response(JSON.stringify({ message: 'Updated Hello from Cloudflare Worker!' }), {
        headers: { 'Content-Type': 'application/json' },
      });
    }
    
    // New version endpoint
    if (url.pathname === '/api/version') {
      return new Response(JSON.stringify({ version: 'v2', updated: true }), {
        headers: { 'Content-Type': 'application/json' },
      });
    }
    
    // Default route
    return new Response(
      JSON.stringify({ 
        message: 'Updated Cloudflare Worker Example',
        path: url.pathname,
        method: request.method,
        timestamp: new Date().toISOString()
      }),
      {
        headers: { 'Content-Type': 'application/json' },
      }
    );
  },
};`
	err := os.WriteFile(filepath.Join(srcDir, "index.ts"), []byte(updatedWorkerHandler), 0644)
	require.NoError(t, err, "Failed to update worker handler")
}

// Helper function to create a broken Cloudflare Worker example
func createBrokenCloudflareWorkerExample(t *testing.T, projectDir string) {
	srcDir := filepath.Join(projectDir, "src")

	// Create broken worker handler (syntax error)
	brokenWorkerHandler := `export default {
  async fetch(request: Request): Promise<Response> {
    // This will cause a syntax error
    const url = new URL(request.url;
    
    return new Response('Broken worker', {
      status: 500,
      headers: { 'Content-Type': 'text/plain' },
    });
  },
};`
	err := os.WriteFile(filepath.Join(srcDir, "index.ts"), []byte(brokenWorkerHandler), 0644)
	require.NoError(t, err, "Failed to create broken worker handler")
}