// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

func sidecarParams() manifests.Params {
	return paramsWithMode(v1beta1.ModeSidecar)
}

func TestDesiredPodMonitors(t *testing.T) {
	params := sidecarParams()

	actual, err := PodMonitor(params)
	assert.NoError(t, err)
	assert.Nil(t, actual)

	params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
	actual, err = PodMonitor(params)
	assert.NoError(t, err)
	assert.NotNil(t, actual)
	assert.Equal(t, fmt.Sprintf("%s-collector", params.OtelCol.Name), actual.Name)
	assert.Equal(t, params.OtelCol.Namespace, actual.Namespace)
	assert.Equal(t, "monitoring", *actual.Spec.PodMetricsEndpoints[0].Port)
	expectedSelectorLabels := map[string]string{
		"sidecar.opentelemetry.io/injected": fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
	}
	assert.Equal(t, expectedSelectorLabels, actual.Spec.Selector.MatchLabels)
}

func TestDesiredPodMonitorsWithPrometheus(t *testing.T) {
	params, err := newParams("", "testdata/prometheus-exporter.yaml", nil)
	assert.NoError(t, err)
	params.OtelCol.Spec.Mode = v1beta1.ModeSidecar
	params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
	actual, err := PodMonitor(params)
	assert.NoError(t, err)
	assert.NotNil(t, actual)
	assert.Equal(t, fmt.Sprintf("%s-collector", params.OtelCol.Name), actual.Name)
	assert.Equal(t, params.OtelCol.Namespace, actual.Namespace)
	assert.Equal(t, "monitoring", *actual.Spec.PodMetricsEndpoints[0].Port)
	assert.Equal(t, "prometheus-dev", *actual.Spec.PodMetricsEndpoints[1].Port)
	assert.Equal(t, "prometheus-prod", *actual.Spec.PodMetricsEndpoints[2].Port)
	expectedSelectorLabels := map[string]string{
		"sidecar.opentelemetry.io/injected": fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
	}
	assert.Equal(t, expectedSelectorLabels, actual.Spec.Selector.MatchLabels)
}

func TestDesiredPodMonitorsWithExtraLabels(t *testing.T) {
	params, err := newParams("", "testdata/prometheus-exporter.yaml", nil)
	assert.NoError(t, err)
	params.OtelCol.Spec.Mode = v1beta1.ModeSidecar
	params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
	params.OtelCol.Spec.Observability.Metrics.ExtraLabels = map[string]string{
		"prometheus":  "kube-prometheus",
		"team":        "platform",
		"environment": "production",
	}

	actual, err := PodMonitor(params)
	assert.NoError(t, err)
	assert.NotNil(t, actual)

	expectedLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "default.test",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/name":       "test-collector",
		"app.kubernetes.io/version":    "latest",
		"prometheus":                   "kube-prometheus",
		"team":                         "platform",
		"environment":                  "production",
	}
	assert.Equal(t, expectedLabels, actual.Labels)
}

func TestDesiredPodMonitorsWithAnnotations(t *testing.T) {
	params, err := newParams("", "testdata/prometheus-exporter.yaml", nil)
	assert.NoError(t, err)
	params.Config.AnnotationsFilter = []string{"filtered.*"}
	params.OtelCol.Spec.Mode = v1beta1.ModeSidecar
	params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
	params.OtelCol.Annotations = map[string]string{
		"custom.annotation":   "custom-value",
		"filtered.annotation": "should-be-filtered",
	}

	actual, err := PodMonitor(params)
	assert.NoError(t, err)
	assert.NotNil(t, actual)
	assert.Equal(t, map[string]string{"custom.annotation": "custom-value"}, actual.Annotations)
}

func TestDesiredPodMonitorsPrometheusNotAvailable(t *testing.T) {
	params, err := newParams("", "testdata/prometheus-exporter.yaml", &config.Config{
		CollectorImage:              defaultCollectorImage,
		TargetAllocatorImage:        defaultTaAllocationImage,
		OpenShiftRoutesAvailability: openshift.RoutesAvailable,
		PrometheusCRAvailability:    prometheus.NotAvailable,
	})
	assert.NoError(t, err)
	params.OtelCol.Spec.Mode = v1beta1.ModeSidecar
	params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
	actual, err := PodMonitor(params)
	assert.NoError(t, err)
	assert.Nil(t, actual)
}
