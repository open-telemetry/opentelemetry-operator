package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	gokitlog "github.com/go-kit/log"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/collector"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
	lbdiscovery "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/discovery"
	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/watcher"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	cliConf, err := config.ParseCLI()
	if err != nil {
		setupLog.Error(err, "Failed to parse parameters")
		os.Exit(1)
	}

	cliConf.RootLogger.Info("Starting the Target Allocator")

	ctx := context.Background()

	log := ctrl.Log.WithName("allocator")
	allocator := allocation.NewAllocator(log)
	watcher, err := allocatorWatcher.NewWatcher(setupLog, cliConf, allocator)
	if err != nil {
		setupLog.Error(err, "Can't start the watchers")
		os.Exit(1)
	}
	defer watcher.Close()

	// creates a new discovery manager
	discoveryManager := lbdiscovery.NewManager(log, ctx, gokitlog.NewNopLogger())
	defer discoveryManager.Close()
	discoveryManager.Watch(func(targets []allocation.TargetItem) {
		allocator.SetWaitingTargets(targets)
		allocator.AllocateTargets()
	})

	srv, err := newServer(log, allocator, discoveryManager, cliConf)
	if err != nil {
		setupLog.Error(err, "Can't start the server")
	}

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
			switch event.Source {
			case allocatorWatcher.EventSourceConfigMap:
				setupLog.Info("ConfigMap updated!")
				// Restart the server to pickup the new config.
				if err := srv.Shutdown(ctx); err != nil {
					setupLog.Error(err, "Cannot shutdown the server")
				}
				srv, err = newServer(log, allocator, discoveryManager, cliConf)
				if err != nil {
					setupLog.Error(err, "Error restarting the server with new config")
				}
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
	allocator        *allocation.Allocator
	discoveryManager *lbdiscovery.Manager
	k8sClient        *collector.Client
	server           *http.Server
}

func newServer(log logr.Logger, allocator *allocation.Allocator, discoveryManager *lbdiscovery.Manager, cliConf config.CLIConfig) (*server, error) {
	k8sclient, err := configureFileDiscovery(log, allocator, discoveryManager, context.Background(), cliConf)
	if err != nil {
		return nil, err
	}
	s := &server{
		logger:           log,
		allocator:        allocator,
		discoveryManager: discoveryManager,
		k8sClient:        k8sclient,
	}
	router := mux.NewRouter().UseEncodedPath()
	router.HandleFunc("/jobs", s.JobHandler).Methods("GET")
	router.HandleFunc("/jobs/{job_id}/targets", s.TargetsHandler).Methods("GET")
	s.server = &http.Server{Addr: *cliConf.ListenAddr, Handler: router}
	return s, nil
}

func configureFileDiscovery(log logr.Logger, allocator *allocation.Allocator, discoveryManager *lbdiscovery.Manager, ctx context.Context, cliConfig config.CLIConfig) (*collector.Client, error) {
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

	k8sClient.Watch(ctx, cfg.LabelSelector, func(collectors []string) {
		allocator.SetCollectors(collectors)
		allocator.ReallocateCollectors()
	})
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

func (s *server) JobHandler(w http.ResponseWriter, r *http.Request) {
	displayData := make(map[string]allocation.LinkJSON)
	for _, v := range s.allocator.TargetItems {
		displayData[v.JobName] = allocation.LinkJSON{v.Link.Link}
	}
	jsonHandler(w, r, displayData)
}

func (s *server) TargetsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()["collector_id"]

	var compareMap = make(map[string][]allocation.TargetItem) // CollectorName+jobName -> TargetItem
	for _, v := range s.allocator.TargetItems {
		compareMap[v.Collector.Name+v.JobName] = append(compareMap[v.Collector.Name+v.JobName], *v)
	}
	params := mux.Vars(r)
	jobId, err := url.QueryUnescape(params["job_id"])
	if err != nil {
		errorHandler(err, w, r)
		return
	}

	if len(q) == 0 {
		displayData := allocation.GetAllTargetsByJob(jobId, compareMap, s.allocator)
		jsonHandler(w, r, displayData)

	} else {
		tgs := allocation.GetAllTargetsByCollectorAndJob(q[0], jobId, compareMap, s.allocator)
		// Displays empty list if nothing matches
		if len(tgs) == 0 {
			jsonHandler(w, r, []interface{}{})
			return
		}
		jsonHandler(w, r, tgs)
	}
}

func errorHandler(err error, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(500)
}

func jsonHandler(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
