package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"github.com/otel-allocator/config"
	"path/filepath"
)

func newConfigMapWatcher(logger logr.Logger, config config.CLIConfig) (*fsnotify.Watcher, error) {
	fileWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "Can't start the watcher")
		return nil, err
	}
	err = fileWatcher.Add(filepath.Dir(*config.ConfigFilePath))
	if err != nil {
		return nil, err
	}
	return fileWatcher, nil
}
