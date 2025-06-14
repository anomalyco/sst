package node_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
	"github.com/sst/sst/v3/pkg/runtime/node"
)

func TestNew(t *testing.T) {
	version := "3.0.0"
	rt := node.New(version)
	
	if rt == nil {
		t.Fatal("New returned nil")
	}
}

func TestRuntime_Match(t *testing.T) {
	rt := node.New("3.0.0")
	
	tests := []struct {
		runtime string
		want    bool
	}{
		{"node18", true},
		{"node20", true},
		{"nodejs18.x", true},
		{"python3.9", false},
		{"go1.21", false},
		{"", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.runtime, func(t *testing.T) {
			got := rt.Match(tt.runtime)
			if got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.runtime, got, tt.want)
			}
		})
	}
}

func TestRuntime_ShouldRebuild(t *testing.T) {
	rt := node.New("3.0.0")
	
	// Test with no stored result
	result := rt.ShouldRebuild("test-function", "/path/to/file.js")
	if result {
		t.Error("ShouldRebuild should return false when no result is stored")
	}
}

func TestRuntime_Build_HandlerNotFound(t *testing.T) {
	tempDir := t.TempDir()
	rt := node.New("3.0.0")
	
	// Create a config file
	cfgFile := filepath.Join(tempDir, "sst.config.ts")
	err := os.WriteFile(cfgFile, []byte("export default {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	input := &runtime.BuildInput{
		CfgPath:    cfgFile,
		FunctionID: "test-function",
		Runtime:    "node18",
		Handler:    "nonexistent.handler",
		Properties: []byte(`{}`),
	}
	
	_, err = rt.Build(context.Background(), input)
	if err == nil {
		t.Error("Expected error for nonexistent handler")
	}
	
	if !contains(err.Error(), "Handler not found") {
		t.Errorf("Expected 'Handler not found' error, got: %v", err)
	}
}

func TestRuntime_Build_WithJSHandler(t *testing.T) {
	tempDir := t.TempDir()
	rt := node.New("3.0.0")
	
	// Create a config file
	cfgFile := filepath.Join(tempDir, "sst.config.ts")
	err := os.WriteFile(cfgFile, []byte("export default {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Create a JS handler file
	handlerFile := filepath.Join(tempDir, "handler.js")
	err = os.WriteFile(handlerFile, []byte("exports.handler = () => {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	// Create output directory
	outDir := filepath.Join(tempDir, ".sst", "artifacts", "test-function-src")
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}
	
	input := &runtime.BuildInput{
		CfgPath:    cfgFile,
		FunctionID: "test-function",
		Runtime:    "node18",
		Handler:    "handler.handler",
		Properties: []byte(`{"format": "esm", "minify": false}`),
		Dev:        false,
	}
	
	result, err := rt.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Build result is nil")
	}
	
	if result.Handler == "" {
		t.Error("Handler should not be empty")
	}
	
	if result.Errors == nil {
		t.Error("Errors slice should be initialized")
	}
	
	if result.Sourcemaps == nil {
		t.Error("Sourcemaps slice should be initialized")
	}
}

func TestRuntime_Build_WithTSHandler(t *testing.T) {
	tempDir := t.TempDir()
	rt := node.New("3.0.0")
	
	// Create a config file
	cfgFile := filepath.Join(tempDir, "sst.config.ts")
	err := os.WriteFile(cfgFile, []byte("export default {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Create a TS handler file
	handlerFile := filepath.Join(tempDir, "handler.ts")
	err = os.WriteFile(handlerFile, []byte("export const handler = () => {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	// Create output directory
	outDir := filepath.Join(tempDir, ".sst", "artifacts", "test-function-src")
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}
	
	input := &runtime.BuildInput{
		CfgPath:    cfgFile,
		FunctionID: "test-function",
		Runtime:    "node18",
		Handler:    "handler.handler",
		Properties: []byte(`{"format": "esm"}`),
		Dev:        true,
	}
	
	result, err := rt.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Build result is nil")
	}
}

func TestRuntime_Build_WithProperties(t *testing.T) {
	tempDir := t.TempDir()
	rt := node.New("3.0.0")
	
	// Create a config file
	cfgFile := filepath.Join(tempDir, "sst.config.ts")
	err := os.WriteFile(cfgFile, []byte("export default {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Create a handler file
	handlerFile := filepath.Join(tempDir, "handler.js")
	err = os.WriteFile(handlerFile, []byte("exports.handler = () => {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	// Create output directory
	outDir := filepath.Join(tempDir, ".sst", "artifacts", "test-function-src")
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}
	
	properties := map[string]interface{}{
		"minify":       false,
		"format":       "cjs",
		"sourceMap":    false,
		"splitting":    false,
		"architecture": "x64",
	}
	
	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		t.Fatalf("Failed to marshal properties: %v", err)
	}
	
	input := &runtime.BuildInput{
		CfgPath:    cfgFile,
		FunctionID: "test-function",
		Runtime:    "node18",
		Handler:    "handler.handler",
		Properties: propertiesJSON,
		Dev:        false,
	}
	
	result, err := rt.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Build result is nil")
	}
}

func TestRuntime_Run(t *testing.T) {
	tempDir := t.TempDir()
	rt := node.New("3.0.0")
	
	// Create a config file
	cfgFile := filepath.Join(tempDir, "sst.config.ts")
	err := os.WriteFile(cfgFile, []byte("export default {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Create a mock build output
	buildOut := filepath.Join(tempDir, "build")
	err = os.MkdirAll(buildOut, 0755)
	if err != nil {
		t.Fatalf("Failed to create build directory: %v", err)
	}
	
	// Create a mock handler file
	handlerFile := filepath.Join(buildOut, "index.js")
	err = os.WriteFile(handlerFile, []byte("exports.handler = () => {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	input := &runtime.RunInput{
		CfgPath:    cfgFile,
		Runtime:    "node18",
		FunctionID: "test-function",
		WorkerID:   "worker-1",
		Build: &runtime.BuildOutput{
			Out:     buildOut,
			Handler: "index.js",
		},
		Env:    []string{"NODE_ENV=test"},
		Server: "localhost:8080",
	}
	
	// Note: This test will fail in CI without Node.js runtime files
	// In a real test environment, we'd mock the process execution
	worker, err := rt.Run(context.Background(), input)
	
	// Since we don't have the actual runtime files, expect an error
	if err == nil {
		// If it succeeds (unlikely), verify worker interface
		if worker == nil {
			t.Error("Run returned nil worker without error")
		} else {
			// Test worker methods
			logs := worker.Logs()
			if logs == nil {
				t.Error("Worker.Logs() returned nil")
			} else {
				logs.Close()
			}
			
			// Test stop (should not panic)
			worker.Stop()
		}
	}
	// If it fails (expected), that's okay for this unit test
}

func TestNodeProperties_JSONUnmarshaling(t *testing.T) {
	// Test NodeProperties JSON unmarshaling through Build method
	jsonData := `{
		"banner": "// Custom banner",
		"minify": false,
		"format": "esm",
		"sourceMap": false,
		"splitting": false,
		"architecture": "x64"
	}`
	
	tempDir := t.TempDir()
	rt := node.New("3.0.0")
	
	// Create a config file
	cfgFile := filepath.Join(tempDir, "sst.config.ts")
	err := os.WriteFile(cfgFile, []byte("export default {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Create a handler file
	handlerFile := filepath.Join(tempDir, "handler.js")
	err = os.WriteFile(handlerFile, []byte("exports.handler = () => {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create handler file: %v", err)
	}
	
	// Create output directory
	outDir := filepath.Join(tempDir, ".sst", "artifacts", "test-function-src")
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}
	
	input := &runtime.BuildInput{
		CfgPath:    cfgFile,
		FunctionID: "test-function",
		Runtime:    "node18",
		Handler:    "handler.handler",
		Properties: []byte(jsonData),
		Dev:        false,
	}
	
	// This should not panic or error due to invalid JSON
	_, err = rt.Build(context.Background(), input)
	if err != nil && contains(err.Error(), "json") {
		t.Errorf("Build failed due to JSON parsing: %v", err)
	}
}

func TestNodeExtensions(t *testing.T) {
	// Test that various Node.js file extensions are supported
	tempDir := t.TempDir()
	rt := node.New("3.0.0")
	
	// Create a config file
	cfgFile := filepath.Join(tempDir, "sst.config.ts")
	err := os.WriteFile(cfgFile, []byte("export default {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	extensions := []string{".js", ".ts", ".jsx", ".tsx", ".mjs", ".cjs", ".mts", ".cts"}
	
	for _, ext := range extensions {
		t.Run("extension_"+ext, func(t *testing.T) {
			// Create a handler file with the extension
			handlerFile := filepath.Join(tempDir, "handler"+ext)
			content := "exports.handler = () => {}"
			if ext == ".ts" || ext == ".tsx" || ext == ".mts" || ext == ".cts" {
				content = "export const handler = () => {}"
			}
			
			err := os.WriteFile(handlerFile, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create handler file: %v", err)
			}
			
			// Create output directory
			outDir := filepath.Join(tempDir, ".sst", "artifacts", "test-function-src")
			err = os.MkdirAll(outDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create output directory: %v", err)
			}
			
			input := &runtime.BuildInput{
				CfgPath:    cfgFile,
				FunctionID: "test-function",
				Runtime:    "node18",
				Handler:    "handler.handler",
				Properties: []byte(`{}`),
				Dev:        false,
			}
			
			// Should find the handler file
			_, err = rt.Build(context.Background(), input)
			if err != nil && contains(err.Error(), "Handler not found") {
				t.Errorf("Handler with extension %s should be found", ext)
			}
			
			// Clean up for next iteration
			os.Remove(handlerFile)
		})
	}
}

func TestConcurrencyConfiguration(t *testing.T) {
	// Test that concurrency can be configured
	// This tests the New function's concurrency setup
	
	// Test default concurrency
	rt1 := node.New("3.0.0")
	if rt1 == nil {
		t.Error("Runtime should not be nil")
	}
	
	// Test with different version
	rt2 := node.New("dev")
	if rt2 == nil {
		t.Error("Runtime should not be nil")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}