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

func HorizontalPodAutoscaler(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) client.Object {
	autoscalingVersion := cfg.AutoscalingVersion()

	name := naming.Collector(otelcol)
	labels := Labels(otelcol, name, cfg.LabelsFilter())
	annotations := Annotations(otelcol)
	var result client.Object

	objectMeta := metav1.ObjectMeta{
		Name:        naming.HorizontalPodAutoscaler(otelcol),
		Namespace:   otelcol.Namespace,
		Labels:      labels,
		Annotations: annotations,
	}

	// defaulting webhook should always set this, but if unset then return nil.
	if otelcol.Spec.Autoscaler == nil {
		logger.Info("Autoscaler field is unset in Spec, skipping")
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

	if autoscalingVersion == autodetect.AutoscalingVersionV2Beta2 {
		metrics := []autoscalingv2beta2.MetricSpec{}

		if otelcol.Spec.Autoscaler.TargetMemoryUtilization != nil {
			utilizationTarget := autoscalingv2beta2.MetricSpec{
				Type: autoscalingv2beta2.ResourceMetricSourceType,
				Resource: &autoscalingv2beta2.ResourceMetricSource{
					Name: corev1.ResourceMemory,
					Target: autoscalingv2beta2.MetricTarget{
						Type:               autoscalingv2beta2.UtilizationMetricType,
						AverageUtilization: otelcol.Spec.Autoscaler.TargetMemoryUtilization,
					},
				},
			}
			metrics = append(metrics, utilizationTarget)
		}

		targetCPUUtilization := autoscalingv2beta2.MetricSpec{
			Type: autoscalingv2beta2.ResourceMetricSourceType,
			Resource: &autoscalingv2beta2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2beta2.MetricTarget{
					Type:               autoscalingv2beta2.UtilizationMetricType,
					AverageUtilization: otelcol.Spec.Autoscaler.TargetCPUUtilization,
				},
			},
		}
		metrics = append(metrics, targetCPUUtilization)

		autoscaler := autoscalingv2beta2.HorizontalPodAutoscaler{
			ObjectMeta: objectMeta,
			Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       "OpenTelemetryCollector",
					Name:       naming.OpenTelemetryCollector(otelcol),
				},
				MinReplicas: otelcol.Spec.Autoscaler.MinReplicas,
				MaxReplicas: *otelcol.Spec.Autoscaler.MaxReplicas,
				Metrics:     metrics,
			},
		}

		if otelcol.Spec.Autoscaler.Behavior != nil {
			behavior := ConvertToV2beta2Behavior(*otelcol.Spec.Autoscaler.Behavior)
			autoscaler.Spec.Behavior = &behavior
		}

		// check for custom metrics
		if len(otelcol.Spec.Autoscaler.Metrics) > 0 {
			metrics := ConvertToV2Beta2PodMetrics(otelcol.Spec.Autoscaler.Metrics)
			autoscaler.Spec.Metrics = append(autoscaler.Spec.Metrics, metrics...)
		}

		result = &autoscaler
	} else {
		metrics := []autoscalingv2.MetricSpec{}

		if otelcol.Spec.Autoscaler.TargetMemoryUtilization != nil {
			utilizationTarget := autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceMemory,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: otelcol.Spec.Autoscaler.TargetMemoryUtilization,
					},
				},
			}
			metrics = append(metrics, utilizationTarget)
		}

		if otelcol.Spec.Autoscaler.TargetCPUUtilization != nil {
			targetCPUUtilization := autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: otelcol.Spec.Autoscaler.TargetCPUUtilization,
					},
				},
			}
			metrics = append(metrics, targetCPUUtilization)
		}

		autoscaler := autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: objectMeta,
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       "OpenTelemetryCollector",
					Name:       naming.OpenTelemetryCollector(otelcol),
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
	}

	return result
}

func ConvertToV2Beta2PodMetrics(v2metrics []v1alpha1.MetricSpec) []autoscalingv2beta2.MetricSpec {
	metrics := make([]autoscalingv2beta2.MetricSpec, len(v2metrics))

	for i, v2metric := range v2metrics {
		metrics[i].Type = autoscalingv2beta2.MetricSourceType(v2metric.Type)
		if v2metric.Pods != nil {
			metrics[i].Pods = &autoscalingv2beta2.PodsMetricSource{
				Metric: autoscalingv2beta2.MetricIdentifier{
					Name:     v2metric.Pods.Metric.Name,
					Selector: v2metric.Pods.Metric.Selector,
				},
				Target: autoscalingv2beta2.MetricTarget{
					Type:         autoscalingv2beta2.MetricTargetType(v2metric.Pods.Target.Type),
					AverageValue: v2metric.Pods.Target.AverageValue,
				},
			}
		}
	}

	return metrics
}

// Create a v2beta2 HorizontalPodAutoscalerBehavior from a v2 instance.
func ConvertToV2beta2Behavior(v2behavior autoscalingv2.HorizontalPodAutoscalerBehavior) autoscalingv2beta2.HorizontalPodAutoscalerBehavior {
	behavior := &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{}

	if v2behavior.ScaleUp != nil {
		scaleUpRules := &autoscalingv2beta2.HPAScalingRules{}
		scaleUpTime := *v2behavior.ScaleUp.StabilizationWindowSeconds
		scaleUpRules.StabilizationWindowSeconds = &scaleUpTime

		if v2behavior.ScaleUp.SelectPolicy != nil {
			scaleUpSelectPolicy := ConvertToV2Beta2SelectPolicy(*v2behavior.ScaleUp.SelectPolicy)
			scaleUpRules.SelectPolicy = &scaleUpSelectPolicy
		}
		if v2behavior.ScaleUp.Policies != nil {
			scaleUpPolicies := []autoscalingv2beta2.HPAScalingPolicy{}
			for _, policy := range v2behavior.ScaleUp.Policies {
				v2beta2policy := ConvertToV2Beta2HPAScalingPolicy(policy)
				scaleUpPolicies = append(scaleUpPolicies, v2beta2policy)
			}
			scaleUpRules.Policies = scaleUpPolicies
		}

		behavior.ScaleUp = scaleUpRules
	}

	if v2behavior.ScaleDown != nil {
		scaleDownRules := &autoscalingv2beta2.HPAScalingRules{}
		scaleDownTime := *v2behavior.ScaleDown.StabilizationWindowSeconds
		scaleDownRules.StabilizationWindowSeconds = &scaleDownTime

		if v2behavior.ScaleDown.SelectPolicy != nil {
			scaleDownSelectPolicy := ConvertToV2Beta2SelectPolicy(*v2behavior.ScaleDown.SelectPolicy)
			scaleDownRules.SelectPolicy = &scaleDownSelectPolicy
		}
		if v2behavior.ScaleDown.Policies != nil {
			ScaleDownPolicies := []autoscalingv2beta2.HPAScalingPolicy{}
			for _, policy := range v2behavior.ScaleDown.Policies {
				v2beta2policy := ConvertToV2Beta2HPAScalingPolicy(policy)
				ScaleDownPolicies = append(ScaleDownPolicies, v2beta2policy)
			}
			scaleDownRules.Policies = ScaleDownPolicies
		}

		behavior.ScaleDown = scaleDownRules
	}

	return *behavior
}

func ConvertToV2Beta2HPAScalingPolicy(v2policy autoscalingv2.HPAScalingPolicy) autoscalingv2beta2.HPAScalingPolicy {
	v2beta2Policy := &autoscalingv2beta2.HPAScalingPolicy{
		Value:         v2policy.Value,
		PeriodSeconds: v2policy.PeriodSeconds,
	}

	switch v2policy.Type {
	case autoscalingv2.PodsScalingPolicy:
		v2beta2Policy.Type = autoscalingv2beta2.PodsScalingPolicy
	case autoscalingv2.PercentScalingPolicy:
		v2beta2Policy.Type = autoscalingv2beta2.PercentScalingPolicy
	}

	return *v2beta2Policy
}

func ConvertToV2Beta2SelectPolicy(scalingPolicy autoscalingv2.ScalingPolicySelect) autoscalingv2beta2.ScalingPolicySelect {
	max := autoscalingv2beta2.MaxPolicySelect
	min := autoscalingv2beta2.MinPolicySelect
	disabled := autoscalingv2beta2.DisabledPolicySelect

	switch scalingPolicy {
	case autoscalingv2.MaxChangePolicySelect:
		return max
	case autoscalingv2.MinChangePolicySelect:
		return min
	case autoscalingv2.DisabledPolicySelect:
		return disabled
	}

	return disabled
}
