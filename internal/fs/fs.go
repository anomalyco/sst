package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FindUp(initialPath, fileName string) (string, error) {
	currentDir := initialPath
	for {
		// Check if the current directory contains the target file
		filePath := filepath.Join(currentDir, fileName)
		_, err := os.Stat(filePath)
		if err == nil {
			// File found
			return filePath, nil
		}

		// If we've reached the root directory, stop searching
		if currentDir == filepath.Dir(currentDir) {
			return "", fmt.Errorf("File '%s' not found", fileName)
		}

		// Move up to the parent directory
		currentDir = filepath.Dir(currentDir)
	}
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func IsGitSubmodule(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	return err == nil && !info.IsDir()
}

func FindDown(dir, filename string) []string {
	return FindDownWithIgnore(dir, filename, nil)
}

func FindDownWithIgnore(dir, filename string, ignore []string) []string {
	var result []string
	ignored := make([]string, 0, len(ignore))
	for _, item := range ignore {
		if item == "" {
			continue
		}
		if filepath.IsAbs(item) {
			ignored = append(ignored, filepath.Clean(item))
			continue
		}
		ignored = append(ignored, filepath.Clean(filepath.Join(dir, item)))
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite the error
		}
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if path != dir && IsGitSubmodule(path) {
				return filepath.SkipDir
			}
			for _, ignoredPath := range ignored {
				if isWithin(ignoredPath, path) {
					return filepath.SkipDir
				}
			}
		}
		if !info.IsDir() && info.Name() == filename {
			result = append(result, path)
		}
		return nil
	})

	return result
}

func isWithin(parent, path string) bool {
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
