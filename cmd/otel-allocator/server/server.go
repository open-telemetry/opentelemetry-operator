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
	"fmt"
	"net/http"
	"net/http/pprof"
	"net/url"
	"strings"
	"sync"
	"time"

	yaml2 "github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	jsoniter "github.com/json-iterator/go"
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

var (
	jsonConfig = jsoniter.Config{
		EscapeHTML:                    false,
		MarshalFloatWith6Digits:       true,
		ObjectFieldMustBeSimpleString: true,
	}.Froze()
)

type collectorJSON struct {
	Link string         `json:"_link"`
	Jobs []*target.Item `json:"targets"`
}

type Server struct {
	logger         logr.Logger
	allocator      allocation.Allocator
	server         *http.Server
	jsonMarshaller jsoniter.API

	// Use RWMutex to protect scrapeConfigResponse, since it
	// will be predominantly read and only written when config
	// is applied.
	mtx                  sync.RWMutex
	scrapeConfigResponse []byte
}

func NewServer(log logr.Logger, allocator allocation.Allocator, listenAddr string) *Server {
	s := &Server{
		logger:         log,
		allocator:      allocator,
		jsonMarshaller: jsonConfig,
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.UseRawPath = true
	router.UnescapePathValues = false
	router.Use(s.PrometheusMiddleware)
	router.GET("/scrape_configs", s.ScrapeConfigsHandler)
	router.GET("/jobs", s.JobHandler)
	router.GET("/jobs/:job_id/targets", s.TargetsHandler)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/livez", s.LivenessProbeHandler)
	router.GET("/readyz", s.ReadinessProbeHandler)
	registerPprof(router.Group("/debug/pprof/"))

	s.server = &http.Server{Addr: listenAddr, Handler: router, ReadHeaderTimeout: 90 * time.Second}
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

// UpdateScrapeConfigResponse updates the scrape config response. The target allocator first marshals these
// configurations such that the underlying prometheus marshaling is used. After that, the YAML is converted
// in to a JSON format for consumers to use.
func (s *Server) UpdateScrapeConfigResponse(configs map[string]*promconfig.ScrapeConfig) error {
	var configBytes []byte
	configBytes, err := yaml.Marshal(configs)
	if err != nil {
		return err
	}
	var jsonConfig []byte
	jsonConfig, err = yaml2.YAMLToJSON(configBytes)
	if err != nil {
		return err
	}
	s.mtx.Lock()
	s.scrapeConfigResponse = jsonConfig
	s.mtx.Unlock()
	return nil
}

// ScrapeConfigsHandler returns the available scrape configuration discovered by the target allocator.
func (s *Server) ScrapeConfigsHandler(c *gin.Context) {
	s.mtx.RLock()
	result := s.scrapeConfigResponse
	s.mtx.RUnlock()

	// We don't use the jsonHandler method because we don't want our bytes to be re-encoded
	c.Writer.Header().Set("Content-Type", "application/json")
	_, err := c.Writer.Write(result)
	if err != nil {
		s.errorHandler(c.Writer, err)
	}
}

func (s *Server) ReadinessProbeHandler(c *gin.Context) {
	s.mtx.RLock()
	result := s.scrapeConfigResponse
	s.mtx.RUnlock()

	if result != nil {
		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusServiceUnavailable)
	}
}

func (s *Server) JobHandler(c *gin.Context) {
	displayData := make(map[string]target.LinkJSON)
	for _, v := range s.allocator.TargetItems() {
		displayData[v.JobName] = target.LinkJSON{Link: v.Link.Link}
	}
	s.jsonHandler(c.Writer, displayData)
}

func (s *Server) LivenessProbeHandler(c *gin.Context) {
	c.Status(http.StatusOK)
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
	err := s.jsonMarshaller.NewEncoder(w).Encode(data)
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

// registerPprof registers the pprof handlers and either serves the requested
// specific profile or falls back to index handler.
func registerPprof(g *gin.RouterGroup) {
	g.GET("/*profile", func(c *gin.Context) {
		path := c.Param("profile")
		switch strings.TrimPrefix(path, "/") {
		case "cmdline":
			gin.WrapF(pprof.Cmdline)(c)
		case "profile":
			gin.WrapF(pprof.Profile)(c)
		case "symbol":
			gin.WrapF(pprof.Symbol)(c)
		case "trace":
			gin.WrapF(pprof.Trace)(c)
		default:
			gin.WrapF(pprof.Index)(c)
		}
	})
}
