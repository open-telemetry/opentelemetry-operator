package collector_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/stretchr/testify/assert"
)

var logger = logf.Log.WithName("unit-tests")

func TestContainerNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{}
	cfg := config.New(config.WithCollectorImage("default-image"))

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "default-image", c.Image)
}

func TestContainerWithImageOverridden(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "overridden-image",
		},
	}
	cfg := config.New(config.WithCollectorImage("default-image"))

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "overridden-image", c.Image)
}

func TestContainerConfigFlagIsIgnored(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Args: map[string]string{
				"key":    "value",
				"config": "/some-custom-file.yaml",
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Len(t, c.Args, 2)
	assert.Contains(t, c.Args, "--key=value")
	assert.NotContains(t, c.Args, "--config=/some-custom-file.yaml")
}

func TestContainerCustomVolumes(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			VolumeMounts: []corev1.VolumeMount{{
				Name: "custom-volume-mount",
			}},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Len(t, c.VolumeMounts, 2)
	assert.Equal(t, "custom-volume-mount", c.VolumeMounts[1].Name)
}

func TestContainerEnvVarsOverridden(t *testing.T) {
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Env: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
			},
		},
	}

	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Len(t, c.Env, 1)
	assert.Equal(t, "foo", c.Env[0].Name)
	assert.Equal(t, "bar", c.Env[0].Value)
}

func TestContainerEmptyEnvVarsByDefault(t *testing.T) {
	cfg := config.New()
	otelcol := v1alpha1.OpenTelemetryCollector{}

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Empty(t, c.Env)
}

func TestContainerForDistribution(t *testing.T) {
	// prepare
	cfg := config.New()
	cfg.SetDistributions([]v1alpha1.OpenTelemetryCollectorDistribution{
		{
			ObjectMeta: v1.ObjectMeta{
				Namespace: "my-ns",
				Name:      "my-dist",
			},
			Image:   "quay.io/myns/my-otelcol:v1.0.0",
			Command: []string{"/path/to/command"},
		},
	})

	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "my-ns",
			Name:      "my-otelcol",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			DistributionName: "my-dist",
		},
	}

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Equal(t, []string{"/path/to/command"}, c.Command)
	assert.Equal(t, "quay.io/myns/my-otelcol:v1.0.0", c.Image)
}
