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

var (
	// Limits concurrent builds to prevent system overload when Pulumi builds 100+ functions in parallel.
	// Respects SST_BUILD_CONCURRENCY_FUNCTION env var, defaults to 4.
	globalBuildSemaphore = make(chan struct{}, parseConcurrency())

	// Per-cache-key locks — only one function installs per cache key at a time
	globalDependencyInstallLocks      = make(map[string]*sync.Mutex)
	globalDependencyInstallLocksMutex sync.Mutex

	// Generate requirements.txt once per workspace, reuse for all functions
	globalRequirementsFiles      = make(map[string]string)
	globalRequirementsFilesMutex sync.Mutex

	// Clear .deps/ once per SST run so workspace package changes are picked up
	globalDepsCacheClearOnce sync.Once
)

// parseConcurrency reads SST_BUILD_CONCURRENCY_FUNCTION and returns the desired
// parallelism, defaulting to 4.
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

	cacheDir      string
	deployBuilder *DeployBuilder

	// Mutex for thread-safe access
	mutex sync.RWMutex
}

func New() *PythonRuntime {
	return &PythonRuntime{
		lastBuiltHandler: map[string]string{},
	}
}

func (r *PythonRuntime) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	// Clear deps cache once per SST run
	globalDepsCacheClearOnce.Do(func() {
		artifactsDir := filepath.Dir(input.Out())
		depsDir := filepath.Join(artifactsDir, ".deps")
		if _, err := os.Stat(depsDir); err == nil {
			if err := os.RemoveAll(depsDir); err != nil {
				slog.Warn("failed to clear deps cache", "error", err)
			}
		}
	})

	// Acquire build semaphore (Pulumi calls Build() for all functions in parallel)
	globalBuildSemaphore <- struct{}{}
	defer func() {
		<-globalBuildSemaphore
	}()

	// Fast path for dev mode: skip rebuild if artifact is still valid
	if input.Dev {
		r.mutex.RLock()
		lastBuilt, hasBuilt := r.lastBuiltHandler[input.FunctionID]
		r.mutex.RUnlock()

		if hasBuilt && lastBuilt != "" {
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

	/// UV currently does not support --include-workspace-deps for builds
	/// See: https://github.com/astral-sh/uv/issues/6935
	///
	/// So we manually:
	/// 1. Build all packages
	/// 2. Flatten src/ nesting for lambdaric module resolution
	/// 3. Export uv package index to requirements.txt
	/// 4. Install deps into artifact dir (local for zip, Dockerfile for containers)

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

// ShouldRunEagerly returns false to enable lazy worker startup.
// Python lacks static import analysis, so any file change triggers ShouldRebuild()
// for ALL functions. Lazy startup avoids a startup storm of 50+ processes.
func (r *PythonRuntime) ShouldRunEagerly() bool {
	return false
}

func (r *PythonRuntime) Run(ctx context.Context, input *runtime.RunInput) (runtime.Worker, error) {
	if err := r.syncArtifactsIfNeeded(input); err != nil {
		slog.Error("failed to sync artifacts",
			"functionID", input.FunctionID,
			"error", err)
		return nil, fmt.Errorf("failed to sync artifacts: %v", err)
	}

	// Copy lambda bridge to artifact directory if missing or outdated
	lambdaBridgePath := filepath.Join(input.Build.Out, "lambdaric_python_bridge.py")
	sourceBridgePath := filepath.Join(path.ResolvePlatformDir(input.CfgPath), "/dist/python-runtime/index.py")

	shouldCopy := false
	if _, err := os.Stat(lambdaBridgePath); os.IsNotExist(err) {
		shouldCopy = true
	} else {
		// Check if source is newer
		if srcInfo, err := os.Stat(sourceBridgePath); err == nil {
			if dstInfo, err := os.Stat(lambdaBridgePath); err == nil {
				if srcInfo.ModTime().After(dstInfo.ModTime()) {
					shouldCopy = true
				}
			}
		}
	}

	if shouldCopy {
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
		// Use relative handler since workingDir is the artifact directory
		handlerPath = r.adjustHandlerForFlattenedLayout(input.Build.Handler)
		workingDir = input.Build.Out
	} else {
		// Modern layout: run from source with PYTHONPATH
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

	if !isLegacyLayout {
		pythonPaths := []string{projectRoot}

		// Add src/ if it exists
		srcDir := filepath.Join(projectRoot, "src")
		if _, err := os.Stat(srcDir); err == nil {
			pythonPaths = append(pythonPaths, srcDir)
		}

		// Join paths
		pythonPath := strings.Join(pythonPaths, string(os.PathListSeparator))
		env = append(env, "PYTHONPATH="+pythonPath)

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
	// Rebuild on any relevant Python file change. We don't do per-function dependency
	// tracking because Python imports are dynamic and Build() is fast (~50µs in dev).
	// Combined with lazy startup (ShouldRunEagerly=false), this is efficient.

	if r.shouldIgnoreFile(file) {
		return false
	}

	if !r.isRelevantFile(file) {
		return false
	}

	return true
}

func (r *PythonRuntime) syncArtifactsIfNeeded(input *runtime.RunInput) error {
	projectRoot := path.ResolveRootDir(input.CfgPath)
	artifactDir := input.Build.Out

	// Only sync for legacy workspace layouts that need flattening
	if r.hasWorkspaceLayoutPatterns(projectRoot) {
		if err := r.syncPythonFiles(input.FunctionID, projectRoot, artifactDir); err != nil {
			return err
		}

		return r.flattenWorkspaceLayouts(artifactDir, input.FunctionID)
	}

	return nil
}

// syncPythonFiles syncs Python files from source to artifacts (add, update, delete)
func (r *PythonRuntime) syncPythonFiles(functionID, srcDir, destDir string) error {
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

		if strings.HasSuffix(path, ".py") {
			sourceFiles[relPath] = info
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan source files: %v", err)
	}

	// Collect Python files in artifacts
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

	// Delete files in artifacts that no longer exist in source
	for relPath := range artifactFiles {
		if _, exists := sourceFiles[relPath]; !exists {
			artifactPath := filepath.Join(destDir, relPath)
			if err := os.Remove(artifactPath); err != nil {
				return fmt.Errorf("failed to delete %s: %v", artifactPath, err)
			}
		}
	}

	// Copy/update changed files
	for relPath, sourceInfo := range sourceFiles {
		sourcePath := filepath.Join(srcDir, relPath)
		artifactPath := filepath.Join(destDir, relPath)

		needsCopy := true
		if artifactInfo, exists := artifactFiles[relPath]; exists {
			if !sourceInfo.ModTime().After(artifactInfo.ModTime()) {
				needsCopy = false
			}
		}

		if needsCopy {
			if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %v", artifactPath, err)
			}

			// Copy the file
			if err := copyFile(sourcePath, artifactPath); err != nil {
			}
		}
	}

	return nil
}

// flattenSrcLayout removes the "src/pkg" segment from paths that follow the
// PEP 517 src-layout convention (e.g., "pkg/src/pkg/module" -> "pkg/module").
// Only flattens when the directory after "src" matches the directory before it.
func flattenSrcLayout(filePath string) string {
	parts := strings.Split(filePath, "/")
	if len(parts) < 3 {
		return filePath
	}
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "src" && i > 0 && i+1 < len(parts) && parts[i-1] == parts[i+1] {
			flattened := append([]string{}, parts[:i]...)
			flattened = append(flattened, parts[i+2:]...)
			return strings.Join(flattened, "/")
		}
	}
	return filePath
}

// adjustHandlerForFlattenedLayout adjusts a handler path to account for flattened workspace layouts
// For example: functions/src/functions/user/handler.lambda_handler -> functions/user/handler.lambda_handler
func (r *PythonRuntime) adjustHandlerForFlattenedLayout(handlerPath string) string {
	lastDot := strings.LastIndex(handlerPath, ".")
	if lastDot == -1 {
		return flattenSrcLayout(handlerPath)
	}
	filePath := handlerPath[:lastDot]
	functionName := handlerPath[lastDot+1:]
	return flattenSrcLayout(filePath) + "." + functionName
}

// isRelevantFile checks if a file change is relevant for Python functions
func (r *PythonRuntime) isRelevantFile(file string) bool {
	if r.shouldIgnoreFile(file) {
		return false
	}

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

// shouldIgnoreFile determines if a file should be ignored to prevent rebuild loops
func (r *PythonRuntime) shouldIgnoreFile(file string) bool {
	normalizedFile := filepath.ToSlash(file)

	// Build artifacts and cache directories
	ignorePaths := []string{
		".sst",
		"__pycache__",
		".pytest_cache",
		".mypy_cache",
		".coverage",
		"build",
		"dist",
		".git",
		"node_modules",
		".venv",
		"venv",
		"env",
		".tox",
		".eggs",
		".egg-info",
	}

	ignoreExtensions := []string{
		".pyc", ".pyo", ".pyd",
		".log",
		".tmp", ".temp",
		".swp", ".swo",
		".DS_Store",
		".coverage",
	}

	// Check path components
	pathParts := strings.Split(normalizedFile, "/")
	for _, part := range pathParts {
		for _, ignorePath := range ignorePaths {
			if part == ignorePath || strings.HasPrefix(part, ignorePath) {
				return true
			}
		}
	}

	// Check extensions
	for _, ext := range ignoreExtensions {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}

	// Ignore hidden directories (except specific dotfiles)
	parts := strings.Split(file, string(filepath.Separator))
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
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

	result, err := r.createBuildAssetDeploy(ctx, input, arch, workingDir)
	if err != nil {
		return nil, fmt.Errorf("deploy build failed: %w", err)
	}
	return result, nil
}

// createBuildAssetDeploy uses the shared DeployBuilder for all function builds
func (r *PythonRuntime) createBuildAssetDeploy(ctx context.Context, input *runtime.BuildInput, arch string, workingDir string) (*runtime.BuildOutput, error) {
	startTime := time.Now()

	// Reuse DeployBuilder across all function builds to avoid overhead
	// from creating one per 50+ functions.
	r.mutex.Lock()
	if r.deployBuilder == nil {
		cacheDir := r.cacheDir
		if cacheDir == "" {
			if input.Dev {
				cacheDir = filepath.Join(workingDir, ".sst/cache/dev")
			} else {
				cacheDir = filepath.Join(workingDir, ".sst/cache/deploy")
			}
		}

		var err error
		r.deployBuilder, err = NewDeployBuilder(DeployBuilderConfig{
			CacheDir:    cacheDir,
			ProjectRoot: workingDir,
		})
		if err != nil {
			r.mutex.Unlock()
			return nil, fmt.Errorf("failed to create deploy builder: %w", err)
		}
	}
	builder := r.deployBuilder
	r.mutex.Unlock()

	result, err := builder.Build(ctx, input)

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

	// No Python file found — list what exists for debugging
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

// copyDir recursively copies a directory, applying ContentFilter if provided.
// filterPrefix is prepended to relative paths for filter matching.
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

		// Apply content filter
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

// copyDirUnfiltered copies a directory recursively, skipping only __pycache__,
// .venv, node_modules, and .git. Used for container builds where metadata must be preserved.
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

// createSimpleDevBuild creates a minimal build for dev mode — all work deferred to Run()
func (r *PythonRuntime) createSimpleDevBuild(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	if err := os.MkdirAll(input.Out(), 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %v", err)
	}

	// In dev mode, all copying/flattening happens just-in-time in Run()
	return &runtime.BuildOutput{
		Handler:    input.Handler,
		Sourcemaps: []string{},
		Errors:     []string{},
		Out:        input.Out(),
	}, nil
}

// hasWorkspaceLayoutPatterns checks for package/src/package patterns that need flattening
func (r *PythonRuntime) hasWorkspaceLayoutPatterns(projectRoot string) bool {
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

		// Check for package/src/package pattern
		srcDir := filepath.Join(path, "src")
		if _, err := os.Stat(srcDir); err == nil {
			// Check if there's a subdirectory in src with the same name as the parent
			packageName := dirName
			innerPackageDir := filepath.Join(srcDir, packageName)
			if _, err := os.Stat(innerPackageDir); err == nil {
				hasPatterns = true
				return filepath.SkipDir
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
