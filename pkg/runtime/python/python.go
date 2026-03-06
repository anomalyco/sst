package python

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sst/sst/v3/pkg/flag"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/path"
	"github.com/sst/sst/v3/pkg/runtime"
)

// Global sync tracker shared across all builds
var (
	// Global build semaphore to limit concurrent builds
	// This prevents system overload when Pulumi tries to build 100+ functions in parallel
	// Respects SST_BUILD_CONCURRENCY_FUNCTION env var, defaults to 4
	globalBuildSemaphore = make(chan struct{}, parseConcurrency())

	// Global dependency installation locks - ensures only ONE function installs per cache key
	// Other functions wait for the installation to complete, then copy from disk cache
	globalDependencyInstallLocks      = make(map[string]*sync.Mutex)
	globalDependencyInstallLocksMutex sync.Mutex

	// Global requirements.txt generation - generate once per workspace, reuse for all functions
	globalRequirementsFiles      = make(map[string]string) // workspaceDir -> requirements.txt path
	globalRequirementsFilesMutex sync.Mutex

	// Global deps cache clear - clears .deps/ directory once per SST run
	// This ensures workspace package changes are picked up between deploys
	// The deps cache is only meant to be shared within a single deploy run
	globalDepsCacheClearOnce sync.Once
)

// parseConcurrency reads SST_BUILD_CONCURRENCY_FUNCTION (or the deprecated
// SST_BUILD_CONCURRENCY) and returns the desired parallelism, defaulting to 4.
// Panics if the env var is set but not a valid positive integer.
func parseConcurrency() int {
	raw := flag.SST_BUILD_CONCURRENCY_FUNCTION
	if raw == "" {
		raw = flag.SST_BUILD_CONCURRENCY
	}
	if raw == "" {
		return 4
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Sprintf("SST_BUILD_CONCURRENCY_FUNCTION=%q is not a valid integer", raw))
	}
	if n < 1 {
		panic(fmt.Sprintf("SST_BUILD_CONCURRENCY_FUNCTION=%d must be >= 1", n))
	}
	return n
}

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

	// Cache directory for sharing with incremental builder
	cacheDir string

	// Cached incremental builder - reused across all function builds
	incrementalBuilder *IncrementalBuilder

	// Mutex for thread-safe access
	mutex sync.RWMutex
}

func New() *PythonRuntime {
	return &PythonRuntime{
		lastBuiltHandler: map[string]string{},
	}
}

// NewWithCache creates a new Python runtime with caching enabled
func NewWithCache(cacheDir string) (*PythonRuntime, error) {
	runtime := &PythonRuntime{
		lastBuiltHandler: map[string]string{},
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

	if cacheDir == "" {
		return fmt.Errorf("cache directory cannot be empty")
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}

	// Test write permissions
	testFile := filepath.Join(cacheDir, ".test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cache directory %s is not writable: %w", cacheDir, err)
	}
	os.Remove(testFile)

	r.cacheDir = cacheDir

	return nil
}

func (r *PythonRuntime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Clear the deps cache once per SST run (not per function build)
	// This ensures workspace package changes are picked up between deploys
	// The deps cache is only meant to be shared within a single deploy run
	globalDepsCacheClearOnce.Do(func() {
		artifactsDir := filepath.Dir(input.Out())
		depsDir := filepath.Join(artifactsDir, ".deps")
		if _, err := os.Stat(depsDir); err == nil {
			if err := os.RemoveAll(depsDir); err != nil {
				slog.Warn("failed to clear deps cache", "error", err)
			}
		}
	})

	// Acquire semaphore to limit concurrent builds (prevents system overload)
	// This is critical because Pulumi calls Build() for all functions in parallel
	// Note: artifact directories are cleared by runtime.go's Collection.Build() before this is called
	globalBuildSemaphore <- struct{}{} // Acquire
	defer func() {
		<-globalBuildSemaphore
	}() // Release

	// Fast path for dev mode: If we've already built this function, skip the expensive rebuild
	// Check if we have a record of building this handler before
	if input.Dev {
		r.mutex.RLock()
		lastBuilt, hasBuilt := r.lastBuiltHandler[input.FunctionID]
		r.mutex.RUnlock()

		if hasBuilt && lastBuilt != "" {
			// We've built this before, verify the artifact is complete
			// Check if the handler file exists in the artifact directory
			handlerFile := filepath.Join(input.Out(), input.Handler+".py")
			if _, err := os.Stat(handlerFile); err == nil {
				return &runtime.BuildOutput{
					Out:     input.Out(),
					Handler: input.Handler,
					Errors:  []string{},
				}, nil
			}
		}
	}

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
		slog.Error("handler not found",
			"functionID", input.FunctionID,
			"handler", input.Handler,
			"workingDir", path.ResolveRootDir(input.CfgPath),
			"error", err)
		return nil, fmt.Errorf("python runtime - handler not found: %v", err)
	}

	result, err := r.CreateBuildAsset(ctx, input)
	if err != nil {
		return nil, err
	}

	if input.Dev {
		r.mutex.Lock()
		r.lastBuiltHandler[input.FunctionID] = file
		r.mutex.Unlock()
	}

	return result, nil
}

func (r *PythonRuntime) Match(runtime string) bool {
	return strings.HasPrefix(runtime, "python")
}

// ShouldRunEagerly returns false for Python to enable lazy worker startup.
//
// Unlike Node.js (which uses esbuild's metafile for precise per-function dependency tracking),
// Python lacks static import analysis. A change to shared library code (e.g., backend/lib/)
// triggers ShouldRebuild() returning true for ALL functions, not just the ones that import it.
//
// With 50+ functions, eager startup after every file change means:
// - 50+ Python processes starting simultaneously
// - Each takes ~1-2 seconds to import modules and become ready
// - System becomes unresponsive during this startup storm
//
// By returning false, we opt into lazy startup:
// - Workers are stopped and builds invalidated (normal behavior)
// - Workers only restart when actually invoked (just-in-time)
// - Only actively-used functions incur startup cost
// - Idle functions stay stopped until needed
func (r *PythonRuntime) ShouldRunEagerly() bool {
	return false
}

func (r *PythonRuntime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	// Sync artifacts with source before starting the worker
	if err := r.syncArtifactsIfNeeded(input); err != nil {
		slog.Error("failed to sync artifacts",
			"functionID", input.FunctionID,
			"error", err)
		return nil, fmt.Errorf("failed to sync artifacts: %v", err)
	}

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
			slog.Error("failed to copy lambda bridge",
				"functionID", input.FunctionID,
				"source", sourceBridgePath,
				"dest", lambdaBridgePath,
				"error", err)
			return nil, fmt.Errorf("failed to copy lambda bridge: %v", err)
		}
	}

	projectRoot := path.ResolveRootDir(input.CfgPath)
	isLegacyLayout := r.hasWorkspaceLayoutPatterns(projectRoot)

	var handlerPath string
	var workingDir string

	if isLegacyLayout {
		// Legacy layout: files are copied and flattened in artifact directory
		adjustedHandler := r.adjustHandlerForFlattenedLayout(input.Build.Handler)
		handlerPath = filepath.Join(input.Build.Out, adjustedHandler)
		workingDir = input.Build.Out
	} else {
		// Modern layout: run from source with PYTHONPATH
		// Pass the relative handler path since PYTHONPATH is set to projectRoot
		handlerPath = input.Build.Handler
		workingDir = projectRoot
	}

	cmd := process.CommandContext(
		ctx,
		"uv",
		"run",
		"--with=requests",
		lambdaBridgePath,
		handlerPath,
		input.WorkerID,
	)

	// Set up environment
	env := append(input.Env, "AWS_LAMBDA_RUNTIME_API="+input.Server)

	// For modern layouts, set PYTHONPATH to project root and point to resource.enc in artifact dir
	if !isLegacyLayout {
		// Build PYTHONPATH with common Python paths
		pythonPaths := []string{projectRoot}

		// Add src/ if it exists (common Python pattern)
		srcDir := filepath.Join(projectRoot, "src")
		if _, err := os.Stat(srcDir); err == nil {
			pythonPaths = append(pythonPaths, srcDir)
		}

		// Join paths with OS-specific separator (: on Unix, ; on Windows)
		pythonPath := strings.Join(pythonPaths, string(os.PathListSeparator))
		env = append(env, "PYTHONPATH="+pythonPath)

		// Set SST_KEY_FILE to point to resource.enc in the artifact directory
		resourceEncPath := filepath.Join(input.Build.Out, "resource.enc")
		env = append(env, "SST_KEY_FILE="+resourceEncPath)
	}

	cmd.Env = env
	cmd.Dir = workingDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker process: %v", err)
	}

	return &Worker{
		stdout,
		stderr,
		cmd,
	}, nil

}

func (r *PythonRuntime) ShouldRebuild(functionID string, file string) bool {
	// Simple implementation: rebuild if it's a relevant Python file that we shouldn't ignore.
	//
	// We intentionally don't do complex per-function dependency tracking here because:
	// 1. Python imports are dynamic and hard to analyze statically
	// 2. Build() is very fast (~50µs in dev mode)
	// 3. With ShouldRunEagerly() returning false, workers start lazily on-demand
	//    so we don't pay the ~1-2s Run() cost for idle functions
	//
	// The combination of fast builds + lazy startup means we can afford to rebuild
	// all functions on any Python file change without performance issues.

	if r.shouldIgnoreFile(file) {
		return false
	}

	if !r.isRelevantFile(file) {
		return false
	}

	return true
}

// syncArtifactsIfNeeded checks if artifacts need to be synced with source and does it if necessary
func (r *PythonRuntime) syncArtifactsIfNeeded(input *runtime.RunInput) error {
	projectRoot := path.ResolveRootDir(input.CfgPath)
	artifactDir := input.Build.Out

	// Only sync files for legacy workspace layouts that need flattening
	// Modern layouts (monorepo, standard) run directly from source via PYTHONPATH
	if r.hasWorkspaceLayoutPatterns(projectRoot) {
		if err := r.syncPythonFiles(input.FunctionID, projectRoot, artifactDir); err != nil {
			return err
		}

		// After syncing, flatten workspace layouts
		// This ensures the artifact directory has the correct structure for Python imports
		return r.flattenWorkspaceLayouts(artifactDir, input.FunctionID)
	}

	return nil
}

// syncPythonFiles syncs Python files from source to artifacts (adds, updates, deletes)
func (r *PythonRuntime) syncPythonFiles(functionID, srcDir, destDir string) error {
	// First, collect all Python files in source
	sourceFiles := make(map[string]os.FileInfo)
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip hidden directories and common non-Python directories
		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(path), ".") ||
				strings.Contains(relPath, "node_modules") ||
				strings.Contains(relPath, "__pycache__") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only track Python files
		if strings.HasSuffix(path, ".py") {
			sourceFiles[relPath] = info
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan source files: %v", err)
	}

	// Collect all Python files in artifacts
	artifactFiles := make(map[string]os.FileInfo)
	if _, err := os.Stat(destDir); err == nil {
		err = filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(destDir, path)
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(path, ".py") {
				artifactFiles[relPath] = info
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to scan artifact files: %v", err)
		}
	}

	// Delete files that exist in artifacts but not in source
	for relPath := range artifactFiles {
		if _, exists := sourceFiles[relPath]; !exists {
			artifactPath := filepath.Join(destDir, relPath)
			if err := os.Remove(artifactPath); err != nil {
				return fmt.Errorf("failed to delete %s: %v", artifactPath, err)
			}
		}
	}

	// Copy/update files that exist in source
	for relPath, sourceInfo := range sourceFiles {
		sourcePath := filepath.Join(srcDir, relPath)
		artifactPath := filepath.Join(destDir, relPath)

		// Check if we need to copy (file doesn't exist or is older)
		needsCopy := true
		if artifactInfo, exists := artifactFiles[relPath]; exists {
			if !sourceInfo.ModTime().After(artifactInfo.ModTime()) {
				needsCopy = false
			}
		}

		if needsCopy {
			// Ensure destination directory exists
			if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %v", artifactPath, err)
			}

			// Copy the file
			if err := copyFile(sourcePath, artifactPath); err != nil {
				return fmt.Errorf("failed to copy %s: %v", relPath, err)
			}
		}
	}

	return nil
}

// adjustPathForFlattenedLayout adjusts a file path to account for flattened workspace layouts
// For example: package/src/package/file.py -> package/file.py
func (r *PythonRuntime) adjustPathForFlattenedLayout(destDir, relPath string) string {
	// Check if the path contains the package/src/package pattern
	parts := strings.Split(relPath, string(filepath.Separator))

	// Need at least 3 parts for package/src/package pattern
	if len(parts) < 3 {
		return filepath.Join(destDir, relPath)
	}

	// Look for src in the path
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "src" && i > 0 && i < len(parts)-1 {
			// Check if the part after "src" matches the part before "src"
			packageName := parts[i-1]
			nextPart := parts[i+1]

			if packageName == nextPart {
				// This is a package/src/package pattern, flatten it
				// Remove the "src/package" part
				flattenedParts := append(parts[:i], parts[i+2:]...)
				flattenedPath := filepath.Join(flattenedParts...)

				return filepath.Join(destDir, flattenedPath)
			}
		}
	}

	// No flattening needed
	return filepath.Join(destDir, relPath)
}

// adjustHandlerForFlattenedLayout adjusts a handler path to account for flattened workspace layouts
// For example: functions/src/functions/user/handler.lambda_handler -> functions/user/handler.lambda_handler
func (r *PythonRuntime) adjustHandlerForFlattenedLayout(handlerPath string) string {
	// Split handler path into file path and function name
	// Format: path/to/file.function_name
	lastDot := strings.LastIndex(handlerPath, ".")
	if lastDot == -1 {
		// No function name, just adjust the path
		return r.adjustHandlerPathOnly(handlerPath)
	}

	filePath := handlerPath[:lastDot]
	functionName := handlerPath[lastDot+1:]

	// Adjust the file path
	adjustedFilePath := r.adjustHandlerPathOnly(filePath)

	// Reconstruct handler path
	adjustedHandler := adjustedFilePath + "." + functionName

	return adjustedHandler
}

// adjustHandlerPathOnly adjusts just the path portion of a handler
// For example: functions/src/functions/user/handler -> functions/user/handler
func (r *PythonRuntime) adjustHandlerPathOnly(handlerPath string) string {
	// Check if the path contains the package/src/package pattern
	parts := strings.Split(handlerPath, "/")

	// Need at least 3 parts for package/src/package pattern
	if len(parts) < 3 {
		return handlerPath
	}

	// Look for src in the path
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "src" && i > 0 && i < len(parts)-1 {
			// Check if the part after "src" matches the part before "src"
			packageName := parts[i-1]
			nextPart := parts[i+1]

			if packageName == nextPart {
				// This is a package/src/package pattern, flatten it
				// Remove the "src/package" part
				flattenedParts := append(parts[:i], parts[i+2:]...)
				flattenedPath := strings.Join(flattenedParts, "/")

				return flattenedPath
			}
		}
	}

	// No flattening needed
	return handlerPath
}

// isRelevantFile checks if a file change is relevant for Python functions
func (r *PythonRuntime) isRelevantFile(file string) bool {
	// Quick exclusions first - this prevents infinite rebuild loops
	if r.shouldIgnoreFile(file) {
		return false
	}

	// ONLY Python-related files are relevant for Python runtime
	// Frontend/infrastructure files (.ts, .js, .json, .vue) are NOT relevant
	relevantExtensions := []string{".py", ".toml", ".lock", ".cfg"}
	relevantFiles := []string{"pyproject.toml", "requirements.txt", "uv.lock", "poetry.lock", "Pipfile.lock", "setup.py", "setup.cfg"}

	// Check Python file extensions
	for _, ext := range relevantExtensions {
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

	return false
}

// shouldIgnoreFile determines if a file should be ignored to prevent infinite rebuild loops
func (r *PythonRuntime) shouldIgnoreFile(file string) bool {
	// Normalize path separators for consistent matching
	normalizedFile := filepath.ToSlash(file)

	// Ignore build artifacts and cache directories that could cause feedback loops
	ignorePaths := []string{
		".sst",          // SST cache and build artifacts (matches .sst and .sst/*)
		"__pycache__",   // Python bytecode cache
		".pytest_cache", // Pytest cache
		".mypy_cache",   // MyPy cache
		".coverage",     // Coverage files
		"build",         // Build directories
		"dist",          // Distribution directories
		".git",          // Git directory
		"node_modules",  // Node modules
		".venv",         // Virtual environments
		"venv",
		"env",
		".tox",      // Tox cache
		".eggs",     // Egg cache
		".egg-info", // Egg info
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
	// Split path into parts and check each part
	pathParts := strings.Split(normalizedFile, "/")
	for _, part := range pathParts {
		for _, ignorePath := range ignorePaths {
			if part == ignorePath || strings.HasPrefix(part, ignorePath) {
				return true
			}
		}
	}

	// Check if file has an ignored extension
	for _, ext := range ignoreExtensions {
		if strings.HasSuffix(file, ext) {
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
				return true
			}
		}
	}

	return false
}

func (r *PythonRuntime) CreateBuildAsset(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	if input.Dev {
		// Dev mode: Copy source files but skip dependency management
		return r.createSimpleDevBuild(ctx, input)
	}

	// Deployment mode: Full build with dependency management
	type Properties struct {
		Architecture string          `json:"architecture"`
		Container    json.RawMessage `json:"container"`
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
	startTime := time.Now()

	// CRITICAL OPTIMIZATION: Reuse the IncrementalBuilder across all function builds
	// Creating a new IncrementalBuilder for each of 50+ functions was causing massive
	// CPU/memory overhead. Reuse a single builder per (cacheDir, artifactDir) pair.

	r.mutex.Lock()
	if r.incrementalBuilder == nil {
		cacheDir := r.cacheDir
		if cacheDir == "" {
			if input.Dev {
				cacheDir = filepath.Join(workingDir, ".sst/cache/dev")
			} else {
				cacheDir = filepath.Join(workingDir, ".sst/cache/deploy")
			}
		}

		var err error
		r.incrementalBuilder, err = NewIncrementalBuilder(IncrementalBuilderConfig{
			CacheDir:    cacheDir,
			ArtifactDir: input.Out(),
			FunctionID:  input.FunctionID,
			ProjectRoot: workingDir,
		})
		if err != nil {
			r.mutex.Unlock()
			return nil, fmt.Errorf("failed to create incremental builder: %w", err)
		}
	}
	incrementalBuilder := r.incrementalBuilder
	r.mutex.Unlock()

	// Use the shared incremental builder
	result, err := incrementalBuilder.Build(ctx, input)

	elapsed := time.Since(startTime)
	if err != nil {
		slog.Error("build failed", "functionID", input.FunctionID, "elapsed", elapsed, "error", err)
		return nil, err
	}

	slog.Info("built", "function", input.FunctionID, "elapsed", elapsed)

	return result, nil
}

func (r *PythonRuntime) getFile(input *runtime.BuildInput) (string, error) {
	rootDir := path.ResolveRootDir(input.CfgPath)

	// Handler format is: path/to/file.function_name
	lastDotIndex := strings.LastIndex(input.Handler, ".")
	if lastDotIndex == -1 {
		return "", fmt.Errorf("invalid handler format '%s': expected 'path/to/file.function_name'", input.Handler)
	}

	filePath := input.Handler[:lastDotIndex]

	// Look for .py file
	pythonFile := filepath.Join(rootDir, filePath+".py")
	if _, err := os.Stat(pythonFile); err == nil {
		return pythonFile, nil
	}

	// No Python file found — list what exists to help debug
	dirPath := filepath.Join(rootDir, filepath.Dir(filePath))
	if entries, err := os.ReadDir(dirPath); err == nil {
		var files []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
				files = append(files, entry.Name())
			}
		}
		slog.Error("handler file not found", "expected", pythonFile, "pyFilesInDir", files)
	}

	return "", fmt.Errorf("could not find Python file '%s.py' for handler '%s' (looked in: %s)",
		filepath.Base(filePath),
		input.Handler,
		pythonFile)
}

// copyFile copies a single file from src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", dst, err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

// copyDir recursively copies a directory. If a ContentFilter is provided, it is used
// to exclude files/directories. The optional filterPrefix is prepended to relative paths
// when checking the filter (e.g. pass "backend" so "tests/" matches "backend/tests/**").
func copyDir(src, dst string, filter *ContentFilter, filterPrefix string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)

		// Always skip __pycache__ and .pyc files
		if info.IsDir() && info.Name() == "__pycache__" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".pyc") {
			return nil
		}

		// Apply content filter if provided
		if filter != nil {
			filterPath := relPath
			if filterPrefix != "" && relPath != "." {
				filterPath = filepath.Join(filterPrefix, relPath)
			}
			if filter.ShouldExclude(filterPath) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyDirUnfiltered copies a directory recursively without content filtering.
// It only skips __pycache__, .venv, node_modules, and .git directories.
// Used for container builds where pyproject.toml and metadata must be preserved.
func copyDirUnfiltered(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			if name == "__pycache__" || name == ".venv" || name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// hashFileContents computes a SHA256 hash of a file's contents.
func hashFileContents(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// createSimpleDevBuild creates a simple build for development mode by doing nothing
// All work is deferred to Run() for just-in-time execution
func (r *PythonRuntime) createSimpleDevBuild(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Create artifact directory
	if err := os.MkdirAll(input.Out(), 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %v", err)
	}

	// In dev mode, don't copy any files during build
	// All copying and flattening happens just-in-time in Run()
	return &runtime.BuildOutput{
		Handler:    input.Handler,
		Sourcemaps: []string{},
		Errors:     []string{},
		Out:        input.Out(),
	}, nil
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

// flattenWorkspaceLayouts detects and flattens package/src/package structures for all legacy projects
func (r *PythonRuntime) flattenWorkspaceLayouts(artifactDir, functionID string) error {
	contentFilter := NewContentFilter()
	var flattened []string

	// flattenDir checks all immediate subdirectories of dir for the package/src/package pattern
	flattenDir := func(dir string) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == "__pycache__" || entry.Name() == "node_modules" {
				continue
			}

			packageName := entry.Name()
			packageDir := filepath.Join(dir, packageName)
			srcDir := filepath.Join(packageDir, "src")
			innerPackageDir := filepath.Join(srcDir, packageName)

			// Check if this follows the package/src/package pattern
			if _, err := os.Stat(innerPackageDir); err == nil {
				// Copy contents of package/src/package to package/
				innerEntries, err := os.ReadDir(innerPackageDir)
				if err != nil {
					slog.Warn("failed to read inner package dir", "package", packageName, "error", err)
					continue
				}

				for _, innerEntry := range innerEntries {
					if contentFilter.ShouldExclude(innerEntry.Name()) {
						continue
					}

					srcPath := filepath.Join(innerPackageDir, innerEntry.Name())
					destPath := filepath.Join(packageDir, innerEntry.Name())

					if innerEntry.IsDir() {
						err = copyDir(srcPath, destPath, contentFilter, "")
					} else {
						err = copyFile(srcPath, destPath)
					}

					if err != nil {
						return fmt.Errorf("failed to flatten %s structure: %w", packageName, err)
					}
				}

				// Remove the old src/ directory after flattening to avoid import confusion
				if err := os.RemoveAll(srcDir); err != nil {
					slog.Warn("failed to remove src/ after flattening", "package", packageName, "error", err)
				}

				relPath, _ := filepath.Rel(artifactDir, packageDir)
				flattened = append(flattened, relPath)
			}
		}
		return nil
	}

	// Check top-level directories
	if err := flattenDir(artifactDir); err != nil {
		return err
	}

	// Also check one level deeper (e.g., packages/api/src/api pattern)
	topEntries, err := os.ReadDir(artifactDir)
	if err != nil {
		return err
	}
	for _, entry := range topEntries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == "__pycache__" || entry.Name() == "node_modules" {
			continue
		}
		subDir := filepath.Join(artifactDir, entry.Name())
		if err := flattenDir(subDir); err != nil {
			return err
		}
	}

	return nil
}
