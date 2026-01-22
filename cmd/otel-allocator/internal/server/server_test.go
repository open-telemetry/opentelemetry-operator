// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"gotest.tools/v3/golden"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
	allocatorconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

var (
	logger                  = logf.Log.WithName("server-unit-tests")
	baseLabelSet            = labels.New(labels.Label{Name: "test_label", Value: "test-value"})
	testJobLabelSetTwo      = labels.New(labels.Label{Name: "test_label", Value: "test-value2"})
	baseTargetItem          = target.NewItem("test-job", "test-url", baseLabelSet, "test-collector")
	secondTargetItem        = target.NewItem("test-job", "test-url", baseLabelSet, "test-collector")
	testJobTargetItemTwo    = target.NewItem("test-job", "test-url2", testJobLabelSetTwo, "test-collector2")
	testJobTwoTargetItemTwo = target.NewItem("test-job2", "test-url3", testJobLabelSetTwo, "test-collector2")
)

func TestServer_LivenessProbeHandler(t *testing.T) {
	leastWeighted, _ := allocation.New("least-weighted", logger)
	listenAddr := ":8080"
	s, err := NewServer(logger, leastWeighted, listenAddr)
	require.NoError(t, err)
	request := httptest.NewRequest("GET", "/livez", nil)
	w := httptest.NewRecorder()

	s.server.Handler.ServeHTTP(w, request)
	result := w.Result()

	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestServer_TargetsHandler(t *testing.T) {
	leastWeighted, _ := allocation.New("least-weighted", logger)
	type args struct {
		collector string
		job       string
		targets   []*target.Item
		allocator allocation.Allocator
	}
	type want struct {
		items     []*targetJSON
		errString string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Empty target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				targets:   []*target.Item{},
				allocator: leastWeighted,
			},
			want: want{
				items: []*targetJSON{},
			},
		},
		{
			name: "Single entry target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				targets: []*target.Item{
					baseTargetItem,
				},
				allocator: leastWeighted,
			},
			want: want{
				items: []*targetJSON{
					{
						TargetURL: []string{"test-url"},
						Labels:    labels.New(labels.Label{Name: "test_label", Value: "test-value"}),
					},
				},
			},
		},
		{
			name: "Multiple entry target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				targets: []*target.Item{
					baseTargetItem,
					secondTargetItem,
				},
				allocator: leastWeighted,
			},
			want: want{
				items: []*targetJSON{
					{
						TargetURL: []string{"test-url"},
						Labels:    labels.New(labels.Label{Name: "test_label", Value: "test-value"}),
					},
				},
			},
		},
		{
			name: "Multiple entry target map of same job with label merge",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				targets: []*target.Item{
					baseTargetItem,
					testJobTargetItemTwo,
				},
				allocator: leastWeighted,
			},
			want: want{
				items: []*targetJSON{
					{
						TargetURL: []string{"test-url"},
						Labels:    labels.New(labels.Label{Name: "test_label", Value: "test-value"}),
					},
					{
						TargetURL: []string{"test-url2"},
						Labels:    labels.New(labels.Label{Name: "test_label", Value: "test-value2"}),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listenAddr := ":8080"
			s, err := NewServer(logger, tt.args.allocator, listenAddr)
			require.NoError(t, err)

			tt.args.allocator.SetCollectors(map[string]*allocation.Collector{"test-collector": {Name: "test-collector"}})
			tt.args.allocator.SetTargets(tt.args.targets)
			request := httptest.NewRequest("GET", fmt.Sprintf("/jobs/%s/targets?collector_id=%s", tt.args.job, tt.args.collector), nil)
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, http.StatusOK, result.StatusCode)
			body := result.Body
			bodyBytes, err := io.ReadAll(body)
			assert.NoError(t, err)
			if len(tt.want.errString) != 0 {
				assert.EqualError(t, err, tt.want.errString)
				return
			}
			var itemResponse []*targetJSON
			err = json.Unmarshal(bodyBytes, &itemResponse)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.want.items, itemResponse)
		})
	}
}

func TestServer_ScrapeConfigsHandler(t *testing.T) {
	svrConfig := allocatorconfig.HTTPSServerConfig{}
	tlsConfig, _ := svrConfig.NewTLSConfig()
	tests := []struct {
		description   string
		scrapeConfigs map[string]*promconfig.ScrapeConfig
		expectedCode  int
		expectedBody  []byte
		serverOptions []Option
	}{
		{
			description:   "nil scrape config",
			scrapeConfigs: nil,
			expectedCode:  http.StatusOK,
			expectedBody:  []byte("{}"),
		},
		{
			description:   "empty scrape config",
			scrapeConfigs: map[string]*promconfig.ScrapeConfig{},
			expectedCode:  http.StatusOK,
			expectedBody:  []byte("{}"),
		},
		{
			description: "single entry",
			scrapeConfigs: map[string]*promconfig.ScrapeConfig{
				"serviceMonitor/testapp/testapp/0": {
					JobName:         "serviceMonitor/testapp/testapp/0",
					HonorTimestamps: true,
					ScrapeInterval:  model.Duration(30 * time.Second),
					ScrapeTimeout:   model.Duration(30 * time.Second),
					MetricsPath:     "/metrics",
					Scheme:          "http",
					HTTPClientConfig: config.HTTPClientConfig{
						FollowRedirects: true,
					},
					RelabelConfigs: []*relabel.Config{
						{
							SourceLabels: model.LabelNames{model.LabelName("job")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "__tmp_prometheus_job_name",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
					},
				},
			},
			expectedCode: http.StatusOK,
		},
		{
			description: "multiple entries",
			scrapeConfigs: map[string]*promconfig.ScrapeConfig{
				"serviceMonitor/testapp/testapp/0": {
					JobName:         "serviceMonitor/testapp/testapp/0",
					HonorTimestamps: true,
					ScrapeInterval:  model.Duration(30 * time.Second),
					ScrapeTimeout:   model.Duration(30 * time.Second),
					MetricsPath:     "/metrics",
					Scheme:          "http",
					HTTPClientConfig: config.HTTPClientConfig{
						FollowRedirects: true,
					},
					RelabelConfigs: []*relabel.Config{
						{
							SourceLabels: model.LabelNames{model.LabelName("job")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "__tmp_prometheus_job_name",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{
								model.LabelName("__meta_kubernetes_service_label_app_kubernetes_io_name"),
								model.LabelName("__meta_kubernetes_service_labelpresent_app_kubernetes_io_name"),
							},
							Separator:   ";",
							Regex:       relabel.MustNewRegexp("(testapp);true"),
							Replacement: "$$1",
							Action:      relabel.Keep,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_endpoint_port_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("http"),
							Replacement:  "$$1",
							Action:       relabel.Keep,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_namespace")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "namespace",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_service_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "service",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "pod",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_container_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "container",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
					},
				},
				"serviceMonitor/testapp/testapp1/0": {
					JobName:         "serviceMonitor/testapp/testapp1/0",
					HonorTimestamps: true,
					ScrapeInterval:  model.Duration(5 * time.Minute),
					ScrapeTimeout:   model.Duration(10 * time.Second),
					MetricsPath:     "/v2/metrics",
					Scheme:          "http",
					HTTPClientConfig: config.HTTPClientConfig{
						FollowRedirects: true,
					},
					RelabelConfigs: []*relabel.Config{
						{
							SourceLabels: model.LabelNames{model.LabelName("job")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "__tmp_prometheus_job_name",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{
								model.LabelName("__meta_kubernetes_service_label_app_kubernetes_io_name"),
								model.LabelName("__meta_kubernetes_service_labelpresent_app_kubernetes_io_name"),
							},
							Separator:   ";",
							Regex:       relabel.MustNewRegexp("(testapp);true"),
							Replacement: "$$1",
							Action:      relabel.Keep,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_endpoint_port_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("http"),
							Replacement:  "$$1",
							Action:       relabel.Keep,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_namespace")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "namespace",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_service_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "service",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "pod",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_container_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "container",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
					},
				},
				"serviceMonitor/testapp/testapp2/0": {
					JobName:         "serviceMonitor/testapp/testapp2/0",
					HonorTimestamps: true,
					ScrapeInterval:  model.Duration(30 * time.Minute),
					ScrapeTimeout:   model.Duration(2 * time.Minute),
					MetricsPath:     "/metrics",
					Scheme:          "http",
					HTTPClientConfig: config.HTTPClientConfig{
						FollowRedirects: true,
					},
					RelabelConfigs: []*relabel.Config{
						{
							SourceLabels: model.LabelNames{model.LabelName("job")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "__tmp_prometheus_job_name",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{
								model.LabelName("__meta_kubernetes_service_label_app_kubernetes_io_name"),
								model.LabelName("__meta_kubernetes_service_labelpresent_app_kubernetes_io_name"),
							},
							Separator:   ";",
							Regex:       relabel.MustNewRegexp("(testapp);true"),
							Replacement: "$$1",
							Action:      relabel.Keep,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_endpoint_port_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("http"),
							Replacement:  "$$1",
							Action:       relabel.Keep,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_namespace")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "namespace",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_service_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "service",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "pod",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
						{
							SourceLabels: model.LabelNames{model.LabelName("__meta_kubernetes_pod_container_name")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "container",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
					},
				},
			},
			expectedCode: http.StatusOK,
		},
		{
			description: "https secret handling",
			scrapeConfigs: map[string]*promconfig.ScrapeConfig{
				"serviceMonitor/testapp/testapp3/0": {
					JobName:         "serviceMonitor/testapp/testapp3/0",
					HonorTimestamps: true,
					ScrapeInterval:  model.Duration(30 * time.Second),
					ScrapeTimeout:   model.Duration(30 * time.Second),
					MetricsPath:     "/metrics",
					Scheme:          "http",
					HTTPClientConfig: config.HTTPClientConfig{
						FollowRedirects: true,
						BasicAuth: &config.BasicAuth{
							Username: "test",
							Password: "P@$$w0rd1!?",
						},
					},
				},
			},
			expectedCode: http.StatusOK,
			serverOptions: []Option{
				WithTLSConfig(tlsConfig, ""),
			},
		},
		{
			description: "http secret handling",
			scrapeConfigs: map[string]*promconfig.ScrapeConfig{
				"serviceMonitor/testapp/testapp3/0": {
					JobName:         "serviceMonitor/testapp/testapp3/0",
					HonorTimestamps: true,
					ScrapeInterval:  model.Duration(30 * time.Second),
					ScrapeTimeout:   model.Duration(30 * time.Second),
					MetricsPath:     "/metrics",
					Scheme:          "http",
					HTTPClientConfig: config.HTTPClientConfig{
						FollowRedirects: true,
						BasicAuth: &config.BasicAuth{
							Username: "test",
							Password: "P@$$w0rd1!?",
						},
					},
				},
			},
			expectedCode: http.StatusOK,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			listenAddr := ":8080"
			s, err := NewServer(logger, nil, listenAddr, tc.serverOptions...)
			require.NoError(t, err)
			assert.NoError(t, s.UpdateScrapeConfigResponse(tc.scrapeConfigs))

			request := httptest.NewRequest("GET", "/scrape_configs", nil)
			w := httptest.NewRecorder()

			if s.httpsServer != nil {
				request.TLS = &tls.ConnectionState{}
				s.httpsServer.Handler.ServeHTTP(w, request)
			} else {
				s.server.Handler.ServeHTTP(w, request)
			}
			result := w.Result()

			assert.Equal(t, tc.expectedCode, result.StatusCode)
			bodyBytes, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			if tc.expectedBody != nil {
				assert.Equal(t, tc.expectedBody, bodyBytes)
				return
			}
			scrapeConfigs := map[string]*promconfig.ScrapeConfig{}
			err = yaml.Unmarshal(bodyBytes, scrapeConfigs)
			require.NoError(t, err)

			for _, c := range scrapeConfigs {
				if s.httpsServer == nil && c.HTTPClientConfig.BasicAuth != nil {
					assert.Equal(t, c.HTTPClientConfig.BasicAuth.Password, config.Secret("<secret>"))
				}
			}

			for _, c := range tc.scrapeConfigs {
				if s.httpsServer == nil && c.HTTPClientConfig.BasicAuth != nil {
					c.HTTPClientConfig.BasicAuth.Password = "<secret>"
				}
			}

			assert.Equal(t, tc.scrapeConfigs, scrapeConfigs)
		})
	}
}

func TestServer_JobHandler(t *testing.T) {
	tests := []struct {
		description  string
		targetItems  map[target.ItemHash]*target.Item
		expectedCode int
		expectedJobs map[string]linkJSON
	}{
		{
			description:  "nil jobs",
			targetItems:  nil,
			expectedCode: http.StatusOK,
			expectedJobs: make(map[string]linkJSON),
		},
		{
			description:  "empty jobs",
			targetItems:  map[target.ItemHash]*target.Item{},
			expectedCode: http.StatusOK,
			expectedJobs: make(map[string]linkJSON),
		},
		{
			description: "one job",
			targetItems: map[target.ItemHash]*target.Item{
				0: target.NewItem("job1", "", labels.New(), ""),
			},
			expectedCode: http.StatusOK,
			expectedJobs: map[string]linkJSON{
				"job1": newLink("job1"),
			},
		},
		{
			description: "multiple jobs",
			targetItems: map[target.ItemHash]*target.Item{
				0: target.NewItem("job1", "", labels.New(), ""),
				1: target.NewItem("job2", "", labels.New(), ""),
				2: target.NewItem("job3", "", labels.New(), ""),
				3: target.NewItem("job3", "", labels.New(), ""),
				4: target.NewItem("job3", "", labels.New(), "")},
			expectedCode: http.StatusOK,
			expectedJobs: map[string]linkJSON{
				"job1": newLink("job1"),
				"job2": newLink("job2"),
				"job3": newLink("job3"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			listenAddr := ":8080"
			a := &mockAllocator{targetItems: tc.targetItems}
			s, err := NewServer(logger, a, listenAddr)
			require.NoError(t, err)
			request := httptest.NewRequest("GET", "/jobs", nil)
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, tc.expectedCode, result.StatusCode)
			bodyBytes, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			jobs := map[string]linkJSON{}
			err = json.Unmarshal(bodyBytes, &jobs)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedJobs, jobs)
		})
	}
}
func TestServer_JobsHandler_HTML(t *testing.T) {
	tests := []struct {
		description  string
		targetItems  map[target.ItemHash]*target.Item
		expectedCode int
		Golden       string
	}{
		{
			description:  "nil jobs",
			targetItems:  nil,
			expectedCode: http.StatusOK,
			Golden:       "jobs_empty.html",
		},
		{
			description:  "empty jobs",
			targetItems:  map[target.ItemHash]*target.Item{},
			expectedCode: http.StatusOK,
			Golden:       "jobs_empty.html",
		},
		{
			description: "one job",
			targetItems: map[target.ItemHash]*target.Item{
				0: target.NewItem("job1", "", labels.New(), ""),
			},
			expectedCode: http.StatusOK,
			Golden:       "jobs_one.html",
		},
		{
			description: "multiple jobs",
			targetItems: map[target.ItemHash]*target.Item{
				0: target.NewItem("job1", "1.1.1.1:8080", labels.New(), ""),
				1: target.NewItem("job2", "1.1.1.2:8080", labels.New(), ""),
				2: target.NewItem("job3", "1.1.1.3:8080", labels.New(), ""),
				3: target.NewItem("job3", "1.1.1.4:8080", labels.New(), ""),
				4: target.NewItem("job3", "1.1.1.5:8080", labels.New(), "")},
			expectedCode: http.StatusOK,
			Golden:       "jobs_multiple.html",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			listenAddr := ":8080"
			a := &mockAllocator{targetItems: tc.targetItems}
			s, err := NewServer(logger, a, listenAddr)
			require.NoError(t, err)
			a.SetCollectors(map[string]*allocation.Collector{
				"test-collector":  {Name: "test-collector"},
				"test-collector2": {Name: "test-collector2"},
			})
			request := httptest.NewRequest("GET", "/debug/jobs", nil)
			request.Header.Set("Accept", "text/html")
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, tc.expectedCode, result.StatusCode)
			bodyBytes, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			golden.Assert(t, string(bodyBytes), tc.Golden)
		})
	}
}

func TestServer_JobHandler_HTML(t *testing.T) {
	consistentHashing, _ := allocation.New("consistent-hashing", logger)
	type args struct {
		job  string
		cMap []*target.Item

		allocator allocation.Allocator
	}
	tests := []struct {
		name   string
		args   args
		Golden string
	}{
		{
			name: "Empty target map",
			args: args{
				job:       "test-job",
				cMap:      []*target.Item{},
				allocator: consistentHashing,
			},
			Golden: "job_empty.html",
		},
		{
			name: "Single entry target map",
			args: args{
				job: "test-job",
				cMap: []*target.Item{
					baseTargetItem,
				},
				allocator: consistentHashing,
			},
			Golden: "job_single.html",
		},
		{
			name: "Multiple entry target map",
			args: args{
				job: "test-job",
				cMap: []*target.Item{
					baseTargetItem,
					testJobTwoTargetItemTwo,
				},
				allocator: consistentHashing,
			},
			Golden: "job_multiple.html",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listenAddr := ":8080"
			s, err := NewServer(logger, tt.args.allocator, listenAddr)
			require.NoError(t, err)
			tt.args.allocator.SetCollectors(map[string]*allocation.Collector{
				"test-collector":  {Name: "test-collector"},
				"test-collector2": {Name: "test-collector2"},
			})
			tt.args.allocator.SetTargets(tt.args.cMap)
			request := httptest.NewRequest("GET", fmt.Sprintf("/debug/job?job_id=%s", tt.args.job), nil)
			request.Header.Set("Accept", "text/html")
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, http.StatusOK, result.StatusCode)
			body := result.Body
			bodyBytes, err := io.ReadAll(body)
			assert.NoError(t, err)
			golden.Assert(t, string(bodyBytes), tt.Golden)
		})
	}
}

func TestServer_IndexHandler(t *testing.T) {
	allocator, _ := allocation.New("consistent-hashing", logger)
	tests := []struct {
		description string
		allocator   allocation.Allocator
		targetItems []*target.Item
		Golden      string
	}{
		{
			description: "Empty target map",
			targetItems: []*target.Item{},
			allocator:   allocator,
			Golden:      "index_empty.html",
		},
		{
			description: "Single entry target map",
			targetItems: []*target.Item{
				baseTargetItem,
			},
			allocator: allocator,
			Golden:    "index_single.html",
		},
		{
			description: "Multiple entry target map",
			targetItems: []*target.Item{
				baseTargetItem,
				testJobTargetItemTwo,
				testJobTwoTargetItemTwo,
			},
			allocator: allocator,
			Golden:    "index_multiple.html",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			listenAddr := ":8080"
			s, err := NewServer(logger, tc.allocator, listenAddr)
			require.NoError(t, err)
			tc.allocator.SetCollectors(map[string]*allocation.Collector{
				"test-collector1": {Name: "test-collector1"},
				"test-collector2": {Name: "test-collector2"},
			})
			tc.allocator.SetTargets(tc.targetItems)
			request := httptest.NewRequest("GET", "/", nil)
			request.Header.Set("Accept", "text/html")
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, http.StatusOK, result.StatusCode)
			body := result.Body
			bodyBytes, err := io.ReadAll(body)
			assert.NoError(t, err)
			golden.Assert(t, string(bodyBytes), tc.Golden)
		})
	}
}
func TestServer_TargetsHTMLHandler(t *testing.T) {
	allocator, _ := allocation.New("consistent-hashing", logger)
	tests := []struct {
		description string
		allocator   allocation.Allocator
		targetItems []*target.Item
		Golden      string
	}{
		{
			description: "Empty target map",
			targetItems: []*target.Item{},
			allocator:   allocator,
			Golden:      "targets_empty.html",
		},
		{
			description: "Single entry target map",
			targetItems: []*target.Item{
				baseTargetItem,
			},
			allocator: allocator,
			Golden:    "targets_single.html",
		},
		{
			description: "Multiple entry target map",
			targetItems: []*target.Item{
				baseTargetItem,
				testJobTargetItemTwo,
				testJobTwoTargetItemTwo,
			},
			allocator: allocator,
			Golden:    "targets_multiple.html",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			listenAddr := ":8080"
			s, err := NewServer(logger, tc.allocator, listenAddr)
			require.NoError(t, err)
			tc.allocator.SetCollectors(map[string]*allocation.Collector{
				"test-collector1": {Name: "test-collector1"},
				"test-collector2": {Name: "test-collector2"},
			})
			tc.allocator.SetTargets(tc.targetItems)
			request := httptest.NewRequest("GET", "/debug/targets", nil)
			request.Header.Set("Accept", "text/html")
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, http.StatusOK, result.StatusCode)
			body := result.Body
			bodyBytes, err := io.ReadAll(body)
			assert.NoError(t, err)
			golden.Assert(t, string(bodyBytes), tc.Golden)
		})
	}
}

func TestServer_CollectorHandler(t *testing.T) {
	allocator, _ := allocation.New("consistent-hashing", logger)
	tests := []struct {
		description  string
		collectorId  string
		allocator    allocation.Allocator
		targetItems  []*target.Item
		expectedCode int
		Golden       string
	}{
		{
			description:  "Empty target map",
			collectorId:  "test-collector",
			targetItems:  []*target.Item{},
			allocator:    allocator,
			expectedCode: http.StatusOK,
			Golden:       "collector_empty.html",
		},
		{
			description: "Single entry target map",
			collectorId: "test-collector2",
			targetItems: []*target.Item{
				baseTargetItem,
			},
			allocator:    allocator,
			expectedCode: http.StatusOK,
			Golden:       "collector_single.html",
		},
		{
			description: "Multiple entry target map",
			collectorId: "test-collector2",
			targetItems: []*target.Item{
				baseTargetItem,
				testJobTwoTargetItemTwo,
			},
			allocator:    allocator,
			expectedCode: http.StatusOK,
			Golden:       "collector_multiple.html",
		},
		{
			description: "Multiple entry target map, collector id is empty",
			collectorId: "",
			targetItems: []*target.Item{
				baseTargetItem,
				testJobTwoTargetItemTwo,
			},
			allocator:    allocator,
			expectedCode: http.StatusBadRequest,
			Golden:       "collector_empty_id.html",
		},
		{
			description: "Multiple entry target map, unknown collector id",
			collectorId: "unknown-collector-1",
			targetItems: []*target.Item{
				baseTargetItem,
				testJobTwoTargetItemTwo,
			},
			allocator:    allocator,
			expectedCode: http.StatusNotFound,
			Golden:       "collector_unknown_id.html",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			listenAddr := ":8080"
			s, err := NewServer(logger, tc.allocator, listenAddr)
			require.NoError(t, err)
			tc.allocator.SetCollectors(map[string]*allocation.Collector{
				"test-collector":  {Name: "test-collector"},
				"test-collector2": {Name: "test-collector2"},
			})
			tc.allocator.SetTargets(tc.targetItems)
			request := httptest.NewRequest("GET", "/debug/collector", nil)
			request.Header.Set("Accept", "text/html")
			request.URL.RawQuery = "collector_id=" + tc.collectorId
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, tc.expectedCode, result.StatusCode)
			body := result.Body
			bodyBytes, err := io.ReadAll(body)
			assert.NoError(t, err)
			golden.Assert(t, string(bodyBytes), tc.Golden)
		})
	}
}

func TestServer_TargetHTMLHandler(t *testing.T) {
	allocator, _ := allocation.New("consistent-hashing", logger)
	tests := []struct {
		description  string
		targetHash   target.ItemHash
		allocator    allocation.Allocator
		targetItems  []*target.Item
		expectedCode int
		Golden       string
	}{
		{
			description:  "Missing target hash",
			targetHash:   0,
			targetItems:  []*target.Item{},
			allocator:    allocator,
			expectedCode: http.StatusBadRequest,
			Golden:       "target_empty_hash.html",
		},
		{
			description: "Single entry target map",
			targetHash:  baseTargetItem.Hash(),
			targetItems: []*target.Item{
				baseTargetItem,
			},
			allocator:    allocator,
			expectedCode: http.StatusOK,
			Golden:       "target_single.html",
		},
		{
			description: "Multiple entry target map",
			targetHash:  testJobTwoTargetItemTwo.Hash(),
			targetItems: []*target.Item{
				baseTargetItem,
				testJobTwoTargetItemTwo,
			},
			allocator:    allocator,
			expectedCode: http.StatusOK,
			Golden:       "target_multiple.html",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			listenAddr := ":8080"
			s, err := NewServer(logger, tc.allocator, listenAddr)
			require.NoError(t, err)
			tc.allocator.SetCollectors(map[string]*allocation.Collector{
				"test-collector":  {Name: "test-collector"},
				"test-collector2": {Name: "test-collector2"},
			})
			tc.allocator.SetTargets(tc.targetItems)
			request := httptest.NewRequest("GET", "/debug/target", nil)
			request.Header.Set("Accept", "text/html")
			request.URL.RawQuery = "target_hash=" + tc.targetHash.String()
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, tc.expectedCode, result.StatusCode)
			body := result.Body
			bodyBytes, err := io.ReadAll(body)
			assert.NoError(t, err)
			golden.Assert(t, string(bodyBytes), tc.Golden)
		})
	}
}

func TestServer_Readiness(t *testing.T) {
	tests := []struct {
		description   string
		scrapeConfigs map[string]*promconfig.ScrapeConfig
		expectedCode  int
		expectedBody  []byte
	}{
		{
			description:   "nil scrape config",
			scrapeConfigs: nil,
			expectedCode:  http.StatusServiceUnavailable,
		},
		{
			description:   "empty scrape config",
			scrapeConfigs: map[string]*promconfig.ScrapeConfig{},
			expectedCode:  http.StatusOK,
		},
		{
			description: "single entry",
			scrapeConfigs: map[string]*promconfig.ScrapeConfig{
				"serviceMonitor/testapp/testapp/0": {
					JobName:         "serviceMonitor/testapp/testapp/0",
					HonorTimestamps: true,
					ScrapeInterval:  model.Duration(30 * time.Second),
					ScrapeTimeout:   model.Duration(30 * time.Second),
					MetricsPath:     "/metrics",
					Scheme:          "http",
					HTTPClientConfig: config.HTTPClientConfig{
						FollowRedirects: true,
					},
					RelabelConfigs: []*relabel.Config{
						{
							SourceLabels: model.LabelNames{model.LabelName("job")},
							Separator:    ";",
							Regex:        relabel.MustNewRegexp("(.*)"),
							TargetLabel:  "__tmp_prometheus_job_name",
							Replacement:  "$$1",
							Action:       relabel.Replace,
						},
					},
				},
			},
			expectedCode: http.StatusOK,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			listenAddr := ":8080"
			s, err := NewServer(logger, nil, listenAddr)
			require.NoError(t, err)
			if tc.scrapeConfigs != nil {
				assert.NoError(t, s.UpdateScrapeConfigResponse(tc.scrapeConfigs))
			}

			request := httptest.NewRequest("GET", "/readyz", nil)
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, tc.expectedCode, result.StatusCode)
		})
	}
}

func TestServer_ScrapeConfigResponse(t *testing.T) {
	tests := []struct {
		description  string
		filePath     string
		expectedCode int
	}{
		{
			description:  "Jobs with all actions",
			filePath:     "./testdata/prom-config-all-actions.yaml",
			expectedCode: http.StatusOK,
		},
		{
			description:  "Jobs with config combinations",
			filePath:     "./testdata/prom-config-test.yaml",
			expectedCode: http.StatusOK,
		},
		{
			description:  "Jobs with no config",
			filePath:     "./testdata/prom-no-config.yaml",
			expectedCode: http.StatusOK,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			listenAddr := ":8080"
			s, err := NewServer(logger, nil, listenAddr)
			require.NoError(t, err)

			allocCfg := allocatorconfig.CreateDefaultConfig()
			err = allocatorconfig.LoadFromFile(tc.filePath, &allocCfg)
			require.NoError(t, err)

			jobToScrapeConfig := make(map[string]*promconfig.ScrapeConfig)

			for _, scrapeConfig := range allocCfg.PromConfig.ScrapeConfigs {
				jobToScrapeConfig[scrapeConfig.JobName] = scrapeConfig
			}

			assert.NoError(t, s.UpdateScrapeConfigResponse(jobToScrapeConfig))

			request := httptest.NewRequest("GET", "/scrape_configs", nil)
			w := httptest.NewRecorder()

			s.server.Handler.ServeHTTP(w, request)
			result := w.Result()

			assert.Equal(t, tc.expectedCode, result.StatusCode)
			bodyBytes, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			// Checking to make sure yaml unmarshaling doesn't result in errors for responses containing any supported prometheus relabel action
			scrapeConfigs := map[string]*promconfig.ScrapeConfig{}
			err = yaml.Unmarshal(bodyBytes, scrapeConfigs)
			require.NoError(t, err)
		})
	}
}

func newLink(jobName string) linkJSON {
	return linkJSON{Link: fmt.Sprintf("/jobs/%s/targets", url.QueryEscape(jobName))}
}
