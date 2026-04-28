package watcher

import (
	"context"
	"log/slog"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/project"
)

type FileChangedEvent struct {
	Path string
}

type WatchConfig struct {
	Root  string
	Watch project.Watch
}

type fileChange struct {
	timestamp  time.Time
	discovered bool
}

func resolveWatch(root string, watch project.Watch) ([]string, []string, error) {
	root = filepath.Clean(root)
	paths := watch.Paths
	if watch.UsesLegacyArray() {
		var err error
		paths, err = expandLegacyPaths(root, paths)
		if err != nil {
			return nil, nil, err
		}
	}

	roots, err := resolveRoots(root, paths)
	if err != nil {
		return nil, nil, err
	}

	ignore := make([]string, 0, len(watch.Ignore))
	for _, path := range watch.Ignore {
		ignore = append(ignore, normalizePath(path))
	}

	return roots, ignore, nil
}

func expandLegacyPaths(root string, paths []string) ([]string, error) {
	expanded := make([]string, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		matches := []string{path}
		if strings.ContainsAny(path, "*?[") {
			resolved, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(path)))
			if err != nil {
				return nil, err
			}
			matches = nil
			for _, match := range resolved {
				watchPath, err := legacyWatchPath(root, match)
				if err != nil {
					return nil, err
				}
				matches = append(matches, watchPath)
			}
		}

		for _, match := range matches {
			match = filepath.ToSlash(filepath.Clean(match))
			if seen[match] {
				continue
			}
			seen[match] = true
			expanded = append(expanded, match)
		}
	}

	return expanded, nil
}

func legacyWatchPath(root string, path string) (string, error) {
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(root, filepath.FromSlash(path))
	}

	info, err := os.Stat(resolved)
	if err == nil && !info.IsDir() {
		resolved = filepath.Dir(resolved)
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(rel), nil
}

func resolveRoots(root string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	seen := map[string]bool{}
	var roots []string

	for _, path := range paths {
		resolved := filepath.Clean(filepath.Join(root, filepath.FromSlash(path)))
		if seen[resolved] {
			continue
		}

		info, err := os.Stat(resolved)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		if !info.IsDir() {
			continue
		}

		seen[resolved] = true
		roots = append(roots, resolved)
	}

	return roots, nil
}

func shouldSkipDir(root string, ignore []string, path string, info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}

	if strings.HasPrefix(info.Name(), ".") {
		return true
	}

	if strings.Contains(path, "node_modules") {
		return true
	}

	return isIgnored(root, ignore, path)
}

func isIgnored(root string, ignore []string, path string) bool {
	if len(ignore) == 0 {
		return false
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	rel = normalizePath(rel)
	for _, prefix := range ignore {
		if matchesIgnore(prefix, rel) {
			return true
		}
	}

	return false
}

func matchesIgnore(pattern string, path string) bool {
	if pattern == "." {
		return true
	}

	if strings.Contains(pattern, "/") {
		return path == pattern || strings.HasPrefix(path, pattern+"/")
	}

	for part := range strings.SplitSeq(path, "/") {
		matched, err := pathpkg.Match(pattern, part)
		if err == nil && matched {
			return true
		}
	}

	return false
}

func isWatchedPath(roots []string, path string) bool {
	for _, root := range roots {
		if isWithinPath(root, path) {
			return true
		}
	}

	return false
}

func shouldWatchDir(projectRoot string, roots []string, path string) bool {
	if filepath.Clean(path) == filepath.Clean(projectRoot) {
		return true
	}
	if isWatchedPath(roots, path) {
		return true
	}
	if !isWithinPath(projectRoot, path) {
		return false
	}

	for _, root := range roots {
		if isWithinPath(path, root) {
			return true
		}
	}

	return false
}

func normalizePath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "./" {
		return "."
	}
	return strings.TrimPrefix(path, "./")
}

func Start(ctx context.Context, config WatchConfig) error {
	log := slog.Default().With("service", "watcher")
	log.Info("starting", "root", config.Root)
	defer log.Info("done")

	roots, ignore, err := resolveWatch(config.Root, config.Watch)
	if err != nil {
		return err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.AddWith(config.Root)
	if err != nil {
		return err
	}

	limiter := map[string]fileChange{}

	publishChange := func(path string, discovered bool) {
		last, ok := limiter[path]
		if ok && time.Since(last.timestamp) <= 500*time.Millisecond {
			if discovered || !last.discovered {
				return
			}
		}

		limiter[path] = fileChange{timestamp: time.Now(), discovered: discovered}
		bus.Publish(&FileChangedEvent{Path: path})
	}

	watchTree := func(path string, publish bool) error {
		return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}

			if info.IsDir() {
				if shouldSkipDir(config.Root, ignore, path, info) {
					return filepath.SkipDir
				}
				if !shouldWatchDir(config.Root, roots, path) {
					return filepath.SkipDir
				}

				log.Info("watching", "path", path)
				return watcher.Add(path)
			}

			if publish && isWatchedPath(roots, path) && !isIgnored(config.Root, ignore, path) {
				log.Info("discovered new file in directory", "path", path)
				publishChange(path, true)
			}

			return nil
		})
	}

	err = watchTree(config.Root, false)
	if err != nil {
		return err
	}

	for _, match := range roots {
		if isWithinPath(config.Root, match) {
			continue
		}
		err = watchTree(match, false)
		if err != nil {
			return err
		}
	}

	headFile := filepath.Join(config.Root, ".git/HEAD")
	watcher.Add(headFile)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				log.Info("ignoring file event", "path", event.Name, "op", event.Op)
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				info, err := os.Stat(event.Name)
				if err != nil && !os.IsNotExist(err) {
					return err
				}
				if err == nil && info.IsDir() {
					if shouldSkipDir(config.Root, ignore, event.Name, info) || !shouldWatchDir(config.Root, roots, event.Name) {
						log.Info("ignoring created directory", "path", event.Name)
						continue
					}

					err = watchTree(event.Name, true)
					if err != nil {
						return err
					}
					continue
				}
			}
			if !isWatchedPath(roots, event.Name) {
				log.Info("ignoring unwatched file event", "path", event.Name, "op", event.Op)
				continue
			}
			if isIgnored(config.Root, ignore, event.Name) {
				log.Info("ignoring ignored file event", "path", event.Name, "op", event.Op)
				continue
			}
			log.Info("file event", "path", event.Name, "op", event.Op)
			publishChange(event.Name, false)
		case <-ctx.Done():
			return nil
		}
	}
}

func isWithinPath(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	rel = normalizePath(rel)
	return rel == "." || (!strings.HasPrefix(rel, "../") && rel != "..")
}
