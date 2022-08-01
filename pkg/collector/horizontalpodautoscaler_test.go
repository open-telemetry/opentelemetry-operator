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
	"fmt"
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
		autoscalingVersion string
	}
	v2Test := test{"V2", "v2"}
	v2beta2Test := test{"V2Beta2", config.AutoscalingVersionV2Beta2}
	tests := []test{v2Test, v2beta2Test}

	var minReplicas int32 = 3
	var maxReplicas int32 = 5

	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Replicas:    &minReplicas,
			MaxReplicas: &maxReplicas,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockAutoDetector := &mockAutoDetect{
				HPAVersionFunc: func() (string, error) {
					return test.autoscalingVersion, nil
				},
			}
			configuration := config.New(config.WithAutoDetect(mockAutoDetector))
			raw := HorizontalPodAutoscaler(configuration, logger, otelcol)
			err := configuration.AutoDetect()
			if err != nil {
				t.Errorf("configuration.autodetect failed %v", err)
			}

			logger.Info("Running ", t.Name(), "with autoscaling version", configuration.AutoscalingVersion()) // FIXME this print nothing
			fmt.Printf("In test %s using autoscaling version %s\n", t.Name(), configuration.AutoscalingVersion())
			if configuration.AutoscalingVersion() == config.AutoscalingVersionV2Beta2 {
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

var _ autodetect.AutoDetect = (*mockAutoDetect)(nil)

type mockAutoDetect struct {
	PlatformFunc   func() (platform.Platform, error)
	HPAVersionFunc func() (string, error)
}

func (m *mockAutoDetect) HPAVersion() (string, error) {
	return m.HPAVersionFunc()
}

func (m *mockAutoDetect) Platform() (platform.Platform, error) {
	if m.PlatformFunc != nil {
		return m.PlatformFunc()
	}
	return platform.Unknown, nil
}
