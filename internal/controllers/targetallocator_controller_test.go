// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimecluster "sigs.k8s.io/controller-runtime/pkg/cluster"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/apiserverendpoints"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
)

var testLogger = logf.Log.WithName("opamp-bridge-controller-unit-tests")

// createTestTargetAllocatorReconciler builds a reconciler backed by a cached client, which the reconciler needs to
// query the field indexes it sets up for owned objects.
func createTestTargetAllocatorReconciler(t *testing.T, ctx context.Context, cfg config.Config) *controllers.TargetAllocatorReconciler {
	t.Helper()
	runtimeCluster, err := runtimecluster.New(restCfg, func(options *runtimecluster.Options) {
		options.Scheme = testScheme
		options.Client.Cache = &client.CacheOptions{
			EnableReadYourWritesConsistency: new(true),
		}
	})
	require.NoError(t, err)
	go func() {
		startErr := runtimeCluster.Start(ctx)
		assert.NoError(t, startErr)
	}()

	reconciler := controllers.NewTargetAllocatorReconciler(
		runtimeCluster.GetClient(),
		testScheme,
		events.NewFakeRecorder(10),
		cfg,
		testLogger,
	)
	err = reconciler.SetupCaches(runtimeCluster)
	require.NoError(t, err)
	synced := runtimeCluster.GetCache().WaitForCacheSync(ctx)
	require.True(t, synced, "caches didn't sync successfully")
	return reconciler
}

func TestNewObjectsOnReconciliation_TargetAllocator(t *testing.T) {
	// prepare
	cfg := config.Config{
		TargetAllocatorImage:          "default-ta",
		TargetAllocatorConfigMapEntry: "remoteconfiguration.yaml",
		CollectorConfigMapEntry:       "collector.yaml",
	}
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := createTestTargetAllocatorReconciler(t, ctx, cfg)
	created := &v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TargetAllocatorSpec{},
	}
	err := k8sClient.Create(t.Context(), created)
	require.NoError(t, err)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(t.Context(), req)

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
		err = k8sClient.List(t.Context(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &corev1.ServiceAccountList{}
		err = k8sClient.List(t.Context(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &corev1.ServiceList{}
		err = k8sClient.List(t.Context(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &appsv1.DeploymentList{}
		err = k8sClient.List(t.Context(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}

	// cleanup
	require.NoError(t, k8sClient.Delete(t.Context(), created))
}

func TestSkipWhenInstanceDoesNotExist_TargetAllocator(t *testing.T) {
	// prepare
	cfg := config.New()
	nsn := types.NamespacedName{Name: "non-existing-my-instance", Namespace: "default"}
	reconciler := createTestTargetAllocatorReconciler(t, ctx, cfg)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err := reconciler.Reconcile(t.Context(), req)
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
	err = k8sClient.List(t.Context(), &objList, opts...)
	assert.NoError(t, err)
	assert.Empty(t, objList.Items)
}

func TestUnmanaged_TargetAllocator(t *testing.T) {
	// prepare
	cfg := config.Config{
		TargetAllocatorImage: "default-ta",
	}
	nsn := types.NamespacedName{Name: "my-instance-unmanaged", Namespace: "default"}
	reconciler := createTestTargetAllocatorReconciler(t, ctx, cfg)
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
	err := k8sClient.Create(t.Context(), unmanaged)
	require.NoError(t, err)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(t.Context(), req)

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
	err = k8sClient.List(t.Context(), &objList, opts...)
	assert.NoError(t, err)
	assert.Empty(t, objList.Items)

	// cleanup
	require.NoError(t, k8sClient.Delete(t.Context(), unmanaged))
}

func TestBuildError_TargetAllocator(t *testing.T) {
	// prepare
	cfg := config.Config{
		TargetAllocatorImage: "default-ta",
	}
	nsn := types.NamespacedName{Name: "my-instance-builderror", Namespace: "default"}
	reconciler := createTestTargetAllocatorReconciler(t, ctx, cfg)
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
	err := k8sClient.Create(t.Context(), unmanaged)
	require.NoError(t, err)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(t.Context(), req)

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
	err = k8sClient.List(t.Context(), &objList, opts...)
	assert.NoError(t, err)
	assert.Empty(t, objList.Items)

	// cleanup
	require.NoError(t, k8sClient.Delete(t.Context(), unmanaged))
}

func TestPruneOwnedObjects_TargetAllocator(t *testing.T) {
	// prepare - the NetworkPolicy needs API server endpoints, which envtest doesn't publish
	t.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")
	cfg := config.Config{
		TargetAllocatorImage:          "default-ta",
		TargetAllocatorConfigMapEntry: "remoteconfiguration.yaml",
		CollectorConfigMapEntry:       "collector.yaml",
	}
	cfg.Internal.APIServerEndpoints = apiserverendpoints.NewTracker(syncedReader{k8sClient}, time.Hour)
	nsn := types.NamespacedName{Name: "my-instance-prune", Namespace: "default"}
	reconciler := createTestTargetAllocatorReconciler(t, ctx, cfg)
	created := &v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			NetworkPolicy: v1beta1.NetworkPolicy{Enabled: new(true)},
		},
	}
	require.NoError(t, k8sClient.Create(t.Context(), created))
	t.Cleanup(func() {
		require.NoError(t, k8sClient.Delete(context.Background(), created))
	})

	req := k8sreconcile.Request{NamespacedName: nsn}
	_, err := reconciler.Reconcile(t.Context(), req)
	require.NoError(t, err)

	networkPolicy := &networkingv1.NetworkPolicy{}
	networkPolicyKey := types.NamespacedName{
		Name:      naming.TargetAllocatorNetworkPolicy(nsn.Name),
		Namespace: nsn.Namespace,
	}
	require.NoError(t, k8sClient.Get(t.Context(), networkPolicyKey, networkPolicy))

	// test - disabling the NetworkPolicy should remove the one we just created
	instance := &v1alpha1.TargetAllocator{}
	require.NoError(t, reconciler.Get(t.Context(), nsn, instance))
	instance.Spec.NetworkPolicy.Enabled = new(false)
	require.NoError(t, reconciler.Update(t.Context(), instance))

	_, err = reconciler.Reconcile(t.Context(), req)
	require.NoError(t, err)

	// verify
	err = k8sClient.Get(t.Context(), networkPolicyKey, networkPolicy)
	assert.True(t, apierrors.IsNotFound(err), "expected the NetworkPolicy to be pruned, got %v", err)

	// the resources still in the desired state are left alone
	deployment := &appsv1.Deployment{}
	deploymentKey := types.NamespacedName{
		Name:      naming.TargetAllocator(nsn.Name),
		Namespace: nsn.Namespace,
	}
	assert.NoError(t, k8sClient.Get(t.Context(), deploymentKey, deployment))
}

func TestSetupCaches_SharedWithCollector(t *testing.T) {
	// prepare - both controllers index overlapping resource types on the same cache
	runtimeCluster, err := runtimecluster.New(restCfg, func(options *runtimecluster.Options) {
		options.Scheme = testScheme
	})
	require.NoError(t, err)

	cfg := config.New()
	collectorReconciler := controllers.NewReconciler(controllers.Params{
		Client:   runtimeCluster.GetClient(),
		Log:      testLogger,
		Scheme:   testScheme,
		Recorder: events.NewFakeRecorder(10),
		Config:   cfg,
		Version:  version.Get(),
	})
	targetAllocatorReconciler := controllers.NewTargetAllocatorReconciler(
		runtimeCluster.GetClient(),
		testScheme,
		events.NewFakeRecorder(10),
		cfg,
		testLogger,
	)

	// test
	require.NoError(t, collectorReconciler.SetupCaches(runtimeCluster))

	// verify
	require.NoError(t, targetAllocatorReconciler.SetupCaches(runtimeCluster))
}

func TestNetworkPolicyFollowsAPIServerEndpoints_TargetAllocator(t *testing.T) {
	// make the result independent of the environment the test runs in
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")

	nsn := types.NamespacedName{Name: "network-policy-apiserver", Namespace: "default"}
	cfg := config.Config{
		TargetAllocatorImage:          "default-ta",
		TargetAllocatorConfigMapEntry: "remoteconfiguration.yaml",
		CollectorConfigMapEntry:       "collector.yaml",
	}
	cfg.Internal.APIServerEndpoints = apiserverendpoints.NewTracker(syncedReader{k8sClient}, time.Hour)
	reconciler := createTestTargetAllocatorReconciler(t, ctx, cfg)
	ta := &v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			NetworkPolicy: v1beta1.NetworkPolicy{Enabled: new(true)},
		},
	}
	require.NoError(t, k8sClient.Create(t.Context(), ta))
	t.Cleanup(func() {
		assert.NoError(t, k8sClient.Delete(context.Background(), ta))
	})
	req := k8sreconcile.Request{NamespacedName: nsn}

	// envtest doesn't run the API server endpoint reconciler, so there are no endpoints yet
	_, err := reconciler.Reconcile(t.Context(), req)
	require.ErrorContains(t, err, "no endpoints found for the Kubernetes API server")

	slice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes-network-policy-test",
			Namespace: apiserverendpoints.Namespace,
			Labels:    map[string]string{discoveryv1.LabelServiceName: apiserverendpoints.ServiceName},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"172.18.0.2"}}},
		Ports:       []discoveryv1.EndpointPort{{Name: new("https"), Port: new(int32(6443))}},
	}
	require.NoError(t, k8sClient.Create(t.Context(), slice))
	t.Cleanup(func() {
		assert.NoError(t, k8sClient.Delete(context.Background(), slice))
	})

	getEgressCIDRs := func() []string {
		var np networkingv1.NetworkPolicy
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{
			Name:      naming.TargetAllocatorNetworkPolicy(nsn.Name),
			Namespace: nsn.Namespace,
		}, &np))
		require.Len(t, np.Spec.Egress, 1)
		var cidrs []string
		for _, peer := range np.Spec.Egress[0].To {
			cidrs = append(cidrs, peer.IPBlock.CIDR)
		}
		return cidrs
	}

	result, err := reconciler.Reconcile(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, []string{"172.18.0.2/32"}, getEgressCIDRs())
	assert.Zero(t, result.RequeueAfter)

	// the API server endpoints change, e.g. during a control plane upgrade
	slice.Endpoints = []discoveryv1.Endpoint{{Addresses: []string{"172.18.0.3"}}, {Addresses: []string{"172.18.0.4"}}}
	require.NoError(t, k8sClient.Update(t.Context(), slice))

	// the removed endpoint is still allowed until it expires
	result, err = reconciler.Reconcile(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, []string{"172.18.0.2/32", "172.18.0.3/32", "172.18.0.4/32"}, getEgressCIDRs())
	assert.Positive(t, result.RequeueAfter)
	assert.LessOrEqual(t, result.RequeueAfter, time.Hour)
}

// syncedReader adapts a client to apiserverendpoints.CacheReader.
type syncedReader struct {
	client.Reader
}

func (syncedReader) WaitForCacheSync(context.Context) bool {
	return true
}
