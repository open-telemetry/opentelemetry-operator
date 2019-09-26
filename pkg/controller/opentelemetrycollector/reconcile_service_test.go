package opentelemetrycollector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestProperService(t *testing.T) {
	// test
	s := service(ctx)

	// verify
	assert.Equal(t, s.Name, "my-otelcol-collector")
	assert.Equal(t, s.Annotations["custom-annotation"], "custom-annotation-value")
	assert.Equal(t, s.Labels["custom-label"], "custom-value")
	assert.Equal(t, s.Labels["app.kubernetes.io/name"], s.Name)
	assert.Equal(t, s.Spec.Selector, s.Labels) // shortcut, as they are the same at this point
}

func TestProperHeadlessService(t *testing.T) {
	// test
	s := headless(ctx)

	// verify
	assert.Equal(t, s.Name, "my-otelcol-collector-headless")
	assert.Equal(t, s.Spec.ClusterIP, "None")
}

func TestProperReconcileService(t *testing.T) {
	// prepare
	req := reconcile.Request{}

	// test
	reconciler.Reconcile(req)

	// verify
	list := &corev1.ServiceList{}
	cl.List(ctx, client.InNamespace(instance.Namespace), list)

	// we assert the correctness of the service in another test
	assert.Len(t, list.Items, 2)

	// we assert the correctness of the reference in another test
	for _, item := range list.Items {
		assert.Len(t, item.OwnerReferences, 1)
	}
}

func TestUpdateService(t *testing.T) {
	// prepare
	c := service(ctx)

	// right now, there's nothing we can do at the CR level that influences the underlying
	// service object, so, we simulate a change made manually by some admin, changing the port
	// from 14250 to 12345. Upon reconciliation, this change should be reverted
	c.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "jaeger-grpc",
			Port:       12345,
			TargetPort: intstr.FromInt(12345),
		},
	}
	c.Annotations = nil
	c.Labels = nil

	// the cluster has assigned an IP to this service already
	c.Spec.ClusterIP = "172.172.172.172"

	cl := fake.NewFakeClient(c)
	reconciler := New(cl, schem)

	// sanity check
	persisted := &corev1.Service{}
	assert.NoError(t, cl.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted))
	assert.Len(t, persisted.Spec.Ports, 1)
	assert.Equal(t, int32(12345), persisted.Spec.Ports[0].Port)

	// test
	err := reconciler.reconcileService(ctx)
	assert.NoError(t, err)

	// verify
	persisted = &corev1.Service{}
	err = cl.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted)
	assert.Equal(t, int32(14250), persisted.Spec.Ports[0].Port)
	assert.Equal(t, "172.172.172.172", persisted.Spec.ClusterIP) // the assigned IP is kept
}

func TestDeleteExtraService(t *testing.T) {
	// prepare
	c := service(ctx)
	c.Name = "extra-service"

	cl := fake.NewFakeClient(c)
	reconciler := New(cl, schem)

	// sanity check
	persisted := &corev1.Service{}
	assert.NoError(t, cl.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted))

	// test
	err := reconciler.reconcileService(ctx)
	assert.NoError(t, err)

	// verify
	persisted = &corev1.Service{}
	err = cl.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted)

	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}
