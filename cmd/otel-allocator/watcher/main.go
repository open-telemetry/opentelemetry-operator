package watcher

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
)

type Manager struct {
	Events    chan Event
	Errors    chan error
	allocator *allocation.Allocator
	watchers  []Watcher
}

type Watcher interface {
	// Start watcher and supply channels which will receive change events
	Start(upstreamEvents chan Event, upstreamErrors chan error) error
	Close() error
}

type Event struct {
	Source  EventSource
	Watcher *Watcher
}

type EventSource int

const (
	EventSourceConfigMap EventSource = iota
	EventSourcePrometheusCR
)

func NewWatcher(logger logr.Logger, config config.CLIConfig, allocator *allocation.Allocator) (*Manager, error) {
	watcher := Manager{
		allocator: allocator,
		Events:    make(chan Event),
		Errors:    make(chan error),
	}

	fileWatcher, err := newConfigMapWatcher(logger, config)
	if err != nil {
		return nil, err
	}
	watcher.watchers = append(watcher.watchers, &fileWatcher)

	if *config.PromCRWatcherConf.Enabled {
		promWatcher, err := newCRDMonitorWatcher(logger, config)
		if err != nil {
			return nil, err
		}
		watcher.watchers = append(watcher.watchers, promWatcher)
	}

	startErr := watcher.Start()
	return &watcher, startErr
}

func (manager *Manager) Close() error {
	var errors []error
	for _, watcher := range manager.watchers {
		err := watcher.Close()
		if err != nil {
			errors = append(errors, err)
		}
	}

	close(manager.Events)
	close(manager.Errors)

	if len(errors) > 0 {
		return fmt.Errorf("closing errors: %+v", errors)
	}
	return nil
}

func (manager *Manager) Start() error {
	var errors []error
	for _, watcher := range manager.watchers {
		err := watcher.Start(manager.Events, manager.Errors)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("closing errors: %+v", errors)
	}
	return nil
}
