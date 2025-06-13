package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sst/sst/v3/pkg/global"
)

func TestNeedsInstall(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) *Project
		expected  bool
	}{
		{
			name: "needs install when no lock exists",
			setupFunc: func(t *testing.T) *Project {
				p := &Project{
					app: &App{
						Providers: map[string]interface{}{
							"aws": map[string]interface{}{
								"version": "1.0.0",
							},
						},
					},
					lock: ProviderLock{},
				}
				return p
			},
			expected: true,
		},
		{
			name: "needs install when provider count differs",
			setupFunc: func(t *testing.T) *Project {
				p := &Project{
					app: &App{
						Providers: map[string]interface{}{
							"aws": map[string]interface{}{
								"version": "1.0.0",
							},
							"cloudflare": map[string]interface{}{
								"version": "2.0.0",
							},
						},
					},
					lock: ProviderLock{
						&ProviderLockEntry{
							Name:    "aws",
							Version: "1.0.0",
						},
					},
				}
				return p
			},
			expected: true,
		},
		{
			name: "needs install when version differs",
			setupFunc: func(t *testing.T) *Project {
				p := &Project{
					app: &App{
						Providers: map[string]interface{}{
							"aws": map[string]interface{}{
								"version": "2.0.0",
							},
						},
					},
					lock: ProviderLock{
						&ProviderLockEntry{
							Name:    "aws",
							Version: "1.0.0",
						},
					},
				}
				return p
			},
			expected: true,
		},
		{
			name: "no install needed when versions match",
			setupFunc: func(t *testing.T) *Project {
				p := &Project{
					app: &App{
						Providers: map[string]interface{}{
							"aws": map[string]interface{}{
								"version": "1.0.0",
							},
						},
					},
					lock: ProviderLock{
						&ProviderLockEntry{
							Name:    "aws",
							Version: "1.0.0",
						},
					},
				}
				return p
			},
			expected: false,
		},
		{
			name: "no install needed when version is nil",
			setupFunc: func(t *testing.T) *Project {
				p := &Project{
					app: &App{
						Providers: map[string]interface{}{
							"aws": map[string]interface{}{},
						},
					},
					lock: ProviderLock{
						&ProviderLockEntry{
							Name:    "aws",
							Version: "1.0.0",
						},
					},
				}
				return p
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.setupFunc(t)
			result := p.NeedsInstall()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWritePackageJson(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T) (*Project, string, func())
		expectedDeps   map[string]string
		expectError    bool
	}{
		{
			name: "writes dependencies to new package.json",
			setupFunc: func(t *testing.T) (*Project, string, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-test-*")
				require.NoError(t, err)

				platformDir := filepath.Join(tmpDir, ".sst", "platform")
				err = os.MkdirAll(platformDir, 0755)
				require.NoError(t, err)

				// Create initial package.json
				packageJsonPath := filepath.Join(platformDir, "package.json")
				initialData := map[string]interface{}{
					"name":         "test-project",
					"dependencies": map[string]interface{}{},
				}
				data, err := json.MarshalIndent(initialData, "", "  ")
				require.NoError(t, err)
				err = os.WriteFile(packageJsonPath, data, 0644)
				require.NoError(t, err)

				p := &Project{
					root: tmpDir,
					lock: ProviderLock{
						&ProviderLockEntry{
							Name:    "aws",
							Package: "@pulumi/aws",
							Version: "6.0.0",
						},
						&ProviderLockEntry{
							Name:    "cloudflare",
							Package: "@pulumi/cloudflare",
							Version: "5.0.0",
						},
					},
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return p, packageJsonPath, cleanup
			},
			expectedDeps: map[string]string{
				"@pulumi/aws":        "6.0.0",
				"@pulumi/cloudflare": "5.0.0",
				"@pulumi/pulumi":     global.PULUMI_VERSION,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, packageJsonPath, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := p.writePackageJson()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify package.json was written correctly
			data, err := os.ReadFile(packageJsonPath)
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			dependencies, ok := result["dependencies"].(map[string]interface{})
			require.True(t, ok, "dependencies should be a map")

			for expectedPkg, expectedVersion := range tt.expectedDeps {
				actualVersion, exists := dependencies[expectedPkg]
				assert.True(t, exists, "package %s should exist in dependencies", expectedPkg)
				if exists {
					assert.Equal(t, expectedVersion, actualVersion, "version for %s should match", expectedPkg)
				}
			}
		})
	}
}

func TestWriteTypes(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (*Project, string, func())
		expectError bool
	}{
		{
			name: "writes types file correctly",
			setupFunc: func(t *testing.T) (*Project, string, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-test-*")
				require.NoError(t, err)

				platformDir := filepath.Join(tmpDir, ".sst", "platform")
				err = os.MkdirAll(platformDir, 0755)
				require.NoError(t, err)

				p := &Project{
					root: tmpDir,
					lock: ProviderLock{
						&ProviderLockEntry{
							Name:    "aws",
							Package: "@pulumi/aws",
							Version: "6.0.0",
							Alias:   "aws",
						},
						&ProviderLockEntry{
							Name:    "cloudflare",
							Package: "@pulumi/cloudflare",
							Version: "5.0.0",
							Alias:   "cloudflare",
						},
					},
				}

				typesPath := filepath.Join(platformDir, "config.d.ts")

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return p, typesPath, cleanup
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, typesPath, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := p.writeTypes()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify types file was created and contains expected content
			data, err := os.ReadFile(typesPath)
			require.NoError(t, err)

			content := string(data)
			assert.Contains(t, content, `import "./src/global.d.ts"`)
			assert.Contains(t, content, `import "../types.generated"`)
			assert.Contains(t, content, `import { AppInput, App, Config } from "./src/config"`)
			assert.Contains(t, content, `import * as _aws from "@pulumi/aws";`)
			assert.Contains(t, content, `import * as _cloudflare from "@pulumi/cloudflare";`)
			assert.Contains(t, content, `export import aws = _aws`)
			assert.Contains(t, content, `export import cloudflare = _cloudflare`)
			assert.Contains(t, content, `"aws"?:  (_aws.ProviderArgs & { version?: string }) | boolean | string;`)
			assert.Contains(t, content, `"cloudflare"?:  (_cloudflare.ProviderArgs & { version?: string }) | boolean | string;`)
		})
	}
}

func TestLoadProviderLock(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (*Project, func())
		expectError bool
		expectedLen int
	}{
		{
			name: "loads existing provider lock",
			setupFunc: func(t *testing.T) (*Project, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-test-*")
				require.NoError(t, err)

				// Create .sst directory structure
				sstDir := filepath.Join(tmpDir, ".sst")
				err = os.MkdirAll(sstDir, 0755)
				require.NoError(t, err)

				// Create config file
				configPath := filepath.Join(tmpDir, "sst.config.ts")
				err = os.WriteFile(configPath, []byte("export default {}"), 0644)
				require.NoError(t, err)

				lockData := ProviderLock{
					&ProviderLockEntry{
						Name:    "aws",
						Package: "@pulumi/aws",
						Version: "6.0.0",
						Alias:   "aws",
					},
				}

				lockPath := filepath.Join(sstDir, "provider-lock.json")
				data, err := json.MarshalIndent(lockData, "", "  ")
				require.NoError(t, err)
				err = os.WriteFile(lockPath, data, 0644)
				require.NoError(t, err)

				p := &Project{
					root:   tmpDir,
					config: configPath,
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return p, cleanup
			},
			expectError: false,
			expectedLen: 1,
		},
		{
			name: "handles missing provider lock file",
			setupFunc: func(t *testing.T) (*Project, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-test-*")
				require.NoError(t, err)

				// Create .sst directory structure
				sstDir := filepath.Join(tmpDir, ".sst")
				err = os.MkdirAll(sstDir, 0755)
				require.NoError(t, err)

				// Create config file
				configPath := filepath.Join(tmpDir, "sst.config.ts")
				err = os.WriteFile(configPath, []byte("export default {}"), 0644)
				require.NoError(t, err)

				p := &Project{
					root:   tmpDir,
					config: configPath,
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return p, cleanup
			},
			expectError: false,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := p.loadProviderLock()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, p.lock, tt.expectedLen)
		})
	}
}

func TestWriteProviderLock(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (*Project, string, func())
		expectError bool
	}{
		{
			name: "writes provider lock file",
			setupFunc: func(t *testing.T) (*Project, string, func()) {
				tmpDir, err := os.MkdirTemp("", "sst-test-*")
				require.NoError(t, err)

				// Create .sst directory structure
				sstDir := filepath.Join(tmpDir, ".sst")
				err = os.MkdirAll(sstDir, 0755)
				require.NoError(t, err)

				// Create config file
				configPath := filepath.Join(tmpDir, "sst.config.ts")
				err = os.WriteFile(configPath, []byte("export default {}"), 0644)
				require.NoError(t, err)

				p := &Project{
					root:   tmpDir,
					config: configPath,
					lock: ProviderLock{
						&ProviderLockEntry{
							Name:    "aws",
							Package: "@pulumi/aws",
							Version: "6.0.0",
							Alias:   "aws",
						},
					},
				}

				lockPath := filepath.Join(sstDir, "provider-lock.json")

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return p, lockPath, cleanup
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, lockPath, cleanup := tt.setupFunc(t)
			defer cleanup()

			err := p.writeProviderLock()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify lock file was written correctly
			data, err := os.ReadFile(lockPath)
			require.NoError(t, err)

			var result ProviderLock
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			assert.Len(t, result, 1)
			assert.Equal(t, "aws", result[0].Name)
			assert.Equal(t, "@pulumi/aws", result[0].Package)
			assert.Equal(t, "6.0.0", result[0].Version)
			assert.Equal(t, "aws", result[0].Alias)
		})
	}
}

func TestErrProviderVersionTooLow(t *testing.T) {
	err := &ErrProviderVersionTooLow{
		Name:    "aws",
		Version: "5.0.0",
		Needed:  "6.0.0",
	}

	assert.Equal(t, "provider version too low", err.Error())
}