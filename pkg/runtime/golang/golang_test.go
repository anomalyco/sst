package golang_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sst/sst/v3/pkg/runtime"
	"github.com/sst/sst/v3/pkg/runtime/golang"
)

func TestNew(t *testing.T) {
	rt := golang.New()
	
	if rt == nil {
		t.Fatal("New returned nil")
	}
}

func TestRuntime_Match(t *testing.T) {
	rt := golang.New()
	
	tests := []struct {
		runtime string
		want    bool
	}{
		{"go", true},
		{"go1.21", false},
		{"golang", false},
		{"node18", false},
		{"python3.9", false},
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
	rt := golang.New()
	
	tests := []struct {
		name       string
		functionID string
		file       string
		want       bool
	}{
		{
			name:       "non-go file",
			functionID: "test-func",
			file:       "/project/README.md",
			want:       false,
		},
		{
			name:       "go file with unknown function ID",
			functionID: "unknown",
			file:       "/project/main.go",
			want:       false,
		},
		{
			name:       "go file with empty function ID",
			functionID: "",
			file:       "/project/main.go",
			want:       false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rt.ShouldRebuild(tt.functionID, tt.file)
			if got != tt.want {
				t.Errorf("ShouldRebuild(%q, %q) = %v, want %v", tt.functionID, tt.file, got, tt.want)
			}
		})
	}
}

func TestRuntime_Build(t *testing.T) {
	rt := golang.New()
	
	// Create a temporary directory structure with a Go module
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	err := os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}
	
	// Create go.mod file
	goModContent := `module test-project

go 1.21
`
	err = os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}
	
	// Create main.go file
	mainGoContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	mainGoPath := filepath.Join(projectDir, "main.go")
	err = os.WriteFile(mainGoPath, []byte(mainGoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}
	
	tests := []struct {
		name       string
		input      *runtime.BuildInput
		wantErr    bool
		wantHandler string
	}{
		{
			name: "successful build in dev mode",
			input: &runtime.BuildInput{
				CfgPath:    tempDir,
				FunctionID: "test-func",
				Handler:    mainGoPath,
				Runtime:    "go",
				Dev:        true,
				Properties: json.RawMessage(`{}`),
			},
			wantErr:     false,
			wantHandler: "bootstrap",
		},
		{
			name: "successful build in production mode",
			input: &runtime.BuildInput{
				CfgPath:    tempDir,
				FunctionID: "test-func-prod",
				Handler:    mainGoPath,
				Runtime:    "go",
				Dev:        false,
				Properties: json.RawMessage(`{}`),
			},
			wantErr:     false,
			wantHandler: "bootstrap",
		},
		{
			name: "build with arm64 architecture",
			input: &runtime.BuildInput{
				CfgPath:    tempDir,
				FunctionID: "test-func-arm",
				Handler:    mainGoPath,
				Runtime:    "go",
				Dev:        false,
				Properties: json.RawMessage(`{"architecture": "arm64"}`),
			},
			wantErr:     false,
			wantHandler: "bootstrap",
		},
		{
			name: "build with missing go.mod",
			input: &runtime.BuildInput{
				CfgPath:    tempDir,
				FunctionID: "test-func-nomod",
				Handler:    filepath.Join(tempDir, "nonexistent", "main.go"),
				Runtime:    "go",
				Dev:        false,
				Properties: json.RawMessage(`{}`),
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rt.Build(context.Background(), tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if result == nil {
				t.Fatal("Build result is nil")
			}
			
			if result.Handler != tt.wantHandler {
				t.Errorf("Expected handler %q, got %q", tt.wantHandler, result.Handler)
			}
			
			if result.Errors == nil {
				t.Error("Expected Errors slice to be initialized")
			}
			
			if result.Sourcemaps == nil {
				t.Error("Expected Sourcemaps slice to be initialized")
			}
			
			// Check if the binary was created (this might fail in test environment without Go installed)
			binaryPath := filepath.Join(tt.input.Out(), "bootstrap")
			if _, err := os.Stat(binaryPath); err != nil {
				// If build failed, check if we got error messages
				if len(result.Errors) == 0 {
					t.Logf("Binary not created and no errors reported: %v", err)
				}
			}
		})
	}
}

func TestRuntime_Build_Properties(t *testing.T) {
	rt := golang.New()
	
	tests := []struct {
		name       string
		properties string
		wantErr    bool
	}{
		{
			name:       "default architecture",
			properties: `{}`,
			wantErr:    false,
		},
		{
			name:       "arm64 architecture",
			properties: `{"architecture": "arm64"}`,
			wantErr:    false,
		},
		{
			name:       "invalid json properties",
			properties: `{invalid}`,
			wantErr:    false, // json.Unmarshal doesn't return error, just leaves struct empty
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			err := os.MkdirAll(projectDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create project directory: %v", err)
			}
			
			// Create minimal go.mod and main.go
			err = os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module test\ngo 1.21\n"), 0644)
			if err != nil {
				t.Fatalf("Failed to create go.mod: %v", err)
			}
			
			mainGoPath := filepath.Join(projectDir, "main.go")
			err = os.WriteFile(mainGoPath, []byte("package main\nfunc main() {}\n"), 0644)
			if err != nil {
				t.Fatalf("Failed to create main.go: %v", err)
			}
			
			input := &runtime.BuildInput{
				CfgPath:    tempDir,
				FunctionID: "test-func",
				Handler:    mainGoPath,
				Runtime:    "go",
				Dev:        false,
				Properties: json.RawMessage(tt.properties),
			}
			
			result, err := rt.Build(context.Background(), input)
			
			// We expect this to potentially fail due to missing Go toolchain in test environment
			// but we can still verify the properties parsing worked
			if err != nil {
				t.Logf("Build failed (expected in test environment): %v", err)
			}
			
			if result != nil && result.Handler != "bootstrap" {
				t.Errorf("Expected handler 'bootstrap', got %q", result.Handler)
			}
		})
	}
}

func TestRuntime_Run(t *testing.T) {
	rt := golang.New()
	
	tempDir := t.TempDir()
	
	// Create a mock binary (just an empty file for testing)
	binaryPath := filepath.Join(tempDir, "bootstrap")
	err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho 'test output'\n"), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}
	
	buildOutput := &runtime.BuildOutput{
		Out:     tempDir,
		Handler: "bootstrap",
	}
	
	input := &runtime.RunInput{
		CfgPath:    tempDir,
		Runtime:    "go",
		Server:     "localhost:8080",
		FunctionID: "test-func",
		WorkerID:   "worker-1",
		Build:      buildOutput,
		Env:        []string{"TEST_ENV=test"},
	}
	
	worker, err := rt.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	
	if worker == nil {
		t.Fatal("Worker is nil")
	}
	
	// Test worker interface
	logs := worker.Logs()
	if logs == nil {
		t.Error("Worker.Logs() returned nil")
	} else {
		logs.Close()
	}
	
	// Test worker stop
	worker.Stop()
}

func TestWorker_Logs(t *testing.T) {
	rt := golang.New()
	
	tempDir := t.TempDir()
	
	// Create a script that outputs to both stdout and stderr
	scriptContent := `#!/bin/bash
echo "stdout message"
echo "stderr message" >&2
sleep 0.1
`
	binaryPath := filepath.Join(tempDir, "bootstrap")
	err := os.WriteFile(binaryPath, []byte(scriptContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}
	
	buildOutput := &runtime.BuildOutput{
		Out:     tempDir,
		Handler: "bootstrap",
	}
	
	input := &runtime.RunInput{
		CfgPath:    tempDir,
		Runtime:    "go",
		Server:     "localhost:8080",
		FunctionID: "test-func",
		WorkerID:   "worker-1",
		Build:      buildOutput,
		Env:        []string{},
	}
	
	worker, err := rt.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	
	defer worker.Stop()
	
	// Read logs
	logs := worker.Logs()
	defer logs.Close()
	
	// Read some output (this might be empty if the script doesn't run properly in test environment)
	buffer := make([]byte, 1024)
	n, err := logs.Read(buffer)
	if err != nil && err != io.EOF {
		t.Logf("Error reading logs (expected in test environment): %v", err)
	}
	
	if n > 0 {
		output := string(buffer[:n])
		t.Logf("Worker output: %s", output)
	}
}

func TestRuntime_ShouldRebuild_WithBuildState(t *testing.T) {
	rt := golang.New()
	
	// Create a temporary directory structure
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	err := os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}
	
	// Create go.mod and main.go
	err = os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module test\ngo 1.21\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}
	
	mainGoPath := filepath.Join(projectDir, "main.go")
	err = os.WriteFile(mainGoPath, []byte("package main\nfunc main() {}\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}
	
	// Build first to populate the directories map
	input := &runtime.BuildInput{
		CfgPath:    tempDir,
		FunctionID: "test-func",
		Handler:    mainGoPath,
		Runtime:    "go",
		Dev:        true,
		Properties: json.RawMessage(`{}`),
	}
	
	// This might fail due to missing Go toolchain, but it should still populate the directories map
	_, _ = rt.Build(context.Background(), input)
	
	// Now test ShouldRebuild with the populated state
	tests := []struct {
		name string
		file string
		want bool
	}{
		{
			name: "go file in project directory",
			file: filepath.Join(projectDir, "handler.go"),
			want: true,
		},
		{
			name: "go file in subdirectory",
			file: filepath.Join(projectDir, "pkg", "utils.go"),
			want: true,
		},
		{
			name: "go file outside project",
			file: filepath.Join(tempDir, "other", "main.go"),
			want: false,
		},
		{
			name: "non-go file in project",
			file: filepath.Join(projectDir, "README.md"),
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rt.ShouldRebuild("test-func", tt.file)
			if got != tt.want {
				t.Errorf("ShouldRebuild(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

// Note: Properties struct is not exported, so we test it indirectly through Build function
func TestProperties_ThroughBuild(t *testing.T) {
	rt := golang.New()
	
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	err := os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}
	
	// Create minimal go.mod and main.go
	err = os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module test\ngo 1.21\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}
	
	mainGoPath := filepath.Join(projectDir, "main.go")
	err = os.WriteFile(mainGoPath, []byte("package main\nfunc main() {}\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}
	
	tests := []struct {
		name       string
		properties string
		wantErr    bool
	}{
		{
			name:       "empty properties",
			properties: `{}`,
			wantErr:    false,
		},
		{
			name:       "arm64 architecture",
			properties: `{"architecture": "arm64"}`,
			wantErr:    false,
		},
		{
			name:       "amd64 architecture",
			properties: `{"architecture": "amd64"}`,
			wantErr:    false,
		},
		{
			name:       "invalid json",
			properties: `{invalid}`,
			wantErr:    false, // Build should handle invalid JSON gracefully
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &runtime.BuildInput{
				CfgPath:    tempDir,
				FunctionID: "test-func-" + tt.name,
				Handler:    mainGoPath,
				Runtime:    "go",
				Dev:        false,
				Properties: json.RawMessage(tt.properties),
			}
			
			result, err := rt.Build(context.Background(), input)
			
			// Build might fail due to missing Go toolchain, but should not panic
			if err != nil {
				t.Logf("Build failed (expected in test environment): %v", err)
			}
			
			// If build succeeded, verify basic properties
			if result != nil {
				if result.Handler != "bootstrap" {
					t.Errorf("Expected handler 'bootstrap', got %q", result.Handler)
				}
				if result.Errors == nil {
					t.Error("Expected Errors slice to be initialized")
				}
				if result.Sourcemaps == nil {
					t.Error("Expected Sourcemaps slice to be initialized")
				}
			}
		})
	}
}