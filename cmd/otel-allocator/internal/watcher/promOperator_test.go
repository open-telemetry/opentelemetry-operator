// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus-operator/prometheus-operator/pkg/assets"
	fakemonitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/fake"
	"github.com/prometheus-operator/prometheus-operator/pkg/informers"
	"github.com/prometheus-operator/prometheus-operator/pkg/operator"
	"github.com/prometheus-operator/prometheus-operator/pkg/prometheus"
	prometheusgoclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	kubeDiscovery "github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	fcache "k8s.io/client-go/tools/cache/testing"
	"k8s.io/utils/ptr"

	allocatorconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

func TestLoadConfig(t *testing.T) {
	namespace := "test"
	portName := "web"
	tests := []struct {
		name            string
		serviceMonitors []*monitoringv1.ServiceMonitor
		podMonitors     []*monitoringv1.PodMonitor
		scrapeClasses   []*monitoringv1.ScrapeClass
		scrapeConfigs   []*promv1alpha1.ScrapeConfig
		probes          []*monitoringv1.Probe
		want            *promconfig.Config
		wantErr         bool
		cfg             allocatorconfig.Config
	}{
		{
			name: "simple test",
			serviceMonitors: []*monitoringv1.ServiceMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: namespace,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "test",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
							},
						},
					},
				},
			},
			podMonitors: []*monitoringv1.PodMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: namespace,
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel: "test",
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ServiceMonitorSelector: &metav1.LabelSelector{},
					PodMonitorSelector:     &metav1.LabelSelector{},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/test/simple/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "endpointslice",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
					{
						JobName:         "podMonitor/test/simple/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "pod",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "basic auth (serviceMonitor)",
			serviceMonitors: []*monitoringv1.ServiceMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "auth",
						Namespace: namespace,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "auth",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
								BasicAuth: &monitoringv1.BasicAuth{
									Username: v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{
											Name: "basic-auth",
										},
										Key: "username",
									},
									Password: v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{
											Name: "basic-auth",
										},
										Key: "password",
									},
								},
							},
						},
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "auth",
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ServiceMonitorSelector: &metav1.LabelSelector{},
					PodMonitorSelector:     &metav1.LabelSelector{},
				},
			},
			want: &promconfig.Config{
				GlobalConfig: promconfig.GlobalConfig{},
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/test/auth/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "endpointslice",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig: config.HTTPClientConfig{
							FollowRedirects: true,
							EnableHTTP2:     true,
							BasicAuth: &config.BasicAuth{
								Username: "admin",
								Password: "password",
							},
						},
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "bearer token (podMonitor)",
			podMonitors: []*monitoringv1.PodMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bearer",
						Namespace: namespace,
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel: "bearer",
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
								HTTPConfig: monitoringv1.HTTPConfig{
									Authorization: &monitoringv1.SafeAuthorization{
										Type: "Bearer",
										Credentials: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "bearer",
											},
											Key: "token",
										},
									},
								},
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ServiceMonitorSelector: &metav1.LabelSelector{},
					PodMonitorSelector:     &metav1.LabelSelector{},
				},
			},
			want: &promconfig.Config{
				GlobalConfig: promconfig.GlobalConfig{},
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "podMonitor/test/bearer/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "pod",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig: config.HTTPClientConfig{
							FollowRedirects: true,
							EnableHTTP2:     true,
							Authorization: &config.Authorization{
								Type:        "Bearer",
								Credentials: "bearer-token",
							},
						},
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "invalid pod monitor test",
			serviceMonitors: []*monitoringv1.ServiceMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "valid-sm",
						Namespace: namespace,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "test",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
							},
						},
					},
				},
			},
			podMonitors: []*monitoringv1.PodMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "valid-pm",
						Namespace: namespace,
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel: "test",
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-pm",
						Namespace: namespace,
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel: "test",
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
								RelabelConfigs: []monitoringv1.RelabelConfig{
									{
										Action:      "keep",
										Regex:       ".*(",
										Replacement: ptr.To("invalid"),
										TargetLabel: "city",
									},
								},
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ServiceMonitorSelector: &metav1.LabelSelector{},
					PodMonitorSelector:     &metav1.LabelSelector{},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/test/valid-sm/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "endpointslice",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
					{
						JobName:         "podMonitor/test/valid-pm/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "pod",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "invalid service monitor test",
			serviceMonitors: []*monitoringv1.ServiceMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "valid-sm",
						Namespace: namespace,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "test",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-sm",
						Namespace: namespace,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "test",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
								RelabelConfigs: []monitoringv1.RelabelConfig{
									{
										Action:      "keep",
										Regex:       ".*(",
										Replacement: ptr.To("invalid"),
										TargetLabel: "city",
									},
								},
							},
						},
					},
				},
			},
			podMonitors: []*monitoringv1.PodMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "valid-pm",
						Namespace: namespace,
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel: "test",
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ServiceMonitorSelector: &metav1.LabelSelector{},
					PodMonitorSelector:     &metav1.LabelSelector{},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/test/valid-sm/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "endpointslice",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
					{
						JobName:         "podMonitor/test/valid-pm/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "pod",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "service monitor selector test",
			serviceMonitors: []*monitoringv1.ServiceMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sm-1",
						Namespace: namespace,
						Labels: map[string]string{
							"testsvc": "testsvc",
						},
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "test",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sm-2",
						Namespace: "test",
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: namespace,
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ServiceMonitorSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testsvc": "testsvc",
						},
					},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/test/sm-1/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "endpointslice",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "pod monitor selector test",
			podMonitors: []*monitoringv1.PodMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pm-1",
						Namespace: namespace,
						Labels: map[string]string{
							"testpod": "testpod",
						},
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel: "test",
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pm-2",
						Namespace: namespace,
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel: "test",
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					PodMonitorSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testpod": "testpod",
						},
					},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "podMonitor/test/pm-1/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "pod",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "scrape configs selector test",
			scrapeConfigs: []*promv1alpha1.ScrapeConfig{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "scrapeconfig-test-1",
						Namespace: namespace,
						Labels: map[string]string{
							"testpod": "testpod",
						},
					},
					Spec: promv1alpha1.ScrapeConfigSpec{
						JobName: func() *string {
							j := "scrapeConfig/test/scrapeconfig-test-1"
							return &j
						}(),
						StaticConfigs: []promv1alpha1.StaticConfig{
							{
								Targets: []promv1alpha1.Target{"127.0.0.1:8888"},
								Labels:  nil,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ScrapeConfigSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testpod": "testpod",
						},
					},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "scrapeConfig/test/scrapeconfig-test-1",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							discovery.StaticConfig{
								&targetgroup.Group{
									Targets: []model.LabelSet{
										map[model.LabelName]model.LabelValue{
											"__address__": "127.0.0.1:8888",
										},
									},
									Labels: map[model.LabelName]model.LabelValue{},
									Source: "0",
								},
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "probe selector test",
			probes: []*monitoringv1.Probe{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "probe-test-1",
						Namespace: namespace,
						Labels: map[string]string{
							"testpod": "testpod",
						},
					},
					Spec: monitoringv1.ProbeSpec{
						JobName: "probe/test/probe-1/0",
						ProberSpec: monitoringv1.ProberSpec{
							URL:  "localhost:50671",
							Path: "/metrics",
						},
						Targets: monitoringv1.ProbeTargets{
							StaticConfig: &monitoringv1.ProbeTargetStaticConfig{
								Targets: []string{"prometheus.io"},
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ProbeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"testpod": "testpod",
						},
					},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "probe/test/probe-test-1",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							discovery.StaticConfig{
								&targetgroup.Group{
									Targets: []model.LabelSet{
										map[model.LabelName]model.LabelValue{
											"__address__": "prometheus.io",
										},
									},
									Labels: map[model.LabelName]model.LabelValue{
										"namespace": model.LabelValue(namespace),
									},
									Source: "0",
								},
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "service monitor namespace selector test",
			serviceMonitors: []*monitoringv1.ServiceMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sm-1",
						Namespace: "labellednamespace",
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "test",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sm-2",
						Namespace: namespace,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "test",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ServiceMonitorSelector: &metav1.LabelSelector{},
					PodMonitorSelector:     &metav1.LabelSelector{},
					ServiceMonitorNamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"label1": "label1",
						},
					},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/labellednamespace/sm-1/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "endpointslice",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{"labellednamespace"},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "pod monitor namespace selector test",
			podMonitors: []*monitoringv1.PodMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pm-1",
						Namespace: "labellednamespace",
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel: "test",
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pm-2",
						Namespace: namespace,
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel: "test",
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ServiceMonitorSelector: &metav1.LabelSelector{},
					PodMonitorSelector:     &metav1.LabelSelector{},
					PodMonitorNamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"label1": "label1",
						},
					},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "podMonitor/labellednamespace/pm-1/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "pod",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{"labellednamespace"},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
		{
			name: "pod monitor with referenced scrape class",
			podMonitors: []*monitoringv1.PodMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "simple",
						Namespace: namespace,
					},
					Spec: monitoringv1.PodMonitorSpec{
						JobLabel:        "test",
						ScrapeClassName: ptr.To("attach-node-metadata"),
						PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
							{
								Port: &portName,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					PodMonitorSelector: &metav1.LabelSelector{},
					ScrapeClasses: []monitoringv1.ScrapeClass{
						{
							Name: "attach-node-metadata",
							AttachMetadata: &monitoringv1.AttachMetadata{
								Node: ptr.To(true),
							},
						},
					},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "podMonitor/test/simple/0",
						ScrapeInterval:  model.Duration(60 * time.Second),
						ScrapeProtocols: promconfig.DefaultScrapeProtocols,
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "pod",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{namespace},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
								AttachMetadata: kubeDiscovery.AttachMetadataConfig{
									Node: true, // Added by scrape-class!
								},
							},
						},
						HTTPClientConfig:               config.DefaultHTTPClientConfig,
						EnableCompression:              true,
						AlwaysScrapeClassicHistograms:  ptr.To(false),
						ConvertClassicHistogramsToNHCB: ptr.To(false),
						MetricNameValidationScheme:     model.UTF8Validation,
						MetricNameEscapingScheme:       model.AllowUTF8,
						ScrapeNativeHistograms:         ptr.To(false),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := getTestPrometheusCRWatcher(t, namespace, tt.serviceMonitors, tt.podMonitors, tt.probes, tt.scrapeConfigs, tt.cfg)

			// Start namespace informers in order to populate cache.
			go w.nsInformer.Run(w.stopChannel)
			for !w.nsInformer.HasSynced() {
				time.Sleep(50 * time.Millisecond)
			}

			for _, informer := range w.informers {
				// Start informers in order to populate cache.
				informer.Start(w.stopChannel)
			}

			// Wait for informers to sync.
			for _, informer := range w.informers {
				for !informer.HasSynced() {
					time.Sleep(50 * time.Millisecond)
				}
			}

			got, err := w.LoadConfig(context.Background())
			assert.NoError(t, err)

			sanitizeScrapeConfigsForTest(got.ScrapeConfigs)
			assert.Equal(t, tt.want.ScrapeConfigs, got.ScrapeConfigs)
		})
	}
}

func TestNamespaceLabelUpdate(t *testing.T) {
	var err error
	namespace := "test"
	portName := "web"
	podMonitors := []*monitoringv1.PodMonitor{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pm-1",
				Namespace: "labellednamespace",
			},
			Spec: monitoringv1.PodMonitorSpec{
				JobLabel: "test",
				PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
					{
						Port: &portName,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pm-2",
				Namespace: namespace,
			},
			Spec: monitoringv1.PodMonitorSpec{
				JobLabel: "test",
				PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
					{
						Port: &portName,
					},
				},
			},
		},
	}

	cfg := allocatorconfig.Config{
		PrometheusCR: allocatorconfig.PrometheusCRConfig{
			ServiceMonitorSelector: &metav1.LabelSelector{},
			PodMonitorSelector:     &metav1.LabelSelector{},
			PodMonitorNamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"label1": "label1",
				},
			},
		},
	}

	want_before := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName:         "podMonitor/labellednamespace/pm-1/0",
				ScrapeInterval:  model.Duration(60 * time.Second),
				ScrapeProtocols: promconfig.DefaultScrapeProtocols,
				ScrapeTimeout:   model.Duration(10 * time.Second),
				HonorTimestamps: true,
				HonorLabels:     false,
				Scheme:          "http",
				MetricsPath:     "/metrics",
				ServiceDiscoveryConfigs: []discovery.Config{
					&kubeDiscovery.SDConfig{
						Role: "pod",
						NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
							Names:               []string{"labellednamespace"},
							IncludeOwnNamespace: false,
						},
						HTTPClientConfig: config.DefaultHTTPClientConfig,
					},
				},
				HTTPClientConfig:               config.DefaultHTTPClientConfig,
				EnableCompression:              true,
				AlwaysScrapeClassicHistograms:  ptr.To(false),
				ConvertClassicHistogramsToNHCB: ptr.To(false),
				MetricNameValidationScheme:     model.UTF8Validation,
				MetricNameEscapingScheme:       model.AllowUTF8,
				ScrapeNativeHistograms:         ptr.To(false),
			},
		},
	}

	want_after := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{},
	}

	w, source := getTestPrometheusCRWatcher(t, namespace, nil, podMonitors, nil, nil, cfg)
	events := make(chan Event, 1)
	eventInterval := 5 * time.Millisecond

	defer w.Close()
	w.eventInterval = eventInterval

	go func() {
		watchErr := w.Watch(events, make(chan error))
		require.NoError(t, watchErr)
	}()

	if success := cache.WaitForNamedCacheSync("namespace", w.stopChannel, w.nsInformer.HasSynced); !success {
		require.True(t, success)
	}

	for _, informer := range w.informers {
		success := cache.WaitForCacheSync(w.stopChannel, informer.HasSynced)
		require.True(t, success)
	}

	got, err := w.LoadConfig(context.Background())
	assert.NoError(t, err)

	sanitizeScrapeConfigsForTest(got.ScrapeConfigs)
	assert.Equal(t, want_before.ScrapeConfigs, got.ScrapeConfigs)

	source.Modify(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "labellednamespace",
		Labels: map[string]string{
			"label2": "label2",
		},
	}})

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		got, err = w.LoadConfig(context.Background())
		assert.NoError(collect, err)

		sanitizeScrapeConfigsForTest(got.ScrapeConfigs)
		assert.Equal(collect, want_after.ScrapeConfigs, got.ScrapeConfigs)
	}, time.Second*60, time.Millisecond*100)
}

func TestRateLimit(t *testing.T) {
	var err error
	namespace := "test"
	serviceMonitor := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: namespace,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			JobLabel: "test",
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "web",
				},
			},
		},
	}
	events := make(chan Event, 1)
	eventInterval := 500 * time.Millisecond
	cfg := allocatorconfig.Config{}

	w, _ := getTestPrometheusCRWatcher(t, namespace, nil, nil, nil, nil, cfg)
	defer w.Close()
	w.eventInterval = eventInterval

	go func() {
		watchErr := w.Watch(events, make(chan error))
		require.NoError(t, watchErr)
	}()
	// we don't have a simple way to wait for the watch to actually add event handlers to the informer,
	// instead, we just update a ServiceMonitor periodically and wait until we get a notification
	_, err = w.kubeMonitoringClient.MonitoringV1().ServiceMonitors("test").Create(context.Background(), serviceMonitor, metav1.CreateOptions{})
	require.NoError(t, err)

	// wait for cache sync first
	for _, informer := range w.informers {
		success := cache.WaitForCacheSync(w.stopChannel, informer.HasSynced)
		require.True(t, success)
	}

	require.Eventually(t, func() bool {
		_, createErr := w.kubeMonitoringClient.MonitoringV1().ServiceMonitors("test").Update(context.Background(), serviceMonitor, metav1.UpdateOptions{})
		if createErr != nil {
			return false
		}
		select {
		case <-events:
			return true
		default:
			return false
		}
	}, time.Second*5, eventInterval/10)

	// it's difficult to measure the rate precisely
	// what we do, is send two updates, and then assert that the elapsed time is at least eventInterval
	startTime := time.Now()
	_, err = w.kubeMonitoringClient.MonitoringV1().ServiceMonitors("test").Update(context.Background(), serviceMonitor, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		select {
		case <-events:
			return true
		default:
			return false
		}
	}, time.Second*5, eventInterval/10)
	_, err = w.kubeMonitoringClient.MonitoringV1().ServiceMonitors("test").Update(context.Background(), serviceMonitor, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		select {
		case <-events:
			return true
		default:
			return false
		}
	}, time.Second*5, eventInterval/10)
	elapsedTime := time.Since(startTime)
	assert.Less(t, eventInterval, elapsedTime)
}

func TestDefaultDurations(t *testing.T) {
	namespace := "test"
	portName := "web"
	tests := []struct {
		name            string
		serviceMonitors []*monitoringv1.ServiceMonitor
		cfg             allocatorconfig.Config
		expectedScrape  model.Duration
		expectedEval    model.Duration
	}{
		{
			name: "custom scrape and evaluation intervals",
			serviceMonitors: []*monitoringv1.ServiceMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-sm",
						Namespace: namespace,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "test",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ScrapeInterval:         model.Duration(120 * time.Second),
					EvaluationInterval:     model.Duration(120 * time.Second),
					ServiceMonitorSelector: &metav1.LabelSelector{},
				},
			},
			expectedScrape: model.Duration(120 * time.Second),
			expectedEval:   model.Duration(120 * time.Second),
		},
		{
			name: "prometheus operator applies defaults when intervals nil",
			serviceMonitors: []*monitoringv1.ServiceMonitor{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-sm",
						Namespace: namespace,
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						JobLabel: "test",
						Endpoints: []monitoringv1.Endpoint{
							{
								Port: portName,
							},
						},
					},
				},
			},
			cfg: allocatorconfig.Config{
				PrometheusCR: allocatorconfig.PrometheusCRConfig{
					ServiceMonitorSelector: &metav1.LabelSelector{},
				},
			},
			expectedScrape: model.Duration(60 * time.Second),
			expectedEval:   model.Duration(60 * time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := getTestPrometheusCRWatcher(t, namespace, tt.serviceMonitors, nil, nil, nil, tt.cfg)
			defer w.Close()

			events := make(chan Event, 1)
			eventInterval := 5 * time.Millisecond
			w.eventInterval = eventInterval

			go func() {
				watchErr := w.Watch(events, make(chan error))
				require.NoError(t, watchErr)
			}()

			if success := cache.WaitForNamedCacheSync("namespace", w.stopChannel, w.nsInformer.HasSynced); !success {
				require.True(t, success)
			}

			for _, informer := range w.informers {
				success := cache.WaitForCacheSync(w.stopChannel, informer.HasSynced)
				require.True(t, success)
			}

			got, err := w.LoadConfig(context.Background())
			assert.NoError(t, err)

			assert.NotEmpty(t, got.ScrapeConfigs)

			for _, sc := range got.ScrapeConfigs {
				assert.Equal(t, tt.expectedScrape, sc.ScrapeInterval)
			}
			assert.Equal(t, tt.expectedEval, got.GlobalConfig.EvaluationInterval)
		})
	}
}

// getTestPrometheusCRWatcher creates a test instance of PrometheusCRWatcher with fake clients
// and test secrets.
func getTestPrometheusCRWatcher(
	t *testing.T,
	namespace string,
	svcMonitors []*monitoringv1.ServiceMonitor,
	podMonitors []*monitoringv1.PodMonitor,
	probes []*monitoringv1.Probe,
	scrapeConfigs []*promv1alpha1.ScrapeConfig,
	cfg allocatorconfig.Config,
) (*PrometheusCRWatcher, *fcache.FakeControllerSource) {
	mClient := fakemonitoringclient.NewSimpleClientset()
	for _, sm := range svcMonitors {
		if sm != nil {
			_, err := mClient.MonitoringV1().ServiceMonitors(sm.Namespace).Create(context.Background(), sm, metav1.CreateOptions{})
			if err != nil {
				t.Fatal(t, err)
			}
		}
	}
	for _, pm := range podMonitors {
		if pm != nil {
			_, err := mClient.MonitoringV1().PodMonitors(pm.Namespace).Create(context.Background(), pm, metav1.CreateOptions{})
			if err != nil {
				t.Fatal(t, err)
			}
		}
	}
	for _, prb := range probes {
		if prb != nil {
			_, err := mClient.MonitoringV1().Probes(prb.Namespace).Create(context.Background(), prb, metav1.CreateOptions{})
			if err != nil {
				t.Fatal(t, err)
			}
		}
	}

	for _, scc := range scrapeConfigs {
		if scc != nil {
			_, err := mClient.MonitoringV1alpha1().ScrapeConfigs(scc.Namespace).Create(context.Background(), scc, metav1.CreateOptions{})
			if err != nil {
				t.Fatal(t, err)
			}
		}
	}

	k8sClient := fake.NewSimpleClientset()
	_, err := k8sClient.CoreV1().Secrets("test").Create(context.Background(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic-auth",
			Namespace: "test",
		},
		Data: map[string][]byte{"username": []byte("admin"), "password": []byte("password")},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(t, err)
	}
	_, err = k8sClient.CoreV1().Secrets("test").Create(context.Background(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bearer",
			Namespace: "test",
		},
		Data: map[string][]byte{"token": []byte("bearer-token")},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(t, err)
	}

	factory := informers.NewMonitoringInformerFactories(map[string]struct{}{v1.NamespaceAll: {}}, map[string]struct{}{}, mClient, 0, nil)

	informers, err := getTestInformers(factory)
	if err != nil {
		t.Fatal(t, err)
	}

	serviceDiscoveryRole := monitoringv1.ServiceDiscoveryRole("EndpointSlice")

	prom := &monitoringv1.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: monitoringv1.PrometheusSpec{
			CommonPrometheusFields: monitoringv1.CommonPrometheusFields{
				ScrapeInterval:                  monitoringv1.Duration(cfg.PrometheusCR.ScrapeInterval.String()),
				ServiceMonitorSelector:          cfg.PrometheusCR.ServiceMonitorSelector,
				PodMonitorSelector:              cfg.PrometheusCR.PodMonitorSelector,
				ServiceMonitorNamespaceSelector: cfg.PrometheusCR.ServiceMonitorNamespaceSelector,
				PodMonitorNamespaceSelector:     cfg.PrometheusCR.PodMonitorNamespaceSelector,
				ProbeSelector:                   cfg.PrometheusCR.ProbeSelector,
				ProbeNamespaceSelector:          cfg.PrometheusCR.ProbeNamespaceSelector,
				ScrapeConfigSelector:            cfg.PrometheusCR.ScrapeConfigSelector,
				ScrapeConfigNamespaceSelector:   cfg.PrometheusCR.ScrapeConfigNamespaceSelector,
				ScrapeClasses:                   cfg.PrometheusCR.ScrapeClasses,
				ServiceDiscoveryRole:            &serviceDiscoveryRole,
			},
			EvaluationInterval: monitoringv1.Duration(cfg.PrometheusCR.EvaluationInterval.String()),
		},
	}

	promOperatorLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	generator, err := prometheus.NewConfigGenerator(promOperatorLogger, prom, prometheus.WithEndpointSliceSupport(), prometheus.WithInlineTLSConfig())
	if err != nil {
		t.Fatal(t, err)
	}

	store := assets.NewStoreBuilder(k8sClient.CoreV1(), k8sClient.CoreV1())
	promRegisterer := prometheusgoclient.NewRegistry()
	operatorMetrics := operator.NewMetrics(promRegisterer)
	eventRecorderFactoryFactory := operator.NewEventRecorderFactory(false)
	eventRecorderFactory := eventRecorderFactoryFactory(k8sClient, "target-allocator")
	eventRecorder := eventRecorderFactory(prom)

	source := fcache.NewFakeControllerSource()
	source.Add(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}})
	source.Add(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "labellednamespace",
		Labels: map[string]string{
			"label1": "label1",
		}}})

	// create the shared informer and resync every 1s
	nsMonInf := cache.NewSharedInformer(source, &v1.Namespace{}, 1*time.Second).(cache.SharedIndexInformer)

	resourceSelector, err := prometheus.NewResourceSelector(promOperatorLogger, prom, store, nsMonInf, operatorMetrics, eventRecorder)
	require.NoError(t, err)

	return &PrometheusCRWatcher{
		logger:                          slog.Default(),
		kubeMonitoringClient:            mClient,
		k8sClient:                       k8sClient,
		informers:                       informers,
		nsInformer:                      nsMonInf,
		stopChannel:                     make(chan struct{}),
		configGenerator:                 generator,
		podMonitorNamespaceSelector:     cfg.PrometheusCR.PodMonitorNamespaceSelector,
		serviceMonitorNamespaceSelector: cfg.PrometheusCR.ServiceMonitorNamespaceSelector,
		probeNamespaceSelector:          cfg.PrometheusCR.ProbeNamespaceSelector,
		scrapeConfigNamespaceSelector:   cfg.PrometheusCR.ScrapeConfigNamespaceSelector,
		resourceSelector:                resourceSelector,
		store:                           store,
		prometheusCR:                    prom,
	}, source

}

// Remove relable configs fields from scrape configs for testing,
// since these are mutated and tested down the line with the hook(s).
func sanitizeScrapeConfigsForTest(scs []*promconfig.ScrapeConfig) {
	for _, sc := range scs {
		sc.RelabelConfigs = nil
		sc.MetricRelabelConfigs = nil
	}
}

// getTestInformers creates informers for testing without CRD availability checks.
func getTestInformers(factory informers.FactoriesForNamespaces) (map[string]*informers.ForResource, error) {
	informersMap := make(map[string]*informers.ForResource)

	// Create ServiceMonitor informers
	serviceMonitorInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ServiceMonitorName))
	if err != nil {
		return nil, err
	}
	informersMap[monitoringv1.ServiceMonitorName] = serviceMonitorInformers

	// Create PodMonitor informers
	podMonitorInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.PodMonitorName))
	if err != nil {
		return nil, err
	}
	informersMap[monitoringv1.PodMonitorName] = podMonitorInformers

	// Create Probe informers
	probeInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ProbeName))
	if err != nil {
		return nil, err
	}
	informersMap[monitoringv1.ProbeName] = probeInformers

	// Create ScrapeConfig informers
	scrapeConfigInformers, err := informers.NewInformersForResource(factory, promv1alpha1.SchemeGroupVersion.WithResource(promv1alpha1.ScrapeConfigName))
	if err != nil {
		return nil, err
	}
	informersMap[promv1alpha1.ScrapeConfigName] = scrapeConfigInformers

	return informersMap, nil
}

// TestCRDAvailabilityChecks tests the CRDs' availability.
func TestCRDAvailabilityChecks(t *testing.T) {
	tests := []struct {
		name          string
		availableCRDs []string
		expectedCRDs  []string
	}{
		{
			name:          "ServiceMonitor available",
			availableCRDs: []string{"servicemonitors"},
			expectedCRDs:  []string{"servicemonitors"},
		},
		{
			name:          "All CRDs available",
			availableCRDs: []string{"servicemonitors", "podmonitors", "probes", "scrapeconfigs"},
			expectedCRDs:  []string{"servicemonitors", "podmonitors", "probes", "scrapeconfigs"},
		},
		{
			name:          "No CRDs available",
			availableCRDs: []string{},
			expectedCRDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake discovery client
			fakeDiscovery := &fakediscovery.FakeDiscovery{
				Fake: &fake.NewSimpleClientset().Fake,
			}

			// Set up resources
			fakeDiscovery.Resources = []*metav1.APIResourceList{}

			// Add v1 resources
			v1Resources := &metav1.APIResourceList{
				GroupVersion: "monitoring.coreos.com/v1",
				APIResources: []metav1.APIResource{},
			}
			for _, crd := range tt.availableCRDs {
				if crd == "servicemonitors" || crd == "podmonitors" || crd == "probes" {
					v1Resources.APIResources = append(v1Resources.APIResources, metav1.APIResource{
						Name: crd,
					})
				}
			}
			fakeDiscovery.Resources = append(fakeDiscovery.Resources, v1Resources)

			// Add v1alpha1 resources
			v1alpha1Resources := &metav1.APIResourceList{
				GroupVersion: "monitoring.coreos.com/v1alpha1",
				APIResources: []metav1.APIResource{},
			}
			for _, crd := range tt.availableCRDs {
				if crd == "scrapeconfigs" {
					v1alpha1Resources.APIResources = append(v1alpha1Resources.APIResources, metav1.APIResource{
						Name: crd,
					})
				}
			}
			fakeDiscovery.Resources = append(fakeDiscovery.Resources, v1alpha1Resources)

			// Test each CRD availability
			for _, crd := range []string{"servicemonitors", "podmonitors", "probes", "scrapeconfigs"} {
				available, err := checkCRDAvailability(fakeDiscovery, crd)
				require.NoError(t, err)

				expected := false
				for _, expectedCRD := range tt.expectedCRDs {
					if crd == expectedCRD {
						expected = true
						break
					}
				}

				assert.Equal(t, expected, available, "CRD %s availability should match expectation", crd)
			}
		})
	}
}
