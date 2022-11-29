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
	gokitlog "github.com/go-kit/log"
	"github.com/oklog/run"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/server"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/discovery"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"syscall"

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
	cliConf, err := config.ParseCLI()
	if err != nil {
		setupLog.Error(err, "Failed to parse parameters")
		os.Exit(1)
	}
	cfg, err := config.Load(*cliConf.ConfigFilePath)
	if err != nil {
		setupLog.Error(err, "Unable to load configuration")
	}

	cliConf.RootLogger.Info("Starting the Target Allocator")
	ctx := context.Background()
	log := ctrl.Log.WithName("allocator")

	var (
		// allocatorPrehook will be nil if filterStrategy is not set or
		// unrecognized. No filtering will be used in this case.
		allocatorPrehook prehook.Hook
		allocator        allocation.Allocator
		discoveryManager *discovery.Manager
		fileWatcher      allocatorWatcher.Watcher
		promWatcher      *allocatorWatcher.PrometheusCRWatcher
		targetDiscoverer *target.Discoverer

		events          chan allocatorWatcher.Event
		errors          chan error
		discoveryCancel context.CancelFunc
		runGroup        run.Group
	)
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
	collectorWatcher, err := collector.NewClient(log, cliConf.ClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to initialize collector watcher")
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
		err := promWatcher.Start(events, errors)
		if err != nil {
			setupLog.Error(err, "Failed to start prometheus watcher")
			os.Exit(1)
		}
	}
	srv := server.NewServer(log, allocator, targetDiscoverer, cliConf.ListenAddr)
	interrupts := make(chan os.Signal, 1)
	closer := make(chan bool, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	runGroup.Add(
		func() error {
			err := fileWatcher.Start(events, errors)
			setupLog.Info("File watcher exited")
			return err
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
			err := discoveryManager.Run()
			setupLog.Info("Discovery manager exited")
			return err
		},
		func(err error) {
			setupLog.Info("Closing discovery manager")
			discoveryCancel()
		})
	runGroup.Add(
		func() error {
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
					switch event.Source {
					case allocatorWatcher.EventSourceConfigMap:
						setupLog.Info("Config map changes")
						cfg, err := config.Load(*cliConf.ConfigFilePath)
						if err != nil {
							setupLog.Error(err, "Unable to load configuration")
							return err
						}
						err = targetDiscoverer.ApplyConfig(event.Source, cfg.Config)
						if err != nil {
							setupLog.Error(err, "Unable to apply configuration")
							continue
						}
					case allocatorWatcher.EventSourcePrometheusCR:
						setupLog.Info("PrometheusCRs changed")
						promConfig, err := promWatcher.CreatePromConfig(cliConf.KubeConfigFilePath)
						if err != nil {
							setupLog.Error(err, "failed to compile Prometheus config")
							continue
						}
						err = targetDiscoverer.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, promConfig)
						if err != nil {
							setupLog.Error(err, "failed to apply Prometheus config")
						}
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
		os.Exit(1)
	}
	setupLog.Info("Target allocator exited.")
}
