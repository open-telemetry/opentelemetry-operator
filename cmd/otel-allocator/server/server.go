// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"crypto/tls"
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
	"github.com/goccy/go-json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promcommconfig "github.com/prometheus/common/config"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/labels"
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
	Link string        `json:"_link"`
	Jobs []*targetJSON `json:"targets"`
}

type linkJSON struct {
	Link string `json:"_link"`
}

type targetJSON struct {
	TargetURL []string      `json:"targets"`
	Labels    labels.Labels `json:"labels"`
}

type Server struct {
	logger      logr.Logger
	allocator   allocation.Allocator
	server      *http.Server
	httpsServer *http.Server

	// Use RWMutex to protect scrapeConfigResponse, since it
	// will be predominantly read and only written when config
	// is applied.
	mtx                                  sync.RWMutex
	scrapeConfigResponse                 []byte
	ScrapeConfigMarshalledSecretResponse []byte
}

type Option func(*Server)

// Option to create an additional https server with mTLS configuration.
// Used for getting the scrape config with real secret values.
func WithTLSConfig(tlsConfig *tls.Config, httpsListenAddr string) Option {
	return func(s *Server) {
		httpsRouter := gin.New()
		s.setRouter(httpsRouter)

		s.httpsServer = &http.Server{Addr: httpsListenAddr, Handler: httpsRouter, ReadHeaderTimeout: 90 * time.Second, TLSConfig: tlsConfig}
	}
}

func (s *Server) setRouter(router *gin.Engine) {
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
}

func NewServer(log logr.Logger, allocator allocation.Allocator, listenAddr string, options ...Option) *Server {
	s := &Server{
		logger:    log,
		allocator: allocator,
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	s.setRouter(router)

	s.server = &http.Server{Addr: listenAddr, Handler: router, ReadHeaderTimeout: 90 * time.Second}

	for _, opt := range options {
		opt(s)
	}

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

func (s *Server) StartHTTPS() error {
	s.logger.Info("Starting HTTPS server...")
	return s.httpsServer.ListenAndServeTLS("", "")
}

func (s *Server) ShutdownHTTPS(ctx context.Context) error {
	s.logger.Info("Shutting down HTTPS server...")
	return s.httpsServer.Shutdown(ctx)
}

// RemoveRegexFromRelabelAction is needed specifically for keepequal/dropequal actions because even though the user doesn't specify the
// regex field for these actions the unmarshalling implementations of prometheus adds back the default regex fields
// which in turn causes the receiver to error out since the unmarshaling of the json response doesn't expect anything in the regex fields
// for these actions. Adding this as a fix until the original issue with prometheus unmarshaling is fixed -
// https://github.com/prometheus/prometheus/issues/12534
func RemoveRegexFromRelabelAction(jsonConfig []byte) ([]byte, error) {
	var jobToScrapeConfig map[string]interface{}
	err := json.Unmarshal(jsonConfig, &jobToScrapeConfig)
	if err != nil {
		return nil, err
	}
	for _, scrapeConfig := range jobToScrapeConfig {
		scrapeConfig := scrapeConfig.(map[string]interface{})
		if scrapeConfig["relabel_configs"] != nil {
			relabelConfigs := scrapeConfig["relabel_configs"].([]interface{})
			for _, relabelConfig := range relabelConfigs {
				relabelConfig := relabelConfig.(map[string]interface{})
				// Dropping regex key from the map since unmarshalling this on the client(metrics_receiver.go) results in error
				// because of the bug here - https://github.com/prometheus/prometheus/issues/12534
				if relabelConfig["action"] == "keepequal" || relabelConfig["action"] == "dropequal" {
					delete(relabelConfig, "regex")
				}
			}
		}
		if scrapeConfig["metric_relabel_configs"] != nil {
			metricRelabelConfigs := scrapeConfig["metric_relabel_configs"].([]interface{})
			for _, metricRelabelConfig := range metricRelabelConfigs {
				metricRelabelConfig := metricRelabelConfig.(map[string]interface{})
				// Dropping regex key from the map since unmarshalling this on the client(metrics_receiver.go) results in error
				// because of the bug here - https://github.com/prometheus/prometheus/issues/12534
				if metricRelabelConfig["action"] == "keepequal" || metricRelabelConfig["action"] == "dropequal" {
					delete(metricRelabelConfig, "regex")
				}
			}
		}
	}

	jsonConfigNew, err := json.Marshal(jobToScrapeConfig)
	if err != nil {
		return nil, err
	}
	return jsonConfigNew, nil
}

func (s *Server) MarshalScrapeConfig(configs map[string]*promconfig.ScrapeConfig, marshalSecretValue bool) error {
	promcommconfig.MarshalSecretValue = marshalSecretValue

	configBytes, err := yaml.Marshal(configs)
	if err != nil {
		return err
	}

	var jsonConfig []byte
	jsonConfig, err = yaml2.YAMLToJSON(configBytes)
	if err != nil {
		return err
	}

	jsonConfigNew, err := RemoveRegexFromRelabelAction(jsonConfig)
	if err != nil {
		return err
	}

	s.mtx.Lock()
	if marshalSecretValue {
		s.ScrapeConfigMarshalledSecretResponse = jsonConfigNew
	} else {
		s.scrapeConfigResponse = jsonConfigNew
	}
	s.mtx.Unlock()

	return nil
}

// UpdateScrapeConfigResponse updates the scrape config response. The target allocator first marshals these
// configurations such that the underlying prometheus marshaling is used. After that, the YAML is converted
// in to a JSON format for consumers to use.
func (s *Server) UpdateScrapeConfigResponse(configs map[string]*promconfig.ScrapeConfig) error {
	err := s.MarshalScrapeConfig(configs, false)
	if err != nil {
		return err
	}
	err = s.MarshalScrapeConfig(configs, true)
	if err != nil {
		return err
	}
	return nil
}

// ScrapeConfigsHandler returns the available scrape configuration discovered by the target allocator.
func (s *Server) ScrapeConfigsHandler(c *gin.Context) {
	s.mtx.RLock()
	result := s.scrapeConfigResponse
	if c.Request.TLS != nil {
		result = s.ScrapeConfigMarshalledSecretResponse
	}
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
	displayData := make(map[string]linkJSON)
	for _, v := range s.allocator.TargetItems() {
		displayData[v.JobName] = linkJSON{Link: fmt.Sprintf("/jobs/%s/targets", url.QueryEscape(v.JobName))}
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
		targets := GetAllTargetsByCollectorAndJob(s.allocator, q[0], jobId)
		// Displays empty list if nothing matches
		if len(targets) == 0 {
			s.jsonHandler(c.Writer, []interface{}{})
			return
		}
		s.jsonHandler(c.Writer, targets)
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
		targets := GetAllTargetsByCollectorAndJob(allocator, col.Name, job)
		displayData[col.Name] = collectorJSON{
			Link: fmt.Sprintf("/jobs/%s/targets?collector_id=%s", url.QueryEscape(job), col.Name),
			Jobs: targets,
		}
	}
	return displayData
}

// GetAllTargetsByCollector returns all the targets for a given collector and job.
func GetAllTargetsByCollectorAndJob(allocator allocation.Allocator, collectorName string, jobName string) []*targetJSON {
	items := allocator.GetTargetsForCollectorAndJob(collectorName, jobName)
	targets := make([]*targetJSON, len(items))
	for i, item := range items {
		targets[i] = targetJsonFromTargetItem(item)
	}
	return targets
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

func targetJsonFromTargetItem(item *target.Item) *targetJSON {
	return &targetJSON{
		TargetURL: []string{item.TargetURL},
		Labels:    item.Labels,
	}
}
