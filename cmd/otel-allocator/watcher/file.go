package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
)

func newConfigMapWatcher(logger logr.Logger, configDir string) (*fsnotify.Watcher, error) {
	fileWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "Can't start the watcher")
		return nil, err
	}
	err = fileWatcher.Add(configDir)
	if err != nil {
		return nil, err
	}
	return fileWatcher, nil
}
