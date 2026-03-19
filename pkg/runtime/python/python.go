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
		slog.Warn("invalid SST_BUILD_CONCURRENCY_FUNCTION, using default",
			"value", raw, "default", 4)
		return 4
	}
	if n < 1 {
		slog.Warn("SST_BUILD_CONCURRENCY_FUNCTION must be >= 1, using default",
			"value", n, "default", 4)
		return 4
	}
	return n
}

type worker struct {
	stdout io.ReadCloser
	stderr io.ReadCloser
	cmd    *exec.Cmd
}

func (w *worker) Stop() {
	// Terminate the whole process group
	process.Kill(w.cmd.Process)
}

func (w *worker) Logs() io.ReadCloser {
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
	deployBuilder *deployBuilder

	// Mutex for thread-safe access
	mutex sync.RWMutex
}

func New() *PythonRuntime {
	return &PythonRuntime{}
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

	_, err := r.getFile(input)
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
	isLegacyLayout, err := r.syncArtifactsIfNeeded(input)
	if err != nil {
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

	return &worker{
		stdout,
		stderr,
		cmd,
	}, nil

}

func (r *PythonRuntime) ShouldRebuild(functionID string, file string) bool {
	// Always rebuild — Python imports are dynamic so we can't track per-function deps.
	// This is negligible for now and will get faster when we can move to uv's native build system.
	return true
}

func (r *PythonRuntime) syncArtifactsIfNeeded(input *runtime.RunInput) (bool, error) {
	projectRoot := path.ResolveRootDir(input.CfgPath)
	artifactDir := input.Build.Out

	// Only sync for legacy workspace layouts that need flattening
	if r.hasWorkspaceLayoutPatterns(projectRoot) {
		if err := r.syncPythonFiles(projectRoot, artifactDir); err != nil {
			return true, err
		}

		return true, r.flattenWorkspaceLayouts(artifactDir)
	}

	return false, nil
}

// syncPythonFiles syncs Python files from source to artifacts (add, update, delete)
func (r *PythonRuntime) syncPythonFiles(srcDir, destDir string) error {
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

			if err := copyFile(sourcePath, artifactPath); err != nil {
				return fmt.Errorf("failed to copy %s to %s: %w", sourcePath, artifactPath, err)
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
		arch = "x86_64"
	}

	if arch != "x86_64" && arch != "arm64" {
		return nil, fmt.Errorf("invalid architecture %q - must be x86_64 or arm64 - %v", arch, string(input.Properties))
	}
	workingDir := path.ResolveRootDir(input.CfgPath)

	result, err := r.createBuildAssetDeploy(ctx, input, workingDir)
	if err != nil {
		return nil, fmt.Errorf("deploy build failed: %w", err)
	}
	return result, nil
}

// createBuildAssetDeploy uses the shared deployBuilder for all function builds
func (r *PythonRuntime) createBuildAssetDeploy(ctx context.Context, input *runtime.BuildInput, workingDir string) (*runtime.BuildOutput, error) {
	startTime := time.Now()

	// Reuse deployBuilder across all function builds to avoid overhead
	// from creating one per 50+ functions.
	r.mutex.Lock()
	if r.deployBuilder == nil {
		var cacheDir string
		if input.Dev {
			cacheDir = filepath.Join(workingDir, ".sst/cache/dev")
		} else {
			cacheDir = filepath.Join(workingDir, ".sst/cache/deploy")
		}

		var err error
		r.deployBuilder, err = newDeployBuilder(deployBuilderConfig{
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
func copyDir(src, dst string, filter *contentFilter, filterPrefix string) error {
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
func (r *PythonRuntime) flattenWorkspaceLayouts(artifactDir string) error {
	contentFilter := newContentFilter()

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
