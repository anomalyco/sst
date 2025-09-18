package python

import "time"

// Common file names and patterns used throughout the Python runtime
const (
	// Configuration files
	PyprojectTomlFile   = "pyproject.toml"
	RequirementsTxtFile = "requirements.txt"
	UvLockFile          = "uv.lock"
	PoetryLockFile      = "poetry.lock"
	PipfileLockFile     = "Pipfile.lock"
	SetupPyFile         = "setup.py"

	// Python file extensions and patterns
	PythonFileExt = ".py"
	PycFileExt    = ".pyc"
	PyoFileExt    = ".pyo"
	PydFileExt    = ".pyd"

	// Common directory names
	PycacheDir     = "__pycache__"
	SstDir         = ".sst"
	SstCacheDir    = ".sst/cache"
	GitDir         = ".git"
	VenvDir        = ".venv"
	VenvAltDir     = "venv"
	NodeModulesDir = "node_modules"
	PytestCacheDir = ".pytest_cache"

	// File permissions
	DefaultFileMode = 0644
	DefaultDirMode  = 0755
)

// Default timeout and size values
//
// TODO: Move all caches to content-based invalidation
// Currently these use time-based expiration, but they should watch file changes:
// - DependencyCache should watch: pyproject.toml, uv.lock, requirements.txt
// - UvCommandRunner should watch: input files that affect commands
const (

	// LEGACY: Dependency cache age (should be content-based)
	DefaultDependencyCacheAge = 24 * time.Hour

	// LEGACY: UV command cache timeout (should be content-based)
	DefaultUvCommandCacheTimeout = 10 * time.Minute

	// Retry timeouts
	DefaultRetryDelay        = 1 * time.Second
	DefaultMaxRetryDelay     = 30 * time.Second
	NetworkRetryDelay        = 30 * time.Second
	DependencyRetryDelay     = 60 * time.Second
	TransientErrorRetryDelay = 5 * time.Second

	// Size limits
	DefaultMaxCacheSize      = 1024 * 1024 * 1024 // 1GB
	DefaultMaxParallelBuilds = 4
	DefaultMaxRetries        = 3

	// Size formatting
	BytesPerMB = 1024 * 1024
)

// Common file patterns for different purposes
var (
	// Dependency files that trigger rebuilds
	DependencyFiles = []string{
		PyprojectTomlFile,
		UvLockFile,
		RequirementsTxtFile,
		PoetryLockFile,
		PipfileLockFile,
	}

	// Files to exclude from builds
	ExcludePatterns = []string{
		PycacheDir + "/*",
		"*" + PycFileExt,
		"*" + PyoFileExt,
		"*" + PydFileExt,
		SstDir,
		GitDir,
		PytestCacheDir,
		NodeModulesDir,
		VenvDir,
		VenvAltDir,
		".DS_Store",
		".env",
		"*.dist-info",
		"*.egg-info",
	}

	// Watch patterns for change detection
	WatchPatterns = []string{
		"*" + PythonFileExt,
		PyprojectTomlFile,
		UvLockFile,
		RequirementsTxtFile,
		PoetryLockFile,
		PipfileLockFile,
	}

	// Directories to skip when creating __init__.py files
	SkipInitPyDirs = []string{
		PycacheDir,
		PytestCacheDir,
		GitDir,
		SstDir,
		NodeModulesDir,
		VenvDir,
		VenvAltDir,
		"env",
		".env",
		"tests",
		"test",
	}
)
