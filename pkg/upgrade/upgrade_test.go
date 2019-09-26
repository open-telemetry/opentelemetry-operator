package upgrade

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
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
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion,
		&v1alpha1.OpenTelemetryCollector{},
		&v1alpha1.OpenTelemetryCollectorList{},
	)
	cl := fake.NewFakeClient(objs...)

	// test
	assert.NoError(t, ManagedInstances(ctx, cl))

	// verify
	persisted := &v1alpha1.OpenTelemetryCollector{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
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
	objs := []runtime.Object{existing}

	s := scheme.Scheme
	s.AddKnownTypes(v1alpha1.SchemeGroupVersion,
		&v1alpha1.OpenTelemetryCollector{},
		&v1alpha1.OpenTelemetryCollectorList{},
	)
	cl := fake.NewFakeClient(objs...)

	// test
	assert.NoError(t, ManagedInstances(ctx, cl))

	// verify
	persisted := &v1alpha1.OpenTelemetryCollector{}
	assert.NoError(t, cl.Get(context.Background(), nsn, persisted))
	assert.Equal(t, "0.0.0", persisted.Status.Version)
}
