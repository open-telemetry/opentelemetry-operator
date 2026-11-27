// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

func TestLabelsFilterAppliesToAllObjects(t *testing.T) {
	filtered := map[string]string{
		"argocd.argoproj.io/instance": "my-app",
		"filter.me":                   "1",
	}
	labels := map[string]string{"keep": "1"}
	maps.Copy(labels, filtered)

	params := manifests.Params{
		Config: config.Config{LabelsFilter: []string{"argocd.argoproj.io/*", "filter.*"}},
		Log:    logger,
		OpAMPBridge: v1alpha1.OpAMPBridge{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-instance",
				Namespace: "my-namespace",
				Labels:    labels,
			},
			Spec: v1alpha1.OpAMPBridgeSpec{
				Endpoint: "ws://localhost:4320/v1/opamp",
			},
		},
	}

	testCases := []struct {
		name  string
		build func(manifests.Params) (metav1.Object, error)
	}{
		{"ConfigMap", func(p manifests.Params) (metav1.Object, error) { return ConfigMap(p) }},
		{"Service", func(p manifests.Params) (metav1.Object, error) { return Service(p), nil }},
		{"ServiceAccount", func(p manifests.Params) (metav1.Object, error) { return ServiceAccount(p), nil }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := tc.build(params)
			require.NoError(t, err)
			require.NotNil(t, obj)

			got := obj.GetLabels()
			assert.Equal(t, "1", got["keep"])
			for k := range filtered {
				assert.NotContains(t, got, k)
			}
		})
	}
}
