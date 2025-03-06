// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var testLogger = logf.Log.WithName("opamp-bridge-controller-unit-tests")

func TestNewObjectsOnReconciliation_TargetAllocator(t *testing.T) {
	// prepare
	cfg := config.New(
		config.WithTargetAllocatorImage("default-ta"),
	)
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := controllers.NewTargetAllocatorReconciler(
		k8sClient,
		testScheme,
		record.NewFakeRecorder(10),
		cfg,
		testLogger,
	)
	created := &v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TargetAllocatorSpec{},
	}
	err := k8sClient.Create(context.Background(), created)
	require.NoError(t, err)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(context.Background(), req)

	// verify
	require.NoError(t, err)

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", nsn.Namespace, nsn.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/component":  "opentelemetry-targetallocator",
		}),
	}

	// verify that we have at least one object for each of the types we create
	// whether we have the right ones is up to the specific tests for each type
	{
		list := &corev1.ConfigMapList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &corev1.ServiceAccountList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &corev1.ServiceList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &appsv1.DeploymentList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), created))
}

func TestSkipWhenInstanceDoesNotExist_TargetAllocator(t *testing.T) {
	// prepare
	cfg := config.New()
	nsn := types.NamespacedName{Name: "non-existing-my-instance", Namespace: "default"}
	reconciler := controllers.NewTargetAllocatorReconciler(
		k8sClient,
		testScheme,
		record.NewFakeRecorder(10),
		cfg,
		testLogger,
	)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", nsn.Namespace, nsn.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/component":  "opentelemetry-targetallocator",
		}),
	}

	// verify that no objects have been created
	var objList appsv1.DeploymentList
	err = k8sClient.List(context.Background(), &objList, opts...)
	assert.NoError(t, err)
	assert.Empty(t, objList.Items)
}

func TestUnmanaged_TargetAllocator(t *testing.T) {
	// prepare
	cfg := config.New(
		config.WithTargetAllocatorImage("default-ta"),
	)
	nsn := types.NamespacedName{Name: "my-instance-unmanaged", Namespace: "default"}
	reconciler := controllers.NewTargetAllocatorReconciler(
		k8sClient,
		testScheme,
		record.NewFakeRecorder(10),
		cfg,
		testLogger,
	)
	unmanaged := &v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				ManagementState: v1beta1.ManagementStateUnmanaged,
			},
		},
	}
	err := k8sClient.Create(context.Background(), unmanaged)
	require.NoError(t, err)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(context.Background(), req)

	// verify
	require.NoError(t, err)

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", nsn.Namespace, nsn.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/component":  "opentelemetry-targetallocator",
		}),
	}

	// verify that no objects have been created
	var objList appsv1.DeploymentList
	err = k8sClient.List(context.Background(), &objList, opts...)
	assert.NoError(t, err)
	assert.Empty(t, objList.Items)

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), unmanaged))
}

func TestBuildError_TargetAllocator(t *testing.T) {
	// prepare
	cfg := config.New(
		config.WithTargetAllocatorImage("default-ta"),
	)
	nsn := types.NamespacedName{Name: "my-instance-builderror", Namespace: "default"}
	reconciler := controllers.NewTargetAllocatorReconciler(
		k8sClient,
		testScheme,
		record.NewFakeRecorder(10),
		cfg,
		testLogger,
	)
	unmanaged := &v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				PodDisruptionBudget: &v1beta1.PodDisruptionBudgetSpec{},
			},
			AllocationStrategy: v1beta1.TargetAllocatorAllocationStrategyLeastWeighted,
		},
	}
	err := k8sClient.Create(context.Background(), unmanaged)
	require.NoError(t, err)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(context.Background(), req)

	// verify
	require.Error(t, err)

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", nsn.Namespace, nsn.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/component":  "opentelemetry-targetallocator",
		}),
	}

	// verify that no objects have been created
	var objList appsv1.DeploymentList
	err = k8sClient.List(context.Background(), &objList, opts...)
	assert.NoError(t, err)
	assert.Empty(t, objList.Items)

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), unmanaged))
}
