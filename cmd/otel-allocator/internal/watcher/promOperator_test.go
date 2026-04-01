// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package watcher

// Tests in this file use testing/synctest to make async behavior deterministic.
//
// Tests that call Watch() use time.Sleep(watchSyncDuration) before synctest.Wait()
// to let the informer cache sync complete. This is necessary because Watch()
// calls WaitForNamedCacheSync for each informer sequentially, and each sync poll
// involves mutex operations inside the k8s informer machinery. Mutexes are not
// "durably blocking" in synctest, so synctest.Wait() can return before the
// informers finish syncing, causing the 15s WaitForNamedCacheSync timeout to
// fire. Advancing the fake clock by watchSyncDuration gives the ~6 informers
// enough 100ms poll ticks (client-go's syncedPollPeriod) to each observe that
// their cache has synced.

import (
	"context"
	"log/slog"
	"os"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus-operator/prometheus-operator/pkg/assets"
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	metadatafake "k8s.io/client-go/metadata/fake"
	"k8s.io/client-go/tools/cache"
	fcache "k8s.io/client-go/tools/cache/testing"
	"k8s.io/utils/ptr"

	allocatorconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

// watchSyncDuration is the fake-clock time we advance after starting Watch() to
// let all informer caches sync. Watch() syncs ~6 informers sequentially, each
// requiring at least one 100ms poll tick, so 1s gives comfortable headroom.
const watchSyncDuration = time.Second

// fakeInformLister wraps a SharedIndexInformer to satisfy the informers.InformLister interface.
type fakeInformLister struct {
	informer cache.SharedIndexInformer
	gr       schema.GroupResource
}

func (f *fakeInformLister) Informer() cache.SharedIndexInformer { return f.informer }
func (f *fakeInformLister) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.informer.GetIndexer(), f.gr)
}

// fakeFactoriesForNamespaces implements informers.FactoriesForNamespaces using FakeControllerSource.
type fakeFactoriesForNamespaces struct {
	sources    map[schema.GroupVersionResource]*fcache.FakeControllerSource
	exemplars  map[schema.GroupVersionResource]runtime.Object
	namespaces sets.Set[string]
}

func (f *fakeFactoriesForNamespaces) Namespaces() sets.Set[string] { return f.namespaces }

func (f *fakeFactoriesForNamespaces) ForResource(_ string, resource schema.GroupVersionResource) (informers.InformLister, error) {
	source, ok := f.sources[resource]
	if !ok {
		source = fcache.NewFakeControllerSource()
		f.sources[resource] = source
	}
	exemplar := f.exemplars[resource]
	inf := cache.NewSharedIndexInformer(source, exemplar, 1*time.Second,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	return &fakeInformLister{
		informer: inf,
		gr:       resource.GroupResource(),
	}, nil
}

// testWatcher bundles a PrometheusCRWatcher with fake sources for use in tests.
// Tests access only the fields they need.
type testWatcher struct {
	*PrometheusCRWatcher
	NamespaceSource      *fcache.FakeControllerSource
	ServiceMonitorSource *fcache.FakeControllerSource
	PodMonitorSource     *fcache.FakeControllerSource
	ProbeSource          *fcache.FakeControllerSource
	ScrapeConfigSource   *fcache.FakeControllerSource
	MetadataClient       *metadatafake.FakeMetadataClient
}

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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
								HTTPConfigWithProxyAndTLSFiles: monitoringv1.HTTPConfigWithProxyAndTLSFiles{
									HTTPConfigWithTLSFiles: monitoringv1.HTTPConfigWithTLSFiles{
										HTTPConfigWithoutTLS: monitoringv1.HTTPConfigWithoutTLS{
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
						ExtraScrapeMetrics:             ptr.To(false),
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
								HTTPConfigWithProxy: monitoringv1.HTTPConfigWithProxy{
									HTTPConfig: monitoringv1.HTTPConfig{
										HTTPConfigWithoutTLS: monitoringv1.HTTPConfigWithoutTLS{
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
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
						ExtraScrapeMetrics:             ptr.To(false),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tw := newTestWatcher(t, tt.cfg)
				for _, sm := range tt.serviceMonitors {
					tw.ServiceMonitorSource.Add(sm)
				}
				for _, pm := range tt.podMonitors {
					tw.PodMonitorSource.Add(pm)
				}
				for _, prb := range tt.probes {
					tw.ProbeSource.Add(prb)
				}
				for _, sc := range tt.scrapeConfigs {
					tw.ScrapeConfigSource.Add(sc)
				}

				// Start namespace informers in order to populate cache.
				go tw.nsInformer.Run(tw.stopChannel)
				synctest.Wait()

				for _, informer := range tw.informers {
					// Start informers in order to populate cache.
					informer.Start(tw.stopChannel)
				}
				synctest.Wait()

				got, err := tw.LoadConfig(context.Background())
				assert.NoError(t, err)

				sanitizeScrapeConfigsForTest(got.ScrapeConfigs)
				assert.Equal(t, tt.want.ScrapeConfigs, got.ScrapeConfigs)

				close(tw.stopChannel)
				synctest.Wait()
			})
		})
	}
}

func TestNamespaceLabelUpdate(t *testing.T) {
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
				ExtraScrapeMetrics:             ptr.To(false),
			},
		},
	}

	want_after := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{},
	}

	synctest.Test(t, func(t *testing.T) {
		tw := newTestWatcher(t, cfg)
		for _, pm := range podMonitors {
			tw.PodMonitorSource.Add(pm)
		}
		events := make(chan Event, 1)
		eventInterval := 5 * time.Millisecond

		defer tw.Close()
		tw.eventInterval = eventInterval

		go func() {
			watchErr := tw.Watch(events, make(chan error))
			require.NoError(t, watchErr)
		}()
		// Advance time past the informer sync polling period to let Watch complete setup.
		time.Sleep(watchSyncDuration)
		synctest.Wait()

		got, err := tw.LoadConfig(context.Background())
		assert.NoError(t, err)

		sanitizeScrapeConfigsForTest(got.ScrapeConfigs)
		assert.Equal(t, want_before.ScrapeConfigs, got.ScrapeConfigs)

		tw.NamespaceSource.Modify(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: "labellednamespace",
			Labels: map[string]string{
				"label2": "label2",
			},
		}})
		synctest.Wait()
		time.Sleep(eventInterval)
		synctest.Wait()

		got, err = tw.LoadConfig(context.Background())
		assert.NoError(t, err)

		sanitizeScrapeConfigsForTest(got.ScrapeConfigs)
		assert.Equal(t, want_after.ScrapeConfigs, got.ScrapeConfigs)
	})
}

// TestSecretInformerUpdatesStore verifies that when a secret is updated through the informer,
// the asset store is automatically updated and LoadConfig reflects the new values.
func TestSecretInformerUpdatesStore(t *testing.T) {
	namespace := "test"
	portName := "web"

	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "auth",
			Namespace: namespace,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			JobLabel: "auth",
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: portName,
					HTTPConfigWithProxyAndTLSFiles: monitoringv1.HTTPConfigWithProxyAndTLSFiles{
						HTTPConfigWithTLSFiles: monitoringv1.HTTPConfigWithTLSFiles{
							HTTPConfigWithoutTLS: monitoringv1.HTTPConfigWithoutTLS{
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
					},
				},
			},
			Selector: metav1.LabelSelector{},
		},
	}

	cfg := allocatorconfig.Config{
		PrometheusCR: allocatorconfig.PrometheusCRConfig{
			ServiceMonitorSelector: &metav1.LabelSelector{},
			PodMonitorSelector:     &metav1.LabelSelector{},
		},
	}

	synctest.Test(t, func(t *testing.T) {
		tw := newTestWatcher(t, cfg)
		tw.ServiceMonitorSource.Add(sm)
		defer tw.Close()

		// Add initial secret to the metadata client's tracker so the informer can watch it
		secretGVR := v1.SchemeGroupVersion.WithResource(string(v1.ResourceSecrets))
		initialSecretMeta := &metav1.PartialObjectMetadata{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "basic-auth",
				Namespace:       namespace,
				ResourceVersion: "1",
			},
		}
		err := tw.MetadataClient.Tracker().Add(initialSecretMeta)
		require.NoError(t, err)

		events := make(chan Event, 1)
		errors := make(chan error, 1)
		eventInterval := 5 * time.Millisecond
		tw.eventInterval = eventInterval

		// Start Watch in a goroutine - this registers the secret informer event handlers
		go func() {
			watchErr := tw.Watch(events, errors)
			require.NoError(t, watchErr)
		}()

		// Advance time past the informer sync polling period, then wait for the first event.
		time.Sleep(watchSyncDuration)
		synctest.Wait()
		<-events

		// Initial config should reflect the original secret values.
		got, err := tw.LoadConfig(context.Background())
		require.NoError(t, err)
		require.NotEmpty(t, got.ScrapeConfigs)

		var smSC *promconfig.ScrapeConfig
		for _, sc := range got.ScrapeConfigs {
			if sc.JobName == "serviceMonitor/test/auth/0" {
				smSC = sc
				break
			}
		}
		require.NotNil(t, smSC)
		require.NotNil(t, smSC.HTTPClientConfig.BasicAuth)
		assert.Equal(t, "admin", smSC.HTTPClientConfig.BasicAuth.Username)
		assert.Equal(t, config.Secret("password"), smSC.HTTPClientConfig.BasicAuth.Password)

		// Update the k8sClient first (this is what the informer's UpdateFunc reads from)
		updatedSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "basic-auth",
				Namespace:       namespace,
				ResourceVersion: "2",
			},
			Data: map[string][]byte{
				"username": []byte("newadmin"),
				"password": []byte("newpassword"),
			},
		}
		_, err = tw.k8sClient.CoreV1().Secrets(namespace).Update(context.Background(), updatedSecret, metav1.UpdateOptions{})
		require.NoError(t, err)

		// Update the metadata client's tracker to trigger the informer's UpdateFunc
		updatedSecretMeta := &metav1.PartialObjectMetadata{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "basic-auth",
				Namespace:       namespace,
				ResourceVersion: "2",
			},
		}
		err = tw.MetadataClient.Tracker().Update(secretGVR, updatedSecretMeta, namespace)
		require.NoError(t, err)

		// Wait for the informer event to be processed
		synctest.Wait()
		time.Sleep(eventInterval)
		synctest.Wait()

		got, err = tw.LoadConfig(context.Background())
		require.NoError(t, err)

		smSC = nil
		for _, sc := range got.ScrapeConfigs {
			if sc.JobName == "serviceMonitor/test/auth/0" {
				smSC = sc
				break
			}
		}
		require.NotNil(t, smSC)
		require.NotNil(t, smSC.HTTPClientConfig.BasicAuth)
		assert.Equal(t, "newadmin", smSC.HTTPClientConfig.BasicAuth.Username)
		assert.Equal(t, config.Secret("newpassword"), smSC.HTTPClientConfig.BasicAuth.Password)
	})
}

func TestRateLimit(t *testing.T) {
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
	synctest.Test(t, func(t *testing.T) {
		events := make(chan Event, 1)
		eventInterval := 500 * time.Millisecond
		cfg := allocatorconfig.Config{}

		tw := newTestWatcher(t, cfg)
		defer tw.Close()
		tw.eventInterval = eventInterval

		go func() {
			watchErr := tw.Watch(events, make(chan error))
			require.NoError(t, watchErr)
		}()
		time.Sleep(watchSyncDuration)
		synctest.Wait()

		tw.ServiceMonitorSource.Add(serviceMonitor)
		synctest.Wait()
		time.Sleep(eventInterval)
		synctest.Wait()
		<-events

		// Send two updates and verify that the elapsed time is at least eventInterval
		startTime := time.Now()
		tw.ServiceMonitorSource.Modify(serviceMonitor)
		synctest.Wait()
		time.Sleep(eventInterval)
		synctest.Wait()
		<-events

		tw.ServiceMonitorSource.Modify(serviceMonitor)
		synctest.Wait()
		time.Sleep(eventInterval)
		synctest.Wait()
		<-events

		elapsedTime := time.Since(startTime)
		assert.Less(t, eventInterval, elapsedTime)
	})
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
			synctest.Test(t, func(t *testing.T) {
				tw := newTestWatcher(t, tt.cfg)
				for _, sm := range tt.serviceMonitors {
					tw.ServiceMonitorSource.Add(sm)
				}
				defer tw.Close()

				events := make(chan Event, 1)
				eventInterval := 5 * time.Millisecond
				tw.eventInterval = eventInterval

				go func() {
					watchErr := tw.Watch(events, make(chan error))
					require.NoError(t, watchErr)
				}()
				time.Sleep(watchSyncDuration)
				synctest.Wait()

				got, err := tw.LoadConfig(context.Background())
				assert.NoError(t, err)

				assert.NotEmpty(t, got.ScrapeConfigs)

				for _, sc := range got.ScrapeConfigs {
					assert.Equal(t, tt.expectedScrape, sc.ScrapeInterval)
				}
				assert.Equal(t, tt.expectedEval, got.GlobalConfig.EvaluationInterval)
			})
		})
	}
}

// newTestWatcher creates a testWatcher with fake sources for the given config.
// Callers add resources to the returned sources (e.g. tw.ServiceMonitorSource.Add)
// before starting informers.
func newTestWatcher(t *testing.T, cfg allocatorconfig.Config) *testWatcher {
	t.Helper()

	k8sClient := fake.NewClientset()
	_, err := k8sClient.CoreV1().Secrets("test").Create(context.Background(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic-auth",
			Namespace: "test",
		},
		Data: map[string][]byte{"username": []byte("admin"), "password": []byte("password")},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = k8sClient.CoreV1().Secrets("test").Create(context.Background(), &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bearer",
			Namespace: "test",
		},
		Data: map[string][]byte{"token": []byte("bearer-token")},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	// newSource creates a FakeControllerSource and registers cleanup.
	newSource := func() *fcache.FakeControllerSource {
		s := fcache.NewFakeControllerSource()
		t.Cleanup(func() { s.Broadcaster.Shutdown() })
		return s
	}

	smSource := newSource()
	pmSource := newSource()
	probeSource := newSource()
	scSource := newSource()
	nsSource := newSource()

	// Build fake factories backed by the sources.
	type gvrInfo struct {
		gvr      schema.GroupVersionResource
		source   *fcache.FakeControllerSource
		exemplar runtime.Object
	}
	resources := []gvrInfo{
		{monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ServiceMonitorName), smSource, &monitoringv1.ServiceMonitor{}},
		{monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.PodMonitorName), pmSource, &monitoringv1.PodMonitor{}},
		{monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ProbeName), probeSource, &monitoringv1.Probe{}},
		{promv1alpha1.SchemeGroupVersion.WithResource(promv1alpha1.ScrapeConfigName), scSource, &promv1alpha1.ScrapeConfig{}},
	}

	sources := make(map[schema.GroupVersionResource]*fcache.FakeControllerSource, len(resources))
	exemplars := make(map[schema.GroupVersionResource]runtime.Object, len(resources))
	for _, r := range resources {
		sources[r.gvr] = r.source
		exemplars[r.gvr] = r.exemplar
	}

	fakeFactory := &fakeFactoriesForNamespaces{
		sources:    sources,
		exemplars:  exemplars,
		namespaces: sets.New[string](v1.NamespaceAll),
	}

	// Create fake metadata client for secret informer.
	mdScheme := metadatafake.NewTestScheme()
	_ = metav1.AddMetaToScheme(mdScheme)
	mdClient := metadatafake.NewSimpleMetadataClient(mdScheme)
	metadataFactory := informers.NewMetadataInformerFactory(map[string]struct{}{v1.NamespaceAll: {}}, map[string]struct{}{}, mdClient, 1*time.Second, nil)

	// Build informers via a for-range loop over name→GVR.
	informerDefs := map[string]schema.GroupVersionResource{
		monitoringv1.ServiceMonitorName: monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ServiceMonitorName),
		monitoringv1.PodMonitorName:     monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.PodMonitorName),
		monitoringv1.ProbeName:          monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ProbeName),
		promv1alpha1.ScrapeConfigName:   promv1alpha1.SchemeGroupVersion.WithResource(promv1alpha1.ScrapeConfigName),
	}
	informersMap := make(map[string]*informers.ForResource, len(informerDefs)+1)
	for name, gvr := range informerDefs {
		inf, infErr := informers.NewInformersForResource(fakeFactory, gvr)
		require.NoError(t, infErr)
		informersMap[name] = inf
	}
	// Secret informer from metadata factory.
	secretInformer, err := informers.NewInformersForResourceWithTransform(
		metadataFactory,
		v1.SchemeGroupVersion.WithResource(string(v1.ResourceSecrets)),
		informers.PartialObjectMetadataStrip(operator.SecretGVK()),
	)
	require.NoError(t, err)
	if secretInformer != nil {
		informersMap[string(v1.ResourceSecrets)] = secretInformer
	}

	serviceDiscoveryRole := monitoringv1.ServiceDiscoveryRole("EndpointSlice")

	prom := &monitoringv1.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
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
	require.NoError(t, err)

	store := assets.NewStoreBuilder(k8sClient.CoreV1(), k8sClient.CoreV1())
	promRegisterer := prometheusgoclient.NewRegistry()
	operatorMetrics := operator.NewMetrics(promRegisterer)
	eventRecorder := operator.NewFakeRecorder(10, prom)

	nsSource.Add(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}})
	nsSource.Add(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "labellednamespace",
		Labels: map[string]string{
			"label1": "label1",
		},
	}})

	// create the shared informer and resync every 1s
	nsMonInf := cache.NewSharedInformer(nsSource, &v1.Namespace{}, 1*time.Second).(cache.SharedIndexInformer)

	resourceSelector, err := prometheus.NewResourceSelector(promOperatorLogger, prom, store, nsMonInf, operatorMetrics, eventRecorder)
	require.NoError(t, err)

	return &testWatcher{
		PrometheusCRWatcher: &PrometheusCRWatcher{
			logger:                          slog.Default(),
			k8sClient:                       k8sClient,
			informers:                       informersMap,
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
		},
		NamespaceSource:      nsSource,
		ServiceMonitorSource: smSource,
		PodMonitorSource:     pmSource,
		ProbeSource:          probeSource,
		ScrapeConfigSource:   scSource,
		MetadataClient:       mdClient,
	}
}

// Remove relable configs fields from scrape configs for testing,
// since these are mutated and tested down the line with the hook(s).
func sanitizeScrapeConfigsForTest(scs []*promconfig.ScrapeConfig) {
	for _, sc := range scs {
		sc.RelabelConfigs = nil
		sc.MetricRelabelConfigs = nil
	}
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
				Fake: &fake.NewClientset().Fake,
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

				expected := slices.Contains(tt.expectedCRDs, crd)

				assert.Equal(t, expected, available, "CRD %s availability should match expectation", crd)
			}
		})
	}
}
