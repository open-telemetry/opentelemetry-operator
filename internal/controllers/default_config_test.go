// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestGetParamsMergesDefaultCollectorConfig(t *testing.T) {
	defaultConfig := &v1beta1.Config{
		Processors: &v1beta1.AnyConfig{Object: map[string]any{"memory_limiter": map[string]any{"check_interval": "1s"}}},
	}

	instance := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{Name: "test-collector", Namespace: "test-ns"},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Config: v1beta1.Config{
				Receivers:  v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters:  v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Processors: &v1beta1.AnyConfig{Object: map[string]any{"batch": map[string]any{}}},
			},
		},
	}

	t.Run("merges the default when configured", func(t *testing.T) {
		r := &OpenTelemetryCollectorReconciler{
			Client: fake.NewClientBuilder().Build(),
			log:    logr.Discard(),
			config: config.Config{Internal: config.Internal{DefaultCollectorConfig: defaultConfig}},
		}

		params, err := r.GetParams(t.Context(), *instance.DeepCopy())
		require.NoError(t, err)

		require.NotNil(t, params.OtelCol.Spec.Config.Processors)
		// The collector keeps its own batch processor and gains the default memory_limiter.
		assert.Contains(t, params.OtelCol.Spec.Config.Processors.Object, "batch")
		assert.Contains(t, params.OtelCol.Spec.Config.Processors.Object, "memory_limiter")

		// The operator-level default is not mutated by the merge.
		assert.NotContains(t, defaultConfig.Processors.Object, "batch")
	})

	t.Run("leaves the config untouched when no default is configured", func(t *testing.T) {
		r := &OpenTelemetryCollectorReconciler{
			Client: fake.NewClientBuilder().Build(),
			log:    logr.Discard(),
			config: config.Config{},
		}

		params, err := r.GetParams(t.Context(), *instance.DeepCopy())
		require.NoError(t, err)

		require.NotNil(t, params.OtelCol.Spec.Config.Processors)
		assert.Contains(t, params.OtelCol.Spec.Config.Processors.Object, "batch")
		assert.NotContains(t, params.OtelCol.Spec.Config.Processors.Object, "memory_limiter")
	})
}
