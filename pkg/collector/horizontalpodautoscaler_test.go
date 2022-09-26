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

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

func TestHPA(t *testing.T) {
	type test struct {
		name               string
		autoscalingVersion autodetect.AutoscalingVersion
	}
	v2Test := test{autodetect.AutoscalingVersionV2.String(), autodetect.AutoscalingVersionV2}
	v2beta2Test := test{autodetect.AutoscalingVersionV2Beta2.String(), autodetect.AutoscalingVersionV2Beta2}
	tests := []test{v2Test, v2beta2Test}

	var minReplicas int32 = 3
	var maxReplicas int32 = 5
	var cpuUtilization int32 = 90

	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Replicas:    &minReplicas,
			MaxReplicas: &maxReplicas,
			Autoscaler: &v1alpha1.AutoscalerSpec{
				TargetCPUUtilization: &cpuUtilization,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockAutoDetector := &mockAutoDetect{
				HPAVersionFunc: func() (autodetect.AutoscalingVersion, error) {
					return test.autoscalingVersion, nil
				},
			}
			configuration := config.New(config.WithAutoDetect(mockAutoDetector))
			err := configuration.AutoDetect()
			assert.NoError(t, err)
			raw := HorizontalPodAutoscaler(configuration, logger, otelcol)

			if configuration.AutoscalingVersion() == autodetect.AutoscalingVersionV2Beta2 {
				hpa := raw.(*autoscalingv2beta2.HorizontalPodAutoscaler)

				// verify
				assert.Equal(t, "my-instance-collector", hpa.Name)
				assert.Equal(t, "my-instance-collector", hpa.Labels["app.kubernetes.io/name"])
				assert.Equal(t, int32(3), *hpa.Spec.MinReplicas)
				assert.Equal(t, int32(5), hpa.Spec.MaxReplicas)
				assert.Equal(t, 1, len(hpa.Spec.Metrics))
				assert.Equal(t, corev1.ResourceCPU, hpa.Spec.Metrics[0].Resource.Name)
				assert.Equal(t, int32(90), *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
			} else {
				hpa := raw.(*autoscalingv2.HorizontalPodAutoscaler)

				// verify
				assert.Equal(t, "my-instance-collector", hpa.Name)
				assert.Equal(t, "my-instance-collector", hpa.Labels["app.kubernetes.io/name"])
				assert.Equal(t, int32(3), *hpa.Spec.MinReplicas)
				assert.Equal(t, int32(5), hpa.Spec.MaxReplicas)
				assert.Equal(t, 1, len(hpa.Spec.Metrics))
				assert.Equal(t, corev1.ResourceCPU, hpa.Spec.Metrics[0].Resource.Name)
				assert.Equal(t, int32(90), *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
			}
		})
	}
}

func TestConvertToV2beta2Behavior(t *testing.T) {
	ten := int32(10)
	thirty := int32(30)
	max := autoscalingv2.MaxChangePolicySelect
	min := autoscalingv2.MinChangePolicySelect

	v2ScaleUp := &autoscalingv2.HPAScalingRules{
		StabilizationWindowSeconds: &thirty,
		SelectPolicy:               &max,
	}

	v2ScaleDown := &autoscalingv2.HPAScalingRules{
		StabilizationWindowSeconds: &ten,
		SelectPolicy:               &min,
	}

	v2Behavior := &autoscalingv2.HorizontalPodAutoscalerBehavior{
		ScaleUp:   v2ScaleUp,
		ScaleDown: v2ScaleDown,
	}

	v2Beta2Behavior := ConvertToV2beta2Behavior(*v2Behavior)

	assert.Equal(t, thirty, *v2Beta2Behavior.ScaleUp.StabilizationWindowSeconds)
	assert.Equal(t, ten, *v2Beta2Behavior.ScaleDown.StabilizationWindowSeconds)
	assert.Equal(t, autoscalingv2beta2.MaxPolicySelect, *v2Beta2Behavior.ScaleUp.SelectPolicy)
	assert.EqualValues(t, autoscalingv2beta2.MinPolicySelect, *v2Beta2Behavior.ScaleDown.SelectPolicy)
}

func TestConvertToV2Beta2HPAScalingPolicy(t *testing.T) {
	v2Policy := autoscalingv2.HPAScalingPolicy{
		Type:          autoscalingv2.PodsScalingPolicy,
		Value:         5,
		PeriodSeconds: 10,
	}

	v2Beta2Policy := ConvertToV2Beta2HPAScalingPolicy(v2Policy)
	assert.Equal(t, autoscalingv2beta2.PodsScalingPolicy, v2Beta2Policy.Type)
	assert.Equal(t, int32(5), v2Beta2Policy.Value)
	assert.Equal(t, int32(10), v2Beta2Policy.PeriodSeconds)

}

func TestConvertToV2Beta2SelectPolicy(t *testing.T) {
	min := autoscalingv2.MinChangePolicySelect
	max := autoscalingv2.MaxChangePolicySelect
	disabled := autoscalingv2.DisabledPolicySelect

	assert.Equal(t, autoscalingv2beta2.MinPolicySelect, ConvertToV2Beta2SelectPolicy(min))
	assert.Equal(t, autoscalingv2beta2.MaxPolicySelect, ConvertToV2Beta2SelectPolicy(max))
	assert.Equal(t, autoscalingv2beta2.DisabledPolicySelect, ConvertToV2Beta2SelectPolicy(disabled))
}

var _ autodetect.AutoDetect = (*mockAutoDetect)(nil)

type mockAutoDetect struct {
	PlatformFunc   func() (platform.Platform, error)
	HPAVersionFunc func() (autodetect.AutoscalingVersion, error)
}

func (m *mockAutoDetect) HPAVersion() (autodetect.AutoscalingVersion, error) {
	return m.HPAVersionFunc()
}

func (m *mockAutoDetect) Platform() (platform.Platform, error) {
	if m.PlatformFunc != nil {
		return m.PlatformFunc()
	}
	return platform.Unknown, nil
}
