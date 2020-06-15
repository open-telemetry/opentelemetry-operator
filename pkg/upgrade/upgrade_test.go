package upgrade

import (
	"context"
	"testing"

	fakemon "github.com/coreos/prometheus-operator/pkg/client/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/client"
	fakeotclient "github.com/open-telemetry/opentelemetry-operator/pkg/client/versioned/fake"
)

func TestVersionUpgradeToLatest(t *testing.T) {
	// prepare
	ctx := context.WithValue(context.Background(), opentelemetry.ContextLogger, logf.Log)

	nsn := types.NamespacedName{Name: "my-instance"}
	existing := &v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
	}
	existing.Status.Version = "0.0.1" // this is the first version we have an upgrade function

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion,
		&v1alpha1.OpenTelemetryCollector{},
		&v1alpha1.OpenTelemetryCollectorList{},
	)
	cl := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(existing),
	}

	// test
	assert.NoError(t, ManagedInstances(ctx, cl))

	// verify
	persisted, err := cl.OpenTelemetry.OpentelemetryV1alpha1().OpenTelemetryCollectors("").Get(context.Background(), nsn.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, latest.v, persisted.Status.Version)
}

func TestUnknownVersion(t *testing.T) {
	// prepare
	ctx := context.WithValue(context.Background(), opentelemetry.ContextLogger, logf.Log)
	nsn := types.NamespacedName{Name: "my-instance"}
	existing := &v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
	}
	existing.Status.Version = "0.0.0" // we don't know how to upgrade from 0.0.0

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion,
		&v1alpha1.OpenTelemetryCollector{},
		&v1alpha1.OpenTelemetryCollectorList{},
	)
	cl := &client.Clientset{
		Kubernetes:    fake.NewSimpleClientset(),
		Monitoring:    fakemon.NewSimpleClientset(),
		OpenTelemetry: fakeotclient.NewSimpleClientset(existing),
	}

	// test
	assert.NoError(t, ManagedInstances(ctx, cl))

	// verify
	persisted, err := cl.OpenTelemetry.OpentelemetryV1alpha1().OpenTelemetryCollectors("").Get(context.Background(), nsn.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "0.0.0", persisted.Status.Version)
}
