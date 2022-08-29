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
	return f.watcher.Close()
}
