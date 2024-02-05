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

package convert

import (
	"errors"

	"gopkg.in/yaml.v3"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
)

func V1Alpha1to2(in v1alpha1.OpenTelemetryCollector) (v1alpha2.OpenTelemetryCollector, error) {
	copy := in.DeepCopy()
	out := v1alpha2.OpenTelemetryCollector{
		TypeMeta:   copy.TypeMeta,
		ObjectMeta: copy.ObjectMeta,
	}

	cfg := &v1alpha2.Config{}
	if err := yaml.Unmarshal([]byte(in.Spec.Config), cfg); err != nil {
		return v1alpha2.OpenTelemetryCollector{}, errors.New("could not convert config json to v1alpha2.Config")
	}
	out.Spec.Config = *cfg

	out.Spec.OpenTelemetryCommonFields.ManagementState = v1alpha2.ManagementStateType(copy.Spec.ManagementState)
	out.Spec.OpenTelemetryCommonFields.Resources = copy.Spec.Resources
	out.Spec.OpenTelemetryCommonFields.NodeSelector = copy.Spec.NodeSelector
	out.Spec.OpenTelemetryCommonFields.Args = copy.Spec.NodeSelector
	out.Spec.OpenTelemetryCommonFields.Replicas = copy.Spec.Replicas

	if copy.Spec.Autoscaler != nil {
		metrics := make([]v1alpha2.MetricSpec, len(copy.Spec.Autoscaler.Metrics))
		for i, m := range copy.Spec.Autoscaler.Metrics {
			metrics[i] = v1alpha2.MetricSpec{
				Type: m.Type,
				Pods: m.Pods,
			}
		}
		out.Spec.OpenTelemetryCommonFields.Autoscaler = &v1alpha2.AutoscalerSpec{
			MinReplicas:             copy.Spec.Autoscaler.MinReplicas,
			MaxReplicas:             copy.Spec.Autoscaler.MaxReplicas,
			Behavior:                copy.Spec.Autoscaler.Behavior,
			Metrics:                 metrics,
			TargetCPUUtilization:    copy.Spec.Autoscaler.TargetCPUUtilization,
			TargetMemoryUtilization: copy.Spec.Autoscaler.TargetMemoryUtilization,
		}
	}

	if copy.Spec.PodDisruptionBudget != nil {
		out.Spec.OpenTelemetryCommonFields.PodDisruptionBudget = &v1alpha2.PodDisruptionBudgetSpec{
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

	out.Spec.TargetAllocator.Replicas = copy.Spec.TargetAllocator.Replicas
	out.Spec.TargetAllocator.NodeSelector = copy.Spec.TargetAllocator.NodeSelector
	out.Spec.TargetAllocator.Resources = copy.Spec.TargetAllocator.Resources
	out.Spec.TargetAllocator.AllocationStrategy = copy.Spec.TargetAllocator.AllocationStrategy
	out.Spec.TargetAllocator.FilterStrategy = copy.Spec.TargetAllocator.FilterStrategy
	out.Spec.TargetAllocator.ServiceAccount = copy.Spec.TargetAllocator.ServiceAccount
	out.Spec.TargetAllocator.Image = copy.Spec.TargetAllocator.Image
	out.Spec.TargetAllocator.Enabled = copy.Spec.TargetAllocator.Enabled
	out.Spec.TargetAllocator.Affinity = copy.Spec.TargetAllocator.Affinity
	out.Spec.TargetAllocator.PrometheusCR.Enabled = copy.Spec.TargetAllocator.PrometheusCR.Enabled
	out.Spec.TargetAllocator.PrometheusCR.ScrapeInterval = copy.Spec.TargetAllocator.PrometheusCR.ScrapeInterval
	out.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector = copy.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector
	out.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector = copy.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector
	out.Spec.TargetAllocator.SecurityContext = copy.Spec.TargetAllocator.SecurityContext
	out.Spec.TargetAllocator.PodSecurityContext = copy.Spec.TargetAllocator.PodSecurityContext
	out.Spec.TargetAllocator.TopologySpreadConstraints = copy.Spec.TargetAllocator.TopologySpreadConstraints
	out.Spec.TargetAllocator.Tolerations = copy.Spec.TargetAllocator.Tolerations
	out.Spec.TargetAllocator.Env = copy.Spec.TargetAllocator.Env
	out.Spec.TargetAllocator.Observability = v1alpha1.ObservabilitySpec{
		Metrics: v1alpha1.MetricsConfigSpec{
			EnableMetrics: copy.Spec.TargetAllocator.Observability.Metrics.EnableMetrics,
		},
	}
	out.Spec.TargetAllocator.PodDisruptionBudget = copy.Spec.TargetAllocator.PodDisruptionBudget

	out.Spec.Mode = v1alpha2.Mode(copy.Spec.Mode)
	out.Spec.UpgradeStrategy = v1alpha2.UpgradeStrategy(copy.Spec.UpgradeStrategy)
	out.Spec.Ingress.Type = v1alpha2.IngressType(copy.Spec.Ingress.Type)
	out.Spec.Ingress.RuleType = v1alpha2.IngressRuleType(copy.Spec.Ingress.RuleType)
	out.Spec.Ingress.Hostname = copy.Spec.Ingress.Hostname
	out.Spec.Ingress.Annotations = copy.Spec.Ingress.Annotations
	out.Spec.Ingress.TLS = copy.Spec.Ingress.TLS
	out.Spec.Ingress.IngressClassName = copy.Spec.Ingress.IngressClassName
	out.Spec.Ingress.Route.Termination = v1alpha2.TLSRouteTerminationType(copy.Spec.Ingress.Route.Termination)

	if copy.Spec.LivenessProbe != nil {
		out.Spec.LivenessProbe = &v1alpha2.Probe{
			InitialDelaySeconds:           copy.Spec.LivenessProbe.InitialDelaySeconds,
			TimeoutSeconds:                copy.Spec.LivenessProbe.TimeoutSeconds,
			PeriodSeconds:                 copy.Spec.LivenessProbe.PeriodSeconds,
			SuccessThreshold:              copy.Spec.LivenessProbe.SuccessThreshold,
			FailureThreshold:              copy.Spec.LivenessProbe.FailureThreshold,
			TerminationGracePeriodSeconds: copy.Spec.LivenessProbe.TerminationGracePeriodSeconds,
		}
	}

	out.Spec.Observability.Metrics.EnableMetrics = copy.Spec.Observability.Metrics.EnableMetrics

	out.Spec.ConfigMaps = copy.Spec.ConfigMaps
	out.Spec.DaemonSetUpdateStrategy = copy.Spec.UpdateStrategy
	out.Spec.DeploymentUpdateStrategy.Type = copy.Spec.DeploymentUpdateStrategy.Type
	out.Spec.DeploymentUpdateStrategy.RollingUpdate = copy.Spec.DeploymentUpdateStrategy.RollingUpdate

	return out, nil
}
