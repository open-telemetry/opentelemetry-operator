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

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
)

type FileWatcher struct {
	configFilePath string
	watcher        *fsnotify.Watcher
	closer         chan bool
}

func newConfigMapWatcher(logger logr.Logger, config config.CLIConfig) (FileWatcher, error) {
	fileWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(err, "Can't start the watcher")
		return FileWatcher{}, err
	}

	return FileWatcher{
		configFilePath: *config.ConfigFilePath,
		watcher:        fileWatcher,
		closer:         make(chan bool),
	}, nil
}

func (f *FileWatcher) Start(upstreamEvents chan Event, upstreamErrors chan error) error {
	err := f.watcher.Add(filepath.Dir(f.configFilePath))
	if err != nil {
		return err
	}

	// translate and copy to central event channel
	go func() {
		for {
			select {
			case <-f.closer:
				return
			case fileEvent := <-f.watcher.Events:
				if fileEvent.Op == fsnotify.Create {
					upstreamEvents <- Event{
						Source: EventSourceConfigMap,
					}
				}
			case err := <-f.watcher.Errors:
				upstreamErrors <- err
			}
		}
	}()
	return nil
}

func (f *FileWatcher) Close() error {
	f.closer <- true
	return f.watcher.Close()
}
