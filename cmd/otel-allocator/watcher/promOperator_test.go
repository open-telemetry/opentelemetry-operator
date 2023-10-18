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

package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/go-kit/log"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	fakemonitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/fake"
	"github.com/prometheus-operator/prometheus-operator/pkg/informers"
	"github.com/prometheus-operator/prometheus-operator/pkg/prometheus"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	kubeDiscovery "github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		serviceMonitor *monitoringv1.ServiceMonitor
		podMonitor     *monitoringv1.PodMonitor
		want           *promconfig.Config
		wantErr        bool
	}{
		{
			name: "simple test",
			serviceMonitor: &monitoringv1.ServiceMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple",
					Namespace: "test",
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					JobLabel: "test",
					Endpoints: []monitoringv1.Endpoint{
						{
							Port: "web",
						},
					},
				},
			},
			podMonitor: &monitoringv1.PodMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple",
					Namespace: "test",
				},
				Spec: monitoringv1.PodMonitorSpec{
					JobLabel: "test",
					PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
						{
							Port: "web",
						},
					},
				},
			},
			want: &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/test/simple/0",
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "endpointslice",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{"test"},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig: config.DefaultHTTPClientConfig,
					},
					{
						JobName:         "podMonitor/test/simple/0",
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "pod",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{"test"},
									IncludeOwnNamespace: false,
								},
								HTTPClientConfig: config.DefaultHTTPClientConfig,
							},
						},
						HTTPClientConfig: config.DefaultHTTPClientConfig,
					},
				},
			},
		},
		{
			name: "basic auth (serviceMonitor)",
			serviceMonitor: &monitoringv1.ServiceMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "auth",
					Namespace: "test",
				},
				Spec: monitoringv1.ServiceMonitorSpec{
					JobLabel: "auth",
					Endpoints: []monitoringv1.Endpoint{
						{
							Port: "web",
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
			want: &promconfig.Config{
				GlobalConfig: promconfig.GlobalConfig{},
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "serviceMonitor/test/auth/0",
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "endpointslice",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{"test"},
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
					},
				},
			},
		},
		{
			name: "bearer token (podMonitor)",
			podMonitor: &monitoringv1.PodMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bearer",
					Namespace: "test",
				},
				Spec: monitoringv1.PodMonitorSpec{
					JobLabel: "bearer",
					PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
						{
							Port: "web",
							BearerTokenSecret: v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "bearer",
								},
								Key: "token",
							},
						},
					},
				},
			},
			want: &promconfig.Config{
				GlobalConfig: promconfig.GlobalConfig{},
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName:         "podMonitor/test/bearer/0",
						ScrapeInterval:  model.Duration(30 * time.Second),
						ScrapeTimeout:   model.Duration(10 * time.Second),
						HonorTimestamps: true,
						HonorLabels:     false,
						Scheme:          "http",
						MetricsPath:     "/metrics",
						ServiceDiscoveryConfigs: []discovery.Config{
							&kubeDiscovery.SDConfig{
								Role: "pod",
								NamespaceDiscovery: kubeDiscovery.NamespaceDiscovery{
									Names:               []string{"test"},
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
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := getTestPrometheusCRWatcher(t, tt.serviceMonitor, tt.podMonitor)
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

func TestRateLimit(t *testing.T) {
	var err error
	serviceMonitor := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple",
			Namespace: "test",
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
	eventInterval := 5 * time.Millisecond

	w := getTestPrometheusCRWatcher(t, nil, nil)
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
	}, eventInterval*2, time.Millisecond)

	// it's difficult to measure the rate precisely
	// what we do, is send two updates, and then assert that the elapsed time is between eventInterval and 3*eventInterval
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
	}, eventInterval*2, time.Millisecond)
	_, err = w.kubeMonitoringClient.MonitoringV1().ServiceMonitors("test").Update(context.Background(), serviceMonitor, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		select {
		case <-events:
			return true
		default:
			return false
		}
	}, eventInterval*2, time.Millisecond)
	elapsedTime := time.Since(startTime)
	assert.Less(t, eventInterval, elapsedTime)
	assert.GreaterOrEqual(t, eventInterval*3, elapsedTime)

}

// getTestPrometheuCRWatcher creates a test instance of PrometheusCRWatcher with fake clients
// and test secrets.
func getTestPrometheusCRWatcher(t *testing.T, sm *monitoringv1.ServiceMonitor, pm *monitoringv1.PodMonitor) *PrometheusCRWatcher {
	mClient := fakemonitoringclient.NewSimpleClientset()
	if sm != nil {
		_, err := mClient.MonitoringV1().ServiceMonitors("test").Create(context.Background(), sm, metav1.CreateOptions{})
		if err != nil {
			t.Fatal(t, err)
		}
	}
	if pm != nil {
		_, err := mClient.MonitoringV1().PodMonitors("test").Create(context.Background(), pm, metav1.CreateOptions{})
		if err != nil {
			t.Fatal(t, err)
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
	informers, err := getInformers(factory)
	if err != nil {
		t.Fatal(t, err)
	}

	prom := &monitoringv1.Prometheus{
		Spec: monitoringv1.PrometheusSpec{
			CommonPrometheusFields: monitoringv1.CommonPrometheusFields{
				ScrapeInterval: monitoringv1.Duration("30s"),
			},
		},
	}

	generator, err := prometheus.NewConfigGenerator(log.NewNopLogger(), prom, true)
	if err != nil {
		t.Fatal(t, err)
	}

	return &PrometheusCRWatcher{
		kubeMonitoringClient:   mClient,
		k8sClient:              k8sClient,
		informers:              informers,
		configGenerator:        generator,
		serviceMonitorSelector: getSelector(nil),
		podMonitorSelector:     getSelector(nil),
		stopChannel:            make(chan struct{}),
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
