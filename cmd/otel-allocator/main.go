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

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	gokitlog "github.com/go-kit/log"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/discovery"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/server"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/collector"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/prehook"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/watcher"
)

var (
	setupLog     = ctrl.Log.WithName("setup")
	eventsMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "opentelemetry_allocator_events",
		Help: "Number of events in the channel.",
	}, []string{"source"})
)

func main() {
	var (
		// allocatorPrehook will be nil if filterStrategy is not set or
		// unrecognized. No filtering will be used in this case.
		allocatorPrehook prehook.Hook
		allocator        allocation.Allocator
		discoveryManager *discovery.Manager
		collectorWatcher *collector.Client
		fileWatcher      allocatorWatcher.Watcher
		promWatcher      allocatorWatcher.Watcher
		targetDiscoverer *target.Discoverer

		events          chan allocatorWatcher.Event
		errors          chan error
		discoveryCancel context.CancelFunc
		runGroup        run.Group
	)
	cliConf, err := config.ParseCLI()
	if err != nil {
		setupLog.Error(err, "Failed to parse parameters")
		os.Exit(1)
	}
	cfg, configLoadErr := config.Load(*cliConf.ConfigFilePath)
	if configLoadErr != nil {
		setupLog.Error(configLoadErr, "Unable to load configuration")
	}

	cliConf.RootLogger.Info("Starting the Target Allocator")
	ctx := context.Background()
	log := ctrl.Log.WithName("allocator")

	events = make(chan allocatorWatcher.Event)
	errors = make(chan error)
	allocatorPrehook = prehook.New(cfg.GetTargetsFilterStrategy(), log)
	allocator, err = allocation.New(cfg.GetAllocationStrategy(), log, allocation.WithFilter(allocatorPrehook))
	if err != nil {
		setupLog.Error(err, "Unable to initialize allocation strategy")
		os.Exit(1)
	}
	discoveryCtx, discoveryCancel := context.WithCancel(ctx)
	discoveryManager = discovery.NewManager(discoveryCtx, gokitlog.NewNopLogger())
	targetDiscoverer = target.NewDiscoverer(log, discoveryManager, allocatorPrehook)
	collectorWatcher, collectorWatcherErr := collector.NewClient(log, cliConf.ClusterConfig)
	if collectorWatcherErr != nil {
		setupLog.Error(collectorWatcherErr, "Unable to initialize collector watcher")
		os.Exit(1)
	}
	fileWatcher, err = allocatorWatcher.NewFileWatcher(setupLog.WithName("file-watcher"), cliConf)
	if err != nil {
		setupLog.Error(err, "Can't start the file watcher")
		os.Exit(1)
	}
	if *cliConf.PromCRWatcherConf.Enabled {
		promWatcher, err = allocatorWatcher.NewPrometheusCRWatcher(cfg, cliConf)
		if err != nil {
			setupLog.Error(err, "Can't start the prometheus watcher")
			os.Exit(1)
		}
		promWatcherErr := promWatcher.Start(events, errors)
		if promWatcherErr != nil {
			setupLog.Error(promWatcherErr, "Failed to start prometheus watcher")
			os.Exit(1)
		}
	}
	srv := server.NewServer(log, allocator, targetDiscoverer, cliConf.ListenAddr)
	interrupts := make(chan os.Signal, 1)
	closer := make(chan bool, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	runGroup.Add(
		func() error {
			fileWatcherErr := fileWatcher.Start(events, errors)
			setupLog.Info("File watcher exited")
			return fileWatcherErr
		},
		func(err error) {
			setupLog.Info("Closing file watcher")
			fileWatcherErr := fileWatcher.Close()
			if fileWatcherErr != nil {
				setupLog.Error(err, "file watcher failed to close")
			}
			setupLog.Info("Closing prometheus watcher")
			promWatcherErr := promWatcher.Close()
			if promWatcherErr != nil {
				setupLog.Error(err, "prometheus watcher failed to close")
			}
		})
	runGroup.Add(
		func() error {
			discoveryManagerErr := discoveryManager.Run()
			setupLog.Info("Discovery manager exited")
			return discoveryManagerErr
		},
		func(err error) {
			setupLog.Info("Closing discovery manager")
			discoveryCancel()
		})
	runGroup.Add(
		func() error {
			// Initial loading of the config file's scrape config
			err = targetDiscoverer.ApplyConfig(allocatorWatcher.EventSourceConfigMap, cfg.Config)
			if err != nil {
				setupLog.Error(err, "Unable to apply initial configuration")
				return err
			}
			err := targetDiscoverer.Watch(allocator.SetTargets)
			setupLog.Info("Target discoverer exited")
			return err
		},
		func(err error) {
			setupLog.Info("Closing target discoverer")
			targetDiscoverer.Close()
		})
	runGroup.Add(
		func() error {
			err := collectorWatcher.Watch(ctx, cfg.LabelSelector, allocator.SetCollectors)
			setupLog.Info("Collector watcher exited")
			return err
		},
		func(err error) {
			setupLog.Info("Closing collector watcher")
			collectorWatcher.Close()
		})
	runGroup.Add(
		func() error {
			err := srv.Start()
			setupLog.Info("Server failed to start")
			return err
		},
		func(err error) {
			setupLog.Info("Closing server")
			if err := srv.Shutdown(ctx); err != nil {
				setupLog.Error(err, "Error on server shutdown")
			}
		})
	runGroup.Add(
		func() error {
			for {
				select {
				case <-interrupts:
					setupLog.Info("Received interrupt")
					return nil
				case <-closer:
					setupLog.Info("Closing run loop")
					return nil
				case event := <-events:
					eventsMetric.WithLabelValues(event.Source.String()).Inc()
					loadConfig, err := event.Watcher.LoadConfig()
					if err != nil {
						setupLog.Error(err, "Unable to load configuration")
						continue
					}
					err = targetDiscoverer.ApplyConfig(event.Source, loadConfig)
					if err != nil {
						setupLog.Error(err, "Unable to apply configuration")
						continue
					}
				case err := <-errors:
					setupLog.Error(err, "Watcher error")
				}
			}
		},
		func(err error) {
			setupLog.Info("Received error, shutting down")
			close(closer)
		})

	if err := runGroup.Run(); err != nil {
		setupLog.Error(err, "run group exited")
	}
	setupLog.Info("Target allocator exited.")
}
