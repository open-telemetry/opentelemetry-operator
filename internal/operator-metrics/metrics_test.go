package operatormetrics

import (
	"context"
	"os"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewOperatorMetrics(t *testing.T) {
	config := &rest.Config{}
	scheme := runtime.NewScheme()

	om, err := NewOperatorMetrics(config, scheme)
	assert.NoError(t, err)
	assert.NotNil(t, om.kubeClient)
}

func TestOperatorMetrics_Start(t *testing.T) {
	scheme := runtime.NewScheme()
	err := monitoringv1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	om := OperatorMetrics{
		kubeClient: fakeClient,
	}

	tmpfile, err := os.CreateTemp("", "namespace")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte("test-namespace"))
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	oldNamespaceFile := namespaceFile
	namespaceFile = tmpfile.Name()
	defer func() { namespaceFile = oldNamespaceFile }()

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)

	go func() {
		errChan <- om.Start(ctx)
	}()

	cancel()

	err = <-errChan
	assert.NoError(t, err)

	var sm monitoringv1.ServiceMonitor
	err = fakeClient.Get(context.Background(), client.ObjectKey{
		Name:      "opentelemetry-operator-metrics-monitor",
		Namespace: "test-namespace",
	}, &sm)
	assert.Error(t, err)
}
