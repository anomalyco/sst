package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// UVSource represents a UV source configuration
type UVSource struct {
	Path      string `toml:"path"`
	URL       string `toml:"url"`
	Git       string `toml:"git"`
	Workspace bool   `toml:"workspace"` // For workspace members: { workspace = true }
}

// ProjectResolver provides simplified Python project resolution without layout classification
type ProjectResolver struct {
	projectRoot string
	cache       map[string]*ProjectInfo
	mutex       sync.RWMutex
}

// ProjectInfo contains resolved project information without layout type classification
type ProjectInfo struct {
	// HandlerFile is the absolute path to the Python handler file
	HandlerFile string `json:"handlerFile"`

	// ProjectRoot is the root directory of the project
	ProjectRoot string `json:"projectRoot"`

	// SourceRoot is the directory where Python source files are located
	SourceRoot string `json:"sourceRoot"`

	// PythonPath contains Python import paths for module resolution
	PythonPath []string `json:"pythonPath"`

	// ModulePath is the Python import path for the handler
	ModulePath string `json:"modulePath"`

	// Dependencies contains files that affect the build
	Dependencies []string `json:"dependencies"`

	// PyprojectPath is the path to pyproject.toml if present
	PyprojectPath string `json:"pyprojectPath"`

	// ResolvedAt is when this project was resolved (for cache invalidation)
	ResolvedAt time.Time `json:"resolvedAt"`
}

// PyprojectConfig represents the structure of a pyproject.toml file
// This structure is designed to be permissive and handle various Python project formats
type PyprojectConfig struct {
	// Standard PEP 621 project metadata
	Project struct {
		Name         string   `toml:"name"`
		Dependencies []string `toml:"dependencies"`
	} `toml:"project"`

	// Tool-specific configurations
	Tool struct {
		// UV package manager configuration
		UV struct {
			Sources   map[string]UVSource `toml:"sources"`
			Workspace struct {
				Members []string `toml:"members"`
			} `toml:"workspace"`
		} `toml:"uv"`

		// Poetry configuration (legacy support)
		Poetry struct {
			Name string `toml:"name"`
		} `toml:"poetry"`

		// Hatch configuration
		Hatch struct {
			Build struct {
				Targets struct {
					Wheel struct {
						Packages []string `toml:"packages"`
					} `toml:"wheel"`
				} `toml:"targets"`
			} `toml:"build"`
		} `toml:"hatch"`

		// Setuptools configuration
		Setuptools struct {
			Packages struct {
				Find struct {
					Where []string `toml:"where"`
				} `toml:"find"`
			} `toml:"packages"`
		} `toml:"setuptools"`

		// SST configuration for file inclusion/exclusion and Lambda runtime options
		SST struct {
			Include []string `toml:"include"`
			Exclude []string `toml:"exclude"`
			// IncludeLambdaRuntime when true, includes boto3/botocore in the Lambda package
			// By default (false), these are excluded since Lambda provides them
			IncludeLambdaRuntime bool `toml:"include-lambda-runtime"`
		} `toml:"sst"`
	} `toml:"tool"`
}

// NewProjectResolver creates a new project resolver
func NewProjectResolver(projectRoot string) *ProjectResolver {
	return &ProjectResolver{
		projectRoot: projectRoot,
		cache:       make(map[string]*ProjectInfo),
	}
}

// ResolveHandler finds and resolves a Python handler without layout classification
func (pr *ProjectResolver) ResolveHandler(handlerPath string) (*ProjectInfo, error) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	// Check cache first
	if cached, exists := pr.cache[handlerPath]; exists {
		// Simple cache without expiration for now - can be enhanced later
		return cached, nil
	}

	// Find the actual Python file
	handlerFile, err := pr.findPythonFile(handlerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Python file for handler %s: %w", handlerPath, err)
	}

	// Find pyproject.toml if it exists
	pyprojectPath, err := pr.findPyprojectToml(handlerFile)
	if err != nil {
		// pyproject.toml is optional, continue without it
		pyprojectPath = ""
	}

	// Create project info
	projectInfo := &ProjectInfo{
		HandlerFile:   handlerFile,
		ProjectRoot:   pr.projectRoot,
		PyprojectPath: pyprojectPath,
		Dependencies:  []string{},
		PythonPath:    []string{},
		ResolvedAt:    time.Now(),
	}

	// Determine source root and Python paths
	if err := pr.setupSourceRoot(projectInfo); err != nil {
		return nil, fmt.Errorf("failed to setup source root: %w", err)
	}

	// Setup Python paths using standard resolution rules
	if err := pr.setupPythonPaths(projectInfo); err != nil {
		return nil, fmt.Errorf("failed to setup Python paths: %w", err)
	}

	// Generate module path for the handler
	if err := pr.generateModulePath(projectInfo); err != nil {
		return nil, fmt.Errorf("failed to generate module path: %w", err)
	}

	// Add dependencies
	pr.addDependencies(projectInfo)

	// Cache the result
	pr.cache[handlerPath] = projectInfo

	return projectInfo, nil
}

// findPythonFile locates the Python file for the given handler path
func (pr *ProjectResolver) findPythonFile(handlerPath string) (string, error) {
	// Extract the file path from handler path (remove function name)
	// Handler format: "path/to/file.function_name" -> "path/to/file"
	filePath := pr.extractFilePath(handlerPath)

	// Generate candidate paths
	candidates := pr.generateCandidatePaths(filePath)

	// Try each candidate
	for _, candidate := range candidates {
		if pr.isValidPythonFile(candidate) {
			absPath, err := filepath.Abs(candidate)
			if err != nil {
				return "", fmt.Errorf("failed to get absolute path for %s: %w", candidate, err)
			}
			return absPath, nil
		}
	}

	// If no direct match, create a helpful error
	suggestions := GenerateHandlerSuggestions(handlerPath, "", candidates)
	return "", NewHandlerNotFoundError(handlerPath, candidates, suggestions)
}

// extractFilePath extracts the file path from a handler path
// Handler format: "path/to/file.function_name" -> "path/to/file"
func (pr *ProjectResolver) extractFilePath(handlerPath string) string {
	// Find the last dot to separate file path from function name
	lastDot := strings.LastIndex(handlerPath, ".")
	if lastDot == -1 {
		// No function name specified, use the whole path
		return handlerPath
	}

	// Return everything before the last dot
	return handlerPath[:lastDot]
}

// generateCandidatePaths creates a list of potential file locations
func (pr *ProjectResolver) generateCandidatePaths(handlerPath string) []string {
	candidates := []string{}

	// 1. Direct path as specified
	candidates = append(candidates, filepath.Join(pr.projectRoot, handlerPath))

	// 2. Add .py extension if not present
	if !strings.HasSuffix(handlerPath, ".py") {
		candidates = append(candidates, filepath.Join(pr.projectRoot, handlerPath+".py"))
	}

	// 3. Common Python project directories
	commonDirs := []string{"src", "app", "functions", "lambda", "handlers", "lib"}
	for _, dir := range commonDirs {
		candidates = append(candidates, filepath.Join(pr.projectRoot, dir, handlerPath))
		if !strings.HasSuffix(handlerPath, ".py") {
			candidates = append(candidates, filepath.Join(pr.projectRoot, dir, handlerPath+".py"))
		}
	}

	// 4. Handle nested paths
	if strings.Contains(handlerPath, "/") {
		dir := filepath.Dir(handlerPath)
		base := filepath.Base(handlerPath)

		// Try with common source directories
		for _, commonDir := range commonDirs {
			candidates = append(candidates, filepath.Join(pr.projectRoot, commonDir, dir, base))
			if !strings.HasSuffix(base, ".py") {
				candidates = append(candidates, filepath.Join(pr.projectRoot, commonDir, dir, base+".py"))
			}
		}
	}

	return candidates
}

// isValidPythonFile checks if a file is a valid Python file
func (pr *ProjectResolver) isValidPythonFile(path string) bool {
	// Check file extension
	if !strings.HasSuffix(path, ".py") {
		return false
	}

	// Check if file exists and is readable
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Skip directories and non-regular files
	return info.Mode().IsRegular()
}

// findPyprojectToml searches for pyproject.toml starting from the handler file directory
func (pr *ProjectResolver) findPyprojectToml(handlerFile string) (string, error) {
	currentDir := filepath.Dir(handlerFile)

	for {
		pyprojectPath := filepath.Join(currentDir, "pyproject.toml")
		if _, err := os.Stat(pyprojectPath); err == nil {
			return pyprojectPath, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)

		// Stop if we've reached the project root or can't go up further
		if parentDir == currentDir || !strings.HasPrefix(currentDir, pr.projectRoot) {
			break
		}

		currentDir = parentDir
	}

	return "", fmt.Errorf("no pyproject.toml found")
}

// setupSourceRoot determines the source root directory
func (pr *ProjectResolver) setupSourceRoot(projectInfo *ProjectInfo) error {
	// If we have a pyproject.toml, use its directory as a starting point
	if projectInfo.PyprojectPath != "" {
		pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)

		// Check for common source directories
		srcDir := filepath.Join(pyprojectDir, "src")
		if _, err := os.Stat(srcDir); err == nil {
			projectInfo.SourceRoot = srcDir
			return nil
		}

		// Use pyproject.toml directory as source root
		projectInfo.SourceRoot = pyprojectDir
		return nil
	}

	// Fallback: use the project root as source root to preserve directory structure
	projectInfo.SourceRoot = pr.projectRoot
	return nil
}

// setupPythonPaths configures Python paths using standard resolution rules
func (pr *ProjectResolver) setupPythonPaths(projectInfo *ProjectInfo) error {
	// Add source root to Python path
	projectInfo.PythonPath = append(projectInfo.PythonPath, projectInfo.SourceRoot)

	// If we have a pyproject.toml, add its directory if different from source root
	if projectInfo.PyprojectPath != "" {
		pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)
		if pyprojectDir != projectInfo.SourceRoot {
			projectInfo.PythonPath = append(projectInfo.PythonPath, pyprojectDir)
		}
	}

	// Add project root if different from source root
	if pr.projectRoot != projectInfo.SourceRoot {
		projectInfo.PythonPath = append(projectInfo.PythonPath, pr.projectRoot)
	}

	return nil
}

// generateModulePath creates the Python import path for the handler
func (pr *ProjectResolver) generateModulePath(projectInfo *ProjectInfo) error {
	// Calculate relative path from source root to handler file
	relPath, err := filepath.Rel(projectInfo.SourceRoot, projectInfo.HandlerFile)
	if err != nil {
		return fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Convert file path to Python module path
	modulePath := strings.TrimSuffix(relPath, ".py")
	modulePath = strings.ReplaceAll(modulePath, string(filepath.Separator), ".")

	projectInfo.ModulePath = modulePath
	return nil
}

// addDependencies adds relevant dependency files to the project info
func (pr *ProjectResolver) addDependencies(projectInfo *ProjectInfo) {
	// Add pyproject.toml if present
	if projectInfo.PyprojectPath != "" {
		projectInfo.Dependencies = append(projectInfo.Dependencies, projectInfo.PyprojectPath)

		// Add uv.lock if present
		uvLockPath := filepath.Join(filepath.Dir(projectInfo.PyprojectPath), "uv.lock")
		if _, err := os.Stat(uvLockPath); err == nil {
			projectInfo.Dependencies = append(projectInfo.Dependencies, uvLockPath)
		}

		// Add poetry.lock if present
		poetryLockPath := filepath.Join(filepath.Dir(projectInfo.PyprojectPath), "poetry.lock")
		if _, err := os.Stat(poetryLockPath); err == nil {
			projectInfo.Dependencies = append(projectInfo.Dependencies, poetryLockPath)
		}
	}

	// Add requirements.txt if present
	requirementsPath := filepath.Join(pr.projectRoot, "requirements.txt")
	if _, err := os.Stat(requirementsPath); err == nil {
		projectInfo.Dependencies = append(projectInfo.Dependencies, requirementsPath)
	}
}

// ParsePyprojectToml reads and parses a pyproject.toml file with permissive error handling
func (pr *ProjectResolver) ParsePyprojectToml(path string) (*PyprojectConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, NewConfigurationError(fmt.Sprintf("failed to read pyproject.toml: %v", err), path, "file read error", "ensure the file exists and is readable")
	}

	var config PyprojectConfig
	if err := toml.Unmarshal(content, &config); err != nil {
		// Try to provide helpful error messages for common TOML issues
		errorMsg := err.Error()
		var fix string

		if strings.Contains(errorMsg, "expected") && strings.Contains(errorMsg, "but got") {
			fix = "check TOML syntax - ensure proper quoting of strings and correct data types"
		} else if strings.Contains(errorMsg, "duplicate") {
			fix = "remove duplicate keys in the TOML file"
		} else if strings.Contains(errorMsg, "invalid") {
			fix = "validate TOML syntax using a TOML validator"
		} else {
			fix = "ensure the pyproject.toml file follows standard TOML format"
		}

		return nil, NewConfigurationError(fmt.Sprintf("TOML parsing error: %s", errorMsg), path, "TOML syntax error", fix)
	}

	// Validate that we have at least some project information
	if err := pr.validatePyprojectConfig(&config); err != nil {
		return nil, NewConfigurationError(err.Error(), path, "missing project name", "ensure either [project] or [tool.poetry] section has a name field")
	}

	return &config, nil
}

// validatePyprojectConfig performs basic validation on the parsed configuration
func (pr *ProjectResolver) validatePyprojectConfig(config *PyprojectConfig) error {
	// Check if we have a project name from either standard [project] or legacy [tool.poetry]
	hasProjectName := config.Project.Name != ""
	hasPoetryName := config.Tool.Poetry.Name != ""

	if !hasProjectName && !hasPoetryName {
		return fmt.Errorf("project must have a name in either [project] or [tool.poetry] section")
	}

	// Validate UV sources if present
	for sourceName, source := range config.Tool.UV.Sources {
		if err := pr.validateUVSource(sourceName, source); err != nil {
			return fmt.Errorf("invalid UV source '%s': %w", sourceName, err)
		}
	}

	return nil
}

// validateUVSource validates a UV source configuration
func (pr *ProjectResolver) validateUVSource(sourceName string, source UVSource) error {
	// Count the number of source types specified
	sourceTypes := 0
	if source.Path != "" {
		sourceTypes++
	}
	if source.URL != "" {
		sourceTypes++
	}
	if source.Git != "" {
		sourceTypes++
	}
	if source.Workspace {
		sourceTypes++
	}

	if sourceTypes == 0 {
		return fmt.Errorf("no source specified (must have path, url, git, or workspace)")
	}
	if sourceTypes > 1 {
		return fmt.Errorf("multiple source types specified (use only one of path, url, git, or workspace)")
	}

	// For local path sources, we could validate they exist, but we'll be permissive
	// and let the build process handle missing dependencies

	return nil
}

// ClearCache removes all cached project information
func (pr *ProjectResolver) ClearCache() {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()
	pr.cache = make(map[string]*ProjectInfo)
}
