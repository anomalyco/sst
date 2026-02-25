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
	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/events"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/path"
	"github.com/sst/sst/v3/pkg/runtime"
)

// Global sync tracker shared across all builds
var (
	globalSyncCompleted = make(map[string]bool)
	globalSyncMutex     sync.RWMutex

	// Global build semaphore to limit concurrent builds
	// This prevents system overload when Pulumi tries to build 100+ functions in parallel
	globalBuildSemaphore = make(chan struct{}, 4) // Allow max 4 concurrent builds

	// Global dependency analysis cache - shared across ALL function builds
	// Key is the pyproject.toml path, value is the cached analysis
	globalDependencyCache      = make(map[string]*DependencyAnalysis)
	globalDependencyCacheMutex sync.RWMutex

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

	// Cached incremental builder - reused across all function builds
	// This is critical for performance: creating a new IncrementalBuilder for each
	// of 50+ functions was causing massive CPU/memory overhead
	incrementalBuilder *IncrementalBuilder

	// Track when each function was last rebuilt - used to skip file changes
	// that occurred before the last rebuild (mtime comparison)
	lastRebuildTime map[string]time.Time

	// Track pending changes per function
	pendingChanges map[string][]string // functionID -> list of changed files

	// Track if we've already signaled a restart for pending changes
	// This prevents showing "LazyRestart" when worker is actually running
	restartSignaled map[string]bool // functionID -> whether we've returned a dummy worker

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
	// Check if uv sync is needed (one-time check on initialization)
	if cwd, err := os.Getwd(); err == nil {
		checkUvSyncNeeded(cwd)
	}

	return &PythonRuntime{
		lastBuiltHandler: map[string]string{},
		lastRebuildTime:  map[string]time.Time{},
		pendingChanges:   map[string][]string{},
		restartSignaled:  map[string]bool{},
	}
}

// NewWithCache creates a new Python runtime with caching enabled
func NewWithCache(cacheDir string) (*PythonRuntime, error) {
	runtime := &PythonRuntime{
		lastBuiltHandler: map[string]string{},
		lastRebuildTime:  map[string]time.Time{},
		pendingChanges:   map[string][]string{},
		restartSignaled:  map[string]bool{},
	}

	// Run uv sync if needed (one-time on initialization)
	if cwd, err := os.Getwd(); err == nil {
		checkUvSyncNeeded(cwd)
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
	// Clear the deps cache once per SST run (not per function build)
	// This ensures workspace package changes are picked up between deploys
	// The deps cache is only meant to be shared within a single deploy run
	globalDepsCacheClearOnce.Do(func() {
		artifactsDir := filepath.Dir(input.Out())
		depsDir := filepath.Join(artifactsDir, ".deps")
		if _, err := os.Stat(depsDir); err == nil {
			slog.Info("clearing deps cache for new SST run", "depsDir", depsDir)
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
		bus.Publish(&events.FunctionBuildProgressEvent{
			FunctionID: input.FunctionID,
			Stage:      "BuildFailed",
			Message:    fmt.Sprintf("Handler not found: %s\n  Error: %v\n  Working dir: %s", input.Handler, err, path.ResolveRootDir(input.CfgPath)),
		})
		return nil, fmt.Errorf("python runtime - handler not found: %v", err)
	}

	if input.Dev {
		// Development mode: Simple build without complex dependency management
		result, err := r.CreateBuildAsset(ctx, input)
		if err != nil {
			bus.Publish(&events.FunctionBuildProgressEvent{
				FunctionID: input.FunctionID,
				Stage:      "BuildFailed",
				Message:    fmt.Sprintf("Build failed: %s\n  Error: %v", input.Handler, err),
			})
			return nil, err
		}

		r.mutex.Lock()
		r.lastBuiltHandler[input.FunctionID] = file
		r.lastRebuildTime[input.FunctionID] = time.Now()
		// Clear pending changes and restart signal after successful build
		delete(r.pendingChanges, input.FunctionID)
		delete(r.restartSignaled, input.FunctionID)
		r.mutex.Unlock()

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
		Uv struct {
			Package   bool `toml:"package"`
			Workspace struct {
				Members []string `toml:"members"`
			} `toml:"workspace"`
			Sources map[string]interface{} `toml:"sources"`
		} `toml:"uv"`
	} `toml:"tool"`
}

func (r *PythonRuntime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	runStart := time.Now()

	// Clear any pending state from ShouldRebuild
	r.mutex.Lock()
	delete(r.pendingChanges, input.FunctionID)
	delete(r.restartSignaled, input.FunctionID)
	r.mutex.Unlock()

	// Sync artifacts with source before starting the worker
	if err := r.syncArtifactsIfNeeded(input); err != nil {
		bus.Publish(&events.FunctionBuildProgressEvent{
			FunctionID: input.FunctionID,
			Stage:      "RunFailed",
			Message:    fmt.Sprintf("Failed to sync artifacts: %v", err),
		})
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
			bus.Publish(&events.FunctionBuildProgressEvent{
				FunctionID: input.FunctionID,
				Stage:      "RunFailed",
				Message:    fmt.Sprintf("Failed to copy lambda bridge: %v\n  Source: %s\n  Dest: %s", err, sourceBridgePath, lambdaBridgePath),
			})
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

	cmd.Start()

	// Publish RunComplete event
	bus.Publish(&events.FunctionBuildProgressEvent{
		FunctionID: input.FunctionID,
		Stage:      "RunComplete",
		Message:    fmt.Sprintf("Started %s in %v", input.Build.Handler, time.Since(runStart)),
	})

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

// appendIfMissing appends a string to a slice if it's not already present
func appendIfMissing(slice []string, s string) []string {
	for _, item := range slice {
		if item == s {
			return slice
		}
	}
	return append(slice, s)
}

// syncArtifactsIfNeeded checks if artifacts need to be synced with source and does it if necessary
func (r *PythonRuntime) syncArtifactsIfNeeded(input *runtime.RunInput) error {
	projectRoot := path.ResolveRootDir(input.CfgPath)
	artifactDir := input.Build.Out

	// Only sync files for legacy workspace layouts that need flattening
	// Modern layouts (monorepo, standard) run directly from source via PYTHONPATH
	if r.hasWorkspaceLayoutPatterns(projectRoot) {
		slog.Info("legacy workspace layout detected, syncing files",
			"functionID", input.FunctionID,
			"projectRoot", projectRoot)

		if err := r.syncPythonFiles(input.FunctionID, projectRoot, artifactDir); err != nil {
			return err
		}

		// After syncing, flatten workspace layouts
		// This ensures the artifact directory has the correct structure for Python imports
		return r.flattenWorkspaceLayouts(artifactDir, input.FunctionID)
	}

	slog.Info("modern layout detected, skipping file sync (running from source)",
		"functionID", input.FunctionID,
		"projectRoot", projectRoot)

	// For modern layouts, we don't sync files but we still need to clear pending changes
	// since the worker will run directly from source
	r.mutex.Lock()
	if len(r.pendingChanges[input.FunctionID]) > 0 {
		slog.Info("clearing pending changes for modern layout",
			"functionID", input.FunctionID,
			"pendingChanges", len(r.pendingChanges[input.FunctionID]))
		delete(r.pendingChanges, input.FunctionID)
		delete(r.restartSignaled, input.FunctionID)
	}
	r.mutex.Unlock()

	return nil
}

// syncPythonFiles syncs Python files from source to artifacts (adds, updates, deletes)
func (r *PythonRuntime) syncPythonFiles(functionID, srcDir, destDir string) error {
	// Check if artifact directory is empty or nearly empty (first build)
	// If so, do a full sync regardless of pending changes
	entries, err := os.ReadDir(destDir)
	if err != nil || len(entries) <= 1 { // <= 1 because lambdaric_python_bridge.py might be there
		slog.Info("artifact directory empty or nearly empty, doing full sync",
			"functionID", functionID,
			"destDir", destDir)
		// Clear any pending changes since we're doing a full sync anyway
		r.mutex.Lock()
		delete(r.pendingChanges, functionID)
		delete(r.restartSignaled, functionID)
		r.mutex.Unlock()
		// Fall through to full sync below
	} else {
		// Check if we have pending changes to sync
		r.mutex.Lock()
		pendingFiles := r.pendingChanges[functionID]
		hasPendingChanges := len(pendingFiles) > 0

		// CRITICAL FIX: Clear pending changes immediately after reading them
		// This prevents accumulation if multiple syncs happen
		if hasPendingChanges {
			// Make a copy of the pending files list
			pendingFilesCopy := make([]string, len(pendingFiles))
			copy(pendingFilesCopy, pendingFiles)

			// Clear the pending changes map immediately
			delete(r.pendingChanges, functionID)
			delete(r.restartSignaled, functionID)

			// Use the copy for syncing
			pendingFiles = pendingFilesCopy

			slog.Info("cleared pending changes before sync (preventing accumulation)",
				"functionID", functionID,
				"fileCount", len(pendingFiles))
		}
		r.mutex.Unlock()

		// If we have specific pending changes, only sync those files
		if hasPendingChanges {
			slog.Info("syncing only changed Python files",
				"functionID", functionID,
				"count", len(pendingFiles))

			for _, relPath := range pendingFiles {
				sourcePath := filepath.Join(srcDir, relPath)

				// For legacy projects with workspace layouts, we need to handle flattened paths
				// If the source path contains package/src/package pattern, adjust the artifact path
				artifactPath := r.adjustPathForFlattenedLayout(destDir, relPath)

				// Check if source file exists
				if _, err := os.Stat(sourcePath); err != nil {
					slog.Warn("source file not found, skipping sync",
						"file", relPath,
						"sourcePath", sourcePath,
						"error", err)
					continue
				}

				// Ensure destination directory exists
				if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
					return fmt.Errorf("failed to create directory for %s: %v", artifactPath, err)
				}

				// Copy the file
				if err := copyFile(sourcePath, artifactPath); err != nil {
					return fmt.Errorf("failed to copy %s: %v", relPath, err)
				}
				slog.Info("synced changed file",
					"file", relPath,
					"sourcePath", sourcePath,
					"artifactPath", artifactPath)
			}

			slog.Info("completed sync of changed files",
				"functionID", functionID,
				"syncedCount", len(pendingFiles))

			return nil
		}
	}

	// No pending changes, do full sync (first time or after deploy)
	slog.Info("performing full Python file sync", "functionID", functionID)

	// First, collect all Python files in source
	sourceFiles := make(map[string]os.FileInfo)
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
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
			slog.Info("deleted stale artifact", "file", relPath)
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
			slog.Info("synced artifact", "file", relPath)
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

				slog.Debug("adjusted path for flattened layout",
					"original", relPath,
					"flattened", flattenedPath)

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

	if adjustedHandler != handlerPath {
		slog.Info("adjusted handler path for flattened layout",
			"original", handlerPath,
			"adjusted", adjustedHandler)
	}

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

// isFileRelatedToFunction checks if a changed file is actually related to a specific function
// This prevents rebuilding ALL functions when ANY .py file changes
func (r *PythonRuntime) isFileRelatedToFunction(functionID string, file string) bool {
	// Get the handler path for this function
	r.mutex.RLock()
	handlerPath, hasHandler := r.lastBuiltHandler[functionID]
	r.mutex.RUnlock()

	if !hasHandler || handlerPath == "" {
		// No handler info yet - this is likely the first build
		// Be conservative and return true to allow the build
		slog.Debug("isFileRelatedToFunction: no handler info, allowing rebuild",
			"functionID", functionID,
			"file", file)
		return true
	}

	// Normalize both paths for comparison
	absFile, err := filepath.Abs(file)
	if err != nil {
		absFile = file
	}
	absHandlerPath, err := filepath.Abs(handlerPath)
	if err != nil {
		absHandlerPath = handlerPath
	}

	// Check if the changed file IS the handler file
	if absFile == absHandlerPath {
		slog.Debug("isFileRelatedToFunction: file is the handler",
			"functionID", functionID,
			"file", absFile)
		return true
	}

	// Get the directory containing the handler
	handlerDir := filepath.Dir(absHandlerPath)

	// Check if the changed file is in the SAME directory as the handler
	// (not subdirectories - just the same directory)
	fileDir := filepath.Dir(absFile)
	if fileDir == handlerDir {
		slog.Debug("isFileRelatedToFunction: file is in same directory as handler",
			"functionID", functionID,
			"file", absFile,
			"handlerDir", handlerDir)
		return true
	}

	// Check if the changed file is in a parent directory of the handler
	// This catches changes to __init__.py files in parent packages
	if strings.HasPrefix(handlerDir, fileDir) {
		slog.Debug("isFileRelatedToFunction: file is in parent directory of handler",
			"functionID", functionID,
			"file", absFile,
			"fileDir", fileDir,
			"handlerDir", handlerDir)
		return true
	}

	// For global config files, only rebuild if they're at the project root
	// (not in some random subdirectory)
	basename := filepath.Base(file)
	globalFiles := []string{"pyproject.toml", "uv.lock", "requirements.txt"}
	for _, globalFile := range globalFiles {
		if basename == globalFile {
			// Only trigger if this is likely a root-level config
			// Check if the file is NOT deep in a subdirectory
			relPath, err := filepath.Rel(handlerDir, absFile)
			if err == nil && strings.HasPrefix(relPath, "..") {
				// File is above the handler directory - likely project root
				slog.Debug("isFileRelatedToFunction: global config file changed",
					"functionID", functionID,
					"file", file)
				return true
			}
		}
	}

	slog.Debug("isFileRelatedToFunction: file not related to this function",
		"functionID", functionID,
		"file", absFile,
		"handlerPath", absHandlerPath)
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
				slog.Debug("ignoring file due to path pattern",
					"file", file,
					"pattern", ignorePath,
					"matchedPart", part)
				return true
			}
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
	// CPU/memory overhead because each one creates BuildCache, ChangeDetector,
	// DependencyAnalyzer, UvCommandRunner, DependencyCache, etc.

	r.mutex.Lock()
	if r.incrementalBuilder == nil {
		// Create incremental builder with sensible defaults - only once
		factory := NewRuntimeFactory()

		// Progress callback - only report errors to dev screen
		progressCallback := func(event ProgressEvent) {
			// Only publish error/failure events, not progress
			if strings.Contains(strings.ToLower(event.Stage), "error") ||
				strings.Contains(strings.ToLower(event.Stage), "fail") {
				bus.Publish(&events.FunctionBuildProgressEvent{
					FunctionID: "python-build",
					Stage:      event.Stage,
					Message:    event.Message,
				})
			}
		}

		var err error
		if r.cacheDir != "" {
			r.incrementalBuilder, err = factory.CreateIncrementalBuilderWithCacheDir(workingDir, input, progressCallback, r.cacheDir)
		} else {
			r.incrementalBuilder, err = factory.CreateIncrementalBuilder(workingDir, input, progressCallback)
		}
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
		slog.Error("❌ Build failed",
			"functionID", input.FunctionID,
			"elapsed", elapsed,
			"error", err)
		return nil, err
	}

	slog.Info("✅ Built", "function", input.FunctionID, "elapsed", elapsed)

	return result, nil
}

func (r *PythonRuntime) getFile(input *runtime.BuildInput) (string, error) {
	startTime := time.Now()
	slog.Info("getFile started", "handler", input.Handler)

	rootDir := path.ResolveRootDir(input.CfgPath)

	// Handler format is: path/to/file.function_name
	// We need to split on the LAST dot to separate file path from function name
	lastDotIndex := strings.LastIndex(input.Handler, ".")
	if lastDotIndex == -1 {
		return "", fmt.Errorf("invalid handler format '%s': expected 'path/to/file.function_name'", input.Handler)
	}

	// Everything before the last dot is the file path (without .py extension)
	filePath := input.Handler[:lastDotIndex]
	functionName := input.Handler[lastDotIndex+1:]

	slog.Info("getFile: parsed handler",
		"elapsed", time.Since(startTime),
		"filePath", filePath,
		"functionName", functionName,
		"rootDir", rootDir)

	// Look for .py file
	pythonFile := filepath.Join(rootDir, filePath+".py")
	slog.Info("getFile: checking for Python file", "elapsed", time.Since(startTime), "pythonFile", pythonFile)

	if _, err := os.Stat(pythonFile); err == nil {
		slog.Info("getFile: found Python file", "elapsed", time.Since(startTime), "pythonFile", pythonFile)
		return pythonFile, nil
	}

	// No Python file found for the handler
	slog.Error("getFile: Python file not found", "elapsed", time.Since(startTime), "expectedFile", pythonFile)

	// List what files DO exist in the directory to help debug
	dirPath := filepath.Join(rootDir, filepath.Dir(filePath))
	if entries, err := os.ReadDir(dirPath); err == nil {
		var files []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
				files = append(files, entry.Name())
			}
		}
		slog.Error("getFile: Python files in directory", "dir", dirPath, "files", files)
	}

	return "", fmt.Errorf("could not find Python file '%s.py' for handler '%s' (looked in: %s)",
		filepath.Base(filePath),
		input.Handler,
		pythonFile)
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

	pyprojectPath := filepath.Join(workspaceDir, "pyproject.toml")

	// Read the pyproject.toml file
	pyproject, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return "", fmt.Errorf("failed to read pyproject.toml file: %v", err)
	}

	// Parse the pyproject.toml file
	pyprojectData := PyProject{}
	err = toml.Unmarshal(pyproject, &pyprojectData)
	if err != nil {
		return "", fmt.Errorf("failed to parse pyproject.toml file: %v", err)
	}

	packageName := pyprojectData.Project.Name
	if packageName == "" {
		return "", fmt.Errorf("no project name found in pyproject.toml at %s", pyprojectPath)
	}

	slog.Info("resolved package name from pyproject.toml",
		"packageName", packageName,
		"pyprojectPath", pyprojectPath,
		"functionID", input.FunctionID)

	return packageName, nil

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

		// Skip virtual environments and other unwanted directories
		if info.IsDir() {
			dirName := filepath.Base(path)
			skipDirs := []string{
				"node_modules",
				"__pycache__",
				"venv",
				"env",
				".venv",
				".env",
				".tox",
				"build",
				"dist",
				".pytest_cache",
				".mypy_cache",
			}
			for _, skipDir := range skipDirs {
				if dirName == skipDir {
					slog.Debug("skipping directory during Python file copy", "dir", relPath)
					return filepath.SkipDir
				}
			}
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
	// Create debug log file
	logFile, err := os.Create(filepath.Join(artifactDir, "flatten_debug.log"))
	if err == nil {
		defer logFile.Close()
		logFile.WriteString(fmt.Sprintf("=== Flatten Debug Log ===\n"))
		logFile.WriteString(fmt.Sprintf("artifactDir: %s\n", artifactDir))
		logFile.WriteString(fmt.Sprintf("functionID: %s\n\n", functionID))
	}

	// Create a content filter to exclude test files during flattening
	contentFilter := NewContentFilter()
	// Look for any directories that might have workspace layout (package/src/package)
	entries, err := os.ReadDir(artifactDir)
	if err != nil {
		if logFile != nil {
			logFile.WriteString(fmt.Sprintf("ERROR: Failed to read artifactDir: %v\n", err))
		}
		return err
	}

	if logFile != nil {
		logFile.WriteString(fmt.Sprintf("Found %d entries in artifactDir\n", len(entries)))
	}

	var flattened []string

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			if logFile != nil {
				logFile.WriteString(fmt.Sprintf("Skipping: %s (isDir=%v, hidden=%v)\n", entry.Name(), entry.IsDir(), strings.HasPrefix(entry.Name(), ".")))
			}
			continue
		}

		packageName := entry.Name()
		packageDir := filepath.Join(artifactDir, packageName)
		srcDir := filepath.Join(packageDir, "src")
		innerPackageDir := filepath.Join(srcDir, packageName)

		if logFile != nil {
			logFile.WriteString(fmt.Sprintf("\nChecking package: %s\n", packageName))
			logFile.WriteString(fmt.Sprintf("  packageDir: %s\n", packageDir))
			logFile.WriteString(fmt.Sprintf("  srcDir: %s\n", srcDir))
			logFile.WriteString(fmt.Sprintf("  innerPackageDir: %s\n", innerPackageDir))
		}

		// Check if this follows the package/src/package pattern
		if _, err := os.Stat(innerPackageDir); err == nil {
			if logFile != nil {
				logFile.WriteString(fmt.Sprintf("  ✓ Pattern matches! Flattening %s/src/%s -> %s\n", packageName, packageName, packageName))
			}

			// Copy contents of package/src/package to package/ (excluding test files and build artifacts)
			innerEntries, err := os.ReadDir(innerPackageDir)
			if err == nil {
				if logFile != nil {
					logFile.WriteString(fmt.Sprintf("  Found %d entries in innerPackageDir\n", len(innerEntries)))
				}

				for _, innerEntry := range innerEntries {
					if logFile != nil {
						logFile.WriteString(fmt.Sprintf("    Processing: %s (isDir=%v)\n", innerEntry.Name(), innerEntry.IsDir()))
					}

					// Use ContentFilter to determine if this file/directory should be excluded
					if contentFilter.ShouldExclude(innerEntry.Name()) {
						if logFile != nil {
							logFile.WriteString(fmt.Sprintf("      ✗ Excluded by filter\n"))
						}
						continue
					}

					srcPath := filepath.Join(innerPackageDir, innerEntry.Name())
					destPath := filepath.Join(packageDir, innerEntry.Name())

					if logFile != nil {
						logFile.WriteString(fmt.Sprintf("      Copying: %s -> %s\n", srcPath, destPath))
					}

					if innerEntry.IsDir() {
						err = r.copyDirectoryWithFilter(srcPath, destPath, contentFilter)
					} else {
						err = copyFile(srcPath, destPath)
					}

					if err != nil {
						if logFile != nil {
							logFile.WriteString(fmt.Sprintf("      ✗ ERROR: %v\n", err))
						}
						return fmt.Errorf("failed to flatten %s structure: %w", packageName, err)
					}

					if logFile != nil {
						logFile.WriteString(fmt.Sprintf("      ✓ Success\n"))
					}
				}

				// CRITICAL: Remove the old src/ directory after flattening to avoid import confusion
				if logFile != nil {
					logFile.WriteString(fmt.Sprintf("  Removing old src directory: %s\n", srcDir))
				}

				if err := os.RemoveAll(srcDir); err != nil {
					if logFile != nil {
						logFile.WriteString(fmt.Sprintf("  ✗ Failed to remove src/: %v\n", err))
					}
					// Don't fail the entire operation, just log the warning
				} else {
					if logFile != nil {
						logFile.WriteString(fmt.Sprintf("  ✓ Removed src/ successfully\n"))
					}
				}

				flattened = append(flattened, packageName)
			} else {
				if logFile != nil {
					logFile.WriteString(fmt.Sprintf("  ✗ Failed to read innerPackageDir: %v\n", err))
				}
			}
		} else {
			if logFile != nil {
				logFile.WriteString(fmt.Sprintf("  ✗ Pattern doesn't match (innerPackageDir doesn't exist)\n"))
			}
		}
	}

	if len(flattened) > 0 {
		slog.Info("workspace layout flattening completed",
			"functionID", functionID,
			"flattened", flattened,
			"count", len(flattened))
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

		// Skip unwanted directories and files
		if info.IsDir() {
			// Check if this directory should be skipped
			dirName := filepath.Base(path)
			skipDirs := []string{
				"__pycache__",
				".pytest_cache",
				".mypy_cache",
				".ruff_cache",
				".venv",
				"venv",
				"env",
				".env",
				"node_modules",
				".git",
				".svn",
				".hg",
				".tox",
				"htmlcov",
				".nyc_output",
				"coverage",
				"dist",
				"build",
			}
			for _, skipDir := range skipDirs {
				if dirName == skipDir {
					slog.Debug("skipping directory during copy", "dir", relPath)
					return filepath.SkipDir
				}
			}
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

// copyDirectoryWithFilter recursively copies a directory while applying ContentFilter
func (r *PythonRuntime) copyDirectoryWithFilter(src, dst string, filter *ContentFilter) error {
	// Safety check for nil filter
	if filter == nil {
		return fmt.Errorf("ContentFilter cannot be nil")
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Apply ContentFilter to exclude unwanted files/directories
		if filter.ShouldExclude(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil // Skip this file
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

// checkUvSyncNeeded is disabled - it was causing issues by removing dev/extra dependencies
// Users should run 'uv sync' manually if needed
func checkUvSyncNeeded(projectRoot string) {
	// Disabled - this was stripping dev dependencies from user's venv
	// The build process uses its own isolated venv in .sst/python-cache/build-venv
	return
}
