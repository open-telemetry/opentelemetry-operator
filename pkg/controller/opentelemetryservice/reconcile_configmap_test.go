package opentelemetryservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
)

func TestProperConfigMap(t *testing.T) {
	// test
	c := configMap(ctx)

	// verify
	assert.Equal(t, c.Name, "my-otelsvc-collector")
	assert.Equal(t, c.Annotations["custom-annotation"], "custom-annotation-value")
	assert.Equal(t, c.Labels["custom-label"], "custom-value")
	assert.Equal(t, c.Labels["app.kubernetes.io/name"], c.Name)
	assert.Equal(t, c.Data[opentelemetry.CollectorConfigMapEntry], "the-config-in-yaml-format")
}

func TestProperReconcileConfigMap(t *testing.T) {
	// prepare
	req := reconcile.Request{}

	// test
	reconciler.Reconcile(req)

	// verify
	list := &corev1.ConfigMapList{}
	cl.List(ctx, client.InNamespace(instance.Namespace), list)

	// we assert the correctness of the service in another test
	assert.Len(t, list.Items, 1)

	// we assert the correctness of the reference in another test
	assert.Len(t, list.Items[0].OwnerReferences, 1)
}
