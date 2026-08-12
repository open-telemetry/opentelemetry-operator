// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	operatorconfig "github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/controllers"
)

func TestClusterObservabilityNoOpReconcilePreservesStructuralDefaults(t *testing.T) {
	ctx := context.Background()
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "co-api-defaults-" + utilrand.String(8)}}
	require.NoError(t, k8sClient.Create(ctx, namespace))
	key := types.NamespacedName{Name: "cluster-obs", Namespace: namespace.Name}
	instance := &v1alpha1.ClusterObservability{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.GroupVersion.String(), Kind: "ClusterObservability"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: v1alpha1.ClusterObservabilitySpec{
			Exporter: v1alpha1.OTLPHTTPExporter{Endpoint: "https://example.com:4318"},
		},
	}
	require.NoError(t, k8sClient.Create(ctx, instance))
	t.Cleanup(func() {
		for _, child := range []client.Object{
			&v1beta1.OpenTelemetryCollector{ObjectMeta: metav1.ObjectMeta{Name: key.Name + "-agent", Namespace: key.Namespace}},
			&v1beta1.OpenTelemetryCollector{ObjectMeta: metav1.ObjectMeta{Name: key.Name + "-cluster", Namespace: key.Namespace}},
			&v1alpha1.Instrumentation{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}},
		} {
			_ = client.IgnoreNotFound(k8sClient.Delete(context.Background(), child))
		}
		current := &v1alpha1.ClusterObservability{}
		if getErr := k8sClient.Get(context.Background(), key, current); getErr == nil {
			current.Finalizers = nil
			_ = k8sClient.Update(context.Background(), current)
			_ = k8sClient.Delete(context.Background(), current)
		}
		_ = client.IgnoreNotFound(k8sClient.Delete(context.Background(), namespace))
	})

	testCache, err := cache.New(restCfg, cache.Options{Scheme: testScheme})
	require.NoError(t, err)
	cacheCtx, cancelCache := context.WithCancel(context.Background())
	t.Cleanup(cancelCache)
	indexer := controllers.NewClusterObservabilityReconciler(controllers.ClusterObservabilityReconcilerParams{})
	require.NoError(t, indexer.SetupCaches(testCache))
	go func() {
		_ = testCache.Start(cacheCtx)
	}()
	syncCtx, cancelSync := context.WithTimeout(ctx, 10*time.Second)
	t.Cleanup(cancelSync)
	require.True(t, testCache.WaitForCacheSync(syncCtx))
	cachedClient, err := client.New(restCfg, client.Options{
		Scheme: testScheme,
		Cache:  &client.CacheOptions{Reader: testCache},
	})
	require.NoError(t, err)

	reconciler := controllers.NewClusterObservabilityReconciler(controllers.ClusterObservabilityReconcilerParams{
		Client:   cachedClient,
		Recorder: events.NewFakeRecorder(20),
		Scheme:   testScheme,
		Log:      logr.Discard(),
		Config:   operatorconfig.New(),
	})
	req := ctrl.Request{NamespacedName: key}
	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)

	managedResources := []client.Object{
		&v1beta1.OpenTelemetryCollector{ObjectMeta: metav1.ObjectMeta{Name: key.Name + "-agent", Namespace: key.Namespace}},
		&v1beta1.OpenTelemetryCollector{ObjectMeta: metav1.ObjectMeta{Name: key.Name + "-cluster", Namespace: key.Namespace}},
		&v1alpha1.Instrumentation{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}},
	}
	require.Eventually(t, func() bool {
		for _, resource := range managedResources {
			current := resource.DeepCopyObject().(client.Object)
			if err := cachedClient.Get(ctx, client.ObjectKeyFromObject(resource), current); err != nil {
				return false
			}
		}
		return true
	}, 10*time.Second, 50*time.Millisecond)

	before := map[client.ObjectKey]string{}
	for _, resource := range managedResources {
		current := resource.DeepCopyObject().(client.Object)
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(resource), current))
		owner := metav1.GetControllerOf(current)
		require.NotNil(t, owner)
		assert.Equal(t, key.Name, owner.Name)
		if collector, ok := current.(*v1beta1.OpenTelemetryCollector); ok {
			assert.NotEmpty(t, collector.Spec.Config.Exporters.Object["otlp_http"])
			assert.Equal(t, 3, collector.Spec.ConfigVersions)
		}
		before[client.ObjectKeyFromObject(resource)] = current.GetResourceVersion()
	}

	_, err = reconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	for _, resource := range managedResources {
		current := resource.DeepCopyObject().(client.Object)
		resourceKey := client.ObjectKeyFromObject(resource)
		require.NoError(t, k8sClient.Get(ctx, resourceKey, current))
		assert.Equal(t, before[resourceKey], current.GetResourceVersion(), "no-op reconcile updated %T %s", current, resourceKey)
	}
}
