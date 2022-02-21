package collector_test

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHPA(t *testing.T) {
	// prepare
	enable := true
	var minReplicas int32 = 3
	var maxReplicas int32 = 5

	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			MinReplicas: &minReplicas,
			MaxReplicas: &maxReplicas,
			Autoscale:   &enable,
		},
	}

	cfg := config.New()
	hpa := HorizontalPodAutoscaler(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "my-instance-collector", hpa.Name)
	assert.Equal(t, "my-instance-collector", hpa.Labels["app.kubernetes.io/name"])
	assert.Equal(t, int32(3), *hpa.Spec.MinReplicas)
	assert.Equal(t, int32(5), hpa.Spec.MaxReplicas)
	assert.Equal(t, int32(90), *hpa.Spec.TargetCPUUtilizationPercentage)
}
