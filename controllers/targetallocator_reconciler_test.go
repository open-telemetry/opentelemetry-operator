// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

var testLogger = logf.Log.WithName("opamp-bridge-controller-unit-tests")

var (
	testScheme *runtime.Scheme = scheme.Scheme
)

func init() {
	utilruntime.Must(monitoringv1.AddToScheme(testScheme))
	utilruntime.Must(networkingv1.AddToScheme(testScheme))
	utilruntime.Must(routev1.AddToScheme(testScheme))
	utilruntime.Must(v1alpha1.AddToScheme(testScheme))
	utilruntime.Must(v1beta1.AddToScheme(testScheme))
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
		record.NewFakeRecorder(10),
		config.New(),
		testLogger,
	)

	t.Run("not owned by a collector", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{}
		collector, err := reconciler.getCollector(context.Background(), ta)
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
		collector, err := reconciler.getCollector(context.Background(), ta)
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
		collector, err := reconciler.getCollector(context.Background(), ta)
		assert.Nil(t, collector)
		assert.Errorf(t, err, "error getting owner for TargetAllocator default/test: opentelemetrycollectors.opentelemetry.io \"non_existent\" not found")
	})
	t.Run("collector attached by label", func(t *testing.T) {
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "label-ta",
			},
		}
		collector, err := reconciler.getCollector(context.Background(), ta)
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
			record.NewFakeRecorder(10),
			config.New(),
			testLogger,
		)
		ta := v1alpha1.TargetAllocator{
			ObjectMeta: metav1.ObjectMeta{
				Name: "label-ta",
			},
		}
		collector, err := reconciler.getCollector(context.Background(), ta)
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
	requests := getTargetAllocatorForCollector(context.Background(), testCollector)
	expected := []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: "default",
		},
	}}
	assert.Equal(t, expected, requests)
}

func TestGetTargetAllocatorRequestsFromLabel(t *testing.T) {
	testCollector := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels: map[string]string{
				constants.LabelTargetAllocator: "label-ta",
			},
		},
	}
	requests := getTargetAllocatorRequestsFromLabel(context.Background(), testCollector)
	expected := []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name:      "label-ta",
			Namespace: "default",
		},
	}}
	assert.Equal(t, expected, requests)
}
