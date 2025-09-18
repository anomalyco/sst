package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommonProjectPatterns tests the most common Python project patterns
// that users have been successfully using with SST to ensure they continue
// to work after the runtime simplification.
func TestCommonProjectPatterns(t *testing.T) {
	patterns := []struct {
		name         string
		description  string
		setupProject func(t *testing.T, projectDir string)
		testCases    []struct {
			handler        string
			expectSuccess  bool
			expectedModule string
		}
	}{
		{
			name:        "django_style_apps",
			description: "Django-style project with apps directory",
			setupProject: func(t *testing.T, projectDir string) {
				// Create Django-style structure
				appsDir := filepath.Join(projectDir, "apps")
				apiDir := filepath.Join(appsDir, "api")
				usersDir := filepath.Join(appsDir, "users")

				for _, dir := range []string{apiDir, usersDir} {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("Failed to create directory %s: %v", dir, err)
					}
				}

				// Create __init__.py files
				initFiles := []string{
					filepath.Join(appsDir, "__init__.py"),
					filepath.Join(apiDir, "__init__.py"),
					filepath.Join(usersDir, "__init__.py"),
				}
				for _, initFile := range initFiles {
					if err := os.WriteFile(initFile, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handlers
				apiHandler := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'API handler'}
`
				usersHandler := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Users handler'}
`

				if err := os.WriteFile(filepath.Join(apiDir, "views.py"), []byte(apiHandler), 0644); err != nil {
					t.Fatalf("Failed to create API handler: %v", err)
				}
				if err := os.WriteFile(filepath.Join(usersDir, "views.py"), []byte(usersHandler), 0644); err != nil {
					t.Fatalf("Failed to create users handler: %v", err)
				}

				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "django-style-project"
version = "0.1.0"
dependencies = ["django>=4.0"]
`
				if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}
			},
			testCases: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"apps/api/views.lambda_handler", true, "apps.api.views"},
				{"apps/users/views.lambda_handler", true, "apps.users.views"},
			},
		},
		{
			name:        "flask_blueprints",
			description: "Flask-style project with blueprints",
			setupProject: func(t *testing.T, projectDir string) {
				// Create Flask-style structure
				appDir := filepath.Join(projectDir, "app")
				blueprintsDir := filepath.Join(appDir, "blueprints")
				apiDir := filepath.Join(blueprintsDir, "api")
				authDir := filepath.Join(blueprintsDir, "auth")

				for _, dir := range []string{apiDir, authDir} {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("Failed to create directory %s: %v", dir, err)
					}
				}

				// Create __init__.py files
				initFiles := []string{
					filepath.Join(appDir, "__init__.py"),
					filepath.Join(blueprintsDir, "__init__.py"),
					filepath.Join(apiDir, "__init__.py"),
					filepath.Join(authDir, "__init__.py"),
				}
				for _, initFile := range initFiles {
					if err := os.WriteFile(initFile, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handlers
				apiHandler := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'API blueprint handler'}
`
				authHandler := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Auth blueprint handler'}
`

				if err := os.WriteFile(filepath.Join(apiDir, "routes.py"), []byte(apiHandler), 0644); err != nil {
					t.Fatalf("Failed to create API handler: %v", err)
				}
				if err := os.WriteFile(filepath.Join(authDir, "routes.py"), []byte(authHandler), 0644); err != nil {
					t.Fatalf("Failed to create auth handler: %v", err)
				}

				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "flask-style-project"
version = "0.1.0"
dependencies = ["flask>=2.0"]
`
				if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}
			},
			testCases: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"app/blueprints/api/routes.lambda_handler", true, "app.blueprints.api.routes"},
				{"app/blueprints/auth/routes.lambda_handler", true, "app.blueprints.auth.routes"},
			},
		},
		{
			name:        "fastapi_routers",
			description: "FastAPI-style project with routers",
			setupProject: func(t *testing.T, projectDir string) {
				// Create FastAPI-style structure
				appDir := filepath.Join(projectDir, "app")
				routersDir := filepath.Join(appDir, "routers")

				if err := os.MkdirAll(routersDir, 0755); err != nil {
					t.Fatalf("Failed to create routers directory: %v", err)
				}

				// Create __init__.py files
				initFiles := []string{
					filepath.Join(appDir, "__init__.py"),
					filepath.Join(routersDir, "__init__.py"),
				}
				for _, initFile := range initFiles {
					if err := os.WriteFile(initFile, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handlers
				usersHandler := `
from fastapi import FastAPI

def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Users router handler'}
`
				itemsHandler := `
from fastapi import FastAPI

def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Items router handler'}
`

				if err := os.WriteFile(filepath.Join(routersDir, "users.py"), []byte(usersHandler), 0644); err != nil {
					t.Fatalf("Failed to create users handler: %v", err)
				}
				if err := os.WriteFile(filepath.Join(routersDir, "items.py"), []byte(itemsHandler), 0644); err != nil {
					t.Fatalf("Failed to create items handler: %v", err)
				}

				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "fastapi-style-project"
version = "0.1.0"
dependencies = ["fastapi>=0.104.0"]
`
				if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}
			},
			testCases: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"app/routers/users.lambda_handler", true, "app.routers.users"},
				{"app/routers/items.lambda_handler", true, "app.routers.items"},
			},
		},
		{
			name:        "microservices_monorepo",
			description: "Microservices monorepo with multiple services",
			setupProject: func(t *testing.T, projectDir string) {
				// Create monorepo structure
				servicesDir := filepath.Join(projectDir, "services")
				userServiceDir := filepath.Join(servicesDir, "user-service")
				orderServiceDir := filepath.Join(servicesDir, "order-service")
				sharedDir := filepath.Join(projectDir, "shared")

				for _, dir := range []string{userServiceDir, orderServiceDir, sharedDir} {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("Failed to create directory %s: %v", dir, err)
					}
				}

				// Create root pyproject.toml with workspace
				rootPyprojectContent := `
[project]
name = "microservices-monorepo"
version = "0.1.0"

[tool.uv.workspace]
members = ["services/*", "shared"]
`
				if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(rootPyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create root pyproject.toml: %v", err)
				}

				// Create service pyproject.toml files
				userServicePyproject := `
[project]
name = "user-service"
version = "0.1.0"
dependencies = ["shared", "boto3>=1.34.0"]
`
				orderServicePyproject := `
[project]
name = "order-service"
version = "0.1.0"
dependencies = ["shared", "boto3>=1.34.0"]
`
				sharedPyproject := `
[project]
name = "shared"
version = "0.1.0"
dependencies = []
`

				if err := os.WriteFile(filepath.Join(userServiceDir, "pyproject.toml"), []byte(userServicePyproject), 0644); err != nil {
					t.Fatalf("Failed to create user service pyproject.toml: %v", err)
				}
				if err := os.WriteFile(filepath.Join(orderServiceDir, "pyproject.toml"), []byte(orderServicePyproject), 0644); err != nil {
					t.Fatalf("Failed to create order service pyproject.toml: %v", err)
				}
				if err := os.WriteFile(filepath.Join(sharedDir, "pyproject.toml"), []byte(sharedPyproject), 0644); err != nil {
					t.Fatalf("Failed to create shared pyproject.toml: %v", err)
				}

				// Create handlers
				userHandler := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'User service handler'}
`
				orderHandler := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Order service handler'}
`

				if err := os.WriteFile(filepath.Join(userServiceDir, "handler.py"), []byte(userHandler), 0644); err != nil {
					t.Fatalf("Failed to create user handler: %v", err)
				}
				if err := os.WriteFile(filepath.Join(orderServiceDir, "handler.py"), []byte(orderHandler), 0644); err != nil {
					t.Fatalf("Failed to create order handler: %v", err)
				}

				// Create shared utility
				sharedUtil := `
def get_timestamp():
    import datetime
    return datetime.datetime.now().isoformat()
`
				if err := os.WriteFile(filepath.Join(sharedDir, "utils.py"), []byte(sharedUtil), 0644); err != nil {
					t.Fatalf("Failed to create shared utils: %v", err)
				}
			},
			testCases: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"services/user-service/handler.lambda_handler", true, "handler"},
				{"services/order-service/handler.lambda_handler", true, "handler"},
			},
		},
		{
			name:        "src_layout_package",
			description: "Standard src layout with package structure",
			setupProject: func(t *testing.T, projectDir string) {
				// Create src layout structure
				srcDir := filepath.Join(projectDir, "src")
				packageDir := filepath.Join(srcDir, "mypackage")
				handlersDir := filepath.Join(packageDir, "handlers")

				if err := os.MkdirAll(handlersDir, 0755); err != nil {
					t.Fatalf("Failed to create handlers directory: %v", err)
				}

				// Create __init__.py files
				initFiles := []string{
					filepath.Join(packageDir, "__init__.py"),
					filepath.Join(handlersDir, "__init__.py"),
				}
				for _, initFile := range initFiles {
					if err := os.WriteFile(initFile, []byte(""), 0644); err != nil {
						t.Fatalf("Failed to create __init__.py: %v", err)
					}
				}

				// Create handlers
				apiHandler := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'API handler from src layout'}
`
				workerHandler := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Worker handler from src layout'}
`

				if err := os.WriteFile(filepath.Join(handlersDir, "api.py"), []byte(apiHandler), 0644); err != nil {
					t.Fatalf("Failed to create API handler: %v", err)
				}
				if err := os.WriteFile(filepath.Join(handlersDir, "worker.py"), []byte(workerHandler), 0644); err != nil {
					t.Fatalf("Failed to create worker handler: %v", err)
				}

				// Create pyproject.toml
				pyprojectContent := `
[project]
name = "mypackage"
version = "0.1.0"
dependencies = ["boto3>=1.34.0"]

[build-system]
requires = ["setuptools", "wheel"]
build-backend = "setuptools.build_meta"

[tool.setuptools.packages.find]
where = ["src"]
`
				if err := os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}
			},
			testCases: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"src/mypackage/handlers/api.lambda_handler", true, "mypackage.handlers.api"},
				{"src/mypackage/handlers/worker.lambda_handler", true, "mypackage.handlers.worker"},
			},
		},
		{
			name:        "poetry_project_structure",
			description: "Poetry project with standard structure",
			setupProject: func(t *testing.T, projectDir string) {
				// Create Poetry-style structure
				packageDir := filepath.Join(projectDir, "myproject")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}

				// Create __init__.py
				initPath := filepath.Join(packageDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				// Create handler
				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Poetry project handler'}
`
				handlerPath := filepath.Join(packageDir, "main.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				// Create pyproject.toml with Poetry configuration
				pyprojectContent := `
[tool.poetry]
name = "myproject"
version = "0.1.0"
description = "A Poetry project"
authors = ["Your Name <you@example.com>"]

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.31.0"

[tool.poetry.group.dev.dependencies]
pytest = "^7.0.0"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}
			},
			testCases: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"myproject/main.lambda_handler", true, "myproject.main"},
			},
		},
		{
			name:        "simple_flat_structure",
			description: "Simple flat structure without pyproject.toml",
			setupProject: func(t *testing.T, projectDir string) {
				// Create simple handlers
				handlers := map[string]string{
					"api.py": `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Simple API handler'}
`,
					"worker.py": `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Simple worker handler'}
`,
					"utils.py": `
def helper_function():
    return "Helper"
`,
				}

				for filename, content := range handlers {
					handlerPath := filepath.Join(projectDir, filename)
					if err := os.WriteFile(handlerPath, []byte(content), 0644); err != nil {
						t.Fatalf("Failed to create %s: %v", filename, err)
					}
				}

				// Create requirements.txt
				requirementsContent := "requests>=2.31.0\nboto3>=1.34.0\n"
				requirementsPath := filepath.Join(projectDir, "requirements.txt")
				if err := os.WriteFile(requirementsPath, []byte(requirementsContent), 0644); err != nil {
					t.Fatalf("Failed to create requirements.txt: %v", err)
				}
			},
			testCases: []struct {
				handler        string
				expectSuccess  bool
				expectedModule string
			}{
				{"api.lambda_handler", true, "api"},
				{"worker.lambda_handler", true, "worker"},
			},
		},
	}

	for _, pattern := range patterns {
		t.Run(pattern.name, func(t *testing.T) {
			// Create temporary directory
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project directory: %v", err)
			}

			// Setup project structure
			pattern.setupProject(t, projectDir)

			// Test each handler case
			for _, testCase := range pattern.testCases {
				t.Run(testCase.handler, func(t *testing.T) {
					// Test ProjectResolver can handle this structure
					resolver := NewProjectResolver(projectDir)
					projectInfo, err := resolver.ResolveHandler(testCase.handler)

					if testCase.expectSuccess {
						if err != nil {
							t.Fatalf("Failed to resolve handler %s in %s: %v", testCase.handler, pattern.description, err)
						}

						// Verify the module path contains expected components
						if !strings.Contains(projectInfo.ModulePath, testCase.expectedModule) {
							t.Errorf("Expected module path to contain %s, got %s", testCase.expectedModule, projectInfo.ModulePath)
						}

						// Verify basic properties
						if projectInfo.ProjectRoot != projectDir {
							t.Errorf("Expected project root %s, got %s", projectDir, projectInfo.ProjectRoot)
						}

						if len(projectInfo.PythonPath) == 0 {
							t.Error("Expected non-empty Python path")
						}

						// Test that DependencyAnalyzer works with this structure
						analyzer := NewDependencyAnalyzer(DependencyAnalyzerConfig{
							ProjectResolver: resolver,
						})

						ctx := context.Background()
						analysis, err := analyzer.AnalyzeDependencies(ctx, projectInfo)
						if err != nil {
							t.Errorf("Failed to analyze dependencies: %v", err)
						} else {
							// Should have at least the handler file as a dependency
							if len(analysis.DependencyFiles) == 0 {
								t.Error("Expected at least one dependency file")
							}
						}

						t.Logf("✅ %s/%s: Successfully resolved - module=%s", pattern.description, testCase.handler, projectInfo.ModulePath)
					} else {
						if err == nil {
							t.Errorf("Expected handler resolution to fail for %s in %s but it succeeded", testCase.handler, pattern.description)
						}
						t.Logf("❌ %s/%s: Failed as expected - %v", pattern.description, testCase.handler, err)
					}
				})
			}
		})
	}
}

// TestPoetrySpecificFeatures tests Poetry-specific features and configurations
func TestPoetrySpecificFeatures(t *testing.T) {
	poetryTests := []struct {
		name          string
		description   string
		setupProject  func(t *testing.T, projectDir string)
		handler       string
		expectSuccess bool
	}{
		{
			name:        "poetry_with_groups",
			description: "Poetry project with dependency groups",
			setupProject: func(t *testing.T, projectDir string) {
				pyprojectContent := `
[tool.poetry]
name = "poetry-groups-project"
version = "0.1.0"
description = "Poetry project with dependency groups"

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.31.0"

[tool.poetry.group.dev.dependencies]
pytest = "^7.0.0"
black = "^23.0.0"

[tool.poetry.group.test.dependencies]
coverage = "^7.0.0"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Poetry groups handler'}
`
				handlerPath := filepath.Join(projectDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}
			},
			handler:       "handler.lambda_handler",
			expectSuccess: true,
		},
		{
			name:        "poetry_with_extras",
			description: "Poetry project with optional dependencies",
			setupProject: func(t *testing.T, projectDir string) {
				pyprojectContent := `
[tool.poetry]
name = "poetry-extras-project"
version = "0.1.0"
description = "Poetry project with extras"

[tool.poetry.dependencies]
python = "^3.9"
requests = "^2.31.0"
boto3 = {version = "^1.34.0", optional = true}
psycopg2 = {version = "^2.9.0", optional = true}

[tool.poetry.extras]
aws = ["boto3"]
postgres = ["psycopg2"]
all = ["boto3", "psycopg2"]

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Poetry extras handler'}
`
				handlerPath := filepath.Join(projectDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}
			},
			handler:       "handler.lambda_handler",
			expectSuccess: true,
		},
		{
			name:        "poetry_with_scripts",
			description: "Poetry project with scripts configuration",
			setupProject: func(t *testing.T, projectDir string) {
				pyprojectContent := `
[tool.poetry]
name = "poetry-scripts-project"
version = "0.1.0"
description = "Poetry project with scripts"

[tool.poetry.dependencies]
python = "^3.9"
click = "^8.0.0"

[tool.poetry.scripts]
my-script = "myproject.cli:main"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`
				pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
				if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
					t.Fatalf("Failed to create pyproject.toml: %v", err)
				}

				// Create package structure
				packageDir := filepath.Join(projectDir, "myproject")
				if err := os.MkdirAll(packageDir, 0755); err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}

				initPath := filepath.Join(packageDir, "__init__.py")
				if err := os.WriteFile(initPath, []byte(""), 0644); err != nil {
					t.Fatalf("Failed to create __init__.py: %v", err)
				}

				handlerContent := `
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Poetry scripts handler'}
`
				handlerPath := filepath.Join(packageDir, "handler.py")
				if err := os.WriteFile(handlerPath, []byte(handlerContent), 0644); err != nil {
					t.Fatalf("Failed to create handler: %v", err)
				}

				cliContent := `
import click

@click.command()
def main():
    click.echo("Hello from CLI")
`
				cliPath := filepath.Join(packageDir, "cli.py")
				if err := os.WriteFile(cliPath, []byte(cliContent), 0644); err != nil {
					t.Fatalf("Failed to create CLI: %v", err)
				}
			},
			handler:       "myproject/handler.lambda_handler",
			expectSuccess: true,
		},
	}

	for _, test := range poetryTests {
		t.Run(test.name, func(t *testing.T) {
			// Create temporary directory
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("Failed to create project directory: %v", err)
			}

			// Setup project structure
			test.setupProject(t, projectDir)

			// Test ProjectResolver can handle this structure
			resolver := NewProjectResolver(projectDir)
			projectInfo, err := resolver.ResolveHandler(test.handler)

			if test.expectSuccess {
				if err != nil {
					t.Fatalf("Failed to resolve handler %s in %s: %v", test.handler, test.description, err)
				}

				// Verify basic properties
				if projectInfo.ProjectRoot != projectDir {
					t.Errorf("Expected project root %s, got %s", projectDir, projectInfo.ProjectRoot)
				}

				// Verify pyproject.toml was found
				if projectInfo.PyprojectPath == "" {
					t.Error("Expected pyproject.toml to be found")
				}

				t.Logf("✅ %s: Successfully resolved - module=%s", test.description, projectInfo.ModulePath)
			} else {
				if err == nil {
					t.Errorf("Expected handler resolution to fail for %s but it succeeded", test.description)
				}
				t.Logf("❌ %s: Failed as expected - %v", test.description, err)
			}
		})
	}
}
