// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestDesiredServiceMonitors(t *testing.T) {
	params := deploymentParams()

	actual, err := ServiceMonitor(params)
	assert.NoError(t, err)
	assert.Nil(t, actual)

	params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
	actual, err = ServiceMonitor(params)
	assert.NoError(t, err)
	assert.Nil(t, actual)

	// Check the monitoring SM
	actual, err = ServiceMonitorMonitoring(params)
	assert.NoError(t, err)
	assert.NotNil(t, actual)
	assert.Equal(t, fmt.Sprintf("%s-monitoring-collector", params.OtelCol.Name), actual.Name)
	assert.Equal(t, params.OtelCol.Namespace, actual.Namespace)
	assert.Equal(t, "monitoring", actual.Spec.Endpoints[0].Port)
	expectedSelectorLabelsMonitor := map[string]string{
		"app.kubernetes.io/component":                      "opentelemetry-collector",
		"app.kubernetes.io/instance":                       "default.test",
		"app.kubernetes.io/managed-by":                     "opentelemetry-operator",
		"app.kubernetes.io/part-of":                        "opentelemetry",
		"operator.opentelemetry.io/collector-service-type": "monitoring",
	}
	assert.Equal(t, expectedSelectorLabelsMonitor, actual.Spec.Selector.MatchLabels)

}

func TestDesiredServiceMonitorsWithPrometheus(t *testing.T) {
	params, err := newParams("", "testdata/prometheus-exporter.yaml")
	assert.NoError(t, err)
	params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
	actual, err := ServiceMonitor(params)
	assert.NoError(t, err)
	assert.NotNil(t, actual)
	assert.Equal(t, fmt.Sprintf("%s-collector", params.OtelCol.Name), actual.Name)
	assert.Equal(t, params.OtelCol.Namespace, actual.Namespace)
	assert.Equal(t, "prometheus-dev", actual.Spec.Endpoints[0].Port)
	assert.Equal(t, "prometheus-prod", actual.Spec.Endpoints[1].Port)
	expectedSelectorLabels := map[string]string{
		"app.kubernetes.io/component":                      "opentelemetry-collector",
		"app.kubernetes.io/instance":                       "default.test",
		"app.kubernetes.io/managed-by":                     "opentelemetry-operator",
		"app.kubernetes.io/part-of":                        "opentelemetry",
		"operator.opentelemetry.io/collector-service-type": "base",
	}
	assert.Equal(t, expectedSelectorLabels, actual.Spec.Selector.MatchLabels)
}

func TestDesiredServiceMonitorsPrometheusNotAvailable(t *testing.T) {
	params, err := newParams("", "testdata/prometheus-exporter.yaml", config.WithPrometheusCRAvailability(prometheus.NotAvailable))
	assert.NoError(t, err)
	params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
	actual, err := ServiceMonitor(params)
	assert.NoError(t, err)
	assert.Nil(t, actual)
}
