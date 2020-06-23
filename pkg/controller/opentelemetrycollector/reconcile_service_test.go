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

func TestOverridePorts(t *testing.T) {
	// prepare
	i := *instance
	i.Spec.Ports = []corev1.ServicePort{{
		Name: "my-port",
		Port: int32(1234),
	}}
	c := context.WithValue(ctx, opentelemetry.ContextInstance, &i)

	// test
	s := service(c)

	// verify
	assert.NotNil(t, s)
	assert.Len(t, s.Spec.Ports, 1)
	assert.Equal(t, int32(1234), s.Spec.Ports[0].Port)
	assert.Equal(t, "my-port", s.Spec.Ports[0].Name)
}

func TestAddExplicitPorts(t *testing.T) {
	for _, tt := range []struct {
		name          string // the test case name
		instancePorts []corev1.ServicePort
		expectedPorts map[int32]bool
		expectedNames map[string]bool
	}{
		{
			name: "NewPort",
			instancePorts: []corev1.ServicePort{{
				Name: "my-port",
				Port: int32(1235),
			}},
			// "first" (1234) comes from a receiver in .Spec.Config
			expectedNames: map[string]bool{"first": false, "my-port": false},
			expectedPorts: map[int32]bool{1234: false, 1235: false},
		},
		{
			name: "FallbackPortName",
			instancePorts: []corev1.ServicePort{{
				Name: "first", // clashes with a receiver from .Spec.Config
				Port: int32(1235),
			}},
			expectedNames: map[string]bool{"port-1234": false, "first": false},
			expectedPorts: map[int32]bool{1234: false, 1235: false},
		},
		{
			name: "FallbackPortNameClashes",
			instancePorts: []corev1.ServicePort{
				{
					Name: "first", // clashes with the port 1234 from the receiver in .Spec.Config
					Port: int32(1235),
				},
				{
					Name: "port-1234", // the "first" port will be renamed to port-1234, clashes with this one
					Port: int32(1236),
				},
			},
			expectedNames: map[string]bool{"first": false, "port-1234": false},
			expectedPorts: map[int32]bool{1235: false, 1236: false}, // the inferred port 1234 is skipped
		},
		{
			name: "SkipExistingPortNumber",
			instancePorts: []corev1.ServicePort{{
				Name: "my-port",
				Port: int32(1234),
			}},
			expectedNames: map[string]bool{"my-port": false},
			expectedPorts: map[int32]bool{1234: false},
		},
	} {
		t.Run("TestAddExplicitPorts-"+tt.name, func(t *testing.T) {
			// prepare
			i := *instance
			i.Spec.Ports = tt.instancePorts
			c := context.WithValue(ctx, opentelemetry.ContextInstance, &i)

			// test
			s := service(c)

			// verify
			assert.NotNil(t, s)

			for _, p := range s.Spec.Ports {
				if _, ok := tt.expectedPorts[p.Port]; !ok {
					assert.Fail(t, "found a port that we didn't expect", "port number: %d", p.Port)
				}
				tt.expectedPorts[p.Port] = true

				if _, ok := tt.expectedNames[p.Name]; !ok {
					assert.Fail(t, "found a port name that we didn't expect", "port name: %s", p.Name)
				}
				tt.expectedNames[p.Name] = true
			}

			for k, v := range tt.expectedPorts {
				assert.True(t, v, "the port %s should have been part of the result", k)
			}
			for k, v := range tt.expectedNames {
				assert.True(t, v, "the port name %s should have been part of the result", k)
			}
		})
	}
}
