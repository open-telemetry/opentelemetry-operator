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
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	promconfig "github.com/prometheus/prometheus/config"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
)

var _ Watcher = &FileWatcher{}

type FileWatcher struct {
	logger         logr.Logger
	configFilePath string
	watcher        *fsnotify.Watcher
	closer         chan bool
}

func NewFileWatcher(logger logr.Logger, config config.CLIConfig) (*FileWatcher, error) {
	fileWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "Can't start the watcher")
		return &FileWatcher{}, err
	}

	return &FileWatcher{
		logger:         logger,
		configFilePath: *config.ConfigFilePath,
		watcher:        fileWatcher,
		closer:         make(chan bool),
	}, nil
}

func (f *FileWatcher) LoadConfig() (*promconfig.Config, error) {
	cfg, err := config.Load(f.configFilePath)
	if err != nil {
		f.logger.Error(err, "Unable to load configuration")
		return nil, err
	}
	return cfg.Config, nil
}

func (f *FileWatcher) Watch(upstreamEvents chan Event, upstreamErrors chan error) error {
	err := f.watcher.Add(filepath.Dir(f.configFilePath))
	if err != nil {
		return err
	}

	for {
		select {
		case <-f.closer:
			return nil
		case fileEvent := <-f.watcher.Events:
			if fileEvent.Op == fsnotify.Create {
				upstreamEvents <- Event{
					Source:  EventSourceConfigMap,
					Watcher: Watcher(f),
				}
			}
		case err := <-f.watcher.Errors:
			upstreamErrors <- err
		}
	}
}

func (f *FileWatcher) Close() error {
	f.closer <- true
	return f.watcher.Close()
}
