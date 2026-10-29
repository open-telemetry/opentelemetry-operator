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
	"go.opentelemetry.io/otel/metric"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/collector"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/server"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/telemetry"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/watcher"
)

var setupLog = ctrl.Log.WithName("setup")

// crdRecheckInterval is how often the target allocator checks whether Prometheus CRDs absent at
// startup have been installed. See open-telemetry/opentelemetry-operator#5577.
const crdRecheckInterval = 5 * time.Minute

// promCRWatcher is the subset of the Prometheus CR watcher used by runPromWatcher.
type promCRWatcher interface {
	Watch(upstreamEvents chan allocatorWatcher.Event, upstreamErrors chan error) error
	Close() error
	MissingCRDsAvailable() (bool, error)
}

// runPromWatcher runs the Prometheus CR watcher until stop is closed or the watcher returns.
// On every recheck tick, if a Prometheus CRD absent when the watcher was created has been installed,
// the watcher is replaced by one created with newWatcher, so that the new CRD is watched.
func runPromWatcher(
	watcher promCRWatcher,
	newWatcher func() (promCRWatcher, error),
	recheck <-chan time.Time,
	eventChan chan allocatorWatcher.Event,
	errChan chan error,
	stop <-chan struct{},
) error {
	for {
		watchErr := make(chan error, 1)
		go func(w promCRWatcher) {
			watchErr <- w.Watch(eventChan, errChan)
		}(watcher)

	wait:
		for {
			select {
			case err := <-watchErr:
				return err
			case <-stop:
				if err := watcher.Close(); err != nil {
					setupLog.Error(err, "prometheus watcher failed to close")
				}
				if err := <-watchErr; err != nil {
					setupLog.Error(err, "prometheus watcher exited with an error")
				}
				return nil
			case <-recheck:
				available, err := watcher.MissingCRDsAvailable()
				if err != nil {
					setupLog.Error(err, "Unable to check availability of Prometheus CRDs")
					continue
				}
				if !available {
					continue
				}
				setupLog.Info("Prometheus CRDs absent at startup are now available, recreating the prometheus watcher")
				replacement, err := newWatcher()
				if err != nil {
					setupLog.Error(err, "Unable to recreate the prometheus watcher")
					continue
				}
				if err := watcher.Close(); err != nil {
					setupLog.Error(err, "prometheus watcher failed to close")
				}
				if err := <-watchErr; err != nil {
					setupLog.Error(err, "prometheus watcher exited with an error")
				}
				watcher = replacement
				break wait
			}
		}
	}
}

func main() {
	var (
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

	metricsGatherer, shutdownMeterProvider, mpErr := telemetry.SetupMeterProvider(ctx, cfg)
	if mpErr != nil {
		setupLog.Error(mpErr, "Unable to set up metrics")
		os.Exit(1)
	}
	defer func() {
		// Flush and close exporters (notably the OTLP reader) on shutdown so the final
		// batch of metrics is delivered.
		if shutdownErr := shutdownMeterProvider(context.Background()); shutdownErr != nil {
			setupLog.Error(shutdownErr, "Error shutting down meter provider")
		}
	}()

	strategyConfig := allocation.StrategyConfig{
		ConsistentHashing: allocation.ConsistentHashingStrategyConfig(cfg.AllocationStrategyConfig.ConsistentHashing),
		LeastWeighted:     allocation.LeastWeightedStrategyConfig(cfg.AllocationStrategyConfig.LeastWeighted),
	}
	if fallback := cfg.GetTargetAllocatorFallbackStrategy(); fallback != nil {
		strategyConfig.PerNode.FallbackStrategy = &allocation.FallbackStrategyConfig{
			Name:              fallback.Name,
			ConsistentHashing: allocation.ConsistentHashingStrategyConfig(fallback.ConsistentHashing),
			LeastWeighted:     allocation.LeastWeightedStrategyConfig(fallback.LeastWeighted),
		}
	}
	allocator, allocErr := allocation.New(cfg.AllocationStrategy, log, allocation.WithStrategyConfig(strategyConfig))
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
	// Serve /metrics from the union of the default registry (Prometheus service discovery,
	// Go runtime and process collectors) and the dedicated SDK registry (OTel SDK metrics),
	// as assembled by setupMeterProvider.
	httpOptions = append(httpOptions, server.WithMetricsGatherer(metricsGatherer))
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

	targetDiscoverer, targetErr := target.NewDiscoverer(log, discoveryManager, cfg.FilterStrategy, srv, allocator.SetTargets)
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
		newPromWatcher := func() (promCRWatcher, error) {
			promWatcher, watcherErr := allocatorWatcher.NewPrometheusCRWatcher(
				ctx, setupLog.WithName("prometheus-cr-watcher"), k8sClient, monitoringClient, *cfg)
			if watcherErr != nil {
				return nil, fmt.Errorf("can't start the prometheus watcher: %w", watcherErr)
			}
			// apply the initial configuration
			promConfig, watcherErr := promWatcher.LoadConfig(ctx)
			if watcherErr != nil {
				return nil, fmt.Errorf("can't load initial Prometheus configuration from Prometheus CRs: %w", watcherErr)
			}
			watcherErr = targetDiscoverer.ApplyConfig(allocatorWatcher.EventSourcePrometheusCR, promConfig.ScrapeConfigs)
			if watcherErr != nil {
				return nil, fmt.Errorf("can't load initial scrape targets from Prometheus CRs: %w", watcherErr)
			}
			return promWatcher, nil
		}
		promWatcher, promWatcherErr := newPromWatcher()
		if promWatcherErr != nil {
			setupLog.Error(promWatcherErr, "Can't start the prometheus watcher")
			os.Exit(1)
		}
		promWatcherStop := make(chan struct{})
		runGroup.Add(
			func() error {
				ticker := time.NewTicker(crdRecheckInterval)
				defer ticker.Stop()
				runErr := runPromWatcher(promWatcher, newPromWatcher, ticker.C, eventChan, errChan, promWatcherStop)
				setupLog.Info("Prometheus watcher exited")
				return runErr
			},
			func(_ error) {
				setupLog.Info("Closing prometheus watcher")
				close(promWatcherStop)
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
