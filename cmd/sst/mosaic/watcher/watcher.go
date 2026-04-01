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
	at         time.Time
	discovered bool
}

func resolveWatch(root string, watch project.Watch) ([]string, []string, error) {
	root = filepath.Clean(root)

	roots, err := resolveRoots(root, watch.Paths)
	if err != nil {
		return nil, nil, err
	}

	ignore := make([]string, 0, len(watch.Ignore))
	for _, path := range watch.Ignore {
		ignore = append(ignore, normalizePath(path))
	}

	return roots, ignore, nil
}

func resolveRoots(root string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	seen := map[string]bool{}
	var roots []string

	for _, path := range paths {
		resolved := filepath.Clean(filepath.Join(root, path))

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

func normalizePath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "./" {
		return "."
	}
	return strings.TrimPrefix(path, "./")
}

func watchTree(log *slog.Logger, watcher *fsnotify.Watcher, root string, ignore []string, path string, onFile func(string)) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if info.IsDir() {
			if shouldSkipDir(root, ignore, path, info) {
				return filepath.SkipDir
			}

			log.Info("watching", "path", path)
			return watcher.Add(path)
		}

		if onFile != nil && !isIgnored(root, ignore, path) {
			onFile(path)
		}

		return nil
	})
}

func publishFileChanged(limiter map[string]fileChange, path string, discovered bool) {
	last, ok := limiter[path]
	if ok && time.Since(last.at) <= 500*time.Millisecond {
		if discovered || !last.discovered {
			return
		}
	}

	limiter[path] = fileChange{at: time.Now(), discovered: discovered}
	bus.Publish(&FileChangedEvent{Path: path})
}

func Start(ctx context.Context, config WatchConfig) error {
	log := slog.Default().With("service", "watcher")
	defer log.Info("done")
	log.Info("starting watcher", "root", config.Root)
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

	for _, match := range roots {
		err = watchTree(log, watcher, config.Root, ignore, match, nil)
		if err != nil {
			return err
		}
	}

	headFile := filepath.Join(config.Root, ".git/HEAD")
	watcher.Add(headFile)
	limiter := map[string]fileChange{}
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
					if shouldSkipDir(config.Root, ignore, event.Name, info) {
						log.Info("ignoring created directory", "path", event.Name)
						continue
					}

					err = watchTree(log, watcher, config.Root, ignore, event.Name, func(path string) {
						log.Info("discovered file in new directory", "path", path)
						publishFileChanged(limiter, path, true)
					})
					if err != nil {
						return err
					}
					continue
				}
			}
			if isIgnored(config.Root, ignore, event.Name) {
				log.Info("ignoring ignored file event", "path", event.Name, "op", event.Op)
				continue
			}
			log.Info("file event", "path", event.Name, "op", event.Op)
			publishFileChanged(limiter, event.Name, false)
		case <-ctx.Done():
			return nil
		}
	}
}
