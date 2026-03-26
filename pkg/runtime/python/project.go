package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// uvSource represents a UV source configuration
type uvSource struct {
	Path      string `toml:"path"`
	Workspace bool   `toml:"workspace"`
}

// projectInfo contains resolved project information
type projectInfo struct {
	ProjectRoot   string
	SourceRoot    string
	PyprojectPath string
}

// pyprojectConfig represents the structure of a pyproject.toml file.
type pyprojectConfig struct {
	Project struct {
		Name string `toml:"name"`
	} `toml:"project"`

	Tool struct {
		UV struct {
			Sources   map[string]uvSource `toml:"sources"`
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
	} `toml:"tool"`
}

// resolveHandler finds and resolves a Python handler.
func resolveHandler(projectRoot, handlerPath string) (*projectInfo, error) {
	handlerFile, err := findPythonFile(projectRoot, handlerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Python file for handler %s: %w", handlerPath, err)
	}

	pyprojectPath, _ := findPyprojectToml(projectRoot, handlerFile)

	info := &projectInfo{
		ProjectRoot:   projectRoot,
		PyprojectPath: pyprojectPath,
	}

	info.SourceRoot = resolveSourceRoot(projectRoot, pyprojectPath)

	return info, nil
}

// findPythonFile locates the Python file for the given handler path.
func findPythonFile(projectRoot, handlerPath string) (string, error) {
	filePath := extractFilePath(handlerPath)
	candidates := generateCandidatePaths(projectRoot, filePath)

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && strings.HasSuffix(candidate, ".py") {
			absPath, err := filepath.Abs(candidate)
			if err != nil {
				return "", fmt.Errorf("failed to get absolute path for %s: %w", candidate, err)
			}
			return absPath, nil
		}
	}

	return "", fmt.Errorf("handler not found: %s (searched %d candidate paths)", handlerPath, len(candidates))
}

// extractFilePath extracts the file path from a handler path.
func extractFilePath(handlerPath string) string {
	if lastDot := strings.LastIndex(handlerPath, "."); lastDot != -1 {
		return handlerPath[:lastDot]
	}
	return handlerPath
}

// generateCandidatePaths creates a list of potential file locations.
func generateCandidatePaths(projectRoot, handlerPath string) []string {
	var candidates []string

	// Direct path and with .py extension
	candidates = append(candidates, filepath.Join(projectRoot, handlerPath))
	if !strings.HasSuffix(handlerPath, ".py") {
		candidates = append(candidates, filepath.Join(projectRoot, handlerPath+".py"))
	}

	// Common Python project directories
	for _, dir := range []string{"src", "app", "functions", "lambda", "handlers", "lib"} {
		candidates = append(candidates, filepath.Join(projectRoot, dir, handlerPath))
		if !strings.HasSuffix(handlerPath, ".py") {
			candidates = append(candidates, filepath.Join(projectRoot, dir, handlerPath+".py"))
		}
	}

	// Nested paths with common source directories
	if strings.Contains(handlerPath, "/") {
		dir := filepath.Dir(handlerPath)
		base := filepath.Base(handlerPath)
		for _, commonDir := range []string{"src", "app", "functions", "lambda", "handlers", "lib"} {
			candidates = append(candidates, filepath.Join(projectRoot, commonDir, dir, base))
			if !strings.HasSuffix(base, ".py") {
				candidates = append(candidates, filepath.Join(projectRoot, commonDir, dir, base+".py"))
			}
		}
	}

	return candidates
}

// findPyprojectToml searches for pyproject.toml starting from the handler file directory.
func findPyprojectToml(projectRoot, handlerFile string) (string, error) {
	currentDir := filepath.Dir(handlerFile)
	for {
		pyprojectPath := filepath.Join(currentDir, "pyproject.toml")
		if _, err := os.Stat(pyprojectPath); err == nil {
			return pyprojectPath, nil
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir || !strings.HasPrefix(currentDir, projectRoot) {
			break
		}
		currentDir = parentDir
	}
	return "", fmt.Errorf("no pyproject.toml found")
}

// resolveSourceRoot determines the source root directory.
func resolveSourceRoot(projectRoot, pyprojectPath string) string {
	if pyprojectPath != "" {
		pyprojectDir := filepath.Dir(pyprojectPath)
		srcDir := filepath.Join(pyprojectDir, "src")
		if _, err := os.Stat(srcDir); err == nil {
			return srcDir
		}
		return pyprojectDir
	}
	return projectRoot
}

// parsePyprojectToml reads and parses a pyproject.toml file.
func parsePyprojectToml(path string) (*pyprojectConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read pyproject.toml at %s: %w", path, err)
	}

	var config pyprojectConfig
	if err := toml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("TOML parsing error in %s: %w", path, err)
	}

	if config.Project.Name == "" && config.Tool.Poetry.Name == "" {
		return nil, fmt.Errorf("pyproject.toml at %s must have a name in [project] or [tool.poetry]", path)
	}

	return &config, nil
}
