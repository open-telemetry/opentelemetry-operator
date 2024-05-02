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
	"fmt"

	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

var _ conversion.Convertible = &OpenTelemetryCollector{}

func (src *OpenTelemetryCollector) ConvertTo(dstRaw conversion.Hub) error {
	switch t := dstRaw.(type) {
	case *v1beta1.OpenTelemetryCollector:
		dst := dstRaw.(*v1beta1.OpenTelemetryCollector)
		convertedSrc, err := tov1beta1(*src)
		if err != nil {
			return fmt.Errorf("failed to convert to v1beta1: %w", err)
		}
		dst.ObjectMeta = convertedSrc.ObjectMeta
		dst.Spec = convertedSrc.Spec
		dst.Status = convertedSrc.Status
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
	return nil
}

func (dst *OpenTelemetryCollector) ConvertFrom(srcRaw conversion.Hub) error {
	switch t := srcRaw.(type) {
	case *v1beta1.OpenTelemetryCollector:
		src := srcRaw.(*v1beta1.OpenTelemetryCollector)
		srcConverted, err := tov1alpha1(*src)
		if err != nil {
			return fmt.Errorf("failed to convert to v1alpha1: %w", err)
		}
		dst.ObjectMeta = srcConverted.ObjectMeta
		dst.Spec = srcConverted.Spec
		dst.Status = srcConverted.Status
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
	return nil
}

func tov1beta1(in OpenTelemetryCollector) (v1beta1.OpenTelemetryCollector, error) {
	cfgCopy := in.DeepCopy()
	cfg := &v1beta1.Config{}
	if err := yaml.Unmarshal([]byte(cfgCopy.Spec.Config), cfg); err != nil {
		return v1beta1.OpenTelemetryCollector{}, errors.New("could not convert config json to v1beta1.Config")
	}
	fmt.Println("-------------- convert to v1beta1")
	fmt.Println(in.Spec.Config)
	fmt.Println("-------------- convert to v1beta1")
	return v1beta1.OpenTelemetryCollector{
		ObjectMeta: cfgCopy.ObjectMeta,
		Status: v1beta1.OpenTelemetryCollectorStatus{
			Scale: v1beta1.ScaleSubresourceStatus{
				Selector:       in.Status.Scale.Selector,
				Replicas:       in.Status.Scale.Replicas,
				StatusReplicas: in.Status.Scale.StatusReplicas,
			},
			Version: in.Status.Version,
			Image:   in.Status.Image,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				ManagementState:               v1beta1.ManagementStateType(cfgCopy.Spec.ManagementState),
				Resources:                     cfgCopy.Spec.Resources,
				NodeSelector:                  cfgCopy.Spec.NodeSelector,
				Args:                          cfgCopy.Spec.Args,
				Replicas:                      cfgCopy.Spec.Replicas,
				Autoscaler:                    tov1beta1Autoscaler(cfgCopy.Spec.Autoscaler, cfgCopy.Spec.MinReplicas, cfgCopy.Spec.MaxReplicas),
				PodDisruptionBudget:           tov1beta1PodDisruptionBudget(cfgCopy.Spec.PodDisruptionBudget),
				SecurityContext:               cfgCopy.Spec.SecurityContext,
				PodSecurityContext:            cfgCopy.Spec.PodSecurityContext,
				PodAnnotations:                cfgCopy.Spec.PodAnnotations,
				ServiceAccount:                cfgCopy.Spec.ServiceAccount,
				Image:                         cfgCopy.Spec.Image,
				ImagePullPolicy:               cfgCopy.Spec.ImagePullPolicy,
				VolumeMounts:                  cfgCopy.Spec.VolumeMounts,
				Ports:                         tov1beta1Ports(cfgCopy.Spec.Ports),
				Env:                           cfgCopy.Spec.Env,
				EnvFrom:                       cfgCopy.Spec.EnvFrom,
				VolumeClaimTemplates:          cfgCopy.Spec.VolumeClaimTemplates,
				Tolerations:                   cfgCopy.Spec.Tolerations,
				Volumes:                       cfgCopy.Spec.Volumes,
				Affinity:                      cfgCopy.Spec.Affinity,
				Lifecycle:                     cfgCopy.Spec.Lifecycle,
				TerminationGracePeriodSeconds: cfgCopy.Spec.TerminationGracePeriodSeconds,
				TopologySpreadConstraints:     cfgCopy.Spec.TopologySpreadConstraints,
				HostNetwork:                   cfgCopy.Spec.HostNetwork,
				ShareProcessNamespace:         cfgCopy.Spec.ShareProcessNamespace,
				PriorityClassName:             cfgCopy.Spec.PriorityClassName,
				InitContainers:                cfgCopy.Spec.InitContainers,
				AdditionalContainers:          cfgCopy.Spec.AdditionalContainers,
			},
			TargetAllocator: tov1beta1TA(cfgCopy.Spec.TargetAllocator),
			Mode:            v1beta1.Mode(cfgCopy.Spec.Mode),
			UpgradeStrategy: v1beta1.UpgradeStrategy(cfgCopy.Spec.UpgradeStrategy),
			Config:          *cfg,
			Ingress: v1beta1.Ingress{
				Type:             v1beta1.IngressType(cfgCopy.Spec.Ingress.Type),
				RuleType:         v1beta1.IngressRuleType(cfgCopy.Spec.Ingress.RuleType),
				Hostname:         cfgCopy.Spec.Ingress.Hostname,
				Annotations:      cfgCopy.Spec.Ingress.Annotations,
				TLS:              cfgCopy.Spec.Ingress.TLS,
				IngressClassName: cfgCopy.Spec.Ingress.IngressClassName,
				Route: v1beta1.OpenShiftRoute{
					Termination: v1beta1.TLSRouteTerminationType(cfgCopy.Spec.Ingress.Route.Termination),
				},
			},
			LivenessProbe: tov1beta1Probe(cfgCopy.Spec.LivenessProbe),
			Observability: v1beta1.ObservabilitySpec{
				Metrics: v1beta1.MetricsConfigSpec{
					EnableMetrics:                cfgCopy.Spec.Observability.Metrics.EnableMetrics,
					DisablePrometheusAnnotations: cfgCopy.Spec.Observability.Metrics.DisablePrometheusAnnotations,
				},
			},
			ConfigMaps:              tov1beta1ConfigMaps(cfgCopy.Spec.ConfigMaps),
			DaemonSetUpdateStrategy: cfgCopy.Spec.UpdateStrategy,
			DeploymentUpdateStrategy: appsv1.DeploymentStrategy{
				Type:          cfgCopy.Spec.DeploymentUpdateStrategy.Type,
				RollingUpdate: cfgCopy.Spec.DeploymentUpdateStrategy.RollingUpdate,
			},
		},
	}, nil
}

func tov1beta1Ports(in []PortsSpec) []v1beta1.PortsSpec {
	var ports []v1beta1.PortsSpec

	for _, p := range in {
		ports = append(ports, v1beta1.PortsSpec{
			ServicePort: v1.ServicePort{
				Name:        p.ServicePort.Name,
				Protocol:    p.ServicePort.Protocol,
				AppProtocol: p.ServicePort.AppProtocol,
				Port:        p.ServicePort.Port,
				TargetPort:  p.ServicePort.TargetPort,
				NodePort:    p.ServicePort.NodePort,
			},
			HostPort: p.HostPort,
		})
	}

	return ports
}

func tov1beta1TA(in OpenTelemetryTargetAllocator) v1beta1.TargetAllocatorEmbedded {
	return v1beta1.TargetAllocatorEmbedded{
		Replicas:           in.Replicas,
		NodeSelector:       in.NodeSelector,
		Resources:          in.Resources,
		AllocationStrategy: tov1beta1TAAllocationStrategy(in.AllocationStrategy),
		FilterStrategy:     tov1beta1TAFilterStrategy(in.FilterStrategy),
		ServiceAccount:     in.ServiceAccount,
		Image:              in.Image,
		Enabled:            in.Enabled,
		Affinity:           in.Affinity,
		PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
			Enabled:        in.PrometheusCR.Enabled,
			ScrapeInterval: in.PrometheusCR.ScrapeInterval,
			// prometheus_cr.pod_monitor_selector shouldn't be nil when selector is empty
			PodMonitorSelector: &metav1.LabelSelector{
				MatchLabels: in.PrometheusCR.PodMonitorSelector,
			},
			ServiceMonitorSelector: &metav1.LabelSelector{
				MatchLabels: in.PrometheusCR.ServiceMonitorSelector,
			},
		},
		SecurityContext:           in.SecurityContext,
		PodSecurityContext:        in.PodSecurityContext,
		TopologySpreadConstraints: in.TopologySpreadConstraints,
		Tolerations:               in.Tolerations,
		Env:                       in.Env,
		Observability: v1beta1.ObservabilitySpec{
			Metrics: v1beta1.MetricsConfigSpec{
				EnableMetrics:                in.Observability.Metrics.EnableMetrics,
				DisablePrometheusAnnotations: in.Observability.Metrics.DisablePrometheusAnnotations,
			},
		},
		PodDisruptionBudget: tov1beta1PodDisruptionBudget(in.PodDisruptionBudget),
	}
}

// The conversion takes into account deprecated v1alpha1 spec.minReplicas and spec.maxReplicas.
func tov1beta1Autoscaler(in *AutoscalerSpec, minReplicas, maxReplicas *int32) *v1beta1.AutoscalerSpec {
	if in == nil && minReplicas == nil && maxReplicas == nil {
		return nil
	}
	if in == nil {
		in = &AutoscalerSpec{}
	}

	var metrics []v1beta1.MetricSpec
	for _, m := range in.Metrics {
		metrics = append(metrics, v1beta1.MetricSpec{
			Type: m.Type,
			Pods: m.Pods,
		})
	}
	if maxReplicas != nil && in.MaxReplicas == nil {
		in.MaxReplicas = maxReplicas
	}
	if minReplicas != nil && in.MinReplicas == nil {
		in.MinReplicas = minReplicas
	}

	return &v1beta1.AutoscalerSpec{
		MinReplicas:             in.MinReplicas,
		MaxReplicas:             in.MaxReplicas,
		Behavior:                in.Behavior,
		Metrics:                 metrics,
		TargetCPUUtilization:    in.TargetCPUUtilization,
		TargetMemoryUtilization: in.TargetMemoryUtilization,
	}
}

func tov1beta1PodDisruptionBudget(in *PodDisruptionBudgetSpec) *v1beta1.PodDisruptionBudgetSpec {
	if in == nil {
		return nil
	}
	return &v1beta1.PodDisruptionBudgetSpec{
		MinAvailable:   in.MinAvailable,
		MaxUnavailable: in.MaxUnavailable,
	}
}

func tov1beta1Probe(in *Probe) *v1beta1.Probe {
	if in == nil {
		return nil
	}
	return &v1beta1.Probe{
		InitialDelaySeconds:           in.InitialDelaySeconds,
		TimeoutSeconds:                in.TimeoutSeconds,
		PeriodSeconds:                 in.PeriodSeconds,
		SuccessThreshold:              in.SuccessThreshold,
		FailureThreshold:              in.FailureThreshold,
		TerminationGracePeriodSeconds: in.TerminationGracePeriodSeconds,
	}
}

func tov1beta1ConfigMaps(in []ConfigMapsSpec) []v1beta1.ConfigMapsSpec {
	var mapsSpecs []v1beta1.ConfigMapsSpec
	for _, m := range in {
		mapsSpecs = append(mapsSpecs, v1beta1.ConfigMapsSpec{
			Name:      m.Name,
			MountPath: m.MountPath,
		})
	}
	return mapsSpecs
}

func tov1alpha1Ports(in []v1beta1.PortsSpec) []PortsSpec {
	var ports []PortsSpec

	for _, p := range in {
		ports = append(ports, PortsSpec{
			ServicePort: v1.ServicePort{
				Name:        p.ServicePort.Name,
				Protocol:    p.ServicePort.Protocol,
				AppProtocol: p.ServicePort.AppProtocol,
				Port:        p.ServicePort.Port,
				TargetPort:  p.ServicePort.TargetPort,
				NodePort:    p.ServicePort.NodePort,
			},
			HostPort: p.HostPort,
		})
	}

	return ports
}

func tov1alpha1(in v1beta1.OpenTelemetryCollector) (*OpenTelemetryCollector, error) {
	betaCopy := in.DeepCopy()
	configYaml, err := betaCopy.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}

	return &OpenTelemetryCollector{
		ObjectMeta: betaCopy.ObjectMeta,
		Status: OpenTelemetryCollectorStatus{
			Scale: ScaleSubresourceStatus{
				Selector:       in.Status.Scale.Selector,
				Replicas:       in.Status.Scale.Replicas,
				StatusReplicas: in.Status.Scale.StatusReplicas,
			},
			Version: in.Status.Version,
			Image:   in.Status.Image,
		},

		Spec: OpenTelemetryCollectorSpec{
			ManagementState:      ManagementStateType(betaCopy.Spec.ManagementState),
			Resources:            betaCopy.Spec.Resources,
			NodeSelector:         betaCopy.Spec.NodeSelector,
			Args:                 betaCopy.Spec.Args,
			Replicas:             betaCopy.Spec.Replicas,
			Autoscaler:           tov1alpha1Autoscaler(betaCopy.Spec.Autoscaler),
			PodDisruptionBudget:  tov1alpha1PodDisruptionBudget(betaCopy.Spec.PodDisruptionBudget),
			SecurityContext:      betaCopy.Spec.SecurityContext,
			PodSecurityContext:   betaCopy.Spec.PodSecurityContext,
			PodAnnotations:       betaCopy.Spec.PodAnnotations,
			TargetAllocator:      tov1alpha1TA(betaCopy.Spec.TargetAllocator),
			Mode:                 Mode(betaCopy.Spec.Mode),
			ServiceAccount:       betaCopy.Spec.ServiceAccount,
			Image:                betaCopy.Spec.Image,
			UpgradeStrategy:      UpgradeStrategy(betaCopy.Spec.UpgradeStrategy),
			ImagePullPolicy:      betaCopy.Spec.ImagePullPolicy,
			Config:               configYaml,
			VolumeMounts:         betaCopy.Spec.VolumeMounts,
			Ports:                tov1alpha1Ports(betaCopy.Spec.Ports),
			Env:                  betaCopy.Spec.Env,
			EnvFrom:              betaCopy.Spec.EnvFrom,
			VolumeClaimTemplates: betaCopy.Spec.VolumeClaimTemplates,
			Tolerations:          betaCopy.Spec.Tolerations,
			Volumes:              betaCopy.Spec.Volumes,
			Ingress: Ingress{
				Type:             IngressType(betaCopy.Spec.Ingress.Type),
				RuleType:         IngressRuleType(betaCopy.Spec.Ingress.RuleType),
				Hostname:         betaCopy.Spec.Ingress.Hostname,
				Annotations:      betaCopy.Spec.Ingress.Annotations,
				TLS:              betaCopy.Spec.Ingress.TLS,
				IngressClassName: betaCopy.Spec.Ingress.IngressClassName,
				Route: OpenShiftRoute{
					Termination: TLSRouteTerminationType(betaCopy.Spec.Ingress.Route.Termination),
				},
			},
			HostNetwork:                   betaCopy.Spec.HostNetwork,
			ShareProcessNamespace:         betaCopy.Spec.ShareProcessNamespace,
			PriorityClassName:             betaCopy.Spec.PriorityClassName,
			Affinity:                      betaCopy.Spec.Affinity,
			Lifecycle:                     betaCopy.Spec.Lifecycle,
			TerminationGracePeriodSeconds: betaCopy.Spec.TerminationGracePeriodSeconds,
			LivenessProbe:                 tov1alpha1Probe(betaCopy.Spec.LivenessProbe),
			InitContainers:                betaCopy.Spec.InitContainers,
			AdditionalContainers:          betaCopy.Spec.AdditionalContainers,
			Observability: ObservabilitySpec{
				Metrics: MetricsConfigSpec{
					EnableMetrics:                betaCopy.Spec.Observability.Metrics.EnableMetrics,
					DisablePrometheusAnnotations: betaCopy.Spec.Observability.Metrics.DisablePrometheusAnnotations,
				},
			},
			TopologySpreadConstraints: betaCopy.Spec.TopologySpreadConstraints,
			ConfigMaps:                tov1alpha1ConfigMaps(betaCopy.Spec.ConfigMaps),
			UpdateStrategy:            betaCopy.Spec.DaemonSetUpdateStrategy,
			DeploymentUpdateStrategy:  betaCopy.Spec.DeploymentUpdateStrategy,
		},
	}, nil
}

func tov1alpha1PodDisruptionBudget(in *v1beta1.PodDisruptionBudgetSpec) *PodDisruptionBudgetSpec {
	if in == nil {
		return nil
	}
	return &PodDisruptionBudgetSpec{
		MinAvailable:   in.MinAvailable,
		MaxUnavailable: in.MaxUnavailable,
	}
}

func tov1alpha1Probe(in *v1beta1.Probe) *Probe {
	if in == nil {
		return nil
	}
	return &Probe{
		InitialDelaySeconds:           in.InitialDelaySeconds,
		TimeoutSeconds:                in.TimeoutSeconds,
		PeriodSeconds:                 in.PeriodSeconds,
		SuccessThreshold:              in.SuccessThreshold,
		FailureThreshold:              in.FailureThreshold,
		TerminationGracePeriodSeconds: in.TerminationGracePeriodSeconds,
	}
}

func tov1alpha1Autoscaler(in *v1beta1.AutoscalerSpec) *AutoscalerSpec {
	if in == nil {
		return nil
	}

	var metrics []MetricSpec
	for _, m := range in.Metrics {
		metrics = append(metrics, MetricSpec{
			Type: m.Type,
			Pods: m.Pods,
		})
	}

	return &AutoscalerSpec{
		MinReplicas:             in.MinReplicas,
		MaxReplicas:             in.MaxReplicas,
		Behavior:                in.Behavior,
		Metrics:                 metrics,
		TargetCPUUtilization:    in.TargetCPUUtilization,
		TargetMemoryUtilization: in.TargetMemoryUtilization,
	}
}

func tov1alpha1ConfigMaps(in []v1beta1.ConfigMapsSpec) []ConfigMapsSpec {
	var mapsSpecs []ConfigMapsSpec
	for _, m := range in {
		mapsSpecs = append(mapsSpecs, ConfigMapsSpec{
			Name:      m.Name,
			MountPath: m.MountPath,
		})
	}
	return mapsSpecs
}

func tov1alpha1TA(in v1beta1.TargetAllocatorEmbedded) OpenTelemetryTargetAllocator {
	var podMonitorSelector map[string]string
	if in.PrometheusCR.PodMonitorSelector != nil {
		podMonitorSelector = in.PrometheusCR.PodMonitorSelector.MatchLabels
	}
	var serviceMonitorSelector map[string]string
	if in.PrometheusCR.ServiceMonitorSelector != nil {
		serviceMonitorSelector = in.PrometheusCR.ServiceMonitorSelector.MatchLabels
	}

	return OpenTelemetryTargetAllocator{
		Replicas:           in.Replicas,
		NodeSelector:       in.NodeSelector,
		Resources:          in.Resources,
		AllocationStrategy: tov1alpha1TAAllocationStrategy(in.AllocationStrategy),
		FilterStrategy:     tov1alpha1TAFilterStrategy(in.FilterStrategy),
		ServiceAccount:     in.ServiceAccount,
		Image:              in.Image,
		Enabled:            in.Enabled,
		Affinity:           in.Affinity,
		PrometheusCR: OpenTelemetryTargetAllocatorPrometheusCR{
			Enabled:                in.PrometheusCR.Enabled,
			ScrapeInterval:         in.PrometheusCR.ScrapeInterval,
			PodMonitorSelector:     podMonitorSelector,
			ServiceMonitorSelector: serviceMonitorSelector,
		},
		SecurityContext:           in.SecurityContext,
		PodSecurityContext:        in.PodSecurityContext,
		TopologySpreadConstraints: in.TopologySpreadConstraints,
		Tolerations:               in.Tolerations,
		Env:                       in.Env,
		Observability: ObservabilitySpec{
			Metrics: MetricsConfigSpec{
				EnableMetrics:                in.Observability.Metrics.EnableMetrics,
				DisablePrometheusAnnotations: in.Observability.Metrics.DisablePrometheusAnnotations,
			},
		},
		PodDisruptionBudget: tov1alpha1PodDisruptionBudget(in.PodDisruptionBudget),
	}
}

func tov1alpha1TAFilterStrategy(strategy v1beta1.TargetAllocatorFilterStrategy) string {
	switch strategy {
	case v1beta1.TargetAllocatorFilterStrategyRelabelConfig:
		return string(strategy)
	}
	return ""
}

func tov1alpha1TAAllocationStrategy(strategy v1beta1.TargetAllocatorAllocationStrategy) OpenTelemetryTargetAllocatorAllocationStrategy {
	switch strategy {
	case v1beta1.TargetAllocatorAllocationStrategyConsistentHashing:
		return OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashing
	case v1beta1.TargetAllocatorAllocationStrategyPerNode:
		return OpenTelemetryTargetAllocatorAllocationStrategyPerNode
	case v1beta1.TargetAllocatorAllocationStrategyLeastWeighted:
		return OpenTelemetryTargetAllocatorAllocationStrategyLeastWeighted
	}
	return ""
}

func tov1beta1TAFilterStrategy(strategy string) v1beta1.TargetAllocatorFilterStrategy {
	if strategy == string(v1beta1.TargetAllocatorFilterStrategyRelabelConfig) {
		return v1beta1.TargetAllocatorFilterStrategyRelabelConfig
	}
	return ""
}

func tov1beta1TAAllocationStrategy(strategy OpenTelemetryTargetAllocatorAllocationStrategy) v1beta1.TargetAllocatorAllocationStrategy {
	switch strategy {
	case OpenTelemetryTargetAllocatorAllocationStrategyPerNode:
		return v1beta1.TargetAllocatorAllocationStrategyPerNode
	case OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashing:
		return v1beta1.TargetAllocatorAllocationStrategyConsistentHashing
	case OpenTelemetryTargetAllocatorAllocationStrategyLeastWeighted:
		return v1beta1.TargetAllocatorAllocationStrategyLeastWeighted
	}
	return ""
}
