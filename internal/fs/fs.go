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

func FindDown(dir, filename string) []string {
	return FindDownWithExcludes(dir, filename, []string{})
}

func FindDownWithExcludes(dir, filename string, excludePatterns []string) []string {
	var result []string

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite the error
		}
		if info.IsDir() {
			name := info.Name()
			// Always exclude node_modules and dot directories
			if name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			// Check custom exclude patterns
			if shouldExclude(name, excludePatterns) {
				return filepath.SkipDir
			}
		}
		if !info.IsDir() && info.Name() == filename {
			result = append(result, path)
		}
		return nil
	})

	return result
}

// shouldExclude checks if a directory name matches any of the exclude patterns
func shouldExclude(name string, patterns []string) bool {
	for _, pattern := range patterns {
		// Support glob patterns
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			// If pattern is invalid, try exact match as fallback
			if name == pattern {
				return true
			}
			continue
		}
		if matched {
			return true
		}
	}
	return false
}
