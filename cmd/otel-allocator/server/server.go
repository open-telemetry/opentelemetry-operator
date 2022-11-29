package server

import (
	"context"
	"encoding/json"
	yaml2 "github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/mitchellh/hashstructure"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
	"net/http"
	"net/url"
	"time"
)

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "opentelemetry_allocator_http_duration_seconds",
		Help: "Duration of received HTTP requests.",
	}, []string{"path"})
)

type Server struct {
	logger           logr.Logger
	allocator        allocation.Allocator
	discoveryManager *target.Discoverer
	server           *http.Server

	compareHash          uint64
	scrapeConfigResponse []byte
}

func NewServer(log logr.Logger, allocator allocation.Allocator, discoveryManager *target.Discoverer, listenAddr *string) *Server {
	s := &Server{
		logger:           log,
		allocator:        allocator,
		discoveryManager: discoveryManager,
		compareHash:      uint64(0),
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

func (s *Server) Start() error {
	s.logger.Info("Starting server...")
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")
	return s.server.Shutdown(ctx)
}

// ScrapeConfigsHandler returns the available scrape configuration discovered by the target allocator.
// The target allocator first marshals these configurations such that the underlying prometheus marshaling is used.
// After that, the YAML is converted in to a JSON format for consumers to use.
func (s *Server) ScrapeConfigsHandler(w http.ResponseWriter, r *http.Request) {
	configs := s.discoveryManager.GetScrapeConfigs()

	hash, err := hashstructure.Hash(configs, nil)
	if err != nil {
		s.logger.Error(err, "failed to hash the config")
		s.errorHandler(w, err)
		return
	}
	// if the hashes are different, we need to recompute the scrape config
	if hash != s.compareHash {
		configBytes, err := yaml.Marshal(configs)
		if err != nil {
			s.errorHandler(w, err)
			return
		}
		jsonConfig, err := yaml2.YAMLToJSON(configBytes)
		if err != nil {
			s.errorHandler(w, err)
			return
		}
		s.scrapeConfigResponse = jsonConfig
	}
	// We don't use the jsonHandler method because we don't want our bytes to be re-encoded
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(s.scrapeConfigResponse)
	if err != nil {
		s.errorHandler(w, err)
	}
}

func (s *Server) JobHandler(w http.ResponseWriter, r *http.Request) {
	displayData := make(map[string]target.LinkJSON)
	for _, v := range s.allocator.TargetItems() {
		displayData[v.JobName] = target.LinkJSON{Link: v.Link.Link}
	}
	s.jsonHandler(w, displayData)
}

// PrometheusMiddleware implements mux.MiddlewareFunc.
func (s *Server) PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
	})
}

func (s *Server) TargetsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()["collector_id"]

	params := mux.Vars(r)
	jobId, err := url.QueryUnescape(params["job_id"])
	if err != nil {
		s.errorHandler(w, err)
		return
	}

	if len(q) == 0 {
		displayData := allocation.GetAllTargetsByJob(s.allocator, jobId)
		s.jsonHandler(w, displayData)

	} else {
		tgs := allocation.GetAllTargetsByCollectorAndJob(s.allocator, q[0], jobId)
		// Displays empty list if nothing matches
		if len(tgs) == 0 {
			s.jsonHandler(w, []interface{}{})
			return
		}
		s.jsonHandler(w, tgs)
	}
}

func (s *Server) errorHandler(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	s.jsonHandler(w, err)
}

func (s *Server) jsonHandler(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.logger.Error(err, "failed to encode data for http response")
	}
}
