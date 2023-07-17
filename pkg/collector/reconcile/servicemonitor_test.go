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

package reconcile

import (
	"context"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func TestDesiredServiceMonitors(t *testing.T) {
	params := params()

	actual := desiredServiceMonitors(context.Background(), params)
	assert.NotNil(t, actual)
}

func TestExpectedServiceMonitors(t *testing.T) {
	originalVal := featuregate.PrometheusOperatorIsAvailable.IsEnabled()
	require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.PrometheusOperatorIsAvailable.ID(), false))
	t.Cleanup(func() {
		require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.PrometheusOperatorIsAvailable.ID(), originalVal))
	})

	t.Run("should create the service monitor", func(t *testing.T) {
		p := params()
		p.Instance.Spec.Observability.Metrics.EnableMetrics = true

		err := expectedServiceMonitors(
			context.Background(),
			p,
			[]monitoringv1.ServiceMonitor{servicemonitor("test-collector")},
		)
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &monitoringv1.ServiceMonitor{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
}

func TestDeleteServiceMonitors(t *testing.T) {
	t.Run("should delete excess service monitors", func(t *testing.T) {
		name := "sm-to-delete"
		deleteServiceMonitor := servicemonitor(name)
		createObjectIfNotExists(t, name, &deleteServiceMonitor)

		exists, err := populateObjectIfExists(t, &monitoringv1.ServiceMonitor{}, types.NamespacedName{Namespace: "default", Name: name})
		assert.NoError(t, err)
		assert.True(t, exists)

		desired := desiredServiceMonitors(context.Background(), params())
		err = deleteServiceMonitors(context.Background(), params(), desired)
		assert.NoError(t, err)

		exists, err = populateObjectIfExists(t, &v1.Service{}, types.NamespacedName{Namespace: "default", Name: name})
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func servicemonitor(name string) monitoringv1.ServiceMonitor {
	return monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{"default"},
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "monitoring",
				},
			},
		},
	}
}

func TestServiceMonitors(t *testing.T) {
	t.Run("not enabled", func(t *testing.T) {
		ctx := context.Background()
		err := ServiceMonitors(ctx, params())
		assert.Nil(t, err)
	})

	t.Run("enabled but featuregate not enabled", func(t *testing.T) {
		ctx := context.Background()
		p := params()
		p.Instance.Spec.Observability.Metrics.CreateServiceMonitors = true
		err := ServiceMonitors(ctx, p)
		assert.Nil(t, err)
	})

	t.Run("enabled and featuregate enabled", func(t *testing.T) {
		originalVal := featuregate.PrometheusOperatorIsAvailable.IsEnabled()
		require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.PrometheusOperatorIsAvailable.ID(), false))
		t.Cleanup(func() {
			require.NoError(t, colfeaturegate.GlobalRegistry().Set(featuregate.PrometheusOperatorIsAvailable.ID(), originalVal))
		})

		ctx := context.Background()
		p := params()
		p.Instance.Spec.Observability.Metrics.EnableMetrics = true
		err := ServiceMonitors(ctx, p)
		assert.Nil(t, err)
	})

}
