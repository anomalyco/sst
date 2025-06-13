package runtime_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
)

// MockRuntime implements the Runtime interface for testing
type MockRuntime struct {
	matchPattern string
	buildResult  *runtime.BuildOutput
	buildError   error
	worker       *MockWorker
	runError     error
	shouldRebuild bool
}

func (m *MockRuntime) Match(runtimeStr string) bool {
	return strings.Contains(runtimeStr, m.matchPattern)
}

func (m *MockRuntime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	if m.buildError != nil {
		return nil, m.buildError
	}
	return m.buildResult, nil
}

func (m *MockRuntime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	if m.runError != nil {
		return nil, m.runError
	}
	return m.worker, nil
}

func (m *MockRuntime) ShouldRebuild(functionID string, path string) bool {
	return m.shouldRebuild
}

// MockWorker implements the Worker interface for testing
type MockWorker struct {
	logs io.ReadCloser
}

func (m *MockWorker) Stop() {
	// Mock implementation
}

func (m *MockWorker) Logs() io.ReadCloser {
	return m.logs
}

func TestNewCollection(t *testing.T) {
	mockRuntime := &MockRuntime{matchPattern: "node"}
	collection := runtime.NewCollection("test-platform", mockRuntime)

	if collection == nil {
		t.Fatal("NewCollection returned nil")
	}

	// Test runtime matching
	foundRuntime, ok := collection.Runtime("node18")
	if !ok {
		t.Error("Expected to find node runtime")
	}
	if foundRuntime != mockRuntime {
		t.Error("Expected to get the same mock runtime instance")
	}

	// Test runtime not found
	_, ok = collection.Runtime("unknown")
	if ok {
		t.Error("Expected not to find unknown runtime")
	}
}

func TestBuildInput_Out(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "sst.config.ts")
	
	input := &runtime.BuildInput{
		CfgPath:    cfgPath,
		FunctionID: "test-function",
		Dev:        false,
	}

	out := input.Out()
	expected := filepath.Join(tempDir, ".sst", "artifacts", "test-function-src")
	if out != expected {
		t.Errorf("Expected out path %s, got %s", expected, out)
	}

	// Test dev mode
	input.Dev = true
	out = input.Out()
	expected = filepath.Join(tempDir, ".sst", "artifacts", "test-function-dev")
	if out != expected {
		t.Errorf("Expected dev out path %s, got %s", expected, out)
	}
}

func TestCollection_Build(t *testing.T) {
	tempDir := t.TempDir()
	
	mockRuntime := &MockRuntime{
		matchPattern: "node",
		buildResult: &runtime.BuildOutput{
			Handler:    "index.handler",
			Errors:     []string{},
			Sourcemaps: []string{},
		},
	}
	
	collection := runtime.NewCollection(tempDir, mockRuntime)
	
	input := &runtime.BuildInput{
		CfgPath:    tempDir,
		FunctionID: "test-function",
		Runtime:    "node18",
		Handler:    "index.handler",
		Dev:        false,
	}

	result, err := collection.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if result.Handler != "index.handler" {
		t.Errorf("Expected handler 'index.handler', got '%s'", result.Handler)
	}

	if result.Out == "" {
		t.Error("Expected out path to be set")
	}

	if result.Sourcemaps == nil {
		t.Error("Expected sourcemaps to be initialized")
	}
}

func TestCollection_Build_WithBundle(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "bundle")
	
	collection := runtime.NewCollection(tempDir)
	
	input := &runtime.BuildInput{
		CfgPath:    tempDir,
		FunctionID: "test-function",
		Runtime:    "node18",
		Handler:    "index.handler",
		Bundle:     bundlePath,
	}

	result, err := collection.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("Build with bundle failed: %v", err)
	}

	if result.Out != bundlePath {
		t.Errorf("Expected out path to be bundle path %s, got %s", bundlePath, result.Out)
	}
}

func TestCollection_Build_RuntimeNotFound(t *testing.T) {
	tempDir := t.TempDir()
	
	collection := runtime.NewCollection(tempDir)
	
	input := &runtime.BuildInput{
		CfgPath:    tempDir,
		FunctionID: "test-function",
		Runtime:    "unknown",
		Handler:    "index.handler",
	}

	_, err := collection.Build(context.Background(), input)
	if err == nil {
		t.Error("Expected error for unknown runtime")
	}
	
	if !strings.Contains(err.Error(), "Runtime not found") {
		t.Errorf("Expected 'Runtime not found' error, got: %v", err)
	}
}

func TestCollection_Build_WithCopyFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	mockRuntime := &MockRuntime{
		matchPattern: "node",
		buildResult: &runtime.BuildOutput{
			Handler: "index.handler",
			Errors:  []string{},
		},
	}
	
	collection := runtime.NewCollection(tempDir, mockRuntime)
	
	input := &runtime.BuildInput{
		CfgPath:    tempDir,
		FunctionID: "test-function",
		Runtime:    "node18",
		Handler:    "index.handler",
		Dev:        true,
		CopyFiles: []struct {
			From string `json:"from"`
			To   string `json:"to"`
		}{
			{From: sourceFile, To: "dest.txt"},
		},
	}

	result, err := collection.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("Build with copy files failed: %v", err)
	}

	// Check if symlink was created (dev mode)
	destPath := filepath.Join(result.Out, "dest.txt")
	if _, err := os.Lstat(destPath); err != nil {
		t.Errorf("Expected destination file to exist: %v", err)
	}
}

func TestCollection_Build_WithEncryption(t *testing.T) {
	tempDir := t.TempDir()
	
	mockRuntime := &MockRuntime{
		matchPattern: "node",
		buildResult: &runtime.BuildOutput{
			Handler: "index.handler",
			Errors:  []string{},
		},
	}
	
	collection := runtime.NewCollection(tempDir, mockRuntime)
	
	// Create a valid base64 encoded 32-byte key for AES-256
	key := "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=" // base64 of 32 bytes
	
	links := map[string]json.RawMessage{
		"database": json.RawMessage(`{"url": "test-url"}`),
	}
	
	input := &runtime.BuildInput{
		CfgPath:       tempDir,
		FunctionID:    "test-function",
		Runtime:       "node18",
		Handler:       "index.handler",
		EncryptionKey: key,
		Links:         links,
	}

	result, err := collection.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("Build with encryption failed: %v", err)
	}

	// Check if encrypted file was created
	encryptedFile := filepath.Join(result.Out, "resource.enc")
	if _, err := os.Stat(encryptedFile); err != nil {
		t.Errorf("Expected encrypted file to exist: %v", err)
	}
}

func TestCollection_Run(t *testing.T) {
	tempDir := t.TempDir()
	
	mockWorker := &MockWorker{
		logs: io.NopCloser(strings.NewReader("test logs")),
	}
	
	mockRuntime := &MockRuntime{
		matchPattern: "node",
		worker:       mockWorker,
	}
	
	collection := runtime.NewCollection(tempDir, mockRuntime)
	
	input := &runtime.RunInput{
		CfgPath:    tempDir,
		Runtime:    "node18",
		FunctionID: "test-function",
		WorkerID:   "worker-1",
		Build: &runtime.BuildOutput{
			Out:     tempDir,
			Handler: "index.handler",
		},
		Env: []string{"NODE_ENV=test"},
	}

	worker, err := collection.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if worker != mockWorker {
		t.Error("Expected to get the same mock worker instance")
	}
}

func TestCollection_Run_RuntimeNotFound(t *testing.T) {
	tempDir := t.TempDir()
	
	collection := runtime.NewCollection(tempDir)
	
	input := &runtime.RunInput{
		CfgPath:    tempDir,
		Runtime:    "unknown",
		FunctionID: "test-function",
	}

	_, err := collection.Run(context.Background(), input)
	if err == nil {
		t.Error("Expected error for unknown runtime")
	}
	
	if !strings.Contains(err.Error(), "runtime not found") {
		t.Errorf("Expected 'runtime not found' error, got: %v", err)
	}
}

func TestCollection_ShouldRebuild(t *testing.T) {
	tempDir := t.TempDir()
	
	mockRuntime := &MockRuntime{
		matchPattern:  "node",
		shouldRebuild: true,
	}
	
	collection := runtime.NewCollection(tempDir, mockRuntime)
	
	result := collection.ShouldRebuild("node18", "test-function", "test-file.js")
	if !result {
		t.Error("Expected ShouldRebuild to return true")
	}

	// Test with unknown runtime
	result = collection.ShouldRebuild("unknown", "test-function", "test-file.js")
	if result {
		t.Error("Expected ShouldRebuild to return false for unknown runtime")
	}
}

func TestCollection_AddTarget(t *testing.T) {
	tempDir := t.TempDir()
	
	collection := runtime.NewCollection(tempDir)
	
	input := &runtime.BuildInput{
		FunctionID: "test-function",
		Runtime:    "node18",
		Handler:    "index.handler",
	}

	collection.AddTarget(input)
	
	if input.CfgPath != tempDir {
		t.Errorf("Expected CfgPath to be set to %s, got %s", tempDir, input.CfgPath)
	}
}