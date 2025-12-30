package python

import (
	"testing"
	"time"
)

func TestInfrastructureFileChanges(t *testing.T) {
	runtime := &PythonRuntime{}

	testCases := []struct {
		filename string
		expected bool
		desc     string
	}{
		// Python files (should trigger rebuild)
		{"handler.py", true, "Python files should trigger rebuild"},
		{"lib/utils.py", true, "Python files in subdirs should trigger rebuild"},

		// Infrastructure files (should trigger rebuild)
		{"sst.config.ts", true, "SST config should trigger rebuild"},
		{"sst.config.js", true, "SST config JS should trigger rebuild"},
		{"package.json", true, "package.json should trigger rebuild"},
		{"infra/api.ts", true, "TypeScript files should trigger rebuild"},
		{"components/database.js", true, "JavaScript files should trigger rebuild"},
		{"lib/shared.mjs", true, "ES modules should trigger rebuild"},

		// Python dependency files (should trigger rebuild)
		{"requirements.txt", true, "requirements.txt should trigger rebuild"},
		{"pyproject.toml", true, "pyproject.toml should trigger rebuild"},
		{"uv.lock", true, "uv.lock should trigger rebuild"},

		// Non-relevant files (should NOT trigger rebuild)
		{"README.md", false, "Markdown files should not trigger rebuild"},
		{"docs/guide.html", false, "HTML files should not trigger rebuild"},
		{".gitignore", false, "Git files should not trigger rebuild"},
		{"image.png", false, "Image files should not trigger rebuild"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := runtime.isRelevantFile(tc.filename)
			if result != tc.expected {
				t.Errorf("isRelevantFile(%q) = %v, expected %v", tc.filename, result, tc.expected)
			}
		})
	}
}

func TestShouldRebuildInfrastructureChanges(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
		desc     string
	}{
		// Infrastructure files should NOT trigger immediate rebuild (lazy rebuild on invoke)
		{"sst.config.ts", false, "SST config changes should NOT trigger immediate rebuild"},
		{"infra/api.ts", false, "Infrastructure TypeScript should NOT trigger immediate rebuild"},
		{"components/auth.js", false, "Infrastructure JavaScript should NOT trigger immediate rebuild"},
		{"package.json", false, "package.json changes should NOT trigger immediate rebuild"},

		// Python files should trigger rebuild (worker restart)
		{"handler.py", true, "Python files should trigger rebuild"},
		{"lib/models.py", true, "Python modules should trigger rebuild"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Create fresh runtime for each test to avoid rate limiting interference
			runtime := &PythonRuntime{
				lastRebuildCheck: make(map[string]time.Time),
				rebuildCooldown:  time.Millisecond * 1, // Very short cooldown for testing
				pendingChanges:   make(map[string][]string),
				restartSignaled:  make(map[string]bool),
				fileChangeTime:   make(map[string]time.Time),
				lastWorkerStart:  make(map[string]time.Time),
			}
			result := runtime.ShouldRebuild("test-function", tc.filename)
			if result != tc.expected {
				t.Errorf("ShouldRebuild('test-function', %q) = %v, expected %v", tc.filename, result, tc.expected)
			}
		})
	}
}
