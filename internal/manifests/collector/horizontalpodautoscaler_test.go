// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
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
					Log: testLogger,
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
					switch metric.Resource.Name {
					case corev1.ResourceCPU:
						assert.Equal(t, cpuUtilization, *metric.Resource.Target.AverageUtilization)
					case corev1.ResourceMemory:
						assert.Equal(t, memoryUtilization, *metric.Resource.Target.AverageUtilization)
					default:
					}
				}
			})
		}
	}
}

func TestHPA_WithResourceMetrics(t *testing.T) {
	var minReplicas int32 = 2
	var maxReplicas int32 = 10

	cpuQuantity := resource.MustParse("100m")
	memoryQuantity := resource.MustParse("512Mi")

	resourceMetrics := []v1beta1.MetricSpec{
		{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:         autoscalingv2.AverageValueMetricType,
					AverageValue: &cpuQuantity,
				},
			},
		},
		{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:         autoscalingv2.AverageValueMetricType,
					AverageValue: &memoryQuantity,
				},
			},
		},
	}

	configuration := config.New()
	params := manifests.Params{
		Config: configuration,
		OtelCol: v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-metrics",
			},
			Spec: v1beta1.OpenTelemetryCollectorSpec{
				Autoscaler: &v1beta1.AutoscalerSpec{
					MinReplicas: &minReplicas,
					MaxReplicas: &maxReplicas,
					Metrics:     resourceMetrics,
				},
			},
		},
		Log: testLogger,
	}

	hpa, err := HorizontalPodAutoscaler(params)
	require.NoError(t, err)

	assert.Equal(t, "test-metrics-collector", hpa.Name)
	assert.Equal(t, &minReplicas, hpa.Spec.MinReplicas)
	assert.Equal(t, maxReplicas, hpa.Spec.MaxReplicas)
	assert.Equal(t, 2, len(hpa.Spec.Metrics))

	for _, metric := range hpa.Spec.Metrics {
		switch metric.Resource.Name {
		case corev1.ResourceCPU:
			assert.Equal(t, cpuQuantity, *metric.Resource.Target.AverageValue)
		case corev1.ResourceMemory:
			assert.Equal(t, memoryQuantity, *metric.Resource.Target.AverageValue)
		default:
			t.Fatalf("unexpected resource metric: %s", metric.Resource.Name)
		}
	}
}
