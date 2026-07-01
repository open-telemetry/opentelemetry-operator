// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/oklog/run"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/discovery"
	prometheusbridge "go.opentelemetry.io/contrib/bridges/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
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

	// The OTel SDK metrics are exported to Prometheus through a dedicated registry
	// rather than the default one. This keeps them separate from the metrics that are
	// registered directly on the default Prometheus registry (Prometheus service
	// discovery internals, Go runtime and process collectors). That separation lets the
	// OTLP reader below pull those Prometheus-only metrics via the Prometheus bridge
	// without double-counting the SDK metrics, which it already collects natively.
	sdkRegistry := prometheus.NewRegistry()
	metricExporter, promErr := otelprom.New(otelprom.WithRegisterer(sdkRegistry))
	if promErr != nil {
		panic(promErr)
	}
	meterProviderOpts := []sdkmetric.Option{sdkmetric.WithReader(metricExporter)}

	if cfg.Telemetry.Metrics.OTLP != nil {
		otlpReader, otlpErr := newOTLPMetricReader(ctx, cfg.Telemetry.Metrics.OTLP)
		if otlpErr != nil {
			setupLog.Error(otlpErr, "Failed to create OTLP metric reader")
			os.Exit(1)
		}
		telemetryResource, resErr := telemetryResource(ctx)
		if resErr != nil {
			setupLog.Error(resErr, "Failed to build telemetry resource")
			os.Exit(1)
		}
		meterProviderOpts = append(meterProviderOpts, sdkmetric.WithReader(otlpReader), sdkmetric.WithResource(telemetryResource))
	}

	meterProvider := sdkmetric.NewMeterProvider(meterProviderOpts...)
	otel.SetMeterProvider(meterProvider)
	defer func() {
		// Flush and close exporters (notably the OTLP reader) on shutdown so the final
		// batch of metrics is delivered.
		if shutdownErr := meterProvider.Shutdown(context.Background()); shutdownErr != nil {
			setupLog.Error(shutdownErr, "Error shutting down meter provider")
		}
	}()

	allocatorPrehook = prehook.New(cfg.FilterStrategy, log)
	allocator, allocErr := allocation.New(cfg.AllocationStrategy, log, allocation.WithFilter(allocatorPrehook), allocation.WithFallbackStrategy(cfg.AllocationFallbackStrategy))
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
	// Serve /metrics from both the default registry (Prometheus service discovery, Go
	// runtime and process collectors) and the dedicated SDK registry (OTel SDK metrics),
	// preserving the full metric set that used to be exposed via the default registry alone.
	httpOptions = append(httpOptions, server.WithMetricsGatherer(prometheus.Gatherers{prometheus.DefaultGatherer, sdkRegistry}))
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

// envRefPattern matches ${env:VAR} references, mirroring the OTel Collector's
// environment-variable substitution syntax.
var envRefPattern = regexp.MustCompile(`\$\{env:([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// expandEnvRefs replaces ${env:VAR} occurrences in s with the value of the
// environment variable VAR (empty if unset). This lets sensitive header values
// (e.g. an API token) be sourced from the pod environment / a Secret at runtime
// rather than being written in plaintext into the Target Allocator ConfigMap.
func expandEnvRefs(s string) string {
	return envRefPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := envRefPattern.FindStringSubmatch(match)[1]
		return os.Getenv(name)
	})
}

// expandHeaders returns a copy of headers with ${env:VAR} references expanded in
// every value.
func expandHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	expanded := make(map[string]string, len(headers))
	for k, v := range headers {
		expanded[k] = expandEnvRefs(v)
	}
	return expanded
}

// newOTLPMetricReader creates a PeriodicExportingMetricReader backed by either
// an OTLP/gRPC or OTLP/HTTP exporter depending on cfg.Protocol.
func newOTLPMetricReader(ctx context.Context, cfg *config.OTLPExporterConfig) (sdkmetric.Reader, error) {
	temporality := temporalitySelector(cfg.Temporality)
	readerOpts := periodicReaderOptions(cfg)
	endpoint := expandEnvRefs(cfg.Endpoint)
	headers := expandHeaders(cfg.Headers)
	switch cfg.Protocol {
	case "http":
		// WithEndpointURL does not append /v1/metrics automatically. We do it here
		// so the endpoint convention matches the OTel collector's otlphttp exporter
		// (where users specify the base URL, not the full signal path).
		// Skip if the caller already included the signal path.
		endpoint = strings.TrimRight(endpoint, "/")
		if !strings.HasSuffix(endpoint, "/v1/metrics") {
			endpoint += "/v1/metrics"
		}
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpointURL(endpoint),
			otlpmetrichttp.WithTemporalitySelector(temporality),
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(headers))
		}
		exp, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, readerOpts...), nil
	default: // "grpc"
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithTemporalitySelector(temporality),
		}
		// Accept both a full URL (with scheme, consistent with the HTTP protocol and
		// the collector's otlp exporter) and a bare host:port. WithEndpointURL requires
		// a scheme; WithEndpoint expects host:port.
		if strings.Contains(endpoint, "://") {
			opts = append(opts, otlpmetricgrpc.WithEndpointURL(endpoint))
		} else {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(endpoint))
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(headers))
		}
		exp, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, readerOpts...), nil
	}
}

// periodicReaderOptions builds PeriodicReaderOptions from the export interval and timeout config.
func periodicReaderOptions(cfg *config.OTLPExporterConfig) []sdkmetric.PeriodicReaderOption {
	opts := []sdkmetric.PeriodicReaderOption{
		// Bridge the metrics registered directly on the default Prometheus registry
		// (Prometheus service discovery internals, Go runtime and process collectors)
		// into the OTLP export. Without this, those metrics would be visible on the
		// Prometheus /metrics endpoint but missing from OTLP. The SDK's own metrics live
		// on a separate registry and are collected natively by this reader, so bridging
		// the default registry does not double-count them.
		sdkmetric.WithProducer(prometheusbridge.NewMetricProducer()),
	}
	if cfg.ExportInterval > 0 {
		opts = append(opts, sdkmetric.WithInterval(cfg.ExportInterval))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, sdkmetric.WithTimeout(cfg.Timeout))
	}
	return opts
}

// telemetryResource builds the OpenTelemetry resource attached to exported metrics.
// It sets a default service.name and honors the standard OTEL_SERVICE_NAME /
// OTEL_RESOURCE_ATTRIBUTES environment variables, so additional attributes (e.g. k8s.*)
// can be injected without code changes.
func telemetryResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName("target-allocator")),
		// Applied last so environment configuration wins on conflict.
		resource.WithFromEnv(),
	)
}

// temporalitySelector maps a config string to an SDK TemporalitySelector.
// "delta" → all instruments delta; "lowmemory" → delta for counters/histograms,
// cumulative for gauges; anything else → cumulative (SDK default).
func temporalitySelector(t string) sdkmetric.TemporalitySelector {
	switch t {
	case "delta":
		return sdkmetric.DeltaTemporalitySelector
	case "lowmemory":
		return sdkmetric.LowMemoryTemporalitySelector
	default:
		return sdkmetric.DefaultTemporalitySelector
	}
}
