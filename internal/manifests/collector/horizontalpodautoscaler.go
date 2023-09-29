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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func HorizontalPodAutoscaler(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector, otelColConfig manifests.OtelConfig) client.Object {
	name := naming.Collector(otelcol.Name)
	labels := Labels(otelcol, name, cfg.LabelsFilter())
	annotations := Annotations(otelcol)
	var result client.Object

	objectMeta := metav1.ObjectMeta{
		Name:        naming.HorizontalPodAutoscaler(otelcol.Name),
		Namespace:   otelcol.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	// defaulting webhook should always set this, but if unset then return nil.
	if otelcol.Spec.Autoscaler == nil {
		logger.Info("hpa field is unset in Spec, skipping autoscaler creation")
		return nil
	}

	if otelcol.Spec.Autoscaler.MaxReplicas == nil {
		otelcol.Spec.Autoscaler.MaxReplicas = otelcol.Spec.MaxReplicas
	}

	if otelcol.Spec.Autoscaler.MinReplicas == nil {
		if otelcol.Spec.MinReplicas != nil {
			otelcol.Spec.Autoscaler.MinReplicas = otelcol.Spec.MinReplicas
		} else {
			otelcol.Spec.Autoscaler.MinReplicas = otelcol.Spec.Replicas
		}
	}

	metrics := []autoscalingv2.MetricSpec{}

	if otelcol.Spec.Autoscaler.TargetMemoryUtilization != nil {
		memoryTarget := autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: otelcol.Spec.Autoscaler.TargetMemoryUtilization,
				},
			},
		}
		metrics = append(metrics, memoryTarget)
	}

	if otelcol.Spec.Autoscaler.TargetCPUUtilization != nil {
		cpuTarget := autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: otelcol.Spec.Autoscaler.TargetCPUUtilization,
				},
			},
		}
		metrics = append(metrics, cpuTarget)
	}

	autoscaler := autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: objectMeta,
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "OpenTelemetryCollector",
				Name:       naming.OpenTelemetryCollector(otelcol.Name),
			},
			MinReplicas: otelcol.Spec.Autoscaler.MinReplicas,
			MaxReplicas: *otelcol.Spec.Autoscaler.MaxReplicas,
			Metrics:     metrics,
		},
	}
	if otelcol.Spec.Autoscaler.Behavior != nil {
		autoscaler.Spec.Behavior = otelcol.Spec.Autoscaler.Behavior
	}

	// convert from v1alpha1.MetricSpec into a autoscalingv2.MetricSpec.
	for _, metric := range otelcol.Spec.Autoscaler.Metrics {
		if metric.Type == autoscalingv2.PodsMetricSourceType {
			v2metric := autoscalingv2.MetricSpec{
				Type: metric.Type,
				Pods: metric.Pods,
			}
			autoscaler.Spec.Metrics = append(autoscaler.Spec.Metrics, v2metric)
		} // pod metrics
	}
	result = &autoscaler

	return result
}
