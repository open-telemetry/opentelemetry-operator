package opentelemetrycollector

import (
	"testing"

	fakemon "github.com/coreos/prometheus-operator/pkg/client/versioned/fake"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
)

func TestProperServiceMonitor(t *testing.T) {
	// test
	s := serviceMonitor(ctx)
	backingSvc := monitoringService(ctx)

	// verify
	assert.Equal(t, "my-otelcol-collector", s.Name)
	assert.Equal(t, "custom-annotation-value", s.Annotations["custom-annotation"])
	assert.Equal(t, "custom-value", s.Labels["custom-label"])
	assert.Equal(t, s.Name, s.Labels["app.kubernetes.io/name"])
	assert.Equal(t, backingSvc.Labels, s.Spec.Selector.MatchLabels)
}

func TestProperReconcileServiceMonitor(t *testing.T) {
	// prepare
	viper.Set(opentelemetry.SvcMonitorAvailable, true)
	defer viper.Reset()

	clients := &Clients{
		client:    fake.NewFakeClient(instance),
		monclient: fakemon.NewSimpleClientset().MonitoringV1(),
	}
	reconciler := New(schem, clients)

	// test
	req := reconcile.Request{}
	_, err := reconciler.Reconcile(req)
	assert.NoError(t, err)

	// verify
	list, err := clients.monclient.ServiceMonitors(instance.Namespace).List(metav1.ListOptions{})
	require.NoError(t, err)

	// we assert the correctness of the service in another test
	assert.Len(t, list.Items, 1)

	// we assert the correctness of the reference in another test
	for _, item := range list.Items {
		assert.Len(t, item.OwnerReferences, 1)
	}
}

func TestUpdateServiceMonitor(t *testing.T) {
	// prepare
	viper.Set(opentelemetry.SvcMonitorAvailable, true)
	defer viper.Reset()

	c := serviceMonitor(ctx)
	c.Annotations = nil
	c.Labels = nil

	clients := &Clients{
		monclient: fakemon.NewSimpleClientset(c).MonitoringV1(),
	}
	reconciler := New(schem, clients)

	// sanity check
	_, err := clients.monclient.ServiceMonitors(c.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// test
	assert.NoError(t, reconciler.reconcileServiceMonitor(ctx))

	// verify
	_, err = clients.monclient.ServiceMonitors(c.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.NoError(t, err)
}

func TestDeleteExtraServiceMonitor(t *testing.T) {
	// prepare
	viper.Set(opentelemetry.SvcMonitorAvailable, true)
	defer viper.Reset()

	c := serviceMonitor(ctx)
	c.Name = "extra-service"

	clients := &Clients{
		monclient: fakemon.NewSimpleClientset(c).MonitoringV1(),
	}
	reconciler := New(schem, clients)

	// sanity check
	_, err := clients.monclient.ServiceMonitors(c.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	// test
	assert.NoError(t, reconciler.reconcileServiceMonitor(ctx))

	// verify
	persisted, err := clients.monclient.ServiceMonitors(c.Namespace).Get(c.Name, metav1.GetOptions{})
	assert.Nil(t, persisted)
	assert.Error(t, err) // not found
}
