// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/oklog/run"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/discovery"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/collector"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/prehook"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/server"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/watcher"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	var (
		// allocatorPrehook will be nil if filterStrategy is not set or
		// unrecognized. No filtering will be used in this case.
		allocatorPrehook prehook.Hook
		allocator        allocation.Allocator
		discoveryManager *discovery.Manager
		collectorWatcher *collector.Watcher
		targetDiscoverer *target.Discoverer

		discoveryCancel context.CancelFunc
		runGroup        run.Group
		eventChan       = make(chan allocatorWatcher.Event)
		eventCloser     = make(chan bool, 1)
		interrupts      = make(chan os.Signal, 1)
		errChan         = make(chan error)
	)
	cfg, loadErr := config.Load(os.Args)
	if loadErr != nil {
		fmt.Printf("Failed to load config: %v", loadErr)
		os.Exit(1)
	}
	ctrl.SetLogger(cfg.RootLogger)

	if validationErr := config.ValidateConfig(cfg); validationErr != nil {
		setupLog.Error(validationErr, "Invalid configuration")
		os.Exit(1)
	}

	cfg.RootLogger.Info("Starting the Target Allocator")
	ctx := context.Background()
	log := ctrl.Log.WithName("allocator")

	k8sClient, err := kubernetes.NewForConfig(cfg.ClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to initialize kubernetes client")
		os.Exit(1)
	}
	monitoringClient, err := monitoringclient.NewForConfig(cfg.ClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to initialize monitoring client")
		os.Exit(1)
	}

	metricExporter, promErr := otelprom.New()
	if promErr != nil {
		panic(promErr)
	}
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricExporter))
	otel.SetMeterProvider(meterProvider)

	allocatorPrehook = prehook.New(cfg.FilterStrategy, log)
	allocator, allocErr := allocation.New(cfg.AllocationStrategy, log, allocation.WithFilter(allocatorPrehook), allocation.WithFallbackStrategy(cfg.AllocationFallbackStrategy))
	if allocErr != nil {
		setupLog.Error(allocErr, "Unable to initialize allocation strategy")
		os.Exit(1)
	}

	httpOptions := []server.Option{}
	if cfg.HTTPS.Enabled {
		tlsConfig, confErr := cfg.HTTPS.NewTLSConfig()
		if confErr != nil {
			setupLog.Error(confErr, "Unable to initialize TLS configuration")
			os.Exit(1)
		}
		httpOptions = append(httpOptions, server.WithTLSConfig(tlsConfig, cfg.HTTPS.ListenAddr))
	}
	srv, serverErr := server.NewServer(log, allocator, cfg.ListenAddr, httpOptions...)
	if serverErr != nil {
		panic(serverErr)
	}

	discoveryCtx, discoveryCancel := context.WithCancel(ctx)
	sdMetrics, discErr := discovery.CreateAndRegisterSDMetrics(prometheus.DefaultRegisterer)
	if discErr != nil {
		setupLog.Error(discErr, "Unable to register metrics for Prometheus service discovery")
		os.Exit(1)
	}
	discoveryManager = discovery.NewManager(discoveryCtx, config.NopLogger, prometheus.DefaultRegisterer, sdMetrics)

	targetDiscoverer, targetErr := target.NewDiscoverer(log, discoveryManager, allocatorPrehook, srv, allocator.SetTargets)
	if targetErr != nil {
		panic(targetErr)
	}
	collectorWatcher, collectorWatcherErr := collector.NewCollectorWatcher(log, k8sClient, cfg.CollectorNotReadyGracePeriod)
	if collectorWatcherErr != nil {
		setupLog.Error(collectorWatcherErr, "Unable to initialize collector watcher")
		os.Exit(1)
	}
	signal.Notify(interrupts, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer close(interrupts)

	if cfg.PrometheusCR.Enabled {
		promWatcher, allocErr := allocatorWatcher.NewPrometheusCRWatcher(
			ctx, setupLog.WithName("prometheus-cr-watcher"), k8sClient, monitoringClient, *cfg)
		if allocErr != nil {
			setupLog.Error(allocErr, "Can't start the prometheus watcher")
			os.Exit(1)
		}
		// apply the initial configuration
		promConfig, loadErr := promWatcher.LoadConfig(ctx)
		if loadErr != nil {
			setupLog.Error(loadErr, "Can't load initial Prometheus configuration from Prometheus CRs")
			os.Exit(1)
		}
		loadErr = targetDiscoverer.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, promConfig.ScrapeConfigs)
		if loadErr != nil {
			setupLog.Error(loadErr, "Can't load initial scrape targets from Prometheus CRs")
			os.Exit(1)
		}
		runGroup.Add(
			func() error {
				promWatcherErr := promWatcher.Watch(eventChan, errChan)
				setupLog.Info("Prometheus watcher exited")
				return promWatcherErr
			},
			func(_ error) {
				setupLog.Info("Closing prometheus watcher")
				promWatcherErr := promWatcher.Close()
				if promWatcherErr != nil {
					setupLog.Error(promWatcherErr, "prometheus watcher failed to close")
				}
			})
	}
	runGroup.Add(
		func() error {
			discoveryManagerErr := discoveryManager.Run()
			setupLog.Info("Discovery manager exited")
			return discoveryManagerErr
		},
		func(_ error) {
			setupLog.Info("Closing discovery manager")
			discoveryCancel()
		})
	runGroup.Add(
		func() error {
			// Initial loading of the config file's scrape config
			if cfg.PromConfig != nil && len(cfg.PromConfig.ScrapeConfigs) > 0 {
				applyErr := targetDiscoverer.ApplyConfig(allocatorWatcher.EventSourceConfigMap, cfg.PromConfig.ScrapeConfigs)
				if applyErr != nil {
					setupLog.Error(err, "Unable to apply initial configuration")
					return err
				}
			} else {
				setupLog.Info("Prometheus config empty, skipping initial discovery configuration")
			}

			tErr := targetDiscoverer.Run()
			setupLog.Info("Target discoverer exited")
			return tErr
		},
		func(_ error) {
			setupLog.Info("Closing target discoverer")
			targetDiscoverer.Close()
		})
	runGroup.Add(
		func() error {
			watchErr := collectorWatcher.Watch(cfg.CollectorNamespace, cfg.CollectorSelector, allocator.SetCollectors)
			setupLog.Info("Collector watcher exited")
			return watchErr
		},
		func(_ error) {
			setupLog.Info("Closing collector watcher")
			collectorWatcher.Close()
		})
	runGroup.Add(
		func() error {
			startErr := srv.Start()
			setupLog.Info("Server failed to start")
			return startErr
		},
		func(_ error) {
			setupLog.Info("Closing server")
			if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
				setupLog.Error(shutdownErr, "Error on server shutdown")
			}
		})
	if cfg.HTTPS.Enabled {
		runGroup.Add(
			func() error {
				startErr := srv.StartHTTPS()
				setupLog.Info("HTTPS Server failed to start")
				return startErr
			},
			func(_ error) {
				setupLog.Info("Closing HTTPS server")
				if shutdownErr := srv.ShutdownHTTPS(ctx); shutdownErr != nil {
					setupLog.Error(shutdownErr, "Error on HTTPS server shutdown")
				}
			})
	}
	meter := otel.GetMeterProvider().Meter("targetallocator")
	eventsMetric, err := meter.Int64Counter("opentelemetry_allocator_events", metric.WithDescription("Number of events in the channel."))
	if err != nil {
		panic(err)
	}
	runGroup.Add(
		func() error {
			for {
				select {
				case event := <-eventChan:
					eventsMetric.Add(context.Background(), 1, metric.WithAttributes(attribute.String("source", event.Source.String())))
					loadConfig, err := event.Watcher.LoadConfig(ctx)
					if err != nil {
						setupLog.Error(err, "Unable to load configuration")
						continue
					}
					err = targetDiscoverer.ApplyConfig(event.Source, loadConfig.ScrapeConfigs)
					if err != nil {
						setupLog.Error(err, "Unable to apply configuration")
						continue
					}
				case err := <-errChan:
					setupLog.Error(err, "Watcher error")
				case <-eventCloser:
					return nil
				}
			}
		},
		func(_ error) {
			setupLog.Info("Closing watcher loop")
			close(eventCloser)
		})
	runGroup.Add(
		func() error {
			for {
				select {
				case <-interrupts:
					setupLog.Info("Received interrupt")
					return nil
				case <-eventCloser:
					return nil
				}
			}
		},
		func(_ error) {
			setupLog.Info("Closing interrupt loop")
		})
	if runErr := runGroup.Run(); runErr != nil {
		setupLog.Error(runErr, "run group exited")
	}
	setupLog.Info("Target allocator exited.")
}
