// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

// TestGetParams_TargetAllocatorMTLSViaLabel guards against a regression of
// https://github.com/open-telemetry/opentelemetry-operator/issues/4297, where a
// collector associated with a standalone TargetAllocator CR via the
// opentelemetry.io/target-allocator label kept talking to it over plaintext HTTP
// even when the TargetAllocator had mTLS enabled. It also pins down that the
// mTLS endpoint is addressed by the TargetAllocator CR's own name, not the
// collector's name, since the two differ in the label-based association.
func TestGetParams_TargetAllocatorMTLSViaLabel(t *testing.T) {
	ta := &v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{Name: "ta", Namespace: "test-ns"},
		Spec: v1alpha1.TargetAllocatorSpec{
			Mtls: &v1beta1.TargetAllocatorMTLS{Enabled: true},
		},
	}
	instance := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
			Labels:    map[string]string{constants.LabelTargetAllocator: "ta"},
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Config: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]any{
						"prometheus": map[string]any{
							"config": map[string]any{},
						},
					},
				},
				Exporters: v1beta1.AnyConfig{
					Object: map[string]any{
						"debug": map[string]any{},
					},
				},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"metrics": {
							Receivers: []string{"prometheus"},
							Exporters: []string{"debug"},
						},
					},
				},
			},
		},
	}

	cl := fake.NewClientBuilder().WithScheme(reconcilerTestScheme).WithObjects(ta).Build()
	r := &OpenTelemetryCollectorReconciler{
		Client: cl,
		config: config.New(),
		log:    logr.Discard(),
		scheme: reconcilerTestScheme,
	}

	params, err := r.GetParams(context.Background(), instance)
	require.NoError(t, err)
	require.NotNil(t, params.TargetAllocator)
	assert.True(t, manifestutils.IsTAMTLSEnabled(params.TargetAllocator))

	cm, err := collector.ConfigMap(params)
	require.NoError(t, err)
	// The TargetAllocator CR is named "ta" while the collector is named "test": the
	// rendered endpoint must resolve to the TA's own Service (ta-targetallocator),
	// not one derived from the collector's name.
	assert.Contains(t, cm.Data["collector.yaml"], "https://ta-targetallocator:443")
	assert.NotContains(t, cm.Data["collector.yaml"], "test-targetallocator")
}
