package opentelemetrycollector

import (
	"context"
	"testing"

	fakemon "github.com/coreos/prometheus-operator/pkg/client/versioned/fake"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/client"
	fakeotclient "github.com/open-telemetry/opentelemetry-operator/pkg/client/versioned/fake"
)

func TestProperDaemonSet(t *testing.T) {
	// test
	d := daemonSet(ctx)

	// verify
	assert.Equal(t, d.Name, "my-otelcol-collector")
	assert.Equal(t, d.Annotations["custom-annotation"], "custom-annotation-value")
	assert.Equal(t, d.Labels["custom-label"], "custom-value")
	assert.Equal(t, d.Labels["app.kubernetes.io/name"], d.Name)
}

func TestProperDaemonSets(t *testing.T) {
	// prepare
	instance := &v1alpha1.OpenTelemetryCollector{}
	instance.Spec.Mode = opentelemetry.ModeDaemonSet
	ctx := context.WithValue(ctx, opentelemetry.ContextInstance, instance)

	// test
	d := daemonSets(ctx)

	// verify
	assert.Len(t, d, 1)
}

func TestNoDaemonSetsWhenModeDaemonSet(t *testing.T) {
	// prepare
	d := daemonSets(ctx)

	// verify
	assert.Len(t, d, 0)
}

func TestDaemonSetOverridesConfig(t *testing.T) {
	// prepare
	instance := &v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Args: map[string]string{"config": "custom-path"},
			Mode: opentelemetry.ModeDaemonSet,
		},
	}
	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, instance)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, logf.Log.WithName("unit-tests"))

	// test
	d := daemonSet(ctx)

	// verify
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Len(t, d.Spec.Template.Spec.Containers[0].Args, 1)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args[0], "/conf/collector.yaml")
}

func TestProperReconcileDaemonSet(t *testing.T) {
	// prepare
	instance := *instance
	instance.Spec.Mode = opentelemetry.ModeDaemonSet

	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(&instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}}

	// test
	reconciler.Reconcile(req)

	// verify
	list, err := clients.Kubernetes.AppsV1().DaemonSets(instance.Namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)

	// we assert the correctness of the service in another test
	assert.Len(t, list.Items, 1)

	// we assert the correctness of the reference in another test
	assert.Len(t, list.Items[0].OwnerReferences, 1)
}

func TestOverrideDaemonSetImageFromCustomResource(t *testing.T) {
	// prepare
	instance := &v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "myrepo/custom-image:version",
			Mode:  opentelemetry.ModeDaemonSet,
		},
	}
	ctx := context.WithValue(ctx, opentelemetry.ContextInstance, instance)

	// test
	d := daemonSet(ctx)

	// verify
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "myrepo/custom-image:version", d.Spec.Template.Spec.Containers[0].Image)
}

func TestOverrideDaemonSetImageFromCLI(t *testing.T) {
	// prepare
	viper.Set(opentelemetry.OtelColImageConfigKey, "myrepo/custom-image-cli:version")
	defer viper.Reset()
	defer opentelemetry.ResetFlagSet()

	// test
	d := daemonSet(ctx)

	// verify
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "myrepo/custom-image-cli:version", d.Spec.Template.Spec.Containers[0].Image)
}

func TestDefaultDaemonSetImage(t *testing.T) {
	// prepare
	opentelemetry.FlagSet()
	defer opentelemetry.ResetFlagSet()

	// test
	d := daemonSet(ctx)

	// verify
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Image, "quay.io/opentelemetry/opentelemetry-collector")
}

func TestUpdateDaemonSet(t *testing.T) {
	// prepare
	instance := *instance
	instance.Spec.Mode = opentelemetry.ModeDaemonSet
	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(&instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}}
	reconciler.Reconcile(req)

	// sanity check
	name := resourceName(instance.Name)
	persisted, err := clients.Kubernetes.AppsV1().DaemonSets(instance.Namespace).Get(context.Background(), name, metav1.GetOptions{})
	assert.NoError(t, err)

	// prepare the test object
	updated := instance
	updated.Spec.Image = "custom-image"

	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, &updated)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, logf.Log.WithName("unit-tests"))

	// test
	reconciler.reconcileDaemonSet(ctx)

	// verify
	persisted, err = clients.Kubernetes.AppsV1().DaemonSets(instance.Namespace).Get(context.Background(), name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Len(t, persisted.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "custom-image", persisted.Spec.Template.Spec.Containers[0].Image)
}

func TestDeleteExtraDaemonSet(t *testing.T) {
	// prepare
	c := daemonSet(ctx)
	c.Name = "extra-daemonSet"
	c.Namespace = instance.Namespace

	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(c),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(),
	}
	reconciler := New(schem, clients)

	// sanity check
	persisted, err := clients.Kubernetes.AppsV1().DaemonSets(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// test
	err = reconciler.reconcileDaemonSet(ctx)
	assert.NoError(t, err)

	// verify
	persisted, err = clients.Kubernetes.AppsV1().DaemonSets(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
	assert.Nil(t, persisted)
	assert.Error(t, err) // not found
}
