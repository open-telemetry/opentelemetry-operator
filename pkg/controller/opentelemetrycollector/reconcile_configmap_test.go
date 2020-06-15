package opentelemetrycollector

import (
	"context"
	"testing"

	fakemon "github.com/coreos/prometheus-operator/pkg/client/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/client"
	fakeotclient "github.com/open-telemetry/opentelemetry-operator/pkg/client/versioned/fake"
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
	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}}

	// test
	_, err := reconciler.Reconcile(req)
	assert.NoError(t, err)

	// verify
	list, err := clients.Kubernetes.CoreV1().ConfigMaps(instance.Namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)

	// we assert the correctness of the service in another test
	assert.Len(t, list.Items, 1)

	// we assert the correctness of the reference in another test
	assert.Len(t, list.Items[0].OwnerReferences, 1)
}

func TestUpdateConfigMap(t *testing.T) {
	// prepare
	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}}
	reconciler.Reconcile(req)

	// sanity check
	name := resourceName(instance.Name)
	persisted, err := clients.Kubernetes.CoreV1().ConfigMaps(instance.Namespace).Get(context.Background(), name, metav1.GetOptions{})
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
	persisted, err = clients.Kubernetes.CoreV1().ConfigMaps(instance.Namespace).Get(context.Background(), name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "updated-config-map", persisted.Data[opentelemetry.CollectorConfigMapEntry])
}

func TestDeleteExtraConfigMap(t *testing.T) {
	// prepare
	c := configMap(ctx)
	c.Name = "extra-config-map"
	c.Namespace = instance.Namespace

	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(c),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)

	// sanity check
	persisted, err := clients.Kubernetes.CoreV1().ConfigMaps(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// test
	err = reconciler.reconcileConfigMap(ctx)
	assert.NoError(t, err)

	// verify
	persisted, err = clients.Kubernetes.CoreV1().ConfigMaps(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
	assert.Nil(t, persisted)
	assert.Error(t, err) // not found
}
