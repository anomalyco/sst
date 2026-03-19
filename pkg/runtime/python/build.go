package python

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/path"
	"github.com/sst/sst/v3/pkg/runtime"
)

type deployBuilder struct {
	projectResolver *projectResolver
	uvRunner        *uvCommandRunner
	contentFilter   *contentFilter
	config          deployBuilderConfig
}

type deployBuilderConfig struct {
	CacheDir    string
	ProjectRoot string
}

func newDeployBuilder(config deployBuilderConfig) (*deployBuilder, error) {
	if config.CacheDir == "" {
		return nil, fmt.Errorf("cache directory is required")
	}

	if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &deployBuilder{
		projectResolver: newProjectResolver(config.ProjectRoot),
		uvRunner:        newUvCommandRunner(uvCommandRunnerConfig{}),
		contentFilter:   newContentFilterForProject(config.ProjectRoot),
		config:          config,
	}, nil
}

// findWorkspaceRoot walks up from PyprojectPath to find the UV workspace root
// ([tool.uv.workspace]). Falls back to the pyproject dir, or SourceRoot if unset.
//
// Intentionally walks above the SST project root (sst.config.ts directory) because
// UV workspaces can legitimately live above it — e.g. a monorepo where sst.config.ts
// is nested inside a larger Python workspace. The [tool.uv.workspace] declaration
// acts as the natural stop condition for well-configured projects.
func (ib *deployBuilder) findWorkspaceRoot(projectInfo *projectInfo) string {
	if projectInfo.PyprojectPath == "" {
		return projectInfo.SourceRoot
	}

	pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)
	best := pyprojectDir

	currentDir := filepath.Dir(pyprojectDir)
	for currentDir != filepath.Dir(currentDir) && currentDir != "." {
		parentPyproject := filepath.Join(currentDir, "pyproject.toml")
		if _, err := os.Stat(parentPyproject); err == nil {
			best = currentDir
			if config, parseErr := ib.projectResolver.ParsePyprojectToml(parentPyproject); parseErr == nil {
				if len(config.Tool.UV.Workspace.Members) > 0 {
					return currentDir
				}
			}
		}
		currentDir = filepath.Dir(currentDir)
	}

	return best
}

func (ib *deployBuilder) Build(ctx context.Context, input *runtime.BuildInput) (*runtime.BuildOutput, error) {
	startTime := time.Now()

	workingDir := filepath.Dir(input.CfgPath)
	if workingDir == "" {
		workingDir = "."
	}
	ib.projectResolver.projectRoot = workingDir

	projectInfo, err := ib.projectResolver.ResolveHandler(input.Handler)
	if err != nil {
		return nil, fmt.Errorf("project resolution: %w", err)
	}

	localPackages, err := discoverBuildablePackages(projectInfo, ib.projectResolver)
	if err != nil {
		return nil, fmt.Errorf("package discovery: %w", err)
	}

	var packagesBuilt []string
	for _, pkg := range localPackages {
		if err := ib.buildPackage(ctx, input, projectInfo, pkg); err != nil {
			return nil, fmt.Errorf("build %s: %w", pkg.Name, err)
		}
		packagesBuilt = append(packagesBuilt, pkg.Name)
	}

	if err := ib.installDependenciesForBuild(ctx, input, projectInfo); err != nil {
		return nil, fmt.Errorf("dependency installation: %w", err)
	}

	output, err := ib.createFinalBuildOutput(ctx, input, projectInfo)
	if err != nil {
		return nil, fmt.Errorf("build output: %w", err)
	}

	slog.Info("built",
		"functionID", input.FunctionID,
		"duration", time.Since(startTime),
		"packagesBuilt", len(packagesBuilt))

	return output, nil
}

func (ib *deployBuilder) buildPackage(ctx context.Context, input *runtime.BuildInput, projectInfo *projectInfo, pkg *localPackageInfo) error {
	buildType := "sdist"
	if input.Dev {
		buildType = "wheel"
	}

	buildCmd := &uvBuildCommand{
		PackageName: pkg.Name,
		PackageDir:  pkg.Path,
		OutputDir:   input.Out(),
		BuildType:   buildType,
	}

	if err := ib.uvRunner.ExecuteBuildCommand(ctx, buildCmd); err != nil {
		return fmt.Errorf("failed to build package %s: %w", pkg.Name, err)
	}

	if err := ib.extractAndProcessPackageArchive(input.Out(), projectInfo, pkg); err != nil {
		return fmt.Errorf("package post-processing: %w", err)
	}

	return nil
}

func (ib *deployBuilder) createFinalBuildOutput(ctx context.Context, input *runtime.BuildInput, projectInfo *projectInfo) (*runtime.BuildOutput, error) {
	adjustedHandler, err := ib.adjustHandlerPath(input, projectInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to adjust handler path: %w", err)
	}

	if input.IsContainer {
		if err := ib.ensureDockerfile(input, projectInfo); err != nil {
			return nil, fmt.Errorf("failed to ensure Dockerfile: %w", err)
		}
	}

	return &runtime.BuildOutput{
		Out:        input.Out(),
		Handler:    adjustedHandler,
		Errors:     []string{},
		Sourcemaps: []string{},
	}, nil
}

// ensureDockerfile ensures a Dockerfile exists in the output directory for container builds.
func (ib *deployBuilder) ensureDockerfile(input *runtime.BuildInput, projectInfo *projectInfo) error {
	outputDockerfile := filepath.Join(input.Out(), "Dockerfile")

	// Ensure pyproject.toml is in the build context for `pip install .`
	outputPyproject := filepath.Join(input.Out(), "pyproject.toml")
	if _, err := os.Stat(outputPyproject); err != nil && projectInfo.PyprojectPath != "" {
		if _, err := os.Stat(projectInfo.PyprojectPath); err == nil {
			_ = copyFile(projectInfo.PyprojectPath, outputPyproject)
		}
	}

	if _, err := os.Stat(outputDockerfile); err == nil {
		return nil
	}

	projectRoot := projectInfo.ProjectRoot
	if projectRoot == "" {
		projectRoot = path.ResolveRootDir(input.CfgPath)
	}

	customDockerfile := filepath.Join(projectRoot, "Dockerfile")
	if _, err := os.Stat(customDockerfile); err == nil {
		return copyFile(customDockerfile, outputDockerfile)
	}

	if projectInfo.PyprojectPath != "" {
		handlerPkgDir := filepath.Dir(projectInfo.PyprojectPath)
		if handlerPkgDir != projectRoot {
			handlerDockerfile := filepath.Join(handlerPkgDir, "Dockerfile")
			if _, err := os.Stat(handlerDockerfile); err == nil {
				return copyFile(handlerDockerfile, outputDockerfile)
			}
		}
	}

	defaultDockerfile := filepath.Join(path.ResolvePlatformDir(input.CfgPath), "dist", "dockerfiles", "python.Dockerfile")
	if _, err := os.Stat(defaultDockerfile); err != nil {
		return fmt.Errorf("default Python Dockerfile not found at %s: %w", defaultDockerfile, err)
	}

	return copyFile(defaultDockerfile, outputDockerfile)
}

// adjustHandlerPath adjusts the handler path for the artifact structure.
// Strips workspace prefixes for containers and flattens src/ layouts.
func (ib *deployBuilder) adjustHandlerPath(input *runtime.BuildInput, projectInfo *projectInfo) (string, error) {
	handler := strings.TrimPrefix(input.Handler, "./")

	// For container builds, strip the workspace prefix since source files
	// are copied to the root of the build context.
	if input.IsContainer && projectInfo.ProjectRoot != "" && projectInfo.PyprojectPath != "" {
		workspaceDir := filepath.Dir(projectInfo.PyprojectPath)
		if workspaceDir != projectInfo.ProjectRoot {
			relWorkspacePath, err := filepath.Rel(projectInfo.ProjectRoot, workspaceDir)
			if err == nil && relWorkspacePath != "." {
				prefix := filepath.ToSlash(relWorkspacePath) + "/"
				handler = strings.TrimPrefix(handler, prefix)
			}
		}
	}

	lastDot := strings.LastIndex(handler, ".")
	if lastDot == -1 {
		return handler, nil
	}
	filePath := handler[:lastDot]
	funcName := handler[lastDot+1:]

	// Flatten src/ layout (PEP 517): pkg/src/pkg -> pkg
	adjustedPath := flattenSrcLayout(filePath)
	if adjustedPath != filePath {
		adjustedHandler := adjustedPath + "." + funcName
		if input.IsContainer {
			return adjustedHandler, nil
		}
		adjustedFile := filepath.Join(input.Out(), adjustedPath+".py")
		if _, err := os.Stat(adjustedFile); err == nil {
			return adjustedHandler, nil
		}
	}

	return handler, nil
}

func (ib *deployBuilder) extractAndProcessPackageArchive(outputDir string, projectInfo *projectInfo, pkg *localPackageInfo) error {
	// Python normalizes package names: dashes become underscores
	normalizedName := strings.ReplaceAll(pkg.Name, "-", "_")

	// Try wheel files first
	patterns := []string{
		filepath.Join(outputDir, normalizedName+"-*.whl"),
		filepath.Join(outputDir, normalizedName+"-*.tar.gz"),
		filepath.Join(outputDir, pkg.Name+"-*.whl"),
		filepath.Join(outputDir, pkg.Name+"-*.tar.gz"),
	}

	var files []string
	var err error

	for _, pattern := range patterns {
		files, err = filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("failed to find package archive: %w", err)
		}
		if len(files) > 0 {
			break
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no package archive found for %s (tried patterns: %s-*.whl, %s-*.tar.gz, %s-*.whl, %s-*.tar.gz)",
			pkg.Name, normalizedName, normalizedName, pkg.Name, pkg.Name)
	}

	// Process each archive file
	for _, archiveFile := range files {
		if err := ib.processPackageArchive(archiveFile, outputDir, projectInfo); err != nil {
			return fmt.Errorf("failed to process archive %s: %w", archiveFile, err)
		}
	}

	return nil
}

// processPackageArchive extracts and cleans up a single package archive
func (ib *deployBuilder) processPackageArchive(archiveFile, outputDir string, projectInfo *projectInfo) error {
	if strings.HasSuffix(archiveFile, ".whl") {
		if err := extractZip(archiveFile, outputDir); err != nil {
			return fmt.Errorf("failed to extract wheel: %w", err)
		}

		os.Remove(archiveFile)

		// Remove .dist-info directories (not needed for Lambda)
		distInfoPattern := filepath.Join(outputDir, "*.dist-info")
		distInfoDirs, _ := filepath.Glob(distInfoPattern)
		for _, distInfoDir := range distInfoDirs {
			os.RemoveAll(distInfoDir)
		}

		return nil
	}

	// Extract tar.gz
	if err := extractTarGz(archiveFile, outputDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Get the directory name without version number
	archiveBaseName := filepath.Base(archiveFile)
	dirName := strings.TrimSuffix(archiveBaseName, ".tar.gz")
	lastHyphen := strings.LastIndex(dirName, "-")
	if lastHyphen == -1 {
		return fmt.Errorf("invalid archive name format: %s", archiveBaseName)
	}

	baseName := dirName[:lastHyphen]
	extractedDir := filepath.Join(outputDir, dirName)
	targetDir := filepath.Join(outputDir, baseName)

	// Move extracted directory to target
	if err := ib.moveExtractedPackage(extractedDir, targetDir, baseName, projectInfo); err != nil {
		return fmt.Errorf("failed to move extracted package: %w", err)
	}

	// Remove the original archive
	os.Remove(archiveFile)

	return nil
}

// extractZip extracts a zip archive (used for .whl files) to the destination directory.
func extractZip(archiveFile, destDir string) error {
	r, err := zip.OpenReader(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Guard against zip slip
		target := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		out.Close()
		rc.Close()
	}
	return nil
}

// extractTarGz extracts a .tar.gz archive to the destination directory.
func extractTarGz(archiveFile, destDir string) error {
	f, err := os.Open(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		target := filepath.Join(destDir, hdr.Name)
		// Guard against tar slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in tar: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

// moveExtractedPackage moves the extracted package to the correct location
func (ib *deployBuilder) moveExtractedPackage(extractedDir, targetDir, baseName string, projectInfo *projectInfo) error {
	// For src layout, flatten src/{package_name} to {package_name}
	srcPath := filepath.Join(extractedDir, "src", baseName)
	if _, err := os.Stat(srcPath); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove old directory: %w", err)
		}

		// Move src/{package_name} to target
		if err := os.Rename(srcPath, targetDir); err != nil {
			return fmt.Errorf("failed to move src directory contents: %w", err)
		}

		// Clean up extracted directory
		if err := os.RemoveAll(extractedDir); err != nil {
			return fmt.Errorf("failed to clean up extracted directory: %w", err)
		}
	} else {
		// No src directory — check if package needs flattening
		if ib.shouldFlattenPackage(extractedDir) {
			return ib.flattenPackageToRoot(extractedDir, targetDir)
		}

		// Standard case: rename directory
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove old directory: %w", err)
		}

		if err := os.Rename(extractedDir, targetDir); err != nil {
			return fmt.Errorf("failed to rename directory: %w", err)
		}
	}

	return nil
}

// shouldFlattenPackage checks if the directory contains root-level Python files
func (ib *deployBuilder) shouldFlattenPackage(extractedDir string) bool {
	entries, err := os.ReadDir(extractedDir)
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

// flattenPackageToRoot moves Python files from a package directory to the root level
func (ib *deployBuilder) flattenPackageToRoot(extractedDir, outputDir string) error {
	var pythonFiles []string
	err := filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".py" || ext == ".pyi" || info.Name() == "py.typed" {
			pythonFiles = append(pythonFiles, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk extracted directory: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, srcFile := range pythonFiles {
		fileName := filepath.Base(srcFile)
		destFile := filepath.Join(outputDir, fileName)

		if err := copyFile(srcFile, destFile); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", srcFile, destFile, err)
		}
	}

	// Clean up the extracted directory
	os.RemoveAll(extractedDir)

	return nil
}

func (ib *deployBuilder) installDependenciesForBuild(ctx context.Context, input *runtime.BuildInput, projectInfo *projectInfo) error {
	if err := os.MkdirAll(input.Out(), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	requirementsFile := filepath.Join(input.Out(), "requirements.txt")
	if err := ib.generateOrCopyRequirementsFile(ctx, input, projectInfo, requirementsFile); err != nil {
		return fmt.Errorf("failed to generate requirements file: %w", err)
	}

	// Determine architecture for Lambda
	architecture := "x86_64"
	if props, err := ib.parseInputProperties(input); err == nil && props.Architecture != "" {
		architecture = props.Architecture
	}

	// Install dependencies for the target platform (Linux)
	if err := ib.installDependenciesForLambda(ctx, input, projectInfo, requirementsFile, architecture); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	return nil
}

// generateOrCopyRequirementsFile generates requirements.txt once per workspace,
// then copies it to each function's output directory.
func (ib *deployBuilder) generateOrCopyRequirementsFile(ctx context.Context, input *runtime.BuildInput, projectInfo *projectInfo, outputFile string) error {
	// Include dev dependencies for non-buildable source-only projects
	noDev := true
	if projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			if config.Tool.SST.Buildable != nil && !*config.Tool.SST.Buildable {
				noDev = false
			}
		}
		// Legacy: magic comment strings for backwards compatibility
		if noDev {
			if content, err := os.ReadFile(projectInfo.PyprojectPath); err == nil {
				contentStr := string(content)
				if strings.Contains(contentStr, "NOT a buildable package") ||
					strings.Contains(contentStr, "Development environment - not a buildable package") ||
					strings.Contains(contentStr, "SST will treat this as source code") {
					noDev = false
				}
			}
		}
	}

	// Determine if this is a workspace member (has its own pyproject.toml in a subdirectory)
	packageName := ""
	useAllPackages := true
	workspaceRoot := ib.findWorkspaceRoot(projectInfo)

	if projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			if config.Project.Name != "" {
				pyprojectDir := filepath.Dir(projectInfo.PyprojectPath)
				if workspaceRoot != pyprojectDir {
					packageName = config.Project.Name
					useAllPackages = false
				}
			}
		} else {
			slog.Warn("failed to parse pyproject.toml", "path", projectInfo.PyprojectPath, "error", err)
		}
	}

	exportCmd := &uvExportCommand{
		WorkspaceDir:    workspaceRoot,
		PackageName:     packageName,
		OutputFile:      outputFile,
		NoEmitWorkspace: false,
		NoDev:           noDev,
		AllPackages:     useAllPackages,
		NoEmitProject:   !useAllPackages,
		NoEditable:      true,
	}

	// uv export is fast (~300ms, no network/installs) so we run it per function
	// rather than caching. The .deps disk cache handles the expensive uv pip install.
	if err := ib.uvRunner.ExecuteExportCommand(ctx, exportCmd); err != nil {
		return err
	}

	return nil
}

// inputProperties represents the input properties structure
type inputProperties struct {
	Architecture string `json:"architecture"`
}

// parseInputProperties parses the input properties JSON
func (ib *deployBuilder) parseInputProperties(input *runtime.BuildInput) (*inputProperties, error) {
	var props inputProperties
	if err := json.Unmarshal(input.Properties, &props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	return &props, nil
}

func (ib *deployBuilder) installDependenciesForLambda(ctx context.Context, input *runtime.BuildInput, projectInfo *projectInfo, requirementsFile string, architecture string) error {
	if err := ib.copySourceFilesSimple(ctx, input, projectInfo); err != nil {
		return fmt.Errorf("failed to copy source files: %w", err)
	}

	// Container builds: Dockerfile handles deps; zip builds: install here
	if input.IsContainer {
		if err := ib.copyWorkspacePackagesForContainer(input, projectInfo); err != nil {
			return fmt.Errorf("failed to copy workspace packages for container: %w", err)
		}
	} else {
		if err := ib.copySyncedDependencies(ctx, input, projectInfo, architecture); err != nil {
			return fmt.Errorf("failed to copy synced dependencies: %w", err)
		}
	}

	return nil
}

// copyWorkspacePackagesForContainer copies workspace package directories into the artifact
// so the Dockerfile's `uv pip install -r requirements.txt` can resolve relative paths.
func (ib *deployBuilder) copyWorkspacePackagesForContainer(input *runtime.BuildInput, projectInfo *projectInfo) error {
	workspaceRoot := ib.findWorkspaceRoot(projectInfo)

	requirementsPath := filepath.Join(input.Out(), "requirements.txt")
	content, err := os.ReadFile(requirementsPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		if !strings.HasPrefix(line, "./") && !strings.HasPrefix(line, "../") {
			continue
		}

		// Strip extras or markers (e.g., "./core[extra] ; python_version >= '3.11'")
		pkgPath := line
		for _, sep := range []string{" ", "[", ";"} {
			if idx := strings.Index(pkgPath, sep); idx > 0 {
				pkgPath = pkgPath[:idx]
			}
		}

		// Resolve full path relative to workspace root
		fullPath := filepath.Join(workspaceRoot, pkgPath)
		if _, err := os.Stat(fullPath); err != nil {
			slog.Warn("workspace package directory not found", "path", fullPath, "line", line)
			continue
		}

		// Copy to artifact at the same relative path
		destPath := filepath.Join(input.Out(), pkgPath)
		if _, err := os.Stat(destPath); err == nil {
			// Already exists — just ensure pyproject.toml is present for uv pip install
			srcPyproject := filepath.Join(fullPath, "pyproject.toml")
			destPyproject := filepath.Join(destPath, "pyproject.toml")
			if _, err := os.Stat(srcPyproject); err == nil {
				if _, err := os.Stat(destPyproject); err != nil {
					data, readErr := os.ReadFile(srcPyproject)
					if readErr != nil {
						return fmt.Errorf("failed to read pyproject.toml for workspace package %s: %w", pkgPath, readErr)
					}
					if err := os.WriteFile(destPyproject, data, 0644); err != nil {
						return fmt.Errorf("failed to copy pyproject.toml for workspace package %s: %w", pkgPath, err)
					}
				}
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for workspace package %s: %w", pkgPath, err)
		}

		// Unfiltered copy — workspace packages need pyproject.toml and metadata for uv pip install
		if err := copyDirUnfiltered(fullPath, destPath); err != nil {
			return fmt.Errorf("failed to copy workspace package %s: %w", pkgPath, err)
		}

	}

	return nil
}

// copySourceFilesSimple copies handler source files to the build output.
// Workspace packages are installed via uv pip install separately.
func (ib *deployBuilder) copySourceFilesSimple(ctx context.Context, input *runtime.BuildInput, projectInfo *projectInfo) error {
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	handlerPath := input.Handler

	// Strip workspace prefix from handler path
	var outputPrefix string
	if projectInfo.ProjectRoot != "" && workspaceDir != projectInfo.ProjectRoot {
		relWorkspacePath, err := filepath.Rel(projectInfo.ProjectRoot, workspaceDir)
		if err == nil && relWorkspacePath != "." {
			relWorkspacePath = filepath.ToSlash(relWorkspacePath)
			prefix := relWorkspacePath + "/"
			if strings.HasPrefix(handlerPath, prefix) {
				handlerPath = strings.TrimPrefix(handlerPath, prefix)
				outputPrefix = relWorkspacePath
			}
		}
	}

	outputBase := input.Out()
	// For container builds, source files go at root of build context (Dockerfile expects it).
	// For zip builds, preserve the nested directory structure via outputPrefix.
	if outputPrefix != "" && !input.IsContainer {
		outputBase = filepath.Join(input.Out(), outputPrefix)
	}

	if strings.Contains(handlerPath, "/") {
		// Find the top-level directory that exists in the workspace
		parts := strings.Split(handlerPath, "/")
		copied := false
		for i := 0; i < len(parts)-1; i++ {
			candidate := parts[i]
			candidatePath := filepath.Join(workspaceDir, candidate)
			if info, err := os.Stat(candidatePath); err == nil && info.IsDir() {
				if err := copyDir(candidatePath, filepath.Join(outputBase, candidate), ib.contentFilter, ""); err != nil {
					return fmt.Errorf("failed to copy directory %s: %w", candidate, err)
				}
				copied = true
				break
			}
		}
		if !copied {
			// Handler path fully resolved by workspaceDir — root .py files will be copied below
		}
	}

	// Also copy root-level .py files
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return fmt.Errorf("failed to read workspace directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
			if err := copyFile(filepath.Join(workspaceDir, entry.Name()), filepath.Join(outputBase, entry.Name())); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// copySyncedDependencies installs dependencies with correct platform targeting
func (ib *deployBuilder) copySyncedDependencies(ctx context.Context, input *runtime.BuildInput, projectInfo *projectInfo, architecture string) error {
	requirementsPath := filepath.Join(input.Out(), "requirements.txt")

	if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
		slog.Warn("requirements.txt not found, skipping dependency installation", "path", requirementsPath)
		return nil
	}

	workspaceRoot := ib.findWorkspaceRoot(projectInfo)

	// Filter editable installs from requirements
	filteredRequirementsPath := filepath.Join(input.Out(), "requirements-filtered.txt")
	err := ib.filterEditableInstalls(requirementsPath, filteredRequirementsPath)
	if err != nil {
		return fmt.Errorf("failed to filter requirements: %w", err)
	}
	requirementsPath = filteredRequirementsPath

	// Cache key from requirements hash + architecture
	requirementsHash, err := hashFileContents(requirementsPath)
	var cacheKey string
	var depsCacheDir string

	if err == nil {
		cacheKey = fmt.Sprintf("%s-%s", requirementsHash, architecture)
		depsCacheDir = filepath.Join(filepath.Dir(input.Out()), ".deps", cacheKey)

		// Acquire lock to prevent concurrent installs for the same cache key
		globalDependencyInstallLocksMutex.Lock()
		cacheLock, exists := globalDependencyInstallLocks[cacheKey]
		if !exists {
			cacheLock = &sync.Mutex{}
			globalDependencyInstallLocks[cacheKey] = cacheLock
		}
		globalDependencyInstallLocksMutex.Unlock()

		// Acquire lock with timeout (5 minutes)
		lockDeadline := time.Now().Add(5 * time.Minute)
		gotLock := false
		for time.Now().Before(lockDeadline) {
			if cacheLock.TryLock() {
				gotLock = true
				break
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled while waiting for dependency install lock")
			case <-time.After(500 * time.Millisecond):
				// Try again
			}
		}
		if !gotLock {
			return fmt.Errorf("timed out waiting for dependency install lock after 5 minutes")
		}
		defer cacheLock.Unlock()

		// Check disk cache
		if entries, err := os.ReadDir(depsCacheDir); err == nil && len(entries) > 0 {
			if err := ib.copyDependencyPackages(depsCacheDir, input.Out()); err != nil {
				slog.Warn("failed to copy from disk cache, will reinstall", "error", err)
				// Remove bad cache and continue to reinstall
				os.RemoveAll(depsCacheDir)
			} else {
				return nil
			}
		}

		// Cache miss - create the cache directory
		if err := os.MkdirAll(depsCacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create deps cache directory: %w", err)
		}
	} else {
		depsCacheDir = input.Out()
	}

	// Use --reinstall-package for workspace packages to bypass uv's stale cache
	workspacePackages := ib.getWorkspacePackageNames(projectInfo)

	// We use --reinstall-package (not --reinstall) to avoid re-fetching slow git dependencies
	args := []string{"pip", "install", "-r", requirementsPath, "--target", depsCacheDir}

	for _, pkg := range workspacePackages {
		args = append(args, "--reinstall-package", pkg)
	}

	// Platform targeting for Lambda deployments (skip in dev mode and containers)
	// Skip if already on the target platform to use native cached wheels
	if !input.Dev && !input.IsContainer {
		needsCrossPlatform := false
		currentArch := goruntime.GOARCH
		currentOS := goruntime.GOOS

		// Cross-platform needed if not on Linux or architecture mismatch
		targetIsArm := architecture == "arm64"
		currentIsArm := currentArch == "arm64"

		if currentOS != "linux" || targetIsArm != currentIsArm {
			needsCrossPlatform = true
		}

		if needsCrossPlatform {
			// Manylinux tags for GLIBC compatibility:
			// python3.11 and below → AL2 (manylinux2014, GLIBC 2.17)
			// python3.12 and above → AL2023 (manylinux_2_28, GLIBC 2.28)
			pythonVersion := strings.TrimPrefix(input.Runtime, "python")
			if pythonVersion == "" || pythonVersion == input.Runtime {
				pythonVersion = "3.13"
			}

			manylinuxTag := "manylinux_2_28"
			if parts := strings.SplitN(pythonVersion, ".", 2); len(parts) == 2 {
				if minor, err := strconv.Atoi(parts[1]); err == nil && minor <= 11 {
					manylinuxTag = "manylinux2014"
				}
			}

			archPrefix := "x86_64"
			if architecture == "arm64" {
				archPrefix = "aarch64"
			}
			pythonPlatform := archPrefix + "-" + manylinuxTag

			args = append(args, "--python-platform", pythonPlatform, "--python-version", pythonVersion)
		}
	}

	// Run from workspace root (requirements.txt has relative paths like ./vendored_sst)
	installWorkspaceDir := workspaceRoot

	installCtx, installCancel := context.WithTimeout(ctx, 15*time.Minute)
	defer installCancel()

	installCmd := process.CommandContext(installCtx, "uv", args...)
	installCmd.Dir = installWorkspaceDir

	// Use a channel for timeout handling
	type cmdResult struct {
		output []byte
		err    error
	}
	resultChan := make(chan cmdResult, 1)

	// Progress ticker to diagnose hangs
	installStartTime := time.Now()
	progressDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-progressDone:
				return
			case <-ticker.C:
				elapsed := time.Since(installStartTime)
				slog.Warn("uv pip install still running", "elapsed", elapsed)
			}
		}
	}()

	go func() {
		output, err := installCmd.CombinedOutput()
		resultChan <- cmdResult{output, err}
	}()

	var installOutput []byte
	select {
	case result := <-resultChan:
		close(progressDone)
		installOutput = result.output
		err = result.err
	case <-installCtx.Done():
		close(progressDone)
		if installCmd.Process != nil {
			installCmd.Process.Kill()
		}
		// Remove partial cache on timeout
		if cacheKey != "" {
			os.RemoveAll(depsCacheDir)
		}
		return fmt.Errorf("uv pip install timed out after 15 minutes - check network connectivity and try again")
	}

	if err != nil {
		slog.Error("uv pip install failed",
			"command", strings.Join(installCmd.Args, " "),
			"error", err,
			"output", string(installOutput),
			"functionID", input.FunctionID,
			"handler", input.Handler,
			"workingDir", installWorkspaceDir,
			"pyprojectPath", projectInfo.PyprojectPath)
		if cacheKey != "" {
			os.RemoveAll(depsCacheDir)
		}
		return fmt.Errorf("failed to run uv pip install: %v\n%s\n\nFunction: %s\nHandler: %s\nWorking directory: %s\nPyproject path: %s",
			err, string(installOutput), input.FunctionID, input.Handler, installWorkspaceDir, projectInfo.PyprojectPath)
	}

	if err := ib.cleanupInstalledDependencies(depsCacheDir, projectInfo); err != nil {
		slog.Warn("failed to clean up installed dependencies", "error", err)
	}

	if err := ib.copyDependencyPackages(depsCacheDir, input.Out()); err != nil {
		return fmt.Errorf("failed to copy dependencies to artifact: %w", err)
	}

	os.Remove(filteredRequirementsPath)

	return nil
}

// filterEditableInstalls removes editable (-e) local path installs from requirements.txt.
// Editable installs create symlinks which won't work in Lambda.
func (ib *deployBuilder) filterEditableInstalls(inputPath, outputPath string) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read requirements file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var filteredLines []string

	for _, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			filteredLines = append(filteredLines, originalLine)
			continue
		}

		if strings.HasPrefix(line, "-e ") {
			editablePath := strings.TrimSpace(strings.TrimPrefix(line, "-e "))

			if strings.HasPrefix(editablePath, "./") || strings.HasPrefix(editablePath, "../") ||
				strings.HasPrefix(editablePath, "/") && !strings.Contains(editablePath, "://") {
				continue
			}
		}

		// NON-editable local paths and boto3/botocore are handled downstream
		filteredLines = append(filteredLines, originalLine)
	}

	// Write filtered requirements
	filteredContent := strings.Join(filteredLines, "\n")
	if err := os.WriteFile(outputPath, []byte(filteredContent), 0644); err != nil {
		return fmt.Errorf("failed to write filtered requirements file: %w", err)
	}

	return nil
}

// cleanupInstalledDependencies removes __pycache__, .pyc files, .dist-info, test dirs,
// and boto3/botocore (Lambda provides them, saves ~22MB) unless user opts in.
func (ib *deployBuilder) cleanupInstalledDependencies(targetDir string, projectInfo *projectInfo) error {
	includeLambdaRuntime := false
	if projectInfo != nil && projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			includeLambdaRuntime = config.Tool.SST.IncludeLambdaRuntime
		}
	}

	// Remove boto3/botocore unless user opted in
	if !includeLambdaRuntime {
		lambdaRuntimePackages := []string{"boto3", "botocore"}
		for _, pkg := range lambdaRuntimePackages {
			pkgDir := filepath.Join(targetDir, pkg)
			if info, err := os.Stat(pkgDir); err == nil && info.IsDir() {
				if err := os.RemoveAll(pkgDir); err != nil {
					slog.Warn("failed to remove Lambda runtime package", "package", pkg, "error", err)
				}
			}
			// Also remove the dist-info directory
			entries, _ := os.ReadDir(targetDir)
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), pkg+"-") && strings.HasSuffix(entry.Name(), ".dist-info") {
					distInfoDir := filepath.Join(targetDir, entry.Name())
					if err := os.RemoveAll(distInfoDir); err != nil {
						slog.Warn("failed to remove dist-info directory", "dir", entry.Name(), "error", err)
					}
				}
			}
		}
	}

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}

		if path == targetDir {
			return nil
		}

		// Remove __pycache__ directories
		if info.IsDir() && info.Name() == "__pycache__" {
			os.RemoveAll(path)
			return filepath.SkipDir
		}

		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			fileName := info.Name()
			if ext == ".pyc" || ext == ".pyo" || ext == ".pyd" || fileName == ".DS_Store" {
				os.Remove(path)
			}
			if ext == ".pyi" || fileName == "py.typed" {
				os.Remove(path)
			}
		}

		// Remove .dist-info directories
		if info.IsDir() && strings.HasSuffix(info.Name(), ".dist-info") {
			os.RemoveAll(path)
			return filepath.SkipDir
		}

		// Remove test directories
		if info.IsDir() {
			dirName := info.Name()
			if dirName == "SelfTest" || dirName == "tests" || dirName == "test" {
				os.RemoveAll(path)
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking target directory during cleanup: %w", err)
	}

	return nil
}

// getWorkspacePackageNames returns workspace package names from pyproject.toml
func (ib *deployBuilder) getWorkspacePackageNames(projectInfo *projectInfo) []string {
	var packages []string

	// Add the main package name
	if projectInfo.PyprojectPath != "" {
		if config, err := ib.projectResolver.ParsePyprojectToml(projectInfo.PyprojectPath); err == nil {
			if config.Project.Name != "" {
				packages = append(packages, config.Project.Name)
			} else if config.Tool.Poetry.Name != "" {
				packages = append(packages, config.Tool.Poetry.Name)
			}

			// Add workspace packages referenced via { workspace = true }
			for name, source := range config.Tool.UV.Sources {
				if source.Workspace {
					packages = append(packages, name)
				}
			}
		}
	}

	return packages
}

// copyDependencyPackages copies installed dependency packages (not requirements.txt, etc.)
func (ib *deployBuilder) copyDependencyPackages(srcDir, destDir string) error {
	slog.Debug("copying dependency packages", "src", srcDir)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	copiedCount := 0
	copiedFiles := 0
	copiedPthPackages := 0
	for _, entry := range entries {
		name := entry.Name()

		// Skip special directories and non-package files
		if name == "__pycache__" || strings.HasPrefix(name, ".") {
			continue
		}

		// Skip non-package files
		if name == "requirements.txt" || name == "requirements-filtered.txt" || name == "resource.enc" {
			continue
		}

		srcPath := filepath.Join(srcDir, name)
		destPath := filepath.Join(destDir, name)

		if entry.IsDir() {
			if err := copyDir(srcPath, destPath, ib.contentFilter, ""); err != nil {
				slog.Warn("failed to copy package", "package", name, "error", err)
				continue
			}
			copiedCount++
		} else if strings.HasSuffix(name, ".pth") {
			// .pth files are UV path config files pointing to workspace package sources
			pthContent, err := os.ReadFile(srcPath)
			if err != nil {
				slog.Warn("failed to read .pth file", "file", name, "error", err)
				continue
			}

			packageSourcePath := strings.TrimSpace(string(pthContent))
			if packageSourcePath == "" {
				slog.Warn(".pth file is empty", "file", name)
				continue
			}

			// Resolve symlinked .pth files
			if info, err := os.Lstat(srcPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
				realPath, err := filepath.EvalSymlinks(srcPath)
				if err != nil {
					slog.Warn("failed to resolve .pth symlink", "file", name, "error", err)
					continue
				}
				pthContent, err = os.ReadFile(realPath)
				if err != nil {
					slog.Warn("failed to read resolved .pth file", "file", name, "realPath", realPath, "error", err)
					continue
				}
				packageSourcePath = strings.TrimSpace(string(pthContent))
			}

			// Extract package name from .pth filename (e.g., "_sst.pth" -> "sst")
			pthBaseName := strings.TrimSuffix(name, ".pth")
			packageName := strings.TrimPrefix(pthBaseName, "_")

			// Find the actual package directory
			var packageDir string

			candidatePath := filepath.Join(packageSourcePath, packageName)
			if info, err := os.Stat(candidatePath); err == nil && info.IsDir() {
				packageDir = candidatePath
			} else {
				// Try pyproject.toml for hatch build targets
				pyprojectPath := filepath.Join(packageSourcePath, "pyproject.toml")
				if _, err := os.Stat(pyprojectPath); err == nil {
					if config, err := ib.projectResolver.ParsePyprojectToml(pyprojectPath); err == nil {
						// Check hatch build targets
						if len(config.Tool.Hatch.Build.Targets.Wheel.Packages) > 0 {
							pkgName := config.Tool.Hatch.Build.Targets.Wheel.Packages[0]
							candidatePath = filepath.Join(packageSourcePath, pkgName)
							if info, err := os.Stat(candidatePath); err == nil && info.IsDir() {
								packageDir = candidatePath
								packageName = pkgName // Use the actual package name from config
							}
						}
					}
				}
			}

			if packageDir == "" {
				slog.Warn("could not find package directory for .pth file",
					"pthFile", name,
					"packageSourcePath", packageSourcePath,
					"packageName", packageName)
				continue
			}

			// Copy the package
			packageDestPath := filepath.Join(destDir, packageName)
			slog.Debug("copying package from .pth reference",
				"packageName", packageName,
				"source", packageDir)

			if err := copyDir(packageDir, packageDestPath, ib.contentFilter, ""); err != nil {
				slog.Warn("failed to copy package from .pth", "package", packageName, "error", err)
				continue
			}
			copiedPthPackages++
		} else if strings.HasSuffix(name, ".so") || strings.HasSuffix(name, ".py") {
			if err := copyFile(srcPath, destPath); err != nil {
				slog.Warn("failed to copy root file", "file", name, "error", err)
				continue
			}
			copiedFiles++
		}
	}

	slog.Debug("copied dependency packages", "directories", copiedCount, "rootFiles", copiedFiles, "pthPackages", copiedPthPackages)
	return nil
}

// localPackageInfo contains information about a local package
type localPackageInfo struct {
	Name string
	Path string
}

// discoverBuildablePackages finds local packages that need building.
// Checks pyproject.toml workspace members first, then falls back to directory scanning.
func discoverBuildablePackages(projectInfo *projectInfo, resolver *projectResolver) ([]*localPackageInfo, error) {
	workspaceDir := projectInfo.SourceRoot
	if projectInfo.PyprojectPath != "" {
		workspaceDir = filepath.Dir(projectInfo.PyprojectPath)
	}

	// Try workspace members from pyproject.toml first
	pyprojectPath := filepath.Join(workspaceDir, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		if config, err := resolver.ParsePyprojectToml(pyprojectPath); err == nil {
			paths := workspacePackagePaths(config, workspaceDir)
			if len(paths) > 0 {
				return buildableFromPaths(paths, resolver)
			}
		}
	}

	// Fallback: scan common directories for buildable packages
	return buildableFromScan(workspaceDir, resolver)
}

// workspacePackagePaths extracts package paths from a parsed pyproject.toml
func workspacePackagePaths(config *pyprojectConfig, workspaceDir string) []string {
	var paths []string

	// UV workspace members
	for _, member := range config.Tool.UV.Workspace.Members {
		paths = append(paths, filepath.Join(workspaceDir, member))
	}

	// UV sources with local paths
	for _, source := range config.Tool.UV.Sources {
		if source.Path != "" {
			paths = append(paths, filepath.Join(workspaceDir, source.Path))
		}
	}

	// Setuptools packages.find.where
	for _, where := range config.Tool.Setuptools.Packages.Find.Where {
		paths = append(paths, filepath.Join(workspaceDir, where))
	}

	// Hatch build targets
	for _, pkg := range config.Tool.Hatch.Build.Targets.Wheel.Packages {
		paths = append(paths, filepath.Join(workspaceDir, pkg))
	}

	return paths
}

// buildableFromPaths filters a list of paths to only those with build configuration
func buildableFromPaths(paths []string, resolver *projectResolver) ([]*localPackageInfo, error) {
	var packages []*localPackageInfo
	for _, p := range paths {
		if !hasBuildConfig(p) {
			continue
		}
		name := filepath.Base(p)
		pyprojectPath := filepath.Join(p, "pyproject.toml")
		if config, err := resolver.ParsePyprojectToml(pyprojectPath); err == nil && config.Project.Name != "" {
			name = config.Project.Name
		}
		packages = append(packages, &localPackageInfo{Name: name, Path: p})
	}
	return packages, nil
}

// buildableFromScan scans common directories for buildable packages
func buildableFromScan(workspaceDir string, resolver *projectResolver) ([]*localPackageInfo, error) {
	var packages []*localPackageInfo

	// Check workspace root
	if hasBuildConfig(workspaceDir) {
		name := filepath.Base(workspaceDir)
		if config, err := resolver.ParsePyprojectToml(filepath.Join(workspaceDir, "pyproject.toml")); err == nil && config.Project.Name != "" {
			name = config.Project.Name
		}
		packages = append(packages, &localPackageInfo{Name: name, Path: workspaceDir})
	}

	// Check common package locations one level deep
	for _, dir := range []string{"src", "packages", "libs"} {
		searchDir := filepath.Join(workspaceDir, dir)
		entries, err := os.ReadDir(searchDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			candidatePath := filepath.Join(searchDir, entry.Name())
			if !hasBuildConfig(candidatePath) {
				continue
			}
			name := entry.Name()
			if config, err := resolver.ParsePyprojectToml(filepath.Join(candidatePath, "pyproject.toml")); err == nil && config.Project.Name != "" {
				name = config.Project.Name
			}
			packages = append(packages, &localPackageInfo{Name: name, Path: candidatePath})
		}
	}

	// Check immediate subdirectories of workspace root
	entries, err := os.ReadDir(workspaceDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			candidatePath := filepath.Join(workspaceDir, entry.Name())
			if !hasBuildConfig(candidatePath) {
				continue
			}
			// Skip if already found
			found := false
			for _, p := range packages {
				if p.Path == candidatePath {
					found = true
					break
				}
			}
			if found {
				continue
			}
			name := entry.Name()
			if config, err := resolver.ParsePyprojectToml(filepath.Join(candidatePath, "pyproject.toml")); err == nil && config.Project.Name != "" {
				name = config.Project.Name
			}
			packages = append(packages, &localPackageInfo{Name: name, Path: candidatePath})
		}
	}

	return packages, nil
}

// hasBuildConfig checks if a directory has build configuration (pyproject.toml with [build-system], setup.py, etc.)
func hasBuildConfig(dir string) bool {
	// setup.py or setup.cfg → buildable
	for _, f := range []string{"setup.py", "setup.cfg"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}

	// pyproject.toml with build configuration
	pyprojectPath := filepath.Join(dir, "pyproject.toml")
	content, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return false
	}

	contentStr := string(content)

	// Check [tool.sst].buildable first (structured config, preferred)
	var config pyprojectConfig
	if err := toml.Unmarshal(content, &config); err == nil {
		if config.Tool.SST.Buildable != nil {
			return *config.Tool.SST.Buildable
		}
	}

	// Legacy: magic comment strings for backwards compatibility
	if strings.Contains(contentStr, "NOT a buildable package") ||
		strings.Contains(contentStr, "Development environment - not a buildable package") ||
		strings.Contains(contentStr, "SST will treat this as source code") {
		return false
	}

	// Check for build system or tool-specific build config
	buildIndicators := []string{
		"[build-system]",
		"[tool.setuptools]",
		"[tool.poetry]",
		"[tool.hatch]",
		"[tool.flit]",
		"[tool.pdm]",
	}
	for _, indicator := range buildIndicators {
		if strings.Contains(contentStr, indicator) {
			return true
		}
	}

	return false
}

// --- UV command runner (merged from uv_runner.go) ---

// uvCommandRunner executes uv commands
type uvCommandRunner struct {
	commandTimeout time.Duration
}

// uvCommandRunnerConfig configures the UV command runner
type uvCommandRunnerConfig struct {
	CommandTimeout time.Duration
}

// commandResult represents the result of a UV command execution
type commandResult struct {
	ExitCode int
	Stderr   string
	Success  bool
}

// uvBuildCommand represents a UV build command
type uvBuildCommand struct {
	PackageName string
	PackageDir  string
	OutputDir   string
	BuildType   string
}

// uvExportCommand represents a UV export command
type uvExportCommand struct {
	WorkspaceDir    string
	PackageName     string
	OutputFile      string
	NoEmitWorkspace bool
	NoDev           bool
	NoEditable      bool
	NoEmitProject   bool
	AllPackages     bool
}

// newUvCommandRunner creates a new UV command runner
func newUvCommandRunner(config uvCommandRunnerConfig) *uvCommandRunner {
	timeout := config.CommandTimeout
	if timeout == 0 {
		timeout = 1 * time.Minute
	}
	return &uvCommandRunner{commandTimeout: timeout}
}

// ExecuteBuildCommand executes a UV build command for a single package
func (ur *uvCommandRunner) ExecuteBuildCommand(ctx context.Context, cmd *uvBuildCommand) error {
	if ur == nil {
		return fmt.Errorf("UV command runner is nil")
	}
	if cmd == nil {
		return fmt.Errorf("UV build command is nil")
	}

	args := []string{"build"}

	if cmd.PackageName != "" {
		args = append(args, "--package="+cmd.PackageName)
	}

	if cmd.BuildType == "wheel" {
		args = append(args, "--wheel")
	} else {
		args = append(args, "--sdist")
	}

	if cmd.OutputDir != "" {
		args = append(args, "--out-dir="+cmd.OutputDir)
	}

	args = append(args, "--no-sources")

	workingDir := cmd.PackageDir
	if workingDir == "" {
		workingDir = "."
	}

	result, err := ur.executeCommand(ctx, "uv", args, workingDir)
	if err != nil {
		slog.Error("UV build command failed",
			"package", cmd.PackageName,
			"command", "uv "+strings.Join(args, " "),
			"error", err)
		return fmt.Errorf("uv build failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("uv build failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}

	return nil
}

// ExecuteExportCommand executes a UV export command
func (ur *uvCommandRunner) ExecuteExportCommand(ctx context.Context, cmd *uvExportCommand) error {
	args := []string{"export"}

	if cmd.AllPackages {
		args = append(args, "--all-packages")
	} else if cmd.PackageName != "" {
		args = append(args, "--package="+cmd.PackageName)
	}

	if cmd.OutputFile != "" {
		args = append(args, "--output-file="+cmd.OutputFile)
	}
	if cmd.NoEmitWorkspace {
		args = append(args, "--no-emit-workspace")
	}
	if cmd.NoEditable {
		args = append(args, "--no-editable")
	}
	if cmd.NoEmitProject {
		args = append(args, "--no-emit-project")
	}
	if cmd.NoDev {
		args = append(args, "--no-dev")
	}

	result, err := ur.executeCommand(ctx, "uv", args, cmd.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("UV export failed: %w", err)
	}

	if !result.Success {
		return ur.createDetailedExportError(result, cmd)
	}

	return nil
}

// createDetailedExportError creates a detailed error message for export failures
func (ur *uvCommandRunner) createDetailedExportError(result *commandResult, cmd *uvExportCommand) error {
	errorMsg := fmt.Sprintf("UV export failed with exit code %d", result.ExitCode)

	if cmd.PackageName != "" {
		errorMsg += fmt.Sprintf(" (exporting package: %s)", cmd.PackageName)
	}
	if cmd.OutputFile != "" {
		errorMsg += fmt.Sprintf(" to file: %s", cmd.OutputFile)
	}
	errorMsg += fmt.Sprintf(" in workspace: %s", cmd.WorkspaceDir)

	if result.Stderr != "" {
		errorMsg += fmt.Sprintf("\nError output: %s", result.Stderr)
	}

	if strings.Contains(result.Stderr, "package") && strings.Contains(result.Stderr, "not found") {
		errorMsg += "\nSuggestion: Check if the package name is correct and exists in the workspace"
	} else if strings.Contains(result.Stderr, "lock") {
		errorMsg += "\nSuggestion: Run 'uv sync' first to ensure dependencies are resolved"
	}

	return fmt.Errorf("%s", errorMsg)
}

// executeCommand executes a command with timeout and progress logging
func (ur *uvCommandRunner) executeCommand(ctx context.Context, command string, args []string, workingDir string) (*commandResult, error) {
	if ur == nil {
		return nil, fmt.Errorf("UV command runner is nil")
	}

	startTime := time.Now()

	cmdCtx, cancel := context.WithTimeout(ctx, ur.commandTimeout)
	defer cancel()

	cmd := process.CommandContext(cmdCtx, command, args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	cmd.Env = os.Environ()

	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				slog.Warn("UV command still running",
					"command", command,
					"elapsed", time.Since(startTime),
					"workingDir", workingDir)
			}
		}
	}()

	output, err := cmd.CombinedOutput()
	close(done)

	result := &commandResult{}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Success = false
		result.Stderr = string(output)
		slog.Error("command error output", "stderr", result.Stderr)
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	return result, nil
}

// --- Content filter (merged from content_filter.go) ---

// contentFilter filters out unnecessary files and directories from deployment artifacts
type contentFilter struct {
	excludePatterns []string
	projectRoot     string
	pyprojectCache  *pyprojectConfig
}

// newContentFilter creates a new content filter with default exclude patterns
func newContentFilter() *contentFilter {
	return &contentFilter{
		excludePatterns: getDefaultExcludePatterns(),
	}
}

// newContentFilterForProject creates a content filter for a specific project
func newContentFilterForProject(projectRoot string) *contentFilter {
	return &contentFilter{
		excludePatterns: getDefaultExcludePatterns(),
		projectRoot:     projectRoot,
	}
}

// getDefaultExcludePatterns returns patterns to exclude from deployment artifacts.
// Directory names (e.g. ".git") match all files underneath them.
func getDefaultExcludePatterns() []string {
	return []string{
		".sst", ".git", ".gitignore", ".gitattributes",

		"__pycache__", "*.pyc", "*.pyo", "*.pyd",
		".pytest_cache", "*.egg-info", ".coverage", "htmlcov",

		".venv", "venv", ".env", "env",

		".vscode", ".idea", "*.swp", "*.swo", "*~", ".DS_Store",

		"node_modules", "package-lock.json", "yarn.lock", "bun.lockb",

		"README.md", "README.rst", "README.txt",
		"CHANGELOG.md", "CHANGELOG.rst", "CHANGELOG.txt",
		"LICENSE", "LICENSE.txt", "MANIFEST.in",
		"setup.cfg", "tox.ini", "Makefile",
		"Dockerfile", "docker-compose.yml", "docker-compose.yaml",

		"tests", "test",

		"pyproject.toml", "setup.py",
		"requirements-dev.txt", "requirements.dev.txt", "dev-requirements.txt",
		".python-version", ".pre-commit-config.yaml",

		"*.log", "*.tmp", "tmp", "temp",
	}
}

// ShouldExclude checks if a file or directory should be excluded:
// 1. Check pyproject.toml [tool.sst] overrides first
// 2. Apply standard Python conventions
func (cf *contentFilter) ShouldExclude(path string) bool {
	normalizedPath := filepath.ToSlash(path)

	if shouldSkip, found := cf.checkPyprojectConfig(normalizedPath); found {
		return shouldSkip
	}

	for _, pattern := range cf.excludePatterns {
		if cf.matchesPattern(normalizedPath, pattern) {
			return true
		}
	}
	return false
}

// matchesPattern checks if a path matches a pattern.
// Supports wildcards (*.pyc), directory names (.git matches .git/anything),
// and ** glob patterns.
func (cf *contentFilter) matchesPattern(path, pattern string) bool {
	if dir, ok := strings.CutSuffix(pattern, "/"); ok {
		return strings.HasPrefix(path, dir+"/") || path == dir
	}

	if strings.Contains(pattern, "**") {
		pattern = strings.ReplaceAll(pattern, "**/", "")
		pattern = strings.ReplaceAll(pattern, "/**", "")
		pattern = strings.ReplaceAll(pattern, "**", "")
		if pattern == "" {
			return true
		}
	}

	for _, part := range strings.Split(path, "/") {
		if part == pattern {
			return true
		}
		if matched, err := filepath.Match(pattern, part); err == nil && matched {
			return true
		}
	}

	if matched, err := filepath.Match(pattern, path); err == nil && matched {
		return true
	}

	return false
}

// checkPyprojectConfig checks [tool.sst] include/exclude configuration.
func (cf *contentFilter) checkPyprojectConfig(path string) (bool, bool) {
	if cf.projectRoot == "" {
		return false, false
	}

	if cf.pyprojectCache == nil {
		cf.loadPyprojectConfig()
	}
	if cf.pyprojectCache == nil {
		return false, false
	}

	for _, pattern := range cf.pyprojectCache.Tool.SST.Include {
		if cf.matchesPattern(path, pattern) {
			return false, true
		}
	}
	for _, pattern := range cf.pyprojectCache.Tool.SST.Exclude {
		if cf.matchesPattern(path, pattern) {
			return true, true
		}
	}
	return false, false
}

// loadPyprojectConfig loads and parses the pyproject.toml file
func (cf *contentFilter) loadPyprojectConfig() {
	pyprojectPath := filepath.Join(cf.projectRoot, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); os.IsNotExist(err) {
		return
	}

	var config pyprojectConfig
	if _, err := toml.DecodeFile(pyprojectPath, &config); err != nil {
		slog.Warn("failed to parse pyproject.toml", "path", pyprojectPath, "error", err)
		return
	}

	if len(config.Tool.SST.Include) > 0 || len(config.Tool.SST.Exclude) > 0 {
		cf.pyprojectCache = &config
	}
}
