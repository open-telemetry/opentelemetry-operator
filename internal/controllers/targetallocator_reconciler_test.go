// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"testing"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

var testLogger = logf.Log.WithName("opamp-bridge-controller-unit-tests")

var testScheme *runtime.Scheme = scheme.Scheme

func init() {
	utilruntime.Must(monitoringv1.AddToScheme(testScheme))
	utilruntime.Must(networkingv1.AddToScheme(testScheme))
	utilruntime.Must(routev1.AddToScheme(testScheme))
	utilruntime.Must(v1alpha1.AddToScheme(testScheme))
	utilruntime.Must(v1beta1.AddToScheme(testScheme))
	utilruntime.Must(cmv1.AddToScheme(testScheme))
}

func TestTargetAllocatorReconciler_GetCollector(t *testing.T) {
	testCollector := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Labels: map[string]string{
				constants.LabelTargetAllocator: "label-ta",
			},
		},
	}
	fakeClient := fake.NewFakeClient(testCollector)
	reconciler := NewTargetAllocatorReconciler(
		fakeClient,
		testScheme,
		events.NewFakeRecorder(10),
		config.New(),
		testLogger,
	)

	t.Run("not owned by a collector", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{}
		collector, err := reconciler.getCollector(t.Context(), ta)
		require.NoError(t, err)
		assert.Nil(t, collector)
	})
	t.Run("owned by a collector", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "OpenTelemetryCollector",
						Name: testCollector.Name,
					},
				},
			},
		}
		collector, err := reconciler.getCollector(t.Context(), ta)
		require.NoError(t, err)
		assert.Equal(t, testCollector, collector)
	})
	t.Run("owning collector doesn't exist", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "OpenTelemetryCollector",
						Name: "non_existent",
					},
				},
			},
		}
		collector, err := reconciler.getCollector(t.Context(), ta)
		assert.Nil(t, collector)
		assert.Errorf(t, err, "error getting owner for TargetAllocator default/test: opentelemetrycollectors.opentelemetry.io \"non_existent\" not found")
	})
	t.Run("collector attached by label", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "label-ta",
			},
		}
		collector, err := reconciler.getCollector(t.Context(), ta)
		require.NoError(t, err)
		assert.Equal(t, testCollector, collector)
	})
	t.Run("multiple collectors attached by label", func(t *testing.T) {
		testCollector2 := testCollector.DeepCopy()
		testCollector2.SetName("test2")
		fakeClient := fake.NewFakeClient(testCollector, testCollector2)
		reconciler := NewTargetAllocatorReconciler(
			fakeClient,
			testScheme,
			events.NewFakeRecorder(10),
			config.New(),
			testLogger,
		)
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "label-ta",
			},
		}
		collector, err := reconciler.getCollector(t.Context(), ta)
		assert.Nil(t, collector)
		assert.Errorf(t, err, "found multiple OpenTelemetry collectors annotated with the same Target Allocator: %s/%s", ta.Namespace, ta.Name)
	})
}

func TestGetTargetAllocatorForCollector(t *testing.T) {
	testCollector := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}
	requests := getTargetAllocatorForCollector(t.Context(), testCollector)
	expected := []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: "default",
		},
	}}
	assert.Equal(t, expected, requests)
}

func TestGetTargetAllocatorRequestsFromLabel(t *testing.T) {
	t.Run("collector with label enqueues the named target allocator", func(t *testing.T) {
		testCollector := &v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				Labels: map[string]string{
					constants.LabelTargetAllocator: "label-ta",
				},
			},
		}
		requests := getTargetAllocatorRequestsFromLabel(t.Context(), testCollector)
		expected := []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Name:      "label-ta",
				Namespace: "default",
			},
		}}
		assert.Equal(t, expected, requests)
	})

	t.Run("collector without label enqueues nothing", func(t *testing.T) {
		testCollector := &v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		}
		requests := getTargetAllocatorRequestsFromLabel(t.Context(), testCollector)
		assert.Empty(t, requests)
	})
}

func TestTargetAllocatorReconciler_GetOwnedResourceTypes(t *testing.T) {
	// prepare - a Target Allocator exercising every optional resource the builder can produce
	cfg := config.New()
	cfg.PrometheusCRAvailability = prometheus.Available
	cfg.CertManagerAvailability = certmanager.Available
	cfg.Internal.KubeAPIServerPort = 443
	reconciler := NewTargetAllocatorReconciler(
		fake.NewFakeClient(),
		testScheme,
		events.NewFakeRecorder(10),
		cfg,
		testLogger,
	)
	params := targetallocator.Params{
		Config: cfg,
		Log:    testLogger,
		Scheme: testScheme,
		TargetAllocator: v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: v1alpha1.TargetAllocatorSpec{
				NetworkPolicy: v1beta1.NetworkPolicy{Enabled: new(true)},
				Observability: v1beta1.ObservabilitySpec{
					Metrics: v1beta1.MetricsConfigSpec{EnableMetrics: true},
				},
				Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true},
			},
		},
	}
	desiredObjects, err := BuildTargetAllocator(params)
	require.NoError(t, err)
	require.NotEmpty(t, desiredObjects)

	ownedTypes := map[schema.GroupVersionKind]bool{}
	for _, ownedType := range reconciler.GetOwnedResourceTypes() {
		gvk, gvkErr := apiutil.GVKForObject(ownedType, testScheme)
		require.NoError(t, gvkErr)
		ownedTypes[gvk] = true
	}

	// verify that every resource the controller creates is one it also watches and prunes
	for _, desired := range desiredObjects {
		gvk, gvkErr := apiutil.GVKForObject(desired, testScheme)
		require.NoError(t, gvkErr)
		assert.True(t, ownedTypes[gvk], "%s is built but not returned by GetOwnedResourceTypes", gvk.Kind)
	}
}
