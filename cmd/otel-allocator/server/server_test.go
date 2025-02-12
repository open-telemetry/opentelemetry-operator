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
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	allocatorconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

var (
	logger       = logf.Log.WithName("server-unit-tests")
	baseLabelSet = labels.Labels{
		{Name: "test_label", Value: "test-value"},
	}
	testJobLabelSetTwo = labels.Labels{
		{Name: "test_label", Value: "test-value2"},
	}
	baseTargetItem       = target.NewItem("test-job", "test-url", baseLabelSet, "test-collector")
	secondTargetItem     = target.NewItem("test-job", "test-url", baseLabelSet, "test-collector")
	testJobTargetItemTwo = target.NewItem("test-job", "test-url2", testJobLabelSetTwo, "test-collector2")
)

func TestServer_LivenessProbeHandler(t *testing.T) {
	leastWeighted, _ := allocation.New("least-weighted", logger)
	listenAddr := ":8080"
	s := NewServer(logger, leastWeighted, listenAddr)
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
		cMap      map[string]*target.Item
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
				cMap:      map[string]*target.Item{},
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
				cMap: map[string]*target.Item{
					baseTargetItem.Hash(): baseTargetItem,
				},
				allocator: leastWeighted,
			},
			want: want{
				items: []*targetJSON{
					{
						TargetURL: []string{"test-url"},
						Labels: labels.Labels{
							{Name: "test_label", Value: "test-value"},
						},
					},
				},
			},
		},
		{
			name: "Multiple entry target map",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string]*target.Item{
					baseTargetItem.Hash():   baseTargetItem,
					secondTargetItem.Hash(): secondTargetItem,
				},
				allocator: leastWeighted,
			},
			want: want{
				items: []*targetJSON{
					{
						TargetURL: []string{"test-url"},
						Labels: labels.Labels{
							{Name: "test_label", Value: "test-value"},
						},
					},
				},
			},
		},
		{
			name: "Multiple entry target map of same job with label merge",
			args: args{
				collector: "test-collector",
				job:       "test-job",
				cMap: map[string]*target.Item{
					baseTargetItem.Hash():       baseTargetItem,
					testJobTargetItemTwo.Hash(): testJobTargetItemTwo,
				},
				allocator: leastWeighted,
			},
			want: want{
				items: []*targetJSON{
					{
						TargetURL: []string{"test-url"},
						Labels: labels.Labels{
							{Name: "test_label", Value: "test-value"},
						},
					},
					{
						TargetURL: []string{"test-url2"},
						Labels: labels.Labels{
							{Name: "test_label", Value: "test-value2"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listenAddr := ":8080"
			s := NewServer(logger, tt.args.allocator, listenAddr)
			tt.args.allocator.SetCollectors(map[string]*allocation.Collector{"test-collector": {Name: "test-collector"}})
			tt.args.allocator.SetTargets(tt.args.cMap)
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
			s := NewServer(logger, nil, listenAddr, tc.serverOptions...)
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
		targetItems  map[string]*target.Item
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
			targetItems:  map[string]*target.Item{},
			expectedCode: http.StatusOK,
			expectedJobs: make(map[string]linkJSON),
		},
		{
			description: "one job",
			targetItems: map[string]*target.Item{
				"targetitem": target.NewItem("job1", "", labels.Labels{}, ""),
			},
			expectedCode: http.StatusOK,
			expectedJobs: map[string]linkJSON{
				"job1": newLink("job1"),
			},
		},
		{
			description: "multiple jobs",
			targetItems: map[string]*target.Item{
				"a": target.NewItem("job1", "", labels.Labels{}, ""),
				"b": target.NewItem("job2", "", labels.Labels{}, ""),
				"c": target.NewItem("job3", "", labels.Labels{}, ""),
				"d": target.NewItem("job3", "", labels.Labels{}, ""),
				"e": target.NewItem("job3", "", labels.Labels{}, "")},
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
			s := NewServer(logger, a, listenAddr)
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
			s := NewServer(logger, nil, listenAddr)
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

func TestServer_ScrapeConfigRespose(t *testing.T) {
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
			s := NewServer(logger, nil, listenAddr)

			allocCfg := allocatorconfig.CreateDefaultConfig()
			err := allocatorconfig.LoadFromFile(tc.filePath, &allocCfg)
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
