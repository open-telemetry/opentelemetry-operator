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
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ghodss/yaml"
	gokitlog "github.com/go-kit/log"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	yaml2 "gopkg.in/yaml.v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/collector"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
	lbdiscovery "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/discovery"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/watcher"
)

var (
	setupLog     = ctrl.Log.WithName("setup")
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "opentelemetry_allocator_http_duration_seconds",
		Help: "Duration of received HTTP requests.",
	}, []string{"path"})
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

	// allocatorPrehook will be nil if filterStrategy is not set or
	// unrecognized. No filtering will be used in this case.
	allocatorPrehook := prehook.New(cfg.GetTargetsFilterStrategy(), log)

	allocator, err := allocation.New(cfg.GetAllocationStrategy(), log, allocation.WithFilter(allocatorPrehook))
	if err != nil {
		setupLog.Error(err, "Unable to initialize allocation strategy")
		os.Exit(1)
	}

	watcher, err := allocatorWatcher.NewWatcher(setupLog, cliConf, allocator)
	if err != nil {
		setupLog.Error(err, "Can't start the watchers")
		os.Exit(1)
	}
	defer func() {
		err := watcher.Close()
		if err != nil {
			log.Error(err, "failed to close watcher")
		}
	}()

	// creates a new discovery manager
	discoveryManager := lbdiscovery.NewManager(log, ctx, gokitlog.NewNopLogger(), allocatorPrehook)
	defer discoveryManager.Close()

	discoveryManager.Watch(allocator.SetTargets)

	k8sclient, err := configureFileDiscovery(log, allocator, discoveryManager, context.Background(), cliConf)
	if err != nil {
		setupLog.Error(err, "Can't start the k8s client")
		os.Exit(1)
	}

	srv := newServer(log, allocator, discoveryManager, k8sclient, cliConf.ListenAddr)

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
				srv = newServer(log, allocator, discoveryManager, k8sclient, cliConf.ListenAddr)
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

type server struct {
	logger           logr.Logger
	allocator        allocation.Allocator
	discoveryManager *lbdiscovery.Manager
	k8sClient        *collector.Client
	server           *http.Server
}

func newServer(log logr.Logger, allocator allocation.Allocator, discoveryManager *lbdiscovery.Manager, k8sclient *collector.Client, listenAddr *string) *server {
	s := &server{
		logger:           log,
		allocator:        allocator,
		discoveryManager: discoveryManager,
		k8sClient:        k8sclient,
	}
	router := mux.NewRouter().UseEncodedPath()
	router.Use(s.PrometheusMiddleware)
	router.HandleFunc("/scrape_configs", s.ScrapeConfigsHandler).Methods("GET")
	router.HandleFunc("/jobs", s.JobHandler).Methods("GET")
	router.HandleFunc("/jobs/{job_id}/targets", s.TargetsHandler).Methods("GET")
	router.Path("/metrics").Handler(promhttp.Handler())
	s.server = &http.Server{Addr: *listenAddr, Handler: router, ReadHeaderTimeout: 90 * time.Second}
	return s
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

func (s *server) Start() error {
	setupLog.Info("Starting server...")
	return s.server.ListenAndServe()
}

func (s *server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")
	s.k8sClient.Close()
	return s.server.Shutdown(ctx)
}

// ScrapeConfigsHandler returns the available scrape configuration discovered by the target allocator.
// The target allocator first marshals these configurations such that the underlying prometheus marshaling is used.
// After that, the YAML is converted in to a JSON format for consumers to use.
func (s *server) ScrapeConfigsHandler(w http.ResponseWriter, r *http.Request) {
	configs := s.discoveryManager.GetScrapeConfigs()
	configBytes, err := yaml2.Marshal(configs)
	if err != nil {
		s.errorHandler(w, err)
	}
	jsonConfig, err := yaml.YAMLToJSON(configBytes)
	if err != nil {
		s.errorHandler(w, err)
	}
	// We don't use the jsonHandler method because we don't want our bytes to be re-encoded
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonConfig)
	if err != nil {
		s.errorHandler(w, err)
	}
}

func (s *server) JobHandler(w http.ResponseWriter, r *http.Request) {
	displayData := make(map[string]target.LinkJSON)
	for _, v := range s.allocator.TargetItems() {
		displayData[v.JobName] = target.LinkJSON{Link: v.Link.Link}
	}
	s.jsonHandler(w, displayData)
}

// PrometheusMiddleware implements mux.MiddlewareFunc.
func (s *server) PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
	})
}

func (s *server) TargetsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()["collector_id"]

	var compareMap = make(map[string][]target.Item) // CollectorName+jobName -> TargetItem
	for _, v := range s.allocator.TargetItems() {
		compareMap[v.CollectorName+v.JobName] = append(compareMap[v.CollectorName+v.JobName], *v)
	}
	params := mux.Vars(r)
	jobId, err := url.QueryUnescape(params["job_id"])
	if err != nil {
		s.errorHandler(w, err)
		return
	}

	if len(q) == 0 {
		displayData := allocation.GetAllTargetsByJob(jobId, compareMap, s.allocator)
		s.jsonHandler(w, displayData)

	} else {
		tgs := allocation.GetAllTargetsByCollectorAndJob(q[0], jobId, compareMap, s.allocator)
		// Displays empty list if nothing matches
		if len(tgs) == 0 {
			s.jsonHandler(w, []interface{}{})
			return
		}
		s.jsonHandler(w, tgs)
	}
}

func (s *server) errorHandler(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	s.jsonHandler(w, err)
}

func (s *server) jsonHandler(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.logger.Error(err, "failed to encode data for http response")
	}
}
