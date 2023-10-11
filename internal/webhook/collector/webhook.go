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

package collectorwebhook

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

var (
	_ admission.CustomValidator = &Webhook{}
	_ admission.CustomDefaulter = &Webhook{}
)

// +kubebuilder:webhook:path=/mutate-opentelemetry-io-v1alpha1-opentelemetrycollector,mutating=true,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=create;update,versions=v1alpha1,name=mopentelemetrycollector.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=create;update,path=/validate-opentelemetry-io-v1alpha1-opentelemetrycollector,mutating=false,failurePolicy=fail,groups=opentelemetry.io,resources=opentelemetrycollectors,versions=v1alpha1,name=vopentelemetrycollectorcreateupdate.kb.io,sideEffects=none,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=delete,path=/validate-opentelemetry-io-v1alpha1-opentelemetrycollector,mutating=false,failurePolicy=ignore,groups=opentelemetry.io,resources=opentelemetrycollectors,versions=v1alpha1,name=vopentelemetrycollectordelete.kb.io,sideEffects=none,admissionReviewVersions=v1

// Webhook is isolated because there are known registration issues when a custom webhook is in the same package
// as the types.
// See here: https://github.com/kubernetes-sigs/controller-runtime/issues/780#issuecomment-713408479
type Webhook struct {
	logger logr.Logger
	cfg    config.Config
	scheme *runtime.Scheme
}

func (c Webhook) Default(ctx context.Context, obj runtime.Object) error {
	otelcol, ok := obj.(*v1alpha1.OpenTelemetryCollector)
	if !ok {
		return fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}
	return c.defaulter(otelcol)
}

func (c Webhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	otelcol, ok := obj.(*v1alpha1.OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}
	return c.validate(otelcol)
}

func (c Webhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	otelcol, ok := newObj.(*v1alpha1.OpenTelemetryCollector)
	if !ok {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", newObj)
	}
	return c.validate(otelcol)
}

func (c Webhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	otelcol, ok := obj.(*v1alpha1.OpenTelemetryCollector)
	if !ok || otelcol == nil {
		return nil, fmt.Errorf("expected an OpenTelemetryCollector, received %T", obj)
	}
	return c.validate(otelcol)
}

func (c Webhook) defaulter(r *v1alpha1.OpenTelemetryCollector) error {
	if len(r.Spec.Mode) == 0 {
		r.Spec.Mode = v1alpha1.ModeDeployment
	}
	if len(r.Spec.UpgradeStrategy) == 0 {
		r.Spec.UpgradeStrategy = v1alpha1.UpgradeStrategyAutomatic
	}

	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	if r.Labels["app.kubernetes.io/managed-by"] == "" {
		r.Labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	}

	// We can default to one because dependent objects Deployment and HorizontalPodAutoScaler
	// default to 1 as well.
	one := int32(1)
	if r.Spec.Replicas == nil {
		r.Spec.Replicas = &one
	}
	if r.Spec.TargetAllocator.Enabled && r.Spec.TargetAllocator.Replicas == nil {
		r.Spec.TargetAllocator.Replicas = &one
	}

	if r.Spec.MaxReplicas != nil || (r.Spec.Autoscaler != nil && r.Spec.Autoscaler.MaxReplicas != nil) {
		if r.Spec.Autoscaler == nil {
			r.Spec.Autoscaler = &v1alpha1.AutoscalerSpec{}
		}

		if r.Spec.Autoscaler.MaxReplicas == nil {
			r.Spec.Autoscaler.MaxReplicas = r.Spec.MaxReplicas
		}
		if r.Spec.Autoscaler.MinReplicas == nil {
			if r.Spec.MinReplicas != nil {
				r.Spec.Autoscaler.MinReplicas = r.Spec.MinReplicas
			} else {
				r.Spec.Autoscaler.MinReplicas = r.Spec.Replicas
			}
		}

		if r.Spec.Autoscaler.TargetMemoryUtilization == nil && r.Spec.Autoscaler.TargetCPUUtilization == nil {
			defaultCPUTarget := int32(90)
			r.Spec.Autoscaler.TargetCPUUtilization = &defaultCPUTarget
		}
	}

	// if pod isn't provided, we set MaxUnavailable 1,
	// which will work even if there is just one replica,
	// not blocking node drains but preventing out-of-the-box
	// from disruption generated by them with replicas > 1
	if r.Spec.PodDisruptionBudget == nil {
		r.Spec.PodDisruptionBudget = &v1alpha1.PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		}
	}

	if r.Spec.Ingress.Type == v1alpha1.IngressTypeRoute && r.Spec.Ingress.Route.Termination == "" {
		r.Spec.Ingress.Route.Termination = v1alpha1.TLSRouteTerminationTypeEdge
	}
	if r.Spec.Ingress.Type == v1alpha1.IngressTypeNginx && r.Spec.Ingress.RuleType == "" {
		r.Spec.Ingress.RuleType = v1alpha1.IngressRuleTypePath
	}
	// If someone upgrades to a later version without upgrading their CRD they will not have a management state set.
	// This results in a default state of unmanaged preventing reconciliation from continuing.
	if len(r.Spec.ManagementState) == 0 {
		r.Spec.ManagementState = v1alpha1.ManagementStateManaged
	}
	return nil
}

func (c Webhook) validate(r *v1alpha1.OpenTelemetryCollector) (admission.Warnings, error) {
	warnings := admission.Warnings{}
	// validate volumeClaimTemplates
	if r.Spec.Mode != v1alpha1.ModeStatefulSet && len(r.Spec.VolumeClaimTemplates) > 0 {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'volumeClaimTemplates'", r.Spec.Mode)
	}

	// validate tolerations
	if r.Spec.Mode == v1alpha1.ModeSidecar && len(r.Spec.Tolerations) > 0 {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'tolerations'", r.Spec.Mode)
	}

	// validate priorityClassName
	if r.Spec.Mode == v1alpha1.ModeSidecar && r.Spec.PriorityClassName != "" {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'priorityClassName'", r.Spec.Mode)
	}

	// validate affinity
	if r.Spec.Mode == v1alpha1.ModeSidecar && r.Spec.Affinity != nil {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'affinity'", r.Spec.Mode)
	}

	if r.Spec.Mode == v1alpha1.ModeSidecar && len(r.Spec.AdditionalContainers) > 0 {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the attribute 'AdditionalContainers'", r.Spec.Mode)
	}

	// validate target allocation
	if r.Spec.TargetAllocator.Enabled && r.Spec.Mode != v1alpha1.ModeStatefulSet {
		return warnings, fmt.Errorf("the OpenTelemetry Collector mode is set to %s, which does not support the target allocation deployment", r.Spec.Mode)
	}

	// validate Prometheus config for target allocation
	if r.Spec.TargetAllocator.Enabled {
		promCfg, err := ta.ConfigToPromConfig(r.Spec.Config)
		if err != nil {
			return warnings, fmt.Errorf("the OpenTelemetry Spec Prometheus configuration is incorrect, %w", err)
		}
		err = ta.ValidatePromConfig(promCfg, r.Spec.TargetAllocator.Enabled, featuregate.EnableTargetAllocatorRewrite.IsEnabled())
		if err != nil {
			return warnings, fmt.Errorf("the OpenTelemetry Spec Prometheus configuration is incorrect, %w", err)
		}
		err = ta.ValidateTargetAllocatorConfig(r.Spec.TargetAllocator.PrometheusCR.Enabled, promCfg)
		if err != nil {
			return warnings, fmt.Errorf("the OpenTelemetry Spec Prometheus configuration is incorrect, %w", err)
		}
	}

	// validator port config
	for _, p := range r.Spec.Ports {
		nameErrs := validation.IsValidPortName(p.Name)
		numErrs := validation.IsValidPortNum(int(p.Port))
		if len(nameErrs) > 0 || len(numErrs) > 0 {
			return warnings, fmt.Errorf("the OpenTelemetry Spec Ports configuration is incorrect, port name '%s' errors: %s, num '%d' errors: %s",
				p.Name, nameErrs, p.Port, numErrs)
		}
	}

	var maxReplicas *int32
	if r.Spec.Autoscaler != nil && r.Spec.Autoscaler.MaxReplicas != nil {
		maxReplicas = r.Spec.Autoscaler.MaxReplicas
	}

	// check deprecated .Spec.MaxReplicas if maxReplicas is not set
	if maxReplicas == nil && r.Spec.MaxReplicas != nil {
		warnings = append(warnings, "MaxReplicas is deprecated")
		maxReplicas = r.Spec.MaxReplicas
	}

	var minReplicas *int32
	if r.Spec.Autoscaler != nil && r.Spec.Autoscaler.MinReplicas != nil {
		minReplicas = r.Spec.Autoscaler.MinReplicas
	}

	// check deprecated .Spec.MinReplicas if minReplicas is not set
	if minReplicas == nil {
		if r.Spec.MinReplicas != nil {
			warnings = append(warnings, "MinReplicas is deprecated")
			minReplicas = r.Spec.MinReplicas
		} else {
			minReplicas = r.Spec.Replicas
		}
	}

	if r.Spec.Ingress.Type == v1alpha1.IngressTypeNginx && r.Spec.Mode == v1alpha1.ModeSidecar {
		return warnings, fmt.Errorf("the OpenTelemetry Spec Ingress configuration is incorrect. Ingress can only be used in combination with the modes: %s, %s, %s",
			v1alpha1.ModeDeployment, v1alpha1.ModeDaemonSet, v1alpha1.ModeStatefulSet,
		)
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

	if r.Spec.Ingress.Type == v1alpha1.IngressTypeNginx && r.Spec.Mode == v1alpha1.ModeSidecar {
		return warnings, fmt.Errorf("the OpenTelemetry Spec Ingress configuiration is incorrect. Ingress can only be used in combination with the modes: %s, %s, %s",
			v1alpha1.ModeDeployment, v1alpha1.ModeDaemonSet, v1alpha1.ModeStatefulSet,
		)
	}
	if r.Spec.Ingress.RuleType == v1alpha1.IngressRuleTypeSubdomain && (r.Spec.Ingress.Hostname == "" || r.Spec.Ingress.Hostname == "*") {
		return warnings, fmt.Errorf("a valid Ingress hostname has to be defined for subdomain ruleType")
	}

	if r.Spec.LivenessProbe != nil {
		if r.Spec.LivenessProbe.InitialDelaySeconds != nil && *r.Spec.LivenessProbe.InitialDelaySeconds < 0 {
			return warnings, fmt.Errorf("the OpenTelemetry Spec LivenessProbe InitialDelaySeconds configuration is incorrect. InitialDelaySeconds should be greater than or equal to 0")
		}
		if r.Spec.LivenessProbe.PeriodSeconds != nil && *r.Spec.LivenessProbe.PeriodSeconds < 1 {
			return warnings, fmt.Errorf("the OpenTelemetry Spec LivenessProbe PeriodSeconds configuration is incorrect. PeriodSeconds should be greater than or equal to 1")
		}
		if r.Spec.LivenessProbe.TimeoutSeconds != nil && *r.Spec.LivenessProbe.TimeoutSeconds < 1 {
			return warnings, fmt.Errorf("the OpenTelemetry Spec LivenessProbe TimeoutSeconds configuration is incorrect. TimeoutSeconds should be greater than or equal to 1")
		}
		if r.Spec.LivenessProbe.SuccessThreshold != nil && *r.Spec.LivenessProbe.SuccessThreshold < 1 {
			return warnings, fmt.Errorf("the OpenTelemetry Spec LivenessProbe SuccessThreshold configuration is incorrect. SuccessThreshold should be greater than or equal to 1")
		}
		if r.Spec.LivenessProbe.FailureThreshold != nil && *r.Spec.LivenessProbe.FailureThreshold < 1 {
			return warnings, fmt.Errorf("the OpenTelemetry Spec LivenessProbe FailureThreshold configuration is incorrect. FailureThreshold should be greater than or equal to 1")
		}
		if r.Spec.LivenessProbe.TerminationGracePeriodSeconds != nil && *r.Spec.LivenessProbe.TerminationGracePeriodSeconds < 1 {
			return warnings, fmt.Errorf("the OpenTelemetry Spec LivenessProbe TerminationGracePeriodSeconds configuration is incorrect. TerminationGracePeriodSeconds should be greater than or equal to 1")
		}
	}

	return warnings, nil
}

func checkAutoscalerSpec(autoscaler *v1alpha1.AutoscalerSpec) error {
	if autoscaler.Behavior != nil {
		if autoscaler.Behavior.ScaleDown != nil && autoscaler.Behavior.ScaleDown.StabilizationWindowSeconds != nil &&
			*autoscaler.Behavior.ScaleDown.StabilizationWindowSeconds < int32(1) {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, scaleDown should be one or more")
		}

		if autoscaler.Behavior.ScaleUp != nil && autoscaler.Behavior.ScaleUp.StabilizationWindowSeconds != nil &&
			*autoscaler.Behavior.ScaleUp.StabilizationWindowSeconds < int32(1) {
			return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, scaleUp should be one or more")
		}
	}
	if autoscaler.TargetCPUUtilization != nil && (*autoscaler.TargetCPUUtilization < int32(1) || *autoscaler.TargetCPUUtilization > int32(99)) {
		return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, targetCPUUtilization should be greater than 0 and less than 100")
	}
	if autoscaler.TargetMemoryUtilization != nil && (*autoscaler.TargetMemoryUtilization < int32(1) || *autoscaler.TargetMemoryUtilization > int32(99)) {
		return fmt.Errorf("the OpenTelemetry Spec autoscale configuration is incorrect, targetMemoryUtilization should be greater than 0 and less than 100")
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

func SetupWebhook(mgr ctrl.Manager, cfg config.Config) error {
	cvw := &Webhook{
		logger: mgr.GetLogger().WithValues("handler", "CollectorWebhook"),
		scheme: mgr.GetScheme(),
		cfg:    cfg,
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.OpenTelemetryCollector{}).
		WithValidator(cvw).
		WithDefaulter(cvw).
		Complete()
}