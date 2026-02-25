package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sst/sst/v3/pkg/bus"
)

type FileChangedEvent struct {
	Path string
}

type WatchConfig struct {
	Root  string
	Watch []string
}

func Start(ctx context.Context, config WatchConfig) error {
	log := slog.Default().With("service", "watcher")
	defer log.Info("done")
	log.Info("starting watcher", "root", config.Root)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = watcher.AddWith(config.Root)
	if err != nil {
		return err
	}

	matches := []string{}
	if len(config.Watch) == 0 {
		matches = append(matches, config.Root)
	} else {
		for _, watch := range config.Watch {
			pattern := filepath.Join(config.Root, watch)
			patternMatches, err := filepath.Glob(pattern)
			if err != nil {
				return err
			}
			matches = append(matches, patternMatches...)
		}
	}

	ignoreSubstrings := []string{"node_modules"}
	for _, match := range matches {
		err = filepath.Walk(match, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if strings.HasPrefix(info.Name(), ".") {
					return filepath.SkipDir
				}
				for _, substring := range ignoreSubstrings {
					if strings.Contains(path, substring) {
						return filepath.SkipDir
					}
				}

				log.Info("watching", "path", path)
				err = watcher.Add(path)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	headFile := filepath.Join(config.Root, ".git/HEAD")
	watcher.Add(headFile)
	limiter := map[string]time.Time{}
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// if event.Name == headFile {
			// 	return nil
			// }
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				log.Info("ignoring file event", "path", event.Name, "op", event.Op)
				continue
			}
			log.Info("file event", "path", event.Name, "op", event.Op)
			if time.Since(limiter[event.Name]) > 500*time.Millisecond {
				limiter[event.Name] = time.Now()
				bus.Publish(&FileChangedEvent{Path: event.Name})
			}
		case <-ctx.Done():
			return nil
		}
	}
}
