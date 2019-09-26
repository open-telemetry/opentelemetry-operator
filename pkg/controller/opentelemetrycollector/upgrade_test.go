package opentelemetrycollector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/version"
)

func TestApplyUpgrades(t *testing.T) {
	// prepare
	instance := &v1alpha1.OpenTelemetryCollector{
		Status: v1alpha1.OpenTelemetryCollectorStatus{
			Version: "0.0.1",
		},
	}
	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, instance)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, logf.Log.WithName("unit-tests"))
	cl = fake.NewFakeClient(instance)
	reconciler = New(cl, schem)

	// test
	err := reconciler.applyUpgrades(ctx)

	// verify
	assert.NoError(t, err)
	assert.NotEqual(t, "0.0.1", instance.Status.Version)
	assert.Equal(t, version.Get().OpenTelemetryCollector, instance.Status.Version)
}
