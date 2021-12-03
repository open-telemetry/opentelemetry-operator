package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
)

type Watcher struct {
	configMapWatcher *fsnotify.Watcher
	Events           chan Event
	Errors           chan error
}

type Event struct {
	fsnotify.Event
}

func NewWatcher(logger logr.Logger, configDir string) (*Watcher, error) {
	fileWatcher, err := newConfigMapWatcher(logger, configDir)
	if err != nil {
		return nil, err
	}

	watcher := Watcher{
		configMapWatcher: fileWatcher,
	}
	watcher.start()
	return &watcher, nil
}

func (watcher Watcher) Close() error {
	return watcher.configMapWatcher.Close()
}

func (watcher Watcher) start() {
	go func() {
		for {
			select {
			case fileEvent := <-watcher.configMapWatcher.Events:
				watcher.Events <- Event{fileEvent}
			case errorEvent := <-watcher.configMapWatcher.Errors:
				watcher.Errors <- errorEvent
			}
		}
	}()
}
