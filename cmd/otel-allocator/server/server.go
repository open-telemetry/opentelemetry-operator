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

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"net/url"
	"time"

	yaml2 "github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/mitchellh/hashstructure"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promconfig "github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "opentelemetry_allocator_http_duration_seconds",
		Help: "Duration of received HTTP requests.",
	}, []string{"path"})
)

type collectorJSON struct {
	Link string         `json:"_link"`
	Jobs []*target.Item `json:"targets"`
}

type DiscoveryManager interface {
	GetScrapeConfigs() map[string]*promconfig.ScrapeConfig
}

type Server struct {
	logger           logr.Logger
	allocator        allocation.Allocator
	discoveryManager DiscoveryManager
	server           *http.Server

	compareHash          uint64
	scrapeConfigResponse []byte
}

func NewServer(log logr.Logger, allocator allocation.Allocator, discoveryManager DiscoveryManager, listenAddr *string) *Server {
	s := &Server{
		logger:           log,
		allocator:        allocator,
		discoveryManager: discoveryManager,
		compareHash:      uint64(0),
	}

	router := gin.Default()
	router.UseRawPath = true
	router.UnescapePathValues = false
	router.Use(s.PrometheusMiddleware)
	router.GET("/scrape_configs", s.ScrapeConfigsHandler)
	router.GET("/jobs", s.JobHandler)
	router.GET("/jobs/:job_id/targets", s.TargetsHandler)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	registerPprof(router.Group("/debug/pprof/"))

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
func (s *Server) ScrapeConfigsHandler(c *gin.Context) {
	configs := s.discoveryManager.GetScrapeConfigs()

	hash, err := hashstructure.Hash(configs, nil)
	if err != nil {
		s.logger.Error(err, "failed to hash the config")
		s.errorHandler(c.Writer, err)
		return
	}
	// if the hashes are different, we need to recompute the scrape config
	if hash != s.compareHash {
		var configBytes []byte
		configBytes, err = yaml.Marshal(configs)
		if err != nil {
			s.errorHandler(c.Writer, err)
			return
		}
		var jsonConfig []byte
		jsonConfig, err = yaml2.YAMLToJSON(configBytes)
		if err != nil {
			s.errorHandler(c.Writer, err)
			return
		}
		s.scrapeConfigResponse = jsonConfig
		s.compareHash = hash
	}
	// We don't use the jsonHandler method because we don't want our bytes to be re-encoded
	c.Writer.Header().Set("Content-Type", "application/json")
	_, err = c.Writer.Write(s.scrapeConfigResponse)
	if err != nil {
		s.errorHandler(c.Writer, err)
	}
}

func (s *Server) JobHandler(c *gin.Context) {
	displayData := make(map[string]target.LinkJSON)
	for _, v := range s.allocator.TargetItems() {
		displayData[v.JobName] = target.LinkJSON{Link: v.Link.Link}
	}
	s.jsonHandler(c.Writer, displayData)
}

func (s *Server) PrometheusMiddleware(c *gin.Context) {
	path := c.FullPath()
	timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
	c.Next()
	timer.ObserveDuration()
}

func (s *Server) TargetsHandler(c *gin.Context) {
	q := c.Request.URL.Query()["collector_id"]

	jobIdParam := c.Params.ByName("job_id")
	jobId, err := url.QueryUnescape(jobIdParam)
	if err != nil {
		s.errorHandler(c.Writer, err)
		return
	}

	if len(q) == 0 {
		displayData := GetAllTargetsByJob(s.allocator, jobId)
		s.jsonHandler(c.Writer, displayData)

	} else {
		tgs := s.allocator.GetTargetsForCollectorAndJob(q[0], jobId)
		// Displays empty list if nothing matches
		if len(tgs) == 0 {
			s.jsonHandler(c.Writer, []interface{}{})
			return
		}
		s.jsonHandler(c.Writer, tgs)
	}
}

func (s *Server) errorHandler(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	s.jsonHandler(w, err)
}

func (s *Server) jsonHandler(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		s.logger.Error(err, "failed to encode data for http response")
	}
}

// GetAllTargetsByJob is a relatively expensive call that is usually only used for debugging purposes.
func GetAllTargetsByJob(allocator allocation.Allocator, job string) map[string]collectorJSON {
	displayData := make(map[string]collectorJSON)
	for _, col := range allocator.Collectors() {
		items := allocator.GetTargetsForCollectorAndJob(col.Name, job)
		displayData[col.Name] = collectorJSON{Link: fmt.Sprintf("/jobs/%s/targets?collector_id=%s", url.QueryEscape(job), col.Name), Jobs: items}
	}
	return displayData
}

func registerPprof(g *gin.RouterGroup) {
	g.GET("/", gin.WrapF(pprof.Index))
	g.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	g.GET("/profile", gin.WrapF(pprof.Profile))
	g.GET("/symbol", gin.WrapF(pprof.Symbol))
	g.GET("/trace", gin.WrapF(pprof.Trace))
}
