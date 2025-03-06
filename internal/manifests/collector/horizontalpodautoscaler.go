// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func HorizontalPodAutoscaler(params manifests.Params) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	name := naming.Collector(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter())
	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter())
	if err != nil {
		return nil, err
	}

	var result *autoscalingv2.HorizontalPodAutoscaler

	objectMeta := metav1.ObjectMeta{
		Name:        naming.HorizontalPodAutoscaler(params.OtelCol.Name),
		Namespace:   params.OtelCol.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	// defaulting webhook should always set this, but if unset then return nil.
	if params.OtelCol.Spec.Autoscaler == nil {
		params.Log.V(4).Info("hpa field is unset in Spec, skipping autoscaler creation")
		return nil, nil
	}

	metrics := []autoscalingv2.MetricSpec{}

	if params.OtelCol.Spec.Autoscaler.TargetMemoryUtilization != nil {
		memoryTarget := autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: params.OtelCol.Spec.Autoscaler.TargetMemoryUtilization,
				},
			},
		}
		metrics = append(metrics, memoryTarget)
	}

	if params.OtelCol.Spec.Autoscaler.TargetCPUUtilization != nil {
		cpuTarget := autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: params.OtelCol.Spec.Autoscaler.TargetCPUUtilization,
				},
			},
		}
		metrics = append(metrics, cpuTarget)
	}

	autoscaler := autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: objectMeta,
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: v1beta1.GroupVersion.String(),
				Kind:       "OpenTelemetryCollector",
				Name:       naming.OpenTelemetryCollector(params.OtelCol.Name),
			},
			MinReplicas: params.OtelCol.Spec.Autoscaler.MinReplicas,
			MaxReplicas: func(max *int32) int32 {
				if max == nil {
					return 0
				}
				return *max
			}(params.OtelCol.Spec.Autoscaler.MaxReplicas),
			Metrics: metrics,
		},
	}
	if params.OtelCol.Spec.Autoscaler.Behavior != nil {
		autoscaler.Spec.Behavior = params.OtelCol.Spec.Autoscaler.Behavior
	}

	// convert from v1alpha1.MetricSpec into a autoscalingv2.MetricSpec.
	for _, metric := range params.OtelCol.Spec.Autoscaler.Metrics {
		if metric.Type == autoscalingv2.PodsMetricSourceType {
			v2metric := autoscalingv2.MetricSpec{
				Type: metric.Type,
				Pods: metric.Pods,
			}
			autoscaler.Spec.Metrics = append(autoscaler.Spec.Metrics, v2metric)
		} // pod metrics
	}
	result = &autoscaler

	return result, nil
}
