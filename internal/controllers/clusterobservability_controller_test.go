// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	operatorconfig "github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestClusterObservabilitySpecUpdateReconcilesManagedResources(t *testing.T) {
	ctx := context.Background()
	key := types.NamespacedName{Name: "cluster-obs", Namespace: "observability"}
	initialExporter := v1alpha1.OTLPHTTPExporter{
		Endpoint: "https://old.example.com:4318",
		Timeout:  "30s",
	}
	initialExporterConfig := map[string]any{
		"endpoint": "https://old.example.com:4318",
		"timeout":  "30s",
	}
	instance := &v1alpha1.ClusterObservability{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "ClusterObservability",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       key.Name,
			Namespace:  key.Namespace,
			UID:        types.UID("cluster-observability-uid"),
			Generation: 1,
		},
		Spec: v1alpha1.ClusterObservabilitySpec{
			Exporter: initialExporter,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))
	cli := newClusterObservabilityFakeClientBuilder(scheme).
		WithStatusSubresource(&v1alpha1.ClusterObservability{}).
		WithObjects(instance).
		Build()
	reconciler := NewClusterObservabilityReconciler(ClusterObservabilityReconcilerParams{
		Client:   cli,
		Recorder: events.NewFakeRecorder(20),
		Scheme:   scheme,
		Log:      logr.Discard(),
		Config:   operatorconfig.New(),
	})

	_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	assertManagedCollectorExporter(t, ctx, cli, key.Namespace, key.Name+"-agent", initialExporterConfig)
	assertManagedCollectorExporter(t, ctx, cli, key.Namespace, key.Name+"-cluster", initialExporterConfig)

	updatedExporter := v1alpha1.OTLPHTTPExporter{
		Endpoint:    "https://new.example.com:4318",
		Timeout:     "45s",
		Compression: "none",
	}
	updatedExporterConfig := map[string]any{
		"endpoint":    "https://new.example.com:4318",
		"timeout":     "45s",
		"compression": "none",
	}
	updated := &v1alpha1.ClusterObservability{}
	require.NoError(t, cli.Get(ctx, key, updated))
	updated.Spec.Exporter = updatedExporter
	// The fake client does not increment metadata.generation automatically.
	updated.Generation++
	require.NoError(t, cli.Update(ctx, updated))

	_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	assertManagedCollectorExporter(t, ctx, cli, key.Namespace, key.Name+"-agent", updatedExporterConfig)
	assertManagedCollectorExporter(t, ctx, cli, key.Namespace, key.Name+"-cluster", updatedExporterConfig)

	instrumentation := &v1alpha1.Instrumentation{}
	require.NoError(t, cli.Get(ctx, client.ObjectKey(key), instrumentation))
	assert.Equal(t, "http://$(OTEL_NODE_IP):4318", instrumentation.Spec.Endpoint)

	reconciled := &v1alpha1.ClusterObservability{}
	require.NoError(t, cli.Get(ctx, key, reconciled))
	assert.Equal(t, reconciled.Generation, reconciled.Status.ObservedGeneration)
}

func TestClusterObservabilityManagedResourcesAreStableAndFollowOperatorImage(t *testing.T) {
	ctx := context.Background()
	key := types.NamespacedName{Name: "cluster-obs", Namespace: "observability"}
	instance := &v1alpha1.ClusterObservability{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.GroupVersion.String(), Kind: "ClusterObservability"},
		ObjectMeta: metav1.ObjectMeta{
			Name:       key.Name,
			Namespace:  key.Namespace,
			UID:        types.UID("cluster-observability-uid"),
			Generation: 1,
		},
		Spec: v1alpha1.ClusterObservabilitySpec{
			Exporter: v1alpha1.OTLPHTTPExporter{Endpoint: "https://example.com:4318"},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))
	var childUpdates []string
	cli := newClusterObservabilityFakeClientBuilder(scheme).
		WithStatusSubresource(&v1alpha1.ClusterObservability{}).
		WithObjects(instance).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, cli client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				switch obj.(type) {
				case *v1beta1.OpenTelemetryCollector, *v1alpha1.Instrumentation:
					childUpdates = append(childUpdates, fmt.Sprintf("%T/%s", obj, obj.GetName()))
				}
				return cli.Update(ctx, obj, opts...)
			},
		}).
		Build()

	oldImage := "registry.example.com/otelcol-k8s:old"
	oldJavaImage := "registry.example.com/autoinstrumentation-java:old"
	initialConfig := operatorconfig.New()
	initialConfig.ClusterObservabilityCollectorImage = oldImage
	initialConfig.AutoInstrumentationJavaImage = oldJavaImage
	reconciler := NewClusterObservabilityReconciler(ClusterObservabilityReconcilerParams{
		Client:   cli,
		Recorder: events.NewFakeRecorder(20),
		Scheme:   scheme,
		Log:      logr.Discard(),
		Config:   initialConfig,
	})

	_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	for _, name := range []string{key.Name + "-agent", key.Name + "-cluster"} {
		collector := &v1beta1.OpenTelemetryCollector{}
		require.NoError(t, cli.Get(ctx, client.ObjectKey{Namespace: key.Namespace, Name: name}, collector))
		assert.Equal(t, oldImage, collector.Spec.Image)
		assert.Equal(t, v1beta1.UpgradeStrategyAutomatic, collector.Spec.UpgradeStrategy)
		assert.Equal(t, v1beta1.ManagementStateManaged, collector.Spec.ManagementState)
		assert.NotEmpty(t, collector.Spec.Config.Exporters.Object["otlp_http"])
	}
	instrumentation := &v1alpha1.Instrumentation{}
	require.NoError(t, cli.Get(ctx, client.ObjectKey(key), instrumentation))
	assert.Equal(t, oldJavaImage, instrumentation.Spec.Java.Image)
	stale := &v1alpha1.Instrumentation{ObjectMeta: metav1.ObjectMeta{
		Name: "stale", Namespace: key.Namespace, UID: types.UID("stale-instrumentation-uid"),
	}}
	require.NoError(t, ctrl.SetControllerReference(instance, stale, scheme))
	require.NoError(t, cli.Create(ctx, stale))

	agent := &v1beta1.OpenTelemetryCollector{}
	require.NoError(t, cli.Get(ctx, client.ObjectKey{Namespace: key.Namespace, Name: key.Name + "-agent"}, agent))
	agent.Labels["opentelemetry.io/opamp-reporting"] = "true"
	if agent.Annotations == nil {
		agent.Annotations = map[string]string{}
	}
	agent.Annotations["example.com/external-metadata"] = "preserved"
	require.NoError(t, cli.Update(ctx, agent))
	childUpdates = nil

	// A no-op reconciliation must not rewrite webhook-defaulted child CRs.
	_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	assert.Empty(t, childUpdates)
	assert.True(t, apierrors.IsNotFound(cli.Get(ctx, client.ObjectKeyFromObject(stale), &v1alpha1.Instrumentation{})))

	// A new manager configuration simulates an operator upgrade. The parent CR
	// generation is unchanged, but the new operand image must still propagate.
	newImage := "registry.example.com/otelcol-k8s:new"
	newJavaImage := "registry.example.com/autoinstrumentation-java:new"
	upgradedConfig := initialConfig
	upgradedConfig.ClusterObservabilityCollectorImage = newImage
	upgradedConfig.AutoInstrumentationJavaImage = newJavaImage
	upgradedReconciler := NewClusterObservabilityReconciler(ClusterObservabilityReconcilerParams{
		Client:   cli,
		Recorder: events.NewFakeRecorder(20),
		Scheme:   scheme,
		Log:      logr.Discard(),
		Config:   upgradedConfig,
	})
	_, err = upgradedReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"*v1beta1.OpenTelemetryCollector/cluster-obs-agent",
		"*v1beta1.OpenTelemetryCollector/cluster-obs-cluster",
		"*v1alpha1.Instrumentation/cluster-obs",
	}, childUpdates)

	for _, name := range []string{key.Name + "-agent", key.Name + "-cluster"} {
		collector := &v1beta1.OpenTelemetryCollector{}
		require.NoError(t, cli.Get(ctx, client.ObjectKey{Namespace: key.Namespace, Name: name}, collector))
		assert.Equal(t, newImage, collector.Spec.Image)
	}
	require.NoError(t, cli.Get(ctx, client.ObjectKey{Namespace: key.Namespace, Name: key.Name + "-agent"}, agent))
	assert.Equal(t, "true", agent.Labels["opentelemetry.io/opamp-reporting"])
	assert.Equal(t, "preserved", agent.Annotations["example.com/external-metadata"])
	require.NoError(t, cli.Get(ctx, client.ObjectKey(key), instrumentation))
	assert.Equal(t, newJavaImage, instrumentation.Spec.Java.Image)

	reconciled := &v1alpha1.ClusterObservability{}
	require.NoError(t, cli.Get(ctx, key, reconciled))
	assert.Equal(t, int64(1), reconciled.Generation)
	assert.Equal(t, reconciled.Generation, reconciled.Status.ObservedGeneration)
}

func TestClusterObservabilityCreatesOpenShiftSCCBeforeCollectors(t *testing.T) {
	ctx := context.Background()
	key := types.NamespacedName{Name: "cluster-obs", Namespace: "observability"}
	instance := &v1alpha1.ClusterObservability{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.GroupVersion.String(), Kind: "ClusterObservability"},
		ObjectMeta: metav1.ObjectMeta{
			Name:       key.Name,
			Namespace:  key.Namespace,
			UID:        types.UID("cluster-observability-uid"),
			Generation: 1,
		},
		Spec: v1alpha1.ClusterObservabilitySpec{
			Exporter: v1alpha1.OTLPHTTPExporter{Endpoint: "https://example.com:4318"},
		},
	}

	sccGVK := schema.GroupVersionKind{Group: "security.openshift.io", Version: "v1", Kind: "SecurityContextConstraints"}
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))
	scheme.AddKnownTypeWithName(sccGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(sccGVK.GroupVersion().WithKind(sccGVK.Kind+"List"), &unstructured.UnstructuredList{})

	var created []string
	cli := newClusterObservabilityFakeClientBuilder(scheme).
		WithStatusSubresource(&v1alpha1.ClusterObservability{}).
		WithObjects(instance).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, cli client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				switch obj := obj.(type) {
				case *unstructured.Unstructured:
					created = append(created, obj.GetKind())
				case *v1beta1.OpenTelemetryCollector:
					created = append(created, "OpenTelemetryCollector/"+obj.Name)
				}
				return cli.Create(ctx, obj, opts...)
			},
		}).
		Build()
	cfg := operatorconfig.New()
	cfg.OpenShiftRoutesAvailability = openshift.RoutesAvailable
	reconciler := NewClusterObservabilityReconciler(ClusterObservabilityReconcilerParams{
		Client: cli, Recorder: events.NewFakeRecorder(10), Scheme: scheme, Log: logr.Discard(), Config: cfg,
	})

	_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(created), 3)
	assert.Equal(t, "SecurityContextConstraints", created[0])
	assert.Equal(t, "OpenTelemetryCollector/cluster-obs-agent", created[1])
	assert.Equal(t, "OpenTelemetryCollector/cluster-obs-cluster", created[2])
}

func TestApplyDesiredUnstructuredResourceDoesNotAddEmptyMetadataMaps(t *testing.T) {
	existing := &unstructured.Unstructured{Object: map[string]any{"metadata": map[string]any{"name": "example"}}}
	desired := existing.DeepCopy()

	applyDesiredUnstructuredResource(existing, desired)

	assert.Nil(t, existing.GetLabels())
	assert.Nil(t, existing.GetAnnotations())
}

func TestReconcileUnstructuredResourceAvoidsNoOpUpdates(t *testing.T) {
	ctx := context.Background()
	gvk := schema.GroupVersionKind{Group: "security.openshift.io", Version: "v1", Kind: "SecurityContextConstraints"}
	desired := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "security.openshift.io/v1",
		"kind":       "SecurityContextConstraints",
		"metadata": map[string]any{
			"name":        "cluster-obs-agent-hostaccess",
			"labels":      map[string]any{"app.kubernetes.io/managed-by": "opentelemetry-operator"},
			"annotations": map[string]any{"kubernetes.io/description": "managed"},
		},
		"priority":         float64(10),
		"allowHostNetwork": true,
	}}
	desired.SetGroupVersionKind(gvk)
	existing := desired.DeepCopy()
	existing.SetResourceVersion("1")
	existing.SetUID(types.UID("scc-uid"))
	existing.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"example.com/external":         "preserved",
	})
	existing.SetAnnotations(map[string]string{
		"kubernetes.io/description": "managed",
		"example.com/external":      "preserved",
	})
	existing.Object["groups"] = []any{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(gvk.GroupVersion().WithKind(gvk.Kind+"List"), &unstructured.UnstructuredList{})
	updates := 0
	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, cli client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				updates++
				return cli.Update(ctx, obj, opts...)
			},
		}).
		Build()
	reconciler := NewClusterObservabilityReconciler(ClusterObservabilityReconcilerParams{Client: cli, Scheme: scheme, Log: logr.Discard()})

	require.NoError(t, reconciler.reconcileUnstructuredResource(ctx, logr.Discard(), desired))
	assert.Zero(t, updates)

	desired.Object["priority"] = float64(20)
	require.NoError(t, reconciler.reconcileUnstructuredResource(ctx, logr.Discard(), desired))
	assert.Equal(t, 1, updates)

	updated := &unstructured.Unstructured{}
	updated.SetGroupVersionKind(gvk)
	require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(existing), updated))
	assert.Equal(t, int64(20), updated.Object["priority"])
	assert.Equal(t, []any{}, updated.Object["groups"])
	assert.Equal(t, "preserved", updated.GetLabels()["example.com/external"])
	assert.Equal(t, "preserved", updated.GetAnnotations()["example.com/external"])
}

func assertManagedCollectorExporter(
	t *testing.T,
	ctx context.Context,
	cli client.Client,
	namespace string,
	name string,
	want map[string]any,
) {
	t.Helper()

	collector := &v1beta1.OpenTelemetryCollector{}
	require.NoError(t, cli.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, collector))
	assert.Equal(t, want, collector.Spec.Config.Exporters.Object["otlp_http"])
}

func newClusterObservabilityFakeClientBuilder(scheme *runtime.Scheme) *fake.ClientBuilder {
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(&v1beta1.OpenTelemetryCollector{}, clusterObservabilityResourceOwnerKey, clusterObservabilityOwnerName).
		WithIndex(&v1alpha1.Instrumentation{}, clusterObservabilityResourceOwnerKey, clusterObservabilityOwnerName)
}
