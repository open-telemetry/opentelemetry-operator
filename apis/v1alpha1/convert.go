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

package v1alpha1

import (
	"errors"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func Tov1beta1(in OpenTelemetryCollector) (v1beta1.OpenTelemetryCollector, error) {
	copy := in.DeepCopy()
	out := v1beta1.OpenTelemetryCollector{
		ObjectMeta: copy.ObjectMeta,

		Status: v1beta1.OpenTelemetryCollectorStatus{
			Scale: v1beta1.ScaleSubresourceStatus{
				Selector:       in.Status.Scale.Selector,
				Replicas:       in.Status.Scale.Replicas,
				StatusReplicas: in.Status.Scale.StatusReplicas,
			},
			Version: in.Status.Version,
			Image:   in.Spec.Image,
		},
	}

	cfg := &v1beta1.Config{}
	if err := yaml.Unmarshal([]byte(in.Spec.Config), cfg); err != nil {
		return v1beta1.OpenTelemetryCollector{}, errors.New("could not convert config json to v1beta1.Config")
	}
	out.Spec.Config = *cfg

	out.Spec.OpenTelemetryCommonFields.ManagementState = v1beta1.ManagementStateType(copy.Spec.ManagementState)
	out.Spec.OpenTelemetryCommonFields.Resources = copy.Spec.Resources
	out.Spec.OpenTelemetryCommonFields.NodeSelector = copy.Spec.NodeSelector
	out.Spec.OpenTelemetryCommonFields.Args = copy.Spec.Args
	out.Spec.OpenTelemetryCommonFields.Replicas = copy.Spec.Replicas

	if copy.Spec.Autoscaler != nil {
		metrics := make([]v1beta1.MetricSpec, len(copy.Spec.Autoscaler.Metrics))
		for i, m := range copy.Spec.Autoscaler.Metrics {
			metrics[i] = v1beta1.MetricSpec{
				Type: m.Type,
				Pods: m.Pods,
			}
		}
		if copy.Spec.MaxReplicas != nil && copy.Spec.Autoscaler.MaxReplicas == nil {
			copy.Spec.Autoscaler.MaxReplicas = copy.Spec.MaxReplicas
		}
		if copy.Spec.MinReplicas != nil && copy.Spec.Autoscaler.MinReplicas == nil {
			copy.Spec.Autoscaler.MinReplicas = copy.Spec.MinReplicas
		}
		out.Spec.OpenTelemetryCommonFields.Autoscaler = &v1beta1.AutoscalerSpec{
			MinReplicas:             copy.Spec.Autoscaler.MinReplicas,
			MaxReplicas:             copy.Spec.Autoscaler.MaxReplicas,
			Behavior:                copy.Spec.Autoscaler.Behavior,
			Metrics:                 metrics,
			TargetCPUUtilization:    copy.Spec.Autoscaler.TargetCPUUtilization,
			TargetMemoryUtilization: copy.Spec.Autoscaler.TargetMemoryUtilization,
		}
	}

	if copy.Spec.PodDisruptionBudget != nil {
		out.Spec.OpenTelemetryCommonFields.PodDisruptionBudget = &v1beta1.PodDisruptionBudgetSpec{
			MinAvailable:   copy.Spec.PodDisruptionBudget.MinAvailable,
			MaxUnavailable: copy.Spec.PodDisruptionBudget.MaxUnavailable,
		}
	}
	if copy.Spec.SecurityContext != nil {
		out.Spec.OpenTelemetryCommonFields.SecurityContext = copy.Spec.SecurityContext
	}
	if copy.Spec.PodSecurityContext != nil {
		out.Spec.OpenTelemetryCommonFields.PodSecurityContext = copy.Spec.PodSecurityContext
	}
	out.Spec.OpenTelemetryCommonFields.PodAnnotations = copy.Spec.PodAnnotations
	out.Spec.OpenTelemetryCommonFields.ServiceAccount = copy.Spec.ServiceAccount
	out.Spec.OpenTelemetryCommonFields.Image = copy.Spec.Image
	out.Spec.OpenTelemetryCommonFields.ImagePullPolicy = copy.Spec.ImagePullPolicy
	out.Spec.OpenTelemetryCommonFields.VolumeMounts = copy.Spec.VolumeMounts
	out.Spec.OpenTelemetryCommonFields.Ports = copy.Spec.Ports
	out.Spec.OpenTelemetryCommonFields.Env = copy.Spec.Env
	out.Spec.OpenTelemetryCommonFields.EnvFrom = copy.Spec.EnvFrom
	out.Spec.OpenTelemetryCommonFields.VolumeClaimTemplates = copy.Spec.VolumeClaimTemplates
	out.Spec.OpenTelemetryCommonFields.Tolerations = copy.Spec.Tolerations
	out.Spec.OpenTelemetryCommonFields.Volumes = copy.Spec.Volumes
	out.Spec.OpenTelemetryCommonFields.Affinity = copy.Spec.Affinity
	out.Spec.OpenTelemetryCommonFields.Lifecycle = copy.Spec.Lifecycle
	out.Spec.OpenTelemetryCommonFields.TerminationGracePeriodSeconds = copy.Spec.TerminationGracePeriodSeconds
	out.Spec.OpenTelemetryCommonFields.TopologySpreadConstraints = copy.Spec.TopologySpreadConstraints
	out.Spec.OpenTelemetryCommonFields.HostNetwork = copy.Spec.HostNetwork
	out.Spec.OpenTelemetryCommonFields.ShareProcessNamespace = copy.Spec.ShareProcessNamespace
	out.Spec.OpenTelemetryCommonFields.PriorityClassName = copy.Spec.PriorityClassName
	out.Spec.OpenTelemetryCommonFields.InitContainers = copy.Spec.InitContainers
	out.Spec.OpenTelemetryCommonFields.AdditionalContainers = copy.Spec.AdditionalContainers

	out.Spec.TargetAllocator = TargetAllocatorEmbedded(copy.Spec.TargetAllocator)

	out.Spec.Mode = v1beta1.Mode(copy.Spec.Mode)
	out.Spec.UpgradeStrategy = v1beta1.UpgradeStrategy(copy.Spec.UpgradeStrategy)
	out.Spec.Ingress.Type = v1beta1.IngressType(copy.Spec.Ingress.Type)
	out.Spec.Ingress.RuleType = v1beta1.IngressRuleType(copy.Spec.Ingress.RuleType)
	out.Spec.Ingress.Hostname = copy.Spec.Ingress.Hostname
	out.Spec.Ingress.Annotations = copy.Spec.Ingress.Annotations
	out.Spec.Ingress.TLS = copy.Spec.Ingress.TLS
	out.Spec.Ingress.IngressClassName = copy.Spec.Ingress.IngressClassName
	out.Spec.Ingress.Route.Termination = v1beta1.TLSRouteTerminationType(copy.Spec.Ingress.Route.Termination)

	if copy.Spec.LivenessProbe != nil {
		out.Spec.LivenessProbe = &v1beta1.Probe{
			InitialDelaySeconds:           copy.Spec.LivenessProbe.InitialDelaySeconds,
			TimeoutSeconds:                copy.Spec.LivenessProbe.TimeoutSeconds,
			PeriodSeconds:                 copy.Spec.LivenessProbe.PeriodSeconds,
			SuccessThreshold:              copy.Spec.LivenessProbe.SuccessThreshold,
			FailureThreshold:              copy.Spec.LivenessProbe.FailureThreshold,
			TerminationGracePeriodSeconds: copy.Spec.LivenessProbe.TerminationGracePeriodSeconds,
		}
	}

	out.Spec.Observability.Metrics.EnableMetrics = copy.Spec.Observability.Metrics.EnableMetrics

	for _, cm := range copy.Spec.ConfigMaps {
		out.Spec.ConfigMaps = append(out.Spec.ConfigMaps, v1beta1.ConfigMapsSpec{
			Name:      cm.Name,
			MountPath: cm.MountPath,
		})
	}
	out.Spec.DaemonSetUpdateStrategy = copy.Spec.UpdateStrategy
	out.Spec.DeploymentUpdateStrategy.Type = copy.Spec.DeploymentUpdateStrategy.Type
	out.Spec.DeploymentUpdateStrategy.RollingUpdate = copy.Spec.DeploymentUpdateStrategy.RollingUpdate

	return out, nil
}

func TargetAllocatorEmbedded(in OpenTelemetryTargetAllocator) v1beta1.TargetAllocatorEmbedded {
	out := v1beta1.TargetAllocatorEmbedded{}
	out.Replicas = in.Replicas
	out.NodeSelector = in.NodeSelector
	out.Resources = in.Resources
	out.AllocationStrategy = v1beta1.TargetAllocatorAllocationStrategy(in.AllocationStrategy)
	out.FilterStrategy = v1beta1.TargetAllocatorFilterStrategy(in.FilterStrategy)
	out.ServiceAccount = in.ServiceAccount
	out.Image = in.Image
	out.Enabled = in.Enabled
	out.Affinity = in.Affinity
	out.PrometheusCR.Enabled = in.PrometheusCR.Enabled
	out.PrometheusCR.ScrapeInterval = in.PrometheusCR.ScrapeInterval
	out.SecurityContext = in.SecurityContext
	out.PodSecurityContext = in.PodSecurityContext
	out.TopologySpreadConstraints = in.TopologySpreadConstraints
	out.Tolerations = in.Tolerations
	out.Env = in.Env
	out.Observability = v1beta1.ObservabilitySpec{
		Metrics: v1beta1.MetricsConfigSpec{
			EnableMetrics: in.Observability.Metrics.EnableMetrics,
		},
	}

	out.PrometheusCR.PodMonitorSelector = &metav1.LabelSelector{
		MatchLabels: in.PrometheusCR.PodMonitorSelector,
	}
	out.PrometheusCR.ServiceMonitorSelector = &metav1.LabelSelector{
		MatchLabels: in.PrometheusCR.ServiceMonitorSelector,
	}
	if in.PodDisruptionBudget != nil {
		out.PodDisruptionBudget = &v1beta1.PodDisruptionBudgetSpec{
			MinAvailable:   in.PodDisruptionBudget.MinAvailable,
			MaxUnavailable: in.PodDisruptionBudget.MaxUnavailable,
		}
	}
	return out
}
