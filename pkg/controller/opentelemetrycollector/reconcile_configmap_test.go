package opentelemetrycollector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
)

func TestProperConfigMap(t *testing.T) {
	// test
	c := configMap(ctx)

	// verify
	assert.Equal(t, c.Name, "my-otelcol-collector")
	assert.Equal(t, c.Annotations["custom-annotation"], "custom-annotation-value")
	assert.Equal(t, c.Labels["custom-label"], "custom-value")
	assert.Equal(t, c.Labels["app.kubernetes.io/name"], c.Name)
	assert.Equal(t, c.Data[opentelemetry.CollectorConfigMapEntry], "the-config-in-yaml-format")
}

func TestProperReconcileConfigMap(t *testing.T) {
	// prepare
	clients := &Clients{
		client: fake.NewFakeClient(instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{}

	// test
	_, err := reconciler.Reconcile(req)
	assert.NoError(t, err)

	// verify
	list := &corev1.ConfigMapList{}
	clients.client.List(ctx, list, client.InNamespace(instance.Namespace))

	// we assert the correctness of the service in another test
	assert.Len(t, list.Items, 1)

	// we assert the correctness of the reference in another test
	assert.Len(t, list.Items[0].OwnerReferences, 1)
}

func TestUpdateConfigMap(t *testing.T) {
	// prepare
	clients := &Clients{
		client: fake.NewFakeClient(instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{}
	reconciler.Reconcile(req)

	// sanity check
	name := resourceName(instance.Name)
	persisted := &corev1.ConfigMap{}
	err := clients.client.Get(ctx, types.NamespacedName{Name: name, Namespace: instance.Namespace}, persisted)
	assert.NoError(t, err)
	assert.Equal(t, "the-config-in-yaml-format", persisted.Data[opentelemetry.CollectorConfigMapEntry])

	// prepare the test object
	updated := *instance
	updated.Spec.Config = "updated-config-map"

	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, &updated)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, logf.Log.WithName("unit-tests"))

	// test
	reconciler.reconcileConfigMap(ctx)

	// verify
	persisted = &corev1.ConfigMap{}
	assert.NoError(t, clients.client.Get(ctx, types.NamespacedName{Name: name, Namespace: updated.Namespace}, persisted))
	assert.Equal(t, "updated-config-map", persisted.Data[opentelemetry.CollectorConfigMapEntry])
}

func TestDeleteExtraConfigMap(t *testing.T) {
	// prepare
	c := configMap(ctx)
	c.Name = "extra-config-map"

	clients := &Clients{
		client: fake.NewFakeClient(c),
	}
	reconciler := New(schem, clients)

	// sanity check
	persisted := &corev1.ConfigMap{}
	assert.NoError(t, clients.client.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted))

	// test
	err := reconciler.reconcileConfigMap(ctx)
	assert.NoError(t, err)

	// verify
	persisted = &corev1.ConfigMap{}
	err = clients.client.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted)

	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}
