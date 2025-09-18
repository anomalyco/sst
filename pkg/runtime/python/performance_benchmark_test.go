package python

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sst/sst/v3/pkg/runtime"
)

// BenchmarkBuildPipeline benchmarks the build pipeline performance
func BenchmarkBuildPipeline(b *testing.B) {
	benchmarks := []struct {
		name         string
		setupProject func(b *testing.B, projectDir string) string
		handler      string
	}{
		{
			name: "flat_project",
			setupProject: func(b *testing.B, projectDir string) string {
				handlerContent := `
import json

def lambda_handler(event, context):
    return {'statusCode': 200, 'body': json.dumps({'message': 'hello'})}
`
				handlerPath := filepath.Join(projectDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					b.Fatalf("Failed to create handler: %v", err)
				}

				requirementsContent := "requests>=2.31.0\nboto3>=1.34.0\n"
				requirementsPath := filepath.Join(projectDir, "requirements.txt")
				if err := os.WriteFile(requirementsPath, []byte(requirementsContent), 0644); err != nil {
					b.Fatalf("Failed to create requirements.txt: %v", err)
				}

				return "handler.py"
			},
			handler: "handler.lambda_handler",
		},
		{
			name: "workspace_project",
			setupProject: func(b *testing.B, projectDir string) string {
				pyprojectContent := `
[project]
name = "benchmark-workspace"
version = "0.1.0"
dependencies = ["requests>=2.31.0", "boto3>=1.34.0"]
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					b.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				functionsDir := filepath.Join(projectDir, "functions")
				if err := os.MkdirAll(functionsDir, 0755); err != nil {
					b.Fatalf("Failed to create functions directory: %v", err)
				}

				handlerContent := `
import json
import requests
import boto3

def lambda_handler(event, context):
    return {'statusCode': 200, 'body': json.dumps({'message': 'hello'})}
`
				handlerPath := filepath.Join(functionsDir, "api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					b.Fatalf("Failed to create handler: %v", err)
				}

				initPath := filepath.Join(functionsDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					b.Fatalf("Failed to create __init__.py: %v", err)
				}

				return "api.py"
			},
			handler: "functions.api.lambda_handler",
		},
		{
			name: "nested_project",
			setupProject: func(b *testing.B, projectDir string) string {
				pyprojectContent := `
[project]
name = "benchmark-nested"
version = "0.1.0"
dependencies = ["requests>=2.31.0", "boto3>=1.34.0"]
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					b.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				packageDir := filepath.Join(projectDir, "src", "mypackage", "handlers")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					b.Fatalf("Failed to create package directory: %v", err)
				}

				initPaths := []string{
					filepath.Join(projectDir, "src", "mypackage", "__init__.py"),
					filepath.Join(packageDir, "__init__.py"),
				}
				for _, initPath := range initPaths {
					if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
						b.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				handlerContent := `
import json
import requests
import boto3

def lambda_handler(event, context):
    return {'statusCode': 200, 'body': json.dumps({'message': 'hello'})}
`
				handlerPath := filepath.Join(packageDir, "api.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					b.Fatalf("Failed to create handler: %v", err)
				}

				return "handlers/api.py"
			},
			handler: "mypackage.handlers.api.lambda_handler",
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Setup once
			tempDir := b.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				b.Fatalf("Failed to create project directory: %v", err)
			}

			_ = bm.setupProject(b, projectDir)

			pipeline, err := NewBuildPipeline(BuildPipelineConfig{
				ProjectRoot:             projectDir,
				ArtifactDir:             filepath.Join(tempDir, "artifacts"),
				EnableCaching:           false, // Disable caching for pure build performance
				EnableProgressReporting: false,
			})
			if err != nil {
				b.Fatalf("Failed to create build pipeline: %v", err)
			}

			input := createBenchmarkBuildInput(b, tempDir, bm.handler)
			ctx := context.Background()

			// Reset timer after setup
			b.ResetTimer()

			// Run benchmark
			for i := 0; i < b.N; i++ {
				// Note: Each build uses the same output directory for benchmarking purposes

				_, err := pipeline.Build(ctx, input)
				if err != nil {
					b.Fatalf("Build failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkProjectResolver benchmarks project resolution performance
func BenchmarkProjectResolver(b *testing.B) {
	// Create a complex project structure
	tempDir := b.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		b.Fatalf("Failed to create project directory: %v", err)
	}

	// Create pyproject.toml
	pyprojectContent := `
[project]
name = "benchmark-project"
version = "0.1.0"
dependencies = ["requests>=2.31.0", "boto3>=1.34.0", "fastapi>=0.104.0"]
`
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		b.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create multiple handler files in different locations
	handlers := []struct {
		path    string
		content string
	}{
		{
			path: "handler.py",
			content: `
def lambda_handler(event, context):
    return {'statusCode': 200}
`,
		},
		{
			path: "src/mypackage/handler.py",
			content: `
def lambda_handler(event, context):
    return {'statusCode': 200}
`,
		},
		{
			path: "functions/api.py",
			content: `
def lambda_handler(event, context):
    return {'statusCode': 200}
`,
		},
		{
			path: "app/services/handlers/users.py",
			content: `
def lambda_handler(event, context):
    return {'statusCode': 200}
`,
		},
	}

	for _, handler := range handlers {
		handlerPath := filepath.Join(projectDir, handler.path)
		handlerDir := filepath.Dir(handlerPath)
		if err := os.MkdirAll(handlerDir, 0755); err != nil {
			b.Fatalf("Failed to create handler directory: %v", err)
		}
		if err := os.WriteFile(handlerPath, []byte(handler.content), 0644); err != nil {
			b.Fatalf("Failed to create handler file: %v", err)
		}

		// Create __init__.py files for packages
		if filepath.Dir(handler.path) != "." {
			parts := filepath.SplitList(filepath.Dir(handler.path))
			currentPath := projectDir
			for _, part := range parts {
				currentPath = filepath.Join(currentPath, part)
				initPath := filepath.Join(currentPath, "__init__.py")
				if _, err := os.Stat(initPath); os.IsNotExist(err) {
					if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
						b.Fatalf("Failed to create __init__.py: %v", err)
					}
				}
			}
		}
	}

	resolver := NewProjectResolver(projectDir)

	b.ResetTimer()

	// Benchmark resolution of different handlers
	for i := 0; i < b.N; i++ {
		for _, handler := range handlers {
			_, err := resolver.ResolveHandler(handler.path)
			if err != nil {
				b.Fatalf("Failed to resolve handler %s: %v", handler.path, err)
			}
		}
	}
}

// BenchmarkBuildCache benchmarks build cache performance
func BenchmarkBuildCache(b *testing.B) {
	tempDir := b.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: cacheDir,
		MaxSize:  100,
	})
	if err != nil {
		b.Fatalf("Failed to create build cache: %v", err)
	}

	// Create sample cache entries
	entries := make([]*CacheEntry, 100)
	for i := 0; i < 100; i++ {
		functionID := fmt.Sprintf("function-%d", i)
		entries[i] = &CacheEntry{
			FunctionID:   functionID,
			BuildTime:    time.Now(),
			FileHashes:   map[string]string{"handler.py": fmt.Sprintf("hash%d", i)},
			Dependencies: []string{"handler.py", "pyproject.toml"},
		}
	}

	b.ResetTimer()

	b.Run("cache_operations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			entry := entries[i%len(entries)]

			// Store entry
			if err := cache.Set(entry.FunctionID, entry); err != nil {
				b.Fatalf("Failed to store cache entry: %v", err)
			}

			// Retrieve entry
			_, exists := cache.Get(entry.FunctionID)
			if !exists {
				b.Fatalf("Failed to get cache entry")
			}

			// Check validity
			valid, err := cache.IsValid(entry)
			if err != nil {
				b.Fatalf("Failed to check cache validity: %v", err)
			}
			_ = valid // Use the variable to avoid unused variable error
		}
	})
}

// BenchmarkContentHashing benchmarks file content hashing performance
func BenchmarkContentHashing(b *testing.B) {
	// Create test files of different sizes
	tempDir := b.TempDir()

	testFiles := []struct {
		name string
		size int
	}{
		{"small.py", 1024},       // 1KB
		{"medium.py", 10 * 1024}, // 10KB
		{"large.py", 100 * 1024}, // 100KB
	}

	for _, tf := range testFiles {
		content := make([]byte, tf.size)
		for i := range content {
			content[i] = byte('a' + (i % 26))
		}

		filePath := filepath.Join(tempDir, tf.name)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: filepath.Join(tempDir, "cache"),
		MaxSize:  100,
	})
	if err != nil {
		b.Fatalf("Failed to create build cache: %v", err)
	}

	b.ResetTimer()

	for _, tf := range testFiles {
		b.Run("hash_"+tf.name, func(b *testing.B) {
			filePath := filepath.Join(tempDir, tf.name)
			for i := 0; i < b.N; i++ {
				_, err := cache.calculateFileHash(filePath)
				if err != nil {
					b.Fatalf("Failed to hash file content: %v", err)
				}
			}
		})
	}
}

// BenchmarkDependencyAnalysis benchmarks dependency analysis performance
func BenchmarkDependencyAnalysis(b *testing.B) {
	tempDir := b.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		b.Fatalf("Failed to create project directory: %v", err)
	}

	// Create a project with many dependencies
	pyprojectContent := `
[project]
name = "benchmark-deps"
version = "0.1.0"
dependencies = [
    "requests>=2.31.0",
    "boto3>=1.34.0",
    "fastapi>=0.104.0",
    "pydantic>=2.5.0",
    "sqlalchemy>=2.0.0",
    "alembic>=1.13.0",
    "pytest>=7.4.0",
    "pytest-asyncio>=0.21.0",
    "httpx>=0.25.0",
    "uvicorn>=0.24.0"
]
`
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		b.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create handler with imports
	handlerContent := `
import json
import os
import sys
from datetime import datetime
from typing import Dict, List, Optional

def lambda_handler(event, context):
    return {'statusCode': 200, 'body': json.dumps({'message': 'hello'})}
`
	handlerPath := filepath.Join(projectDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		b.Fatalf("Failed to create handler: %v", err)
	}

	resolver := NewProjectResolver(projectDir)
	analyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
		ProjectResolver: resolver,
	})

	projectInfo, err := resolver.ResolveHandler("handler.py")
	if err != nil {
		b.Fatalf("Failed to resolve handler: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeDependencies(ctx, projectInfo)
		if err != nil {
			b.Fatalf("Failed to analyze dependencies: %v", err)
		}
	}
}

// createBenchmarkBuildInput creates a build input for benchmarking
func createBenchmarkBuildInput(b *testing.B, tempDir, handler string) *runtime.BuildInput {
	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		b.Fatalf("Failed to create output directory: %v", err)
	}

	properties := map[string]interface{}{
		"architecture": "x86_64",
		"container":    false,
	}

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		b.Fatalf("Failed to marshal properties: %v", err)
	}

	return &runtime.BuildInput{
		FunctionID: "benchmark-function",
		Handler:    handler,
		Runtime:    "python3.12",
		Properties: propertiesJSON,
	}
}

// BenchmarkCacheHitRates benchmarks cache hit rates and invalidation behavior
func BenchmarkCacheHitRates(b *testing.B) {
	tempDir := b.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := NewBuildCache(BuildCacheConfig{
		CacheDir: cacheDir,
		MaxSize:  100,
	})
	if err != nil {
		b.Fatalf("Failed to create build cache: %v", err)
	}

	// Pre-populate cache with entries
	for i := 0; i < 50; i++ {
		functionID := fmt.Sprintf("function-%d", i)
		entry := &CacheEntry{
			FunctionID:   functionID,
			BuildTime:    time.Now(),
			FileHashes:   map[string]string{"handler.py": fmt.Sprintf("hash%d", i)},
			Dependencies: []string{"handler.py"},
		}
		cache.Set(functionID, entry)
	}

	b.ResetTimer()

	b.Run("cache_hit_rate", func(b *testing.B) {
		hitCount := 0
		for i := 0; i < b.N; i++ {
			functionID := fmt.Sprintf("function-%d", i%50) // 50% hit rate expected
			_, exists := cache.Get(functionID)
			if exists {
				hitCount++
			}
		}

		hitRate := float64(hitCount) / float64(b.N)
		b.ReportMetric(hitRate*100, "hit_rate_%")
	})

	b.Run("cache_invalidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			functionID := fmt.Sprintf("function-%d", i%50)
			entry, exists := cache.Get(functionID)
			if exists {
				// Simulate file change by updating hash
				entry.FileHashes["handler.py"] = fmt.Sprintf("newhash%d", i)
				valid, _ := cache.IsValid(entry)
				_ = valid // Use the variable
			}
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage during builds
func BenchmarkMemoryUsage(b *testing.B) {
	tempDir := b.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		b.Fatalf("Failed to create project directory: %v", err)
	}

	// Create a large project with many files
	pyprojectContent := `
[project]
name = "memory-test"
version = "0.1.0"
dependencies = ["requests>=2.31.0", "boto3>=1.34.0"]
`
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		b.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	// Create multiple handler files to simulate memory usage
	for i := 0; i < 100; i++ {
		handlerContent := fmt.Sprintf(`
import json
import requests
import boto3
from datetime import datetime

# Handler %d with unique content
def lambda_handler_%d(event, context):
    data = {"handler": %d, "timestamp": datetime.now().isoformat()}
    return {'statusCode': 200, 'body': json.dumps(data)}
`, i, i, i)

		handlerPath := filepath.Join(projectDir, fmt.Sprintf("handler_%d.py", i))
		if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
			b.Fatalf("Failed to create handler %d: %v", i, err)
		}
	}

	pipeline, err := NewBuildPipeline(BuildPipelineConfig{
		ProjectRoot:             projectDir,
		ArtifactDir:             filepath.Join(tempDir, "artifacts"),
		EnableCaching:           true,
		EnableProgressReporting: false,
	})
	if err != nil {
		b.Fatalf("Failed to create build pipeline: %v", err)
	}

	input := createBenchmarkBuildInput(b, tempDir, "handler_0.lambda_handler_0")
	ctx := context.Background()

	b.ResetTimer()

	// Measure memory usage during builds
	for i := 0; i < b.N; i++ {
		// Use different handlers to test memory usage patterns
		handlerNum := i % 100
		input.Handler = fmt.Sprintf("handler_%d.lambda_handler_%d", handlerNum, handlerNum)
		input.FunctionID = fmt.Sprintf("memory-test-function-%d", handlerNum)

		_, err := pipeline.Build(ctx, input)
		if err != nil {
			b.Fatalf("Build failed: %v", err)
		}
	}
}

// BenchmarkFileResolutionOptimization benchmarks file resolution performance optimizations
func BenchmarkFileResolutionOptimization(b *testing.B) {
	tempDir := b.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		b.Fatalf("Failed to create project directory: %v", err)
	}

	// Create a complex directory structure
	directories := []string{
		"src/package1/subpackage1",
		"src/package1/subpackage2",
		"src/package2/handlers",
		"functions/api",
		"functions/workers",
		"app/services/auth",
		"app/services/data",
		"lib/utils",
		"lib/helpers",
	}

	for _, dir := range directories {
		fullDir := filepath.Join(projectDir, dir)
		if err := os.MkdirAll(fullDir, 0755); err != nil {
			b.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Create handler files in each directory
		handlerContent := fmt.Sprintf(`
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Handler from %s'}
`, dir)

		handlerPath := filepath.Join(fullDir, "handler.py")
		if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
			b.Fatalf("Failed to create handler in %s: %v", dir, err)
		}

		// Create __init__.py files
		initPath := filepath.Join(fullDir, "__init__.py")
		if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
			b.Fatalf("Failed to create __init__.py in %s: %v", dir, err)
		}
	}

	resolver := NewProjectResolver(projectDir)

	b.ResetTimer()

	b.Run("sequential_resolution", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			dir := directories[i%len(directories)]
			handlerPath := filepath.Join(dir, "handler.py")

			_, err := resolver.ResolveHandler(handlerPath)
			if err != nil {
				b.Fatalf("Failed to resolve handler %s: %v", handlerPath, err)
			}
		}
	})

	b.Run("cached_resolution", func(b *testing.B) {
		// Pre-warm cache
		for _, dir := range directories {
			handlerPath := filepath.Join(dir, "handler.py")
			resolver.ResolveHandler(handlerPath)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			dir := directories[i%len(directories)]
			handlerPath := filepath.Join(dir, "handler.py")

			_, err := resolver.ResolveHandler(handlerPath)
			if err != nil {
				b.Fatalf("Failed to resolve cached handler %s: %v", handlerPath, err)
			}
		}
	})
}

// BenchmarkBuildTimeComparison benchmarks build times before and after optimizations
func BenchmarkBuildTimeComparison(b *testing.B) {
	scenarios := []struct {
		name           string
		enableCaching  bool
		enableProgress bool
		projectSize    string
	}{
		{"small_no_cache", false, false, "small"},
		{"small_with_cache", true, false, "small"},
		{"medium_no_cache", false, false, "medium"},
		{"medium_with_cache", true, false, "medium"},
		{"large_no_cache", false, false, "large"},
		{"large_with_cache", true, false, "large"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			tempDir := b.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				b.Fatalf("Failed to create project directory: %v", err)
			}

			// Create project based on size
			createTestProject(b, projectDir, scenario.projectSize)

			pipeline, err := NewBuildPipeline(BuildPipelineConfig{
				ProjectRoot:             projectDir,
				ArtifactDir:             filepath.Join(tempDir, "artifacts"),
				EnableCaching:           scenario.enableCaching,
				EnableProgressReporting: scenario.enableProgress,
			})
			if err != nil {
				b.Fatalf("Failed to create build pipeline: %v", err)
			}

			input := createBenchmarkBuildInput(b, tempDir, "handler.lambda_handler")
			ctx := context.Background()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				input.FunctionID = fmt.Sprintf("benchmark-function-%d", i)

				start := time.Now()
				_, err := pipeline.Build(ctx, input)
				if err != nil {
					b.Fatalf("Build failed: %v", err)
				}
				buildTime := time.Since(start)

				b.ReportMetric(float64(buildTime.Milliseconds()), "build_time_ms")
			}
		})
	}
}

// createTestProject creates a test project of specified size
func createTestProject(b *testing.B, projectDir, size string) {
	pyprojectContent := `
[project]
name = "test-project"
version = "0.1.0"
dependencies = ["requests>=2.31.0", "boto3>=1.34.0"]
`
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		b.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	var fileCount int
	switch size {
	case "small":
		fileCount = 5
	case "medium":
		fileCount = 25
	case "large":
		fileCount = 100
	default:
		fileCount = 5
	}

	// Create handler files
	for i := 0; i < fileCount; i++ {
		handlerContent := fmt.Sprintf(`
import json
import requests
import boto3
from datetime import datetime

def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': json.dumps({
            'message': 'Handler %d',
            'timestamp': datetime.now().isoformat()
        })
    }
`, i)

		if i == 0 {
			// Main handler
			handlerPath := filepath.Join(projectDir, "handler.py")
			if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
				b.Fatalf("Failed to create main handler: %v", err)
			}
		} else {
			// Additional handlers in subdirectories
			subDir := filepath.Join(projectDir, fmt.Sprintf("handlers_%d", i/10))
			if err := os.MkdirAll(subDir, 0755); err != nil {
				b.Fatalf("Failed to create subdirectory: %v", err)
			}

			handlerPath := filepath.Join(subDir, fmt.Sprintf("handler_%d.py", i))
			if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
				b.Fatalf("Failed to create handler %d: %v", i, err)
			}
		}
	}
}

// TestBenchmarkResults runs benchmarks and reports results for performance monitoring
func TestBenchmarkResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark results in short mode")
	}

	// Run benchmarks and capture results
	result := testing.Benchmark(BenchmarkBuildPipeline)
	t.Logf("Build Pipeline Benchmark: %s", result.String())

	result = testing.Benchmark(BenchmarkProjectResolver)
	t.Logf("Project Resolver Benchmark: %s", result.String())

	result = testing.Benchmark(BenchmarkBuildCache)
	t.Logf("Build Cache Benchmark: %s", result.String())

	result = testing.Benchmark(BenchmarkContentHashing)
	t.Logf("Content Hashing Benchmark: %s", result.String())

	result = testing.Benchmark(BenchmarkDependencyAnalysis)
	t.Logf("Dependency Analysis Benchmark: %s", result.String())

	result = testing.Benchmark(BenchmarkCacheHitRates)
	t.Logf("Cache Hit Rates Benchmark: %s", result.String())

	result = testing.Benchmark(BenchmarkMemoryUsage)
	t.Logf("Memory Usage Benchmark: %s", result.String())

	result = testing.Benchmark(BenchmarkFileResolutionOptimization)
	t.Logf("File Resolution Optimization Benchmark: %s", result.String())

	result = testing.Benchmark(BenchmarkBuildTimeComparison)
	t.Logf("Build Time Comparison Benchmark: %s", result.String())
}

// TestPerformanceRegression runs performance regression tests
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression tests in short mode")
	}

	// Define performance thresholds
	thresholds := map[string]time.Duration{
		"project_resolution":  100 * time.Millisecond,
		"cache_operations":    10 * time.Millisecond,
		"file_hashing":        50 * time.Millisecond,
		"dependency_analysis": 500 * time.Millisecond,
	}

	// Test project resolution performance
	t.Run("project_resolution_performance", func(t *testing.T) {
		tempDir := t.TempDir()
		projectDir := filepath.Join(tempDir, "project")
		createTestProjectForTesting(t, projectDir, "medium")

		resolver := NewProjectResolver(projectDir)

		start := time.Now()
		_, err := resolver.ResolveHandler("handler.py")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to resolve handler: %v", err)
		}

		if duration > thresholds["project_resolution"] {
			t.Errorf("Project resolution took %v, expected less than %v", duration, thresholds["project_resolution"])
		}

		t.Logf("Project resolution completed in %v", duration)
	})

	// Test cache operation performance
	t.Run("cache_operations_performance", func(t *testing.T) {
		tempDir := t.TempDir()
		cache, err := NewBuildCache(BuildCacheConfig{
			CacheDir: filepath.Join(tempDir, "cache"),
			MaxSize:  100,
		})
		if err != nil {
			t.Fatalf("Failed to create cache: %v", err)
		}

		entry := &CacheEntry{
			FunctionID:   "test-function",
			BuildTime:    time.Now(),
			FileHashes:   map[string]string{"handler.py": "testhash"},
			Dependencies: []string{"handler.py"},
		}

		start := time.Now()
		err = cache.Set("test-function", entry)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to set cache entry: %v", err)
		}

		if duration > thresholds["cache_operations"] {
			t.Errorf("Cache set operation took %v, expected less than %v", duration, thresholds["cache_operations"])
		}

		start = time.Now()
		_, exists := cache.Get("test-function")
		duration = time.Since(start)

		if !exists {
			t.Fatalf("Cache entry not found")
		}

		if duration > thresholds["cache_operations"] {
			t.Errorf("Cache get operation took %v, expected less than %v", duration, thresholds["cache_operations"])
		}

		t.Logf("Cache operations completed in acceptable time")
	})
}

// createTestProjectForTesting creates a test project for performance testing
func createTestProjectForTesting(t *testing.T, projectDir, size string) {
	// Create the project directory first
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	pyprojectContent := `
[project]
name = "test-project"
version = "0.1.0"
dependencies = ["requests>=2.31.0"]
`
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatalf("Failed to create pyproject.toml: %v", err)
	}

	handlerContent := `
import json

def lambda_handler(event, context):
    return {'statusCode': 200, 'body': json.dumps({'message': 'test'})}
`
	handlerPath := filepath.Join(projectDir, "handler.py")
	if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
}
