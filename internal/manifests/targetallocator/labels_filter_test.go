// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestLabelsFilterAppliesToAllObjects(t *testing.T) {
	filtered := map[string]string{
		"argocd.argoproj.io/instance": "my-app",
		"filter.me":                   "1",
	}
	labels := map[string]string{"keep": "1"}
	maps.Copy(labels, filtered)

	maxUnavailable := intstr.FromInt32(1)

	params := Params{
		Config:          config.Config{LabelsFilter: []string{"argocd.argoproj.io/*", "filter.*"}},
		Log:             logger,
		Collector:       collectorInstance(),
		TargetAllocator: targetAllocatorInstance(),
	}
	params.TargetAllocator.Labels = labels
	params.TargetAllocator.Spec.AllocationStrategy = v1beta1.TargetAllocatorAllocationStrategyConsistentHashing
	params.TargetAllocator.Spec.PodDisruptionBudget = &v1beta1.PodDisruptionBudgetSpec{MaxUnavailable: &maxUnavailable}

	testCases := []struct {
		name  string
		build func(Params) (metav1.Object, error)
	}{
		{"ConfigMap", func(p Params) (metav1.Object, error) { return ConfigMap(p) }},
		{"Deployment", func(p Params) (metav1.Object, error) { return Deployment(p) }},
		{"Service", func(p Params) (metav1.Object, error) { return Service(p), nil }},
		{"ServiceAccount", func(p Params) (metav1.Object, error) { return ServiceAccount(p), nil }},
		{"ServiceMonitor", func(p Params) (metav1.Object, error) { return ServiceMonitor(p), nil }},
		{"PodDisruptionBudget", func(p Params) (metav1.Object, error) { return PodDisruptionBudget(p) }},
		{"SelfSignedIssuer", func(p Params) (metav1.Object, error) { return SelfSignedIssuer(p), nil }},
		{"CAIssuer", func(p Params) (metav1.Object, error) { return CAIssuer(p), nil }},
		{"CACertificate", func(p Params) (metav1.Object, error) { return CACertificate(p), nil }},
		{"ServingCertificate", func(p Params) (metav1.Object, error) { return ServingCertificate(p), nil }},
		{"ClientCertificate", func(p Params) (metav1.Object, error) { return ClientCertificate(p), nil }},
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
