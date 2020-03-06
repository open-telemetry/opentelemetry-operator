package opentelemetrycollector

import (
	"context"
	"testing"

	fakemon "github.com/coreos/prometheus-operator/pkg/client/versioned/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/client"
	fakeotclient "github.com/open-telemetry/opentelemetry-operator/pkg/client/versioned/fake"
)

func TestProperServiceAccount(t *testing.T) {
	// test
	s := serviceAccount(ctx)

	// verify
	assert.Equal(t, s.Name, "my-otelcol-collector")
	assert.Equal(t, s.Annotations["custom-annotation"], "custom-annotation-value")
	assert.Equal(t, s.Labels["custom-label"], "custom-value")
	assert.Equal(t, s.Labels["app.kubernetes.io/name"], s.Name)
}

func TestOverrideServiceAccountName(t *testing.T) {
	// test
	overridden := *instance
	overridden.Spec.ServiceAccount = "custom-sa"

	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, &overridden)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, logf.Log.WithName("unit-tests"))

	s := ServiceAccountNameFor(ctx)

	// verify
	assert.Equal(t, "custom-sa", s)
}

func TestUnmanagedServiceAccount(t *testing.T) {
	// customize the CR, to override the service account name
	overridden := *instance
	overridden.Spec.ServiceAccount = "custom-sa"

	// build a context with an instance of our custom CR
	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, &overridden)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, logf.Log.WithName("unit-tests"))

	c := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        overridden.Spec.ServiceAccount,
			Namespace:   instance.Namespace,
			Annotations: instance.Annotations,
		},
	}

	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(c),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)

	// sanity check
	persisted, err := clients.Kubernetes.CoreV1().ServiceAccounts(instance.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, persisted)

	// test
	reconciler.reconcileServiceAccount(ctx)

	// verify that the service account still exists and is not with the operator's labels
	existing, err := clients.Kubernetes.CoreV1().ServiceAccounts(instance.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Nil(t, existing.Labels)

	// verify that a service account with the name that the operator would create does *not* exist
	managed := serviceAccount(ctx)
	managed, err = clients.Kubernetes.CoreV1().ServiceAccounts(instance.Namespace).Get(managed.Name, metav1.GetOptions{})
	assert.Error(t, err) // not found
	assert.Nil(t, managed)
}

func TestProperReconcileServiceAccount(t *testing.T) {
	// prepare
	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: instance.Namespace}}

	// test
	reconciler.Reconcile(req)

	// verify
	list, err := clients.Kubernetes.CoreV1().ServiceAccounts(instance.Namespace).List(metav1.ListOptions{})
	assert.NoError(t, err)

	// we assert the correctness of the service account in another test
	assert.Len(t, list.Items, 1)

	// we assert the correctness of the reference in another test (TestSetControllerReference)
	for _, item := range list.Items {
		assert.Len(t, item.OwnerReferences, 1)
	}
}

func TestUpdateServiceAccount(t *testing.T) {
	// prepare
	c := serviceAccount(ctx)
	c.Namespace = instance.Namespace
	c.Labels = map[string]string{"some-key": "some-value"}

	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(c),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)

	// sanity check
	persisted, err := clients.Kubernetes.CoreV1().ServiceAccounts(instance.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Contains(t, persisted.Labels, "some-key")
	assert.NotContains(t, persisted.Labels, "app.kubernetes.io/name")

	// test
	reconciler.reconcileServiceAccount(ctx)

	// verify
	persisted, err = clients.Kubernetes.CoreV1().ServiceAccounts(instance.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Contains(t, persisted.Labels, "some-key")
	assert.Equal(t, persisted.Name, persisted.Labels["app.kubernetes.io/name"])
}

func TestDeleteExtraServiceAccount(t *testing.T) {
	// prepare
	c := serviceAccount(ctx)
	c.Name = "extra-service"
	c.Namespace = instance.Namespace

	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(c),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)

	// sanity check
	persisted, err := clients.Kubernetes.CoreV1().ServiceAccounts(c.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// test
	err = reconciler.reconcileServiceAccount(ctx)
	assert.NoError(t, err)

	// verify
	persisted, err = clients.Kubernetes.CoreV1().ServiceAccounts(c.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.Error(t, err) // not found
	assert.Nil(t, persisted)
}
