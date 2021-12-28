package watcher

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"github.com/otel-allocator/allocation"
	"github.com/otel-allocator/config"
)

type Manager struct {
	configMapWatcher *fsnotify.Watcher
	PromCrWatcher    *PrometheusCRWatcher
	Events           chan Event
	Errors           chan error
	allocator        *allocation.Allocator
}

type Event struct {
	Source EventSource
}

type EventSource int

const (
	EventSourceConfigMap EventSource = iota
	EventSourcePrometheusCR
)

func NewWatcher(logger logr.Logger, config config.CLIConfig, allocator *allocation.Allocator) (*Manager, error) {
	fileWatcher, err := newConfigMapWatcher(logger, config)
	if err != nil {
		return nil, err
	}

	promWatcher, err := newCRDMonitorWatcher(logger, config)
	if err != nil {
		return nil, err
	}

	watcher := Manager{
		configMapWatcher: fileWatcher,
		PromCrWatcher:    promWatcher,
		allocator:        allocator,
		Events:           make(chan Event),
		Errors:           make(chan error),
	}
	startErr := watcher.Start()
	return &watcher, startErr
}

func (watcher Manager) Close() error {
	configMapErr := watcher.configMapWatcher.Close()
	prometheusCRErr := watcher.PromCrWatcher.Close()

	if configMapErr != nil && prometheusCRErr != nil {
		return fmt.Errorf("combined error: %v %v", configMapErr.Error(), prometheusCRErr.Error())
	}
	if configMapErr != nil {
		return configMapErr
	}
	if prometheusCRErr != nil {
		return prometheusCRErr
	}
	return nil
}

func (watcher Manager) Start() error {
	// translate and copy to central event channel
	go func() {
		for {
			select {
			case fileEvent := <-watcher.configMapWatcher.Events:
				if fileEvent.Op == fsnotify.Create {
					watcher.Events <- Event{
						Source: EventSourceConfigMap,
					}
				}
			case err := <-watcher.configMapWatcher.Errors:
				watcher.Errors <- err
			}
		}
	}()

	// copy to central event stream
	err := watcher.PromCrWatcher.Start()
	go func() {
		for {
			select {
			case event := <-watcher.PromCrWatcher.Events:
				watcher.Events <- event
			case err := <-watcher.PromCrWatcher.Errors:
				watcher.Errors <- err
			}
		}
	}()
	return err
}
