// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

func TestLabelsFilterAppliesToAllObjects(t *testing.T) {
	filtered := map[string]string{
		"argocd.argoproj.io/instance": "my-app",
		"filter.me":                   "1",
	}

	withLabels := func(params manifests.Params) manifests.Params {
		params.OtelCol.Labels = map[string]string{"keep": "1"}
		for k, v := range filtered {
			params.OtelCol.Labels[k] = v
		}
		params.Config.LabelsFilter = []string{"argocd.argoproj.io/*", "filter.*"}
		params.OtelCol.Spec.Observability.Metrics.EnableMetrics = true
		return params
	}

	extensionParams := func() manifests.Params {
		params := deploymentParams()
		params.OtelCol.Spec.Config.Service.Extensions = []string{"jaeger_query"}
		params.OtelCol.Spec.Config.Extensions = &v1beta1.AnyConfig{
			Object: map[string]any{
				"jaeger_query": map[string]any{
					"http": map[string]any{"endpoint": "0.0.0.0:16686"},
				},
			},
		}
		return params
	}

	testCases := []struct {
		name   string
		params manifests.Params
		build  func(manifests.Params) (metav1.Object, error)
	}{
		{
			name:   "Service",
			params: deploymentParams(),
			build: func(p manifests.Params) (metav1.Object, error) {
				return Service(p)
			},
		},
		{
			name:   "HeadlessService",
			params: deploymentParams(),
			build: func(p manifests.Params) (metav1.Object, error) {
				return HeadlessService(p)
			},
		},
		{
			name:   "MonitoringService",
			params: deploymentParams(),
			build: func(p manifests.Params) (metav1.Object, error) {
				return MonitoringService(p)
			},
		},
		{
			name:   "ExtensionService",
			params: extensionParams(),
			build: func(p manifests.Params) (metav1.Object, error) {
				return ExtensionService(p)
			},
		},
		{
			name:   "ServiceAccount",
			params: deploymentParams(),
			build: func(p manifests.Params) (metav1.Object, error) {
				return ServiceAccount(p)
			},
		},
		{
			name:   "ConfigMap",
			params: deploymentParams(),
			build: func(p manifests.Params) (metav1.Object, error) {
				return ConfigMap(p)
			},
		},
		{
			name:   "ServiceMonitorMonitoring",
			params: deploymentParams(),
			build: func(p manifests.Params) (metav1.Object, error) {
				return ServiceMonitorMonitoring(p)
			},
		},
		{
			name:   "PodMonitor",
			params: paramsWithMode(v1beta1.ModeSidecar),
			build: func(p manifests.Params) (metav1.Object, error) {
				return PodMonitor(p)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := tc.build(withLabels(tc.params))
			require.NoError(t, err)
			require.NotNil(t, obj)

			labels := obj.GetLabels()
			assert.Equal(t, "1", labels["keep"])
			for k := range filtered {
				assert.NotContains(t, labels, k)
			}
		})
	}
}
