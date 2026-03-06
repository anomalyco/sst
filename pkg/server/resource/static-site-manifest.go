package resource

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	iofs "io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type StaticSiteManifest struct {
	run *Run
}

type StaticSiteManifestFile struct {
	Source       string  `json:"source"`
	Key          string  `json:"key"`
	Hash         string  `json:"hash"`
	ContentType  string  `json:"contentType"`
	CacheControl *string `json:"cacheControl,omitempty"`
}

type StaticSiteManifestFileOption struct {
	Files        []string `json:"files"`
	Ignore       []string `json:"ignore,omitempty"`
	CacheControl *string  `json:"cacheControl,omitempty"`
	ContentType  *string  `json:"contentType,omitempty"`
}

type StaticSiteManifestInputs struct {
	SitePath     string                         `json:"sitePath"`
	OutputPath   string                         `json:"outputPath"`
	BuildCommand *string                        `json:"buildCommand,omitempty"`
	Environment  map[string]string              `json:"environment,omitempty"`
	FileOptions  []StaticSiteManifestFileOption `json:"fileOptions"`
	TextEncoding string                         `json:"textEncoding"`
	KeyPrefix    *string                        `json:"keyPrefix,omitempty"`
	AssetPath    *string                        `json:"assetPath,omitempty"`
	AssetRoutes  []string                       `json:"assetRoutes,omitempty"`
	BucketDomain *string                        `json:"bucketDomain,omitempty"`
	ErrorPage    *string                        `json:"errorPage,omitempty"`
	Base         *string                        `json:"base,omitempty"`
	Trigger      string                         `json:"trigger"`
}

type StaticSiteManifestOutputs struct {
	Files               []StaticSiteManifestFile `json:"files,omitempty"`
	AssetManifest       map[string]string        `json:"assetManifest,omitempty"`
	KvEntries           map[string]string        `json:"kvEntries,omitempty"`
	InvalidationVersion string                   `json:"invalidationVersion,omitempty"`
	OutputPath          string                   `json:"outputPath,omitempty"`
}

func (r *StaticSiteManifest) Create(input *StaticSiteManifestInputs, output *CreateResult[StaticSiteManifestOutputs]) error {
	outs, err := r.resolve(input)
	if err != nil {
		return err
	}
	*output = CreateResult[StaticSiteManifestOutputs]{
		ID:   "manifest",
		Outs: *outs,
	}
	return nil
}

func (r *StaticSiteManifest) Update(input *UpdateInput[StaticSiteManifestInputs, StaticSiteManifestOutputs], output *UpdateResult[StaticSiteManifestOutputs]) error {
	outs, err := r.resolve(&input.News)
	if err != nil {
		return err
	}
	*output = UpdateResult[StaticSiteManifestOutputs]{
		Outs: *outs,
	}
	return nil
}

func (r *StaticSiteManifest) resolve(input *StaticSiteManifestInputs) (*StaticSiteManifestOutputs, error) {
	if input.BuildCommand != nil && strings.TrimSpace(*input.BuildCommand) != "" {
		err := r.run.executeCommand(&RunInputs{
			Command: *input.BuildCommand,
			Cwd:     input.SitePath,
			Env:     input.Environment,
			Version: input.Trigger,
		})
		if err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(input.OutputPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("No build output found at %q.", filepath.Clean(input.OutputPath))
		}
		return nil, err
	}

	files, err := buildStaticSiteFiles(input)
	if err != nil {
		return nil, err
	}

	return &StaticSiteManifestOutputs{
		Files:               files,
		AssetManifest:       buildStaticSiteAssetManifest(input, files),
		KvEntries:           buildStaticSiteKvEntries(input, files),
		InvalidationVersion: buildStaticSiteInvalidationVersion(input, files),
		OutputPath:          input.OutputPath,
	}, nil
}

func buildStaticSiteFiles(input *StaticSiteManifestInputs) ([]StaticSiteManifestFile, error) {
	allFiles, err := collectStaticSiteFiles(input.OutputPath)
	if err != nil {
		return nil, err
	}

	processed := map[string]bool{}
	result := []StaticSiteManifestFile{}
	textEncoding := input.TextEncoding
	if textEncoding == "" {
		textEncoding = "utf-8"
	}

	for i := len(input.FileOptions) - 1; i >= 0; i-- {
		option := input.FileOptions[i]
		for _, file := range allFiles {
			if processed[file] || !matchesStaticSitePatterns(file, option.Files) || matchesStaticSitePatterns(file, option.Ignore) {
				continue
			}

			source := filepath.Join(input.OutputPath, filepath.FromSlash(file))
			content, err := os.ReadFile(source)
			if err != nil {
				return nil, err
			}
			hash := sha256.Sum256(content)
			contentType := valueOrDefault(option.ContentType, staticSiteContentType(file, textEncoding))
			result = append(result, StaticSiteManifestFile{
				Source:       source,
				Key:          staticSiteBuildKey(input.KeyPrefix, file),
				Hash:         hex.EncodeToString(hash[:]),
				ContentType:  contentType,
				CacheControl: option.CacheControl,
			})
			processed[file] = true
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})
	return result, nil
}

func collectStaticSiteFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(current string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if current == root {
			return nil
		}

		rel, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		rel = toStaticSitePosix(rel)
		if d.IsDir() {
			if rel == ".sst" || strings.HasPrefix(rel, ".sst/") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(rel, ".sst/") {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

func buildStaticSiteAssetManifest(input *StaticSiteManifestInputs, files []StaticSiteManifestFile) map[string]string {
	result := map[string]string{}
	for _, file := range files {
		result[staticSiteRelativeFile(input.OutputPath, file.Source)] = file.Hash
	}
	return result
}

func buildStaticSiteKvEntries(input *StaticSiteManifestInputs, files []StaticSiteManifestFile) map[string]string {
	if input.BucketDomain == nil || *input.BucketDomain == "" {
		return map[string]string{}
	}

	entries := map[string]string{}
	directories := map[string]bool{}
	for _, file := range files {
		relative := staticSiteRelativeFile(input.OutputPath, file.Source)
		entries[path.Join("/", relative)] = "s3"

		firstSegment, _, ok := strings.Cut(relative, "/")
		if !ok || firstSegment == "" || firstSegment == ".well-known" {
			continue
		}
		directories["/"+firstSegment] = true
	}

	routes := append([]string{}, input.AssetRoutes...)
	for directory := range directories {
		routes = append(routes, directory)
	}
	slices.Sort(routes[len(input.AssetRoutes):])

	metadata, _ := json.Marshal(map[string]any{
		"base":      normalizeStaticSiteBase(input.Base),
		"custom404": derefStaticSiteString(input.ErrorPage),
		"s3": map[string]any{
			"domain": derefStaticSiteString(input.BucketDomain),
			"dir":    staticSiteDirPrefix(input.AssetPath),
			"routes": routes,
		},
	})
	entries["metadata"] = string(metadata)
	return entries
}

func buildStaticSiteInvalidationVersion(input *StaticSiteManifestInputs, files []StaticSiteManifestFile) string {
	type config struct {
		AssetPath    *string                        `json:"assetPath,omitempty"`
		AssetRoutes  []string                       `json:"assetRoutes,omitempty"`
		Base         *string                        `json:"base,omitempty"`
		BucketDomain *string                        `json:"bucketDomain,omitempty"`
		ErrorPage    *string                        `json:"errorPage,omitempty"`
		FileOptions  []StaticSiteManifestFileOption `json:"fileOptions,omitempty"`
		KeyPrefix    *string                        `json:"keyPrefix,omitempty"`
	}

	hash := md5.New()
	data, _ := json.Marshal(config{
		AssetPath:    input.AssetPath,
		AssetRoutes:  input.AssetRoutes,
		Base:         input.Base,
		BucketDomain: input.BucketDomain,
		ErrorPage:    input.ErrorPage,
		FileOptions:  input.FileOptions,
		KeyPrefix:    input.KeyPrefix,
	})
	hash.Write(data)
	for _, file := range files {
		hash.Write([]byte(file.Key))
		hash.Write([]byte(file.Hash))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func staticSiteBuildKey(prefix *string, file string) string {
	if prefix == nil || *prefix == "" {
		return file
	}
	return path.Join(*prefix, file)
}

func staticSiteRelativeFile(outputPath, source string) string {
	relative, err := filepath.Rel(outputPath, source)
	if err != nil {
		return toStaticSitePosix(source)
	}
	return toStaticSitePosix(relative)
}

func matchesStaticSitePatterns(file string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := doublestar.PathMatch(toStaticSitePosix(pattern), file)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func normalizeStaticSiteBase(base *string) any {
	if base == nil || *base == "/" {
		return nil
	}
	return *base
}

func staticSiteDirPrefix(assetPath *string) string {
	if assetPath == nil || *assetPath == "" {
		return ""
	}
	return "/" + *assetPath
}

func derefStaticSiteString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func valueOrDefault(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func toStaticSitePosix(value string) string {
	return strings.ReplaceAll(value, string(filepath.Separator), "/")
}

func staticSiteContentType(filename string, textEncoding string) string {
	ext := path.Ext(filename)
	if strings.HasSuffix(filename, ".well-known/site-association-json") || strings.HasSuffix(filename, ".well-known/apple-app-site-association") {
		ext = ".json"
	}
	type contentType struct {
		Mime   string
		IsText bool
	}
	data := map[string]contentType{
		".txt":         {Mime: "text/plain", IsText: true},
		".htm":         {Mime: "text/html", IsText: true},
		".html":        {Mime: "text/html", IsText: true},
		".xhtml":       {Mime: "application/xhtml+xml", IsText: true},
		".css":         {Mime: "text/css", IsText: true},
		".js":          {Mime: "text/javascript", IsText: true},
		".mjs":         {Mime: "text/javascript", IsText: true},
		".apng":        {Mime: "image/apng"},
		".avif":        {Mime: "image/avif"},
		".gif":         {Mime: "image/gif"},
		".jpeg":        {Mime: "image/jpeg"},
		".jpg":         {Mime: "image/jpeg"},
		".png":         {Mime: "image/png"},
		".svg":         {Mime: "image/svg+xml", IsText: true},
		".bmp":         {Mime: "image/bmp"},
		".tiff":        {Mime: "image/tiff"},
		".webp":        {Mime: "image/webp"},
		".ico":         {Mime: "image/vnd.microsoft.icon"},
		".eot":         {Mime: "application/vnd.ms-fontobject"},
		".ttf":         {Mime: "font/ttf"},
		".otf":         {Mime: "font/otf"},
		".woff":        {Mime: "font/woff"},
		".woff2":       {Mime: "font/woff2"},
		".json":        {Mime: "application/json", IsText: true},
		".jsonld":      {Mime: "application/ld+json", IsText: true},
		".xml":         {Mime: "application/xml", IsText: true},
		".pdf":         {Mime: "application/pdf"},
		".zip":         {Mime: "application/zip"},
		".wasm":        {Mime: "application/wasm"},
		".webmanifest": {Mime: "application/manifest+json", IsText: true},
	}
	result, ok := data[ext]
	if !ok {
		result = contentType{Mime: "application/octet-stream"}
	}
	if result.IsText && textEncoding != "none" {
		return result.Mime + ";charset=" + textEncoding
	}
	return result.Mime
}
