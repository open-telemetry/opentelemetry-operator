// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/collector"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/prehook"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/server"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/watcher"
)

var setupLog = ctrl.Log.WithName("setup")

func main() {
	var (
		// allocatorPrehook will be nil if filterStrategy is not set or
		// unrecognized. No filtering will be used in this case.
		allocatorPrehook prehook.Hook
		allocator        allocation.Allocator
		discoveryManager *discovery.Manager
		collectorWatcher *collector.Watcher
		targetDiscoverer *target.Discoverer
		certWatcher      *certwatcher.CertWatcher

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

	// When zone-aware allocation is enabled we build a ZoneTopology that
	// tracks per-zone collector/target distribution. This drives the
	// /zones API endpoint, the per-zone metrics, and the zone-aware
	// allocation strategies. It is created before the allocator so it can
	// be passed in as an option.
	var zoneTopology *allocation.ZoneTopology
	if cfg.Topology.ZoneAware {
		var ztErr error
		zoneTopology, ztErr = allocation.NewZoneTopology(log, cfg.Topology.TargetZoneLabel)
		if ztErr != nil {
			setupLog.Error(ztErr, "Unable to initialize zone topology tracker")
			os.Exit(1)
		}
	}

	allocatorOpts := []allocation.Option{
		allocation.WithFilter(allocatorPrehook),
		allocation.WithFallbackStrategy(cfg.AllocationFallbackStrategy),
	}
	if zoneTopology != nil {
		// Order matters: the strategy reads maxSkew from the allocator at
		// the moment WithZoneTopology fires (it triggers SetZoneAwareness),
		// so install maxSkew first.
		allocatorOpts = append(allocatorOpts,
			allocation.WithMaxSkew(cfg.Topology.MaxSkew),
			allocation.WithZoneTopology(zoneTopology),
		)
	}
	allocator, allocErr := allocation.New(cfg.AllocationStrategy, log, allocatorOpts...)
	if allocErr != nil {
		setupLog.Error(allocErr, "Unable to initialize allocation strategy")
		os.Exit(1)
	}

	httpOptions := []server.Option{}
	if cfg.HTTPS.Enabled {
		var tlsConfig *tls.Config
		var confErr error
		tlsConfig, certWatcher, confErr = cfg.HTTPS.NewTLSConfig(log)
		if confErr != nil {
			setupLog.Error(confErr, "Unable to initialize TLS configuration")
			os.Exit(1)
		}
		httpOptions = append(httpOptions, server.WithTLSConfig(tlsConfig, cfg.HTTPS.ListenAddr))
	}
	if cfg.AllowInsecureAuthSecrets {
		httpOptions = append(httpOptions, server.WithInsecureAuthSecrets())
	}
	srv, serverErr := server.NewServer(log, allocator, cfg.ListenAddr, httpOptions...)
	if serverErr != nil {
		panic(serverErr)
	}

	discoveryCtx, discoveryCancel := context.WithCancel(ctx)
	defer discoveryCancel()
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
	var zoneResolver *allocation.NodeZoneResolver
	if cfg.Topology.ZoneAware {
		zoneResolver = allocation.NewNodeZoneResolver(log, k8sClient, cfg.Topology.ZoneLabel)
		if syncErr := zoneResolver.SyncNodes(ctx); syncErr != nil {
			setupLog.Error(syncErr, "Failed to sync node zones on startup, zone-aware allocation may be degraded")
		}
	}
	collectorWatcher, collectorWatcherErr := collector.NewCollectorWatcher(log, k8sClient, cfg.CollectorNotReadyGracePeriod, zoneResolver)
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

		// Start certificate watchers for hot-reload
		certWatcherCtx, certWatcherCancel := context.WithCancel(ctx)
		defer certWatcherCancel()
		// Server certificate watcher
		runGroup.Add(
			func() error {
				watchErr := certWatcher.Start(certWatcherCtx)
				setupLog.Info("Certificate watcher exited")
				return watchErr
			},
			func(_ error) {
				setupLog.Info("Closing certificate watcher")
				certWatcherCancel()
			})
	}
	// Periodic node-zone re-sync. The collector pod informer already re-syncs
	// at its own cadence (~30s) and applies the latest resolver state when
	// pods are listed, so the worst-case staleness an operator sees is
	// roughly NodeSyncInterval + the pod informer resync. A NodeSyncInterval
	// of 0 disables the periodic re-sync entirely (kept for tests and for
	// operators who prefer to manage cluster topology out-of-band).
	if zoneResolver != nil && cfg.Topology.NodeSyncInterval > 0 {
		nodeSyncCtx, nodeSyncCancel := context.WithCancel(ctx)
		defer nodeSyncCancel()
		runGroup.Add(
			func() error {
				ticker := time.NewTicker(cfg.Topology.NodeSyncInterval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						if syncErr := zoneResolver.SyncNodes(nodeSyncCtx); syncErr != nil {
							// We log and continue: a transient API error here
							// keeps the previous (still-valid) zone map in
							// place rather than wiping it. The next tick
							// retries.
							setupLog.Error(syncErr, "Periodic node zone re-sync failed; using previously cached zone map")
						}
					case <-nodeSyncCtx.Done():
						return nil
					}
				}
			},
			func(_ error) {
				setupLog.Info("Closing node zone re-syncer")
				nodeSyncCancel()
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
