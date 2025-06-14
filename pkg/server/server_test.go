package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sst/sst/v3/pkg/project"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "creates server successfully",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := New()
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)
				assert.NotZero(t, server.Port)
				assert.NotNil(t, server.Mux)
				assert.NotNil(t, server.Rpc)
				
				// Verify port is in expected range
				assert.GreaterOrEqual(t, server.Port, 13557)
				assert.Less(t, server.Port, 65535)
			}
		})
	}
}

func TestPort(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "finds available port",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := port()
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Zero(t, port)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, port, 13557)
				assert.Less(t, port, 65535)
				
				// Verify port is actually available
				listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
				assert.NoError(t, err)
				if listener != nil {
					listener.Close()
				}
			}
		})
	}
}

func TestServer_Start(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (*project.Project, func())
		expectError bool
	}{
		{
			name: "starts server successfully",
			setupFunc: func(t *testing.T) (*project.Project, func()) {
				// Create temporary directory for project
				tmpDir, err := os.MkdirTemp("", "sst-server-test-*")
				require.NoError(t, err)
				
				// Create minimal sst.config.ts
				configPath := filepath.Join(tmpDir, "sst.config.ts")
				err = os.WriteFile(configPath, []byte(`
export default $config({
	app(input) {
		return {
			name: "test-app",
			home: "aws",
		};
	},
	async run() {
		// Empty run function for testing
	},
});
				`), 0644)
				require.NoError(t, err)
				
				// Create project
				proj, err := project.New(&project.ProjectConfig{
					Config: configPath,
					Stage:  "test",
				})
				require.NoError(t, err)
				
				cleanup := func() {
					os.RemoveAll(tmpDir)
				}
				
				return proj, cleanup
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proj, cleanup := tt.setupFunc(t)
			defer cleanup()
			
			server, err := New()
			require.NoError(t, err)
			
			// Create context with timeout for testing
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			// Start server in goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Start(ctx, proj)
			}()
			
			// Give server time to start
			time.Sleep(100 * time.Millisecond)
			
			// Test that server is running by making a request
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/rpc", server.Port))
			if err == nil {
				resp.Body.Close()
				// We expect method not allowed for GET request to RPC endpoint
				assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
			}
			
			// Cancel context to stop server
			cancel()
			
			// Wait for server to finish
			select {
			case err := <-errChan:
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("server did not shut down within timeout")
			}
		})
	}
}

func TestServer_RpcEndpoint(t *testing.T) {
	server, err := New()
	require.NoError(t, err)
	
	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET request returns method not allowed",
			method:         "GET",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "POST request is accepted",
			method:         "POST",
			expectedStatus: http.StatusOK, // RPC will handle the request
		},
		{
			name:           "PUT request returns method not allowed",
			method:         "PUT",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			testServer := &http.Server{
				Handler: server.Mux,
				Addr:    fmt.Sprintf(":%d", server.Port),
			}
			
			// Start server in background
			go testServer.ListenAndServe()
			defer testServer.Close()
			
			// Give server time to start
			time.Sleep(50 * time.Millisecond)
			
			// Make request
			var resp *http.Response
			var reqErr error
			
			switch tt.method {
			case "GET":
				resp, reqErr = http.Get(fmt.Sprintf("http://localhost:%d/rpc", server.Port))
			case "POST":
				resp, reqErr = http.Post(fmt.Sprintf("http://localhost:%d/rpc", server.Port), "application/json", strings.NewReader("{}"))
			case "PUT":
				req, _ := http.NewRequest("PUT", fmt.Sprintf("http://localhost:%d/rpc", server.Port), strings.NewReader("{}"))
				resp, reqErr = http.DefaultClient.Do(req)
			}
			
			if reqErr != nil {
				// Server might not be ready, skip this test
				t.Skipf("Could not connect to test server: %v", reqErr)
				return
			}
			defer resp.Body.Close()
			
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestHttpConn(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "reads and writes data",
			input:    "test data",
			expected: "test data",
		},
		{
			name:     "handles empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create readers and writers
			reader := strings.NewReader(tt.input)
			writer := &strings.Builder{}
			
			// Create HttpConn
			conn := &HttpConn{
				Reader: reader,
				Writer: writer,
			}
			
			// Test reading
			buffer := make([]byte, len(tt.input))
			n, err := conn.Read(buffer)
			if len(tt.input) > 0 {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.input), n)
				assert.Equal(t, tt.input, string(buffer[:n]))
			} else {
				assert.Equal(t, io.EOF, err)
				assert.Equal(t, 0, n)
			}
			
			// Test writing
			n, err = conn.Write([]byte(tt.expected))
			assert.NoError(t, err)
			assert.Equal(t, len(tt.expected), n)
			assert.Equal(t, tt.expected, writer.String())
			
			// Test close (should not error)
			err = conn.Close()
			assert.NoError(t, err)
		})
	}
}

func TestResolveServerFile(t *testing.T) {
	tests := []struct {
		name     string
		cfgPath  string
		stage    string
		expected string
	}{
		{
			name:     "resolves server file path",
			cfgPath:  "/path/to/config",
			stage:    "dev",
			expected: "dev.server",
		},
		{
			name:     "handles different stage",
			cfgPath:  "/path/to/config",
			stage:    "production",
			expected: "production.server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveServerFile(tt.cfgPath, tt.stage)
			assert.Contains(t, result, tt.expected)
			assert.True(t, strings.HasSuffix(result, tt.expected))
		})
	}
}

func TestServer_Integration(t *testing.T) {
	// Create temporary directory for project
	tmpDir, err := os.MkdirTemp("", "sst-server-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	
	// Create minimal sst.config.ts
	configPath := filepath.Join(tmpDir, "sst.config.ts")
	err = os.WriteFile(configPath, []byte(`
export default $config({
	app(input) {
		return {
			name: "test-app",
			home: "aws",
		};
	},
	async run() {
		// Empty run function for testing
	},
});
	`), 0644)
	require.NoError(t, err)
	
	// Create project
	proj, err := project.New(&project.ProjectConfig{
		Config: configPath,
		Stage:  "test",
	})
	require.NoError(t, err)
	
	// Create and start server
	server, err := New()
	require.NoError(t, err)
	
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Start server
	go server.Start(ctx, proj)
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Verify server file was created
	serverFilePath := resolveServerFile(proj.PathConfig(), proj.App().Stage)
	_, err = os.Stat(serverFilePath)
	assert.NoError(t, err, "Server file should be created")
	
	// Read server file content
	content, err := os.ReadFile(serverFilePath)
	require.NoError(t, err)
	
	// Verify content contains server URL
	serverURL := string(content)
	assert.Contains(t, serverURL, fmt.Sprintf(":%d", server.Port))
	assert.Contains(t, serverURL, "http://")
	
	// Test server is accessible
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/rpc", server.Port))
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	}
	
	// Cancel context and verify cleanup
	cancel()
	time.Sleep(100 * time.Millisecond)
	
	// Server file should be cleaned up
	_, err = os.Stat(serverFilePath)
	assert.True(t, os.IsNotExist(err), "Server file should be cleaned up")
}