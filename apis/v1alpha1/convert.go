// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	copy := in.DeepCopy()
	cfg := &v1beta1.Config{}
	if err := yaml.Unmarshal([]byte(copy.Spec.Config), cfg); err != nil {
		return v1beta1.OpenTelemetryCollector{}, errors.New("could not convert config json to v1beta1.Config")
	}

	return v1beta1.OpenTelemetryCollector{
		ObjectMeta: copy.ObjectMeta,
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
				ManagementState:               v1beta1.ManagementStateType(copy.Spec.ManagementState),
				Resources:                     copy.Spec.Resources,
				NodeSelector:                  copy.Spec.NodeSelector,
				Args:                          copy.Spec.Args,
				Replicas:                      copy.Spec.Replicas,
				PodDisruptionBudget:           tov1beta1PodDisruptionBudget(copy.Spec.PodDisruptionBudget),
				SecurityContext:               copy.Spec.SecurityContext,
				PodSecurityContext:            copy.Spec.PodSecurityContext,
				PodAnnotations:                copy.Spec.PodAnnotations,
				ServiceAccount:                copy.Spec.ServiceAccount,
				Image:                         copy.Spec.Image,
				ImagePullPolicy:               copy.Spec.ImagePullPolicy,
				VolumeMounts:                  copy.Spec.VolumeMounts,
				Ports:                         tov1beta1Ports(copy.Spec.Ports),
				Env:                           copy.Spec.Env,
				EnvFrom:                       copy.Spec.EnvFrom,
				Tolerations:                   copy.Spec.Tolerations,
				Volumes:                       copy.Spec.Volumes,
				Affinity:                      copy.Spec.Affinity,
				Lifecycle:                     copy.Spec.Lifecycle,
				TerminationGracePeriodSeconds: copy.Spec.TerminationGracePeriodSeconds,
				TopologySpreadConstraints:     copy.Spec.TopologySpreadConstraints,
				HostNetwork:                   copy.Spec.HostNetwork,
				ShareProcessNamespace:         copy.Spec.ShareProcessNamespace,
				PriorityClassName:             copy.Spec.PriorityClassName,
				InitContainers:                copy.Spec.InitContainers,
				AdditionalContainers:          copy.Spec.AdditionalContainers,
			},
			StatefulSetCommonFields: v1beta1.StatefulSetCommonFields{
				VolumeClaimTemplates: copy.Spec.VolumeClaimTemplates,
			},
			Autoscaler:      tov1beta1Autoscaler(copy.Spec.Autoscaler, copy.Spec.MinReplicas, copy.Spec.MaxReplicas),
			TargetAllocator: tov1beta1TA(copy.Spec.TargetAllocator),
			Mode:            v1beta1.Mode(copy.Spec.Mode),
			UpgradeStrategy: v1beta1.UpgradeStrategy(copy.Spec.UpgradeStrategy),
			Config:          *cfg,
			Ingress: v1beta1.Ingress{
				Type:             v1beta1.IngressType(copy.Spec.Ingress.Type),
				RuleType:         v1beta1.IngressRuleType(copy.Spec.Ingress.RuleType),
				Hostname:         copy.Spec.Ingress.Hostname,
				Annotations:      copy.Spec.Ingress.Annotations,
				TLS:              copy.Spec.Ingress.TLS,
				IngressClassName: copy.Spec.Ingress.IngressClassName,
				Route: v1beta1.OpenShiftRoute{
					Termination: v1beta1.TLSRouteTerminationType(copy.Spec.Ingress.Route.Termination),
				},
			},
			LivenessProbe: tov1beta1Probe(copy.Spec.LivenessProbe),
			Observability: v1beta1.ObservabilitySpec{
				Metrics: v1beta1.MetricsConfigSpec{
					EnableMetrics:                copy.Spec.Observability.Metrics.EnableMetrics,
					DisablePrometheusAnnotations: copy.Spec.Observability.Metrics.DisablePrometheusAnnotations,
				},
			},
			ConfigMaps:              tov1beta1ConfigMaps(copy.Spec.ConfigMaps),
			DaemonSetUpdateStrategy: copy.Spec.UpdateStrategy,
			DeploymentUpdateStrategy: appsv1.DeploymentStrategy{
				Type:          copy.Spec.DeploymentUpdateStrategy.Type,
				RollingUpdate: copy.Spec.DeploymentUpdateStrategy.RollingUpdate,
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
	copy := in.DeepCopy()
	configYaml, err := copy.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}

	return &OpenTelemetryCollector{
		ObjectMeta: copy.ObjectMeta,
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
			ManagementState:      ManagementStateType(copy.Spec.ManagementState),
			Resources:            copy.Spec.Resources,
			NodeSelector:         copy.Spec.NodeSelector,
			Args:                 copy.Spec.Args,
			Replicas:             copy.Spec.Replicas,
			Autoscaler:           tov1alpha1Autoscaler(copy.Spec.Autoscaler),
			PodDisruptionBudget:  tov1alpha1PodDisruptionBudget(copy.Spec.PodDisruptionBudget),
			SecurityContext:      copy.Spec.SecurityContext,
			PodSecurityContext:   copy.Spec.PodSecurityContext,
			PodAnnotations:       copy.Spec.PodAnnotations,
			TargetAllocator:      tov1alpha1TA(copy.Spec.TargetAllocator),
			Mode:                 Mode(copy.Spec.Mode),
			ServiceAccount:       copy.Spec.ServiceAccount,
			Image:                copy.Spec.Image,
			UpgradeStrategy:      UpgradeStrategy(copy.Spec.UpgradeStrategy),
			ImagePullPolicy:      copy.Spec.ImagePullPolicy,
			Config:               configYaml,
			VolumeMounts:         copy.Spec.VolumeMounts,
			Ports:                tov1alpha1Ports(copy.Spec.Ports),
			Env:                  copy.Spec.Env,
			EnvFrom:              copy.Spec.EnvFrom,
			VolumeClaimTemplates: copy.Spec.VolumeClaimTemplates,
			Tolerations:          copy.Spec.Tolerations,
			Volumes:              copy.Spec.Volumes,
			Ingress: Ingress{
				Type:             IngressType(copy.Spec.Ingress.Type),
				RuleType:         IngressRuleType(copy.Spec.Ingress.RuleType),
				Hostname:         copy.Spec.Ingress.Hostname,
				Annotations:      copy.Spec.Ingress.Annotations,
				TLS:              copy.Spec.Ingress.TLS,
				IngressClassName: copy.Spec.Ingress.IngressClassName,
				Route: OpenShiftRoute{
					Termination: TLSRouteTerminationType(copy.Spec.Ingress.Route.Termination),
				},
			},
			HostNetwork:                   copy.Spec.HostNetwork,
			ShareProcessNamespace:         copy.Spec.ShareProcessNamespace,
			PriorityClassName:             copy.Spec.PriorityClassName,
			Affinity:                      copy.Spec.Affinity,
			Lifecycle:                     copy.Spec.Lifecycle,
			TerminationGracePeriodSeconds: copy.Spec.TerminationGracePeriodSeconds,
			LivenessProbe:                 tov1alpha1Probe(copy.Spec.LivenessProbe),
			InitContainers:                copy.Spec.InitContainers,
			AdditionalContainers:          copy.Spec.AdditionalContainers,
			Observability: ObservabilitySpec{
				Metrics: MetricsConfigSpec{
					EnableMetrics:                copy.Spec.Observability.Metrics.EnableMetrics,
					DisablePrometheusAnnotations: copy.Spec.Observability.Metrics.DisablePrometheusAnnotations,
				},
			},
			TopologySpreadConstraints: copy.Spec.TopologySpreadConstraints,
			ConfigMaps:                tov1alpha1ConfigMaps(copy.Spec.ConfigMaps),
			UpdateStrategy:            copy.Spec.DaemonSetUpdateStrategy,
			DeploymentUpdateStrategy:  copy.Spec.DeploymentUpdateStrategy,
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
