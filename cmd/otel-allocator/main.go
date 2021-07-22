package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
	gokitlog "github.com/go-kit/log"
	"github.com/otel-allocator/allocation"
	"github.com/otel-allocator/collector"
	"github.com/otel-allocator/config"
	lbdiscovery "github.com/otel-allocator/discovery"

	"github.com/gorilla/mux"
)

const (
	configDir  = "/conf/"
	listenAddr = ":443"
)

func main() {
	ctx := context.Background()

	// watcher to monitor file changes in ConfigMap
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Can't start the watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(configDir); err != nil {
		log.Fatalf("Can't add directory to watcher: %v", err)
	}

	srv, err := newServer(listenAddr)
	if err != nil {
		log.Fatalf("Can't start the server: %v", err)
	}

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != http.ErrServerClosed {
			log.Fatalf("Can't start the server: %v", err)
		}
	}()

	for {
		select {
		case <-interrupts:
			if err := srv.Shutdown(ctx); err != nil {
				log.Println(err)
				os.Exit(1)
			}
			os.Exit(0)
		case event := <-watcher.Events:
			switch event.Op {
			case fsnotify.Create:
				log.Println("ConfigMap updated!")
				// Restart the server to pickup the new config.
				if err := srv.Shutdown(ctx); err != nil {
					log.Fatalf("Cannot shutdown the server: %v", err)
				}
				srv, err = newServer(listenAddr)
				if err != nil {
					log.Fatalf("Error restarting the server with new config: %v", err)
				}
				go func() {
					if err := srv.Start(); err != http.ErrServerClosed {
						log.Fatalf("Can't restart the server: %v", err)
					}
				}()
			}
		case err := <-watcher.Errors:
			log.Printf("Watcher error: %v", err)
		}
	}
}

type server struct {
	allocator        *allocation.Allocator
	discoveryManager *lbdiscovery.Manager
	server           *http.Server
}

func newServer(addr string) (*server, error) {
	allocator, discoveryManager, err := newAllocator(context.Background())
	if err != nil {
		return nil, err
	}
	s := &server{
		allocator:        allocator,
		discoveryManager: discoveryManager,
	}
	router := mux.NewRouter()
	router.HandleFunc("/jobs", allocator.JobHandler).Methods("GET")
	router.HandleFunc("/jobs/{job_id}/targets", allocator.TargetsHandler).Methods("GET")
	s.server = &http.Server{Addr: addr, Handler: router}
	return s, nil
}

func newAllocator(ctx context.Context) (*allocation.Allocator, *lbdiscovery.Manager, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, err
	}

	k8sClient, err := collector.NewClient()
	if err != nil {
		return nil, nil, err
	}

	// creates a new discovery manager
	discoveryManager := lbdiscovery.NewManager(ctx, gokitlog.NewNopLogger())

	// returns the list of targets
	if err := discoveryManager.ApplyConfig(cfg); err != nil {
		return nil, nil, err
	}

	allocator := allocation.NewAllocator()
	discoveryManager.Watch(func(targets []allocation.TargetItem) {
		allocator.SetTargets(targets)
		allocator.Reallocate()
	})
	k8sClient.Watch(ctx, cfg.LabelSelector, func(collectors []string) {
		allocator.SetCollectors(collectors)
		allocator.ReallocateCollectors()
	})
	return allocator, discoveryManager, nil
}

func (s *server) Start() error {
	log.Println("Starting server...")
	return s.server.ListenAndServe()
}

func (s *server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	s.discoveryManager.Close()
	return s.server.Shutdown(ctx)
}
