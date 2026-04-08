// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"

	go_yaml "github.com/goccy/go-yaml"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

var _ conversion.Convertible = &OpenTelemetryCollector{}

func (otc *OpenTelemetryCollector) ConvertTo(dstRaw conversion.Hub) error {
	switch t := dstRaw.(type) {
	case *v1beta1.OpenTelemetryCollector:
		dst := dstRaw.(*v1beta1.OpenTelemetryCollector)
		convertedSrc := tov1beta1(*otc)
		dst.ObjectMeta = convertedSrc.ObjectMeta
		dst.Spec = convertedSrc.Spec
		dst.Status = convertedSrc.Status
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
	return nil
}

func (otc *OpenTelemetryCollector) ConvertFrom(srcRaw conversion.Hub) error {
	switch t := srcRaw.(type) {
	case *v1beta1.OpenTelemetryCollector:
		src := srcRaw.(*v1beta1.OpenTelemetryCollector)
		srcConverted, err := tov1alpha1(*src)
		if err != nil {
			return fmt.Errorf("failed to convert to v1alpha1: %w", err)
		}
		otc.ObjectMeta = srcConverted.ObjectMeta
		otc.Spec = srcConverted.Spec
		otc.Status = srcConverted.Status
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
	return nil
}

func tov1beta1(in OpenTelemetryCollector) v1beta1.OpenTelemetryCollector {
	c := in.DeepCopy()
	cfg := &v1beta1.Config{}
	if err := go_yaml.Unmarshal([]byte(c.Spec.Config), cfg); err != nil {
		// It is critical that the conversion does not fail!
		// See https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#response
		// Thus, if unmarshalling fails, we return a valid, empty config.
		cfg = &v1beta1.Config{
			Receivers: v1beta1.AnyConfig{Object: map[string]any{}},
			Exporters: v1beta1.AnyConfig{Object: map[string]any{}},
			Service: v1beta1.Service{
				Pipelines: map[string]*v1beta1.Pipeline{},
			},
		}
	}

	// Ensure required fields are not nil even if unmarshalling succeeded.
	if cfg.Receivers.Object == nil {
		cfg.Receivers.Object = map[string]any{}
	}
	if cfg.Exporters.Object == nil {
		cfg.Exporters.Object = map[string]any{}
	}
	if cfg.Service.Pipelines == nil {
		cfg.Service.Pipelines = map[string]*v1beta1.Pipeline{}
	}

	return v1beta1.OpenTelemetryCollector{
		ObjectMeta: c.ObjectMeta,
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
				ManagementState:               v1beta1.ManagementStateType(c.Spec.ManagementState),
				Resources:                     c.Spec.Resources,
				NodeSelector:                  c.Spec.NodeSelector,
				Args:                          c.Spec.Args,
				Replicas:                      c.Spec.Replicas,
				PodDisruptionBudget:           tov1beta1PodDisruptionBudget(c.Spec.PodDisruptionBudget),
				SecurityContext:               c.Spec.SecurityContext,
				PodSecurityContext:            c.Spec.PodSecurityContext,
				PodAnnotations:                c.Spec.PodAnnotations,
				ServiceAccount:                c.Spec.ServiceAccount,
				Image:                         c.Spec.Image,
				ImagePullPolicy:               c.Spec.ImagePullPolicy,
				VolumeMounts:                  c.Spec.VolumeMounts,
				Ports:                         tov1beta1Ports(c.Spec.Ports),
				Env:                           c.Spec.Env,
				EnvFrom:                       c.Spec.EnvFrom,
				Tolerations:                   c.Spec.Tolerations,
				Volumes:                       c.Spec.Volumes,
				Affinity:                      c.Spec.Affinity,
				Lifecycle:                     c.Spec.Lifecycle,
				TerminationGracePeriodSeconds: c.Spec.TerminationGracePeriodSeconds,
				TopologySpreadConstraints:     c.Spec.TopologySpreadConstraints,
				HostNetwork:                   c.Spec.HostNetwork,
				ShareProcessNamespace:         c.Spec.ShareProcessNamespace,
				PriorityClassName:             c.Spec.PriorityClassName,
				InitContainers:                c.Spec.InitContainers,
				AdditionalContainers:          c.Spec.AdditionalContainers,
				TrafficDistribution:           c.Spec.TrafficDistribution,
			},
			StatefulSetCommonFields: v1beta1.StatefulSetCommonFields{
				VolumeClaimTemplates: c.Spec.VolumeClaimTemplates,
			},
			Autoscaler:      tov1beta1Autoscaler(c.Spec.Autoscaler, c.Spec.MinReplicas, c.Spec.MaxReplicas),
			TargetAllocator: tov1beta1TA(c.Spec.TargetAllocator),
			Mode:            v1beta1.Mode(c.Spec.Mode),
			UpgradeStrategy: v1beta1.UpgradeStrategy(c.Spec.UpgradeStrategy),
			Config:          *cfg,
			Ingress: v1beta1.Ingress{
				Type:             v1beta1.IngressType(c.Spec.Ingress.Type),
				RuleType:         v1beta1.IngressRuleType(c.Spec.Ingress.RuleType),
				Hostname:         c.Spec.Ingress.Hostname,
				Annotations:      c.Spec.Ingress.Annotations,
				TLS:              c.Spec.Ingress.TLS,
				IngressClassName: c.Spec.Ingress.IngressClassName,
				Route: v1beta1.OpenShiftRoute{
					Termination: v1beta1.TLSRouteTerminationType(c.Spec.Ingress.Route.Termination),
				},
			},
			LivenessProbe: tov1beta1Probe(c.Spec.LivenessProbe),
			Observability: v1beta1.ObservabilitySpec{
				Metrics: v1beta1.MetricsConfigSpec{
					EnableMetrics:                c.Spec.Observability.Metrics.EnableMetrics,
					DisablePrometheusAnnotations: c.Spec.Observability.Metrics.DisablePrometheusAnnotations,
				},
			},
			ConfigMaps:              tov1beta1ConfigMaps(c.Spec.ConfigMaps),
			DaemonSetUpdateStrategy: c.Spec.UpdateStrategy,
			DeploymentUpdateStrategy: appsv1.DeploymentStrategy{
				Type:          c.Spec.DeploymentUpdateStrategy.Type,
				RollingUpdate: c.Spec.DeploymentUpdateStrategy.RollingUpdate,
			},
		},
	}
}

func tov1beta1Ports(in []PortsSpec) []v1beta1.PortsSpec {
	var ports []v1beta1.PortsSpec

	for _, p := range in {
		ports = append(ports, v1beta1.PortsSpec{
			ServicePort: v1.ServicePort{
				Name:        p.Name,
				Protocol:    p.Protocol,
				AppProtocol: p.AppProtocol,
				Port:        p.Port,
				TargetPort:  p.TargetPort,
				NodePort:    p.NodePort,
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
			ScrapeClasses:  in.PrometheusCR.ScrapeClasses,
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
				Name:        p.Name,
				Protocol:    p.Protocol,
				AppProtocol: p.AppProtocol,
				Port:        p.Port,
				TargetPort:  p.TargetPort,
				NodePort:    p.NodePort,
			},
			HostPort: p.HostPort,
		})
	}

	return ports
}

func tov1alpha1(in v1beta1.OpenTelemetryCollector) (*OpenTelemetryCollector, error) {
	c := in.DeepCopy()
	configYaml, err := c.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}

	return &OpenTelemetryCollector{
		ObjectMeta: c.ObjectMeta,
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
			ManagementState:      ManagementStateType(c.Spec.ManagementState),
			Resources:            c.Spec.Resources,
			NodeSelector:         c.Spec.NodeSelector,
			Args:                 c.Spec.Args,
			Replicas:             c.Spec.Replicas,
			Autoscaler:           tov1alpha1Autoscaler(c.Spec.Autoscaler),
			PodDisruptionBudget:  tov1alpha1PodDisruptionBudget(c.Spec.PodDisruptionBudget),
			SecurityContext:      c.Spec.SecurityContext,
			PodSecurityContext:   c.Spec.PodSecurityContext,
			PodAnnotations:       c.Spec.PodAnnotations,
			TargetAllocator:      tov1alpha1TA(c.Spec.TargetAllocator),
			Mode:                 Mode(c.Spec.Mode),
			ServiceAccount:       c.Spec.ServiceAccount,
			Image:                c.Spec.Image,
			UpgradeStrategy:      UpgradeStrategy(c.Spec.UpgradeStrategy),
			ImagePullPolicy:      c.Spec.ImagePullPolicy,
			Config:               configYaml,
			VolumeMounts:         c.Spec.VolumeMounts,
			Ports:                tov1alpha1Ports(c.Spec.Ports),
			Env:                  c.Spec.Env,
			EnvFrom:              c.Spec.EnvFrom,
			VolumeClaimTemplates: c.Spec.VolumeClaimTemplates,
			Tolerations:          c.Spec.Tolerations,
			Volumes:              c.Spec.Volumes,
			Ingress: Ingress{
				Type:             IngressType(c.Spec.Ingress.Type),
				RuleType:         IngressRuleType(c.Spec.Ingress.RuleType),
				Hostname:         c.Spec.Ingress.Hostname,
				Annotations:      c.Spec.Ingress.Annotations,
				TLS:              c.Spec.Ingress.TLS,
				IngressClassName: c.Spec.Ingress.IngressClassName,
				Route: OpenShiftRoute{
					Termination: TLSRouteTerminationType(c.Spec.Ingress.Route.Termination),
				},
			},
			HostNetwork:                   c.Spec.HostNetwork,
			ShareProcessNamespace:         c.Spec.ShareProcessNamespace,
			PriorityClassName:             c.Spec.PriorityClassName,
			Affinity:                      c.Spec.Affinity,
			Lifecycle:                     c.Spec.Lifecycle,
			TerminationGracePeriodSeconds: c.Spec.TerminationGracePeriodSeconds,
			LivenessProbe:                 tov1alpha1Probe(c.Spec.LivenessProbe),
			InitContainers:                c.Spec.InitContainers,
			AdditionalContainers:          c.Spec.AdditionalContainers,
			Observability: ObservabilitySpec{
				Metrics: MetricsConfigSpec{
					EnableMetrics:                c.Spec.Observability.Metrics.EnableMetrics,
					DisablePrometheusAnnotations: c.Spec.Observability.Metrics.DisablePrometheusAnnotations,
				},
			},
			TopologySpreadConstraints: c.Spec.TopologySpreadConstraints,
			ConfigMaps:                tov1alpha1ConfigMaps(c.Spec.ConfigMaps),
			UpdateStrategy:            c.Spec.DaemonSetUpdateStrategy,
			DeploymentUpdateStrategy:  c.Spec.DeploymentUpdateStrategy,
			TrafficDistribution:       c.Spec.TrafficDistribution,
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
			ScrapeClasses:          in.PrometheusCR.ScrapeClasses,
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
	if strategy == v1beta1.TargetAllocatorFilterStrategyRelabelConfig {
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
