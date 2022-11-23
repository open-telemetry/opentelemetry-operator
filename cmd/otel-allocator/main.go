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
	"github.com/go-logr/logr"
	"github.com/oklog/run"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"net/http"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"syscall"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/collector"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
	lbdiscovery "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/discovery"
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
		fileWatcher      allocatorWatcher.Watcher
		promWatcher      allocatorWatcher.Watcher
		discoveryManager *lbdiscovery.Manager
		runGroup         run.Group
	)
	allocatorPrehook = prehook.New(cfg.GetTargetsFilterStrategy(), log)
	allocator, err = allocation.New(cfg.GetAllocationStrategy(), log, allocation.WithFilter(allocatorPrehook))
	if err != nil {
		setupLog.Error(err, "Unable to initialize allocation strategy")
		os.Exit(1)
	}
	fileWatcher, err = allocatorWatcher.NewFileWatcher(setupLog.WithName("file-watcher"), cliConf)
	if err != nil {
		setupLog.Error(err, "Can't start the watchers")
		os.Exit(1)
	}
	if *cliConf.PromCRWatcherConf.Enabled {
		promWatcher, err = allocatorWatcher.NewFileWatcher(setupLog.WithName("file-watcher"), cliConf)
		if err != nil {
			setupLog.Error(err, "Can't start the watchers")
			os.Exit(1)
		}
	}

	runGroup.Add(
		func() error {

		},
		func(err error) {

		})
	defer func() {
		err := watcher.Close()
		if err != nil {
			log.Error(err, "failed to close watcher")
		}
	}()

	// creates a new discovery manager
	discoveryManager = lbdiscovery.NewManager(log, ctx, gokitlog.NewNopLogger(), allocatorPrehook)
	defer discoveryManager.Close()

	discoveryManager.Watch(allocator.SetTargets)

	k8sclient, err := configureFileDiscovery(log, allocator, discoveryManager, context.Background(), cliConf)
	if err != nil {
		setupLog.Error(err, "Can't start the k8s client")
		os.Exit(1)
	}

	srv := server.NewServer(log, allocator, discoveryManager, k8sclient, cliConf.ListenAddr)

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != http.ErrServerClosed {
			setupLog.Error(err, "Can't start the server")
		}
	}()

	for {
		select {
		case <-interrupts:
			if err := srv.Shutdown(ctx); err != nil {
				setupLog.Error(err, "Error on server shutdown")
				os.Exit(1)
			}
			os.Exit(0)
		case event := <-watcher.Events:
			eventsMetric.WithLabelValues(event.Source.String()).Inc()
			switch event.Source {
			case allocatorWatcher.EventSourceConfigMap:
				setupLog.Info("ConfigMap updated!")
				// Restart the server to pickup the new config.
				if err := srv.Shutdown(ctx); err != nil {
					setupLog.Error(err, "Cannot shutdown the server")
				}
				srv = server.NewServer(log, allocator, discoveryManager, k8sclient, cliConf.ListenAddr)
				go func() {
					if err := srv.Start(); err != http.ErrServerClosed {
						setupLog.Error(err, "Can't restart the server")
					}
				}()

			case allocatorWatcher.EventSourcePrometheusCR:
				setupLog.Info("PrometheusCRs changed")
				promConfig, err := interface{}(*event.Watcher).(*allocatorWatcher.PrometheusCRWatcher).CreatePromConfig(cliConf.KubeConfigFilePath)
				if err != nil {
					setupLog.Error(err, "failed to compile Prometheus config")
				}
				err = discoveryManager.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, promConfig)
				if err != nil {
					setupLog.Error(err, "failed to apply Prometheus config")
				}
			}
		case err := <-watcher.Errors:
			setupLog.Error(err, "Watcher error")
		}
	}
}

func configureFileDiscovery(log logr.Logger, allocator allocation.Allocator, discoveryManager *lbdiscovery.Manager, ctx context.Context, cliConfig config.CLIConfig) (*collector.Client, error) {
	cfg, err := config.Load(*cliConfig.ConfigFilePath)
	if err != nil {
		return nil, err
	}

	k8sClient, err := collector.NewClient(log, cliConfig.ClusterConfig)
	if err != nil {
		return nil, err
	}

	// returns the list of targets
	if err := discoveryManager.ApplyConfig(allocatorWatcher.EventSourceConfigMap, cfg.Config); err != nil {
		return nil, err
	}

	k8sClient.Watch(ctx, cfg.LabelSelector, allocator.SetCollectors)
	return k8sClient, nil
}
