package opentelemetrycollector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

func TestNilLabels(t *testing.T) {
	// prepare
	instance := &v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-otelcol",
			Namespace: "observability",
			Labels:    nil,
		},
	}
	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, instance)

	// test
	labels := commonLabels(ctx)

	// verify
	assert.NotNil(t, labels)
	assert.Contains(t, labels, "app.kubernetes.io/managed-by")
	assert.Contains(t, labels, "app.kubernetes.io/instance")
	assert.Contains(t, labels, "app.kubernetes.io/part-of")
	assert.Contains(t, labels, "app.kubernetes.io/component")
}

func TestInstanceName(t *testing.T) {
	// prepare
	instance := &v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-otelcol",
			Namespace: "observability",
			Labels:    nil,
		},
	}
	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, instance)

	// test
	labels := commonLabels(ctx)

	// verify
	assert.Equal(t, labels["app.kubernetes.io/instance"], "observability.my-otelcol")
}
