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
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
)

func TestHPA(t *testing.T) {
	type test struct {
		name string
	}
	v2Test := test{}
	tests := []test{v2Test}

	var minReplicas int32 = 3
	var maxReplicas int32 = 5
	var cpuUtilization int32 = 66
	var memoryUtilization int32 = 77

	otelcols := []v1beta1.OpenTelemetryCollector{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
			Spec: v1beta1.OpenTelemetryCollectorSpec{
				Autoscaler: &v1beta1.AutoscalerSpec{
					MinReplicas:             &minReplicas,
					MaxReplicas:             &maxReplicas,
					TargetCPUUtilization:    &cpuUtilization,
					TargetMemoryUtilization: &memoryUtilization,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
			Spec: v1beta1.OpenTelemetryCollectorSpec{
				Autoscaler: &v1beta1.AutoscalerSpec{
					MinReplicas:             &minReplicas,
					MaxReplicas:             &maxReplicas,
					TargetCPUUtilization:    &cpuUtilization,
					TargetMemoryUtilization: &memoryUtilization,
				},
			},
		},
	}

	for _, otelcol := range otelcols {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				configuration := config.New()
				params := manifests.Params{
					Config: configuration,
					OtelCol: v1beta1.OpenTelemetryCollector{
						ObjectMeta: otelcol.ObjectMeta,
						Spec: v1beta1.OpenTelemetryCollectorSpec{
							Autoscaler: &v1beta1.AutoscalerSpec{
								MinReplicas:             otelcol.Spec.Autoscaler.MinReplicas,
								MaxReplicas:             otelcol.Spec.Autoscaler.MaxReplicas,
								TargetCPUUtilization:    otelcol.Spec.Autoscaler.TargetCPUUtilization,
								TargetMemoryUtilization: otelcol.Spec.Autoscaler.TargetMemoryUtilization,
							},
						},
					},
					Log: logger,
				}
				hpa, err := HorizontalPodAutoscaler(params)
				require.NoError(t, err)

				// verify
				assert.Equal(t, "my-instance-collector", hpa.Name)
				assert.Equal(t, "my-instance-collector", hpa.Labels["app.kubernetes.io/name"])
				assert.Equal(t, &minReplicas, hpa.Spec.MinReplicas)
				assert.Equal(t, maxReplicas, hpa.Spec.MaxReplicas)
				assert.Equal(t, 2, len(hpa.Spec.Metrics))

				for _, metric := range hpa.Spec.Metrics {
					if metric.Resource.Name == corev1.ResourceCPU {
						assert.Equal(t, cpuUtilization, *metric.Resource.Target.AverageUtilization)
					} else if metric.Resource.Name == corev1.ResourceMemory {
						assert.Equal(t, memoryUtilization, *metric.Resource.Target.AverageUtilization)
					}
				}
			})
		}
	}

}
