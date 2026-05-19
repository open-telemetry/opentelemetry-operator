// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"log/slog"
	"os"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/fake"
	metadatafake "k8s.io/client-go/metadata/fake"
	"k8s.io/client-go/tools/cache"
	fcache "k8s.io/client-go/tools/cache/testing"

	allocatorconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

// TestDenyFSAccessThroughSMsBasic verifies that when DenyFSAccessThroughSMs
// is enabled, ServiceMonitors with authorization.credentials_file are rejected
// and the resulting scrape configs are cleared.
func TestDenyFSAccessThroughSMsBasic(t *testing.T) {
	namespace := "test"
	portName := "web"

	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "attacker-sm",
			Namespace: namespace,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			JobLabel: "test",
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: portName,
					HTTPConfigWithProxyAndTLSFiles: monitoringv1.HTTPConfigWithProxyAndTLSFiles{
						HTTPConfigWithTLSFiles: monitoringv1.HTTPConfigWithTLSFiles{
							HTTPConfigWithoutTLS: monitoringv1.HTTPConfigWithoutTLS{
								Authorization: &monitoringv1.SafeAuthorization{
									Type: "Bearer",
									CredentialsFile: func() *string {
										s := "/var/run/secrets/kubernetes.io/serviceaccount/token"
										return &s
									}(),
								},
							},
						},
					},
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "attacker"},
			},
		},
	}

	cfg := allocatorconfig.Config{
		PrometheusCR: allocatorconfig.PrometheusCRConfig{
			ServiceMonitorSelector: &metav1.LabelSelector{},
			DenyFSAccessThroughSMs: true,
		},
	}

	synctest.Test(t, func(t *testing.T) {
		tw := newTestWatcherWithDenyFlag(t, cfg)
		tw.ServiceMonitorSource.Add(sm)
		defer tw.Close()

		time.Sleep(watchSyncDuration)
		synctest.Wait()

		got, err := tw.LoadConfig(context.Background())
		require.NoError(t, err)

		assert.Empty(t, got.ScrapeConfigs,
			"expected no scrape configs when DenyFSAccessThroughSMs rejects credentials_file")
	})
}

// TestDenyFSAccessThroughSMsDisabled verifies that when DenyFSAccessThroughSMs
// is NOT enabled, ServiceMonitors without file references are allowed through.
func TestDenyFSAccessThroughSMsDisabled(t *testing.T) {
	namespace := "test"
	portName := "web"

	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-sm",
			Namespace: namespace,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			JobLabel: "test",
			Endpoints: []monitoringv1.Endpoint{
				{Port: portName},
			},
		},
	}

	cfg := allocatorconfig.Config{
		PrometheusCR: allocatorconfig.PrometheusCRConfig{
			ServiceMonitorSelector: &metav1.LabelSelector{},
			DenyFSAccessThroughSMs: false,
		},
	}

	synctest.Test(t, func(t *testing.T) {
		tw := newTestWatcherWithDenyFlag(t, cfg)
		tw.ServiceMonitorSource.Add(sm)
		defer tw.Close()

		time.Sleep(watchSyncDuration)
		synctest.Wait()

		got, err := tw.LoadConfig(context.Background())
		require.NoError(t, err)

		assert.NotEmpty(t, got.ScrapeConfigs,
			"expected scrape configs when DenyFSAccessThroughSMs is disabled")
	})
}

// TestDenyFSAccessThroughSMsWithBearerTokenFile verifies the exact attack
// scenario: a tenant sets bearerTokenFile to the Collector's service account
// token path, and the guard should reject it.
func TestDenyFSAccessThroughSMsWithBearerTokenFile(t *testing.T) {
	namespace := "test"

	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "attacker-bearer-sm",
			Namespace: namespace,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			JobLabel: "test",
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "web",
					HTTPConfigWithProxyAndTLSFiles: monitoringv1.HTTPConfigWithProxyAndTLSFiles{
						HTTPConfigWithTLSFiles: monitoringv1.HTTPConfigWithTLSFiles{
							HTTPConfigWithoutTLS: monitoringv1.HTTPConfigWithoutTLS{
								Authorization: &monitoringv1.SafeAuthorization{
									Type: "Bearer",
									CredentialsFile: func() *string {
										s := "/var/run/secrets/kubernetes.io/serviceaccount/token"
										return &s
									}(),
								},
							},
						},
					},
				},
			},
		},
	}

	cfg := allocatorconfig.Config{
		PrometheusCR: allocatorconfig.PrometheusCRConfig{
			ServiceMonitorSelector: &metav1.LabelSelector{},
			DenyFSAccessThroughSMs: true,
		},
	}

	synctest.Test(t, func(t *testing.T) {
		tw := newTestWatcherWithDenyFlag(t, cfg)
		tw.ServiceMonitorSource.Add(sm)
		defer tw.Close()

		time.Sleep(watchSyncDuration)
		synctest.Wait()

		got, err := tw.LoadConfig(context.Background())
		require.NoError(t, err)

		assert.Empty(t, got.ScrapeConfigs,
			"expected no scrape configs when credentials_file points to SA token")
	})
}

// TestDenyFSAccessThroughSMsNormalSMsWithoutFileReferences verifies that
// ServiceMonitors without file references (credentials_file, caFile, etc.)
// are allowed through even when DenyFSAccessThroughSMs is enabled.
func TestDenyFSAccessThroughSMsNormalSMsWithoutFileReferences(t *testing.T) {
	namespace := "test"

	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "normal-sm",
			Namespace: namespace,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			JobLabel: "test",
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "web",
					HTTPConfigWithProxyAndTLSFiles: monitoringv1.HTTPConfigWithProxyAndTLSFiles{
						HTTPConfigWithTLSFiles: monitoringv1.HTTPConfigWithTLSFiles{
							HTTPConfigWithoutTLS: monitoringv1.HTTPConfigWithoutTLS{
								BasicAuth: &monitoringv1.BasicAuth{
									Username: &v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: "secret"},
										Key:                  "user",
									},
									Password: &v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: "secret"},
										Key:                  "pass",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	cfg := allocatorconfig.Config{
		PrometheusCR: allocatorconfig.PrometheusCRConfig{
			ServiceMonitorSelector: &metav1.LabelSelector{},
			DenyFSAccessThroughSMs: true,
		},
	}

	synctest.Test(t, func(t *testing.T) {
		tw := newTestWatcherWithDenyFlag(t, cfg)
		tw.ServiceMonitorSource.Add(sm)
		defer tw.Close()

		time.Sleep(watchSyncDuration)
		synctest.Wait()

		got, err := tw.LoadConfig(context.Background())
		require.NoError(t, err)

		assert.NotEmpty(t, got.ScrapeConfigs,
			"expected scrape configs for ServiceMonitor without file references")
	})
}

// newTestWatcherWithDenyFlag creates a testWatcher with the deny flag set
// to match cfg.PrometheusCR.DenyFSAccessThroughSMs.
func newTestWatcherWithDenyFlag(t *testing.T, cfg allocatorconfig.Config) *testWatcher {
	t.Helper()

	k8sClient := fake.NewClientset()

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

	mdScheme := metadatafake.NewTestScheme()
	_ = metav1.AddMetaToScheme(mdScheme)
	mdClient := metadatafake.NewSimpleMetadataClient(mdScheme)
	metadataFactory := informers.NewMetadataInformerFactory(
		map[string]struct{}{v1.NamespaceAll: {}},
		map[string]struct{}{},
		mdClient, 1*time.Second, nil,
	)

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
		if inf != nil {
			informersMap[name] = inf
		}
	}
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
				ServiceMonitorSelector:          cfg.PrometheusCR.ServiceMonitorSelector,
				PodMonitorSelector:              cfg.PrometheusCR.PodMonitorSelector,
				ServiceMonitorNamespaceSelector: cfg.PrometheusCR.ServiceMonitorNamespaceSelector,
				PodMonitorNamespaceSelector:     cfg.PrometheusCR.PodMonitorNamespaceSelector,
				ScrapeConfigSelector:            cfg.PrometheusCR.ScrapeConfigSelector,
				ScrapeConfigNamespaceSelector:   cfg.PrometheusCR.ScrapeConfigNamespaceSelector,
				ProbeSelector:                   cfg.PrometheusCR.ProbeSelector,
				ProbeNamespaceSelector:          cfg.PrometheusCR.ProbeNamespaceSelector,
				ScrapeClasses:                   cfg.PrometheusCR.ScrapeClasses,
				ServiceDiscoveryRole:            &serviceDiscoveryRole,
			},
			EvaluationInterval: monitoringv1.Duration(cfg.PrometheusCR.EvaluationInterval.String()),
		},
	}

	promOperatorLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	generator, err := prometheus.NewConfigGenerator(promOperatorLogger, prom,
		prometheus.WithEndpointSliceSupport(),
		prometheus.WithInlineTLSConfig())
	require.NoError(t, err)

	store := assets.NewStoreBuilder(k8sClient.CoreV1(), k8sClient.CoreV1())
	promRegisterer := prometheusgoclient.NewRegistry()
	operatorMetrics := operator.NewMetrics(promRegisterer)
	eventRecorder := operator.NewFakeRecorder(10, prom)

	nsSource.Add(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}})

	nsMonInf := cache.NewSharedInformer(nsSource, &v1.Namespace{}, 1*time.Second).(cache.SharedIndexInformer)

	resourceSelector, err := prometheus.NewResourceSelector(
		promOperatorLogger, prom, store, nsMonInf, operatorMetrics, eventRecorder)
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
			denyFSAccessThroughSMs:          cfg.PrometheusCR.DenyFSAccessThroughSMs,
		},
		NamespaceSource:      nsSource,
		ServiceMonitorSource: smSource,
		PodMonitorSource:     pmSource,
		ProbeSource:          probeSource,
		ScrapeConfigSource:   scSource,
		MetadataClient:       mdClient,
	}
}
