package opentelemetryservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestProperService(t *testing.T) {
	// test
	s := service(ctx)

	// verify
	assert.Equal(t, s.Name, "my-otelsvc-collector")
	assert.Equal(t, s.Annotations["custom-annotation"], "custom-annotation-value")
	assert.Equal(t, s.Labels["custom-label"], "custom-value")
	assert.Equal(t, s.Labels["app.kubernetes.io/name"], s.Name)
	assert.Equal(t, s.Spec.Selector, s.Labels) // shortcut, as they are the same at this point
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
	assert.Len(t, list.Items, 1)

	// we assert the correctness of the reference in another test
	assert.Len(t, list.Items[0].OwnerReferences, 1)
}
