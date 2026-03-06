package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// UVSource represents a UV source configuration
type UVSource struct {
	Path      string `toml:"path"`
	URL       string `toml:"url"`
	Git       string `toml:"git"`
	Workspace bool   `toml:"workspace"`
}

// ProjectResolver provides Python project resolution
type ProjectResolver struct {
	projectRoot string
}

// ProjectInfo contains resolved project information
type ProjectInfo struct {
	HandlerFile   string   `json:"handlerFile"`
	ProjectRoot   string   `json:"projectRoot"`
	SourceRoot    string   `json:"sourceRoot"`
	PythonPath    []string `json:"pythonPath"`
	ModulePath    string   `json:"modulePath"`
	PyprojectPath string   `json:"pyprojectPath"`
}

// PyprojectConfig represents the structure of a pyproject.toml file.
// Only fields actually used by the runtime are included.
type PyprojectConfig struct {
	Project struct {
		Name string `toml:"name"`
	} `toml:"project"`

	Tool struct {
		UV struct {
			Sources   map[string]UVSource `toml:"sources"`
			Workspace struct {
				Members []string `toml:"members"`
			} `toml:"workspace"`
		} `toml:"uv"`

		Poetry struct {
			Name string `toml:"name"`
		} `toml:"poetry"`

		Hatch struct {
			Build struct {
				Targets struct {
					Wheel struct {
						Packages []string `toml:"packages"`
					} `toml:"wheel"`
				} `toml:"targets"`
			} `toml:"build"`
		} `toml:"hatch"`

		Setuptools struct {
			Packages struct {
				Find struct {
					Where []string `toml:"where"`
				} `toml:"find"`
			} `toml:"packages"`
		} `toml:"setuptools"`

		SST struct {
			Include              []string `toml:"include"`
			Exclude              []string `toml:"exclude"`
			IncludeLambdaRuntime bool     `toml:"include-lambda-runtime"`
		} `toml:"sst"`
	} `toml:"tool"`
}

// NewProjectResolver creates a new project resolver
func NewProjectResolver(projectRoot string) *ProjectResolver {
	return &ProjectResolver{projectRoot: projectRoot}
}

// ResolveHandler finds and resolves a Python handler
func (pr *ProjectResolver) ResolveHandler(handlerPath string) (*ProjectInfo, error) {
	handlerFile, err := pr.findPythonFile(handlerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Python file for handler %s: %w", handlerPath, err)
	}

	pyprojectPath, _ := pr.findPyprojectToml(handlerFile)

	info := &ProjectInfo{
		HandlerFile:   handlerFile,
		ProjectRoot:   pr.projectRoot,
		PyprojectPath: pyprojectPath,
	}

	pr.setupSourceRoot(info)
	pr.setupPythonPaths(info)

	if err := pr.generateModulePath(info); err != nil {
		return nil, fmt.Errorf("failed to generate module path: %w", err)
	}

	return info, nil
}

// findPythonFile locates the Python file for the given handler path
func (pr *ProjectResolver) findPythonFile(handlerPath string) (string, error) {
	filePath := pr.extractFilePath(handlerPath)
	candidates := pr.generateCandidatePaths(filePath)

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && strings.HasSuffix(candidate, ".py") {
			absPath, err := filepath.Abs(candidate)
			if err != nil {
				return "", fmt.Errorf("failed to get absolute path for %s: %w", candidate, err)
			}
			return absPath, nil
		}
	}

	suggestions := GenerateHandlerSuggestions(handlerPath, "", candidates)
	return "", NewHandlerNotFoundError(handlerPath, candidates, suggestions)
}

// extractFilePath extracts the file path from a handler path
// Handler format: "path/to/file.function_name" -> "path/to/file"
func (pr *ProjectResolver) extractFilePath(handlerPath string) string {
	if lastDot := strings.LastIndex(handlerPath, "."); lastDot != -1 {
		return handlerPath[:lastDot]
	}
	return handlerPath
}

// generateCandidatePaths creates a list of potential file locations
func (pr *ProjectResolver) generateCandidatePaths(handlerPath string) []string {
	var candidates []string

	// Direct path and with .py extension
	candidates = append(candidates, filepath.Join(pr.projectRoot, handlerPath))
	if !strings.HasSuffix(handlerPath, ".py") {
		candidates = append(candidates, filepath.Join(pr.projectRoot, handlerPath+".py"))
	}

	// Common Python project directories
	for _, dir := range []string{"src", "app", "functions", "lambda", "handlers", "lib"} {
		candidates = append(candidates, filepath.Join(pr.projectRoot, dir, handlerPath))
		if !strings.HasSuffix(handlerPath, ".py") {
			candidates = append(candidates, filepath.Join(pr.projectRoot, dir, handlerPath+".py"))
		}
	}

	// Nested paths with common source directories
	if strings.Contains(handlerPath, "/") {
		dir := filepath.Dir(handlerPath)
		base := filepath.Base(handlerPath)
		for _, commonDir := range []string{"src", "app", "functions", "lambda", "handlers", "lib"} {
			candidates = append(candidates, filepath.Join(pr.projectRoot, commonDir, dir, base))
			if !strings.HasSuffix(base, ".py") {
				candidates = append(candidates, filepath.Join(pr.projectRoot, commonDir, dir, base+".py"))
			}
		}
	}

	return candidates
}

// findPyprojectToml searches for pyproject.toml starting from the handler file directory
func (pr *ProjectResolver) findPyprojectToml(handlerFile string) (string, error) {
	currentDir := filepath.Dir(handlerFile)
	for {
		pyprojectPath := filepath.Join(currentDir, "pyproject.toml")
		if _, err := os.Stat(pyprojectPath); err == nil {
			return pyprojectPath, nil
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir || !strings.HasPrefix(currentDir, pr.projectRoot) {
			break
		}
		currentDir = parentDir
	}
	return "", fmt.Errorf("no pyproject.toml found")
}

// setupSourceRoot determines the source root directory
func (pr *ProjectResolver) setupSourceRoot(info *ProjectInfo) {
	if info.PyprojectPath != "" {
		pyprojectDir := filepath.Dir(info.PyprojectPath)
		srcDir := filepath.Join(pyprojectDir, "src")
		if _, err := os.Stat(srcDir); err == nil {
			info.SourceRoot = srcDir
			return
		}
		info.SourceRoot = pyprojectDir
		return
	}
	info.SourceRoot = pr.projectRoot
}

// setupPythonPaths configures Python paths for module resolution
func (pr *ProjectResolver) setupPythonPaths(info *ProjectInfo) {
	info.PythonPath = append(info.PythonPath, info.SourceRoot)
	if info.PyprojectPath != "" {
		if pyprojectDir := filepath.Dir(info.PyprojectPath); pyprojectDir != info.SourceRoot {
			info.PythonPath = append(info.PythonPath, pyprojectDir)
		}
	}
	if pr.projectRoot != info.SourceRoot {
		info.PythonPath = append(info.PythonPath, pr.projectRoot)
	}
}

// generateModulePath creates the Python import path for the handler
func (pr *ProjectResolver) generateModulePath(info *ProjectInfo) error {
	relPath, err := filepath.Rel(info.SourceRoot, info.HandlerFile)
	if err != nil {
		return fmt.Errorf("failed to calculate relative path: %w", err)
	}
	modulePath := strings.TrimSuffix(relPath, ".py")
	info.ModulePath = strings.ReplaceAll(modulePath, string(filepath.Separator), ".")
	return nil
}

// ParsePyprojectToml reads and parses a pyproject.toml file
func (pr *ProjectResolver) ParsePyprojectToml(path string) (*PyprojectConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, NewConfigurationError(
			fmt.Sprintf("failed to read pyproject.toml: %v", err),
			path, "file read error", "ensure the file exists and is readable",
		)
	}

	var config PyprojectConfig
	if err := toml.Unmarshal(content, &config); err != nil {
		return nil, NewConfigurationError(
			fmt.Sprintf("TOML parsing error: %s", err.Error()),
			path, "TOML syntax error", "ensure the pyproject.toml file follows standard TOML format",
		)
	}

	if config.Project.Name == "" && config.Tool.Poetry.Name == "" {
		return nil, NewConfigurationError(
			"project must have a name in either [project] or [tool.poetry] section",
			path, "missing project name",
			"ensure either [project] or [tool.poetry] section has a name field",
		)
	}

	return &config, nil
}
