// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

const (
	testCollectorName      = "my-instance"
	testCollectorNamespace = "my-ns"
)

func TestLabelsWithSpecImage(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCollectorName,
			Namespace: testCollectorNamespace,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Image: "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.129.1",
			},
		},
	}

	cfg := config.Config{
		CollectorImage: "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.100.0",
	}

	labels := Labels(otelcol, testCollectorName, cfg, []string{})
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "0.129.1", labels["app.kubernetes.io/version"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-collector", labels["app.kubernetes.io/component"])
	assert.Equal(t, testCollectorName, labels["app.kubernetes.io/name"])
}

func TestLabelsFallbackToConfigCollectorImage(t *testing.T) {
	// When .spec.image is omitted/empty, the version label should be derived from cfg.CollectorImage
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCollectorName,
			Namespace: testCollectorNamespace,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.Config{
		CollectorImage: "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.129.1",
	}

	labels := Labels(otelcol, testCollectorName, cfg, []string{})
	assert.Equal(t, "opentelemetry-operator", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-ns.my-instance", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "0.129.1", labels["app.kubernetes.io/version"])
	assert.Equal(t, "opentelemetry", labels["app.kubernetes.io/part-of"])
	assert.Equal(t, "opentelemetry-collector", labels["app.kubernetes.io/component"])
}

func TestLabelsFallbackToLatestWhenAllEmpty(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCollectorName,
			Namespace: testCollectorNamespace,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.Config{}

	labels := Labels(otelcol, testCollectorName, cfg, []string{})
	assert.Equal(t, "latest", labels["app.kubernetes.io/version"])
}

func TestLabelsFilter(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCollectorName,
			Namespace: testCollectorNamespace,
			Labels: map[string]string{
				"test.bar.io": "foo",
				"test.foo.io": "bar",
			},
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.Config{
		CollectorImage: "otel/opentelemetry-collector-contrib:0.129.1",
		LabelsFilter:   []string{".*.bar.io"},
	}

	labels := Labels(otelcol, testCollectorName, cfg, cfg.LabelsFilter)
	assert.NotContains(t, labels, "test.bar.io")
	assert.Equal(t, "bar", labels["test.foo.io"])
	assert.Equal(t, "0.129.1", labels["app.kubernetes.io/version"])
}
