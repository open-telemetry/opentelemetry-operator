// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	allocator allocation.Allocator
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

var (
	eventSourceToString = map[EventSource]string{
		EventSourceConfigMap:    "EventSourceConfigMap",
		EventSourcePrometheusCR: "EventSourcePrometheusCR",
	}
)

func (e EventSource) String() string {
	return eventSourceToString[e]
}

func NewWatcher(logger logr.Logger, cfg config.Config, cliConfig config.CLIConfig, allocator allocation.Allocator) (*Manager, error) {
	watcher := Manager{
		allocator: allocator,
		Events:    make(chan Event),
		Errors:    make(chan error),
	}

	fileWatcher, err := NewFileWatcher(logger, cliConfig)
	if err != nil {
		return nil, err
	}
	watcher.watchers = append(watcher.watchers, fileWatcher)

	if *cliConfig.PromCRWatcherConf.Enabled {
		promWatcher, err := newCRDMonitorWatcher(cfg, cliConfig)
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
