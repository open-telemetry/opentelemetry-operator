package opentelemetrycollector

import (
	"context"
	"testing"

	fakemon "github.com/coreos/prometheus-operator/pkg/client/versioned/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/client"
	fakeotclient "github.com/open-telemetry/opentelemetry-operator/pkg/client/versioned/fake"
)

var logger = logf.Log.WithName("logger")

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

func TestProperMonitoringService(t *testing.T) {
	// test
	s := monitoringService(ctx)

	// verify
	assert.Equal(t, s.Name, "my-otelcol-collector-monitoring")
	assert.Len(t, s.Spec.Ports, 1)
	assert.Equal(t, int32(8888), s.Spec.Ports[0].Port)
}

func TestProperReconcileService(t *testing.T) {
	// prepare
	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}}

	// test
	reconciler.Reconcile(req)

	// verify
	list, err := clients.Kubernetes.CoreV1().Services(instance.Namespace).List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)

	// we assert the correctness of the service in another test
	assert.Len(t, list.Items, 3)

	// we assert the correctness of the reference in another test
	for _, item := range list.Items {
		assert.Len(t, item.OwnerReferences, 1)
	}
}

func TestUpdateService(t *testing.T) {
	// prepare
	c := service(ctx)
	c.Namespace = instance.Namespace

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

	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(c),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)

	// sanity check
	persisted, err := clients.Kubernetes.CoreV1().Services(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Len(t, persisted.Spec.Ports, 1)
	assert.Equal(t, int32(12345), persisted.Spec.Ports[0].Port)

	// test
	err = reconciler.reconcileService(ctx)
	assert.NoError(t, err)

	// verify
	persisted, err = clients.Kubernetes.CoreV1().Services(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(1234), persisted.Spec.Ports[0].Port)
	assert.Equal(t, "172.172.172.172", persisted.Spec.ClusterIP) // the assigned IP is kept
}

func TestDeleteExtraService(t *testing.T) {
	// prepare
	c := service(ctx)
	c.Name = "extra-service"
	c.Namespace = instance.Namespace

	clients := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(c),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(instance),
	}
	reconciler := New(schem, clients)

	// sanity check
	persisted, err := clients.Kubernetes.CoreV1().Services(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// test
	err = reconciler.reconcileService(ctx)
	assert.NoError(t, err)

	// verify
	persisted, err = clients.Kubernetes.CoreV1().Services(c.Namespace).Get(context.Background(), c.Name, metav1.GetOptions{})
	assert.Error(t, err) // not found
	assert.Nil(t, persisted)
}

func TestServiceWithoutPorts(t *testing.T) {
	for _, tt := range []string{
		"",
		"🦄",
		"receivers:\n  myreceiver:\n    endpoint:",
		"receivers:\n  myreceiver:\n    endpoint: 0.0.0.0",
	} {
		// prepare
		i := *instance
		i.Spec.Config = tt
		c := context.WithValue(ctx, opentelemetry.ContextInstance, &i)

		// test
		s := service(c)

		// verify
		assert.Nil(t, s, "expected no ports from a configuration like: %s", tt)
	}
}
