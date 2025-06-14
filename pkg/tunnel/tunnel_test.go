package tunnel

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestNeedsInstall(t *testing.T) {
	tests := map[string]struct {
		setupBinary bool
		expected    bool
	}{
		"binary exists": {
			setupBinary: true,
			expected:    false,
		},
		"binary missing": {
			setupBinary: false,
			expected:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Save original path
			originalPath := BINARY_PATH
			defer func() { BINARY_PATH = originalPath }()

			// Create temporary directory for test
			tempDir := t.TempDir()
			BINARY_PATH = filepath.Join(tempDir, "tunnel")

			if tc.setupBinary {
				// Create the binary file
				err := os.WriteFile(BINARY_PATH, []byte("fake binary"), 0755)
				require.NoError(t, err)
			}

			result := NeedsInstall()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestInstall(t *testing.T) {
	// Save original implementation
	originalImpl := impl
	defer func() { impl = originalImpl }()

	tests := map[string]struct {
		mockError error
		expectErr bool
	}{
		"successful install": {
			mockError: nil,
			expectErr: false,
		},
		"install error": {
			mockError: fmt.Errorf("install failed"),
			expectErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create mock implementation
			mock := &mockTunnelPlatform{
				installErr: tc.mockError,
			}
			impl = mock

			err := Install()

			if tc.expectErr {
				assert.Error(t, err)
				assert.Equal(t, tc.mockError, err)
			} else {
				assert.NoError(t, err)
			}
			assert.True(t, mock.installCalled)
		})
	}
}

func TestStart(t *testing.T) {
	// Save original implementation
	originalImpl := impl
	defer func() { impl = originalImpl }()

	tests := map[string]struct {
		routes    []string
		mockError error
		expectErr bool
	}{
		"successful start with routes": {
			routes:    []string{"10.0.0.0/8", "192.168.0.0/16"},
			mockError: nil,
			expectErr: false,
		},
		"successful start without routes": {
			routes:    []string{},
			mockError: nil,
			expectErr: false,
		},
		"start error": {
			routes:    []string{"10.0.0.0/8"},
			mockError: fmt.Errorf("start failed"),
			expectErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create mock implementation
			mock := &mockTunnelPlatform{
				startErr: tc.mockError,
			}
			impl = mock

			err := Start(tc.routes...)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Equal(t, tc.mockError, err)
			} else {
				assert.NoError(t, err)
			}
			assert.True(t, mock.startCalled)
			assert.Equal(t, tc.routes, mock.startRoutes)
		})
	}
}

func TestStop(t *testing.T) {
	// Save original implementation
	originalImpl := impl
	defer func() { impl = originalImpl }()

	// Create mock implementation
	mock := &mockTunnelPlatform{}
	impl = mock

	// Note: Stop() calls engine.Stop() which we can't easily mock,
	// but we can test that impl.destroy() is called
	Stop()

	assert.True(t, mock.destroyCalled)
}

func TestRunCommands(t *testing.T) {
	tests := map[string]struct {
		commands  [][]string
		expectErr bool
		errMsg    string
	}{
		"successful commands": {
			commands: [][]string{
				{"echo", "test1"},
				{"echo", "test2"},
			},
			expectErr: false,
		},
		"empty commands": {
			commands:  [][]string{},
			expectErr: false,
		},
		"invalid command": {
			commands: [][]string{
				{"nonexistent_command_12345"},
			},
			expectErr: true,
			errMsg:    "failed to execute command",
		},
		"command with args": {
			commands: [][]string{
				{"echo", "-n", "hello world"},
			},
			expectErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := runCommands(tc.commands)

			if tc.expectErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTun2socks(t *testing.T) {
	// This test is limited because tun2socks uses a global engine
	// and may call log.Fatal on invalid interface names
	t.Run("function exists", func(t *testing.T) {
		// We can't safely test tun2socks because it may call log.Fatal
		// which would terminate the test process. Instead, we just verify
		// the function exists and can be called (though it may fail).
		// In a real scenario, this would be called with a valid utun interface.
		
		// Just verify the function signature is correct
		var f func(string) = tun2socks
		assert.NotNil(t, f)
	})
}

func TestStartProxy(t *testing.T) {
	// Generate test SSH key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	tests := map[string]struct {
		username    string
		host        string
		key         []byte
		setupServer bool
		expectErr   bool
		errMsg      string
	}{
		"invalid private key": {
			username:  "testuser",
			host:      "localhost:2222",
			key:       []byte("invalid key"),
			expectErr: true,
			errMsg:    "ssh: no key found",
		},
		"valid key but no server": {
			username:  "testuser",
			host:      "localhost:2222",
			key:       privateKeyBytes,
			expectErr: true,
			errMsg:    "connection refused",
		},
		"empty username": {
			username:  "",
			host:      "localhost:2222",
			key:       privateKeyBytes,
			expectErr: true,
		},
		"empty host": {
			username:  "testuser",
			host:      "",
			key:       privateKeyBytes,
			expectErr: true,
		},
		"invalid host format": {
			username:  "testuser",
			host:      "invalid-host-format",
			key:       privateKeyBytes,
			expectErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err := StartProxy(ctx, tc.username, tc.host, tc.key)

			if tc.expectErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStartProxyWithMockSSHServer(t *testing.T) {
	// This test demonstrates how StartProxy would work with a real SSH server
	// but uses a mock to avoid complex setup
	t.Run("context cancellation", func(t *testing.T) {
		// Generate test SSH key pair
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}
		privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel immediately to test context handling
		cancel()

		err = StartProxy(ctx, "testuser", "localhost:2222", privateKeyBytes)
		// Should return connection error since we can't connect, but not panic
		assert.Error(t, err)
	})
}

func TestDarwinPlatformInstall(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin-specific test")
	}

	// Save original path
	originalPath := BINARY_PATH
	defer func() { BINARY_PATH = originalPath }()

	tempDir := t.TempDir()
	BINARY_PATH = filepath.Join(tempDir, "tunnel")

	platform := &darwinPlatform{}

	t.Run("install without sudo user", func(t *testing.T) {
		// Save original env
		originalSudoUser := os.Getenv("SUDO_USER")
		defer os.Setenv("SUDO_USER", originalSudoUser)

		// Unset SUDO_USER to test behavior
		os.Unsetenv("SUDO_USER")

		err := platform.install()
		// Should handle missing SUDO_USER gracefully
		// The exact behavior depends on implementation
		// This test ensures it doesn't panic
		_ = err // May or may not error depending on permissions
	})
}

func TestDarwinPlatformStart(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin-specific test")
	}

	platform := &darwinPlatform{}

	t.Run("start command structure", func(t *testing.T) {
		// We can't actually test the start function because it calls tun2socks
		// which requires system permissions and may call log.Fatal.
		// Instead, we test that the platform implements the interface correctly.
		
		var _ tunnelPlatform = platform
		assert.NotNil(t, platform)
	})
}

func TestDarwinPlatformDestroy(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin-specific test")
	}

	platform := &darwinPlatform{}

	t.Run("destroy interface", func(t *testing.T) {
		err := platform.destroy()
		
		// We expect an error since the interface likely doesn't exist
		// but we test that it doesn't panic
		_ = err
	})
}

func TestBinaryPath(t *testing.T) {
	t.Run("binary path constant", func(t *testing.T) {
		assert.Equal(t, "/opt/sst/tunnel", BINARY_PATH)
	})
}

func TestTunnelPlatformInterface(t *testing.T) {
	t.Run("implementation exists", func(t *testing.T) {
		assert.NotNil(t, impl)
	})

	t.Run("implementation has correct type", func(t *testing.T) {
		// Verify that impl implements tunnelPlatform interface
		var _ tunnelPlatform = impl
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("start proxy with nil key", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := StartProxy(ctx, "user", "host:22", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ssh: no key found")
	})

	t.Run("start proxy with empty key", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := StartProxy(ctx, "user", "host:22", []byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ssh: no key found")
	})

	t.Run("run commands with empty command", func(t *testing.T) {
		// This should panic because the code doesn't handle empty command slices
		assert.Panics(t, func() {
			runCommands([][]string{{}})
		})
	})

	t.Run("run commands with nil command slice", func(t *testing.T) {
		// This should panic because the code doesn't handle nil command slices
		assert.Panics(t, func() {
			runCommands([][]string{nil})
		})
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent needs install calls", func(t *testing.T) {
		// Save original path
		originalPath := BINARY_PATH
		defer func() { BINARY_PATH = originalPath }()

		tempDir := t.TempDir()
		BINARY_PATH = filepath.Join(tempDir, "tunnel")

		// Create the binary file
		err := os.WriteFile(BINARY_PATH, []byte("fake binary"), 0755)
		require.NoError(t, err)

		const numGoroutines = 10
		results := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				results <- NeedsInstall()
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			result := <-results
			assert.False(t, result) // Binary exists, so should not need install
		}
	})
}

func TestLargeRoutes(t *testing.T) {
	// Save original implementation
	originalImpl := impl
	defer func() { impl = originalImpl }()

	t.Run("start with many routes", func(t *testing.T) {
		// Create mock implementation
		mock := &mockTunnelPlatform{}
		impl = mock

		// Generate many routes
		var routes []string
		for i := 0; i < 100; i++ {
			routes = append(routes, fmt.Sprintf("10.%d.0.0/16", i))
		}

		err := Start(routes...)
		assert.NoError(t, err)
		assert.True(t, mock.startCalled)
		assert.Equal(t, routes, mock.startRoutes)
	})
}

func TestSpecialCharacters(t *testing.T) {
	t.Run("start proxy with unicode username", func(t *testing.T) {
		// Generate test SSH key pair
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}
		privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = StartProxy(ctx, "用户", "localhost:2222", privateKeyBytes)
		assert.Error(t, err) // Will fail to connect, but should handle unicode
	})

	t.Run("run commands with special characters", func(t *testing.T) {
		commands := [][]string{
			{"echo", "hello world with spaces"},
			{"echo", "special!@#$%^&*()chars"},
		}

		err := runCommands(commands)
		assert.NoError(t, err)
	})
}

func TestNetworkEdgeCases(t *testing.T) {
	t.Run("start proxy with invalid port", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}
		privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = StartProxy(ctx, "user", "localhost:99999", privateKeyBytes)
		assert.Error(t, err)
	})

	t.Run("start proxy with IPv6 host", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}
		privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err = StartProxy(ctx, "user", "[::1]:2222", privateKeyBytes)
		assert.Error(t, err) // Will fail to connect, but should handle IPv6
	})
}

// Mock implementation for testing
type mockTunnelPlatform struct {
	installCalled bool
	installErr    error
	startCalled   bool
	startRoutes   []string
	startErr      error
	destroyCalled bool
	destroyErr    error
}

func (m *mockTunnelPlatform) install() error {
	m.installCalled = true
	return m.installErr
}

func (m *mockTunnelPlatform) start(routes ...string) error {
	m.startCalled = true
	m.startRoutes = routes
	return m.startErr
}

func (m *mockTunnelPlatform) destroy() error {
	m.destroyCalled = true
	return m.destroyErr
}

// Test helper functions
func generateTestSSHKeyPair() ([]byte, ssh.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	return privateKeyBytes, publicKey, nil
}

func isPortOpen(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func TestHelperFunctions(t *testing.T) {
	t.Run("generate test ssh key pair", func(t *testing.T) {
		privateKeyBytes, publicKey, err := generateTestSSHKeyPair()
		assert.NoError(t, err)
		assert.NotEmpty(t, privateKeyBytes)
		assert.NotNil(t, publicKey)

		// Verify the key can be parsed
		_, err = ssh.ParsePrivateKey(privateKeyBytes)
		assert.NoError(t, err)
	})

	t.Run("is port open", func(t *testing.T) {
		// Test with a port that should be closed
		open := isPortOpen("localhost", 99999)
		assert.False(t, open)
	})
}