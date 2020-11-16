package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

func TestVolumeNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{}
	cfg := config.New()

	// test
	volumes := Volumes(cfg, otelcol)

	// verify
	assert.Len(t, volumes, 1)

	// check that it's the otc-internal volume, with the config map
	assert.Equal(t, naming.ConfigMapVolume(), volumes[0].Name)
}

func TestVolumeAllowsMoreToBeAdded(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Volumes: []corev1.Volume{{
				Name: "my-volume",
			}},
		},
	}
	cfg := config.New()

	// test
	volumes := Volumes(cfg, otelcol)

	// verify
	assert.Len(t, volumes, 2)

	// check that it's the otc-internal volume, with the config map
	assert.Equal(t, "my-volume", volumes[1].Name)
}
