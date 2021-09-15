package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
	gokitlog "github.com/go-kit/log"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/otel-allocator/allocation"
	"github.com/otel-allocator/collector"
	"github.com/otel-allocator/config"
	lbdiscovery "github.com/otel-allocator/discovery"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	// Trying zap logger
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	var listenAddr string
	var configDir string
	pflag.StringVar(&listenAddr, "listen-addr", ":8080", "The address where this service serves.")
	pflag.StringVar(&configDir, "config-dir", "/conf/", "The directory for the config file.")
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	logger.Info("Starting the Target Allocator")

	//

	ctx := context.Background()

	// watcher to monitor file changes in ConfigMap
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		setupLog.Error(err, "Can't start the watcher")
		os.Exit(1)
	}
	defer watcher.Close()

	if err := watcher.Add(configDir); err != nil {
		setupLog.Error(err, "Can't add directory to watcher")
	}
	log := ctrl.Log.WithName("allocator")
	srv, err := newServer(log, listenAddr)
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
			switch event.Op {
			case fsnotify.Create:
				setupLog.Info("ConfigMap updated!")
				// Restart the server to pickup the new config.
				if err := srv.Shutdown(ctx); err != nil {
					setupLog.Error(err, "Cannot shutdown the server")
				}
				srv, err = newServer(log, listenAddr)
				if err != nil {
					setupLog.Error(err, "Error restarting the server with new config")
				}
				go func() {
					if err := srv.Start(); err != http.ErrServerClosed {
						setupLog.Error(err, "Can't restart the server")
					}
				}()
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

func newServer(log logr.Logger, addr string) (*server, error) {
	allocator, discoveryManager, k8sclient, err := newAllocator(log, context.Background())
	if err != nil {
		return nil, err
	}
	s := &server{
		logger:           log,
		allocator:        allocator,
		discoveryManager: discoveryManager,
		k8sClient:        k8sclient,
	}
	router := mux.NewRouter()
	router.HandleFunc("/jobs", s.JobHandler).Methods("GET")
	router.HandleFunc("/jobs/{job_id}/targets", s.TargetsHandler).Methods("GET")
	s.server = &http.Server{Addr: addr, Handler: router}
	return s, nil
}

func newAllocator(log logr.Logger, ctx context.Context) (*allocation.Allocator, *lbdiscovery.Manager, *collector.Client, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, nil, err
	}

	k8sClient, err := collector.NewClient(log)
	if err != nil {
		return nil, nil, nil, err
	}

	// creates a new discovery manager
	discoveryManager := lbdiscovery.NewManager(log, ctx, gokitlog.NewNopLogger())

	// returns the list of targets
	if err := discoveryManager.ApplyConfig(cfg); err != nil {
		return nil, nil, nil, err
	}

	allocator := allocation.NewAllocator(log)
	discoveryManager.Watch(func(targets []allocation.TargetItem) {
		allocator.SetWaitingTargets(targets)
		allocator.AllocateTargets()
	})
	k8sClient.Watch(ctx, cfg.LabelSelector, func(collectors []string) {
		allocator.SetCollectors(collectors)
		allocator.ReallocateCollectors()
	})
	return allocator, discoveryManager, k8sClient, nil
}

func (s *server) Start() error {
	setupLog.Info("Starting server...")
	return s.server.ListenAndServe()
}

func (s *server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")
	s.k8sClient.Close()
	s.discoveryManager.Close()
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

	if len(q) == 0 {
		displayData := allocation.GetAllTargetsByJob(params["job_id"], compareMap, s.allocator)
		jsonHandler(w, r, displayData)

	} else {
		tgs := allocation.GetAllTargetsByCollectorAndJob(q[0], params["job_id"], compareMap, s.allocator)
		// Displays empty list if nothing matches
		if len(tgs) == 0 {
			jsonHandler(w, r, []interface{}{})
			return
		}
		jsonHandler(w, r, tgs)
	}
}

func jsonHandler(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
