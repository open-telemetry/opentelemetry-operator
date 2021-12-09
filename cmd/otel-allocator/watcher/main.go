package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
)

type Manager struct {
	configMapWatcher *fsnotify.Watcher
	promCrWatcher    *PrometheusCRWatcher
	Events           chan Event
	Errors           chan error
}

type Event struct {
	Source EventSource
}

type EventSource int

const (
	EventSourceConfigMap EventSource = iota
	EventSourcePrometheusCR
)

func NewWatcher(logger logr.Logger, configDir string) (*Manager, error) {
	fileWatcher, err := newConfigMapWatcher(logger, configDir)
	if err != nil {
		return nil, err
	}

	promWatcher, err := newCRDMonitorWatcher()
	if err != nil {
		return nil, err
	}

	watcher := Manager{
		configMapWatcher: fileWatcher,
		promCrWatcher:    promWatcher,
	}
	return &watcher, nil
}

func (watcher Manager) Close() error {
	return watcher.configMapWatcher.Close()
}

func (watcher Manager) Start() {
	go func() {
		for {
			select {
			case fileEvent := <-watcher.configMapWatcher.Events:
				if fileEvent.Op == fsnotify.Create {
					watcher.Events <- Event{Source: EventSourceConfigMap}
				}
			case errorEvent := <-watcher.configMapWatcher.Errors:
				watcher.Errors <- errorEvent
			}
		}
	}()
}
