package python

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sst/sst/v3/cmd/sst/mosaic/ui/common"
	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/events"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/path"
	"github.com/sst/sst/v3/pkg/runtime"
)

type Worker struct {
	stdout io.ReadCloser
	stderr io.ReadCloser
	cmd    *exec.Cmd
}

func (w *Worker) Stop() {
	// Terminate the whole process group
	process.Kill(w.cmd.Process)
}

func (w *Worker) Logs() io.ReadCloser {
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		var wg sync.WaitGroup
		wg.Add(2)

		copyStream := func(dst io.Writer, src io.Reader, name string) {
			defer wg.Done()
			buf := make([]byte, 1024)
			for {
				n, err := src.Read(buf)
				if n > 0 {
					_, werr := dst.Write(buf[:n])
					if werr != nil {
						slog.Error("error writing to pipe", "stream", name, "err", werr)
						return
					}
				}
				if err != nil {
					if err != io.EOF {
						slog.Error("error reading from stream", "stream", name, "err", err)
					}
					return
				}
			}
		}

		go copyStream(writer, w.stdout, "stdout")
		go copyStream(writer, w.stderr, "stderr")

		wg.Wait()
	}()

	return reader
}

type PythonRuntime struct {
	lastBuiltHandler map[string]string

	// Build cache and change detection components
	buildCache      *BuildCache
	changeDetector  *ChangeDetector
	projectResolver *ProjectResolver

	// Cache directory for sharing with incremental builder
	cacheDir string

	// Rate limiting for rebuild checks
	lastRebuildCheck map[string]time.Time
	rebuildCooldown  time.Duration

	// Mutex for thread-safe access
	mutex sync.RWMutex
}

// FunctionLogEvent represents a function log event (matches the AWS function log event)
type FunctionLogEvent struct {
	FunctionID string `json:"functionID"`
	WorkerID   string `json:"workerID"`
	RequestID  string `json:"requestID"`
	Line       string `json:"line"`
}

func New() *PythonRuntime {
	return &PythonRuntime{
		lastBuiltHandler: map[string]string{},
		lastRebuildCheck: map[string]time.Time{},
		rebuildCooldown:  10 * time.Second, // Prevent rebuilds more than once every 10 seconds
	}
}

// NewWithCache creates a new Python runtime with caching enabled
func NewWithCache(cacheDir string) (*PythonRuntime, error) {
	runtime := &PythonRuntime{
		lastBuiltHandler: map[string]string{},
		lastRebuildCheck: map[string]time.Time{},
		rebuildCooldown:  10 * time.Second, // Prevent rebuilds more than once every 10 seconds
	}

	// Initialize cache and detection systems
	if err := runtime.initializeCacheSystem(cacheDir); err != nil {
		slog.Warn("failed to initialize cache system, falling back to non-cached runtime", "error", err)
		// Don't fail completely, just continue without caching
		return runtime, nil
	}

	return runtime, nil
}

// initializeCacheSystem sets up the build cache and change detection
func (r *PythonRuntime) initializeCacheSystem(cacheDir string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Validate cache directory
	if cacheDir == "" {
		return fmt.Errorf("cache directory cannot be empty")
	}

	// Ensure cache directory exists and is writable
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}

	// Test write permissions
	testFile := filepath.Join(cacheDir, ".test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cache directory %s is not writable: %w", cacheDir, err)
	}
	os.Remove(testFile) // Clean up test file

	// Store cache directory for sharing with incremental builder
	r.cacheDir = cacheDir

	// Use factory to create cache system with sensible defaults
	factory := NewRuntimeFactory()
	buildCache, changeDetector, projectResolver, err := factory.CreateCacheSystem(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to create cache system: %w", err)
	}

	// Validate that all components were created successfully
	if buildCache == nil {
		return fmt.Errorf("build cache was not created")
	}
	if changeDetector == nil {
		return fmt.Errorf("change detector was not created")
	}
	if projectResolver == nil {
		return fmt.Errorf("project resolver was not created")
	}

	r.buildCache = buildCache
	r.changeDetector = changeDetector
	r.projectResolver = projectResolver

	slog.Info("cache system initialized successfully",
		"cacheDir", cacheDir,
		"hasBuildCache", r.buildCache != nil,
		"hasChangeDetector", r.changeDetector != nil,
		"hasProjectResolver", r.projectResolver != nil)

	return nil
}

// EnableCaching enables caching for an existing runtime
func (r *PythonRuntime) EnableCaching(cacheDir string) error {
	return r.initializeCacheSystem(cacheDir)
}

// DisableCaching disables caching and cleans up resources
func (r *PythonRuntime) DisableCaching() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.buildCache != nil {
		if err := r.buildCache.Clear(); err != nil {
			return fmt.Errorf("failed to clear build cache: %w", err)
		}
	}

	r.buildCache = nil
	r.changeDetector = nil
	r.projectResolver = nil

	return nil
}

func (r *PythonRuntime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {

	/// Workspaces are the most challenging part of the build process
	/// UV currently does not support --include-workspace-deps for builds
	/// See: https://github.com/astral-sh/uv/issues/6935 hopefully this lands soon

	/// As a result, we have to manually construct the dependency tree
	/// So we need to:
	///
	/// 1. Build all packages (future tree shaking would be nice)
	/// 2. Ensure local packages are built for lambdaric acccess (remove src/ nesting)
	///			To future readers: we need to do this because of the way python packages are resolved
	///			if you have a package called "mypackage" and it contains a sub-package called "src/mypackage"
	///			then within the package you can resolve code via "import mypackage" but not "import mypackage.src.mypackage"
	///			this means that builds get a little strange for aws lambda which does module level imports via lambdaric
	///			so we need to ensure that the package is built such that lambdaric can resolve paths in the output bundle
	///			but the full package is available for local development
	/// 3. Export the uv package index to requirements.txt
	/// 4. Install the dependencies into the artifact directory as a target (local for zip and delegate to the dockerfile for containers)

	file, err := r.getFile(input)
	if err != nil {

		return nil, fmt.Errorf("python runtime - handler not found: %v", err)
	}

	if input.Dev {
		// Development mode: Simple build without complex dependency management

		// Still call CreateBuildAsset but it will be much simpler for dev
		result, err := r.CreateBuildAsset(ctx, input)
		if err != nil {

			return nil, err
		}

		r.lastBuiltHandler[input.FunctionID] = file

		return result, nil
	} else {
		// Deployment mode: Use the original complex build system

		result, err := r.CreateBuildAsset(ctx, input)
		if err != nil {

			return nil, err
		}

		return result, nil
	}

}

func (r *PythonRuntime) Match(runtime string) bool {
	return strings.HasPrefix(runtime, "python")
}

type Source struct {
	URL          string  `toml:"url,omitempty"`
	Git          string  `toml:"git,omitempty"`
	Subdirectory *string `toml:"subdirectory,omitempty"`
	Branch       string  `toml:"branch,omitempty"`
}

type PyProject struct {
	Project struct {
		Name string `toml:"name"`
	} `toml:"project"`
	Tool struct {
		Setuptools struct {
			Packages struct {
				Find struct {
					Where []string `toml:"where"`
				} `toml:"find"`
			} `toml:"packages"`
		} `toml:"setuptools"`
	} `toml:"tool"`
}

type PackageLayoutInfo struct {
	needsFlattening bool
	layoutType      string
	sourcePath      string
	targetPath      string
}

func (r *PythonRuntime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	// We need the lambda bridge in the artifact directory so that we can run the handler
	// without having to manually isolate the runtime, So if it is not present then we need to copy it from
	// the platform directory into the artifact directory

	// Check if the lambda bridge needs to be copied or updated
	lambdaBridgePath := filepath.Join(input.Build.Out, "lambdaric_python_bridge.py")
	sourceBridgePath := filepath.Join(path.ResolvePlatformDir(input.CfgPath), "/dist/python-runtime/index.py")

	shouldCopy := false
	if _, err := os.Stat(lambdaBridgePath); os.IsNotExist(err) {
		// Bridge file doesn't exist, need to copy
		shouldCopy = true
	} else {
		// Bridge file exists, check if source is newer
		if srcInfo, err := os.Stat(sourceBridgePath); err == nil {
			if dstInfo, err := os.Stat(lambdaBridgePath); err == nil {
				if srcInfo.ModTime().After(dstInfo.ModTime()) {
					shouldCopy = true
				}
			}
		}
	}

	if shouldCopy {
		// Copy the lambda bridge from the platform directory into the artifact directory
		err := copyFile(sourceBridgePath, lambdaBridgePath)
		if err != nil {
			return nil, fmt.Errorf("failed to copy lambda bridge: %v", err)
		}
	}

	// Always use handler path relative to artifact directory
	handlerPath := filepath.Join(input.Build.Out, input.Build.Handler)

	cmd := process.CommandContext(
		ctx,
		"uv",
		"run",
		"--with=requests",
		lambdaBridgePath,
		handlerPath,
		input.WorkerID,
	)
	cmd.Env = append(input.Env, "AWS_LAMBDA_RUNTIME_API="+input.Server)

	// Always run from artifact directory (both dev and deployment)
	cmd.Dir = input.Build.Out
	slog.Info("Python runtime: running from artifact directory", "dir", input.Build.Out)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	slog.Info("starting worker", "args", cmd.Args)
	cmd.Start()

	return &Worker{
		stdout,
		stderr,
		cmd,
	}, nil

}

func (r *PythonRuntime) ShouldRebuild(functionID string, file string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Fast path: Check if the file is relevant for this function first
	// This prevents expensive operations for irrelevant files
	if !r.isRelevantFile(file) {
		slog.Debug("file not relevant for Python function, skipping rebuild check",
			"functionID", functionID,
			"file", file)
		return false
	}

	// Rate limiting: prevent excessive rebuild checks
	// Apply rate limiting early to avoid expensive operations
	now := time.Now()
	if lastCheck, exists := r.lastRebuildCheck[functionID]; exists {
		if now.Sub(lastCheck) < r.rebuildCooldown {
			slog.Debug("rebuild check rate limited",
				"functionID", functionID,
				"file", file,
				"timeSinceLastCheck", now.Sub(lastCheck),
				"cooldown", r.rebuildCooldown)
			return false
		}
	}
	r.lastRebuildCheck[functionID] = now

	slog.Debug("ShouldRebuild processing relevant file",
		"functionID", functionID,
		"file", file,
		"hasChangeDetector", r.changeDetector != nil,
		"hasBuildCache", r.buildCache != nil)

	// For Python files, always rebuild since we can't easily track import dependencies
	// This is simpler and more reliable than trying to parse Python imports
	if strings.HasSuffix(file, ".py") {
		slog.Info("Python file changed, rebuilding",
			"functionID", functionID,
			"file", file)
		return true
	}

	// For infrastructure files, always rebuild since they affect function configuration
	if strings.HasSuffix(file, ".ts") || strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".mjs") ||
		filepath.Base(file) == "sst.config.ts" || filepath.Base(file) == "sst.config.js" || filepath.Base(file) == "package.json" {
		slog.Info("Infrastructure file changed, rebuilding",
			"functionID", functionID,
			"file", file)
		return true
	}

	// If caching is not enabled, always rebuild
	if r.changeDetector == nil {
		slog.Warn("caching not enabled, rebuilding",
			"functionID", functionID,
			"reason", "changeDetector is nil")
		return true
	}

	// Get the handler for this function from the build cache
	handler := r.getHandlerForFunction(functionID)
	if handler == "" {
		slog.Debug("no handler found for function, rebuilding",
			"functionID", functionID,
			"file", file)
		return true
	}

	// Use change detection to determine if rebuild is needed for non-Python files
	result, err := r.changeDetector.DetectChanges(functionID, handler)
	if err != nil {
		slog.Warn("failed to detect changes, rebuilding",
			"functionID", functionID,
			"file", file,
			"handler", handler,
			"error", err)
		return true
	}

	if result.HasChanges {
		slog.Info("changes detected, rebuilding",
			"functionID", functionID,
			"file", file,
			"handler", handler,
			"reason", result.Reason,
			"changeTypes", result.ChangeTypes,
			"changedFiles", len(result.ChangedFiles))
	} else {
		slog.Debug("no changes detected, using cached build",
			"functionID", functionID,
			"file", file,
			"handler", handler)
	}

	return result.HasChanges
}

// getHandlerForFunction retrieves the handler path for a given function ID
func (r *PythonRuntime) getHandlerForFunction(functionID string) string {
	if r.buildCache == nil {
		return ""
	}

	// Try to get the handler from the build cache
	cacheEntry, exists := r.buildCache.Get(functionID)
	if !exists || cacheEntry == nil {
		return ""
	}

	// Return the handler path from the cache entry
	return cacheEntry.Handler
}

// isRelevantFile checks if a file change is relevant for Python functions
func (r *PythonRuntime) isRelevantFile(file string) bool {
	// Quick exclusions first - this prevents infinite rebuild loops
	if r.shouldIgnoreFile(file) {
		return false
	}

	// Python-related files (removed .txt to be more specific)
	relevantExtensions := []string{".py", ".toml", ".lock", ".cfg"}
	relevantFiles := []string{"pyproject.toml", "requirements.txt", "uv.lock", "poetry.lock", "Pipfile.lock", "setup.py", "setup.cfg"}

	// Infrastructure files that affect function deployment
	infrastructureExtensions := []string{".ts", ".js", ".mjs", ".json"}
	infrastructureFiles := []string{"sst.config.ts", "sst.config.js", "package.json"}

	// Check Python file extensions
	for _, ext := range relevantExtensions {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}

	// Check infrastructure file extensions
	for _, ext := range infrastructureExtensions {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}

	// Check specific filenames
	basename := filepath.Base(file)
	for _, relevantFile := range relevantFiles {
		if basename == relevantFile {
			return true
		}
	}

	// Check infrastructure filenames
	for _, infraFile := range infrastructureFiles {
		if basename == infraFile {
			return true
		}
	}

	return false
}

// shouldIgnoreFile determines if a file should be ignored to prevent infinite rebuild loops
func (r *PythonRuntime) shouldIgnoreFile(file string) bool {
	// Ignore build artifacts and cache directories that could cause feedback loops
	ignorePaths := []string{
		".sst/",          // SST cache and build artifacts
		"__pycache__/",   // Python bytecode cache
		".pytest_cache/", // Pytest cache
		".mypy_cache/",   // MyPy cache
		".coverage",      // Coverage files
		"build/",         // Build directories
		"dist/",          // Distribution directories
		".git/",          // Git directory
		"node_modules/",  // Node modules
		".venv/",         // Virtual environments
		"venv/",
		"env/",
		".tox/",       // Tox cache
		".eggs/",      // Egg cache
		"*.egg-info/", // Egg info
	}

	// Ignore file extensions that are build artifacts
	ignoreExtensions := []string{
		".pyc", ".pyo", ".pyd", // Python bytecode
		".log",          // Log files
		".tmp", ".temp", // Temporary files
		".swp", ".swo", // Vim swap files
		".DS_Store", // macOS files
		".coverage", // Coverage files
	}

	// Check if file path contains any ignore patterns
	for _, ignorePath := range ignorePaths {
		if strings.Contains(file, ignorePath) {
			slog.Debug("ignoring file due to path pattern",
				"file", file,
				"pattern", ignorePath)
			return true
		}
	}

	// Check if file has an ignored extension
	for _, ext := range ignoreExtensions {
		if strings.HasSuffix(file, ext) {
			slog.Debug("ignoring file due to extension",
				"file", file,
				"extension", ext)
			return true
		}
	}

	// Ignore files in hidden directories (starting with .)
	parts := strings.Split(file, string(filepath.Separator))
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			// Allow some specific dotfiles that are relevant
			allowedDotFiles := []string{".env", ".gitignore", ".dockerignore"}
			allowed := false
			for _, allowedFile := range allowedDotFiles {
				if part == allowedFile {
					allowed = true
					break
				}
			}
			if !allowed {
				slog.Debug("ignoring file in hidden directory",
					"file", file,
					"hiddenPart", part)
				return true
			}
		}
	}

	return false
}

// GetCacheStats returns statistics about the build cache
func (r *PythonRuntime) GetCacheStats() *CacheStats {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.buildCache == nil {
		return nil
	}

	stats := r.buildCache.GetStats()
	return &stats
}

// ClearCache clears the build cache
func (r *PythonRuntime) ClearCache() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.buildCache == nil {
		return fmt.Errorf("caching not enabled")
	}

	return r.buildCache.Clear()
}

// InvalidateCacheEntry removes a specific cache entry
func (r *PythonRuntime) InvalidateCacheEntry(functionID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.buildCache == nil {
		return fmt.Errorf("caching not enabled")
	}

	return r.buildCache.Delete(functionID)
}

// ForceRebuild forces a rebuild for a specific function
func (r *PythonRuntime) ForceRebuild(functionID string, reason string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.buildCache != nil {
		// Remove from cache to force rebuild
		r.buildCache.Delete(functionID)
	}

	slog.Info("forced rebuild requested", "functionID", functionID, "reason", reason)
}

func (r *PythonRuntime) CreateBuildAsset(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	if input.Dev {
		// Dev mode: Copy source files but skip dependency management
		return r.createSimpleDevBuild(ctx, input)
	}

	// Deployment mode: Full build with dependency management
	type Properties struct {
		Architecture string `json:"architecture"`
		Container    bool   `json:"container"`
	}
	var props Properties
	if err := json.Unmarshal(input.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %v", err)
	}

	arch := props.Architecture
	if arch == "" {
		arch = "x86_64" // Default to x86_64
	}

	if arch != "x86_64" && arch != "arm64" {
		return nil, fmt.Errorf("invalid architecture %q - must be x86_64 or arm64 - %v", arch, string(input.Properties))
	}
	workingDir := path.ResolveRootDir(input.CfgPath)

	// Always use incremental build system
	result, err := r.createBuildAssetIncremental(ctx, input, arch, workingDir)
	if err != nil {
		return nil, fmt.Errorf("incremental build failed: %w", err)
	}
	return result, nil
}

// createBuildAssetIncremental creates build assets using incremental building
func (r *PythonRuntime) createBuildAssetIncremental(ctx context.Context, input *runtime.BuildInput, arch string, workingDir string) (*runtime.BuildOutput, error) {
	// Create incremental builder with sensible defaults
	factory := NewRuntimeFactory()
	progressCallback := func(event ProgressEvent) {
		// Emit FunctionBuildProgressEvent if FunctionID is available
		if input.FunctionID != "" {
			bus.Publish(&events.FunctionBuildProgressEvent{
				FunctionID: input.FunctionID,
				Stage:      event.Stage,
				Message:    event.Message,
			})
		} else {
			// Fallback to StdoutEvent if FunctionID is not available
			bus.Publish(&common.StdoutEvent{
				Line: fmt.Sprintf("🔨 Build %s: %s", event.Stage, event.Message),
			})
		}
	}

	// Use the runtime's cache directory if available, otherwise use default
	var incrementalBuilder *IncrementalBuilder
	var err error
	if r.cacheDir != "" {
		incrementalBuilder, err = factory.CreateIncrementalBuilderWithCacheDir(workingDir, input, progressCallback, r.cacheDir)
	} else {
		incrementalBuilder, err = factory.CreateIncrementalBuilder(workingDir, input, progressCallback)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create incremental builder: %w", err)
	}

	// Use incremental builder
	return incrementalBuilder.Build(ctx, input)
}

func (r *PythonRuntime) getFile(input *runtime.BuildInput) (string, error) {
	startTime := time.Now()
	slog.Info("getFile started", "handler", input.Handler)

	dir := filepath.Dir(input.Handler)
	base := strings.TrimSuffix(filepath.Base(input.Handler), filepath.Ext(input.Handler))
	rootDir := path.ResolveRootDir(input.CfgPath)

	slog.Info("getFile: resolved paths", "elapsed", time.Since(startTime), "dir", dir, "base", base, "rootDir", rootDir)

	// Look for .py file
	pythonFile := filepath.Join(rootDir, dir, base+".py")
	slog.Info("getFile: checking for Python file", "elapsed", time.Since(startTime), "pythonFile", pythonFile)

	if _, err := os.Stat(pythonFile); err == nil {
		slog.Info("getFile: found Python file", "elapsed", time.Since(startTime), "pythonFile", pythonFile)
		return pythonFile, nil
	}

	// No Python file found for the handler
	slog.Error("getFile: Python file not found", "elapsed", time.Since(startTime), "expectedFile", pythonFile)
	return "", fmt.Errorf("could not find Python file '%s.py' in directory '%s'",
		base,
		filepath.Join(rootDir, dir))
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %v", err)
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// copyDirectory recursively copies a directory from src to dst
func copyDirectory(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory: %s", src)
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip unwanted files and directories
		if shouldSkipFile(name, entry.IsDir()) {
			continue
		}

		srcPath := filepath.Join(src, name)
		dstPath := filepath.Join(dst, name)

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy subdirectory %s: %w", name, err)
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", name, err)
			}
		}
	}

	return nil
}

// shouldSkipFile determines if a file or directory should be skipped during copying
func shouldSkipFile(name string, isDir bool) bool {
	// Skip hidden files and directories
	if strings.HasPrefix(name, ".") {
		return true
	}

	if isDir {
		// Skip common development and cache directories
		skipDirs := []string{
			"__pycache__",
			".pytest_cache",
			".mypy_cache",
			".ruff_cache",
			".coverage",
			"venv",
			".venv",
			"env",
			".env",
			"node_modules",
			".git",
			".svn",
			".hg",
			"tests",
			"test",
			".tox",
			"htmlcov",
			".nyc_output",
			"coverage",
			"dist",
			"build",
			"*.egg-info",
		}

		for _, skipDir := range skipDirs {
			if name == skipDir || strings.HasSuffix(name, ".egg-info") {
				return true
			}
		}
	} else {
		// Skip common development files
		skipFiles := []string{
			"coverage.json",
			"coverage.xml",
			".coverage",
			"pytest.ini",
			"pyproject.toml", // This is handled separately if needed
			"setup.py",       // This is handled separately if needed
			"setup.cfg",
			"tox.ini",
			".gitignore",
			".gitattributes",
			"README.md",
			"README.rst",
			"LICENSE",
			"MANIFEST.in",
			"requirements-dev.txt",
			"requirements-test.txt",
		}

		for _, skipFile := range skipFiles {
			if name == skipFile {
				return true
			}
		}

		// Skip files with certain extensions
		skipExtensions := []string{
			".pyc",
			".pyo",
			".pyd",
			".so",
			".dylib",
			".dll",
			".log",
			".tmp",
			".temp",
			".bak",
			".swp",
			".DS_Store",
		}

		for _, ext := range skipExtensions {
			if strings.HasSuffix(name, ext) {
				return true
			}
		}
	}

	return false
}

// detectFlatLayout detects if this is a flat-layout project based on the artifact contents
func (r *PythonRuntime) detectFlatLayout(artifactDir string) bool {
	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		slog.Warn("failed to read artifact directory for layout detection", "error", err)
		return false
	}

	var hasSST, hasPyProjectToml, hasDirectPyFiles bool
	var moduleCount int

	for _, entry := range entries {
		name := entry.Name()

		// Check for SST directory (indicates entire project was included)
		if name == ".sst" && entry.IsDir() {
			hasSST = true
		}

		// Check for project configuration files
		if name == "pyproject.toml" || name == "setup.py" {
			hasPyProjectToml = true
		}

		// Check for direct Python files in root
		if !entry.IsDir() && strings.HasSuffix(name, ".py") {
			hasDirectPyFiles = true
		}

		// Count module directories
		if entry.IsDir() {
			modulePath := filepath.Join(artifactDir, name)
			if initFile := filepath.Join(modulePath, "__init__.py"); fileExists(initFile) {
				moduleCount++
			}
		}
	}

	// Check if modules contain project-level files that indicate flat layout
	var hasProjectFilesInModules bool
	for _, entry := range entries {
		if entry.IsDir() {
			modulePath := filepath.Join(artifactDir, entry.Name())
			// Check if module contains project-level files
			if moduleEntries, err := os.ReadDir(modulePath); err == nil {
				for _, moduleEntry := range moduleEntries {
					if moduleEntry.Name() == "pyproject.toml" ||
						moduleEntry.Name() == "uv.lock" ||
						moduleEntry.Name() == "setup.py" {
						hasProjectFilesInModules = true
						break
					}
				}
			}
			if hasProjectFilesInModules {
				break
			}
		}
	}

	// Flat layout indicators:
	// 1. Has .sst directory (entire project included)
	// 2. Has pyproject.toml (project config included)
	// 3. Has direct Python files in root
	// 4. Low module count (typically just one package) with project files in modules
	// 5. Modules contain project-level files like pyproject.toml or uv.lock
	isFlat := hasSST || (hasPyProjectToml && (hasDirectPyFiles || moduleCount <= 1)) ||
		(moduleCount <= 2 && hasProjectFilesInModules)

	slog.Info("flat layout detection results",
		"artifactDir", artifactDir,
		"hasSST", hasSST,
		"hasPyProjectToml", hasPyProjectToml,
		"hasDirectPyFiles", hasDirectPyFiles,
		"moduleCount", moduleCount,
		"isFlat", isFlat)

	return isFlat
}

// applyMinimalFiltering applies minimal content filtering for workspace layouts
func (r *PythonRuntime) applyMinimalFiltering(artifactDir string, filter *ContentFilter) error {
	slog.Info("applying minimal content filtering", "artifactDir", artifactDir)

	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		return fmt.Errorf("failed to read artifact directory: %w", err)
	}

	var removedItems []string

	for _, entry := range entries {
		// Only remove obvious build artifacts and cache directories
		if filter.ShouldExclude(entry.Name()) {
			itemPath := filepath.Join(artifactDir, entry.Name())

			// Only remove specific items that are safe to remove
			if entry.Name() == "__pycache__" ||
				strings.HasSuffix(entry.Name(), ".pyc") ||
				strings.HasSuffix(entry.Name(), ".pyo") ||
				entry.Name() == ".pytest_cache" ||
				entry.Name() == ".sst" ||
				strings.HasPrefix(entry.Name(), ".sst-") {

				if err := os.RemoveAll(itemPath); err != nil {
					slog.Warn("failed to remove item during minimal filtering",
						"item", entry.Name(),
						"path", itemPath,
						"error", err)
				} else {
					removedItems = append(removedItems, entry.Name())
					slog.Debug("removed item during minimal filtering",
						"item", entry.Name())
				}
			}
		}
	}

	slog.Info("minimal content filtering completed",
		"artifactDir", artifactDir,
		"removedItems", removedItems,
		"count", len(removedItems))

	return nil
}

// ensureInitPyFiles ensures that __init__.py files are created in all necessary directories
func (r *PythonRuntime) ensureInitPyFiles(targetDir, moduleName string) error {
	slog.Info("ensuring __init__.py files", "targetDir", targetDir, "moduleName", moduleName)

	// Create __init__.py in the main module directory
	mainInitFile := filepath.Join(targetDir, "__init__.py")
	if err := r.createInitPyFile(mainInitFile); err != nil {
		return fmt.Errorf("failed to create main __init__.py: %w", err)
	}

	// Recursively ensure __init__.py files in subdirectories that contain Python files
	return r.ensureInitPyFilesRecursive(targetDir)
}

// createInitPyFile creates an __init__.py file if it doesn't exist
func (r *PythonRuntime) createInitPyFile(initPath string) error {
	if _, err := os.Stat(initPath); os.IsNotExist(err) {
		slog.Info("creating __init__.py file", "path", initPath)

		// Create a basic __init__.py file with a docstring
		content := fmt.Sprintf(`"""
%s module.

This file makes Python treat the directory as a package.
"""
`, filepath.Base(filepath.Dir(initPath)))

		if err := os.WriteFile(initPath, []byte(content), 0644); err != nil {
			slog.Error("failed to create __init__.py", "path", initPath, "error", err)
			return fmt.Errorf("failed to create __init__.py at %s: %w", initPath, err)
		}

		slog.Info("successfully created __init__.py", "path", initPath)
	} else if err != nil {
		return fmt.Errorf("failed to stat __init__.py at %s: %w", initPath, err)
	} else {
		slog.Debug("__init__.py already exists", "path", initPath)

		// Verify the existing file is readable
		if content, err := os.ReadFile(initPath); err != nil {
			slog.Warn("existing __init__.py file is not readable", "path", initPath, "error", err)
		} else {
			slog.Debug("existing __init__.py file validated",
				"path", initPath,
				"size", len(content))
		}
	}

	return nil
}

// ensureInitPyFilesRecursive recursively ensures __init__.py files in subdirectories
func (r *PythonRuntime) ensureInitPyFilesRecursive(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip common non-package directories
		if r.shouldSkipDirectory(entry.Name()) {
			continue
		}

		subDir := filepath.Join(dir, entry.Name())

		// Check if this directory contains Python files
		if r.containsPythonFiles(subDir) {
			initFile := filepath.Join(subDir, "__init__.py")
			if err := r.createInitPyFile(initFile); err != nil {
				slog.Warn("failed to create __init__.py in subdirectory",
					"subDir", subDir,
					"error", err)
				// Don't fail the entire process for subdirectory __init__.py issues
			}

			// Recursively process subdirectories
			if err := r.ensureInitPyFilesRecursive(subDir); err != nil {
				slog.Warn("failed to ensure __init__.py files in subdirectory",
					"subDir", subDir,
					"error", err)
				// Don't fail the entire process for subdirectory issues
			}
		}
	}

	return nil
}

// shouldSkipDirectory checks if a directory should be skipped for __init__.py creation
func (r *PythonRuntime) shouldSkipDirectory(dirName string) bool {
	skipDirs := []string{
		"__pycache__",
		".pytest_cache",
		".git",
		".sst",
		"node_modules",
		".venv",
		"venv",
		"env",
		".env",
		"tests",
		"test",
	}

	for _, skipDir := range skipDirs {
		if dirName == skipDir {
			return true
		}
	}

	// Skip .dist-info and .egg-info directories
	if strings.HasSuffix(dirName, ".dist-info") || strings.HasSuffix(dirName, ".egg-info") {
		return true
	}

	return false
}

// containsPythonFiles checks if a directory contains Python files
func (r *PythonRuntime) containsPythonFiles(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
			return true
		}
	}
	return false
}

func (r *PythonRuntime) getWorkspaceDirectory(input *runtime.BuildInput) (string, error) {
	startTime := time.Now()
	slog.Info("getWorkspaceDirectory started", "functionID", input.FunctionID, "handler", input.Handler)

	slog.Info("getWorkspaceDirectory: calling getFile", "elapsed", time.Since(startTime))
	file, err := r.getFile(input)
	if err != nil {
		slog.Error("getWorkspaceDirectory: getFile failed", "elapsed", time.Since(startTime), "error", err)
		return "", err
	}
	slog.Info("getWorkspaceDirectory: getFile completed", "elapsed", time.Since(startTime), "file", file)

	projectRoot := path.ResolveRootDir(input.CfgPath)
	currentDir := filepath.Dir(file)

	slog.Info("getWorkspaceDirectory: resolved paths", "elapsed", time.Since(startTime), "projectRoot", projectRoot, "currentDir", currentDir)

	// First verify that the current directory is within the project root
	if !strings.HasPrefix(currentDir, projectRoot) {
		slog.Error("getWorkspaceDirectory: handler file not within project root", "elapsed", time.Since(startTime), "file", file, "projectRoot", projectRoot)
		return "", fmt.Errorf("handler file %s is not within the project root %s", file, projectRoot)
	}

	// Traverse up the file tree to find the pyproject.toml file
	// If we reach the project root then return an error
	slog.Info("getWorkspaceDirectory: searching for pyproject.toml", "elapsed", time.Since(startTime), "startingDir", currentDir)
	searchCount := 0
	for {
		searchCount++
		pyprojectPath := filepath.Join(currentDir, "pyproject.toml")
		slog.Debug("getWorkspaceDirectory: checking for pyproject.toml", "elapsed", time.Since(startTime), "searchCount", searchCount, "path", pyprojectPath)

		if _, err := os.Stat(pyprojectPath); err == nil {
			// We found the pyproject.toml file
			slog.Info("getWorkspaceDirectory: found pyproject.toml", "elapsed", time.Since(startTime), "searchCount", searchCount, "workspaceDir", currentDir)
			return currentDir, nil
		}

		// Move up the directory tree
		parentDir := filepath.Dir(currentDir)

		// Check if we have reached the project root or cannot move up anymore
		if parentDir == currentDir || currentDir == projectRoot {
			slog.Error("getWorkspaceDirectory: no pyproject.toml found", "elapsed", time.Since(startTime), "searchCount", searchCount, "startDir", filepath.Dir(file), "projectRoot", projectRoot)
			return "", fmt.Errorf("no pyproject.toml found in directory tree from %s up to project root %s", filepath.Dir(file), projectRoot)
		}

		currentDir = parentDir
	}
}

func (r *PythonRuntime) getPackageName(input *runtime.BuildInput) (string, error) {
	workspaceDir, err := r.getWorkspaceDirectory(input)
	if err != nil {
		return "", err
	}

	// Read the pyproject.toml file
	pyproject, err := os.ReadFile(filepath.Join(workspaceDir, "pyproject.toml"))
	if err != nil {
		return "", fmt.Errorf("failed to read pyproject.toml file: %v", err)
	}

	// Parse the pyproject.toml file
	pyprojectData := PyProject{}
	err = toml.Unmarshal(pyproject, &pyprojectData)
	if err != nil {
		return "", fmt.Errorf("failed to parse pyproject.toml file: %v", err)
	}

	return pyprojectData.Project.Name, nil

}

func (r *PythonRuntime) adjustHandlerPath(input *runtime.BuildInput) (string, error) {
	originalHandler := input.Handler

	// Check if handler path contains the pattern {package_name}/src/{package_name}
	// If so, we need to adjust it because the build process flattens this structure
	if strings.Contains(originalHandler, "/src/") {
		// Pattern: functions/src/functions/api.handler -> functions/api.handler
		parts := strings.Split(originalHandler, "/")
		var adjustedParts []string

		skipNext := false
		for i, part := range parts {
			if skipNext {
				skipNext = false
				continue
			}

			if part == "src" && i > 0 && i < len(parts)-1 {
				// Check if the next part matches the previous part (package name)
				if i+1 < len(parts) && parts[i+1] == parts[i-1] {
					// Skip both "src" and the duplicate package name
					skipNext = true
					continue
				}
			}
			adjustedParts = append(adjustedParts, part)
		}

		adjustedHandler := strings.Join(adjustedParts, "/")
		slog.Info("handler path adjustment",
			"original", originalHandler,
			"adjusted", adjustedHandler,
			"reason", "flattened src structure")
		return adjustedHandler, nil
	}

	slog.Info("handler path adjustment", "original", originalHandler, "adjusted", "no change needed")
	return originalHandler, nil
}

// validateDeploymentReadiness performs final validation to ensure the artifact is ready for deployment
func (r *PythonRuntime) validateDeploymentReadiness(artifactDir, handlerPath string, validationResult *ArtifactValidationResult, progressReporter *BuildProgressReporter) error {
	slog.Info("validating deployment readiness",
		"artifactDir", artifactDir,
		"handlerPath", handlerPath)

	// 1. Ensure artifact directory exists and is not empty
	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		return fmt.Errorf("failed to read artifact directory: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("artifact directory is empty: %s", artifactDir)
	}

	// 2. Validate that critical validation checks passed
	if validationResult != nil {
		if !validationResult.Success {
			return fmt.Errorf("artifact validation failed with %d errors", len(validationResult.ErrorMessages))
		}

		if !validationResult.HandlerCompatible {
			return fmt.Errorf("handler path '%s' is not compatible with deployed modules", handlerPath)
		}

		if len(validationResult.PythonModules) == 0 {
			return fmt.Errorf("no Python modules found in artifact")
		}

		if len(validationResult.MissingModules) > 0 {
			return fmt.Errorf("missing required modules: %s", strings.Join(validationResult.MissingModules, ", "))
		}
	}

	// 3. Validate essential files exist
	essentialChecks := []struct {
		name        string
		path        string
		required    bool
		description string
	}{
		{
			name:        "requirements.txt",
			path:        filepath.Join(artifactDir, "requirements.txt"),
			required:    false, // Not always required (e.g., no external deps)
			description: "Python dependencies file",
		},
	}

	for _, check := range essentialChecks {
		exists := fileExists(check.path)
		if check.required && !exists {
			return fmt.Errorf("required file missing: %s (%s)", check.name, check.description)
		}

		slog.Debug("essential file check",
			"file", check.name,
			"path", check.path,
			"exists", exists,
			"required", check.required)
	}

	// 4. Validate handler module structure
	if handlerPath != "" {
		var expectedModule string
		var handlerParts []string

		if strings.Contains(handlerPath, "/") {
			// Workspace layout: functions/handler.lambda_handler
			handlerParts = strings.Split(handlerPath, "/")
			if len(handlerParts) > 0 {
				expectedModule = handlerParts[0]
			}
		} else {
			// Flat layout: handler.lambda_handler or module.handler.lambda_handler
			// For flat layouts, we need to check if the handler path contains a module prefix
			dotParts := strings.Split(handlerPath, ".")
			if len(dotParts) >= 3 {
				// Format: module.handler.lambda_handler
				expectedModule = dotParts[0]
				handlerParts = []string{expectedModule, strings.Join(dotParts[1:len(dotParts)-1], ".")}
			} else if len(dotParts) == 2 {
				// Format: handler.lambda_handler (no module prefix)
				// In this case, we need to find which module contains the handler file
				handlerFile := dotParts[0] + ".py"
				if validationResult != nil {
					for _, module := range validationResult.PythonModules {
						handlerFilePath := filepath.Join(artifactDir, module, handlerFile)
						if fileExists(handlerFilePath) {
							expectedModule = module
							handlerParts = []string{module, dotParts[0]}
							break
						}
					}
				}
				if expectedModule == "" {
					// If no module contains the handler file, this might be an error
					// but we'll let the later validation catch it
					expectedModule = dotParts[0] // This will likely fail validation
					handlerParts = []string{expectedModule}
				}
			} else {
				return fmt.Errorf("invalid handler path format: %s", handlerPath)
			}
		}

		if expectedModule != "" {
			modulePath := filepath.Join(artifactDir, expectedModule)

			if !fileExists(modulePath) {
				return fmt.Errorf("handler module directory not found: %s", modulePath)
			}

			// Check if it's a proper Python module
			initPyPath := filepath.Join(modulePath, "__init__.py")
			if !fileExists(initPyPath) {
				slog.Warn("handler module missing __init__.py file",
					"module", expectedModule,
					"path", initPyPath)

				// Report warning about missing __init__.py
				progressReporter.ReportWarning("validate",
					fmt.Sprintf("Module '%s' missing __init__.py file", expectedModule),
					map[string]interface{}{
						"module": expectedModule,
						"path":   initPyPath,
					})
			}

			// If handler specifies a file, check if it exists
			if len(handlerParts) > 1 {
				// Parse handler path: module/file.function -> module, file.py, function
				// For example: functions/handler.lambda_handler -> functions, handler.py, lambda_handler
				handlerFileAndFunc := strings.Join(handlerParts[1:], "/")

				// Split on the last dot to separate file from function
				lastDotIndex := strings.LastIndex(handlerFileAndFunc, ".")
				var handlerFile string
				if lastDotIndex != -1 {
					// Extract just the file part (before the last dot)
					handlerFile = handlerFileAndFunc[:lastDotIndex] + ".py"
				} else {
					// No function specified, treat entire thing as file
					handlerFile = handlerFileAndFunc + ".py"
				}

				handlerFilePath := filepath.Join(modulePath, handlerFile)

				if !fileExists(handlerFilePath) {
					return fmt.Errorf("handler file not found: %s", handlerFilePath)
				}

				slog.Debug("handler file validation successful",
					"handlerPath", handlerPath,
					"handlerFile", handlerFilePath)
			}
		}
	}

	// 5. Calculate and log final artifact statistics
	var totalSize int64
	var fileCount int
	var pythonFileCount int

	err = filepath.Walk(artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue despite errors
		}

		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()

			if strings.HasSuffix(info.Name(), ".py") {
				pythonFileCount++
			}
		}

		return nil
	})

	if err != nil {
		slog.Warn("failed to calculate artifact statistics", "error", err)
	}

	slog.Info("deployment readiness validation successful",
		"artifactDir", artifactDir,
		"handlerPath", handlerPath,
		"totalSize", totalSize,
		"fileCount", fileCount,
		"pythonFileCount", pythonFileCount,
		"ready", true)

	return nil
}

// calculateArtifactSummary calculates summary information about the deployment artifact
func (r *PythonRuntime) calculateArtifactSummary(artifactDir string) ArtifactSummary {
	summary := ArtifactSummary{}

	err := filepath.Walk(artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue despite errors
		}

		if !info.IsDir() {
			summary.FileCount++
			summary.TotalSize += info.Size()

			if strings.HasSuffix(info.Name(), ".py") {
				summary.PythonFileCount++
			}
		} else {
			// Count directories that look like Python modules
			if strings.Contains(path, artifactDir) && path != artifactDir {
				relPath, _ := filepath.Rel(artifactDir, path)
				// Count top-level directories as potential modules
				if !strings.Contains(relPath, string(filepath.Separator)) {
					summary.ModuleCount++
				}
			}
		}

		return nil
	})

	if err != nil {
		slog.Warn("failed to calculate artifact summary", "error", err, "artifactDir", artifactDir)
	}

	return summary
}

// updateCacheAfterBuild updates the build cache after a successful build
func (r *PythonRuntime) updateCacheAfterBuild(input *runtime.BuildInput, buildOutput *runtime.BuildOutput) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Skip if caching is not enabled
	if r.changeDetector == nil || r.projectResolver == nil {
		return
	}

	// Update project resolver project root
	workingDir := path.ResolveRootDir(input.CfgPath)
	r.projectResolver.projectRoot = workingDir

	// Resolve project structure for this build
	projectInfo, err := r.projectResolver.ResolveHandler(input.Handler)
	if err != nil {
		slog.Warn("failed to resolve project structure for cache update",
			"functionID", input.FunctionID,
			"handler", input.Handler,
			"error", err)
		return
	}

	// Create cached build output
	cachedBuildOutput := &CachedBuildOutput{
		Handler:       buildOutput.Handler,
		OutputDir:     input.Out(),
		Errors:        buildOutput.Errors,
		Sourcemaps:    buildOutput.Sourcemaps,
		ArtifactPaths: []string{}, // TODO: collect actual artifact paths
		BuildDuration: 0,          // TODO: track build duration
	}

	// Update cache
	err = r.changeDetector.UpdateCacheAfterBuild(input.FunctionID, input.Handler, projectInfo, cachedBuildOutput)
	if err != nil {
		slog.Warn("failed to update cache after build",
			"functionID", input.FunctionID,
			"error", err)
	} else {
		slog.Debug("updated build cache",
			"functionID", input.FunctionID,
			"handler", input.Handler)
	}
}

// cleanupModuleBuildArtifacts removes build artifacts from extracted modules
func (r *PythonRuntime) cleanupModuleBuildArtifacts(moduleDir string) error {
	slog.Info("cleaning up build artifacts from module", "moduleDir", moduleDir)

	// List of directories and files to remove from modules
	artifactsToRemove := []string{
		".sst",
		"__pycache__",
		".pytest_cache",
		".coverage",
		"htmlcov",
		".git",
		".gitignore",
		".gitattributes",
		"node_modules",
		".vscode",
		".idea",
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.yaml",
		"Makefile",
		"README.md",
		"README.rst",
		"README.txt",
		"CHANGELOG.md",
		"CHANGELOG.rst",
		"CHANGELOG.txt",
		"LICENSE",
		"LICENSE.txt",
		"MANIFEST.in",
		"setup.cfg",
		"tox.ini",
		".pre-commit-config.yaml",
	}

	var removedItems []string

	for _, artifact := range artifactsToRemove {
		artifactPath := filepath.Join(moduleDir, artifact)
		if _, err := os.Stat(artifactPath); err == nil {
			if err := os.RemoveAll(artifactPath); err != nil {
				slog.Warn("failed to remove build artifact",
					"artifact", artifact,
					"path", artifactPath,
					"error", err)
			} else {
				removedItems = append(removedItems, artifact)
				slog.Debug("removed build artifact",
					"artifact", artifact,
					"path", artifactPath)
			}
		}
	}

	// Also remove any .pyc files
	err := filepath.Walk(moduleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}

		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".pyc") || strings.HasSuffix(info.Name(), ".pyo")) {
			if err := os.Remove(path); err != nil {
				slog.Warn("failed to remove compiled Python file",
					"path", path,
					"error", err)
			} else {
				slog.Debug("removed compiled Python file", "path", path)
			}
		}

		return nil
	})

	if err != nil {
		slog.Warn("error walking module directory for cleanup",
			"moduleDir", moduleDir,
			"error", err)
	}

	slog.Info("module build artifact cleanup completed",
		"moduleDir", moduleDir,
		"removedItems", removedItems,
		"count", len(removedItems))

	return nil
}

// cleanupAbsolutePaths removes any directories with absolute paths from the artifact directory
func (r *PythonRuntime) cleanupAbsolutePaths(artifactDir string) error {
	slog.Info("cleaning up absolute paths from artifact directory", "artifactDir", artifactDir)

	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		return fmt.Errorf("failed to read artifact directory: %w", err)
	}

	var removedItems []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if directory name contains absolute path indicators
			name := entry.Name()
			if strings.Contains(name, ":") && (strings.Contains(name, "/Users/") || strings.Contains(name, "/home/") || strings.Contains(name, "C:\\")) {
				itemPath := filepath.Join(artifactDir, name)
				slog.Info("removing absolute path directory", "path", itemPath)

				if err := os.RemoveAll(itemPath); err != nil {
					slog.Warn("failed to remove absolute path directory", "path", itemPath, "error", err)
				} else {
					removedItems = append(removedItems, name)
				}
			}
		}
	}

	if len(removedItems) > 0 {
		slog.Info("cleaned up absolute path directories",
			"artifactDir", artifactDir,
			"removedItems", removedItems,
			"count", len(removedItems))
	}

	return nil
}

// addSSTModule adds the SST Python runtime module to the deployment package
func (r *PythonRuntime) addSSTModule(artifactDir string) error {
	slog.Info("adding SST runtime module to deployment package", "artifactDir", artifactDir)

	// Get the path to the SST module file
	// The module is embedded in the Go binary at compile time
	sstModuleContent := `"""
SST Python Runtime Module

This module provides access to SST resources in Python Lambda functions.
It reads encrypted resource data from resource.enc and provides a Resource API.
"""

import os
import json
import base64
from typing import Any, Dict


class ResourceProxy:
    """Proxy object that provides attribute access to resource properties."""
    
    def __init__(self, data: Dict[str, Any]):
        self._data = data
    
    def __getattr__(self, name: str) -> Any:
        if name in self._data:
            value = self._data[name]
            if isinstance(value, dict):
                return ResourceProxy(value)
            return value
        raise AttributeError(f"Resource has no attribute '{name}'")


class ResourceManager:
    """Manages SST resources by decrypting and parsing resource.enc file."""
    
    def __init__(self):
        self._resources = None
        self._load_resources()
    
    def _load_resources(self):
        """Load and decrypt resources from resource.enc file."""
        try:
            # Get encryption key from environment
            key_b64 = os.environ.get('SST_KEY')
            if not key_b64:
                # In development mode, resources might be provided directly via env vars
                self._load_dev_resources()
                return
            
            # Try to import cryptography for decryption
            try:
                from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
                from cryptography.hazmat.backends import default_backend
            except ImportError:
                print("Warning: cryptography package not available, falling back to development mode")
                self._load_dev_resources()
                return
            
            # Decode the base64 key
            key = base64.b64decode(key_b64)
            
            # Read the encrypted resource file
            resource_file = os.environ.get('SST_KEY_FILE', 'resource.enc')
            if not os.path.exists(resource_file):
                print(f"Warning: Resource file {resource_file} not found, falling back to development mode")
                self._load_dev_resources()
                return
            
            with open(resource_file, 'rb') as f:
                ciphertext = f.read()
            
            # Decrypt using AES-GCM (matching Go implementation)
            # Go uses 12-byte nonce (all zeros) and no additional data
            nonce = b'\x00' * 12
            cipher = Cipher(algorithms.AES(key), modes.GCM(nonce), backend=default_backend())
            decryptor = cipher.decryptor()
            
            # The ciphertext includes the auth tag at the end (16 bytes for GCM)
            if len(ciphertext) < 16:
                raise ValueError("Invalid ciphertext: too short")
            
            # Split ciphertext and auth tag
            actual_ciphertext = ciphertext[:-16]
            auth_tag = ciphertext[-16:]
            
            # Set the auth tag and decrypt
            decryptor.authenticate_additional_data(b'')
            plaintext = decryptor.update(actual_ciphertext)
            decryptor.finalize_with_tag(auth_tag)
            
            # Parse JSON
            self._resources = json.loads(plaintext.decode('utf-8'))
            
        except Exception as e:
            # Fallback to development mode if decryption fails
            print(f"Warning: Failed to decrypt resources ({e}), falling back to development mode")
            self._load_dev_resources()
    
    def _load_dev_resources(self):
        """Load resources from environment variables (development mode)."""
        self._resources = {}
        
        # Look for SST_RESOURCE_* environment variables
        for key, value in os.environ.items():
            if key.startswith('SST_RESOURCE_') and key != 'SST_RESOURCE_App':
                resource_name = key[13:]  # Remove 'SST_RESOURCE_' prefix
                try:
                    self._resources[resource_name] = json.loads(value)
                except json.JSONDecodeError:
                    self._resources[resource_name] = value
    
    def __getattr__(self, name: str) -> Any:
        if self._resources is None:
            raise RuntimeError("Resources not loaded")
        
        if name in self._resources:
            value = self._resources[name]
            if isinstance(value, dict):
                return ResourceProxy(value)
            return value
        
        raise AttributeError(f"Resource '{name}' not found")


# Global resource manager instance
Resource = ResourceManager()
`

	// Create the sst.py file in the artifact directory
	sstModulePath := filepath.Join(artifactDir, "sst.py")
	err := os.WriteFile(sstModulePath, []byte(sstModuleContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write SST module: %w", err)
	}

	slog.Info("SST runtime module added successfully",
		"path", sstModulePath,
		"size", len(sstModuleContent))

	return nil
}

// createSimpleDevBuild creates a simple build for development mode by copying source files
func (r *PythonRuntime) createSimpleDevBuild(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Create artifact directory
	if err := os.MkdirAll(input.Out(), 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %v", err)
	}

	projectRoot := path.ResolveRootDir(input.CfgPath)

	// Check if this is a properly structured project that doesn't need full copying
	if r.isProperlyStructuredProject(projectRoot) {
		// Fast path: copy only essential files instead of everything
		err := r.copyEssentialFiles(projectRoot, input.Out(), input.Handler)
		if err != nil {
			// Fallback to full copying if essential copy fails
			err := r.copyPythonFilesWithProgress(projectRoot, input.Out(), input.FunctionID)
			if err != nil {
				return nil, fmt.Errorf("failed to copy Python files: %v", err)
			}
		}

		return &runtime.BuildOutput{
			Handler:    input.Handler,
			Sourcemaps: []string{},
			Errors:     []string{},
			Out:        input.Out(),
		}, nil
	}

	// Fallback: Copy Python files with directory structure (for projects that need layout fixes)
	err := r.copyPythonFilesWithProgress(projectRoot, input.Out(), input.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("failed to copy Python files: %v", err)
	}

	// Fix workspace layouts: flatten any package/src/package structures for import compatibility
	err = r.flattenWorkspaceLayouts(input.Out(), input.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("failed to flatten workspace layouts: %v", err)
	}

	return &runtime.BuildOutput{
		Handler:    input.Handler,
		Sourcemaps: []string{},
		Errors:     []string{},
		Out:        input.Out(),
	}, nil
}

// isProperlyStructuredProject checks if a project is properly structured and doesn't need file copying
func (r *PythonRuntime) isProperlyStructuredProject(projectRoot string) bool {
	// Check for workspace layout patterns that would need flattening
	hasWorkspaceLayouts := r.hasWorkspaceLayoutPatterns(projectRoot)

	// If there are workspace layouts that need flattening, we need to copy
	if hasWorkspaceLayouts {
		return false
	}

	// Additional checks could be added here:
	// - Check if all Python files are in standard locations
	// - Check if there are any complex import patterns that need resolution
	// - For now, if no workspace layouts need flattening, consider it properly structured

	return true
}

// copyEssentialFiles copies only the essential files needed for properly structured projects
func (r *PythonRuntime) copyEssentialFiles(projectRoot, artifactDir, handler string) error {
	// For properly structured projects, we only need to copy Python files
	// This is much faster than full copying but avoids symlink issues
	return r.copyPythonFiles(projectRoot, artifactDir)
}

// hasWorkspaceLayoutPatterns checks if the project has package/src/package patterns that need flattening
func (r *PythonRuntime) hasWorkspaceLayoutPatterns(projectRoot string) bool {
	// Walk through the project looking for package/src/package patterns
	var hasPatterns bool

	filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}

		// Skip hidden directories and common non-package directories
		dirName := filepath.Base(path)
		if strings.HasPrefix(dirName, ".") ||
			dirName == "__pycache__" ||
			dirName == "node_modules" ||
			dirName == ".sst" {
			return filepath.SkipDir
		}

		// Check if this directory follows the package/src/package pattern
		srcDir := filepath.Join(path, "src")
		if _, err := os.Stat(srcDir); err == nil {
			// Check if there's a subdirectory in src with the same name as the parent
			packageName := dirName
			innerPackageDir := filepath.Join(srcDir, packageName)
			if _, err := os.Stat(innerPackageDir); err == nil {
				hasPatterns = true
				return filepath.SkipDir // Found one, no need to continue this branch
			}
		}

		return nil
	})

	return hasPatterns
}

// copySourceFiles copies Python source files and installs local packages
func (r *PythonRuntime) copySourceFiles(srcDir, destDir string) error {
	// First, copy all Python source files

	err := r.copyPythonFiles(srcDir, destDir)
	if err != nil {

		return fmt.Errorf("failed to copy Python files: %w", err)
	}

	// Then, install local Python packages

	err = r.installLocalPackages(srcDir, destDir)
	if err != nil {

		return fmt.Errorf("failed to install local packages: %w", err)
	}

	return nil
}

// hasComplexImports checks if the handler has complex imports that require full project copy
func (r *PythonRuntime) hasComplexImports(input *runtime.BuildInput, projectRoot string) (bool, error) {
	file, err := r.getFile(input)
	if err != nil {
		return true, err // If we can't get the file, assume complex
	}

	// Read the handler file to check for imports
	content, err := os.ReadFile(file)
	if err != nil {
		return true, err // If we can't read, assume complex
	}

	contentStr := string(content)

	// Check for indicators of complex imports:
	// 1. Relative imports (from . import, from .. import)
	// 2. Local package imports (not standard library)
	// 3. Multiple directories in import paths

	lines := strings.Split(contentStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Check for relative imports
		if strings.Contains(line, "from .") || strings.Contains(line, "from ..") {
			return true, nil
		}

		// Check for local package imports (imports with multiple path segments)
		if strings.HasPrefix(line, "from ") || strings.HasPrefix(line, "import ") {
			// Extract the import path
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				importPath := parts[1]
				// If import has multiple segments and doesn't look like standard library
				if strings.Contains(importPath, ".") && !isStandardLibrary(importPath) {
					return true, nil
				}
			}
		}
	}

	// Check if there are other Python files in parent directories (workspace layout)
	handlerDir := filepath.Dir(file)
	parentDir := filepath.Dir(handlerDir)

	// If handler is not in project root and parent has Python files, likely complex
	if parentDir != projectRoot {
		entries, err := os.ReadDir(parentDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// isStandardLibrary checks if an import looks like standard library
func isStandardLibrary(importPath string) bool {
	// Common standard library modules
	stdLibModules := []string{
		"os", "sys", "json", "time", "datetime", "re", "urllib", "http",
		"logging", "collections", "itertools", "functools", "pathlib",
		"typing", "dataclasses", "enum", "abc", "contextlib", "copy",
		"pickle", "base64", "hashlib", "hmac", "secrets", "uuid",
		"math", "random", "statistics", "decimal", "fractions",
	}

	// Get the root module name
	rootModule := strings.Split(importPath, ".")[0]

	for _, stdMod := range stdLibModules {
		if rootModule == stdMod {
			return true
		}
	}

	return false
}

// copyHandlerFilesOnly copies only the handler file and its directory for dev mode (much faster)
func (r *PythonRuntime) copyHandlerFilesOnly(input *runtime.BuildInput, projectRoot, destDir string) error {
	// Get the handler file path
	file, err := r.getFile(input)
	if err != nil {
		return fmt.Errorf("failed to get handler file: %w", err)
	}

	// Get the directory containing the handler
	handlerDir := filepath.Dir(file)

	// Calculate relative path from project root to handler directory
	relHandlerDir, err := filepath.Rel(projectRoot, handlerDir)
	if err != nil {
		return fmt.Errorf("failed to get relative handler directory: %w", err)
	}

	// Create the handler directory in destination
	destHandlerDir := filepath.Join(destDir, relHandlerDir)
	if err := os.MkdirAll(destHandlerDir, 0755); err != nil {
		return fmt.Errorf("failed to create handler directory: %w", err)
	}

	// Copy all Python files from the handler directory only
	entries, err := os.ReadDir(handlerDir)
	if err != nil {
		return fmt.Errorf("failed to read handler directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
			srcFile := filepath.Join(handlerDir, entry.Name())
			destFile := filepath.Join(destHandlerDir, entry.Name())

			if err := copyFile(srcFile, destFile); err != nil {
				return fmt.Errorf("failed to copy %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// copyPythonFilesWithProgress copies Python files and reports progress via bus events
func (r *PythonRuntime) copyPythonFilesWithProgress(srcDir, destDir, functionID string) error {
	fileCount := 0

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip certain directories
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip hidden directories, .git, node_modules, etc.
		if strings.HasPrefix(filepath.Base(path), ".") && info.IsDir() {
			return filepath.SkipDir
		}
		if strings.Contains(relPath, "node_modules") || strings.Contains(relPath, "__pycache__") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only copy Python files and directories
		if info.IsDir() {
			destPath := filepath.Join(destDir, relPath)
			return os.MkdirAll(destPath, info.Mode())
		}

		if strings.HasSuffix(path, ".py") {
			destPath := filepath.Join(destDir, relPath)
			fileCount++

			// Ensure destination directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			return copyFile(path, destPath)
		}

		return nil
	})
}

// copyPythonFiles copies Python source files from project root to artifact directory
func (r *PythonRuntime) copyPythonFiles(srcDir, destDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip certain directories
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip hidden directories, .git, node_modules, etc.
		if strings.HasPrefix(filepath.Base(path), ".") && info.IsDir() {
			return filepath.SkipDir
		}
		if strings.Contains(relPath, "node_modules") || strings.Contains(relPath, "__pycache__") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only copy Python files and directories
		if info.IsDir() {
			destPath := filepath.Join(destDir, relPath)
			return os.MkdirAll(destPath, info.Mode())
		}

		if strings.HasSuffix(path, ".py") {
			destPath := filepath.Join(destDir, relPath)

			// Ensure destination directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			return copyFile(path, destPath)
		}

		return nil
	})
}

// installLocalPackages finds and installs local Python packages
func (r *PythonRuntime) installLocalPackages(srcDir, destDir string) error {
	startTime := time.Now()
	slog.Info("installLocalPackages started", "srcDir", srcDir, "destDir", destDir)

	// Find directories with pyproject.toml files (local packages)
	var packages []string

	slog.Info("installLocalPackages: scanning for pyproject.toml files", "elapsed", time.Since(startTime))
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == "pyproject.toml" {
			packageDir := filepath.Dir(path)
			// Skip the root pyproject.toml
			if packageDir != srcDir {
				packages = append(packages, packageDir)
			}
		}

		return nil
	})

	if err != nil {
		slog.Error("installLocalPackages: failed to scan for packages", "elapsed", time.Since(startTime), "error", err)
		return err
	}

	slog.Info("installLocalPackages: found packages", "elapsed", time.Since(startTime), "count", len(packages), "packages", packages)

	// Handle each local package
	for i, packageDir := range packages {
		packageStartTime := time.Now()
		slog.Info("installLocalPackages: processing package", "elapsed", time.Since(startTime), "packageIndex", i+1, "totalPackages", len(packages), "package", packageDir)

		// Special handling for packages with src layout
		packageName := filepath.Base(packageDir)
		srcPath := filepath.Join(packageDir, "src", packageName)

		if _, err := os.Stat(srcPath); err == nil {
			// This package uses src layout (like lib/src/lib/)
			// Copy the inner package directly to the target
			destPackagePath := filepath.Join(destDir, packageName)

			slog.Info("installLocalPackages: copying src-layout package", "elapsed", time.Since(startTime), "packageElapsed", time.Since(packageStartTime), "from", srcPath, "to", destPackagePath)

			err := r.copyDirectory(srcPath, destPackagePath)
			if err != nil {
				slog.Warn("installLocalPackages: failed to copy src-layout package", "elapsed", time.Since(startTime), "packageElapsed", time.Since(packageStartTime), "package", packageDir, "error", err)
				continue
			}

			slog.Info("installLocalPackages: successfully copied src-layout package", "elapsed", time.Since(startTime), "packageElapsed", time.Since(packageStartTime), "package", packageName)
		} else {
			// Try regular pip install for other packages
			slog.Info("installLocalPackages: installing package with uv pip", "elapsed", time.Since(startTime), "packageElapsed", time.Since(packageStartTime), "package", packageDir, "target", destDir)

			cmd := exec.Command("uv", "pip", "install", "-e", packageDir, "--target", destDir)
			cmd.Dir = srcDir

			output, err := cmd.CombinedOutput()
			if err != nil {
				slog.Warn("installLocalPackages: failed to install package", "elapsed", time.Since(startTime), "packageElapsed", time.Since(packageStartTime), "package", packageDir, "error", err, "output", string(output))
				continue
			}

			slog.Info("installLocalPackages: successfully installed package", "elapsed", time.Since(startTime), "packageElapsed", time.Since(packageStartTime), "package", packageDir)
		}
	}

	slog.Info("installLocalPackages completed", "elapsed", time.Since(startTime), "processedPackages", len(packages))
	return nil
}

// flattenWorkspaceLayouts detects and flattens package/src/package structures for all legacy projects
func (r *PythonRuntime) flattenWorkspaceLayouts(artifactDir, functionID string) error {
	// Look for any directories that might have workspace layout (package/src/package)
	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		return err
	}

	var flattened []string

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		packageName := entry.Name()
		packageDir := filepath.Join(artifactDir, packageName)
		srcDir := filepath.Join(packageDir, "src")
		innerPackageDir := filepath.Join(srcDir, packageName)

		// Check if this follows the package/src/package pattern
		if _, err := os.Stat(innerPackageDir); err == nil {

			// Copy contents of package/src/package to package/ (excluding __pycache__)
			entries, err := os.ReadDir(innerPackageDir)
			if err == nil {
				for _, innerEntry := range entries {
					if innerEntry.Name() == "__pycache__" {
						continue
					}

					srcPath := filepath.Join(innerPackageDir, innerEntry.Name())
					destPath := filepath.Join(packageDir, innerEntry.Name())

					if innerEntry.IsDir() {
						err = r.copyDirectory(srcPath, destPath)
					} else {
						err = copyFile(srcPath, destPath)
					}

					if err != nil {
						return fmt.Errorf("failed to flatten %s structure: %w", packageName, err)
					}
				}
				flattened = append(flattened, packageName)
			}
		}
	}

	if len(flattened) > 0 {

	}

	return nil
}

// copyDirectory recursively copies a directory
func (r *PythonRuntime) copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return copyFile(path, destPath)
	})
}
