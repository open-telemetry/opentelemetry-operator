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

package collector

import (
	"github.com/go-logr/logr"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

const defaultCPUTarget int32 = 90

func HorizontalPodAutoscaler(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) client.Object {
	autoscalingVersion := cfg.AutoscalingVersion()

	labels := Labels(otelcol, cfg.LabelsFilter())
	labels["app.kubernetes.io/name"] = naming.Collector(otelcol)

	annotations := Annotations(otelcol)
	var cpuTarget int32
	if otelcol.Spec.TargetCPUUtilization != nil {
		cpuTarget = *otelcol.Spec.TargetCPUUtilization
	} else {
		cpuTarget = defaultCPUTarget
	}
	var result client.Object

	objectMeta := metav1.ObjectMeta{
		Name:        naming.HorizontalPodAutoscaler(otelcol),
		Namespace:   otelcol.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	if autoscalingVersion == autodetect.AutoscalingVersionV2Beta2 {
		targetCPUUtilization := autoscalingv2beta2.MetricSpec{
			Type: autoscalingv2beta2.ResourceMetricSourceType,
			Resource: &autoscalingv2beta2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2beta2.MetricTarget{
					Type:               autoscalingv2beta2.UtilizationMetricType,
					AverageUtilization: &cpuTarget,
				},
			},
		}
		metrics := []autoscalingv2beta2.MetricSpec{targetCPUUtilization}

		autoscaler := autoscalingv2beta2.HorizontalPodAutoscaler{
			ObjectMeta: objectMeta,
			Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       "OpenTelemetryCollector",
					Name:       naming.OpenTelemetryCollector(otelcol),
				},
				MinReplicas: otelcol.Spec.Replicas,
				MaxReplicas: *otelcol.Spec.MaxReplicas,
				Metrics:     metrics,
			},
		}
		result = &autoscaler
	} else {
		targetCPUUtilization := autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: &cpuTarget,
				},
			},
		}
		metrics := []autoscalingv2.MetricSpec{targetCPUUtilization}

		autoscaler := autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: objectMeta,
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       "OpenTelemetryCollector",
					Name:       naming.OpenTelemetryCollector(otelcol),
				},
				MinReplicas: otelcol.Spec.Replicas,
				MaxReplicas: *otelcol.Spec.MaxReplicas,
				Metrics:     metrics,
			},
		}
		result = &autoscaler
	}

	return result
}
