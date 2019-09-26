package opentelemetrycollector

import (
	"context"
	"os"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

var (
	instance   *v1alpha1.OpenTelemetryCollector
	ctx        context.Context
	reconciler *ReconcileOpenTelemetryCollector
	schem      *runtime.Scheme
	cl         client.Client
)

// TestMain ensures that all tests in this package have a fresh and sane instance of the common resources
func TestMain(m *testing.M) {
	schem = scheme.Scheme
	schem.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.OpenTelemetryCollector{})

	gvk := v1alpha1.SchemeGroupVersion.WithKind("OpenTelemetryCollector")
	instance = &v1alpha1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			// TypeMeta is added by Kubernetes, there's no need for consumers to set this manually
			Kind:       gvk.Kind,
			APIVersion: gvk.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-otelcol",
			Namespace:   "observability",
			Labels:      map[string]string{"custom-label": "custom-value"},
			Annotations: map[string]string{"custom-annotation": "custom-annotation-value"},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: "the-config-in-yaml-format",
		},
	}
	ctx = context.WithValue(context.Background(), opentelemetry.ContextInstance, instance)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, logf.Log.WithName("unit-tests"))

	cl = fake.NewFakeClient(instance)
	reconciler = New(cl, schem)

	os.Exit(m.Run())
}
