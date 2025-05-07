// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/fips"
	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

var (
	_ admission.CustomValidator = &CollectorWebhook{}
	_ admission.CustomDefaulter = &CollectorWebhook{}
)

// +kubebuilder:webhook:path=/mutate-opentelemetry-io-v1beta1-opentelemetrycollector,mutating=true,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=create;update,versions=v1beta1,name=mopentelemetrycollectorbeta.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=create;update,path=/validate-opentelemetry-io-v1beta1-opentelemetrycollector,mutating=false,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,versions=v1beta1,name=vopentelemetrycollectorcreateupdatebeta.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=delete,path=/validate-opentelemetry-io-v1beta1-opentelemetrycollector,mutating=false,failurePolicy=ignore,groups=opentelemetry.io,resources=opentelemetrycollectors,versions=v1beta1,name=vopentelemetrycollectordeletebeta.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:object:generate=false

type CollectorWebhook struct {
	logger   logr.Logger
	cfg      config.Config
	scheme   *runtime.Scheme
	reviewer *rbac.Reviewer
	metrics  *Metrics
	bv       BuildValidator
	fips     fips.FIPSCheck
}

func (c CollectorWebhook) Default(_ context.Context, obj runtime.Object) error {
	otelcol, ok := obj.(*OpenTelemetryCollector)
	if !ok {
		return fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}
	if len(otelcol.Spec.Mode) == 0 {
		otelcol.Spec.Mode = ModeDeployment
	}
	if len(otelcol.Spec.UpgradeStrategy) == 0 {
		otelcol.Spec.UpgradeStrategy = UpgradeStrategyAutomatic
	}

	if otelcol.Labels == nil {
		otelcol.Labels = map[string]string{}
	}

	// We can default to one because dependent objects Deployment and HorizontalPodAutoScaler
	// default to 1 as well.
	one := int32(1)
	if otelcol.Spec.Replicas == nil {
		otelcol.Spec.Replicas = &one
	}
	if otelcol.Spec.TargetAllocator.Enabled && otelcol.Spec.TargetAllocator.Replicas == nil {
		otelcol.Spec.TargetAllocator.Replicas = &one
	}

	if otelcol.Spec.Autoscaler != nil && otelcol.Spec.Autoscaler.MaxReplicas != nil {
		if otelcol.Spec.Autoscaler.MinReplicas == nil {
			otelcol.Spec.Autoscaler.MinReplicas = otelcol.Spec.Replicas
		}

		if otelcol.Spec.Autoscaler.TargetMemoryUtilization == nil && otelcol.Spec.Autoscaler.TargetCPUUtilization == nil {
			defaultCPUTarget := int32(90)
			otelcol.Spec.Autoscaler.TargetCPUUtilization = &defaultCPUTarget
		}
	}

	if otelcol.Spec.Ingress.Type == IngressTypeRoute && otelcol.Spec.Ingress.Route.Termination == "" {
		otelcol.Spec.Ingress.Route.Termination = TLSRouteTerminationTypeEdge
	}
	if otelcol.Spec.Ingress.Type == IngressTypeIngress && otelcol.Spec.Ingress.RuleType == "" {
		otelcol.Spec.Ingress.RuleType = IngressRuleTypePath
	}
	// If someone upgrades to a later version without upgrading their CRD they will not have a management state set.
	// This results in a default state of unmanaged preventing reconciliation from continuing.
	if len(otelcol.Spec.ManagementState) == 0 {
		otelcol.Spec.ManagementState = ManagementStateManaged
	}
	if !featuregate.EnableConfigDefaulting.IsEnabled() {
		return nil
	}
	return otelcol.Spec.Config.ApplyDefaults(c.logger)
}

func (c CollectorWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	otelcol, ok := obj.(*OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}

	warnings, err := c.Validate(ctx, otelcol)
	if err != nil {
		return warnings, err
	}
	if c.metrics != nil {
		c.metrics.create(ctx, otelcol)
	}
	if c.bv != nil {
		newWarnings := c.bv(ctx, *otelcol)
		warnings = append(warnings, newWarnings...)
	}
	return warnings, nil
}

func (c CollectorWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	otelcol, ok := newObj.(*OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", newObj)
	}

	otelcolOld, ok := oldObj.(*OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", oldObj)
	}

	if otelcolOld.Spec.Mode != otelcol.Spec.Mode {
		return admission.Warnings{}, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support modification", otelcolOld.Spec.Mode)
	}
	warnings, err := c.Validate(ctx, otelcol)
	if err != nil {
		return warnings, err
	}

	if c.metrics != nil {
		c.metrics.update(ctx, otelcolOld, otelcol)
	}

	if c.bv != nil {
		newWarnings := c.bv(ctx, *otelcol)
		warnings = append(warnings, newWarnings...)
	}
	return warnings, nil
}

func (c CollectorWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	otelcol, ok := obj.(*OpenTelemetryCollector)
	if !ok || otelcol == nil {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}

	warnings, err := c.Validate(ctx, otelcol)
	if err != nil {
		return warnings, err
	}

	if c.metrics != nil {
		c.metrics.delete(ctx, otelcol)
	}

	return warnings, nil
}

func (c CollectorWebhook) Validate(ctx context.Context, r *OpenTelemetryCollector) (admission.Warnings, error) {
	warnings := admission.Warnings{}

	nullObjects := r.Spec.Config.nullObjects()
	if len(nullObjects) > 0 {
		warnings = append(warnings, fmt.Sprintf("Collector config spec.config has null objects: %s. For compatibility with other tooling, such as kustomize and kubectl edit, it is recommended to use empty objects e.g. batch: {}.", strings.Join(nullObjects, ", ")))
	}

	// validate volumeClaimTemplates
	if r.Spec.Mode != ModeStatefulSet && len(r.Spec.VolumeClaimTemplates) > 0 {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'volumeClaimTemplates'", r.Spec.Mode)
	}

	// validate persistentVolumeClaimRetentionPolicy
	if r.Spec.Mode != ModeStatefulSet && r.Spec.PersistentVolumeClaimRetentionPolicy != nil {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'persistentVolumeClaimRetentionPolicy'", r.Spec.Mode)
	}

	// validate tolerations
	if r.Spec.Mode == ModeSidecar && len(r.Spec.Tolerations) > 0 {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'tolerations'", r.Spec.Mode)
	}

	// validate priorityClassName
	if r.Spec.Mode == ModeSidecar && r.Spec.PriorityClassName != "" {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'priorityClassName'", r.Spec.Mode)
	}

	// validate affinity
	if r.Spec.Mode == ModeSidecar && r.Spec.Affinity != nil {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'affinity'", r.Spec.Mode)
	}

	if r.Spec.Mode == ModeSidecar && len(r.Spec.AdditionalContainers) > 0 {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'AdditionalContainers'", r.Spec.Mode)
	}

	// validate target allocator configs
	if r.Spec.TargetAllocator.Enabled {
		taWarnings, err := c.validateTargetAllocatorConfig(ctx, r)
		if taWarnings != nil {
			warnings = append(warnings, taWarnings...)
		}
		if err != nil {
			return warnings, err
		}
	}

	// validate port config
	if err := ValidatePorts(r.Spec.Ports); err != nil {
		return warnings, err
	}

	var maxReplicas *int32
	if r.Spec.Autoscaler != nil && r.Spec.Autoscaler.MaxReplicas != nil {
		maxReplicas = r.Spec.Autoscaler.MaxReplicas
	}
	var minReplicas *int32
	if r.Spec.Autoscaler != nil && r.Spec.Autoscaler.MinReplicas != nil {
		minReplicas = r.Spec.Autoscaler.MinReplicas
	}
	// check deprecated .Spec.MinReplicas if minReplicas is not set
	if minReplicas == nil {
		minReplicas = r.Spec.Replicas
	}

	// validate autoscale with horizontal pod autoscaler
	if maxReplicas != nil {
		if *maxReplicas < int32(1) {
			return warnings, fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, maxReplicas should be defined and one or more")
		}

		if r.Spec.Replicas != nil && *r.Spec.Replicas > *maxReplicas {
			return warnings, fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, replicas must not be greater than maxReplicas")
		}

		if minReplicas != nil && *minReplicas > *maxReplicas {
			return warnings, fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, minReplicas must not be greater than maxReplicas")
		}

		if minReplicas != nil && *minReplicas < int32(1) {
			return warnings, fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, minReplicas should be one or more")
		}

		if r.Spec.Autoscaler != nil {
			return warnings, checkAutoscalerSpec(r.Spec.Autoscaler)
		}
	}

	if r.Spec.Ingress.Type == IngressTypeIngress && r.Spec.Mode == ModeSidecar {
		return warnings, fmt.Errorf("the OpenTelemetry Spec Ingress configuration is incorrect. Ingress can only be used in combination with the modes: %s, %s, %s",
			ModeDeployment, ModeDaemonSet, ModeStatefulSet,
		)
	}

	if r.Spec.Ingress.Type == IngressTypeIngress && r.Spec.Mode == ModeSidecar {
		return warnings, fmt.Errorf("the OpenTelemetry Spec Ingress configuiration is incorrect. Ingress can only be used in combination with the modes: %s, %s, %s",
			ModeDeployment, ModeDaemonSet, ModeStatefulSet,
		)
	}
	if r.Spec.Ingress.RuleType == IngressRuleTypeSubdomain && (r.Spec.Ingress.Hostname == "" || r.Spec.Ingress.Hostname == "*") {
		return warnings, fmt.Errorf("a valid Ingress hostname has to be defined for subdomain ruleType")
	}

	// validate probes Liveness/Readiness
	err := ValidateProbe("LivenessProbe", r.Spec.LivenessProbe)
	if err != nil {
		return warnings, err
	}
	err = ValidateProbe("ReadinessProbe", r.Spec.ReadinessProbe)
	if err != nil {
		return warnings, err
	}

	// validate updateStrategy for DaemonSet
	if r.Spec.Mode != ModeDaemonSet && len(r.Spec.DaemonSetUpdateStrategy.Type) > 0 {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'updateStrategy'", r.Spec.Mode)
	}

	// validate updateStrategy for Deployment
	if r.Spec.Mode != ModeDeployment && len(r.Spec.DeploymentUpdateStrategy.Type) > 0 {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'deploymentUpdateStrategy'", r.Spec.Mode)
	}

	if c.fips != nil {
		components := r.Spec.Config.GetEnabledComponents()
		if notAllowedComponents := c.fips.DisabledComponents(components[KindReceiver], components[KindExporter], components[KindProcessor], components[KindExtension]); notAllowedComponents != nil {
			return nil, fmt.Errorf("the collector configuration contains not FIPS compliant components: %s. Please remove it from the config", notAllowedComponents)
		}
	}

	return warnings, nil
}

func (c CollectorWebhook) validateTargetAllocatorConfig(ctx context.Context, r *OpenTelemetryCollector) (admission.Warnings, error) {
	if r.Spec.Mode != ModeStatefulSet && r.Spec.Mode != ModeDaemonSet {
		return nil, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the target allocation deployment", r.Spec.Mode)
	}

	if r.Spec.Mode == ModeDaemonSet && r.Spec.TargetAllocator.AllocationStrategy != TargetAllocatorAllocationStrategyPerNode {
		return nil, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which must be used with target allocation strategy %s ", r.Spec.Mode, TargetAllocatorAllocationStrategyPerNode)
	}

	if r.Spec.TargetAllocator.AllocationStrategy == TargetAllocatorAllocationStrategyPerNode && r.Spec.Mode != ModeDaemonSet {
		return nil, fmt.Errorf("target allocation strategy %s is only supported in OpenTelemetry Collector mode %s", TargetAllocatorAllocationStrategyPerNode, ModeDaemonSet)
	}

	cfgYaml, err := r.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}
	// validate Prometheus config for target allocation
	promCfg, err := ta.ConfigToPromConfig(cfgYaml)
	if err != nil {
		return nil, fmt.Errorf("the OpenTelemetry Spec Prometheus configuration is incorrect, %w", err)
	}
	err = ta.ValidatePromConfig(promCfg, r.Spec.TargetAllocator.Enabled)
	if err != nil {
		return nil, fmt.Errorf("the OpenTelemetry Spec Prometheus configuration is incorrect, %w", err)
	}
	err = ta.ValidateTargetAllocatorConfig(r.Spec.TargetAllocator.PrometheusCR.Enabled, promCfg)
	if err != nil {
		return nil, fmt.Errorf("the OpenTelemetry Spec Prometheus configuration is incorrect, %w", err)
	}
	// if the prometheusCR is enabled, it needs a suite of permissions to function
	if r.Spec.TargetAllocator.PrometheusCR.Enabled {
		saname := r.Spec.TargetAllocator.ServiceAccount
		if len(r.Spec.TargetAllocator.ServiceAccount) == 0 {
			saname = naming.TargetAllocatorServiceAccount(r.Name)
		}
		warnings, err := CheckTargetAllocatorPrometheusCRPolicyRules(
			ctx, c.reviewer, r.GetNamespace(), saname)
		if err != nil || len(warnings) > 0 {
			return warnings, err
		}
	}

	return nil, nil
}

func ValidateProbe(probeName string, probe *Probe) error {
	if probe != nil {
		if probe.InitialDelaySeconds != nil && *probe.InitialDelaySeconds < 0 {
			return fmt.Errorf("the OpenTelemetry Spec %s InitialDelaySeconds configuration is incorrect. InitialDelaySeconds should be greater than or equal to 0", probeName)
		}
		if probe.PeriodSeconds != nil && *probe.PeriodSeconds < 1 {
			return fmt.Errorf("the OpenTelemetry Spec %s PeriodSeconds configuration is incorrect. PeriodSeconds should be greater than or equal to 1", probeName)
		}
		if probe.TimeoutSeconds != nil && *probe.TimeoutSeconds < 1 {
			return fmt.Errorf("the OpenTelemetry Spec %s TimeoutSeconds configuration is incorrect. TimeoutSeconds should be greater than or equal to 1", probeName)
		}
		if probe.SuccessThreshold != nil && *probe.SuccessThreshold < 1 {
			return fmt.Errorf("the OpenTelemetry Spec %s SuccessThreshold configuration is incorrect. SuccessThreshold should be greater than or equal to 1", probeName)
		}
		if probe.FailureThreshold != nil && *probe.FailureThreshold < 1 {
			return fmt.Errorf("the OpenTelemetry Spec %s FailureThreshold configuration is incorrect. FailureThreshold should be greater than or equal to 1", probeName)
		}
		if probe.TerminationGracePeriodSeconds != nil && *probe.TerminationGracePeriodSeconds < 1 {
			return fmt.Errorf("the OpenTelemetry Spec %s TerminationGracePeriodSeconds configuration is incorrect. TerminationGracePeriodSeconds should be greater than or equal to 1", probeName)
		}
	}
	return nil
}

func ValidatePorts(ports []PortsSpec) error {
	for _, p := range ports {
		nameErrs := validation.IsValidPortName(p.Name)
		numErrs := validation.IsValidPortNum(int(p.Port))
		if len(nameErrs) > 0 || len(numErrs) > 0 {
			return fmt.Errorf("the OpenTelemetry Spec Ports configuration is incorrect, port name '%s' errors: %s, num '%d' errors: %s",
				p.Name, nameErrs, p.Port, numErrs)
		}
	}
	return nil
}

func checkAutoscalerSpec(autoscaler *AutoscalerSpec) error {
	if autoscaler.Behavior != nil {
		if autoscaler.Behavior.ScaleDown != nil && autoscaler.Behavior.ScaleDown.StabilizationWindowSeconds != nil &&
			(*autoscaler.Behavior.ScaleDown.StabilizationWindowSeconds < int32(0) || *autoscaler.Behavior.ScaleDown.StabilizationWindowSeconds > 3600) {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, scaleDown.stabilizationWindowSeconds should be >=0 and <=3600")
		}

		if autoscaler.Behavior.ScaleUp != nil && autoscaler.Behavior.ScaleUp.StabilizationWindowSeconds != nil &&
			(*autoscaler.Behavior.ScaleUp.StabilizationWindowSeconds < int32(0) || *autoscaler.Behavior.ScaleUp.StabilizationWindowSeconds > 3600) {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, scaleUp.stabilizationWindowSeconds should be >=0 and <=3600")
		}
	}
	if autoscaler.TargetCPUUtilization != nil && *autoscaler.TargetCPUUtilization < int32(1) {
		return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, targetCPUUtilization should be greater than 0")
	}
	if autoscaler.TargetMemoryUtilization != nil && *autoscaler.TargetMemoryUtilization < int32(1) {
		return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, targetMemoryUtilization should be greater than 0")
	}

	for _, metric := range autoscaler.Metrics {
		if metric.Type != autoscalingv2.PodsMetricSourceType {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, metric type unsupported. Expected metric of source type Pod")
		}

		// pod metrics target only support value and averageValue.
		if metric.Pods.Target.Type == autoscalingv2.AverageValueMetricType {
			if val, ok := metric.Pods.Target.AverageValue.AsInt64(); !ok || val < int64(1) {
				return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, average value should be greater than 0")
			}
		} else if metric.Pods.Target.Type == autoscalingv2.ValueMetricType {
			if val, ok := metric.Pods.Target.Value.AsInt64(); !ok || val < int64(1) {
				return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, value should be greater than 0")
			}
		} else {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, invalid pods target type")
		}
	}

	return nil
}

// BuildValidator enables running the manifest generators for the collector reconciler
// +kubebuilder:object:generate=false
type BuildValidator func(ctx context.Context, c OpenTelemetryCollector) admission.Warnings

func NewCollectorWebhook(
	logger logr.Logger,
	scheme *runtime.Scheme,
	cfg config.Config,
	reviewer *rbac.Reviewer,
	metrics *Metrics,
	bv BuildValidator,
	fips fips.FIPSCheck,
) *CollectorWebhook {
	return &CollectorWebhook{
		logger:   logger,
		scheme:   scheme,
		cfg:      cfg,
		reviewer: reviewer,
		metrics:  metrics,
		bv:       bv,
		fips:     fips,
	}
}

func SetupCollectorWebhook(mgr ctrl.Manager, cfg config.Config, reviewer *rbac.Reviewer, metrics *Metrics, bv BuildValidator, fipsCheck fips.FIPSCheck) error {
	cvw := NewCollectorWebhook(mgr.GetLogger().WithValues("handler", "CollectorWebhook", "version", "v1beta1"), mgr.GetScheme(), cfg, reviewer, metrics, bv, fipsCheck)
	return ctrl.NewWebhookManagedBy(mgr).
		For(&OpenTelemetryCollector{}).
		WithValidator(cvw).
		WithDefaulter(cvw).
		Complete()
}
