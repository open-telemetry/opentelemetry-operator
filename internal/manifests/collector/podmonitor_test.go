// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"fmt"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, "monitoring", actual.Spec.PodMetricsEndpoints[0].Port)
	expectedSelectorLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
	}
	assert.Equal(t, expectedSelectorLabels, actual.Spec.Selector.MatchLabels)
}

func TestDesiredPodMonitorsWithPrometheus(t *testing.T) {
	params, err := newParams("", "testdata/prometheus-exporter.yaml")
	assert.NoError(t, err)
	params.OtelCol.Spec.Mode = v1beta1.ModeSidecar
	params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
	actual, err := PodMonitor(params)
	assert.NoError(t, err)
	assert.NotNil(t, actual)
	assert.Equal(t, fmt.Sprintf("%s-collector", params.OtelCol.Name), actual.Name)
	assert.Equal(t, params.OtelCol.Namespace, actual.Namespace)
	assert.Equal(t, "monitoring", actual.Spec.PodMetricsEndpoints[0].Port)
	assert.Equal(t, "prometheus-dev", actual.Spec.PodMetricsEndpoints[1].Port)
	assert.Equal(t, "prometheus-prod", actual.Spec.PodMetricsEndpoints[2].Port)
	expectedSelectorLabels := map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/component":  "opentelemetry-collector",
	}
	assert.Equal(t, expectedSelectorLabels, actual.Spec.Selector.MatchLabels)
}
